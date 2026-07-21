package shop

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/store"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "shop.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestDailyOffersReturnsCorrectCount(t *testing.T) {
	svc, _ := testService(t)
	offers := svc.DailyOffers(10)
	assert.Len(t, offers, 10)
}

func TestDailyOffersReturnsAllItemsWhenCountExceeds(t *testing.T) {
	svc, _ := testService(t)
	offers := svc.DailyOffers(999)
	assert.Len(t, offers, len(items.AllItems()))
}

func TestBuyItemSuccess(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 1000)
	require.NoError(t, err)

	err = svc.BuyItem(1, "charbon", 2)
	require.NoError(t, err)
}

func TestBuyItemNotFound(t *testing.T) {
	svc, _ := testService(t)
	err := svc.BuyItem(1, "nonexistent_item", 1)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestBuyItemInsufficientFunds(t *testing.T) {
	svc, _ := testService(t)
	err := svc.BuyItem(1, "charbon", 1000)
	assert.ErrorIs(t, err, ErrNoMoney)
}
