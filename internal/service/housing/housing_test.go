package housing

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

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
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "house.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 1000, DailyAmount: 50}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestGetHousingNotFound(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.GetHousing(1)
	assert.Error(t, err)
}

func TestBuyHouseInvalidType(t *testing.T) {
	svc, _ := testService(t)
	err := svc.BuyHouse(1, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid house type")
}

func TestBuyHouseNotEnoughMoney(t *testing.T) {
	svc, _ := testService(t)
	err := svc.BuyHouse(1, "brick_house")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enough money")
}

func TestBuyHouseSuccess(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 1000)
	require.NoError(t, err)

	err = svc.BuyHouse(1, "cardboard_box")
	require.NoError(t, err)

	h, err := svc.GetHousing(1)
	require.NoError(t, err)
	assert.Equal(t, "cardboard_box", h.HouseType)
	assert.Equal(t, 1, h.Level)
	assert.NotNil(t, h.LastCollected)
}

func TestUpgradeLevelMax(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 1000)
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "cardboard_box", Level: 1, LastCollected: &now,
		IsActive: true,
	}).Error)

	err = svc.UpgradeLevel(1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max level")
}

func TestUpgradeLevelSuccess(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 10000)
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "wooden_shack", Level: 1, LastCollected: &now,
		IsActive: true,
	}).Error)

	err = svc.UpgradeLevel(1)
	require.NoError(t, err)

	h, err := svc.GetHousing(1)
	require.NoError(t, err)
	assert.Equal(t, 2, h.Level)
}

func TestRename(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "cardboard_box", Level: 1, LastCollected: &now,
		IsActive: true,
	}).Error)

	err := svc.Rename(1, "Mon château")
	require.NoError(t, err)

	h, _ := svc.GetHousing(1)
	require.NotNil(t, h.CustomName)
	assert.Equal(t, "Mon château", *h.CustomName)
}

func TestSetColor(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "cardboard_box", Level: 1, LastCollected: &now,
		IsActive: true,
	}).Error)

	err := svc.SetColor(1, "#FF0000")
	require.NoError(t, err)

	h, _ := svc.GetHousing(1)
	require.NotNil(t, h.CustomColor)
	assert.Equal(t, "#FF0000", *h.CustomColor)
}

func TestCollectNotOwned(t *testing.T) {
	svc, _ := testService(t)
	_, _, err := svc.Collect(1)
	assert.Error(t, err)
}

func TestCollectWithNoElapsedTime(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	_, err := st.UpdateBalance(1, 1000)
	require.NoError(t, err)

	require.NoError(t, st.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "cardboard_box", Level: 1, LastCollected: &now,
		IsActive: true,
	}).Error)

	income, items, err := svc.Collect(1)
	require.NoError(t, err)
	assert.Equal(t, 0, income)
	assert.Empty(t, items)
}

func TestGetCollectInfo(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	_, err := st.UpdateBalance(1, 1000)
	require.NoError(t, err)
	require.NoError(t, st.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "cardboard_box", Level: 1, LastCollected: &now,
		IsActive: true,
	}).Error)

	info, err := svc.GetCollectInfo(1)
	require.NoError(t, err)
	assert.Equal(t, 0, info.Income)
	assert.Empty(t, info.Items)
}

func TestGetStoredItemsDefault(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "cardboard_box", Level: 1, LastCollected: &now,
		IsActive: true,
	}).Error)

	items, err := svc.GetStoredItems(1)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestStartConstruction(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "wooden_shack", Level: 1, LastCollected: &now,
		IsActive: true,
	}).Error)

	err := svc.StartConstruction("1", "merchant_office", 4)
	require.NoError(t, err)

	h, _ := svc.GetHousing(1)
	require.NotNil(t, h.UnderConstruction)
	assert.Equal(t, "merchant_office", *h.UnderConstruction)
	assert.NotNil(t, h.FinishTime)
}

func TestCompleteConstructionNoConstruction(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "wooden_shack", Level: 1, LastCollected: &now,
		IsActive: true,
	}).Error)

	err := svc.CompleteConstruction("1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no construction")
}

func TestCompleteConstructionSuccess(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	uc := "merchant_office"
	require.NoError(t, st.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "wooden_shack", Level: 1, LastCollected: &now,
		IsActive:          true,
		UnderConstruction: &uc, FinishTime: &now,
	}).Error)

	err := svc.CompleteConstruction("1")
	require.NoError(t, err)

	var upg model.UserHousingUpgrade
	err = st.DB.Where("user_id = ? AND upgrade_id = ?", 1, "merchant_office").First(&upg).Error
	require.NoError(t, err)
	assert.Equal(t, "merchant_office", upg.UpgradeID)

	h, _ := svc.GetHousing(1)
	assert.Nil(t, h.UnderConstruction)
	assert.Nil(t, h.FinishTime)
}

func TestBuyHouseAlreadyOwned(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 10000)
	require.NoError(t, err)

	require.NoError(t, svc.BuyHouse(1, "cardboard_box"))

	err = svc.BuyHouse(1, "cardboard_box")
	assert.True(t, errors.Is(err, ErrAlreadyOwned))
}

func TestBuyMultipleHousesNewOneActive(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 10000)
	require.NoError(t, err)

	require.NoError(t, svc.BuyHouse(1, "cardboard_box"))
	require.NoError(t, svc.BuyHouse(1, "wooden_shack"))

	houses, err := svc.ListHouses(1)
	require.NoError(t, err)
	assert.Len(t, houses, 2)

	h, err := svc.GetHousing(1)
	require.NoError(t, err)
	assert.Equal(t, "wooden_shack", h.HouseType)
	assert.True(t, h.IsActive)

	var old model.UserHousing
	require.NoError(t, st.DB.Where("user_id = ? AND house_type = ?", 1, "cardboard_box").First(&old).Error)
	assert.False(t, old.IsActive)
}

func TestSwitchHouse(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 10000)
	require.NoError(t, err)

	require.NoError(t, svc.BuyHouse(1, "cardboard_box"))
	require.NoError(t, svc.BuyHouse(1, "wooden_shack"))

	err = svc.SwitchHouse(1, "cardboard_box")
	require.NoError(t, err)

	h, err := svc.GetHousing(1)
	require.NoError(t, err)
	assert.Equal(t, "cardboard_box", h.HouseType)
	assert.True(t, h.IsActive)

	var newHouse model.UserHousing
	require.NoError(t, st.DB.Where("user_id = ? AND house_type = ?", 1, "wooden_shack").First(&newHouse).Error)
	assert.False(t, newHouse.IsActive)
}

func TestSwitchHouseNotOwned(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 10000)
	require.NoError(t, err)

	require.NoError(t, svc.BuyHouse(1, "cardboard_box"))

	err = svc.SwitchHouse(1, "mansion")
	assert.True(t, errors.Is(err, ErrAlreadyOwned))
}

func TestUpgradeOnlyAffectsActiveHouse(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 100000)
	require.NoError(t, err)

	require.NoError(t, svc.BuyHouse(1, "wooden_shack"))
	require.NoError(t, svc.BuyHouse(1, "brick_house"))
	require.NoError(t, svc.UpgradeLevel(1))

	active, err := svc.GetHousing(1)
	require.NoError(t, err)
	assert.Equal(t, 2, active.Level)

	var other model.UserHousing
	require.NoError(t, st.DB.Where("user_id = ? AND house_type = ?", 1, "wooden_shack").First(&other).Error)
	assert.Equal(t, 1, other.Level)
}
