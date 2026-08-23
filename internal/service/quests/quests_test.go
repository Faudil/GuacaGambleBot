package quests

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
	"guacagamblebot/internal/store"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "quest.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestGetQuestDef(t *testing.T) {
	svc, _ := testService(t)
	def := svc.GetQuestDef("tutorial")
	require.NotNil(t, def)
	assert.Equal(t, "main", def.Type)
	assert.Len(t, def.Steps, 32)
}

func TestGetQuestDefMissing(t *testing.T) {
	svc, _ := testService(t)
	assert.Nil(t, svc.GetQuestDef("nonexistent"))
}

func TestIsTutorialOnDelveStep(t *testing.T) {
	svc, st := testService(t)
	delveIdx := tutorialStepIdx(t, "quests.day7_delve.step1_activity")

	// No quest at all.
	assert.False(t, svc.IsTutorialOnDelveStep(1))

	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "tutorial", Status: "ACTIVE",
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "tutorial", StepIndex: delveIdx,
	}).Error)
	assert.True(t, svc.IsTutorialOnDelveStep(1), "on the delve step")

	// A different tutorial step is not the delve step.
	require.NoError(t, st.DB.Model(&model.UserQuestData{}).
		Where("user_id = 1 AND quest_id = 'tutorial'").
		Update("step_index", tutorialStepIdx(t, "quests.day1_strata.step1_activity")).Error)
	assert.False(t, svc.IsTutorialOnDelveStep(1))

	// A completed tutorial is not on the delve step.
	require.NoError(t, st.DB.Model(&model.UserQuest{}).
		Where("user_id = 1 AND quest_id = 'tutorial'").
		Update("status", "COMPLETED").Error)
	assert.False(t, svc.IsTutorialOnDelveStep(1))
}

func tutorialStepIdx(t *testing.T, key string) int {
	def := QuestRegistry["tutorial"]
	require.NotNil(t, def)
	for i, s := range def.Steps {
		if s.TextKey == key {
			return i
		}
	}
	t.Fatalf("step key not found in tutorial: %s", key)
	return -1
}

func TestTutorialOrderProgression(t *testing.T) {
	dig := tutorialStepIdx(t, "quests.day2_alchemy.step3_activity")
	hunt := tutorialStepIdx(t, "quests.day4_will.step1_activity")
	feed := tutorialStepIdx(t, "quests.day4_will.step3_activity")
	egg := tutorialStepIdx(t, "quests.day4_will.step0_dialogue")
	house := tutorialStepIdx(t, "quests.day3_base.step0_req")
	sell := tutorialStepIdx(t, "quests.day5_odds.step3_activity")
	delve := tutorialStepIdx(t, "quests.day7_delve.step1_activity")

	// Archeology must come after hunting and pet care.
	assert.Greater(t, dig, feed, "dig should come after pet feeding")
	assert.Greater(t, feed, hunt, "pet feeding should come after hunting")
	// The egg is granted before the hunting activity.
	assert.Less(t, egg, hunt, "egg should be granted before hunting")
	// Housing must come after selling items at the market.
	assert.Greater(t, house, sell, "housing requirement should come after selling at the market")
	// Delve stays near the end, after housing.
	assert.Greater(t, delve, house, "delve should come after housing")
}

func TestGetAllActiveQuestsEmpty(t *testing.T) {
	svc, _ := testService(t)
	quests, err := svc.GetAllActiveQuests(1)
	require.NoError(t, err)
	assert.Empty(t, quests)
}

func TestGetAllActiveQuestsWithData(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "tutorial", Status: "ACTIVE", StartedAt: now,
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "tutorial", StepIndex: 2, ProgressValue: 5,
	}).Error)

	quests, err := svc.GetAllActiveQuests(1)
	require.NoError(t, err)
	assert.Len(t, quests, 1)
	assert.Equal(t, 2, quests[0].StepIndex)
	assert.Equal(t, 5, quests[0].Progress)
	assert.Equal(t, 32, quests[0].TotalSteps)
}

func TestGetQuestProgressNotFound(t *testing.T) {
	svc, _ := testService(t)
	_, _, err := svc.GetQuestProgress(1, "tutorial")
	assert.Error(t, err)
}

