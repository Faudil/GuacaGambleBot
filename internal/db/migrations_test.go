package db

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/model"
)

func migrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "migrate.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, d.AutoMigrate(&model.UserQuest{}, &model.UserQuestData{}, &model.UserStat{}, &model.DataMigration{}))
	return d
}

func TestMigrateTutorialSteps(t *testing.T) {
	d := migrationTestDB(t)

	// ACTIVE tutorial players at various old step indices.
	oldSteps := map[int64]int{1: 0, 2: 7, 3: 12, 4: 13, 5: 19, 6: 30}
	for uid, step := range oldSteps {
		require.NoError(t, d.Create(&model.UserQuest{UserID: uid, QuestID: "tutorial", Status: "ACTIVE"}).Error)
		require.NoError(t, d.Create(&model.UserQuestData{UserID: uid, QuestID: "tutorial", StepIndex: step}).Error)
	}
	// A completed quest must not be remapped.
	require.NoError(t, d.Create(&model.UserQuest{UserID: 100, QuestID: "tutorial", Status: "COMPLETED"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 100, QuestID: "tutorial", StepIndex: 7}).Error)
	// An unrelated quest must not be touched.
	require.NoError(t, d.Create(&model.UserQuest{UserID: 200, QuestID: "boss_league", Status: "ACTIVE"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 200, QuestID: "boss_league", StepIndex: 7}).Error)

	require.NoError(t, migrateTutorialSteps(d))

	expected := map[int64]int{1: 0, 2: 12, 3: 7, 4: 8, 5: 14, 6: 31}
	for uid, want := range expected {
		var uqd model.UserQuestData
		require.NoError(t, d.Where("user_id = ? AND quest_id = 'tutorial'", uid).First(&uqd).Error)
		assert.Equal(t, want, uqd.StepIndex, "user %d", uid)
	}
	var completed model.UserQuestData
	require.NoError(t, d.Where("user_id = 100 AND quest_id = 'tutorial'").First(&completed).Error)
	assert.Equal(t, 7, completed.StepIndex, "completed quest must not be remapped")

	var other model.UserQuestData
	require.NoError(t, d.Where("user_id = 200 AND quest_id = 'boss_league'").First(&other).Error)
	assert.Equal(t, 7, other.StepIndex, "other quests must not be remapped")
}

func TestMigrateTutorialRewindSkippedHunt(t *testing.T) {
	d := migrationTestDB(t)

	// Skipped players: past the hunt block (step >= 11), no hunt wins.
	require.NoError(t, d.Create(&model.UserQuest{UserID: 1, QuestID: "tutorial", Status: "ACTIVE"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 1, QuestID: "tutorial", StepIndex: 12, ProgressValue: 1, CustomData: `{"target_stat":"items_digged"}`}).Error)
	require.NoError(t, d.Create(&model.UserQuest{UserID: 2, QuestID: "tutorial", Status: "ACTIVE"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 2, QuestID: "tutorial", StepIndex: 16}).Error)

	// Legit player: past the hunt block with hunt wins — must not move.
	require.NoError(t, d.Create(&model.UserQuest{UserID: 3, QuestID: "tutorial", Status: "ACTIVE"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 3, QuestID: "tutorial", StepIndex: 12}).Error)
	require.NoError(t, d.Create(&model.UserStat{UserID: 3, PveWins: 2}).Error)

	// Pre-hunt players and early steps — must not move.
	require.NoError(t, d.Create(&model.UserQuest{UserID: 4, QuestID: "tutorial", Status: "ACTIVE"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 4, QuestID: "tutorial", StepIndex: 6}).Error)
	require.NoError(t, d.Create(&model.UserQuest{UserID: 5, QuestID: "tutorial", Status: "ACTIVE"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 5, QuestID: "tutorial", StepIndex: 3}).Error)

	// Completed quest and unrelated quest — must not move.
	require.NoError(t, d.Create(&model.UserQuest{UserID: 100, QuestID: "tutorial", Status: "COMPLETED"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 100, QuestID: "tutorial", StepIndex: 12}).Error)
	require.NoError(t, d.Create(&model.UserQuest{UserID: 200, QuestID: "boss_league", Status: "ACTIVE"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 200, QuestID: "boss_league", StepIndex: 12}).Error)

	require.NoError(t, migrateTutorialRewindSkippedHunt(d))

	for _, tc := range []struct {
		uid  int64
		want int
	}{
		{1, 7}, {2, 7}, {3, 12}, {4, 6}, {5, 3}, {100, 12}, {200, 12},
	} {
		qid := "tutorial"
		if tc.uid == 200 {
			qid = "boss_league"
		}
		var uqd model.UserQuestData
		require.NoError(t, d.Where("user_id = ? AND quest_id = ?", tc.uid, qid).First(&uqd).Error)
		assert.Equal(t, tc.want, uqd.StepIndex, "user %d", tc.uid)
	}

	// Rewound rows have progress and custom data reset.
	var rewound model.UserQuestData
	require.NoError(t, d.Where("user_id = 1 AND quest_id = 'tutorial'").First(&rewound).Error)
	assert.Equal(t, 0, rewound.ProgressValue)
	assert.Equal(t, "{}", rewound.CustomData)
}

