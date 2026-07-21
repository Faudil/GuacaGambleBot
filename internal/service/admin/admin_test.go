package admin

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
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "admin.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestGiveMoney(t *testing.T) {
	svc, st := testService(t)
	newBal, err := svc.GiveMoney(1, 500)
	require.NoError(t, err)
	assert.Equal(t, 600, newBal)

	bal, err := st.GetBalance(1)
	require.NoError(t, err)
	assert.Equal(t, 600, bal)
}

func TestGiveMoneyNegative(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 500)
	require.NoError(t, err)

	newBal, err := svc.GiveMoney(1, -200)
	require.NoError(t, err)
	assert.Equal(t, 400, newBal)
}

func TestGiveCrowns(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 0) // ensure user exists
	require.NoError(t, err)
	err = svc.GiveCrowns(1, 50)
	require.NoError(t, err)

	var u model.User
	err = st.DB.Where("user_id = ?", 1).First(&u).Error
	require.NoError(t, err)
	assert.Equal(t, 50, u.Crowns)
}

func TestAirdropAll(t *testing.T) {
	svc, st := testService(t)
	// create two users
	_, err := st.UpdateBalance(1, 0)
	require.NoError(t, err)
	_, err = st.UpdateBalance(2, 0)
	require.NoError(t, err)

	count, err := svc.AirdropAll(100)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	b1, _ := st.GetBalance(1)
	b2, _ := st.GetBalance(2)
	assert.Equal(t, 200, b1)
	assert.Equal(t, 200, b2)
}

func TestResetEconomy(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 1000)
	require.NoError(t, err)
	_, err = st.AdjustColumn(1, "bank", 500)
	require.NoError(t, err)

	err = svc.ResetEconomy()
	require.NoError(t, err)

	bal, _ := st.GetBalance(1)
	assert.Equal(t, 100, bal)

	_, bank, err := st.GetBankData(1)
	require.NoError(t, err)
	assert.Equal(t, 0, bank)
}

func TestSetLanguage(t *testing.T) {
	svc, _ := testService(t)
	err := svc.SetLanguage(100, "en")
	require.NoError(t, err)

	lang := svc.store.GetLanguage(100)
	assert.Equal(t, "en", lang)
}
