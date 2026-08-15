package economy

import (
	"errors"
	"math/rand"
	"time"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	charsvc "guacagamblebot/internal/service/character"
	"guacagamblebot/internal/store"
)

var ErrAlreadyClaimed = errors.New("daily already claimed today")

// DailyObjective mirrors the Python DailyQuest objectives.
type DailyObjective struct {
	Stat    string
	Count   int
	TextKey string
}

var DailyObjectives = []DailyObjective{
	{Stat: "blackjack_won", Count: 3, TextKey: "quests.daily.blackjack"},
	{Stat: "items_mined", Count: 10, TextKey: "quests.daily.mining"},
	{Stat: "items_fished", Count: 10, TextKey: "quests.daily.fishing"},
	{Stat: "slots_won", Count: 5, TextKey: "quests.daily.slots"},
	{Stat: "wagers_won", Count: 2, TextKey: "quests.daily.betting"},
}

// Service holds the Economy cog business logic.
type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
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
}

// Daily pays the daily salary, applies debt repayment, starts a daily quest and
// evaluates achievements.
func (s *Service) Daily(userID int64) (*DailyResult, error) {
	ready, err := s.store.CheckCooldown(userID, "daily", 24*time.Hour)
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
	if !has {
		obj := DailyObjectives[rand.Intn(len(DailyObjectives))]
		if err := s.store.StartDailyQuest(userID, obj.Stat, obj.Count, obj.TextKey); err != nil {
			return nil, err
		}
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

	if err := s.store.SetCooldown(userID, "daily"); err != nil {
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
