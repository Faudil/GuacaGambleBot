package bank

import (
	"errors"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/service/housing"
	"guacagamblebot/internal/store"
)

var (
	ErrAmount   = errors.New("amount must be positive")
	ErrNoMoney  = errors.New("insufficient funds")
	ErrBankFull = errors.New("bank full")
)

// DepositResult reports the outcome of a deposit: the amount actually moved
// into the bank (never more than the remaining space under the cap), the new
// wallet and bank balances, and the bank's maximum capacity.
type DepositResult struct {
	Wallet, Bank, Deposited, MaxBank int
}

// Service holds the Bank cog business logic.
type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

// maxBank returns the user's housing-based bank capacity.
func (s *Service) maxBank(userID int64) (int, error) {
	return housing.New(s.store, s.cfg).BankCapacity(userID)
}

// Deposit moves amount from the wallet into the bank atomically, capping the
// bank at the user's housing-based maximum capacity. The returned result
// always carries the user's MaxBank, even on error.
func (s *Service) Deposit(userID, amount int) (DepositResult, error) {
	res := DepositResult{MaxBank: 500}
	if amount <= 0 {
		return res, ErrAmount
	}
	wallet, bank, err := s.clamp(int64(userID))
	if err != nil {
		return res, err
	}
	res.Wallet, res.Bank = wallet, bank
	maxBank, err := s.maxBank(int64(userID))
	if err != nil {
		return res, err
	}
	res.MaxBank = maxBank
	deposited, wallet, bank, err := s.store.BankDeposit(int64(userID), amount, maxBank)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrInsufficientFunds):
			return res, ErrNoMoney
		case errors.Is(err, store.ErrBankFull):
			return res, ErrBankFull
		default:
			return res, err
		}
	}
	res.Wallet, res.Bank, res.Deposited = wallet, bank, deposited
	if deposited > 0 {
		_ = s.store.RecordActivity(int64(userID), "bank_deposits", 1)
	}
	return res, nil
}

// Withdraw moves amount from the bank into the wallet atomically. It also
// returns the user's housing-based bank capacity.
func (s *Service) Withdraw(userID, amount int) (wallet, bank, maxBank int, err error) {
	if amount <= 0 {
		return 0, 0, 0, ErrAmount
	}
	if _, _, err := s.clamp(int64(userID)); err != nil {
		return 0, 0, 0, err
	}
	wallet, bank, err = s.store.BankWithdraw(int64(userID), amount)
	if errors.Is(err, store.ErrInsufficientFunds) {
		return 0, 0, 0, ErrNoMoney
	}
	if err != nil {
		return 0, 0, 0, err
	}
	maxBank, err = s.maxBank(int64(userID))
	if err != nil {
		return 0, 0, 0, err
	}
	return wallet, bank, maxBank, nil
}

// Info returns wallet, bank, projected daily interest and the user's
// housing-based bank capacity.
func (s *Service) Info(userID int64) (wallet, bank, interest, maxBank int, err error) {
	if _, _, err := s.clamp(userID); err != nil {
		return 0, 0, 0, 0, err
	}
	wallet, bank, err = s.store.GetBankData(userID)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	maxBank, err = s.maxBank(userID)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return wallet, bank, (bank / 100) * 10, maxBank, nil
}
