package bank

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

func testStore(t *testing.T) *store.Store {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "x.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	return store.New(d, &config.Config{StartingBalance: 100, DailyAmount: 50})
}

func TestDeposit(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	res, err := svc.Deposit(1, 40)
	require.NoError(t, err)
	assert.Equal(t, 60, res.Wallet)
	assert.Equal(t, 40, res.Bank)
	assert.Equal(t, 40, res.Deposited)
	assert.Equal(t, 500, res.MaxBank)
}

func TestDepositInvalid(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	_, err := svc.Deposit(1, 0)
	assert.ErrorIs(t, err, ErrAmount)
	_, err = svc.Deposit(1, -5)
	assert.ErrorIs(t, err, ErrAmount)
}

func TestDepositInsufficient(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	_, err := svc.Deposit(1, 500)
	assert.ErrorIs(t, err, ErrNoMoney)
}

func TestDepositCappedAtDefaultLimit(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	_, err := s.UpdateBalance(1, 600)
	require.NoError(t, err)

	res, err := svc.Deposit(1, 400)
	require.NoError(t, err)
	assert.Equal(t, 300, res.Wallet)
	assert.Equal(t, 400, res.Bank)
	assert.Equal(t, 400, res.Deposited)

	res, err = svc.Deposit(1, 200)
	require.NoError(t, err)
	assert.Equal(t, 200, res.Wallet)
	assert.Equal(t, 500, res.Bank)
	assert.Equal(t, 100, res.Deposited)

	_, err = svc.Deposit(1, 10)
	assert.ErrorIs(t, err, ErrBankFull)
}

func TestDepositCappedByHouse(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	require.NoError(t, s.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "brick_house", Level: 1, IsActive: true, StoredItems: "{}",
	}).Error)
	_, err := s.UpdateBalance(1, 10000)
	require.NoError(t, err)

	res, err := svc.Deposit(1, 5500)
	require.NoError(t, err)
	assert.Equal(t, 5000, res.Bank)
	assert.Equal(t, 5000, res.Deposited)
	assert.Equal(t, 5000, res.MaxBank)
	assert.Equal(t, 5100, res.Wallet)

	_, err = svc.Deposit(1, 1)
	assert.ErrorIs(t, err, ErrBankFull)
}

func TestDepositCappedByHouseAndMerchantUpgrades(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	require.NoError(t, s.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "gilded_palace", Level: 1, IsActive: true, StoredItems: "{}",
	}).Error)
	require.NoError(t, s.DB.Create(&model.UserHousingUpgrade{UserID: 1, UpgradeID: "merchant_office"}).Error)
	require.NoError(t, s.DB.Create(&model.UserHousingUpgrade{UserID: 1, UpgradeID: "merchant_vault"}).Error)
	_, err := s.UpdateBalance(1, 3000000)
	require.NoError(t, err)

	res, err := svc.Deposit(1, 250000)
	require.NoError(t, err)
	assert.Equal(t, 240000, res.MaxBank)
	assert.Equal(t, 240000, res.Bank)
	assert.Equal(t, 240000, res.Deposited)
}

func TestWithdraw(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	_, err := svc.Deposit(1, 40)
	require.NoError(t, err)

	wallet, bank, _, err := svc.Withdraw(1, 15)
	require.NoError(t, err)
	assert.Equal(t, 75, wallet)
	assert.Equal(t, 25, bank)
}

func TestWithdrawInsufficient(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	_, err := svc.Deposit(1, 40)
	require.NoError(t, err)

	_, _, _, err = svc.Withdraw(1, 100)
	assert.ErrorIs(t, err, ErrNoMoney)
}

func TestInfo(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	_, err := svc.Deposit(1, 100)
	require.NoError(t, err)

	wallet, bank, interest, maxBank, err := svc.Info(1)
	require.NoError(t, err)
	assert.Equal(t, 0, wallet)
	assert.Equal(t, 100, bank)
	assert.Equal(t, 10, interest)
	assert.Equal(t, 500, maxBank)
}

