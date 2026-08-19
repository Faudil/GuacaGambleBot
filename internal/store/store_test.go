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

func TestUpdateBalanceTracksMoneyEarned(t *testing.T) {
	s := newStore(t)

	_, err := s.UpdateBalance(1, 250)
	require.NoError(t, err)
	_, err = s.UpdateBalance(1, -100)
	require.NoError(t, err)

	var us model.UserStat
	require.NoError(t, s.DB.Where("user_id = 1").First(&us).Error)
	assert.Equal(t, 250, us.MoneyEarned, "only positive credits count as earned")

	// The transactional variant accumulates inside the caller's transaction.
	require.NoError(t, s.DB.Transaction(func(tx *gorm.DB) error {
		return s.UpdateBalanceTx(tx, 1, 150)
	}))
	require.NoError(t, s.DB.Where("user_id = 1").First(&us).Error)
	assert.Equal(t, 400, us.MoneyEarned)
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

func TestServerSettingRoundTrip(t *testing.T) {
	s := New(testDB(t), &config.Config{Prefix: "!"})

	require.NoError(t, s.SaveServerSetting(&model.ServerSetting{
		ServerID:  7,
		ChannelID: 123,
		Language:  "en",
		Prefix:    "?",
		Enabled:   true,
	}))

	ss, err := s.GetServerSetting(7)
	require.NoError(t, err)
	require.NotNil(t, ss)
	assert.Equal(t, int64(123), ss.ChannelID)
	assert.Equal(t, "en", ss.Language)
	assert.Equal(t, "?", ss.Prefix)
	assert.True(t, ss.Enabled)

	// Upsert must update rather than duplicate. Like the handlers, we load the
	// row, mutate a few fields, then save so untouched columns are preserved.
	ss, err = s.GetServerSetting(7)
	require.NoError(t, err)
	require.NotNil(t, ss)
	ss.ChannelID = 456
	ss.Language = "fr"
	ss.Prefix = "&"
	require.NoError(t, s.SaveServerSetting(ss))
	ss, err = s.GetServerSetting(7)
	require.NoError(t, err)
	assert.Equal(t, int64(456), ss.ChannelID)
	assert.Equal(t, "fr", ss.Language)
	assert.Equal(t, "&", ss.Prefix)
	assert.True(t, ss.Enabled, "untouched columns keep their value on upsert")

	// Missing row returns nil.
	none, err := s.GetServerSetting(999)
	require.NoError(t, err)
	assert.Nil(t, none)
}

func TestServerPrefixAndEnabled(t *testing.T) {
	s := New(testDB(t), &config.Config{Prefix: "!"})

	// No row -> defaults.
	assert.Equal(t, "!", s.ServerPrefix(1))
	assert.True(t, s.IsEnabled(1))

	// Create the row (enabled defaults to true), then disable via an upsert
	// (the production toggle path). GORM's `default:true` tag makes an explicit
	// false on the initial Create use the DB default, so disabling must go
	// through the UPDATE branch.
	require.NoError(t, s.SaveServerSetting(&model.ServerSetting{ServerID: 1, Prefix: "?"}))
	require.NoError(t, s.SaveServerSetting(&model.ServerSetting{ServerID: 1, Prefix: "?", Enabled: false}))

	assert.Equal(t, "?", s.ServerPrefix(1))
	assert.False(t, s.IsEnabled(1))

	// Guild 0 (DM) always enabled/default prefix.
	assert.True(t, s.IsEnabled(0))
	assert.Equal(t, "!", s.ServerPrefix(0))
}
