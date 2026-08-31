package store

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
)

var (
	// ErrTradeNotFound is returned when a trade offer id does not exist.
	ErrTradeNotFound = errors.New("trade offer not found")
	// ErrTradeNotPending is returned when acting on an offer that has already
	// been accepted or cancelled.
	ErrTradeNotPending = errors.New("trade offer is not pending")
	// ErrTradeWrongUser is returned when someone other than the named buyer
	// tries to accept an offer, or someone other than the seller/buyer tries
	// to cancel one.
	ErrTradeWrongUser = errors.New("not authorized to act on this trade offer")
	// ErrInsufficientItems is returned when the seller no longer holds enough
	// of the offered item.
	ErrInsufficientItems = errors.New("insufficient items")
)

// CreateTradeOffer records a pending trade offer from seller to buyer, after
// checking the seller currently holds enough of the item.
func (s *Store) CreateTradeOffer(sellerID, buyerID int64, itemID string, quantity, price int) (model.TradeOffer, error) {
	canonical := items.Canonical(itemID)
	var inv model.Inventory
	if err := s.DB.Where("user_id = ? AND item_id = ?", sellerID, canonical).First(&inv).Error; err != nil || inv.Quantity < quantity {
		return model.TradeOffer{}, ErrInsufficientItems
	}
	offer := model.TradeOffer{
		SellerID:  sellerID,
		BuyerID:   buyerID,
		ItemID:    canonical,
		Quantity:  quantity,
		Price:     price,
		Status:    "pending",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.DB.Create(&offer).Error; err != nil {
		return model.TradeOffer{}, err
	}
	return offer, nil
}

// GetTradeOffer loads a trade offer by id.
func (s *Store) GetTradeOffer(offerID int64) (model.TradeOffer, error) {
	var offer model.TradeOffer
	if err := s.DB.Where("id = ?", offerID).First(&offer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.TradeOffer{}, ErrTradeNotFound
		}
		return model.TradeOffer{}, err
	}
	return offer, nil
}

// AcceptTradeOffer executes a pending trade offer atomically: it re-validates
// the seller's item stock and the buyer's balance/inventory space at accept
// time (state may have moved since the offer was posted), then moves the
// money and the item between the two accounts in one transaction.
func (s *Store) AcceptTradeOffer(offerID, buyerID int64) (model.TradeOffer, error) {
	if err := s.ensureUser(buyerID); err != nil {
		return model.TradeOffer{}, err
	}
	var offer model.TradeOffer
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", offerID).First(&offer).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTradeNotFound
			}
			return err
		}
		if offer.Status != "pending" {
			return ErrTradeNotPending
		}
		if offer.BuyerID != buyerID {
			return ErrTradeWrongUser
		}
		var inv model.Inventory
		if err := tx.Where("user_id = ? AND item_id = ?", offer.SellerID, offer.ItemID).First(&inv).Error; err != nil || inv.Quantity < offer.Quantity {
			return ErrInsufficientItems
		}
		var bal int
		if err := tx.Model(&model.User{}).Where("user_id = ?", buyerID).Pluck("balance", &bal).Error; err != nil {
			return err
		}
		if bal < offer.Price {
			return ErrInsufficientFunds
		}
		free, err := s.FreeSlots(tx, buyerID)
		if err != nil {
			return err
		}
		if free < offer.Quantity {
			return ErrInventoryFull
		}
		if err := tx.Model(&model.User{}).Where("user_id = ?", buyerID).
			UpdateColumn("balance", gorm.Expr("balance - ?", offer.Price)).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("user_id = ?", offer.SellerID).
			UpdateColumn("balance", gorm.Expr("balance + ?", offer.Price)).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Inventory{}).
			Where("user_id = ? AND item_id = ?", offer.SellerID, offer.ItemID).
			UpdateColumn("quantity", gorm.Expr("quantity - ?", offer.Quantity)).Error; err != nil {
			return err
		}
		if err := s.AddItemRaw(tx, buyerID, offer.ItemID, offer.Quantity); err != nil {
			return err
		}
		offer.Status = "accepted"
		return tx.Model(&model.TradeOffer{}).Where("id = ?", offer.ID).UpdateColumn("status", "accepted").Error
	})
	if err != nil {
		return model.TradeOffer{}, err
	}
	return offer, nil
}

// CancelTradeOffer marks a pending offer cancelled. Either the seller or the
// named buyer may cancel.
func (s *Store) CancelTradeOffer(offerID, actorID int64) (model.TradeOffer, error) {
	var offer model.TradeOffer
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", offerID).First(&offer).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTradeNotFound
			}
			return err
		}
		if offer.Status != "pending" {
			return ErrTradeNotPending
		}
		if actorID != offer.SellerID && actorID != offer.BuyerID {
			return ErrTradeWrongUser
		}
		offer.Status = "cancelled"
		return tx.Model(&model.TradeOffer{}).Where("id = ?", offer.ID).UpdateColumn("status", "cancelled").Error
	})
	if err != nil {
		return model.TradeOffer{}, err
	}
	return offer, nil
}