func TestGetQuestProgressWithoutData(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "tutorial", Status: "ACTIVE", StartedAt: now,
	}).Error)

	uq, uqd, err := svc.GetQuestProgress(1, "tutorial")
	require.NoError(t, err)
	require.NotNil(t, uq)
	assert.Nil(t, uqd)
}

func TestGetQuestProgressWithData(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "tutorial", Status: "ACTIVE", StartedAt: now,
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "tutorial", StepIndex: 1, ProgressValue: 3,
	}).Error)

	uq, uqd, err := svc.GetQuestProgress(1, "tutorial")
	require.NoError(t, err)
	assert.Equal(t, "ACTIVE", uq.Status)
	require.NotNil(t, uqd)
	assert.Equal(t, 1, uqd.StepIndex)
}

func TestAdvanceStep(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "tutorial", Status: "ACTIVE", StartedAt: now,
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "tutorial", StepIndex: 0, ProgressValue: 0,
	}).Error)

	err := svc.AdvanceStep(1, "tutorial", "")
	require.NoError(t, err)

	_, uqd, err := svc.GetQuestProgress(1, "tutorial")
	require.NoError(t, err)
	require.NotNil(t, uqd)
	assert.Equal(t, 1, uqd.StepIndex)
	// CustomData should be set to the activity step's extra data
	assert.Contains(t, uqd.CustomData, `"target_stat":"items_mined"`)
	assert.Contains(t, uqd.CustomData, `"target_count":2`)
}

func TestAdvanceStepThenRecordActivity(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "tutorial", Status: "ACTIVE", StartedAt: now,
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "tutorial", StepIndex: 0, ProgressValue: 0,
	}).Error)

	// Advance from step 0 (dialogue) to step 1 (activity: mine 2)
	require.NoError(t, svc.AdvanceStep(1, "tutorial", ""))

	// Record mining activity
	require.NoError(t, st.RecordActivity(1, "items_mined", 1))

	_, uqd, err := svc.GetQuestProgress(1, "tutorial")
	require.NoError(t, err)
	require.NotNil(t, uqd)
	assert.Equal(t, 1, uqd.StepIndex, "should still be on step 1")
	assert.Equal(t, 1, uqd.ProgressValue, "progress should be 1 after mining 1")

	// Complete the activity step — RecordActivityComplete auto-advances
	require.NoError(t, st.RecordActivity(1, "items_mined", 1))

	_, uqd, err = svc.GetQuestProgress(1, "tutorial")
	require.NoError(t, err)
	require.NotNil(t, uqd)
	assert.Equal(t, 2, uqd.StepIndex, "should auto-advance to step 2 (dialogue)")
	assert.Equal(t, 0, uqd.ProgressValue, "progress should reset")
}

func TestAdvanceStepRewardsMoney(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "tutorial", Status: "ACTIVE", StartedAt: now,
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "tutorial", StepIndex: 0, ProgressValue: 0,
	}).Error)

	require.NoError(t, svc.AdvanceStep(1, "tutorial", ""))

	bal, _ := st.GetBalance(1)
	assert.Equal(t, 200, bal) // starting 100 + 100 reward
}

func TestAdvanceStepCompletesQuest(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "tutorial", Status: "ACTIVE", StartedAt: now,
	}).Error)
	// Start at step 31 (last step, index 31)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "tutorial", StepIndex: 31, ProgressValue: 0,
	}).Error)

	require.NoError(t, svc.AdvanceStep(1, "tutorial", ""))

	var uq model.UserQuest
	st.DB.Where("user_id = ? AND quest_id = ?", 1, "tutorial").First(&uq)
	assert.Equal(t, "COMPLETED", uq.Status)
}

