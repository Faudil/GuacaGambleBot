package item_manager

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
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "im.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestTransferItemSuccess(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(2, 500)
	require.NoError(t, err)

	dbItem := model.Item{Name: "charbon", Price: 5, Description: "", EffectType: "resource"}
	require.NoError(t, st.DB.Create(&dbItem).Error)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: 1, ItemID: dbItem.ID, Quantity: 3}).Error)

	result := svc.TransferItem(1, 2, "charbon", 50)
	assert.Equal(t, TradeSuccess, result)

	var sellerInv model.Inventory
	st.DB.Where("user_id = ? AND item_id = ?", 1, dbItem.ID).First(&sellerInv)
	assert.Equal(t, 2, sellerInv.Quantity)

	buyerBal, _ := st.GetBalance(2)
	assert.Equal(t, 550, buyerBal)

	sellerBal, _ := st.GetBalance(1)
	assert.Equal(t, 150, sellerBal)
}

func TestTransferItemNoItem(t *testing.T) {
	svc, _ := testService(t)
	result := svc.TransferItem(1, 2, "nonexistent", 50)
	assert.Equal(t, TradeNoItem, result)
}

func setQty(t *testing.T, st *store.Store, userID, itemID int64, qty int) {
	t.Helper()
	require.NoError(t, st.DB.Model(&model.Inventory{}).
		Where("user_id = ? AND item_id = ?", userID, itemID).
		UpdateColumn("quantity", qty).Error)
}

func TestTransferItemInsufficientQuantity(t *testing.T) {
	svc, st := testService(t)
	dbItem := model.Item{Name: "charbon", Price: 5, Description: "", EffectType: "resource"}
	require.NoError(t, st.DB.Create(&dbItem).Error)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: 1, ItemID: dbItem.ID}).Error)
	setQty(t, st, 1, dbItem.ID, 0)

	result := svc.TransferItem(1, 2, "charbon", 50)
	assert.Equal(t, TradeNoItem, result)
}

func TestTransferItemNoMoney(t *testing.T) {
	svc, st := testService(t)
	dbItem := model.Item{Name: "charbon", Price: 5, Description: "", EffectType: "resource"}
	require.NoError(t, st.DB.Create(&dbItem).Error)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: 1, ItemID: dbItem.ID}).Error)
	setQty(t, st, 1, dbItem.ID, 3)
	_, err := st.UpdateBalance(2, -100) // set buyer balance to 0
	require.NoError(t, err)

	result := svc.TransferItem(1, 2, "charbon", 50)
	assert.Equal(t, TradeNoMoney, result)
}
