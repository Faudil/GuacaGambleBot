package item_manager

import (
	"errors"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"gorm.io/gorm"
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
	TradeSuccess  TradeResult = "SUCCESS"
	TradeNoMoney  TradeResult = "NO_MONEY"
	TradeNoItem   TradeResult = "NO_ITEM"
	TradeUnknown  TradeResult = "UNKNOWN"
)

func (s *Service) TransferItem(sellerID, buyerID int64, itemName string, price int) TradeResult {
	var dbItem model.Item
	if err := s.store.DB.Where("name = ?", itemName).First(&dbItem).Error; err != nil {
		return TradeNoItem
	}
	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", sellerID, dbItem.ID).First(&inv).Error; err != nil {
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

	err = s.store.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := s.store.UpdateBalance(sellerID, price); err != nil {
			return err
		}
		if _, err := s.store.UpdateBalance(buyerID, -price); err != nil {
			return err
		}
		if err := tx.Model(&model.Inventory{}).
			Where("user_id = ? AND item_id = ?", sellerID, dbItem.ID).
			UpdateColumn("quantity", gorm.Expr("quantity - ?", 1)).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND item_id = ?", buyerID, dbItem.ID).
			FirstOrCreate(&model.Inventory{UserID: buyerID, ItemID: dbItem.ID, Quantity: 0}).
			UpdateColumn("quantity", gorm.Expr("quantity + ?", 1)).Error; err != nil {
			return err
		}
		return nil
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
