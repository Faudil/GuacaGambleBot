package economy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	dailyquest "guacagamblebot/internal/service/dailyquest"
	invsvc "guacagamblebot/internal/service/inventory"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
	"guacagamblebot/internal/universe"
	"guacagamblebot/internal/universe/hoakhaven"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d := testutil.NewDB(t)
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	inv := invsvc.New(s, cfg)
	npc := npcsvc.New(s, cfg, def, inv)
	dq := dailyquest.New(s, npc)
	return New(s, cfg, dq), s
}

func TestBalance(t *testing.T) {
	svc, _ := testService(t)
	res, err := svc.Balance(1)
	require.NoError(t, err)
	assert.Equal(t, 100, res.Wallet)
	assert.Equal(t, 0, res.Bank)
	assert.Equal(t, 0, res.Interest)
}

func TestDailyAddsCharacterXP(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.Daily(1)
	require.NoError(t, err)

	c, err := svc.store.EnsureCharacter(1)
	require.NoError(t, err)
	assert.Greater(t, c.XP, 0)
}

func TestDailyNoDebt(t *testing.T) {
	svc, _ := testService(t)
	res, err := svc.Daily(1)
	require.NoError(t, err)
	// bank is 0 so amount = DAILY_AMOUNT (50), fully gained.
	assert.Equal(t, 50, res.Amount)
	assert.Equal(t, 0, res.Repaid)
	assert.Equal(t, 150, res.NewBalance)
	// daily_uses reached 1 -> daily_1 should unlock.
	unlockIDs := map[string]bool{}
	for _, a := range res.Unlocks {
		unlockIDs[a.ID] = true
	}
	assert.True(t, unlockIDs["daily_1"])
}

func TestDailyOncePerDay(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.Daily(1)
	require.NoError(t, err)

	_, err = svc.Daily(1)
	assert.ErrorIs(t, err, ErrAlreadyClaimed)
}

func TestDailyWithDebt(t *testing.T) {
	svc, st := testService(t)
	// Pre-create accounts with a known balance. A direct Create ignores a
	// zero-value Balance, so set it explicitly afterwards.
	require.NoError(t, st.DB.Create(&model.User{UserID: 10}).Error)
	require.NoError(t, st.DB.Create(&model.User{UserID: 20}).Error)
	require.NoError(t, st.DB.Model(&model.User{}).Where("user_id = ?", 10).Update("balance", 1000).Error)
	require.NoError(t, st.DB.Model(&model.User{}).Where("user_id = ?", 20).Update("balance", 0).Error)
	require.NoError(t, st.DB.Create(&model.Loan{BorrowerID: 10, LenderID: 20, AmountDue: 300}).Error)

	res, err := svc.Daily(10)
	require.NoError(t, err)
	// amount = 50, repayCut = 25, debt 300 -> actualRepay 25.
	assert.Equal(t, 50, res.Amount)
	assert.Equal(t, 25, res.Repaid)
	// borrower net: +25 gain (50 - 25 repaid) applied to 1000 -> 1000.
	assert.Equal(t, 1000, res.NewBalance)

	lenderBal, err := st.GetBalance(20)
	require.NoError(t, err)
	assert.Equal(t, 25, lenderBal)

	debt, err := st.GetTotalDebt(10)
	require.NoError(t, err)
	assert.Equal(t, 275, debt)
}

func TestGive(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 1000) // sender has 1100
	require.NoError(t, err)
	require.NoError(t, st.DB.Create(&model.User{UserID: 2}).Error)
	require.NoError(t, st.DB.Model(&model.User{}).Where("user_id = ?", 2).Update("balance", 0).Error)

	sb, rb, err := svc.Give(1, 2, 100)
	require.NoError(t, err)
	assert.Equal(t, 1000, sb)
	assert.Equal(t, 100, rb)

	_, _, err = svc.Give(1, 1, 100)
	assert.ErrorIs(t, err, ErrSelf)

	_, _, err = svc.Give(1, 2, -5)
	assert.ErrorIs(t, err, ErrAmount)

	_, _, err = svc.Give(1, 2, 100000)
	assert.ErrorIs(t, err, ErrNoMoney)
}

func TestGiveTransfersCorrectly(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(5, 500)
	require.NoError(t, err)
	require.NoError(t, st.DB.Create(&model.User{UserID: 6}).Error)
	require.NoError(t, st.DB.Model(&model.User{}).Where("user_id = ?", 6).Update("balance", 0).Error)
	_, rb, err := svc.Give(5, 6, 200)
	require.NoError(t, err)
	assert.Equal(t, 200, rb)
	sb, _ := st.GetBalance(5)
	assert.Equal(t, 400, sb)
}