func TestRunDataMigrationsIdempotent(t *testing.T) {
	d := migrationTestDB(t)

	// First run applies the tutorial migrations and records the markers.
	require.NoError(t, runDataMigrations(d))
	for _, id := range []string{"tutorial_step_reorder", "tutorial_rewind_skipped_hunt", "job_xp_formula_rebalance"} {
		var count int64
		require.NoError(t, d.Model(&model.DataMigration{}).Where("id = ?", id).Count(&count).Error)
		assert.Equal(t, int64(1), count, "marker %s", id)
	}

	// Second run must be a no-op (markers present, nothing re-applied).
	require.NoError(t, runDataMigrations(d))
	for _, id := range []string{"tutorial_step_reorder", "tutorial_rewind_skipped_hunt", "job_xp_formula_rebalance"} {
		var count int64
		require.NoError(t, d.Model(&model.DataMigration{}).Where("id = ?", id).Count(&count).Error)
		assert.Equal(t, int64(1), count, "marker %s", id)
	}
}

func TestMigrateInventoryCanonicalIDs(t *testing.T) {
	d := migrationTestDB(t)
	require.NoError(t, d.AutoMigrate(&model.Inventory{}))

	// Name-keyed rows created by the legacy daily shop.
	require.NoError(t, d.Create(&model.Inventory{UserID: 1, ItemID: "Fertilizer", Quantity: 1}).Error)
	require.NoError(t, d.Create(&model.Inventory{UserID: 1, ItemID: "Steel Pickaxe", Quantity: 2}).Error)
	// A canonical row already exists for user 2 -> quantities must merge.
	require.NoError(t, d.Create(&model.Inventory{UserID: 2, ItemID: "Fertilizer", Quantity: 3}).Error)
	require.NoError(t, d.Create(&model.Inventory{UserID: 2, ItemID: "fertilizer", Quantity: 2}).Error)

	require.NoError(t, migrateInventoryCanonicalIDs(d))

	var fert model.Inventory
	require.NoError(t, d.Where("user_id = ? AND item_id = ?", 1, "fertilizer").First(&fert).Error)
	assert.Equal(t, 1, fert.Quantity)

	var merged model.Inventory
	require.NoError(t, d.Where("user_id = ? AND item_id = ?", 2, "fertilizer").First(&merged).Error)
	assert.Equal(t, 5, merged.Quantity, "name-keyed and canonical quantities must be merged")

	var pick model.Inventory
	require.NoError(t, d.Where("user_id = ? AND item_id = ?", 1, "steel_pickaxe").First(&pick).Error)
	assert.Equal(t, 2, pick.Quantity)

	var orphan model.Inventory
	err := d.Where("user_id = ? AND item_id = ?", 1, "Fertilizer").First(&orphan).Error
	assert.Error(t, err, "name-keyed rows must be removed")
}

func TestMigrateInventoryCleanupZeroQuantity(t *testing.T) {
	d := migrationTestDB(t)
	require.NoError(t, d.AutoMigrate(&model.Inventory{}))

	require.NoError(t, d.Create(&model.Inventory{UserID: 1, ItemID: "coal", Quantity: 0}).Error)
	require.NoError(t, d.Create(&model.Inventory{UserID: 1, ItemID: "beer", Quantity: -1}).Error)
	require.NoError(t, d.Create(&model.Inventory{UserID: 1, ItemID: "wheat", Quantity: 3}).Error)

	require.NoError(t, migrateInventoryCleanupZeroQuantity(d))

	var count int64
	require.NoError(t, d.Model(&model.Inventory{}).Count(&count).Error)
	assert.Equal(t, int64(1), count, "zero/negative quantity rows must be removed")

	var wheat model.Inventory
	require.NoError(t, d.Where("user_id = ? AND item_id = ?", 1, "wheat").First(&wheat).Error)
	assert.Equal(t, 3, wheat.Quantity)
}

