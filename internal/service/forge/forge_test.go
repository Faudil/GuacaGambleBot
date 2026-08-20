package forge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	furnituresvc "guacagamblebot/internal/service/furniture"
	housingsvc "guacagamblebot/internal/service/housing"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

func testService(t *testing.T) (*Service, *furnituresvc.Service, *housingsvc.Service, *store.Store) {
	t.Helper()
	d := testutil.NewDB(t)
	cfg := &config.Config{StartingBalance: 1000, DailyAmount: 50}
	s := store.New(d, cfg)
	hsvc := housingsvc.New(s, cfg)
	fsvc := furnituresvc.New(s, cfg, hsvc)
	return New(s, cfg), fsvc, hsvc, s
}

func buyHouse(t *testing.T, hsvc *housingsvc.Service, s *store.Store, userID int64) {
	t.Helper()
	_, err := s.UpdateBalance(userID, 1000000)
	require.NoError(t, err)
	require.NoError(t, hsvc.BuyHouse(userID, "mansion"))
}

func placeFurniture(t *testing.T, fsvc *furnituresvc.Service, s *store.Store, userID int64, furnitureID string) {
	t.Helper()
	for itemID, qty := range furnituresvc.FurnitureDefs[furnitureID].CostItems {
		require.NoError(t, s.AddItemRaw(s.DB, userID, itemID, qty))
	}
	require.NoError(t, fsvc.Place(userID, furnitureID))
}

func mustGear(t *testing.T, s *store.Store, userID int64, rarity, slot string, minLevel int) model.UserEquipment {
	t.Helper()
	eq, err := s.CreateEquipment(userID, "test_gear", "Test Gear", "⚔️", rarity, slot,
		minLevel, 0, 0, 0, 0, 0, []byte("[]"), "")
	require.NoError(t, err)
	return *eq
}

func countUserEquipment(t *testing.T, s *store.Store, userID int64) int {
	t.Helper()
	var n int64
	require.NoError(t, s.DB.Model(&model.UserEquipment{}).Where("user_id = ?", userID).Count(&n).Error)
	return int(n)
}

func completeResearch(t *testing.T, s *store.Store, userID int64, researchID string) {
	t.Helper()
	require.NoError(t, s.DB.Create(&model.UserResearch{
		UserID: userID, ResearchID: researchID, Completed: true,
	}).Error)
}

func TestResearchFor(t *testing.T) {
	assert.Equal(t, "fusion_common", ResearchFor(items.RarityCommon))
	assert.Equal(t, "fusion_uncommon", ResearchFor(items.RarityUncommon))
	assert.Equal(t, "fusion_rare", ResearchFor(items.RarityRare))
	assert.Equal(t, "fusion_epic", ResearchFor(items.RarityEpic))
	assert.Empty(t, ResearchFor(items.RarityLegendary))
}

func TestNextRarityChain(t *testing.T) {
	assert.Equal(t, items.RarityUncommon, mustNext(t, items.RarityCommon))
	assert.Equal(t, items.RarityRare, mustNext(t, items.RarityUncommon))
	assert.Equal(t, items.RarityEpic, mustNext(t, items.RarityRare))
	assert.Equal(t, items.RarityLegendary, mustNext(t, items.RarityEpic))
	_, ok := NextRarity(items.RarityLegendary)
	assert.False(t, ok, "legendary cannot be fused further")
}

func mustNext(t *testing.T, from items.Rarity) items.Rarity {
	t.Helper()
	to, ok := NextRarity(from)
	require.True(t, ok)
	return to
}

func TestRequiredFurniture(t *testing.T) {
	assert.Equal(t, "forge", RequiredFurniture(items.RarityCommon))
	assert.Equal(t, "forge", RequiredFurniture(items.RarityRare))
	assert.Equal(t, "arcane_forge", RequiredFurniture(items.RarityEpic))
}

func TestFuseConsumesFiveAndCreatesNextRarity(t *testing.T) {
	svc, fsvc, hsvc, s := testService(t)
	const uid = 1
	buyHouse(t, hsvc, s, uid)
	placeFurniture(t, fsvc, s, uid, "forge")
	completeResearch(t, s, uid, "fusion_common")

	var ids []uint
	for i := 0; i < 5; i++ {
		eq := mustGear(t, s, uid, "common", "weapon", 1)
		ids = append(ids, eq.ID)
	}

	created, err := svc.Fuse(uid, items.RarityCommon, ids)
	require.NoError(t, err)
	require.NotNil(t, created)

	assert.Equal(t, int64(uid), int64(created.UserID))
	assert.Equal(t, string(items.RarityUncommon), created.Rarity)
	assert.False(t, created.IsEquipped)
	assert.Contains(t, items.EquipSlots, created.EquipSlot)
	assert.Equal(t, items.MinLevelForRarity(items.RarityUncommon), created.MinLevel)
	assert.Equal(t, 1, countUserEquipment(t, s, uid), "5 items must be consumed, 1 created")
}

