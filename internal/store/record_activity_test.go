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
			"step_index":    1,
			"progress_value": 0,
			"custom_data":   `{"target_count":10,"target_stat":"items_mined"}`,
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
	assert.Equal(t, 2, d.StepIndex, "step should advance to 2 after completing 10 mining")
	assert.Equal(t, 0, d.ProgressValue, "progress should reset to 0")
	assert.Equal(t, "{}", d.CustomData, "custom_data should be cleared")
}

func TestRecordActivityOnlyMatchesCorrectStat(t *testing.T) {
	s := newStore(t)

	require.NoError(t, s.CreateQuest(1, "tutorial"))

	require.NoError(t, s.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = ?", 1, "tutorial").
		Updates(map[string]any{
			"step_index":    1,
			"progress_value": 0,
			"custom_data":   `{"target_count":5,"target_stat":"items_mined"}`,
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