func TestClampOnInfo(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	_, err := s.AdjustColumn(1, "bank", 800)
	require.NoError(t, err)

	wallet, bank, interest, maxBank, err := svc.Info(1)
	require.NoError(t, err)
	assert.Equal(t, 400, wallet)
	assert.Equal(t, 500, bank)
	assert.Equal(t, 50, interest)
	assert.Equal(t, 500, maxBank)
}

func TestInfoMaxBankFollowsHousing(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	// No house: default 500.
	_, _, _, maxBank, err := svc.Info(1)
	require.NoError(t, err)
	assert.Equal(t, 500, maxBank)

	// Brick house: 5,000.
	require.NoError(t, s.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "brick_house", Level: 1, IsActive: true, StoredItems: "{}",
	}).Error)
	_, _, _, maxBank, err = svc.Info(1)
	require.NoError(t, err)
	assert.Equal(t, 5000, maxBank)

	// Merchant office + vault: 5,000 * 1.2 * 2 = 12,000.
	require.NoError(t, s.DB.Create(&model.UserHousingUpgrade{UserID: 1, UpgradeID: "merchant_office"}).Error)
	require.NoError(t, s.DB.Create(&model.UserHousingUpgrade{UserID: 1, UpgradeID: "merchant_vault"}).Error)
	_, _, _, maxBank, err = svc.Info(1)
	require.NoError(t, err)
	assert.Equal(t, 12000, maxBank)
}

func TestClampOnDeposit(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	_, err := s.AdjustColumn(1, "bank", 800)
	require.NoError(t, err)

	res, err := svc.Deposit(1, 100)
	require.ErrorIs(t, err, ErrBankFull)
	assert.Equal(t, 400, res.Wallet)
	assert.Equal(t, 500, res.Bank)
	assert.Equal(t, 500, res.MaxBank)
}

func TestClampOnDepositWithRoom(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	require.NoError(t, s.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "brick_house", Level: 1, IsActive: true, StoredItems: "{}",
	}).Error)
	_, err := s.AdjustColumn(1, "bank", 1800)
	require.NoError(t, err)
	_, err = s.UpdateBalance(1, 500)
	require.NoError(t, err)

	res, err := svc.Deposit(1, 300)
	require.NoError(t, err)
	assert.Equal(t, 5000, res.MaxBank)
	assert.Equal(t, 2100, res.Bank)
	assert.Equal(t, 300, res.Deposited)
	assert.Equal(t, 300, res.Wallet)
}

func TestClampOnWithdraw(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	_, err := s.AdjustColumn(1, "bank", 800)
	require.NoError(t, err)

	wallet, bank, maxBank, err := svc.Withdraw(1, 100)
	require.NoError(t, err)
	assert.Equal(t, 500, wallet)
	assert.Equal(t, 400, bank)
	assert.Equal(t, 500, maxBank)
}

func TestClampRespectsHouseCap(t *testing.T) {
	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	require.NoError(t, s.DB.Create(&model.UserHousing{
		UserID: 1, HouseType: "brick_house", Level: 1, IsActive: true, StoredItems: "{}",
	}).Error)
	_, err := s.AdjustColumn(1, "bank", 6000)
	require.NoError(t, err)

	wallet, bank, _, maxBank, err := svc.Info(1)
	require.NoError(t, err)
	assert.Equal(t, 1100, wallet)
	assert.Equal(t, 5000, bank)
	assert.Equal(t, 5000, maxBank)
}

func TestClampDisabled(t *testing.T) {
	BankOverCapAutoFix = false
	t.Cleanup(func() { BankOverCapAutoFix = true })

	s := testStore(t)
	svc := New(s, &config.Config{StartingBalance: 100, DailyAmount: 50})

	_, err := s.AdjustColumn(1, "bank", 800)
	require.NoError(t, err)

	wallet, bank, _, _, err := svc.Info(1)
	require.NoError(t, err)
	assert.Equal(t, 100, wallet)
	assert.Equal(t, 800, bank)
}