func TestFuseRequiresForgeFurniture(t *testing.T) {
	svc, fsvc, hsvc, s := testService(t)
	const uid = 1

	var ids []uint
	for i := 0; i < 5; i++ {
		eq := mustGear(t, s, uid, "common", "weapon", 1)
		ids = append(ids, eq.ID)
	}

	_, err := svc.Fuse(uid, items.RarityCommon, ids)
	assert.ErrorIs(t, err, ErrNoForge, "no house at all must be rejected")

	buyHouse(t, hsvc, s, uid)
	_, err = svc.Fuse(uid, items.RarityCommon, ids)
	assert.ErrorIs(t, err, ErrNoForge, "house without forge must be rejected")

	placeFurniture(t, fsvc, s, uid, "forge")
	_, err = svc.Fuse(uid, items.RarityEpic, ids)
	assert.ErrorIs(t, err, ErrNeedArcaneForge, "epic fusion needs the arcane forge")

	placeFurniture(t, fsvc, s, uid, "arcane_forge")
	_, err = svc.Fuse(uid, items.RarityEpic, ids)
	assert.ErrorIs(t, err, ErrResearchRequired, "epic fusion needs the fusion_epic research")

	completeResearch(t, s, uid, "fusion_epic")
	var epicIDs []uint
	for i := 0; i < 5; i++ {
		eq := mustGear(t, s, uid, "epic", "armor", 15)
		epicIDs = append(epicIDs, eq.ID)
	}
	_, err = svc.Fuse(uid, items.RarityEpic, epicIDs)
	assert.NoError(t, err, "arcane forge + fusion_epic research must allow epic fusion")
}

func TestFuseRequiresResearchPerTier(t *testing.T) {
	svc, fsvc, hsvc, s := testService(t)
	const uid = 1
	buyHouse(t, hsvc, s, uid)
	placeFurniture(t, fsvc, s, uid, "forge")

	var ids []uint
	for i := 0; i < 5; i++ {
		eq := mustGear(t, s, uid, "common", "weapon", 1)
		ids = append(ids, eq.ID)
	}

	_, err := svc.Fuse(uid, items.RarityCommon, ids)
	assert.ErrorIs(t, err, ErrResearchRequired, "forge alone must not unlock fusion")

	completeResearch(t, s, uid, "fusion_common")
	created, err := svc.Fuse(uid, items.RarityCommon, ids)
	require.NoError(t, err)
	assert.Equal(t, string(items.RarityUncommon), created.Rarity)
}

func TestFuseWithArcaneForge(t *testing.T) {
	svc, fsvc, hsvc, s := testService(t)
	const uid = 1
	buyHouse(t, hsvc, s, uid)
	placeFurniture(t, fsvc, s, uid, "arcane_forge")
	completeResearch(t, s, uid, "fusion_epic")

	var ids []uint
	for i := 0; i < 5; i++ {
		eq := mustGear(t, s, uid, "epic", "armor", 15)
		ids = append(ids, eq.ID)
	}

	created, err := svc.Fuse(uid, items.RarityEpic, ids)
	require.NoError(t, err)
	assert.Equal(t, string(items.RarityLegendary), created.Rarity)
}

func TestFuseRejectsEquippedItems(t *testing.T) {
	svc, fsvc, hsvc, s := testService(t)
	const uid = 1
	buyHouse(t, hsvc, s, uid)
	placeFurniture(t, fsvc, s, uid, "forge")
	completeResearch(t, s, uid, "fusion_common")

	var ids []uint
	for i := 0; i < 4; i++ {
		eq := mustGear(t, s, uid, "common", "weapon", 1)
		ids = append(ids, eq.ID)
	}
	eq := mustGear(t, s, uid, "common", "weapon", 1)
	require.NoError(t, s.EquipInstance(uid, eq.ID))
	ids = append(ids, eq.ID)

	_, err := svc.Fuse(uid, items.RarityCommon, ids)
	assert.ErrorIs(t, err, ErrEquippedItem)
}

