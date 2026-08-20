package economy

import (
	"encoding/json"
	"errors"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	charsvc "guacagamblebot/internal/service/character"
	dailyquest "guacagamblebot/internal/service/dailyquest"
	"guacagamblebot/internal/store"
)

var ErrAlreadyClaimed = errors.New("daily already claimed today")

// Service holds the Economy cog business logic.
type Service struct {
	store *store.Store
	cfg   *config.Config
	dq    *dailyquest.Service
}

func New(s *store.Store, cfg *config.Config, dq *dailyquest.Service) *Service {
	return &Service{store: s, cfg: cfg, dq: dq}
}

// BalanceResult holds the data shown by the balance view.
type BalanceResult struct {
	Wallet   int
	Bank     int
	Interest int
}

// Balance returns wallet, bank and projected daily interest.
func (s *Service) Balance(userID int64) (*BalanceResult, error) {
	wallet, bank, err := s.store.GetBankData(userID)
	if err != nil {
		return nil, err
	}
	return &BalanceResult{Wallet: wallet, Bank: bank, Interest: (bank / 100) * 10}, nil
}

// DailyResult holds the outcome of claiming the daily reward.
type DailyResult struct {
	Amount     int
	Repaid     int
	NewBalance int
	Lenders    []store.RepaidLender
	Unlocks    []*achievement.Achievement
	LeveledUp  bool
	NewLevel   int
	// Recipe is the day's generated daily quest, when one was started (or
	// already active).
	Recipe *store.DailyRecipe
}

// Daily pays the daily salary, applies debt repayment, starts a daily quest and
// evaluates achievements.
func (s *Service) Daily(userID int64) (*DailyResult, error) {
	ready, _, err := s.store.CheckGameLimit(userID, "daily", 1)
	if err != nil {
		return nil, err
	}
	if !ready {
		return nil, ErrAlreadyClaimed
	}

	_, bank, err := s.store.GetBankData(userID)
	if err != nil {
		return nil, err
	}
	amount := s.cfg.DailyAmount + (bank/100)*10

	debt, err := s.store.GetTotalDebt(userID)
	if err != nil {
		return nil, err
	}
	repayCut := amount / 2
	actualRepay := repayCut
	if debt < actualRepay {
		actualRepay = debt
	}
	gain := amount - actualRepay

	newBalance, err := s.store.UpdateBalance(userID, gain)
	if err != nil {
		return nil, err
	}

	var lenders []store.RepaidLender
	if actualRepay > 0 {
		if _, lenders, err = s.store.RepayDebt(userID, actualRepay); err != nil {
			return nil, err
		}
		if newBalance, err = s.store.GetBalance(userID); err != nil {
			return nil, err
		}
	}

	has, err := s.store.HasDailyQuestToday(userID)
	if err != nil {
		return nil, err
	}
	var recipe *store.DailyRecipe
	if !has {
		gen, err := s.dq.Generate(userID)
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(gen)
		if err != nil {
			return nil, err
		}
		if err := s.store.StartDailyQuest(userID, string(data)); err != nil {
			return nil, err
		}
		recipe = &gen
	} else if r, err := s.store.GetDailyRecipe(userID); err == nil {
		recipe = r
	}

	if err := achievement.IncrementStat(s.store.DB, userID, "daily_uses", 1); err != nil {
		return nil, err
	}
	if err := s.store.RecordActivity(userID, "daily_uses", 1); err != nil {
		return nil, err
	}
	unlocks, err := achievement.CheckAndUnlock(s.store.DB, userID)
	if err != nil {
		return nil, err
	}

	leveled, lvl := charsvc.AddXP(s.store, userID, 5)

	if err := s.store.IncrementGameLimit(userID, "daily"); err != nil {
		return nil, err
	}

	return &DailyResult{
		Amount:     amount,
		Repaid:     actualRepay,
		NewBalance: newBalance,
		Lenders:    lenders,
		Unlocks:    unlocks,
		LeveledUp:  leveled,
		NewLevel:   lvl,
		Recipe:     recipe,
	}, nil
}

var (
	ErrSelf    = errors.New("cannot transfer to yourself")
	ErrAmount  = errors.New("amount must be positive")
	ErrNoMoney = errors.New("sender has insufficient funds")
)

// Give transfers amount from sender to recipient atomically.
func (s *Service) Give(sender, recipient int64, amount int) (senderBal, recipientBal int, err error) {
	if sender == recipient {
		return 0, 0, ErrSelf
	}
	if amount <= 0 {
		return 0, 0, ErrAmount
	}
	senderBal, recipientBal, err = s.store.Transfer(sender, recipient, amount)
	if errors.Is(err, store.ErrInsufficientFunds) {
		return 0, 0, ErrNoMoney
	}
	return senderBal, recipientBal, err
}
