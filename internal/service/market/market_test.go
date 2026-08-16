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
	svc, st := testService(t)

	// The weekly rotation is random, so seed a known mining item to make this
	// test deterministic (a mining item may not be selected for a given week).
	today := time.Now().Format("2006-01-02")
	weekID := currentWeekID()
	st.DB.Where("1=1").Delete(&model.MarketState{})
	st.DB.Create(&model.MarketState{
		ItemID: "coal", CurrentPrice: 5, LastReset: today, WeekID: weekID, IsActive: true,
	})

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

func TestGetPlayerSellItems(t *testing.T) {
	svc, st := testService(t)

	// Deterministic rotation: coal is active at a custom price this week.
	today := time.Now().Format("2006-01-02")
	weekID := currentWeekID()
	require.NoError(t, st.DB.Where("1=1").Delete(&model.MarketState{}).Error)
	require.NoError(t, st.DB.Create(&model.MarketState{
		ItemID: "coal", CurrentPrice: 8, LastReset: today, WeekID: weekID, IsActive: true,
	}).Error)

	// coal: in rotation; wheat: vendor rate (5 * 50% = 2); rotten_plant: price 0, not sellable.
	seedInventory(t, st, 1, "coal", 3)
	seedInventory(t, st, 1, "wheat", 5)
	seedInventory(t, st, 1, "rotten_plant", 9)

	views, total, err := svc.GetPlayerSellItems(1, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, total, "unsellable items must be excluded")

	// Rotation items come first, at the market price.
	assert.Equal(t, "coal", views[0].Item.ID)
	assert.True(t, views[0].InRotation)
	assert.Equal(t, 8, views[0].UnitPrice)
	assert.Equal(t, 3, views[0].Owned)

	// Non-rotation items sell at the vendor rate.
	assert.Equal(t, "wheat", views[1].Item.ID)
	assert.False(t, views[1].InRotation)
	assert.Equal(t, vendorPrice(5), views[1].UnitPrice)
	assert.Equal(t, 5, views[1].Owned)
}

func TestGetPlayerSellItemsPagination(t *testing.T) {
	svc, st := testService(t)

	require.NoError(t, st.DB.Where("1=1").Delete(&model.MarketState{}).Error)
	seedInventory(t, st, 1, "coal", 1)
	seedInventory(t, st, 1, "wheat", 1)
	seedInventory(t, st, 1, "trout", 1)

	page1, total, err := svc.GetPlayerSellItems(1, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, page1, 2)

	page2, total, err := svc.GetPlayerSellItems(1, 2, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, page2, 1)

	// Past the end clamps to an empty page without erroring.
	page3, total, err := svc.GetPlayerSellItems(1, 9, 2)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Empty(t, page3)
}