func TestAdvanceStepGrantsFullRewards(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "tutorial", Status: "ACTIVE",
	}).Error)
	// Start on the finale step (index 31) which carries the full reward package.
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "tutorial", StepIndex: 31, ProgressValue: 0,
	}).Error)

	require.NoError(t, svc.AdvanceStep(1, "tutorial", ""))

	// Money: starting balance 100 + 1500.
	bal, err := st.GetBalance(1)
	require.NoError(t, err)
	assert.Equal(t, 1600, bal)

	// Crowns: 25.
	var crowns int
	require.NoError(t, st.DB.Raw("SELECT crowns FROM users WHERE user_id = ?", 1).Scan(&crowns).Error)
	assert.Equal(t, 25, crowns)

	// Achievement row granted.
	var achCount int64
	require.NoError(t, st.DB.Model(&model.UserAchievement{}).
		Where("user_id = 1 AND achievement_id = 'signal_complete'").Count(&achCount).Error)
	assert.Equal(t, int64(1), achCount)

	// Items: zenith_blade becomes real equipment, boss_trophy lands in inventory.
	var equipCount int64
	require.NoError(t, st.DB.Model(&model.UserEquipment{}).
		Where("user_id = 1 AND base_id = 'zenith_blade'").Count(&equipCount).Error)
	assert.Equal(t, int64(1), equipCount)
	var inv model.Inventory
	require.NoError(t, st.DB.Where("user_id = 1 AND item_id = 'boss_trophy'").First(&inv).Error)
	assert.Equal(t, 1, inv.Quantity)

	// Advancing again must not duplicate the achievement row.
	require.NoError(t, svc.AdvanceStep(1, "tutorial", ""))
	require.NoError(t, st.DB.Model(&model.UserAchievement{}).
		Where("user_id = 1 AND achievement_id = 'signal_complete'").Count(&achCount).Error)
	assert.Equal(t, int64(1), achCount, "achievement must not be granted twice")
}

func TestEnsureTutorialEggGrantsToStuckPlayer(t *testing.T) {
	svc, st := testService(t)
	// Player stuck at the hunting step with no pet and no egg.
	huntIdx := -1
	for i, s := range QuestRegistry["tutorial"].Steps {
		if s.TextKey == "quests.day4_will.step1_activity" {
			huntIdx = i
			break
		}
	}
	require.GreaterOrEqual(t, huntIdx, 0)
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "tutorial", Status: "ACTIVE",
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "tutorial", StepIndex: huntIdx,
	}).Error)

	granted, err := svc.EnsureTutorialEgg(1)
	require.NoError(t, err)
	assert.True(t, granted)

	var inv model.Inventory
	require.NoError(t, st.DB.Where("user_id = ? AND item_id = ?", 1, "forest_egg").First(&inv).Error)
	assert.Equal(t, 1, inv.Quantity)

	// Second call must not grant again.
	granted, err = svc.EnsureTutorialEgg(1)
	require.NoError(t, err)
	assert.False(t, granted)
}

func TestEnsureTutorialEggSkipsEarlySteps(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 2, QuestID: "tutorial", Status: "ACTIVE",
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 2, QuestID: "tutorial", StepIndex: 0,
	}).Error)

	granted, err := svc.EnsureTutorialEgg(2)
	require.NoError(t, err)
	assert.False(t, granted)
}

func TestEnsureTutorialEggSkipsWhenPetExists(t *testing.T) {
	svc, st := testService(t)
	huntIdx := -1
	for i, s := range QuestRegistry["tutorial"].Steps {
		if s.TextKey == "quests.day4_will.step1_activity" {
			huntIdx = i
			break
		}
	}
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 3, QuestID: "tutorial", Status: "ACTIVE",
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 3, QuestID: "tutorial", StepIndex: huntIdx,
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserPet{
		UserID: 3, PetType: "Dragon", Nickname: "Draco",
	}).Error)

	granted, err := svc.EnsureTutorialEgg(3)
	require.NoError(t, err)
	assert.False(t, granted)
}

// ─── Side quest lines (BossLossUnlocks) ───────────────────────

func TestIrianTrainingQuestDef(t *testing.T) {
	svc, _ := testService(t)
	def := svc.GetQuestDef("irian_training")
	require.NotNil(t, def)
	assert.Equal(t, "side", def.Type)
	assert.Equal(t, "irian", def.NPCID)
	require.Len(t, def.Steps, 6)
	assert.Equal(t, StepDialogue, def.Steps[0].Type)
	assert.Equal(t, StepActivity, def.Steps[1].Type)
	assert.Equal(t, StepDialogue, def.Steps[2].Type)
	assert.Equal(t, StepActivity, def.Steps[3].Type)
	assert.Equal(t, StepRequirement, def.Steps[4].Type)
	assert.Equal(t, StepDialogue, def.Steps[5].Type)
	// Activity steps target pets_fed and items_hunted.
	assert.Equal(t, "pets_fed", def.Steps[1].Extra["target_stat"])
	assert.Equal(t, "items_hunted", def.Steps[3].Extra["target_stat"])
	// Requirement step requires pet level 10.
	assert.Equal(t, 10, toInt(def.Steps[4].Extra["req_pet_level"]))
	// Final step rewards stat-boosting food.
	require.NotNil(t, def.Steps[5].Rewards)
	assert.Equal(t, 500, def.Steps[5].Rewards.Money)
	assert.Contains(t, def.Steps[5].Rewards.ItemIDs, "warrior_stew")
}

