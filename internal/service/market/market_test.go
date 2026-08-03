package market

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
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mkt.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func seedInventory(t *testing.T, st *store.Store, userID int64, itemID string, qty int) {
	t.Helper()
	err := st.DB.Create(&model.Inventory{UserID: userID, ItemID: itemID, Quantity: qty}).Error
	require.NoError(t, err)
}

func TestGetMarketInitializesRotation(t *testing.T) {
	svc, st := testService(t)

	views, total, err := svc.GetMarket("all", 1, ItemsPerPage)
	require.NoError(t, err)
	assert.Greater(t, total, 0)
	assert.GreaterOrEqual(t, len(views), 1)

	// Should have created MarketState rows
	var count int64
	st.DB.Model(&model.MarketState{}).Where("is_active = ?", true).Count(&count)
	assert.Equal(t, int64(total), count)
}

func TestGetMarketCategoryFilter(t *testing.T) {
	svc, _ := testService(t)

	views, total, err := svc.GetMarket("mining", 1, ItemsPerPage)
	require.NoError(t, err)
	assert.Greater(t, total, 0)

	for _, v := range views {
		assert.Equal(t, items.Mining, v.Item.Category)
	}
}

func TestGetMarketCategoryFilterEmpty(t *testing.T) {
	svc, _ := testService(t)

	views, total, err := svc.GetMarket("food", 1, ItemsPerPage)
	require.NoError(t, err)
	_ = views
	assert.LessOrEqual(t, total, RotationSize)
}

func TestBuyItemSuccess(t *testing.T) {
	svc, st := testService(t)

	// Force a specific item to be active and cheap
	today := time.Now().Format("2006-01-02")
	weekID := currentWeekID()
	err := st.DB.Where("1=1").Delete(&model.MarketState{}).Error
	require.NoError(t, err)
	err = st.DB.Create(&model.MarketState{
		ItemID: "coal", CurrentPrice: 5, LastReset: today, WeekID: weekID, IsActive: true,
	}).Error
	require.NoError(t, err)

	cost, _, _, err := svc.BuyItem(1, "coal", 3)
	require.NoError(t, err)
	assert.Equal(t, 15, cost)

	var inv model.Inventory
	err = st.DB.Where("user_id = ? AND item_id = ?", 1, "coal").First(&inv).Error
	require.NoError(t, err)
	assert.Equal(t, 3, inv.Quantity)

	bal, _ := st.GetBalance(1)
	assert.Equal(t, 85, bal)
}

func TestBuyItemInsufficientFunds(t *testing.T) {
	svc, st := testService(t)

	today := time.Now().Format("2006-01-02")
	weekID := currentWeekID()
	st.DB.Where("1=1").Delete(&model.MarketState{})
	st.DB.Create(&model.MarketState{
		ItemID: "coal", CurrentPrice: 9999, LastReset: today, WeekID: weekID, IsActive: true,
	})

	_, _, _, err := svc.BuyItem(1, "coal", 1)
	assert.ErrorIs(t, err, ErrNoMoney)
}

func TestBuyItemNotActive(t *testing.T) {
	svc, st := testService(t)

	// Establish the week's rotation first, then deactivate coal
	_ = svc.ensureWeekRotation()
	st.DB.Model(&model.MarketState{}).Where("item_id = ?", "coal").Update("is_active", false)

	_, _, _, err := svc.BuyItem(1, "coal", 1)
	assert.ErrorIs(t, err, ErrNotActive)
}

func TestSellItemSuccess(t *testing.T) {
	svc, st := testService(t)

	seedInventory(t, st, 1, "coal", 10)
	today := time.Now().Format("2006-01-02")
	weekID := currentWeekID()
	st.DB.Where("1=1").Delete(&model.MarketState{})
	st.DB.Create(&model.MarketState{
		ItemID: "coal", CurrentPrice: 5, LastReset: today, WeekID: weekID, IsActive: true,
	})

	gain, _, _, err := svc.SellItem(1, "coal", 3)
	require.NoError(t, err)
	assert.Greater(t, gain, 0)

	var inv model.Inventory
	st.DB.Where("user_id = ? AND item_id = ?", 1, "coal").First(&inv)
	assert.Equal(t, 7, inv.Quantity)

	bal, _ := st.GetBalance(1)
	assert.Equal(t, 100+gain, bal)
}

func TestSellItemNotOwned(t *testing.T) {
	svc, st := testService(t)

	today := time.Now().Format("2006-01-02")
	weekID := currentWeekID()
	st.DB.Where("1=1").Delete(&model.MarketState{})
	st.DB.Create(&model.MarketState{
		ItemID: "coal", CurrentPrice: 5, LastReset: today, WeekID: weekID, IsActive: true,
	})

	_, _, _, err := svc.SellItem(1, "coal", 1)
	assert.ErrorIs(t, err, ErrNoItem)
}

func TestSellItemNotActive(t *testing.T) {
	svc, st := testService(t)

	seedInventory(t, st, 1, "coal", 5)
	// Establish the week's rotation first, then deactivate coal
	_ = svc.ensureWeekRotation()
	st.DB.Model(&model.MarketState{}).Where("item_id = ?", "coal").Update("is_active", false)

	_, _, _, err := svc.SellItem(1, "coal", 1)
	assert.ErrorIs(t, err, ErrNotActive)
}

