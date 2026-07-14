package bank

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

func testStore(t *testing.T) *store.Store {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "x.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	return store.New(d, &config.Config{StartingBalance: 100, DailyAmount: 50})
}

func TestDeposit(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	wallet, bank, err := svc.Deposit(1, 40)
	require.NoError(t, err)
	assert.Equal(t, 60, wallet)
	assert.Equal(t, 40, bank)
}

func TestDepositInvalid(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	_, _, err := svc.Deposit(1, 0)
	assert.ErrorIs(t, err, ErrAmount)
	_, _, err = svc.Deposit(1, -5)
	assert.ErrorIs(t, err, ErrAmount)
}

func TestDepositInsufficient(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	_, _, err := svc.Deposit(1, 500)
	assert.ErrorIs(t, err, ErrNoMoney)
}

func TestWithdraw(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	_, _, err := svc.Deposit(1, 40)
	require.NoError(t, err)

	wallet, bank, err := svc.Withdraw(1, 15)
	require.NoError(t, err)
	assert.Equal(t, 75, wallet)
	assert.Equal(t, 25, bank)
}

func TestWithdrawInsufficient(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	_, _, err := svc.Deposit(1, 40)
	require.NoError(t, err)

	_, _, err = svc.Withdraw(1, 100)
	assert.ErrorIs(t, err, ErrNoMoney)
}

func TestInfo(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	_, _, err := svc.Deposit(1, 100)
	require.NoError(t, err)

	wallet, bank, interest, err := svc.Info(1)
	require.NoError(t, err)
	assert.Equal(t, 0, wallet)
	assert.Equal(t, 100, bank)
	assert.Equal(t, 10, interest)
}