func TestUnlockOnBossLossStartsOnce(t *testing.T) {
	svc, st := testService(t)
	qid, newly := svc.UnlockOnBossLoss(1, 5)
	assert.Equal(t, "irian_training", qid)
	assert.True(t, newly)

	// Second loss must not restart an already active quest.
	qid, newly = svc.UnlockOnBossLoss(1, 5)
	assert.Equal(t, "", qid)
	assert.False(t, newly)

	var uq model.UserQuest
	require.NoError(t, st.DB.Where("user_id = ? AND quest_id = ?", 1, "irian_training").First(&uq).Error)
	assert.Equal(t, "ACTIVE", uq.Status)
}

func TestUnlockOnBossLossNoRegistry(t *testing.T) {
	svc, _ := testService(t)
	qid, newly := svc.UnlockOnBossLoss(1, 3)
	assert.Equal(t, "", qid)
	assert.False(t, newly)
}

func TestUnlockOnBossLossSkipsCompleted(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "irian_training", Status: "COMPLETED",
	}).Error)

	qid, newly := svc.UnlockOnBossLoss(1, 5)
	assert.Equal(t, "", qid)
	assert.False(t, newly)
}

// ─── req_pet_level requirement ─────────────────────────────────

func TestCheckRequirementPetLevelFails(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "irian_training", Status: "ACTIVE",
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "irian_training", StepIndex: 4,
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserPet{
		UserID: 1, PetType: "Dragon", Nickname: "Draco", Level: 7, IsActive: true,
	}).Error)

	err := svc.CheckRequirement(1, "irian_training")
	require.Error(t, err)
	var reqErr *RequirementError
	require.ErrorAs(t, err, &reqErr)
	assert.Equal(t, 10, reqErr.PetLevelNeeded)
	assert.Equal(t, 7, reqErr.PetLevelHave)
}

func TestCheckRequirementPetLevelPasses(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "irian_training", Status: "ACTIVE",
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "irian_training", StepIndex: 4,
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserPet{
		UserID: 1, PetType: "Dragon", Nickname: "Draco", Level: 12, IsActive: true,
	}).Error)

	assert.NoError(t, svc.CheckRequirement(1, "irian_training"))
}

func TestFulfillRequirementPetLevelAdvances(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "irian_training", Status: "ACTIVE",
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "irian_training", StepIndex: 4,
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserPet{
		UserID: 1, PetType: "Dragon", Nickname: "Draco", Level: 12, IsActive: true,
	}).Error)

	require.NoError(t, svc.FulfillRequirement(1, "irian_training"))

	_, uqd, err := svc.GetQuestProgress(1, "irian_training")
	require.NoError(t, err)
	require.NotNil(t, uqd)
	assert.Equal(t, 5, uqd.StepIndex)
}

