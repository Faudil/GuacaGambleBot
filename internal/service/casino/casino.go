package casino

import (
	"errors"
	"math/rand"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/store"
)

var (
	ErrNoMoney   = errors.New("insufficient funds")
	ErrMaxBet    = errors.New("max bet exceeded")
	ErrLimit     = errors.New("daily limit reached")
	ErrChoice    = errors.New("invalid choice")
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
	Symbol1 string
	Symbol2 string
	Symbol3 string
	Payout  int
	IsWin   bool
	WinType string
	WinSym  string
	XpGain  int
}

type CoinflipResult struct {
	Result string
	Win    bool
	XpGain int
}

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) SpinSlots(userID int64, amount int) (*SlotsResult, error) {
	if amount <= 0 {
		return nil, ErrMaxBet
	}
	bal, err := s.store.GetBalance(userID)
	if err != nil {
		return nil, err
	}
	if bal < amount {
		return nil, ErrNoMoney
	}
	ok, _, err := s.store.CheckGameLimit(userID, "slots", 10)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrLimit
	}
	if _, err := s.store.UpdateBalance(userID, -amount); err != nil {
		return nil, err
	}
	if err := s.store.IncrementGameLimit(userID, "slots"); err != nil {
		return nil, err
	}
	if err := achievement.IncrementStat(s.store.DB, userID, "slots_spent", amount); err != nil {
		return nil, err
	}

	r1 := wheel[rand.Intn(len(wheel))]
	r2 := wheel[rand.Intn(len(wheel))]
	r3 := wheel[rand.Intn(len(wheel))]

	res := &SlotsResult{Symbol1: r1, Symbol2: r2, Symbol3: r3}

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
	}

	if res.IsWin {
		if _, err := s.store.UpdateBalance(userID, res.Payout); err != nil {
			return nil, err
		}
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
	bal, err := s.store.GetBalance(userID)
	if err != nil {
		return nil, err
	}
	if bal < amount {
		return nil, ErrNoMoney
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

	res := &CoinflipResult{Result: result, Win: win}
	if win {
		if _, err := s.store.UpdateBalance(userID, amount); err != nil {
			return nil, err
		}
		_ = achievement.IncrementStat(s.store.DB, userID, "coinflip_won", 1)
		_ = achievement.IncrementStat(s.store.DB, userID, "coinflip_money_won", amount)
		res.XpGain = 10
	} else {
		if _, err := s.store.UpdateBalance(userID, -amount); err != nil {
			return nil, err
		}
		_ = achievement.IncrementStat(s.store.DB, userID, "coinflip_lost", 1)
		_ = achievement.IncrementStat(s.store.DB, userID, "coinflip_money_lost", amount)
		res.XpGain = 30
	}

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
