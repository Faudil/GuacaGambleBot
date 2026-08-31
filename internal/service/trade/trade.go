// Package trade implements direct player-to-player item sales: one player
// offers an item to a named buyer for a price, and the buyer accepts or
// declines.
package trade

import (
	"errors"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

var (
	// ErrSelf is returned when a player tries to sell to themselves.
	ErrSelf = errors.New("cannot trade with yourself")
	// ErrAmount is returned when quantity or price is not positive.
	ErrAmount = errors.New("quantity and price must be positive")
	// ErrNotTradeable is returned when the item cannot be resolved or is
	// excluded from player trading (equipment, or activity-exclusive drops).
	ErrNotTradeable = errors.New("item cannot be traded")
)

// Service holds the trade cog's business logic.
type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

// CreateOffer validates and records a pending trade offer from seller to
// buyer. It returns the resolved item alongside the offer so the caller can
// build a display embed without a second lookup.
func (s *Service) CreateOffer(sellerID, buyerID int64, itemID string, quantity, price int) (model.TradeOffer, *items.Item, error) {
	if sellerID == buyerID {
		return model.TradeOffer{}, nil, ErrSelf
	}
	if quantity <= 0 || price <= 0 {
		return model.TradeOffer{}, nil, ErrAmount
	}
	canonical := items.Canonical(itemID)
	if canonical == "" {
		return model.TradeOffer{}, nil, ErrNotTradeable
	}
	it := items.Get(canonical)
	if it == nil || it.EquipSlot != "" || it.ShopExcluded {
		return model.TradeOffer{}, nil, ErrNotTradeable
	}
	offer, err := s.store.CreateTradeOffer(sellerID, buyerID, canonical, quantity, price)
	if err != nil {
		return model.TradeOffer{}, nil, err
	}
	return offer, it, nil
}

// Accept executes a pending offer on behalf of the named buyer.
func (s *Service) Accept(offerID, buyerID int64) (model.TradeOffer, error) {
	return s.store.AcceptTradeOffer(offerID, buyerID)
}

// Cancel marks a pending offer cancelled; the seller or the named buyer may
// call it.
func (s *Service) Cancel(offerID, actorID int64) (model.TradeOffer, error) {
	return s.store.CancelTradeOffer(offerID, actorID)
}

// GetOffer loads a trade offer by id.
func (s *Service) GetOffer(offerID int64) (model.TradeOffer, error) {
	return s.store.GetTradeOffer(offerID)
}
