package casino

import (
	"errors"
	"math/rand"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	charsvc "guacagamblebot/internal/service/character"
	furnituresvc "guacagamblebot/internal/service/furniture"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
)

var (
	ErrNoMoney           = errors.New("insufficient funds")
	ErrMaxBet            = errors.New("max bet exceeded")
	ErrLimit             = errors.New("daily limit reached")
	ErrChoice            = errors.New("invalid choice")
	ErrRequiresFurniture = errors.New("requires gambling parlor")
)

var SLOT_SYMBOLS = map[string]struct {
	Weight int
	Mult   int
}{
	"🍒": {Weight: 40, Mult: 3},
	"🍇": {Weight: 30, Mult: 5},
	"🍋": {Weight: 20, Mult: 10},
	"🔔": {Weight: 8, Mult: 20},
	"💎": {Weight: 2, Mult: 100},
}

func buildWheel() []string {
	var wheel []string
	for sym, data := range SLOT_SYMBOLS {
		for i := 0; i < data.Weight; i++ {
			wheel = append(wheel, sym)
		}
	}
	return wheel
}

var wheel = buildWheel()

type SlotsResult struct {
	Symbol1          string
	Symbol2          string
	Symbol3          string
	Payout           int
	IsWin            bool
	WinType          string
	WinSym           string
	XpGain           int
	LeveledUp        bool
	NewLevel         int
	LuckReroll       bool
	PreRerollSymbol3 string
}

type CoinflipResult struct {
	Result    string
	Win       bool
	XpGain    int
	LeveledUp bool
	NewLevel  int
}

// MegaSlotsResult is the outcome of a 3x3 Mega Slots spin. Grid holds the 9
// symbols row-major; WinLines lists the indices of the winning paylines.
type MegaSlotsResult struct {
	Grid      []string
	Payout    int
	IsWin     bool
	WinLines  []int
	XpGain    int
	LeveledUp bool
	NewLevel  int
}

// megaPaylines are the 8 winning lines of a 3x3 grid: 3 rows, 3 columns and
// the two diagonals.
var megaPaylines = [8][3]int{
	{0, 1, 2}, {3, 4, 5}, {6, 7, 8},
	{0, 3, 6}, {1, 4, 7}, {2, 5, 8},
	{0, 4, 8}, {2, 4, 6},
}

const (
	baseSlotsLimit   = 10
	megaSlotsLimit   = 5
	maxMegaSlotsBet  = 50000
	parlorLimitBoost = 5
)

type Service struct {
	store  *store.Store
	cfg    *config.Config
	npcSvc *npcsvc.Service
}

func New(s *store.Store, cfg *config.Config, npcSvc *npcsvc.Service) *Service {
	return &Service{store: s, cfg: cfg, npcSvc: npcSvc}
}

// casinoLimit returns the daily play limit for a game: players with a Gambling
// Parlor placed in their active house get extra plays.
func (s *Service) casinoLimit(userID int64, base int) int {
	if furnituresvc.HasFurniture(s.store, userID, "gambling_parlor") {
		return base + parlorLimitBoost
	}
	return base
}

// RemainingPlays returns how many daily plays the user has left for each
// casino game (slots, coinflip, mega_slots).
func (s *Service) RemainingPlays(userID int64) (slots, coinflip, mega int) {
	slots = s.remaining(userID, "slots", s.casinoLimit(userID, baseSlotsLimit))
	coinflip = s.remaining(userID, "coinflip", s.casinoLimit(userID, baseSlotsLimit))
	mega = s.remaining(userID, "mega_slots", megaSlotsLimit)
	return
}

func (s *Service) remaining(userID int64, game string, max int) int {
	_, left, err := s.store.CheckGameLimit(userID, game, max)
	if err != nil {
		return max
	}
	return left
}

// wagerParams configures the shared bet-validation/limit-check/debit
// preamble used by every casino game.
type wagerParams struct {
	game     string // game_limit / AddWinRecord key, e.g. "slots", "coinflip", "mega_slots"
	statKey  string // achievement stat incremented by the wagered amount
	limitMax int    // pre-resolved max plays for this game today
	maxBet   int    // 0 = no upper cap
}

// placeWager validates the bet amount, enforces the daily play limit,
// records the wagered achievement stat and debits the stake. The play only
// counts against the daily limit once the stat record and debit both
// succeed, so a failure (e.g. insufficient funds) never costs the player a
// try. It returns a translated sentinel error (ErrMaxBet/ErrLimit/ErrNoMoney)
// on failure.
func (s *Service) placeWager(userID int64, amount int, p wagerParams) error {
	if amount <= 0 {
		return ErrMaxBet
	}
	if p.maxBet > 0 && amount > p.maxBet {
		return ErrMaxBet
	}
	ok, err := s.store.PlayGameLimited(userID, p.game, p.limitMax, func() error {
		if err := achievement.IncrementStat(s.store.DB, userID, p.statKey, amount); err != nil {
			return err
		}
		_, err := s.store.Debit(userID, amount)
		return err
	})
	if err != nil {
		if errors.Is(err, store.ErrInsufficientFunds) {
			return ErrNoMoney
		}
		return err
	}
	if !ok {
		return ErrLimit
	}
	return nil
}

