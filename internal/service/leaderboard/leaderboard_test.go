package leaderboard

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

func testStore(t *testing.T) *store.Store {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
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
