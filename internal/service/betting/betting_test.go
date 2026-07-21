package betting

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

func testService(t *testing.T) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "b.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestCreateBet(t *testing.T) {
	svc, _ := testService(t)
	id, err := svc.CreateBet(1, "Test bet?", "Yes", "No")
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))
}

func TestPlaceBet(t *testing.T) {
	svc, st := testService(t)
	id, err := svc.CreateBet(1, "Test bet?", "Yes", "No")
	require.NoError(t, err)

	_, err = st.UpdateBalance(2, 500)
	require.NoError(t, err)

	err = svc.PlaceBet(2, id, "a", 100)
	require.NoError(t, err)
}

func TestPlaceBetNoMoney(t *testing.T) {
	svc, st := testService(t)
	id, err := svc.CreateBet(1, "Test?", "A", "B")
	require.NoError(t, err)

	// User 99 with balance 0
	require.NoError(t, st.DB.Create(&model.User{UserID: 99}).Error)
	require.NoError(t, st.DB.Model(&model.User{}).Where("user_id = ?", 99).Update("balance", 0).Error)
	err = svc.PlaceBet(99, id, "a", 50)
	assert.ErrorIs(t, err, ErrNoMoney)
}

func TestPlaceBetClosed(t *testing.T) {
	svc, st := testService(t)
	id, err := svc.CreateBet(1, "Test?", "A", "B")
	require.NoError(t, err)

	_, err = st.UpdateBalance(2, 500)
	require.NoError(t, err)

	_, err = svc.CloseBet(1, id, "a")
	require.NoError(t, err)

	err = svc.PlaceBet(2, id, "a", 50)
	assert.ErrorIs(t, err, ErrClosed)
}

func TestCloseBetPayout(t *testing.T) {
	svc, st := testService(t)
	id, err := svc.CreateBet(1, "Who wins?", "Team A", "Team B")
	require.NoError(t, err)

	_, err = st.UpdateBalance(2, 1000)
	require.NoError(t, err)
	_, err = st.UpdateBalance(3, 500)
	require.NoError(t, err)

	require.NoError(t, svc.PlaceBet(2, id, "a", 200))
	require.NoError(t, svc.PlaceBet(3, id, "a", 100))

	res, err := svc.CloseBet(1, id, "a")
	require.NoError(t, err)
	assert.Equal(t, 300, res.TotalPool)
	assert.Equal(t, 300, res.WinningPool)

	b2, _ := st.GetBalance(2)
	_ = b2
}

func TestFreezeBet(t *testing.T) {
	svc, st := testService(t)
	id, err := svc.CreateBet(1, "Test?", "A", "B")
	require.NoError(t, err)

	err = svc.FreezeBet(1, id)
	require.NoError(t, err)

	_, err = st.UpdateBalance(2, 500)
	require.NoError(t, err)

	err = svc.PlaceBet(2, id, "a", 50)
	assert.ErrorIs(t, err, ErrFrozen)
}

func TestShowOdds(t *testing.T) {
	svc, _ := testService(t)
	id, err := svc.CreateBet(1, "Test?", "Option1", "Option2")
	require.NoError(t, err)

	odds, err := svc.ShowOdds(id)
	require.NoError(t, err)
	assert.Equal(t, "Option1", odds.Option1)
	assert.Equal(t, "Option2", odds.Option2)
}
