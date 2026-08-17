package casino

import (
	"errors"
	"math/rand"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	charsvc "guacagamblebot/internal/service/character"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
)

var (
	ErrNoMoney = errors.New("insufficient funds")
	ErrMaxBet  = errors.New("max bet exceeded")
	ErrLimit   = errors.New("daily limit reached")
	ErrChoice  = errors.New("invalid choice")
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

func BuildWheel() []string {
	var wheel []string
	for sym, data := range SLOT_SYMBOLS {
		for i := 0; i < data.Weight; i++ {
			wheel = append(wheel, sym)
		}
	}
	return wheel
}

var wheel = BuildWheel()

type SlotsResult struct {
	Symbol1   string
	Symbol2   string
	Symbol3   string
	Payout    int
	IsWin     bool
	WinType   string
	WinSym    string
	XpGain    int
	LeveledUp bool
	NewLevel  int
}

type CoinflipResult struct {
	Result    string
	Win       bool
	XpGain    int
	LeveledUp bool
	NewLevel  int
}

type Service struct {
	store  *store.Store
	cfg    *config.Config
	npcSvc *npcsvc.Service
}

func New(s *store.Store, cfg *config.Config, npcSvc *npcsvc.Service) *Service {
	return &Service{store: s, cfg: cfg, npcSvc: npcSvc}
}

func (s *Service) SpinSlots(userID int64, amount int) (*SlotsResult, error) {
	if amount <= 0 {
		return nil, ErrMaxBet
	}
	ok, _, err := s.store.CheckGameLimit(userID, "slots", 10)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrLimit
	}
	if err := s.store.IncrementGameLimit(userID, "slots"); err != nil {
		return nil, err
	}
	if err := achievement.IncrementStat(s.store.DB, userID, "slots_spent", amount); err != nil {
		return nil, err
	}
	if _, err := s.store.Debit(userID, amount); err != nil {
		if errors.Is(err, store.ErrInsufficientFunds) {
			return nil, ErrNoMoney
		}
		return nil, err
	}

	lukBonus := charsvc.GetLUKBonus(s.store, userID)
	luckyBreak := charsvc.HasBuff(s.store, userID, "lucky_break")
	jackpotFever := charsvc.HasBuff(s.store, userID, "jackpot_fever")

	r1 := wheel[rand.Intn(len(wheel))]
	r2 := wheel[rand.Intn(len(wheel))]
	r3 := wheel[rand.Intn(len(wheel))]

	res := &SlotsResult{Symbol1: r1, Symbol2: r2, Symbol3: r3}

	// LUK slightly improves odds: reroll one losing symbol if luck is high
	if r1 != r2 && r2 != r3 && r1 != r3 && lukBonus > 0.5 && rand.Float64() < lukBonus*0.05 {
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

	if res.IsWin {
		if res.WinType == "JACKPOT" {
			s.npcSvc.AddActivityReputation(userID, "gambling", 5)
		} else {
			s.npcSvc.AddActivityReputation(userID, "gambling", 1)
		}
		if luckyBreak {
			res.Payout = int(float64(res.Payout) * 1.5)
			charsvc.ConsumeBuff(s.store, userID, "lucky_break")
		}
		if jackpotFever {
			res.Payout *= 3
			charsvc.ConsumeBuff(s.store, userID, "jackpot_fever")
		}
		if _, err := s.store.UpdateBalance(userID, res.Payout); err != nil {
			return nil, err
		}
		_ = s.store.AddWinRecord(userID, "slots", res.Payout-amount)
		if err := achievement.IncrementStat(s.store.DB, userID, "slots_won", 1); err != nil {
			return nil, err
		}
		net := res.Payout - amount
		if net > 0 {
			_ = achievement.IncrementStat(s.store.DB, userID, "slots_money_won", net)
		} else if net < 0 {
			_ = achievement.IncrementStat(s.store.DB, userID, "slots_money_lost", -net)
		}
	} else {
		if err := achievement.IncrementStat(s.store.DB, userID, "slots_lost", 1); err != nil {
			return nil, err
		}
		_ = achievement.IncrementStat(s.store.DB, userID, "slots_money_lost", amount)
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
	if amount <= 0 {
		return nil, ErrMaxBet
	}
	if amount > 2000 {
		return nil, ErrMaxBet
	}
	ok, _, err := s.store.CheckGameLimit(userID, "coinflip", 10)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrLimit
	}
	if err := s.store.IncrementGameLimit(userID, "coinflip"); err != nil {
		return nil, err
	}
	if err := achievement.IncrementStat(s.store.DB, userID, "coinflip_spent", amount); err != nil {
		return nil, err
	}
	if _, err := s.store.Debit(userID, amount); err != nil {
		if errors.Is(err, store.ErrInsufficientFunds) {
			return nil, ErrNoMoney
		}
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
			if choice == "pile" {
				result = "face"
			} else {
				result = "pile"
			}
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
		if choice == "pile" {
			result = "pile"
		} else {
			result = "face"
		}
	}

	if luckyBreak && !win && rand.Float64() < 0.5 {
		win = true
		if choice == "pile" {
			result = "pile"
		} else {
			result = "face"
		}
		charsvc.ConsumeBuff(s.store, userID, "lucky_break")
	}

	if !win && charsvc.HasPassive(s.store, userID, "perk_casino_edge") && rand.Float64() < 0.01 {
		win = true
		if choice == "pile" {
			result = "pile"
		} else {
			result = "face"
		}
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

func (s *Service) normalizeChoice(c string) string {
	switch c {
	case "heads", "face":
		return "face"
	case "tails", "pile":
		return "pile"
	}
	return ""
}
