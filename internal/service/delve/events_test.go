package delve

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

func delveTestService(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	d := testutil.NewDB(t)
	cfg := &config.Config{}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

// TestZoneSetsRegistered checks that every zone set name the delve loot table
// can assign exists in the items set registry with real bonuses.
func TestZoneSetsRegistered(t *testing.T) {
	for zone, set := range zoneSetNames {
		def, ok := items.SetsByName[set.SetID]
		require.True(t, ok, "zone %q set %q must exist in items.SetsByName", zone, set.SetID)
		assert.NotEmpty(t, def.Bonuses, "set %q must have bonuses", set.SetID)
		assert.NotEqual(t, "", set.SetName, "zone %q set must carry a display name", zone)
	}
}

// TestAssignSetNameOnlyRareEpic ensures delve set pieces are mid-game gear:
// Common/Uncommon never become set pieces, and Legendary pieces stay standalone.
func TestAssignSetNameOnlyRareEpic(t *testing.T) {
	for _, r := range []Rarity{Common, Uncommon, Legendary} {
		item := DelveItem{Rarity: r, ID: "delve_test_item"}
		AssignSetName(&item, "crypt")
		assert.Empty(t, item.SetName, "rarity %v must never become a set piece", r)
	}
	for _, r := range []Rarity{Rare, Epic} {
		item := DelveItem{Rarity: r, ID: "delve_test_item"}
		AssignSetName(&item, "crypt")
		if item.SetName != "" {
			assert.Equal(t, "Crypt Lord's Regalia", item.SetName)
		}
	}
}

// TestNewRoomsInZoneTables checks that the new event rooms are present in the
// zone tables they are meant for.
func TestNewRoomsInZoneTables(t *testing.T) {
	hasRoom := func(zone string, rt RoomType) bool {
		for _, w := range zoneRoomTables[zone] {
			if w.Type == rt {
				return true
			}
		}
		return false
	}
	assert.True(t, hasRoom("crypt", RoomArchive))
	assert.True(t, hasRoom("crypt", RoomFountain))
	assert.True(t, hasRoom("crypt", RoomOssuary))
	assert.True(t, hasRoom("crypt", RoomWarden))
	assert.True(t, hasRoom("fungal_wilds", RoomFountain))
	assert.True(t, hasRoom("fungal_wilds", RoomWarden))
	assert.True(t, hasRoom("forge_district", RoomArchive))
	assert.True(t, hasRoom("forge_district", RoomWarden))
	assert.True(t, hasRoom("abyss", RoomArchive))
	assert.True(t, hasRoom("abyss", RoomOssuary))
}

// TestNewRoomButtons validates that every new room type renders its expected
// action buttons.
func TestNewRoomButtons(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	cases := map[RoomType][]string{
		RoomArchive:  {"archive_read", "archive_search", "leave"},
		RoomFountain: {"fountain_coin", "fountain_drink", "leave"},
		RoomOssuary:  {"ossuary_search", "ossuary_rest", "leave"},
		RoomWarden:   {"warden_help", "warden_listen", "leave"},
	}
	for rt, want := range cases {
		btns := roomButtons(rt, rng)
		var actions []string
		for _, b := range btns {
			actions = append(actions, b.Action)
		}
		for _, a := range want {
			assert.Contains(t, actions, a, "room %s must offer %q", rt, a)
		}
	}
}

// TestRollCorridorEventIncludesNewEvents ensures the new bridge and mist
// corridor events are actually reachable.
func TestRollCorridorEventIncludesNewEvents(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	seen := map[CorridorEventType]bool{}
	for range 20000 {
		seen[RollCorridorEvent("crypt", 10, false, rng)] = true
	}
	assert.True(t, seen[CorridorBridge], "bridge event must be reachable")
	assert.True(t, seen[CorridorMist], "mist event must be reachable")
}

// TestWardenQuestFlagSet verifies the chronicle flag is recorded when a set
// piece enters the delve inventory.
func TestSetItemCollectedFlagOnAddItem(t *testing.T) {
	s := &model.DelveSession{Inventory: "[]", Flags: "[]"}
	svc := &Service{}
	svc.AddItem(s, DelveItem{ID: "delve_set_piece", Name: "Crypt Longsword", SetName: "Crypt Lord's Regalia", Rarity: Rare})
	assert.True(t, svc.HasFlag(s, "set_item_collected"))

	s2 := &model.DelveSession{Inventory: "[]", Flags: "[]"}
	svc.AddItem(s2, DelveItem{ID: "delve_plain", Name: "Longsword", Rarity: Rare})
	assert.False(t, svc.HasFlag(s2, "set_item_collected"))
}

func TestEndSessionRecordsFloorActivity(t *testing.T) {
	svc, st := delveTestService(t)

	// A quest waiting on delve_floors_cleared must advance from EndSession.
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "lost_warden", Status: "ACTIVE",
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "lost_warden", StepIndex: 1,
		ProgressValue: 0, CustomData: `{"target_stat":"delve_floors_cleared","target_count":5}`,
	}).Error)

	session := &model.DelveSession{
		UserID: 1, RoomsCleared: 6, Flags: "[]", Inventory: "[]",
	}
	require.NoError(t, svc.EndSession(session, "left"))

	var qd model.UserQuestData
	require.NoError(t, st.DB.Where("user_id = ? AND quest_id = ?", 1, "lost_warden").First(&qd).Error)
	assert.Equal(t, 6, qd.ProgressValue, "6 cleared rooms must count toward delve_floors_cleared")
}
