package character

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

func testStore(t *testing.T) *store.Store {
	d := testutil.NewDB(t)
	return store.New(d, &config.Config{StartingBalance: 100, DailyAmount: 50})
}

func TestProfile(t *testing.T) {
	s := testStore(t)
	const userID int64 = 42

	require.NoError(t, s.DB.Create(&model.User{UserID: userID}).Error)
	require.NoError(t, s.DB.Model(&model.User{}).Where("user_id = ?", userID).Update("balance", 250).Error)
	require.NoError(t, s.DB.Model(&model.User{}).Where("user_id = ?", userID).Update("crowns", 7).Error)
	require.NoError(t, s.DB.Create(&model.UserAchievement{UserID: userID, AchievementID: "first_win"}).Error)
	require.NoError(t, s.DB.Create(&model.UserAchievement{UserID: userID, AchievementID: "high_roller"}).Error)

	svc := New(s, &config.Config{})
	res, err := svc.Profile(userID)
	require.NoError(t, err)
	assert.Equal(t, 250, res.Wallet)
	assert.Equal(t, 0, res.Bank)
	assert.Equal(t, 7, res.Crowns)
	assert.Equal(t, 2, res.AchCount)
}
