package shop

import (
	"path/filepath"
	"testing"
	"time"

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

	err = svc.BuyItem(1, "coal", 2, 1)
	require.NoError(t, err)
}

func TestBuyItemNotFound(t *testing.T) {
	svc, _ := testService(t)
	err := svc.BuyItem(1, "nonexistent_item", 1, 1)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestBuyItemInsufficientFunds(t *testing.T) {
	svc, _ := testService(t)
	err := svc.BuyItem(1, "coal", 1000, 1)
	assert.ErrorIs(t, err, ErrNoMoney)
}

// TestBuyItemSingleConnPool guards against the connection-pool deadlock that
// occurred in production: the DB is opened with a single connection (as the bot
// does), and BuyItem must not block forever by querying the shared pool from
// inside its own transaction.
func TestBuyItemSingleConnPool(t *testing.T) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "shop.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	sqlDB, err := d.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	st := store.New(d, cfg)
	svc := New(st, cfg)
	_, err = st.UpdateBalance(1, 1000)
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		done <- svc.BuyItem(1, "coal", 1, 1)
	}()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("BuyItem deadlocked with a single-connection pool")
	}
}
