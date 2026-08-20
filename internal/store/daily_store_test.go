package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/model"
)

func dayOffset(days int) string {
	return time.Now().AddDate(0, 0, days).Format("2006-01-02")
}

func TestLogDailyQuestUpsertsPerDay(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.LogDailyQuest(1, "elara", "wheat"))
	require.NoError(t, s.LogDailyQuest(1, "thorek", "coal"))

	entries, err := s.RecentDailyQuests(1, 10)
	require.NoError(t, err)
	require.Len(t, entries, 1, "one row per user per day")
	assert.Equal(t, "thorek", entries[0].Requestor)
	assert.Equal(t, "coal", entries[0].TurnIn)
}

func TestCompleteDailyQuestMarksRow(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.LogDailyQuest(1, "elara", "wheat"))
	require.NoError(t, s.CompleteDailyQuest(1, "elara", "wheat"))

	entries, err := s.RecentDailyQuests(1, 5)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.True(t, entries[0].Completed)
}

func TestRecentDailyQuestsOrderAndLimit(t *testing.T) {
	s := newStore(t)
	for i := 0; i < 7; i++ {
		require.NoError(t, s.DB.Create(&model.UserDailyLog{
			UserID: 1, DateStr: dayOffset(-i), Requestor: "elara", TurnInItem: "wheat", Completed: true,
		}).Error)
	}
	entries, err := s.RecentDailyQuests(1, 3)
	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.Equal(t, dayOffset(0), entries[0].DateStr, "newest first")
	assert.Equal(t, dayOffset(-2), entries[2].DateStr)
}

func TestDailyStreak(t *testing.T) {
	s := newStore(t)
	assert.Equal(t, 0, s.DailyStreak(1), "no history")

	// Three consecutive completed days.
	for i := 0; i < 3; i++ {
		require.NoError(t, s.DB.Create(&model.UserDailyLog{
			UserID: 1, DateStr: dayOffset(-i), Requestor: "elara", TurnInItem: "wheat", Completed: true,
		}).Error)
	}
	assert.Equal(t, 3, s.DailyStreak(1))

	// A gap resets the streak (today and yesterday, then a miss).
	require.NoError(t, s.DB.Create(&model.UserDailyLog{
		UserID: 2, DateStr: dayOffset(0), Requestor: "elara", TurnInItem: "wheat", Completed: true,
	}).Error)
	require.NoError(t, s.DB.Create(&model.UserDailyLog{
		UserID: 2, DateStr: dayOffset(-2), Requestor: "elara", TurnInItem: "wheat", Completed: true,
	}).Error)
	assert.Equal(t, 1, s.DailyStreak(2), "yesterday missed breaks the chain")

	// An in-progress today keeps the streak alive through yesterday.
	require.NoError(t, s.DB.Create(&model.UserDailyLog{
		UserID: 3, DateStr: dayOffset(-1), Requestor: "elara", TurnInItem: "wheat", Completed: true,
	}).Error)
	require.NoError(t, s.DB.Create(&model.UserDailyLog{
		UserID: 3, DateStr: dayOffset(-2), Requestor: "elara", TurnInItem: "wheat", Completed: true,
	}).Error)
	assert.Equal(t, 2, s.DailyStreak(3), "unfinished today carries yesterday's streak")
}
