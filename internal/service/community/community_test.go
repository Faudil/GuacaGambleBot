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

func TestGetAllProjects(t *testing.T) {
	svc, _ := testService(t)
	projects, err := svc.GetAllProjects(1)
	require.NoError(t, err)
	assert.Len(t, projects, len(Buildings))

	for _, p := range projects {
		assert.NotEmpty(t, p.Key)
		assert.Greater(t, p.MaxLevel, 0)
		assert.NotEmpty(t, p.Costs)
	}
}

func TestInvestUnknownBuilding(t *testing.T) {
	svc, _ := testService(t)
	ok, err := svc.Invest(1, 1, "nonexistent", "money", 100)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestInvestMaxLevel(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.ServerProject{
		ServerID: 1, ProjectID: "market", Level: 10,
	}).Error)

	ok, err := svc.Invest(1, 1, "market", "money", 100)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestInvestWrongResource(t *testing.T) {
	svc, _ := testService(t)
	ok, err := svc.Invest(1, 1, "market", "nonexistent_resource", 100)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestInvestAlreadyComplete(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.ServerProjectContribution{
		ServerID: 1, ProjectID: "market", ResourceType: "money", AmountContributed: 100000,
	}).Error)

	ok, err := svc.Invest(1, 1, "market", "money", 100)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestInvestNotEnoughMoney(t *testing.T) {
	svc, _ := testService(t)
	ok, err := svc.Invest(1, 1, "market", "money", 1000000)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestInvestSuccess(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 50000)
	require.NoError(t, err)

	ok, err := svc.Invest(1, 1, "market", "money", 10000)
	require.NoError(t, err)
	assert.False(t, ok) // not all resources done yet

	bal, _ := st.GetBalance(1)
	assert.Equal(t, 140000, bal) // starting 100k + 50k - 10k
}

func TestInvestCompletesProject(t *testing.T) {
	svc, st := testService(t)
	_, err := st.UpdateBalance(1, 50000)
	require.NoError(t, err)

	ok, err := svc.Invest(1, 1, "statue", "money", 10000)
	require.NoError(t, err)
	assert.False(t, ok) // Pebble still needed

	ok, err = svc.Invest(1, 1, "statue", "Pebble", 500)
	require.NoError(t, err)
	assert.True(t, ok)

	lvl, _ := svc.GetProjectLevel(1, "statue")
	assert.Equal(t, 1, lvl)
}

func TestGetUserStatsCreatesDefault(t *testing.T) {
	svc, _ := testService(t)
	stats, err := svc.GetUserStats(1, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.UserID)
	assert.Equal(t, int64(1), stats.ServerID)
	assert.Equal(t, 0, stats.TotalMoneyInvested)
}
