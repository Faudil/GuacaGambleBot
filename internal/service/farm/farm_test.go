package farm

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
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "fm.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	s := store.New(d, cfg)
	svc := New(s, cfg)
	return svc, s
}

func TestGetPlotsEmpty(t *testing.T) {
	svc, _ := testService(t)
	plots, err := svc.GetPlots(1, "public")
	require.NoError(t, err)
	assert.Len(t, plots, 3)
	for _, p := range plots {
		assert.Empty(t, p.ItemName)
	}
}

func TestPlantAndHarvest(t *testing.T) {
	svc, s := testService(t)
	// Give user a seed
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "wheat_seed", Quantity: 5})

	err := svc.Plant(1, "public", 0, "wheat_seed", 60)
	require.NoError(t, err)

	plots, err := svc.GetPlots(1, "public")
	require.NoError(t, err)
	assert.Equal(t, "wheat_seed", plots[0].ItemName)
	assert.False(t, plots[0].Ready)

	// Harvest should fail since it hasn't grown
	_, err = svc.Harvest(1, "public", 0)
	assert.Error(t, err)
}

func TestPlantNoSeed(t *testing.T) {
	svc, _ := testService(t)
	err := svc.Plant(1, "public", 0, "wheat_seed", 60)
	assert.ErrorIs(t, err, ErrNoSeed)
}

func TestSeedsDefined(t *testing.T) {
	assert.NotEmpty(t, Crops)
	assert.NotEmpty(t, Seeds)
	assert.Equal(t, len(Crops), len(Seeds))
}
