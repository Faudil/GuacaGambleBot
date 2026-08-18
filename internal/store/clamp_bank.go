package store

import (
	"gorm.io/gorm"

	"guacagamblebot/internal/model"
)

// ClampBank moves any excess over maxBank from the bank back into the wallet
// in a single transaction, restoring the invariant bank <= maxBank. It is a
// no-op when the bank is already within the limit. Returns the amount moved
// and the new wallet and bank balances.
//
// NOTE: this exists solely to repair accounts that exceeded the bank cap
// before the limit was enforced. Once those accounts are fixed, this method
// (and its callers in internal/service/bank) can be deleted.
func (s *Store) ClampBank(userID int64, maxBank int) (moved, wallet, bank int, err error) {
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.ensureUserTx(tx, userID); err != nil {
			return err
		}
		var u model.User
		if err := tx.Where("user_id = ?", userID).First(&u).Error; err != nil {
			return err
		}
		excess := u.Bank - maxBank
		if excess <= 0 {
			wallet, bank = u.Balance, u.Bank
			return nil
		}
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", userID).
			UpdateColumn("bank", gorm.Expr("bank - ?", excess)).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", userID).
			UpdateColumn("balance", gorm.Expr("balance + ?", excess)).Error; err != nil {
			return err
		}
		moved, wallet, bank = excess, u.Balance+excess, u.Bank-excess
		return nil
	})
	return moved, wallet, bank, err
}
