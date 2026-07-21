package market

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
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

func TestGetMarketPricesReturnsCategories(t *testing.T) {
	svc, _ := testService(t)
	cats := svc.GetMarketPrices()
	assert.NotEmpty(t, cats)
	for _, cat := range cats {
		assert.NotEmpty(t, cat.Name)
		assert.NotEmpty(t, cat.Items)
		for _, mi := range cat.Items {
			assert.NotNil(t, mi.Item)
			assert.Greater(t, mi.CurrentPrice, 0)
			assert.Greater(t, mi.Multiplier, 0.0)
		}
	}
}

func TestSellItemNotSellable(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.SellItem(1, "old journal", 1)
	assert.ErrorIs(t, err, ErrNotSellable)
}

func TestSellItemNotFound(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.SellItem(1, "nonexistent", 1)
	assert.ErrorIs(t, err, ErrNotSellable)
}

func TestSellItemNoItem(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.SellItem(1, "charbon", 1)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSellItemInsufficientQuantity(t *testing.T) {
	svc, st := testService(t)
	dbItem := model.Item{Name: "charbon", Price: 5, Description: "", EffectType: "resource"}
	require.NoError(t, st.DB.Create(&dbItem).Error)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: 1, ItemID: dbItem.ID, Quantity: 1}).Error)

	_, err := svc.SellItem(1, "charbon", 5)
	assert.ErrorIs(t, err, ErrNoItem)
}

func TestSellItemSuccess(t *testing.T) {
	svc, st := testService(t)
	dbItem := model.Item{Name: "charbon", Price: 5, Description: "", EffectType: "resource"}
	require.NoError(t, st.DB.Create(&dbItem).Error)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: 1, ItemID: dbItem.ID, Quantity: 10}).Error)

	gain, err := svc.SellItem(1, "charbon", 3)
	require.NoError(t, err)
	assert.Greater(t, gain, 0)

	var inv model.Inventory
	st.DB.Where("user_id = ? AND item_id = ?", 1, dbItem.ID).First(&inv)
	assert.Equal(t, 7, inv.Quantity)

	bal, _ := st.GetBalance(1)
	assert.Equal(t, 100+gain, bal)
}
