package inventory

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
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "inv.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestGetInventoryEmpty(t *testing.T) {
	svc, _ := testService(t)
	res, err := svc.GetInventory(1)
	require.NoError(t, err)
	assert.Empty(t, res.Entries)
	assert.Equal(t, 0, res.Current)
	assert.Equal(t, int64(1), res.UserID)
}

func TestGetInventoryWithItems(t *testing.T) {
	svc, st := testService(t)
	dbItem := model.Item{Name: "charbon", Price: 5, Description: "", EffectType: "resource"}
	require.NoError(t, st.DB.Create(&dbItem).Error)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: 1, ItemID: dbItem.ID, Quantity: 10}).Error)

	res, err := svc.GetInventory(1)
	require.NoError(t, err)
	assert.Len(t, res.Entries, 1)
	assert.Equal(t, 10, res.Current)
	assert.Equal(t, "charbon", res.Entries[0].ItemName)
}

func TestHasItemTrue(t *testing.T) {
	svc, st := testService(t)
	dbItem := model.Item{Name: "charbon", Price: 5, Description: "", EffectType: "resource"}
	require.NoError(t, st.DB.Create(&dbItem).Error)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: 1, ItemID: dbItem.ID, Quantity: 5}).Error)

	assert.True(t, svc.HasItem(1, "charbon", 3))
}

func TestHasItemFalse(t *testing.T) {
	svc, _ := testService(t)
	assert.False(t, svc.HasItem(1, "charbon", 1))
}

func TestHasItemInsufficientQuantity(t *testing.T) {
	svc, st := testService(t)
	dbItem := model.Item{Name: "charbon", Price: 5, Description: "", EffectType: "resource"}
	require.NoError(t, st.DB.Create(&dbItem).Error)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: 1, ItemID: dbItem.ID, Quantity: 2}).Error)

	assert.False(t, svc.HasItem(1, "charbon", 5))
}

func TestAddItem(t *testing.T) {
	svc, st := testService(t)
	err := svc.AddItem(st.DB, 1, "charbon", 3)
	require.NoError(t, err)

	var inv model.Inventory
	err = st.DB.Where("user_id = ? AND item_id = (SELECT id FROM items WHERE name = ?)", 1, "charbon").First(&inv).Error
	require.NoError(t, err)
	assert.Equal(t, 3, inv.Quantity)
}

func TestRemoveItem(t *testing.T) {
	svc, st := testService(t)
	dbItem := model.Item{Name: "charbon", Price: 5, Description: "", EffectType: "resource"}
	require.NoError(t, st.DB.Create(&dbItem).Error)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: 1, ItemID: dbItem.ID, Quantity: 10}).Error)

	err := svc.RemoveItem(st.DB, 1, "charbon", 4)
	require.NoError(t, err)

	var inv model.Inventory
	st.DB.Where("user_id = ? AND item_id = ?", 1, dbItem.ID).First(&inv)
	assert.Equal(t, 6, inv.Quantity)
}
