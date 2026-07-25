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
	assert.Len(t, def.Steps, 27)
}

func TestGetQuestDefMissing(t *testing.T) {
	svc, _ := testService(t)
	assert.Nil(t, svc.GetQuestDef("nonexistent"))
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
	assert.Equal(t, 27, quests[0].TotalSteps)
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
	// Start at step 26 (last step, index 26)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "tutorial", StepIndex: 26, ProgressValue: 0,
	}).Error)

	require.NoError(t, svc.AdvanceStep(1, "tutorial", ""))

	var uq model.UserQuest
	st.DB.Where("user_id = ? AND quest_id = ?", 1, "tutorial").First(&uq)
	assert.Equal(t, "COMPLETED", uq.Status)
}