// settleSlotsWager applies the lucky_break/jackpot_fever buffs, credits the
// payout on a win, records the win and updates the slots_* achievement
// stats shared by SpinSlots and SpinMegaSlots. gameKey is the AddWinRecord
// game name ("slots" or "mega_slots"); repGain is the gambling reputation
// awarded on a win.
func (s *Service) settleSlotsWager(userID int64, amount int, isWin bool, payout *int, gameKey string, repGain int) error {
	if isWin {
		s.npcSvc.AddActivityReputation(userID, "gambling", repGain)
		if charsvc.HasBuff(s.store, userID, "lucky_break") {
			*payout = int(float64(*payout) * 1.5)
			charsvc.ConsumeBuff(s.store, userID, "lucky_break")
		}
		if charsvc.HasBuff(s.store, userID, "jackpot_fever") {
			*payout *= 3
			charsvc.ConsumeBuff(s.store, userID, "jackpot_fever")
		}
		if _, err := s.store.UpdateBalance(userID, *payout); err != nil {
			return err
		}
		_ = s.store.AddWinRecord(userID, gameKey, *payout-amount)
		if err := achievement.IncrementStat(s.store.DB, userID, "slots_won", 1); err != nil {
			return err
		}
		net := *payout - amount
		if net > 0 {
			_ = achievement.IncrementStat(s.store.DB, userID, "slots_money_won", net)
		} else if net < 0 {
			_ = achievement.IncrementStat(s.store.DB, userID, "slots_money_lost", -net)
		}
		return nil
	}
	if err := achievement.IncrementStat(s.store.DB, userID, "slots_lost", 1); err != nil {
		return err
	}
	_ = achievement.IncrementStat(s.store.DB, userID, "slots_money_lost", amount)
	return nil
}

func (s *Service) SpinSlots(userID int64, amount int) (*SlotsResult, error) {
	if err := s.placeWager(userID, amount, wagerParams{
		game:     "slots",
		statKey:  "slots_spent",
		limitMax: s.casinoLimit(userID, baseSlotsLimit),
	}); err != nil {
		return nil, err
	}

	lukBonus := charsvc.GetLUKBonus(s.store, userID)

	r1 := wheel[rand.Intn(len(wheel))]
	r2 := wheel[rand.Intn(len(wheel))]
	r3 := wheel[rand.Intn(len(wheel))]

	res := &SlotsResult{Symbol1: r1, Symbol2: r2, Symbol3: r3}

	// LUK slightly improves odds: reroll one losing symbol if luck is high
	if r1 != r2 && r2 != r3 && r1 != r3 && lukBonus > 0.5 && rand.Float64() < lukBonus*0.03 {
		res.LuckReroll = true
		res.PreRerollSymbol3 = r3
		r3 = r1
		res.Symbol3 = r3
	}

	if r1 == r2 && r2 == r3 {
		res.IsWin = true
		res.WinType = "JACKPOT"
		res.WinSym = r1
		res.Payout = amount * SLOT_SYMBOLS[r1].Mult
		res.XpGain = 100
	} else if r1 == r2 || r2 == r3 || r1 == r3 {
		res.IsWin = true
		res.WinType = "PAIRE"
		if r1 == r2 {
			res.WinSym = r1
		} else if r2 == r3 {
			res.WinSym = r2
		} else {
			res.WinSym = r1
		}
		fullMult := SLOT_SYMBOLS[res.WinSym].Mult
		ratio := 0.18
		res.Payout = int(float64(amount) * float64(fullMult) * ratio)
		if res.Payout < amount {
			res.Payout = amount
		}
		res.XpGain = 30
	} else {
		res.WinType = "LOSE"
		res.XpGain = 10
		if charsvc.HasPassive(s.store, userID, "perk_casino_edge") && rand.Float64() < 0.01 {
			// Card Sharp passive: small chance to turn a loss into a pair.
			res.IsWin = true
			res.WinType = "PAIRE"
			res.WinSym = r1
			res.Symbol2 = r1
			fullMult := SLOT_SYMBOLS[res.WinSym].Mult
			res.Payout = int(float64(amount) * float64(fullMult) * 0.18)
			if res.Payout < amount {
				res.Payout = amount
			}
			res.XpGain = 30
		}
	}

	repGain := 1
	if res.WinType == "JACKPOT" {
		repGain = 5
	}
	if err := s.settleSlotsWager(userID, amount, res.IsWin, &res.Payout, "slots", repGain); err != nil {
		return nil, err
	}

	leveled, lvl := charsvc.AddXP(s.store, userID, res.XpGain)
	res.LeveledUp = leveled
	res.NewLevel = lvl
	return res, nil
}

