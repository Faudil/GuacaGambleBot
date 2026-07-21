package casino

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/store"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "c.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestSlotsInvalid(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.SpinSlots(1, -5)
	assert.ErrorIs(t, err, ErrMaxBet)

	_, err = svc.SpinSlots(1, 0)
	assert.ErrorIs(t, err, ErrMaxBet)
}

func TestSlotsNoMoney(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.SpinSlots(1, 999999) // only 100 starting balance
	assert.ErrorIs(t, err, ErrNoMoney)
}

func TestSlotsWinOrLose(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 1000)
	require.NoError(t, err)

	res, err := svc.SpinSlots(1, 50)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Symbol1)
	assert.NotEmpty(t, res.Symbol2)
	assert.NotEmpty(t, res.Symbol3)
	assert.GreaterOrEqual(t, res.XpGain, 10)
}

func TestCoinflipInvalid(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.Coinflip(1, "invalid", 10, false)
	assert.ErrorIs(t, err, ErrChoice)

	_, err = svc.Coinflip(1, "heads", -5, false)
	assert.ErrorIs(t, err, ErrMaxBet)
}

func TestCoinflipNoMoney(t *testing.T) {
	svc, _ := testService(t)
	// amount must be <= 2000 for coinflip, but user only has 100
	_, err := svc.Coinflip(1, "heads", 2000, false)
	assert.ErrorIs(t, err, ErrNoMoney)
}

func TestCoinflipPlay(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 500)
	require.NoError(t, err)

	res, err := svc.Coinflip(1, "pile", 100, false)
	require.NoError(t, err)
	assert.Contains(t, []string{"pile", "face"}, res.Result)
}

func TestCoinflipChoiceNormalization(t *testing.T) {
	svc := &Service{}
	assert.Equal(t, "face", svc.normalizeChoice("heads"))
	assert.Equal(t, "pile", svc.normalizeChoice("tails"))
	assert.Equal(t, "face", svc.normalizeChoice("face"))
	assert.Equal(t, "pile", svc.normalizeChoice("pile"))
	assert.Equal(t, "", svc.normalizeChoice("invalid"))
}
