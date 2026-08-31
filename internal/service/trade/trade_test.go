package trade

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	db := testutil.NewDB(t)
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	return New(store.New(db, cfg), cfg)
}

func TestCreateOfferRejectsSelfTrade(t *testing.T) {
	s := newTestService(t)
	_, _, err := s.CreateOffer(1, 1, "coal", 1, 10)
	assert.ErrorIs(t, err, ErrSelf)
}

func TestCreateOfferRejectsNonPositiveAmounts(t *testing.T) {
	s := newTestService(t)
	_, _, err := s.CreateOffer(1, 2, "coal", 0, 10)
	assert.ErrorIs(t, err, ErrAmount)
	_, _, err = s.CreateOffer(1, 2, "coal", 1, 0)
	assert.ErrorIs(t, err, ErrAmount)
}

func TestCreateOfferRejectsEquipment(t *testing.T) {
	s := newTestService(t)
	_, _, err := s.CreateOffer(1, 2, "longsword", 1, 10)
	assert.ErrorIs(t, err, ErrNotTradeable)
}

func TestCreateOfferRejectsUnknownItem(t *testing.T) {
	s := newTestService(t)
	_, _, err := s.CreateOffer(1, 2, "not_a_real_item", 1, 10)
	assert.ErrorIs(t, err, ErrNotTradeable)
}

func TestCreateOfferHappyPath(t *testing.T) {
	s := newTestService(t)
	require.NoError(t, s.store.DB.Create(&model.Inventory{UserID: 1, ItemID: "coal", Quantity: 10}).Error)

	offer, it, err := s.CreateOffer(1, 2, "coal", 5, 50)
	require.NoError(t, err)
	assert.Equal(t, "coal", it.ID)
	assert.Equal(t, "pending", offer.Status)
	assert.Equal(t, int64(1), offer.SellerID)
	assert.Equal(t, int64(2), offer.BuyerID)
}
