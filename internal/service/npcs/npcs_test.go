package npcs

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/model"
	invsvc "guacagamblebot/internal/service/inventory"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
	"guacagamblebot/internal/universe/hoakhaven"
	"guacagamblebot/internal/universe/scifi"
	"guacagamblebot/internal/universe/scorch"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "npcs.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	require.NotNil(t, def)
	s := store.New(d, cfg)
	inv := invsvc.New(s, cfg)
	return New(s, cfg, def, inv), s
}

func testServiceWithUniverse(t *testing.T, universeID string) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "npcs_"+universeID+".db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50, Universe: universeID}
	switch universeID {
	case "scorch":
		scorch.Register()
	case "scifi":
		scifi.Register()
	default:
		hoakhaven.Register()
	}
	def := universe.Get(universeID)
	require.NotNil(t, def)
	s := store.New(d, cfg)
	inv := invsvc.New(s, cfg)
	return New(s, cfg, def, inv), s
}

func TestGetNPCData(t *testing.T) {
	svc, _ := testService(t)
	n := svc.GetNPCData("elara")
	require.NotNil(t, n)
	assert.Equal(t, "Elara", n.Name)
	assert.Equal(t, "🌿", n.Emoji)
}

func TestGetNPCDataMissing(t *testing.T) {
	svc, _ := testService(t)
	assert.Nil(t, svc.GetNPCData("nonexistent_npc"))
}

func TestGetAllNPCMeta(t *testing.T) {
	svc, _ := testService(t)
	all := svc.GetAllNPCMeta()
	assert.Len(t, all, len(svc.universe.NPCs))
}

func TestGetReputationCreatesDefault(t *testing.T) {
	svc, _ := testService(t)
	rep, err := svc.GetReputation(1, "elara")
	require.NoError(t, err)
	assert.Equal(t, int64(1), rep.UserID)
	assert.Equal(t, "elara", rep.NPCID)
	assert.Equal(t, 0, rep.Reputation)
	assert.Equal(t, 1, rep.Level)
}

func TestAddReputation(t *testing.T) {
	svc, _ := testService(t)
	added, err := svc.AddReputation(1, "elara", 50)
	require.NoError(t, err)
	assert.Equal(t, 50, added)

	rep, err := svc.GetReputation(1, "elara")
	require.NoError(t, err)
	assert.Equal(t, 50, rep.Reputation)
	assert.Equal(t, 1, rep.Level)
}

func TestAddReputationTriggersLevelUp(t *testing.T) {
	svc, _ := testService(t)
	added, err := svc.AddReputation(1, "elara", 100)
	require.NoError(t, err)
	assert.Equal(t, 100, added)

	rep, err := svc.GetReputation(1, "elara")
	require.NoError(t, err)
	assert.Equal(t, 0, rep.Reputation) // 100 - 100 = 0 after level up
	assert.Equal(t, 2, rep.Level)
}

func TestAddReputationExcessOverLevelUp(t *testing.T) {
	svc, _ := testService(t)
	added, err := svc.AddReputation(1, "elara", 250)
	require.NoError(t, err)
	assert.Equal(t, 250, added)

	rep, err := svc.GetReputation(1, "elara")
	require.NoError(t, err)
	assert.Equal(t, 150, rep.Reputation) // 250 - 100 (lvl1) = 150, not enough for lvl3 (needs 200)
	assert.Equal(t, 2, rep.Level)
}

func TestAddReputationRespectsDailyCap(t *testing.T) {
	svc, st := testService(t)
	today := time.Now().Format("2006-01-02")
	require.NoError(t, st.DB.Create(&model.UserNPCDailyRep{
		UserID: 1, NPCID: "elara", DateStr: today, Amount: 480,
	}).Error)

	added, err := svc.AddReputation(1, "elara", 50)
	require.NoError(t, err)
	assert.Equal(t, 20, added)
}

