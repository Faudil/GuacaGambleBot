package item_manager

import (
	"errors"

	"gorm.io/gorm"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

type TradeResult string

const (
	TradeSuccess TradeResult = "SUCCESS"
	TradeNoMoney TradeResult = "NO_MONEY"
	TradeNoItem  TradeResult = "NO_ITEM"
	TradeNoSpace TradeResult = "NO_SPACE"
	TradeUnknown TradeResult = "UNKNOWN"
)

func (s *Service) TransferItem(sellerID, buyerID int64, itemName string, price int) TradeResult {
	// The key may come from a user-typed name; resolve it to the canonical id
	// so the seller's row and the buyer's grant use the same key.
	canonical := items.Canonical(itemName)
	if canonical == "" {
		return TradeNoItem
	}
	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", sellerID, canonical).First(&inv).Error; err != nil {
		return TradeNoItem
	}
	if inv.Quantity < 1 {
		return TradeNoItem
	}
	buyerBal, err := s.store.GetBalance(buyerID)
	if err != nil {
		return TradeUnknown
	}
	if buyerBal < price {
		return TradeNoMoney
	}
	free, err := s.store.FreeSlots(s.store.DB, buyerID)
	if err != nil {
		return TradeUnknown
	}
	if free < 1 {
		return TradeNoSpace
	}

	err = s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.store.UpdateBalanceTx(tx, sellerID, price); err != nil {
			return err
		}
		if err := s.store.UpdateBalanceTx(tx, buyerID, -price); err != nil {
			return err
		}
		if err := tx.Model(&model.Inventory{}).
			Where("user_id = ? AND item_id = ?", sellerID, canonical).
			UpdateColumn("quantity", gorm.Expr("quantity - ?", 1)).Error; err != nil {
			return err
		}
		return s.store.AddItemRaw(tx, buyerID, canonical, 1)
	})
	if err != nil {
		return TradeUnknown
	}
	return TradeSuccess
}

var (
	ErrSelf    = errors.New("cannot trade with yourself")
	ErrNoItem  = errors.New("no item")
	ErrNoMoney = errors.New("no money")
)
