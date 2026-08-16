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