func TestGetAllReputations(t *testing.T) {
	svc, _ := testService(t)
	reps, err := svc.GetAllReputations(1)
	require.NoError(t, err)
	assert.Empty(t, reps)
}

func TestRankUp(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.UserNPCReputation{
		UserID: 1, NPCID: "elara", Reputation: 150, Level: 1,
	}).Error)

	err := svc.RankUp(1, "elara")
	require.NoError(t, err)

	rep, err := svc.GetReputation(1, "elara")
	require.NoError(t, err)
	assert.Equal(t, 2, rep.Level)
	assert.Equal(t, 0, rep.Reputation)
}

func TestRankUpNotEnough(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.UserNPCReputation{
		UserID: 1, NPCID: "elara", Reputation: 50, Level: 1,
	}).Error)

	err := svc.RankUp(1, "elara")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enough reputation")
}

func TestGetBonuses(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.UserNPCReputation{
		UserID: 1, NPCID: "gamblebot", Reputation: 0, Level: 3,
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserNPCReputation{
		UserID: 1, NPCID: "thorek", Reputation: 0, Level: 2,
	}).Error)

	b := svc.GetBonuses(1)
	assert.Equal(t, 6.0, b.ShopDiscount)      // level 3 * 2
	assert.Equal(t, 4, b.MiningRiskReduction) // level 2 * 2
}

func TestRankName(t *testing.T) {
	assert.Equal(t, "Inconnu", RankName(0))
	assert.Equal(t, "Inconnu", RankName(1))
	assert.Equal(t, "Connaissance", RankName(2))
	assert.Equal(t, "Partenaire", RankName(10))
}

func TestGetDailyRepCap(t *testing.T) {
	cap := GetDailyRepCap()
	assert.Equal(t, 500, cap.Flat)
	assert.Equal(t, 0, cap.PerLevel)
}

func TestChatCooldown(t *testing.T) {
	svc, _ := testService(t)
	svc.cfg.NPCChatCooldownHours = 6

	event, err := svc.Chat(1, "elara", "en")
	require.NoError(t, err)
	assert.Equal(t, 50, event.RepBonus)

	_, err = svc.Chat(1, "elara", "en")
	require.Error(t, err)
	var cde *ChatCooldownError
	require.ErrorAs(t, err, &cde)
	assert.True(t, time.Until(cde.Until) > 0)
}

func TestChatNoCooldown(t *testing.T) {
	svc, _ := testService(t)
	svc.cfg.NPCChatCooldownHours = 0

	_, err := svc.Chat(1, "elara", "en")
	require.NoError(t, err)
	_, err = svc.Chat(1, "elara", "en")
	require.NoError(t, err)
}

func TestChatDiminishingRewards(t *testing.T) {
	svc, _ := testService(t)
	svc.cfg.NPCChatCooldownHours = 0

	for i, want := range []int{50, 25, 10, 5} {
		event, err := svc.Chat(1, "elara", "en")
		require.NoError(t, err)
		assert.Equal(t, want, event.RepBonus, "chat #%d", i+1)
	}

	// Extra chats stay at the floor of 5.
	event, err := svc.Chat(1, "elara", "en")
	require.NoError(t, err)
	assert.Equal(t, 5, event.RepBonus)

	rep, err := svc.GetReputation(1, "elara")
	require.NoError(t, err)
	assert.Equal(t, 95, rep.Reputation) // 50+25+10+5+5, no level-up below 100
	assert.Equal(t, 1, rep.Level)
}

func TestChatSecretBonusOnce(t *testing.T) {
	svc, st := testService(t)
	svc.cfg.NPCChatCooldownHours = 0
	require.NoError(t, st.DB.Create(&model.UserNPCReputation{
		UserID: 1, NPCID: "elara", Reputation: 0, Level: 3,
	}).Error)

	event, err := svc.Chat(1, "elara", "en")
	require.NoError(t, err)
	assert.Equal(t, "secret_elara", event.ID)
	assert.Equal(t, 75, event.RepBonus) // 50 first chat + 25 secret

	event, err = svc.Chat(1, "elara", "en")
	require.NoError(t, err)
	assert.NotEqual(t, "secret_elara", event.ID)
	assert.Equal(t, 25, event.RepBonus)
}