func TestGetPlayerSellItemsEmptyInventory(t *testing.T) {
	svc, _ := testService(t)

	views, total, err := svc.GetPlayerSellItems(1, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, views)
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
	// Laddered: bids 6 + 7 + 8 (coal base 5, buy step 1, 2% spread)
	assert.Equal(t, 21, cost)

	var inv model.Inventory
	err = st.DB.Where("user_id = ? AND item_id = ?", 1, "coal").First(&inv).Error
	require.NoError(t, err)
	assert.Equal(t, 3, inv.Quantity)

	bal, _ := st.GetBalance(1)
	assert.Equal(t, 79, bal)

	var st2 model.MarketState
	st.DB.Where("item_id = ?", "coal").First(&st2)
	assert.Equal(t, 8, st2.CurrentPrice)
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
	// Laddered: asks 4 + 3 + 2 (coal base 5, sell step 1, 2% spread)
	assert.Equal(t, 9, gain)

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

func TestSellItemDeactivatedUsesVendorPrice(t *testing.T) {
	svc, st := testService(t)

	seedInventory(t, st, 1, "coal", 5)
	// Establish the week's rotation first, then deactivate coal
	_ = svc.ensureWeekRotation()
	st.DB.Model(&model.MarketState{}).Where("item_id = ?", "coal").Update("is_active", false)

	gain, _, _, err := svc.SellItem(1, "coal", 1)
	require.NoError(t, err)
	assert.Equal(t, vendorPrice(items.Get("coal").Price), gain)

	// A vendor sale must not touch the market state
	var st2 model.MarketState
	st.DB.Where("item_id = ?", "coal").First(&st2)
	assert.Equal(t, 0, st2.DailySold)
}

func TestSellItemVendorPrice(t *testing.T) {
	svc, st := testService(t)

	// Equipment is not in the rotation but has a price, so it sells at the
	// fixed vendor rate.
	seedInventory(t, st, 1, "stick", 2)

	gain, _, _, err := svc.SellItem(1, "stick", 1)
	require.NoError(t, err)
	assert.Equal(t, vendorPrice(items.Get("stick").Price), gain)

	// No MarketState row should be created for a vendor sale
	var count int64
	st.DB.Model(&model.MarketState{}).Where("item_id = ?", "stick").Count(&count)
	assert.Equal(t, int64(0), count)

	var inv model.Inventory
	st.DB.Where("user_id = ? AND item_id = ?", 1, "stick").First(&inv)
	assert.Equal(t, 1, inv.Quantity)

	bal, _ := st.GetBalance(1)
	assert.Equal(t, 100+gain, bal)
}

func TestSellItemNotSellable(t *testing.T) {
	svc, st := testService(t)

	// Price-0 items (legendary sets, boss trinkets) can never be sold
	seedInventory(t, st, 1, "dragon_slayer_sword", 1)

	_, _, _, err := svc.SellItem(1, "dragon_slayer_sword", 1)
	assert.ErrorIs(t, err, ErrNotSellable)
}

func TestSellSeedVendorPrice(t *testing.T) {
	svc, st := testService(t)

	// Seeds are Materials — not in the rotation, but sellable at vendor rate
	seedInventory(t, st, 1, "wheat_seed", 3)

	gain, _, _, err := svc.SellItem(1, "wheat_seed", 2)
	require.NoError(t, err)
	assert.Equal(t, vendorPrice(items.Get("wheat_seed").Price)*2, gain)

	var inv model.Inventory
	st.DB.Where("user_id = ? AND item_id = ?", 1, "wheat_seed").First(&inv)
	assert.Equal(t, 1, inv.Quantity)
}

func TestSellItemVendorGoldenTouch(t *testing.T) {
	svc, st := testService(t)

	seedInventory(t, st, 1, "stick", 5)
	require.NoError(t, st.SetActiveBuff(1, "golden_touch"))

	gain, _, _, err := svc.SellItem(1, "stick", 1)
	require.NoError(t, err)
	assert.Equal(t, vendorPrice(items.Get("stick").Price)*2, gain)
}

func TestSellPricesFor(t *testing.T) {
	svc, st := testService(t)

	today := time.Now().Format("2006-01-02")
	weekID := currentWeekID()
	st.DB.Where("1=1").Delete(&model.MarketState{})
	st.DB.Create(&model.MarketState{
		ItemID: "coal", CurrentPrice: 9, LastReset: today, WeekID: weekID, IsActive: true,
	})

	prices := svc.SellPricesFor([]string{"coal", "stick", "wheat_seed"})
	assert.Equal(t, 9, prices["coal"])
	assert.Equal(t, vendorPrice(items.Get("stick").Price), prices["stick"])
	assert.Equal(t, vendorPrice(items.Get("wheat_seed").Price), prices["wheat_seed"])
}

func TestBuyItemAdjustsPriceUp(t *testing.T) {
	svc, st := testService(t)

	today := time.Now().Format("2006-01-02")
	weekID := currentWeekID()
	st.DB.Where("1=1").Delete(&model.MarketState{})
	st.DB.Create(&model.MarketState{
		ItemID: "coal", CurrentPrice: 5, LastReset: today, WeekID: weekID, IsActive: true,
	})

	_, err := st.UpdateBalance(1, 1000)
	require.NoError(t, err)

	cost, _, _, err := svc.BuyItem(1, "coal", 10)
	require.NoError(t, err)
	assert.Equal(t, 105, cost)

	var st2 model.MarketState
	st.DB.Where("item_id = ?", "coal").First(&st2)
	assert.Equal(t, 15, st2.CurrentPrice)
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

	gain, _, _, err := svc.SellItem(1, "coal", 10)
	require.NoError(t, err)
	assert.Equal(t, 16, gain)

	var st2 model.MarketState
	st.DB.Where("item_id = ?", "coal").First(&st2)
	assert.Equal(t, 1, st2.CurrentPrice)
	assert.Equal(t, 10, st2.DailySold)
}

func TestBuyThenSellRoundTripLosesMoney(t *testing.T) {
	svc, st := testService(t)

	today := time.Now().Format("2006-01-02")
	weekID := currentWeekID()
	st.DB.Where("1=1").Delete(&model.MarketState{})
	st.DB.Create(&model.MarketState{
		ItemID: "coal", CurrentPrice: 5, LastReset: today, WeekID: weekID, IsActive: true,
	})

	_, err := st.UpdateBalance(1, 100000)
	require.NoError(t, err)
	start, err := st.GetBalance(1)
	require.NoError(t, err)

	cost, _, _, err := svc.BuyItem(1, "coal", 50)
	require.NoError(t, err)
	gain, _, _, err := svc.SellItem(1, "coal", 50)
	require.NoError(t, err)

	bal, _ := st.GetBalance(1)
	assert.Equal(t, start-cost+gain, bal)
	assert.Less(t, gain, cost, "selling back immediately must not cover the buy cost")
	assert.Less(t, bal, start, "a pump-and-dump round trip must lose money")
}

func TestPumpAndDump999LosesMoney(t *testing.T) {
	svc, st := testService(t)

	today := time.Now().Format("2006-01-02")
	weekID := currentWeekID()
	st.DB.Where("1=1").Delete(&model.MarketState{})
	st.DB.Create(&model.MarketState{
		ItemID: "coal", CurrentPrice: 5, LastReset: today, WeekID: weekID, IsActive: true,
	})

	_, err := st.UpdateBalance(1, 1000000)
	require.NoError(t, err)
	start, _ := st.GetBalance(1)

	cost, _, _, err := svc.BuyItem(1, "coal", 999)
	require.NoError(t, err)
	gain, _, _, err := svc.SellItem(1, "coal", 999)
	require.NoError(t, err)

	bal, _ := st.GetBalance(1)
	assert.Less(t, gain, cost)
	assert.Less(t, bal, start, "999-unit pump-and-dump must lose money")

	var st2 model.MarketState
	st.DB.Where("item_id = ?", "coal").First(&st2)
	assert.Equal(t, 1, st2.CurrentPrice, "sellback should crash the price back to the floor")
}

func TestLargeBuyClampsAtCeiling(t *testing.T) {
	svc, st := testService(t)

	today := time.Now().Format("2006-01-02")
	weekID := currentWeekID()
	st.DB.Where("1=1").Delete(&model.MarketState{})
	st.DB.Create(&model.MarketState{
		ItemID: "coal", CurrentPrice: 5, LastReset: today, WeekID: weekID, IsActive: true,
	})

	_, err := st.UpdateBalance(1, 1000000)
	require.NoError(t, err)

	cost, _, _, err := svc.BuyItem(1, "coal", 999)
	require.NoError(t, err)

	var st2 model.MarketState
	st.DB.Where("item_id = ?", "coal").First(&st2)
	assert.Equal(t, 25, st2.CurrentPrice)
	// 20 units climb 5->25 (bids 6..25) plus 979 units at bid 26
	assert.Equal(t, 25764, cost)
}

func TestLargeSellClampsAtFloor(t *testing.T) {
	svc, st := testService(t)

	seedInventory(t, st, 1, "coal", 999)
	today := time.Now().Format("2006-01-02")
	weekID := currentWeekID()
	st.DB.Where("1=1").Delete(&model.MarketState{})
	st.DB.Create(&model.MarketState{
		ItemID: "coal", CurrentPrice: 5, LastReset: today, WeekID: weekID, IsActive: true,
	})

	gain, _, _, err := svc.SellItem(1, "coal", 999)
	require.NoError(t, err)

	var st2 model.MarketState
	st.DB.Where("item_id = ?", "coal").First(&st2)
	assert.Equal(t, 1, st2.CurrentPrice)
	// asks 4 + 3 + 2 then 996 units at 1
	assert.Equal(t, 1005, gain)
}

func TestSpreadEdgesAtFloor(t *testing.T) {
	assert.Equal(t, 2, buyBid(1))
	assert.Equal(t, 6, buyBid(5))
	assert.Equal(t, 26, buyBid(25))
	assert.Equal(t, 1, sellAsk(1))
	assert.Equal(t, 4, sellAsk(5))
	assert.Equal(t, 24, sellAsk(25))
	assert.Equal(t, 1, sellAsk(2))
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