func TestArenaRivalRegistry(t *testing.T) {
	svc, _ := testService(t)
	def := svc.GetQuestDef("arena_rival")
	require.NotNil(t, def)
	assert.Equal(t, "main", def.Type)
	require.Len(t, def.Steps, 9)

	assert.Equal(t, StepDialogue, def.Steps[0].Type)
	require.NotNil(t, def.Steps[0].Rewards)
	assert.Contains(t, def.Steps[0].Rewards.ItemIDs, "warrior_stew")

	assert.Equal(t, StepActivity, def.Steps[1].Type)
	assert.Equal(t, "pets_fed", def.Steps[1].Extra["target_stat"])
	assert.Equal(t, 1, def.Steps[1].Extra["target_count"])

	assert.Equal(t, StepDialogue, def.Steps[2].Type)

	assert.Equal(t, StepRequirement, def.Steps[3].Type)
	assert.Equal(t, 10, def.Steps[3].Extra["req_pet_level"])

	assert.Equal(t, StepDialogue, def.Steps[4].Type)

	assert.Equal(t, StepRequirement, def.Steps[5].Type)
	assert.Equal(t, 2, toInt(def.Steps[5].Extra["req_artifact_level"]))

	assert.Equal(t, StepRequirement, def.Steps[6].Type)
	assert.Equal(t, 1, toInt(def.Steps[6].Extra["req_artifact_points_spent"]))

	assert.Equal(t, StepBossBattle, def.Steps[7].Type)
	assert.Equal(t, 6, def.Steps[7].Extra["boss_stage"])
	require.NotNil(t, def.Steps[7].Rewards)
	assert.Equal(t, 2000, def.Steps[7].Rewards.Money)
	assert.Equal(t, 200, def.Steps[7].Rewards.XP)

	assert.Equal(t, StepDialogue, def.Steps[8].Type)
	require.NotNil(t, def.Steps[8].Rewards)
	assert.Contains(t, def.Steps[8].Rewards.ItemIDs, "gale_draught")
}

func TestGrantRewardsXP(t *testing.T) {
	svc, st := testService(t)
	_, err := st.EnsureCharacter(1)
	require.NoError(t, err)

	require.NoError(t, svc.grantRewards(1, &QuestReward{Money: 100, XP: 250}))

	var c model.UserCharacter
	require.NoError(t, st.DB.Where("user_id = ?", 1).First(&c).Error)
	assert.Equal(t, 250, c.XP, "character XP reward must be granted")
}

func TestRecordBossVictoryArenaRival(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, svc.StartQuest(1, "arena_rival"))
	require.NoError(t, st.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = ?", 1, "arena_rival").
		Update("step_index", 7).Error)

	require.NoError(t, svc.RecordBossVictory(1, 6))

	_, uqd, err := svc.GetQuestProgress(1, "arena_rival")
	require.NoError(t, err)
	require.NotNil(t, uqd)
	assert.Equal(t, 8, uqd.StepIndex, "winning Krag must advance to the outro step")

	bal, _ := st.GetBalance(1)
	assert.Equal(t, 2100, bal, "boss step reward must grant 2000 credits")

	var c model.UserCharacter
	require.NoError(t, st.DB.Where("user_id = ?", 1).First(&c).Error)
	assert.Equal(t, 200, c.XP, "boss step reward must grant character XP")
}

func TestLostWardenQuestFlow(t *testing.T) {
	svc, _ := testService(t)

	// Start on first meeting, advance through dialogue-only steps, never skip
	// activity steps.
	require.NoError(t, svc.StartQuest(1, "lost_warden"))
	_, uqd, err := svc.GetQuestProgress(1, "lost_warden")
	require.NoError(t, err)
	assert.Equal(t, 0, uqd.StepIndex)

	// The intro is a dialogue step: AdvanceIfDialogue should move to the
	// activity step.
	advanced, err := svc.AdvanceIfDialogue(1, "lost_warden")
	require.NoError(t, err)
	assert.True(t, advanced)
	_, uqd, err = svc.GetQuestProgress(1, "lost_warden")
	require.NoError(t, err)
	assert.Equal(t, 1, uqd.StepIndex)

	// On the activity step, AdvanceIfDialogue must refuse to skip it.
	advanced, err = svc.AdvanceIfDialogue(1, "lost_warden")
	require.NoError(t, err)
	assert.False(t, advanced)
	_, uqd, err = svc.GetQuestProgress(1, "lost_warden")
	require.NoError(t, err)
	assert.Equal(t, 1, uqd.StepIndex, "activity step must not be skipped")

	// StartQuest is a no-op when already active.
	assert.Error(t, svc.StartQuest(1, "lost_warden"))
	assert.True(t, svc.HasActiveQuest(1, "lost_warden"))
}