func TestFuseRejectsForeignItem(t *testing.T) {
	svc, fsvc, hsvc, s := testService(t)
	const uid = 1
	buyHouse(t, hsvc, s, uid)
	placeFurniture(t, fsvc, s, uid, "forge")
	completeResearch(t, s, uid, "fusion_common")

	var ids []uint
	for i := 0; i < 4; i++ {
		eq := mustGear(t, s, uid, "common", "weapon", 1)
		ids = append(ids, eq.ID)
	}
	foreign := mustGear(t, s, 2, "common", "weapon", 1)
	ids = append(ids, foreign.ID)

	_, err := svc.Fuse(uid, items.RarityCommon, ids)
	assert.ErrorIs(t, err, ErrNotOwned)
}

func TestFuseRejectsWrongCount(t *testing.T) {
	svc, fsvc, hsvc, s := testService(t)
	const uid = 1
	buyHouse(t, hsvc, s, uid)
	placeFurniture(t, fsvc, s, uid, "forge")
	completeResearch(t, s, uid, "fusion_common")

	var ids []uint
	for i := 0; i < 4; i++ {
		eq := mustGear(t, s, uid, "common", "weapon", 1)
		ids = append(ids, eq.ID)
	}
	_, err := svc.Fuse(uid, items.RarityCommon, ids)
	assert.ErrorIs(t, err, ErrNeedFive)
}

func TestFuseRejectsWrongRarity(t *testing.T) {
	svc, fsvc, hsvc, s := testService(t)
	const uid = 1
	buyHouse(t, hsvc, s, uid)
	placeFurniture(t, fsvc, s, uid, "forge")
	completeResearch(t, s, uid, "fusion_common")

	var ids []uint
	for i := 0; i < 4; i++ {
		eq := mustGear(t, s, uid, "common", "weapon", 1)
		ids = append(ids, eq.ID)
	}
	other := mustGear(t, s, uid, "uncommon", "weapon", 5)
	ids = append(ids, other.ID)

	_, err := svc.Fuse(uid, items.RarityCommon, ids)
	assert.ErrorIs(t, err, ErrWrongRarity)
}

func TestScrapDeletesAndGrantsPoolResources(t *testing.T) {
	svc, _, _, s := testService(t)
	const uid = 1
	eq := mustGear(t, s, uid, "common", "weapon", 1)

	rewards, err := svc.Scrap(uid, eq.ID)
	require.NoError(t, err)
	require.NotEmpty(t, rewards)

	for itemID, qty := range rewards {
		assert.Greater(t, qty, 0)
		inPool := false
		for _, e := range scrapPools[items.RarityCommon] {
			if e.ItemID == itemID {
				inPool = true
				assert.LessOrEqual(t, qty, e.Max)
				break
			}
		}
		assert.True(t, inPool, "reward %s must come from the common pool", itemID)
	}
	assert.Equal(t, 0, countUserEquipment(t, s, uid), "scrapped item must be deleted")

	var inv model.Inventory
	err = s.DB.Where("user_id = ? AND item_id = ?", uid, "pebble").First(&inv).Error
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, inv.Quantity, 2)
}

func TestScrapRejectsEquippedAndForeign(t *testing.T) {
	svc, _, _, s := testService(t)
	const uid = 1

	eq := mustGear(t, s, uid, "common", "weapon", 1)
	require.NoError(t, s.EquipInstance(uid, eq.ID))
	_, err := svc.Scrap(uid, eq.ID)
	assert.ErrorIs(t, err, ErrEquippedItem)

	foreign := mustGear(t, s, 2, "common", "weapon", 1)
	_, err = svc.Scrap(uid, foreign.ID)
	assert.ErrorIs(t, err, ErrNotOwned)

	_, err = svc.Scrap(uid, 99999)
	assert.ErrorIs(t, err, ErrUnknownItem)
}

func TestGenerateFusedItemAlwaysValid(t *testing.T) {
	for i := 0; i < 200; i++ {
		for _, r := range RarityTiers {
			piece := generateFusedItem(r)
			assert.Contains(t, items.EquipSlots, piece.EquipSlot, "fused piece must use a canonical slot")
			assert.Equal(t, items.MinLevelForRarity(r), piece.MinLevel)
			assert.NotEmpty(t, piece.Name)
			assert.NotEmpty(t, piece.ID)
		}
	}
}
