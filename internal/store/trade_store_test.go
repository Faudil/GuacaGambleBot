package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/model"
)

func TestTradeOfferAcceptMovesMoneyAndItem(t *testing.T) {
	s := newStore(t)
	_, err := s.GetBalance(1)
	require.NoError(t, err)
	_, err = s.GetBalance(2)
	require.NoError(t, err)
	require.NoError(t, s.DB.Create(&model.Inventory{UserID: 1, ItemID: "coal", Quantity: 10}).Error)
	require.NoError(t, s.DB.Model(&model.User{}).Where("user_id = ?", 2).UpdateColumn("balance", 100).Error)

	offer, err := s.CreateTradeOffer(1, 2, "coal", 5, 50)
	require.NoError(t, err)
	assert.Equal(t, "pending", offer.Status)

	accepted, err := s.AcceptTradeOffer(offer.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, "accepted", accepted.Status)

	sellerBal, err := s.GetBalance(1)
	require.NoError(t, err)
	assert.Equal(t, 150, sellerBal) // 100 starting balance + 50 sale price

	buyerBal, err := s.GetBalance(2)
	require.NoError(t, err)
	assert.Equal(t, 50, buyerBal)

	var sellerInv, buyerInv model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "coal").First(&sellerInv).Error)
	assert.Equal(t, 5, sellerInv.Quantity)
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 2, "coal").First(&buyerInv).Error)
	assert.Equal(t, 5, buyerInv.Quantity)
}

func TestCreateTradeOfferRejectsInsufficientItems(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.DB.Create(&model.Inventory{UserID: 1, ItemID: "coal", Quantity: 2}).Error)

	_, err := s.CreateTradeOffer(1, 2, "coal", 5, 50)
	assert.ErrorIs(t, err, ErrInsufficientItems)
}

func TestAcceptTradeOfferRejectsWrongBuyer(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.DB.Create(&model.Inventory{UserID: 1, ItemID: "coal", Quantity: 10}).Error)
	offer, err := s.CreateTradeOffer(1, 2, "coal", 5, 50)
	require.NoError(t, err)

	_, err = s.AcceptTradeOffer(offer.ID, 3)
	assert.ErrorIs(t, err, ErrTradeWrongUser)
}

func TestAcceptTradeOfferRejectsInsufficientFunds(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.DB.Create(&model.Inventory{UserID: 1, ItemID: "coal", Quantity: 10}).Error)
	offer, err := s.CreateTradeOffer(1, 2, "coal", 5, 500)
	require.NoError(t, err)

	_, err = s.AcceptTradeOffer(offer.ID, 2)
	assert.ErrorIs(t, err, ErrInsufficientFunds)

	var buyerInv model.Inventory
	err = s.DB.Where("user_id = ? AND item_id = ?", 2, "coal").First(&buyerInv).Error
	assert.Error(t, err)
}

func TestCancelTradeOfferLeavesStateUntouched(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.DB.Create(&model.Inventory{UserID: 1, ItemID: "coal", Quantity: 10}).Error)
	offer, err := s.CreateTradeOffer(1, 2, "coal", 5, 50)
	require.NoError(t, err)

	cancelled, err := s.CancelTradeOffer(offer.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", cancelled.Status)

	_, err = s.AcceptTradeOffer(offer.ID, 2)
	assert.ErrorIs(t, err, ErrTradeNotPending)

	var sellerInv model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "coal").First(&sellerInv).Error)
	assert.Equal(t, 10, sellerInv.Quantity)
}

func TestCancelTradeOfferRejectsUnrelatedUser(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.DB.Create(&model.Inventory{UserID: 1, ItemID: "coal", Quantity: 10}).Error)
	offer, err := s.CreateTradeOffer(1, 2, "coal", 5, 50)
	require.NoError(t, err)

	_, err = s.CancelTradeOffer(offer.ID, 99)
	assert.ErrorIs(t, err, ErrTradeWrongUser)
}