func TestLostWardenRewardItem(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, svc.StartQuest(1, "lost_warden"))
	// Jump to the final dialogue step.
	require.NoError(t, st.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = ?", 1, "lost_warden").
		Update("step_index", 4).Error)
	advanced, err := svc.AdvanceIfDialogue(1, "lost_warden")
	require.NoError(t, err)
	assert.True(t, advanced)

	uq, _, err := svc.GetQuestProgress(1, "lost_warden")
	require.NoError(t, err)
	assert.Equal(t, "COMPLETED", uq.Status)

	eq, err := st.GetAllUserEquipment(1)
	require.NoError(t, err)
	var found bool
	for _, e := range eq {
		if e.BaseID == "warden_badge" {
			found = true
		}
	}
	assert.True(t, found, "quest reward must grant the warden badge")
}

func TestChroniclerQuestSteps(t *testing.T) {
	svc, _ := testService(t)
	def := svc.GetQuestDef("chronicler_legend")
	require.NotNil(t, def)
	assert.Equal(t, "side", def.Type)
	assert.Len(t, def.Steps, 5)
	assert.Equal(t, "delve_completions", def.Steps[1].Extra["target_stat"])
	assert.Equal(t, "expedition_completions", def.Steps[2].Extra["target_stat"])
	assert.Equal(t, 4, def.Steps[3].Extra["boss_stage"])
	require.NotNil(t, def.Steps[4].Rewards)
	assert.Contains(t, def.Steps[4].Rewards.ItemIDs, "chronicler_relic")
	assert.Equal(t, "legend_unwritten", def.Steps[4].Rewards.AchievementID)
}

func TestCompletedMainQuestlinesCount(t *testing.T) {
	_, st := testService(t)
	assert.Equal(t, 0, CompletedMainQuestlines(st, 1))

	// Completed side quests don't count.
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "irian_training", Status: "COMPLETED",
	}).Error)
	assert.Equal(t, 0, CompletedMainQuestlines(st, 1))

	// Completed main questlines count, the tutorial excluded.
	for _, id := range []string{"tutorial", "masked_shadow_falls_hunter", "masked_shadow_falls_shadow"} {
		require.NoError(t, st.DB.Create(&model.UserQuest{
			UserID: 1, QuestID: id, Status: "COMPLETED",
		}).Error)
	}
	assert.Equal(t, 2, CompletedMainQuestlines(st, 1))

	// Active main quests don't count.
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "boss_league", Status: "ACTIVE",
	}).Error)
	assert.Equal(t, 2, CompletedMainQuestlines(st, 1))
}

func TestChroniclerQuestStartHelper(t *testing.T) {
	svc, st := testService(t)
	// StartQuestForUser works on a bare store.
	require.NoError(t, StartQuestForUser(st, 1, "chronicler_legend"))
	assert.True(t, svc.HasActiveQuest(1, "chronicler_legend"))
	// Second start is rejected (already active).
	assert.Error(t, StartQuestForUser(st, 1, "chronicler_legend"))
}

// ─── Questline guidance ─────────────────────────────────────────

func completeTutorial(t *testing.T, st *store.Store, userID int64) {
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: userID, QuestID: "tutorial", Status: "COMPLETED",
	}).Error)
}

func TestQuestlineOrderReservedEntries(t *testing.T) {
	assert.Contains(t, QuestlineOrder, "chronicler_legend")
	assert.Contains(t, QuestlineOrder, "elara_first_bloom")
	assert.NotContains(t, QuestlineOrder, "gamblebot_secret", "abandoned questline must not be listed")
	// No questline is implemented yet; the reserved ones stay out of the registry.
	assert.Nil(t, QuestRegistry["elara_first_bloom"])
}

func TestTutorialCompleted(t *testing.T) {
	_, st := testService(t)
	assert.False(t, TutorialCompleted(st, 1))
	completeTutorial(t, st, 1)
	assert.True(t, TutorialCompleted(st, 1))
}

func TestStarterQuestlineGate(t *testing.T) {
	_, st := testService(t)
	def := &QuestDef{ID: "reserved_starter", NPCID: "elara", Starter: true}

	assert.False(t, QuestlineUnlocked(st, 1, def), "locked while the tutorial runs")

	// Once the tutorial is done, a starter questline opens immediately.
	completeTutorial(t, st, 1)
	assert.True(t, QuestlineUnlocked(st, 1, def))
}

