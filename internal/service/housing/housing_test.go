package housing

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d := testutil.NewDB(t)
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

func TestBuyHouseAppliesInventoryBonus(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 1000000)
	require.NoError(t, err)

	err = svc.BuyHouse(1, "gilded_palace")
	require.NoError(t, err)

	var u model.User
	require.NoError(t, st.DB.Where("user_id = ?", 1).First(&u).Error)
	assert.Equal(t, Houses["gilded_palace"].InventoryBonus, u.ExtraInvSlots)
	assert.Equal(t, Houses["gilded_palace"].PetSlotsBonus, u.ExtraPetSlots)

	limit, err := st.InventoryLimit(st.DB, 1)
	require.NoError(t, err)
	assert.Equal(t, store.BaseInventoryLimit+Houses["gilded_palace"].InventoryBonus, limit)
}

func TestUpgradeLevelReappliesBonus(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 10000)
	require.NoError(t, err)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "wooden_shack", Level: 1, LastCollected: &now,
		IsActive: true,
	}).Error)

	require.NoError(t, svc.UpgradeLevel(1))

	var u model.User
	require.NoError(t, st.DB.Where("user_id = ?", 1).First(&u).Error)
	assert.Equal(t, Houses["wooden_shack"].InventoryBonus, u.ExtraInvSlots)
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
	_, err := st.UpdateBalance(1, 100000)
	require.NoError(t, err)
	require.NoError(t, st.AddItemRaw(st.DB, 1, "coal", 100))
	require.NoError(t, st.AddItemRaw(st.DB, 1, "copper_ore", 100))

	err = svc.StartConstruction(1, "merchant_office")
	require.NoError(t, err)

	h, _ := svc.GetHousing(1)
	require.NotNil(t, h.UnderConstruction)
	assert.Equal(t, "merchant_office", *h.UnderConstruction)
	assert.NotNil(t, h.FinishTime)

	bal, _ := st.GetBalance(1)
	assert.Equal(t, 96000, bal)
}

func TestStartConstructionCharges(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "wooden_shack", Level: 1, LastCollected: &now,
		IsActive: true,
	}).Error)
	// Not enough money.
	err := svc.StartConstruction(1, "merchant_office")
	assert.Contains(t, err.Error(), "not enough money")
	// Missing items.
	_, err = st.UpdateBalance(1, 100000)
	require.NoError(t, err)
	err = svc.StartConstruction(1, "merchant_office")
	assert.Contains(t, err.Error(), "missing coal")
}

func TestStartConstructionGates(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "wooden_shack", Level: 1, LastCollected: &now,
		IsActive: true,
	}).Error)
	_, err := st.UpdateBalance(1, 1000000)
	require.NoError(t, err)
	require.NoError(t, st.AddItemRaw(st.DB, 1, "coal", 100))
	require.NoError(t, st.AddItemRaw(st.DB, 1, "copper_ore", 100))

	// merchant_vault requires merchant_office first.
	err = svc.StartConstruction(1, "merchant_vault")
	assert.Contains(t, err.Error(), "requires merchant_office")

	// Owned upgrades can't be re-started.
	require.NoError(t, st.DB.Create(&model.UserHousingUpgrade{UserID: 1, UpgradeID: "merchant_office"}).Error)
	err = svc.StartConstruction(1, "merchant_office")
	assert.Contains(t, err.Error(), "already owned")

	// With merchant_office owned, the vault can start.
	require.NoError(t, st.AddItemRaw(st.DB, 1, "gold_nugget", 100))
	require.NoError(t, st.AddItemRaw(st.DB, 1, "silver_ore", 100))
	require.NoError(t, svc.StartConstruction(1, "merchant_vault"))

	// A second construction is rejected while one runs.
	err = svc.StartConstruction(1, "mystic_altar")
	assert.Contains(t, err.Error(), "in progress")
}

func TestCompleteConstructionNoConstruction(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	require.NoError(t, st.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "wooden_shack", Level: 1, LastCollected: &now,
		IsActive: true,
	}).Error)

	err := svc.CompleteConstruction(1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no construction")
}

