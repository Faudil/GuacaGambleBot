package lotto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d := testutil.NewDB(t)
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50, BaseJackpot: 500}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestBuyTicketInvalid(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.BuyTicket(1, 100, 0)
	assert.ErrorIs(t, err, ErrInvalidNum)
	_, err = svc.BuyTicket(1, 100, 101)
	assert.ErrorIs(t, err, ErrInvalidNum)
}

func TestBuyTicketNoMoney(t *testing.T) {
	svc, st := testService(t)
	// User 2 has only starting 100, but we don't add funds - they have 100
	// Ticket costs 20 so user can afford it. Create user with 0 balance.
	require.NoError(t, st.DB.Create(&model.User{UserID: 99}).Error)
	require.NoError(t, st.DB.Model(&model.User{}).Where("user_id = ?", 99).Update("balance", 0).Error)
	_, err := svc.BuyTicket(99, 100, 50)
	assert.ErrorIs(t, err, ErrNoMoney)
}

func TestBuyTicketDeductsBalance(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 50)
	require.NoError(t, err)

	res, err := svc.BuyTicket(1, 100, 50)
	require.NoError(t, err)
	assert.False(t, res.Win)
	assert.Equal(t, 50, res.Number)
	assert.GreaterOrEqual(t, res.WinningNum, 1)
	assert.GreaterOrEqual(t, res.Jackpot, 500)
	assert.Equal(t, 20, res.AddedValue)

	bal, _ := st.GetBalance(1)
	assert.Equal(t, 130, bal) // 100 starting + 50 - 20
}

func TestJackpot(t *testing.T) {
	svc, _ := testService(t)
	info, err := svc.Jackpot(100)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, info.Jackpot, 500)
	assert.GreaterOrEqual(t, info.WinningNumber, 1)
	assert.LessOrEqual(t, info.WinningNumber, 100)
}

func TestDailyBonus(t *testing.T) {
	svc, _ := testService(t)
	applied, err := svc.TryDailyBonus(100)
	require.NoError(t, err)
	assert.True(t, applied)

	applied2, err := svc.TryDailyBonus(100)
	require.NoError(t, err)
	assert.False(t, applied2)
}
