package loan

import (
	"errors"

	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

// MaxLoan is the largest single loan the bank will issue.
const MaxLoan = 1000

var (
	ErrAmount      = errors.New("amount must be positive")
	ErrNoDebt      = errors.New("no debt to repay")
	ErrExceedsDebt = errors.New("repayment exceeds total debt")
	ErrMaxLoan     = errors.New("loan exceeds the maximum allowed")
)

// Service holds the Loan cog business logic.
type Service struct {
	store *store.Store
}

func New(s *store.Store) *Service {
	return &Service{store: s}
}

// Borrow issues a bank loan (LenderID = 0) to the user and credits the wallet.
func (s *Service) Borrow(userID, amount int) error {
	if amount <= 0 {
		return ErrAmount
	}
	if amount > MaxLoan {
		return ErrMaxLoan
	}
	uid := int64(userID)
	if err := s.store.DB.Create(&model.Loan{BorrowerID: uid, LenderID: 0, AmountDue: amount}).Error; err != nil {
		return err
	}
	if _, err := s.store.UpdateBalance(uid, amount); err != nil {
		return err
	}
	return nil
}

// Repay applies a payment towards the borrower's debt and debits the wallet.
func (s *Service) Repay(userID, amount int) (int, error) {
	if amount <= 0 {
		return 0, ErrAmount
	}
	uid := int64(userID)
	debt, err := s.store.GetTotalDebt(uid)
	if err != nil {
		return 0, err
	}
	if debt == 0 {
		return 0, ErrNoDebt
	}
	if amount > debt {
		return 0, ErrExceedsDebt
	}
	paid, _, err := s.store.RepayDebt(uid, amount)
	if err != nil {
		return 0, err
	}
	return paid, nil
}

// List returns every open loan for the borrower.
func (s *Service) List(userID int64) ([]model.Loan, error) {
	var loans []model.Loan
	if err := s.store.DB.Where("borrower_id = ?", userID).Find(&loans).Error; err != nil {
		return nil, err
	}
	return loans, nil
}
