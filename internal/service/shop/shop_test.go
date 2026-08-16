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
	assert.Len(t, offers, len(OfferableItems()))
}

// TestDailyOffersOnlyOfferableItems guards against free, collectible or
// activity-exclusive items (legendary sets, criminality gear, boss trinkets,
// quest items) being offered by the daily shop.
func TestDailyOffersOnlyOfferableItems(t *testing.T) {
	svc, _ := testService(t)
	for _, offer := range svc.DailyOffers(999) {
		it := offer.Item
		assert.True(t, it.Price > 0, "%s must have a positive price", it.ID)
		assert.NotEqual(t, "collectible", it.EffectType, "%s must not be collectible", it.ID)
		assert.False(t, it.ShopExcluded, "%s must not be shop-excluded", it.ID)
		assert.Empty(t, it.SetID, "%s is a set piece and must not be offered", it.ID)
	}

	// Explicitly verify the previously-offered exclusive items are gone.
	for _, id := range []string{
		"dragon_slayer_ring", "dragon_slayer_sword", "shadow_stalker_blade",
		"arcane_weaver_staff", "rift_blade", "hounds_cloak", "shadow_cowl",
		"mask_of_malveillance", "spark_shard", "phoenix_crest",
		"mysterious_seed", "rotten_plant", "boss_trophy", "mastery_medallion",
	} {
		_, ok := svc.OfferForItem(id)
		assert.False(t, ok, "%s must not be offered by the daily shop", id)
	}

	// Legendary drops must be earned by playing their activity — never sold
	// in the shop.
	for _, id := range []string{"nova_fruit", "shadow_fossil", "resonance_core"} {
		_, ok := svc.OfferForItem(id)
		assert.False(t, ok, "%s must not be offered by the daily shop", id)
	}
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

func TestBuyItemFullInventoryRejected(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 100000)
	require.NoError(t, err)
	require.NoError(t, st.AddItemRaw(st.DB, 1, "coal", store.BaseInventoryLimit))

	bal, _ := st.GetBalance(1)
	err = svc.BuyItem(1, "coal", 1, 1)
	assert.ErrorIs(t, err, store.ErrInventoryFull)

	newBal, _ := st.GetBalance(1)
	assert.Equal(t, bal, newBal, "a rejected purchase must not charge the player")
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