func TestMigrateEquipSlotJewelry(t *testing.T) {
	d := migrationTestDB(t)
	require.NoError(t, d.AutoMigrate(&model.UserEquipment{}))

	mk := func(uid int64, baseID, slot string, equipped bool) {
		t.Helper()
		require.NoError(t, d.Create(&model.UserEquipment{
			UserID: uid, BaseID: baseID, EquipSlot: slot, IsEquipped: equipped,
		}).Error)
	}

	// Dragon Slayer scenario: ring + talisman both equipped (both become jewelry).
	mk(1, "golden_ring", "accessory", true)
	mk(1, "dragon_slayer_talisman", "trinket", true)
	// Collision: lucky_charm (accessory) + orb (trinket) both become trinket.
	mk(2, "lucky_charm", "accessory", true)
	mk(2, "arcane_weaver_orb", "trinket", true)
	// Delve drops store the generated base id, e.g. delve_<full generated name>.
	mk(3, "delve_flaming_arcane_orb_of_the_dying_star", "accessory", true)
	// Unequipped rows are reclassified too.
	mk(4, "ancient_amulet", "accessory", false)
	// Trinket rows are untouched.
	mk(5, "spark_shard", "trinket", false)

	require.NoError(t, migrateEquipSlotJewelry(d))

	slotOf := func(uid int64, baseID string) string {
		t.Helper()
		var eq model.UserEquipment
		require.NoError(t, d.Where("user_id = ? AND base_id = ?", uid, baseID).First(&eq).Error)
		return eq.EquipSlot
	}
	assert.Equal(t, "jewelry", slotOf(1, "golden_ring"))
	assert.Equal(t, "jewelry", slotOf(1, "dragon_slayer_talisman"))

	// Same-slot collisions keep the oldest equipped piece.
	var ring, talisman model.UserEquipment
	require.NoError(t, d.Where("user_id = 1 AND base_id = 'golden_ring'").First(&ring).Error)
	require.NoError(t, d.Where("user_id = 1 AND base_id = 'dragon_slayer_talisman'").First(&talisman).Error)
	assert.True(t, ring.IsEquipped, "oldest equipped piece must stay equipped")
	assert.False(t, talisman.IsEquipped, "second piece in the same slot must be unequipped")

	assert.Equal(t, "trinket", slotOf(2, "lucky_charm"))
	assert.Equal(t, "trinket", slotOf(2, "arcane_weaver_orb"))
	var charm, orb model.UserEquipment
	require.NoError(t, d.Where("user_id = 2 AND base_id = 'lucky_charm'").First(&charm).Error)
	require.NoError(t, d.Where("user_id = 2 AND base_id = 'arcane_weaver_orb'").First(&orb).Error)
	assert.True(t, charm.IsEquipped, "oldest equipped piece must stay equipped")
	assert.False(t, orb.IsEquipped, "second piece in the same slot must be unequipped")

	assert.Equal(t, "trinket", slotOf(3, "delve_flaming_arcane_orb_of_the_dying_star"))
	assert.Equal(t, "jewelry", slotOf(4, "ancient_amulet"))
	assert.Equal(t, "trinket", slotOf(5, "spark_shard"))
}

func TestMigrateJobXPFormulaRebalance(t *testing.T) {
	d := migrationTestDB(t)
	require.NoError(t, d.AutoMigrate(&model.Job{}))

	// Fresh level 1, no XP earned yet: nothing to convert.
	require.NoError(t, d.Create(&model.Job{UserID: 1, JobName: "miner", Level: 1, XP: 0}).Error)
	// Old formula: 75(L1)+100(L2)+125(L3)+25 = 325 total XP earned.
	// New formula: 100(L1)+150(L2) = 250 consumed, 75 left over, stuck at L3 (needs 200).
	require.NoError(t, d.Create(&model.Job{UserID: 2, JobName: "fisher", Level: 4, XP: 25}).Error)
	// Still mid-level-1 under the old formula (< 75 XP): level and XP unaffected.
	require.NoError(t, d.Create(&model.Job{UserID: 3, JobName: "farmer", Level: 1, XP: 40}).Error)

	require.NoError(t, migrateJobXPFormulaRebalance(d))

	get := func(uid int64, name string) model.Job {
		t.Helper()
		var j model.Job
		require.NoError(t, d.Where("user_id = ? AND job_name = ?", uid, name).First(&j).Error)
		return j
	}

	fresh := get(1, "miner")
	assert.Equal(t, 1, fresh.Level)
	assert.Equal(t, 0, fresh.XP)

	rebalanced := get(2, "fisher")
	assert.Equal(t, 3, rebalanced.Level)
	assert.Equal(t, 75, rebalanced.XP)

	untouched := get(3, "farmer")
	assert.Equal(t, 1, untouched.Level)
	assert.Equal(t, 40, untouched.XP)
}
