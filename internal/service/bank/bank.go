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

// Deposit moves amount from the wallet into the bank.
func (s *Service) Deposit(userID, amount int) (wallet, bank int, err error) {
	if amount <= 0 {
		return 0, 0, ErrAmount
	}
	bal, err := s.store.GetBalance(int64(userID))
	if err != nil {
		return 0, 0, err
	}
	if bal < amount {
		return 0, 0, ErrNoMoney
	}
	if _, err = s.store.UpdateBalance(int64(userID), -amount); err != nil {
		return 0, 0, err
	}
	bank, err = s.store.AdjustColumn(int64(userID), "bank", amount)
	if err != nil {
		return 0, 0, err
	}
	wallet, err = s.store.GetBalance(int64(userID))
	if err != nil {
		return 0, 0, err
	}
	return wallet, bank, nil
}

// Withdraw moves amount from the bank into the wallet.
func (s *Service) Withdraw(userID, amount int) (wallet, bank int, err error) {
	if amount <= 0 {
		return 0, 0, ErrAmount
	}
	_, bankBal, err := s.store.GetBankData(int64(userID))
	if err != nil {
		return 0, 0, err
	}
	if bankBal < amount {
		return 0, 0, ErrNoMoney
	}
	if _, err = s.store.AdjustColumn(int64(userID), "bank", -amount); err != nil {
		return 0, 0, err
	}
	wallet, err = s.store.UpdateBalance(int64(userID), amount)
	if err != nil {
		return 0, 0, err
	}
	_, bank, err = s.store.GetBankData(int64(userID))
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
