package store

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
)

func testDB(t *testing.T) *gorm.DB {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	return d
}

func newStore(t *testing.T) *Store {
	return New(testDB(t), &config.Config{StartingBalance: 100, DailyAmount: 50})
}

func TestBalanceMath(t *testing.T) {
	s := newStore(t)

	bal, err := s.GetBalance(1)
	require.NoError(t, err)
	assert.Equal(t, 100, bal, "new user starts at starting balance")

	bal, err = s.UpdateBalance(1, 50)
	require.NoError(t, err)
	assert.Equal(t, 150, bal)

	// Negative adjustment never drops below zero logic is preserved (here the
	// user has funds, so it subtracts correctly).
	bal, err = s.UpdateBalance(1, -200)
	require.NoError(t, err)
	assert.Equal(t, -50, bal)
}

func TestBankData(t *testing.T) {
	s := newStore(t)
	_, err := s.UpdateBalance(1, 300)
	require.NoError(t, err)
	wallet, bank, err := s.GetBankData(1)
	require.NoError(t, err)
	assert.Equal(t, 400, wallet)
	assert.Equal(t, 0, bank)
}

func TestRepayDebt(t *testing.T) {
	s := newStore(t)
	// Pre-create accounts with a known balance. A direct Create ignores a
	// zero-value Balance, so set the balance explicitly afterwards.
	require.NoError(t, s.DB.Create(&model.User{UserID: 10}).Error)
	require.NoError(t, s.DB.Create(&model.User{UserID: 20}).Error)
	require.NoError(t, s.DB.Model(&model.User{}).Where("user_id = ?", 10).Update("balance", 1000).Error)
	require.NoError(t, s.DB.Model(&model.User{}).Where("user_id = ?", 20).Update("balance", 0).Error)

	require.NoError(t, s.DB.Create(&model.Loan{BorrowerID: 10, LenderID: 20, AmountDue: 300}).Error)

	debt, err := s.GetTotalDebt(10)
	require.NoError(t, err)
	assert.Equal(t, 300, debt)

	paid, _, err := s.RepayDebt(10, 100)
	require.NoError(t, err)
	assert.Equal(t, 100, paid)

	borrowerBal, _ := s.GetBalance(10)
	lenderBal, _ := s.GetBalance(20)
	assert.Equal(t, 900, borrowerBal)
	assert.Equal(t, 100, lenderBal)

	debt, _ = s.GetTotalDebt(10)
	assert.Equal(t, 200, debt)

	// Repay the rest: the loan row must be removed.
	paid, _, err = s.RepayDebt(10, 200)
	require.NoError(t, err)
	assert.Equal(t, 200, paid)
	debt, _ = s.GetTotalDebt(10)
	assert.Equal(t, 0, debt)

	borrowerBal, _ = s.GetBalance(10)
	lenderBal, _ = s.GetBalance(20)
	assert.Equal(t, 700, borrowerBal)
	assert.Equal(t, 300, lenderBal)
}

func TestGameLimit(t *testing.T) {
	s := newStore(t)
	ok, remaining, err := s.CheckGameLimit(1, "daily", 1)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 1, remaining)

	require.NoError(t, s.IncrementGameLimit(1, "daily"))
	ok, remaining, err = s.CheckGameLimit(1, "daily", 1)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, 0, remaining)
}

func TestDailyQuest(t *testing.T) {
	s := newStore(t)
	has, err := s.HasDailyQuestToday(1)
	require.NoError(t, err)
	assert.False(t, has)

	require.NoError(t, s.StartDailyQuest(1, "blackjack_won", 3, "quests.daily.blackjack"))
	has, err = s.HasDailyQuestToday(1)
	require.NoError(t, err)
	assert.True(t, has)

	require.NoError(t, s.RecordActivity(1, "blackjack_won", 2))
	// not yet complete
	var q model.UserQuest
	require.NoError(t, s.DB.Where("user_id = ? AND quest_id = 'daily_quest'", 1).First(&q).Error)
	assert.Equal(t, "ACTIVE", q.Status)

	require.NoError(t, s.RecordActivity(1, "blackjack_won", 1))
	require.NoError(t, s.DB.Where("user_id = ? AND quest_id = 'daily_quest'", 1).First(&q).Error)
	assert.Equal(t, "COMPLETED", q.Status)
}