func TestSellItemNotMarketable(t *testing.T) {
	svc, _ := testService(t)

	// Equipment is not marketable
	_, _, _, err := svc.SellItem(1, "stick", 1)
	assert.ErrorIs(t, err, ErrNotActive)
}

func TestBuyItemAdjustsPriceUp(t *testing.T) {
	svc, st := testService(t)

	today := time.Now().Format("2006-01-02")
	weekID := currentWeekID()
	st.DB.Where("1=1").Delete(&model.MarketState{})
	st.DB.Create(&model.MarketState{
		ItemID: "coal", CurrentPrice: 5, LastReset: today, WeekID: weekID, IsActive: true,
	})

	_, _, _, err := svc.BuyItem(1, "coal", 10)
	require.NoError(t, err)

	var st2 model.MarketState
	st.DB.Where("item_id = ?", "coal").First(&st2)
	assert.Greater(t, st2.CurrentPrice, 5)
	assert.Equal(t, 10, st2.DailyBought)
}

func TestSellItemAdjustsPriceDown(t *testing.T) {
	svc, st := testService(t)

	seedInventory(t, st, 1, "coal", 100)
	today := time.Now().Format("2006-01-02")
	weekID := currentWeekID()
	st.DB.Where("1=1").Delete(&model.MarketState{})
	st.DB.Create(&model.MarketState{
		ItemID: "coal", CurrentPrice: 5, LastReset: today, WeekID: weekID, IsActive: true,
	})

	_, _, _, err := svc.SellItem(1, "coal", 10)
	require.NoError(t, err)

	var st2 model.MarketState
	st.DB.Where("item_id = ?", "coal").First(&st2)
	assert.Less(t, st2.CurrentPrice, 5)
	assert.Equal(t, 10, st2.DailySold)
}

func TestBuyItemInvalidQuantity(t *testing.T) {
	svc, _ := testService(t)
	_, _, _, err := svc.BuyItem(1, "coal", 0)
	assert.ErrorIs(t, err, ErrInvalidQty)
	_, _, _, err = svc.BuyItem(1, "coal", -1)
	assert.ErrorIs(t, err, ErrInvalidQty)
}

func TestSellItemInvalidQuantity(t *testing.T) {
	svc, _ := testService(t)
	_, _, _, err := svc.SellItem(1, "coal", 0)
	assert.ErrorIs(t, err, ErrInvalidQty)
}

func TestBuyItemNotFound(t *testing.T) {
	svc, _ := testService(t)
	_, _, _, err := svc.BuyItem(1, "nonexistent_item", 1)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetMarketPagination(t *testing.T) {
	svc, _ := testService(t)

	_, total, err := svc.GetMarket("all", 1, ItemsPerPage)
	require.NoError(t, err)
	assert.Greater(t, total, 0)
	assert.LessOrEqual(t, total, RotationSize)

	// Get second page
	views2, _, err := svc.GetMarket("all", 2, ItemsPerPage)
	require.NoError(t, err)
	_ = views2
}

func TestDayResetDecaysPrice(t *testing.T) {
	svc, st := testService(t)

	today := time.Now().Format("2006-01-02")
	weekID := currentWeekID()
	st.DB.Where("1=1").Delete(&model.MarketState{})

	// Create item with price below base — should decay upward
	coal := items.Get("coal")
	st.DB.Create(&model.MarketState{
		ItemID: "coal", CurrentPrice: 1, LastReset: "2000-01-01", WeekID: weekID, IsActive: true,
	})

	_ = svc.ensureDayReset()

	var st2 model.MarketState
	st.DB.Where("item_id = ?", "coal").First(&st2)
	assert.Equal(t, today, st2.LastReset)
	assert.Greater(t, st2.CurrentPrice, 1)
	assert.LessOrEqual(t, st2.CurrentPrice, coal.Price)
}

func TestRotateMarketSelectsDistinctItems(t *testing.T) {
	svc, st := testService(t)

	weekID := currentWeekID()
	err := svc.rotateMarket(weekID)
	require.NoError(t, err)

	var active []model.MarketState
	st.DB.Where("is_active = ?", true).Find(&active)
	assert.Equal(t, RotationSize, len(active))

	// All item IDs should be unique
	seen := make(map[string]bool)
	for _, st := range active {
		assert.False(t, seen[st.ItemID], "duplicate item: %s", st.ItemID)
		seen[st.ItemID] = true
	}
}

func TestMarketableItemFilter(t *testing.T) {
	all := items.MarketableItems()
	assert.Greater(t, len(all), 0)

	for _, it := range all {
		assert.True(t, it.IsMarketable())
	}

	// Equipment should NOT be marketable
	stick := items.Get("stick")
	require.NotNil(t, stick)
	assert.False(t, stick.IsMarketable())

	// Seeds are Materials — not marketable
	wheatSeed := items.Get("wheat_seed")
	require.NotNil(t, wheatSeed)
	assert.False(t, wheatSeed.IsMarketable())
}