func TestAvailableQuestlinesAfterTutorial(t *testing.T) {
	_, st := testService(t)
	assert.Empty(t, AvailableQuestlines(st, 1), "nothing before the tutorial")

	// With no questline implemented yet, nothing is available after the
	// tutorial either — the chronicler stays gated.
	completeTutorial(t, st, 1)
	assert.Empty(t, AvailableQuestlines(st, 1))
	assert.Nil(t, SuggestedNext(st, 1))
}

func TestLockedQuestlinesIncludesChronicler(t *testing.T) {
	_, st := testService(t)
	completeTutorial(t, st, 1)

	locked := LockedQuestlines(st, 1)
	require.Len(t, locked, 1)
	assert.Equal(t, "chronicler_legend", locked[0].ID)
	assert.Equal(t, "quests.unlock_hint.chronicler", locked[0].HintKey)
}

func TestQuestlineOfferForNPC(t *testing.T) {
	_, st := testService(t)

	// No questline is implemented, so no NPC offers anything — before or
	// after the tutorial.
	assert.Nil(t, QuestlineOfferForNPC(st, 1, "gamblebot"))
	completeTutorial(t, st, 1)
	assert.Nil(t, QuestlineOfferForNPC(st, 1, "gamblebot"))
	assert.Nil(t, QuestlineOfferForNPC(st, 1, "elara"))
}

func TestQuestlineAffinityGate(t *testing.T) {
	_, st := testService(t)
	completeTutorial(t, st, 1)
	def := &QuestDef{ID: "thorek_heartstone", NPCID: "thorek", RepReq: 2}

	// No reputation yet.
	assert.False(t, QuestlineUnlocked(st, 1, def))
	// Affinity level 1 is not enough.
	require.NoError(t, st.DB.Create(&model.UserNPCReputation{
		UserID: 1, NPCID: "thorek", Reputation: 0, Level: 1,
	}).Error)
	assert.False(t, QuestlineUnlocked(st, 1, def))
	// Level 2 unlocks it.
	require.NoError(t, st.DB.Model(&model.UserNPCReputation{}).
		Where("user_id = 1 AND npc_id = 'thorek'").
		Update("level", 2).Error)
	assert.True(t, QuestlineUnlocked(st, 1, def))
}

func TestQuestlineBossGate(t *testing.T) {
	_, st := testService(t)
	completeTutorial(t, st, 1)
	def := &QuestDef{ID: "irian_leviathan", NPCID: "irian", RepReq: 2, BossReq: 3}
	require.NoError(t, st.DB.Create(&model.UserNPCReputation{
		UserID: 1, NPCID: "irian", Reputation: 0, Level: 2,
	}).Error)

	assert.False(t, QuestlineUnlocked(st, 1, def), "boss not yet defeated")

	// Beat the 3rd boss (battle step index 5 in the boss_league quest).
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "boss_league", Status: "ACTIVE",
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "boss_league", StepIndex: 5,
	}).Error)
	assert.False(t, QuestlineUnlocked(st, 1, def), "still ON the battle step")

	require.NoError(t, st.DB.Model(&model.UserQuestData{}).
		Where("user_id = 1 AND quest_id = 'boss_league'").
		Update("step_index", 6).Error)
	assert.True(t, QuestlineUnlocked(st, 1, def), "past the battle step")
}

func TestQuestlinePathGate(t *testing.T) {
	_, st := testService(t)
	completeTutorial(t, st, 1)
	shadowDef := &QuestDef{ID: "whisper_vault_contract", NPCID: "the_whisper", RepReq: 2, PathReq: "shadow"}
	require.NoError(t, st.DB.Create(&model.UserNPCReputation{
		UserID: 1, NPCID: "the_whisper", Reputation: 0, Level: 2,
	}).Error)

	assert.False(t, QuestlineUnlocked(st, 1, shadowDef), "no criminality alignment")
	require.NoError(t, st.DB.Create(&model.UserCriminality{
		UserID: 1, Alignment: "hunter",
	}).Error)
	assert.False(t, QuestlineUnlocked(st, 1, shadowDef), "hunter does not open the shadow questline")

	require.NoError(t, st.DB.Model(&model.UserCriminality{}).
		Where("user_id = 1").
		Update("alignment", "shadow").Error)
	assert.True(t, QuestlineUnlocked(st, 1, shadowDef))
}
