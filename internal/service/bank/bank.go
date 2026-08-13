package bank

import (
	"errors"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/store"
)

var (
	ErrAmount  = errors.New("amount must be positive")
	ErrNoMoney = errors.New("insufficient funds")
)

// Service holds the Bank cog business logic.
type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

// Deposit moves amount from the wallet into the bank atomically.
func (s *Service) Deposit(userID, amount int) (wallet, bank int, err error) {
	if amount <= 0 {
		return 0, 0, ErrAmount
	}
	wallet, bank, err = s.store.BankDeposit(int64(userID), amount)
	if errors.Is(err, store.ErrInsufficientFunds) {
		return 0, 0, ErrNoMoney
	}
	if err != nil {
		return 0, 0, err
	}
	_ = s.store.RecordActivity(int64(userID), "bank_deposits", 1)
	return wallet, bank, nil
}

// Withdraw moves amount from the bank into the wallet atomically.
func (s *Service) Withdraw(userID, amount int) (wallet, bank int, err error) {
	if amount <= 0 {
		return 0, 0, ErrAmount
	}
	wallet, bank, err = s.store.BankWithdraw(int64(userID), amount)
	if errors.Is(err, store.ErrInsufficientFunds) {
		return 0, 0, ErrNoMoney
	}
	if err != nil {
		return 0, 0, err
	}
	return wallet, bank, nil
}

// Info returns wallet, bank and projected daily interest.
func (s *Service) Info(userID int64) (wallet, bank, interest int, err error) {
	wallet, bank, err = s.store.GetBankData(userID)
	if err != nil {
		return 0, 0, 0, err
	}
	return wallet, bank, (bank / 100) * 10, nil
}