func TestCompleteConstructionNotFinished(t *testing.T) {
	svc, st := testService(t)
	now := time.Now()
	uc := "merchant_office"
	future := now.Add(24 * time.Hour)
	require.NoError(t, st.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "wooden_shack", Level: 1, LastCollected: &now,
		IsActive:          true,
		UnderConstruction: &uc, FinishTime: &future,
	}).Error)

	err := svc.CompleteConstruction(1)
	assert.Contains(t, err.Error(), "not finished")
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

	err := svc.CompleteConstruction(1)
	require.NoError(t, err)

	var upg model.UserHousingUpgrade
	err = st.DB.Where("user_id = ? AND upgrade_id = ?", 1, "merchant_office").First(&upg).Error
	require.NoError(t, err)
	assert.Equal(t, "merchant_office", upg.UpgradeID)

	h, _ := svc.GetHousing(1)
	assert.Nil(t, h.UnderConstruction)
	assert.Nil(t, h.FinishTime)

	// Completing again fails politely.
	err = svc.CompleteConstruction(1)
	assert.Contains(t, err.Error(), "no construction")
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

func TestBankCapacityNoHouse(t *testing.T) {
	svc, _ := testService(t)
	cap, err := svc.BankCapacity(1)
	require.NoError(t, err)
	assert.Equal(t, 500, cap)
}

func TestBankCapacityPerHouse(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 1100000)
	require.NoError(t, err)

	require.NoError(t, svc.BuyHouse(1, "cardboard_box"))
	cap, err := svc.BankCapacity(1)
	require.NoError(t, err)
	assert.Equal(t, 600, cap)

	require.NoError(t, svc.BuyHouse(1, "wooden_shack"))
	cap, err = svc.BankCapacity(1)
	require.NoError(t, err)
	assert.Equal(t, 1000, cap)

	require.NoError(t, svc.BuyHouse(1, "gilded_palace"))
	cap, err = svc.BankCapacity(1)
	require.NoError(t, err)
	assert.Equal(t, 100000, cap)
}

func TestBankCapacityMerchantUpgrades(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 1000000)
	require.NoError(t, err)
	require.NoError(t, svc.BuyHouse(1, "gilded_palace"))

	cap, err := svc.BankCapacity(1)
	require.NoError(t, err)
	assert.Equal(t, 100000, cap)

	require.NoError(t, st.DB.Create(&model.UserHousingUpgrade{UserID: 1, UpgradeID: "merchant_office"}).Error)
	cap, err = svc.BankCapacity(1)
	require.NoError(t, err)
	assert.Equal(t, 120000, cap)

	require.NoError(t, st.DB.Create(&model.UserHousingUpgrade{UserID: 1, UpgradeID: "merchant_vault"}).Error)
	cap, err = svc.BankCapacity(1)
	require.NoError(t, err)
	assert.Equal(t, 240000, cap)
}

func TestBankCapacityRisesAfterCompleteConstruction(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 100000)
	require.NoError(t, err)
	require.NoError(t, svc.BuyHouse(1, "brick_house"))
	cap, err := svc.BankCapacity(1)
	require.NoError(t, err)
	assert.Equal(t, 5000, cap)

	// Build the merchant office through the real flow.
	require.NoError(t, st.AddItemRaw(st.DB, 1, "coal", 100))
	require.NoError(t, st.AddItemRaw(st.DB, 1, "copper_ore", 100))
	require.NoError(t, svc.StartConstruction(1, "merchant_office"))

	finished := time.Now().Add(-time.Minute)
	require.NoError(t, st.DB.Model(&model.UserHousing{}).Where("user_id = ? AND is_active = ?", 1, true).
		Update("finish_time", finished).Error)
	require.NoError(t, svc.CompleteConstruction(1))

	cap, err = svc.BankCapacity(1)
	require.NoError(t, err)
	assert.Equal(t, 6000, cap)
}
