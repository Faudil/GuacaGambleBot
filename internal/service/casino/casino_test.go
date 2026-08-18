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
	"guacagamblebot/internal/model"
	furnituresvc "guacagamblebot/internal/service/furniture"
	housingsvc "guacagamblebot/internal/service/housing"
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

// placeParlor gives the user a brick house with a Gambling Parlor placed.
func placeParlor(t *testing.T, s *store.Store, userID int64) {
	t.Helper()
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	hsvc := housingsvc.New(s, cfg)
	_, err := s.UpdateBalance(userID, 100000)
	require.NoError(t, err)
	require.NoError(t, hsvc.BuyHouse(userID, "brick_house"))
	fsvc := furnituresvc.New(s, cfg, hsvc)
	require.NoError(t, s.AddItemRaw(s.DB, userID, "coal", 50))
	require.NoError(t, s.AddItemRaw(s.DB, userID, "gold_nugget", 50))
	require.NoError(t, fsvc.Place(userID, "gambling_parlor"))
}

func TestEvaluateMegaGrid(t *testing.T) {
	// No winning line.
	lines, payout := evaluateMegaGrid([]string{"🍒", "🍇", "🍋", "🔔", "💎", "🍒", "🍇", "🍋", "🔔"}, 100)
	assert.Empty(t, lines)
	assert.Zero(t, payout)

	// One full row.
	lines, payout = evaluateMegaGrid([]string{"🍒", "🍒", "🍒", "🍇", "🍋", "🔔", "💎", "🍒", "🍇"}, 100)
	assert.Equal(t, []int{0}, lines)
	assert.Equal(t, 100*3, payout)

	// Two crossing lines (row 0 + column 0) stack.
	lines, payout = evaluateMegaGrid([]string{"🍒", "🍒", "🍒", "🍒", "🍋", "🔔", "🍒", "🍇", "💎"}, 100)
	require.Len(t, lines, 2)
	assert.Equal(t, 100*3*2, payout)

	// Full board of diamonds: all 8 lines, biggest jackpot.
	lines, payout = evaluateMegaGrid([]string{"💎", "💎", "💎", "💎", "💎", "💎", "💎", "💎", "💎"}, 50)
	require.Len(t, lines, 8)
	assert.Equal(t, 50*100*8, payout)
}

func TestMegaSlotsRequiresParlor(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 100000)
	require.NoError(t, err)

	_, err = svc.SpinMegaSlots(1, 100)
	assert.ErrorIs(t, err, ErrRequiresFurniture)
}

func TestMegaSlotsPlays(t *testing.T) {
	svc, st := testService(t)
	placeParlor(t, st, 1)
	_, err := st.UpdateBalance(1, 100000)
	require.NoError(t, err)

	res, err := svc.SpinMegaSlots(1, 100)
	require.NoError(t, err)
	require.Len(t, res.Grid, 9)
	for _, sym := range res.Grid {
		assert.NotEmpty(t, sym)
	}
	assert.Equal(t, len(res.WinLines) > 0, res.IsWin)
}

func TestCasinoLimitBoostWithParlor(t *testing.T) {
	svc, st := testService(t)
	assert.Equal(t, baseSlotsLimit, svc.casinoLimit(1, baseSlotsLimit))

	placeParlor(t, st, 1)
	assert.Equal(t, baseSlotsLimit+parlorLimitBoost, svc.casinoLimit(1, baseSlotsLimit))
}

func TestSlotsLimitBoostedByParlor(t *testing.T) {
	svc, st := testService(t)
	placeParlor(t, st, 1)
	_, err := st.UpdateBalance(1, 100000)
	require.NoError(t, err)

	// 15 spins: more than the base 10, allowed with the parlor.
	for i := 0; i < baseSlotsLimit+parlorLimitBoost; i++ {
		_, err := svc.SpinSlots(1, 10)
		require.NoError(t, err)
	}
	_, err = svc.SpinSlots(1, 10)
	assert.ErrorIs(t, err, ErrLimit)
}

func TestMegaSlotsLimit(t *testing.T) {
	svc, st := testService(t)
	placeParlor(t, st, 1)
	_, err := st.UpdateBalance(1, 1000000)
	require.NoError(t, err)

	for i := 0; i < megaSlotsLimit; i++ {
		_, err := svc.SpinMegaSlots(1, 10)
		require.NoError(t, err)
	}
	_, err = svc.SpinMegaSlots(1, 10)
	assert.ErrorIs(t, err, ErrLimit)
}

func TestMegaSlotsMaxBet(t *testing.T) {
	svc, st := testService(t)
	placeParlor(t, st, 1)

	_, err := svc.SpinMegaSlots(1, maxMegaSlotsBet+1)
	assert.ErrorIs(t, err, ErrMaxBet)
}

func TestMegaSlotsRecordsWinOnly(t *testing.T) {
	svc, st := testService(t)
	placeParlor(t, st, 1)
	_, err := st.UpdateBalance(1, 1000000)
	require.NoError(t, err)

	wins := 0
	sumNet := 0
	for i := 0; i < 50; i++ {
		require.NoError(t, st.ResetGameLimit(1, "mega_slots"))
		res, err := svc.SpinMegaSlots(1, 10)
		require.NoError(t, err)
		if res.IsWin {
			wins++
			sumNet += res.Payout - 10
		}
	}
	records, err := st.TopWinRecords("mega_slots", 1000)
	require.NoError(t, err)
	require.Len(t, records, wins)
	recordSum := 0
	for _, r := range records {
		assert.Equal(t, "mega_slots", r.Game)
		recordSum += r.Amount
	}
	assert.Equal(t, sumNet, recordSum)
}

// TestMegaSlotsDailyLimitPersists ensures the game_limit row tracks the game.
func TestMegaSlotsDailyLimitPersists(t *testing.T) {
	svc, st := testService(t)
	placeParlor(t, st, 1)
	_, err := st.UpdateBalance(1, 1000000)
	require.NoError(t, err)

	_, err = svc.SpinMegaSlots(1, 10)
	require.NoError(t, err)

	var gl model.GameLimit
	require.NoError(t, st.DB.Where("user_id = ? AND game_name = ?", 1, "mega_slots").First(&gl).Error)
	assert.Equal(t, 1, gl.Count)
}
