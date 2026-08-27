package leaderboard

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

func TestTop(t *testing.T) {
	s := testStore(t)

	require.NoError(t, s.DB.Create(&model.User{UserID: 10}).Error)
	require.NoError(t, s.DB.Create(&model.User{UserID: 20}).Error)
	require.NoError(t, s.DB.Create(&model.User{UserID: 30}).Error)
	require.NoError(t, s.DB.Create(&model.User{UserID: 40}).Error)
	require.NoError(t, s.DB.Model(&model.User{}).Where("user_id = ?", 10).Update("balance", 500).Error)
	require.NoError(t, s.DB.Model(&model.User{}).Where("user_id = ?", 20).Update("balance", 1500).Error)
	require.NoError(t, s.DB.Model(&model.User{}).Where("user_id = ?", 30).Update("balance", 100).Error)
	require.NoError(t, s.DB.Model(&model.User{}).Where("user_id = ?", 40).Update("balance", 800).Error)

	svc := New(s, &config.Config{})
	users, err := svc.Top(3)
	require.NoError(t, err)
	require.Len(t, users, 3)
	assert.Equal(t, int64(20), users[0].UserID)
	assert.Equal(t, 1500, users[0].Balance)
	assert.Equal(t, int64(40), users[1].UserID)
	assert.Equal(t, 800, users[1].Balance)
	assert.Equal(t, int64(10), users[2].UserID)
	assert.Equal(t, 500, users[2].Balance)
}

func TestTopWinRecords(t *testing.T) {
	s := testStore(t)

	require.NoError(t, s.AddWinRecord(10, "slots", 100))
	require.NoError(t, s.AddWinRecord(20, "slots", 5000))
	require.NoError(t, s.AddWinRecord(30, "slots", 200))
	require.NoError(t, s.AddWinRecord(40, "slots", 200))
	require.NoError(t, s.AddWinRecord(50, "coinflip", 2000))

	svc := New(s, &config.Config{})
	records, err := svc.TopWinRecords("slots", 3)
	require.NoError(t, err)
	require.Len(t, records, 3)
	assert.Equal(t, int64(20), records[0].UserID)
	assert.Equal(t, 5000, records[0].Amount)
	assert.Equal(t, int64(30), records[1].UserID)
	assert.Equal(t, 200, records[1].Amount)
	assert.Equal(t, int64(40), records[2].UserID)
	assert.Equal(t, 200, records[2].Amount)

	cf, err := svc.TopWinRecords("coinflip", 10)
	require.NoError(t, err)
	require.Len(t, cf, 1)
	assert.Equal(t, int64(50), cf[0].UserID)
	assert.Equal(t, 2000, cf[0].Amount)
}