func (s *Service) Coinflip(userID int64, choice string, amount int, useRigged bool) (*CoinflipResult, error) {
	choice = s.normalizeChoice(choice)
	if choice == "" {
		return nil, ErrChoice
	}
	if err := s.placeWager(userID, amount, wagerParams{
		game:     "coinflip",
		statKey:  "coinflip_spent",
		limitMax: s.casinoLimit(userID, baseSlotsLimit),
		maxBet:   2000,
	}); err != nil {
		return nil, err
	}

	// A rigged coin used via /use boosts the next flip to 75% odds.
	if !useRigged && charsvc.ConsumeBuff(s.store, userID, "rigged_coin") {
		useRigged = true
	}

	var result string
	var win bool
	if useRigged {
		if rand.Float64() < 0.75 {
			result = choice
			win = true
		} else {
			result = flipOpposite(choice)
			win = false
		}
	} else {
		if rand.Intn(2) == 0 {
			result = "pile"
		} else {
			result = "face"
		}
		win = result == choice
	}

	lukBonus := charsvc.GetLUKBonus(s.store, userID)
	luckyBreak := charsvc.HasBuff(s.store, userID, "lucky_break")

	if !useRigged && !win && lukBonus > 0.5 && rand.Float64() < lukBonus*0.03 {
		win = true
		result = choice
	}

	if luckyBreak && !win && rand.Float64() < 0.5 {
		win = true
		result = choice
		charsvc.ConsumeBuff(s.store, userID, "lucky_break")
	}

	if !win && charsvc.HasPassive(s.store, userID, "perk_casino_edge") && rand.Float64() < 0.01 {
		win = true
		result = choice
	}

	res := &CoinflipResult{Result: result, Win: win}
	if win {
		// The wager was debited upfront, so a win credits the full 1:1 payout.
		s.npcSvc.AddActivityReputation(userID, "gambling", 1)
		if _, err := s.store.UpdateBalance(userID, 2*amount); err != nil {
			return nil, err
		}
		_ = s.store.AddWinRecord(userID, "coinflip", amount)
		_ = achievement.IncrementStat(s.store.DB, userID, "coinflip_won", 1)
		_ = achievement.IncrementStat(s.store.DB, userID, "coinflip_money_won", amount)
		res.XpGain = 10
	} else {
		_ = achievement.IncrementStat(s.store.DB, userID, "coinflip_lost", 1)
		_ = achievement.IncrementStat(s.store.DB, userID, "coinflip_money_lost", amount)
		res.XpGain = 30
	}

	leveled, lvl := charsvc.AddXP(s.store, userID, res.XpGain)
	res.LeveledUp = leveled
	res.NewLevel = lvl
	return res, nil
}

// flipOpposite returns the other side of a coinflip choice ("pile"/"face").
func flipOpposite(choice string) string {
	if choice == "pile" {
		return "face"
	}
	return "pile"
}

func (s *Service) normalizeChoice(c string) string {
	switch c {
	case "heads", "face":
		return "face"
	case "tails", "pile":
		return "pile"
	}
	return ""
}

// evaluateMegaGrid scores a 3x3 grid: every fully-matching payline pays
// amount × symbol multiplier, and all winning lines stack.
func evaluateMegaGrid(grid []string, amount int) (winLines []int, payout int) {
	for li, line := range megaPaylines {
		sym := grid[line[0]]
		if sym == grid[line[1]] && sym == grid[line[2]] {
			winLines = append(winLines, li)
			payout += amount * SLOT_SYMBOLS[sym].Mult
		}
	}
	return winLines, payout
}

// SpinMegaSlots plays the 3x3 Mega Slots machine. The game is only available
// to players who placed a Gambling Parlor in their active house. A spin rolls
// 9 symbols and pays for every fully-matching payline (rows, columns and
// diagonals), so several lines can win at once.
func (s *Service) SpinMegaSlots(userID int64, amount int) (*MegaSlotsResult, error) {
	if amount <= 0 || amount > maxMegaSlotsBet {
		return nil, ErrMaxBet
	}
	if !furnituresvc.HasFurniture(s.store, userID, "gambling_parlor") {
		return nil, ErrRequiresFurniture
	}
	if err := s.placeWager(userID, amount, wagerParams{
		game:     "mega_slots",
		statKey:  "slots_spent",
		limitMax: megaSlotsLimit,
		maxBet:   maxMegaSlotsBet,
	}); err != nil {
		return nil, err
	}

	grid := make([]string, 9)
	for i := range grid {
		grid[i] = wheel[rand.Intn(len(wheel))]
	}

	res := &MegaSlotsResult{Grid: grid}
	res.WinLines, res.Payout = evaluateMegaGrid(grid, amount)
	res.IsWin = len(res.WinLines) > 0
	if res.IsWin {
		res.XpGain = 100
	} else {
		res.XpGain = 10
	}

	if err := s.settleSlotsWager(userID, amount, res.IsWin, &res.Payout, "mega_slots", 5); err != nil {
		return nil, err
	}

	leveled, lvl := charsvc.AddXP(s.store, userID, res.XpGain)
	res.LeveledUp = leveled
	res.NewLevel = lvl
	return res, nil
}
