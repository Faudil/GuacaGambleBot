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
	invsvc "guacagamblebot/internal/service/inventory"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
	"guacagamblebot/internal/universe/hoakhaven"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "c.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	require.NotNil(t, def)
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)
	return New(s, cfg, npcSvc), s
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

func TestCoinflipWinCreditsPayout(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 10000)
	require.NoError(t, err)

	before, err := st.GetBalance(1)
	require.NoError(t, err)
	const bet = 100
	wins, losses := 0, 0
	for i := 0; i < 10; i++ {
		res, err := svc.Coinflip(1, "pile", bet, true)
		require.NoError(t, err)
		if res.Win {
			wins++
		} else {
			losses++
		}
	}
	after, err := st.GetBalance(1)
	require.NoError(t, err)
	assert.Greater(t, wins, 0)
	assert.Equal(t, before+(wins-losses)*bet, after)
}

func TestSlotsAddsCharacterXP(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 1000)
	require.NoError(t, err)

	_, err = svc.SpinSlots(1, 50)
	require.NoError(t, err)

	c, err := st.GetCharacter(1)
	require.NoError(t, err)
	assert.Greater(t, c.XP, 0)
}

func TestCoinflipAddsCharacterXP(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 500)
	require.NoError(t, err)

	_, err = svc.Coinflip(1, "pile", 100, false)
	require.NoError(t, err)

	c, err := st.GetCharacter(1)
	require.NoError(t, err)
	assert.Greater(t, c.XP, 0)
}

func TestCoinflipChoiceNormalization(t *testing.T) {
	svc := &Service{}
	assert.Equal(t, "face", svc.normalizeChoice("heads"))
	assert.Equal(t, "pile", svc.normalizeChoice("tails"))
	assert.Equal(t, "face", svc.normalizeChoice("face"))
	assert.Equal(t, "pile", svc.normalizeChoice("pile"))
	assert.Equal(t, "", svc.normalizeChoice("invalid"))
}

func TestCoinflipRecordsWinOnly(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 100000)
	require.NoError(t, err)

	winCount := 0
	for i := 0; i < 100; i++ {
		require.NoError(t, st.ResetGameLimit(1, "coinflip"))
		res, err := svc.Coinflip(1, "pile", 100, true)
		require.NoError(t, err)
		if res.Win {
			winCount++
		}
	}
	records, err := st.TopWinRecords("coinflip", 1000)
	require.NoError(t, err)
	require.Len(t, records, winCount)
	for _, r := range records {
		assert.Equal(t, int64(1), r.UserID)
		assert.Equal(t, "coinflip", r.Game)
		assert.Equal(t, 100, r.Amount)
	}
}

func TestSlotsRecordsWinOnly(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 100000)
	require.NoError(t, err)

	wins := 0
	sumNet := 0
	for i := 0; i < 200; i++ {
		require.NoError(t, st.ResetGameLimit(1, "slots"))
		res, err := svc.SpinSlots(1, 50)
		require.NoError(t, err)
		if res.IsWin {
			wins++
			sumNet += res.Payout - 50
		}
	}
	records, err := st.TopWinRecords("slots", 1000)
	require.NoError(t, err)
	require.Len(t, records, wins)
	recordSum := 0
	for _, r := range records {
		assert.Equal(t, int64(1), r.UserID)
		assert.Equal(t, "slots", r.Game)
		recordSum += r.Amount
	}
	assert.Equal(t, sumNet, recordSum)
}
