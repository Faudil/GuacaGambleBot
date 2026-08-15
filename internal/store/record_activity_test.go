package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/model"
)

func TestRecordActivityTutorialQuest(t *testing.T) {
	s := newStore(t)

	require.NoError(t, s.CreateQuest(1, "tutorial"))

	// Simulate AdvanceStep: set step to 1 (activity: mine 10) with custom_data
	require.NoError(t, s.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = ?", 1, "tutorial").
		Updates(map[string]any{
			"step_index":     1,
			"progress_value": 0,
			"custom_data":    `{"target_count":10,"target_stat":"items_mined"}`,
		}).Error)

	// Record one mining activity
	require.NoError(t, s.RecordActivity(1, "items_mined", 1))

	var d model.UserQuestData
	require.NoError(t, s.DB.Where("user_id = ? AND quest_id = ?", 1, "tutorial").First(&d).Error)
	assert.Equal(t, 1, d.ProgressValue, "progress should be 1 after one mining action")
	assert.Equal(t, 1, d.StepIndex, "step should still be at activity step")

	// Record 9 more to reach 10/10 and complete the step
	for i := 0; i < 9; i++ {
		require.NoError(t, s.RecordActivity(1, "items_mined", 1))
	}

	require.NoError(t, s.DB.Where("user_id = ? AND quest_id = ?", 1, "tutorial").First(&d).Error)
	assert.Equal(t, 1, d.StepIndex, "step should stay at activity step (advance on Continue click)")
	assert.Equal(t, 10, d.ProgressValue, "progress should be at target count")
	assert.Equal(t, `{"target_count":10,"target_stat":"items_mined"}`, d.CustomData, "custom_data should keep activity config")

	var uq model.UserQuest
	require.NoError(t, s.DB.Where("user_id = ? AND quest_id = ?", 1, "tutorial").First(&uq).Error)
	assert.Equal(t, "ACTIVE", uq.Status, "tutorial must stay ACTIVE after an activity target is reached")
}

func TestRecordActivityTutorialStaysActiveWithHook(t *testing.T) {
	s := newStore(t)
	s.SetQuestAdvanceFn(func(userID int64, questID string) (bool, string, error) {
		return false, "quests.day1_strata.step2_transition", nil
	})

	require.NoError(t, s.CreateQuest(1, "tutorial"))
	require.NoError(t, s.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = ?", 1, "tutorial").
		Updates(map[string]any{
			"step_index":     1,
			"progress_value": 0,
			"custom_data":    `{"target_count":1,"target_stat":"items_mined"}`,
		}).Error)

	require.NoError(t, s.RecordActivity(1, "items_mined", 1))

	var uq model.UserQuest
	require.NoError(t, s.DB.Where("user_id = ? AND quest_id = ?", 1, "tutorial").First(&uq).Error)
	assert.Equal(t, "ACTIVE", uq.Status, "hook that advances a step must not complete the quest")

	n, ok := s.PopQuestNotification(1)
	require.True(t, ok)
	assert.False(t, n.Completed)
	assert.Equal(t, "quests.day1_strata.step2_transition", n.NextStepKey)
}

func TestRecordActivityTutorialCompletionDelegatedToHook(t *testing.T) {
	s := newStore(t)
	s.SetQuestAdvanceFn(func(userID int64, questID string) (bool, string, error) {
		return true, "", nil
	})

	require.NoError(t, s.CreateQuest(1, "tutorial"))
	require.NoError(t, s.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = ?", 1, "tutorial").
		Updates(map[string]any{
			"step_index":     1,
			"progress_value": 0,
			"custom_data":    `{"target_count":1,"target_stat":"items_mined"}`,
		}).Error)

	require.NoError(t, s.RecordActivity(1, "items_mined", 1))

	var uq model.UserQuest
	require.NoError(t, s.DB.Where("user_id = ? AND quest_id = ?", 1, "tutorial").First(&uq).Error)
	assert.Equal(t, "ACTIVE", uq.Status, "completing the quest is the hook's responsibility, not the store's")

	n, ok := s.PopQuestNotification(1)
	require.True(t, ok)
	assert.True(t, n.Completed)
}

