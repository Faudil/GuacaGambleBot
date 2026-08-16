package community

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
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "comm.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100000, DailyAmount: 50}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestGetProjectLevelDefaultsToZero(t *testing.T) {
	svc, _ := testService(t)
	lvl, err := svc.GetProjectLevel(1, "market")
	require.NoError(t, err)
	assert.Equal(t, 0, lvl)
}

func TestGetProjectLevelExisting(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.ServerProject{
		ServerID: 1, ProjectID: "market", Level: 3,
	}).Error)

	lvl, err := svc.GetProjectLevel(1, "market")
	require.NoError(t, err)
	assert.Equal(t, 3, lvl)
}

func TestGetAllProjectsDeterministicOrder(t *testing.T) {
	svc, _ := testService(t)
	projects, err := svc.GetAllProjects(1)
	require.NoError(t, err)
	assert.Len(t, projects, len(Buildings))
	require.Equal(t, []string{"market", "bank", "hospital", "statue"}, []string{
		projects[0].Key, projects[1].Key, projects[2].Key, projects[3].Key,
	})
	for _, p := range projects {
		assert.NotEmpty(t, p.Key)
		assert.Greater(t, p.MaxLevel, 0)
		assert.NotEmpty(t, p.Costs)
		assert.NotEmpty(t, p.Progress)
	}
}

func TestInvestUnknownBuilding(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.Invest(1, 1, "nonexistent", "money", 100)
	assert.ErrorIs(t, err, ErrBuildingNotFound)
}

func TestInvestMaxLevel(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.ServerProject{
		ServerID: 1, ProjectID: "market", Level: 10,
	}).Error)

	_, err := svc.Invest(1, 1, "market", "money", 100)
	assert.ErrorIs(t, err, ErrMaxLevel)
}

func TestInvestWrongResource(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.Invest(1, 1, "market", "nonexistent_resource", 100)
	assert.ErrorIs(t, err, ErrResourceNotNeeded)
}

func TestInvestInvalidAmount(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.Invest(1, 1, "market", "money", 0)
	assert.ErrorIs(t, err, ErrInvalidAmount)
	_, err = svc.Invest(1, 1, "market", "money", -5)
	assert.ErrorIs(t, err, ErrInvalidAmount)
}

func TestInvestAlreadyComplete(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.ServerProjectContribution{
		ServerID: 1, ProjectID: "market", ResourceType: "money", AmountContributed: 100000,
	}).Error)

	_, err := svc.Invest(1, 1, "market", "money", 100)
	assert.ErrorIs(t, err, ErrResourceFull)
}

func TestInvestNotEnoughMoney(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.Invest(1, 1, "market", "money", 1000000)
	assert.ErrorIs(t, err, ErrNotEnoughMoney)
}

func TestInvestMoneySuccess(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 50000)
	require.NoError(t, err)

	res, err := svc.Invest(1, 1, "market", "money", 10000)
	require.NoError(t, err)
	assert.Equal(t, 10000, res.Invested)
	assert.False(t, res.LeveledUp) // pebbles still needed

	bal, _ := st.GetBalance(1)
	assert.Equal(t, 140000, bal) // starting 100k + 50k - 10k
}

func TestInvestMoneyCapsAtQuota(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 50000)
	require.NoError(t, err)

	res, err := svc.Invest(1, 1, "statue", "money", 999999)
	require.NoError(t, err)
	assert.Equal(t, 10000, res.Invested) // capped at the remaining quota
}

func TestInvestItemNotEnough(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.Inventory{
		UserID: 1, ItemID: "pebble", Quantity: 10,
	}).Error)

	_, err := svc.Invest(1, 1, "statue", "pebble", 500)
	assert.ErrorIs(t, err, ErrNotEnoughItems)
}

func TestInvestItemDeductsInventory(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.Inventory{
		UserID: 1, ItemID: "pebble", Quantity: 600,
	}).Error)

	res, err := svc.Invest(1, 1, "statue", "pebble", 500)
	require.NoError(t, err)
	assert.Equal(t, 500, res.Invested)

	var inv model.Inventory
	require.NoError(t, st.DB.Where("user_id = ? AND item_id = ?", 1, "pebble").First(&inv).Error)
	assert.Equal(t, 100, inv.Quantity)

	stats, _ := svc.GetUserStats(1, 1)
	assert.Equal(t, 500, stats.TotalItemsInvested)
}

func TestInvestCompletesProject(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 50000)
	require.NoError(t, err)
	require.NoError(t, st.DB.Create(&model.Inventory{
		UserID: 1, ItemID: "pebble", Quantity: 500,
	}).Error)

	res, err := svc.Invest(1, 1, "statue", "money", 10000)
	require.NoError(t, err)
	assert.False(t, res.LeveledUp) // pebble still needed

	res, err = svc.Invest(1, 1, "statue", "pebble", 500)
	require.NoError(t, err)
	assert.True(t, res.LeveledUp)
	assert.Equal(t, 1, res.NewLevel)

	lvl, _ := svc.GetProjectLevel(1, "statue")
	assert.Equal(t, 1, lvl)
}

func TestInvestLevelUpClearsContributions(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 50000)
	require.NoError(t, err)
	require.NoError(t, st.DB.Create(&model.Inventory{
		UserID: 1, ItemID: "pebble", Quantity: 500,
	}).Error)

	_, err = svc.Invest(1, 1, "statue", "money", 10000)
	require.NoError(t, err)
	_, err = svc.Invest(1, 1, "statue", "pebble", 500)
	require.NoError(t, err)

	var count int64
	st.DB.Model(&model.ServerProjectContribution{}).
		Where("server_id = ? AND project_id = ?", 1, "statue").Count(&count)
	assert.Zero(t, count)

	info, err := svc.GetProjectInfo(1, "statue")
	require.NoError(t, err)
	assert.Equal(t, 1, info.Level)
	for _, p := range info.Progress {
		assert.Zero(t, p.Contributed)
	}
}

func TestGetProjectInfoUnknown(t *testing.T) {
	svc, _ := testService(t)
	info, err := svc.GetProjectInfo(1, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, info)
}

func TestGetUserStatsCreatesDefault(t *testing.T) {
	svc, _ := testService(t)
	stats, err := svc.GetUserStats(1, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.UserID)
	assert.Equal(t, int64(1), stats.ServerID)
	assert.Equal(t, 0, stats.TotalMoneyInvested)
}

func TestGetTopContributors(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.UserCommunityStat{
		UserID: 1, ServerID: 1, TotalMoneyInvested: 1000,
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserCommunityStat{
		UserID: 2, ServerID: 1, TotalMoneyInvested: 100, TotalItemsInvested: 50,
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserCommunityStat{
		UserID: 3, ServerID: 2, TotalMoneyInvested: 99999,
	}).Error) // different server, must be excluded
	require.NoError(t, st.DB.Create(&model.UserCommunityStat{
		UserID: 4, ServerID: 1, TotalMoneyInvested: 0,
	}).Error) // zero total, must be excluded

	top, err := svc.GetTopContributors(1, 5)
	require.NoError(t, err)
	require.Len(t, top, 2)
	assert.Equal(t, int64(1), top[0].UserID)
	assert.Equal(t, 1000, top[0].Total)
	assert.Equal(t, int64(2), top[1].UserID)
	assert.Equal(t, 150, top[1].Total)
}
