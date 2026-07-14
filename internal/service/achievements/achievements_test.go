package achievements

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

func testDB(t *testing.T) *gorm.DB {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	return d
}

func TestList(t *testing.T) {
	s := store.New(testDB(t), &config.Config{StartingBalance: 100, DailyAmount: 50})

	const userID int64 = 42
	require.NoError(t, s.DB.Create(&model.UserAchievement{UserID: userID, AchievementID: "daily_1"}).Error)
	require.NoError(t, s.DB.Create(&model.UserAchievement{UserID: userID, AchievementID: "eco_1k"}).Error)

	svc := New(s, &config.Config{})
	views, err := svc.List(userID)
	require.NoError(t, err)
	require.NotEmpty(t, views)

	byID := map[string]bool{}
	for _, v := range views {
		byID[v.ID] = v.Unlocked
	}

	assert.True(t, byID["daily_1"], "daily_1 should be unlocked")
	assert.True(t, byID["eco_1k"], "eco_1k should be unlocked")
	assert.False(t, byID["eco_10k"], "eco_10k should remain locked")

	var totalUnlocked int
	for _, v := range views {
		if v.Unlocked {
			totalUnlocked++
		}
	}
	assert.Equal(t, 2, totalUnlocked)

	// An unrelated user has nothing unlocked.
	other, err := svc.List(7)
	require.NoError(t, err)
	for _, v := range other {
		assert.False(t, v.Unlocked)
	}
}