func TestRecordActivityOnlyMatchesCorrectStat(t *testing.T) {
	s := newStore(t)

	require.NoError(t, s.CreateQuest(1, "tutorial"))

	require.NoError(t, s.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = ?", 1, "tutorial").
		Updates(map[string]any{
			"step_index":     1,
			"progress_value": 0,
			"custom_data":    `{"target_count":5,"target_stat":"items_mined"}`,
		}).Error)

	// Record farming activity — should NOT match mining quest
	require.NoError(t, s.RecordActivity(1, "items_farmed", 10))

	var d model.UserQuestData
	require.NoError(t, s.DB.Where("user_id = ? AND quest_id = ?", 1, "tutorial").First(&d).Error)
	assert.Equal(t, 0, d.ProgressValue, "farming should not update mining progress")
}

func TestRecordActivityIgnoresCompletedQuest(t *testing.T) {
	s := newStore(t)

	require.NoError(t, s.CreateQuest(1, "tutorial"))

	// Complete the quest
	require.NoError(t, s.DB.Model(&model.UserQuest{}).
		Where("user_id = ? AND quest_id = ?", 1, "tutorial").
		Update("status", "COMPLETED").Error)

	// Recording activity should not error
	require.NoError(t, s.RecordActivity(1, "items_mined", 5))
}

func TestRecordActivityPushesStepAdvanceNotification(t *testing.T) {
	s := newStore(t)
	s.SetQuestAdvanceFn(func(userID int64, questID string) (bool, string, error) {
		return false, "quests.day1_strata.step2_transition", nil
	})

	require.NoError(t, s.CreateQuest(1, "tutorial"))
	require.NoError(t, s.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = ?", 1, "tutorial").
		Updates(map[string]any{
			"step_index":     1,
			"progress_value": 0,
			"custom_data":    `{"target_count":1,"target_stat":"items_mined"}`,
		}).Error)

	require.NoError(t, s.RecordActivity(1, "items_mined", 1))

	n, ok := s.PopQuestNotification(1)
	require.True(t, ok)
	assert.Equal(t, "tutorial", n.QuestID)
	assert.False(t, n.Completed)
	assert.Equal(t, "quests.day1_strata.step2_transition", n.NextStepKey)

	_, ok = s.PopQuestNotification(1)
	assert.False(t, ok, "notification should be consumed")
}

func TestRecordActivityPushesCompletionNotification(t *testing.T) {
	s := newStore(t)
	s.SetQuestAdvanceFn(func(userID int64, questID string) (bool, string, error) {
		return true, "", nil
	})

	require.NoError(t, s.CreateQuest(1, "tutorial"))
	require.NoError(t, s.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = ?", 1, "tutorial").
		Updates(map[string]any{
			"step_index":     1,
			"progress_value": 0,
			"custom_data":    `{"target_count":1,"target_stat":"items_mined"}`,
		}).Error)

	require.NoError(t, s.RecordActivity(1, "items_mined", 1))

	n, ok := s.PopQuestNotification(1)
	require.True(t, ok)
	assert.True(t, n.Completed)
	assert.Equal(t, "tutorial", n.QuestID)
}

func TestPopQuestNotificationQueuesMultiple(t *testing.T) {
	s := newStore(t)
	s.pushQuestNotification(1, QuestNotification{QuestID: "a", Completed: true})
	s.pushQuestNotification(1, QuestNotification{QuestID: "b", Completed: false, NextStepKey: "k"})

	n1, ok := s.PopQuestNotification(1)
	require.True(t, ok)
	assert.Equal(t, "a", n1.QuestID)

	n2, ok := s.PopQuestNotification(1)
	require.True(t, ok)
	assert.Equal(t, "b", n2.QuestID)
	assert.Equal(t, "k", n2.NextStepKey)

	_, ok = s.PopQuestNotification(1)
	assert.False(t, ok)
}

func TestDailyQuestCompletionGrantsEgg(t *testing.T) {
	s := newStore(t)

	require.NoError(t, s.CreateQuest(1, "daily_quest"))
	require.NoError(t, s.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = ?", 1, "daily_quest").
		Updates(map[string]any{
			"step_index":     0,
			"progress_value": 0,
			"custom_data":    `{"target_count":1,"target_stat":"items_mined"}`,
		}).Error)

	require.NoError(t, s.RecordActivity(1, "items_mined", 1))

	n, ok := s.PopQuestNotification(1)
	require.True(t, ok)
	assert.True(t, n.Completed)

	var inv model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "forest_egg").First(&inv).Error)
	assert.Equal(t, 1, inv.Quantity)
}