func TestNPCIDForActivity(t *testing.T) {
	svc, _ := testService(t)
	assert.Equal(t, "thorek", svc.NPCIDForActivity("mining"))
	assert.Equal(t, "elara", svc.NPCIDForActivity("farming"))
	assert.Equal(t, "elara", svc.NPCIDForActivity("pets"))
	assert.Equal(t, "irian", svc.NPCIDForActivity("fishing"))
	assert.Equal(t, "irian", svc.NPCIDForActivity("hunting"))
	assert.Equal(t, "gamblebot", svc.NPCIDForActivity("gambling"))
	assert.Equal(t, "", svc.NPCIDForActivity("archeology")) // not in hoakhaven
	assert.Equal(t, "", svc.NPCIDForActivity("unknown"))
}

func TestNPCIDForActivityScorch(t *testing.T) {
	svc, _ := testServiceWithUniverse(t, "scorch")
	assert.Equal(t, "riggs", svc.NPCIDForActivity("hunting"))
	assert.Equal(t, "mother", svc.NPCIDForActivity("fishing"))
	assert.Equal(t, "", svc.NPCIDForActivity("gambling"))
}

func TestNPCIDForActivityScifi(t *testing.T) {
	svc, _ := testServiceWithUniverse(t, "scifi")
	assert.Equal(t, "zara", svc.NPCIDForActivity("archeology"))
	assert.Equal(t, "kellan", svc.NPCIDForActivity("mining"))
	assert.Equal(t, "okonkwo", svc.NPCIDForActivity("farming"))
	assert.Equal(t, "arcade", svc.NPCIDForActivity("gambling"))
}

func TestAddActivityReputation(t *testing.T) {
	svc, _ := testService(t)
	added, err := svc.AddActivityReputation(1, "mining", 3)
	require.NoError(t, err)
	assert.Equal(t, 3, added)

	rep, err := svc.GetReputation(1, "thorek")
	require.NoError(t, err)
	assert.Equal(t, 3, rep.Reputation)

	// Unknown activity (no linked NPC) is a silent no-op.
	added, err = svc.AddActivityReputation(1, "unknown", 3)
	require.NoError(t, err)
	assert.Equal(t, 0, added)
}

func TestChroniclerLockedWithoutRank(t *testing.T) {
	svc, _ := testService(t)
	svc.cfg.NPCChatCooldownHours = 0

	event, err := svc.Chat(1, "the_chronicler", "en")
	require.NoError(t, err)
	assert.Equal(t, "chronicler_locked", event.ID)
	assert.Equal(t, 0, event.RepBonus)

	// Chatting with another NPC still works and is independent.
	event, err = svc.Chat(1, "elara", "en")
	require.NoError(t, err)
	assert.NotNil(t, event)
}

func TestChroniclerIntroOnceThenQuips(t *testing.T) {
	svc, st := testService(t)
	svc.cfg.NPCChatCooldownHours = 0

	// Give the player rank 2 on the champion path (the Chronicler's reveal
	// threshold).
	require.NoError(t, st.DB.Create(&model.UserJournalEntry{UserID: 1, PathID: "champion", StepIndex: 4}).Error)

	event, err := svc.Chat(1, "the_chronicler", "en")
	require.NoError(t, err)
	assert.Equal(t, "chronicler_intro", event.ID)

	// Second chat: a regular quip, never the intro again.
	event, err = svc.Chat(1, "the_chronicler", "en")
	require.NoError(t, err)
	assert.Equal(t, "regular", event.ID)
}
