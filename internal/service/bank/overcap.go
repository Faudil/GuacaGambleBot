package bank

import (
	"guacagamblebot/internal/service/housing"
)

// BankOverCapAutoFix silently restores the invariant bank <= capacity on the
// user's next bank command (Deposit, Withdraw, Info) by moving any excess
// back into the wallet. It repairs accounts that deposited beyond the limit
// before the cap was enforced.
//
// To remove the fix entirely: set this to false (or delete this file plus
// the calls to clamp() in Deposit, Withdraw and Info, and
// store.ClampBank in internal/store/clamp_bank.go).
var BankOverCapAutoFix = true

// clamp moves any bank balance above the user's capacity back into the
// wallet and returns the clamped balances. It is a no-op when
// BankOverCapAutoFix is disabled.
func (s *Service) clamp(userID int64) (wallet, bank int, err error) {
	if !BankOverCapAutoFix {
		return 0, 0, nil
	}
	maxBank, err := housing.New(s.store, s.cfg).BankCapacity(userID)
	if err != nil {
		return 0, 0, err
	}
	_, wallet, bank, err = s.store.ClampBank(userID, maxBank)
	return wallet, bank, err
}
