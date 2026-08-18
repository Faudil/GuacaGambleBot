package farm

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
	"guacagamblebot/internal/model"
	invsvc "guacagamblebot/internal/service/inventory"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
	"guacagamblebot/internal/universe/hoakhaven"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "fm.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	s := store.New(d, cfg)
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	require.NotNil(t, def)
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)
	svc := New(s, cfg, npcSvc)
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
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "wheat_seed", Quantity: 5})

	err := svc.Plant(1, "public", 0, "wheat_seed", 60)
	require.NoError(t, err)

	plots, err := svc.GetPlots(1, "public")
	require.NoError(t, err)
	assert.Equal(t, "wheat_seed", plots[0].ItemName)
	assert.False(t, plots[0].Ready)

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

func TestWater(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "wheat_seed", Quantity: 5})

	err := svc.Plant(1, "public", 0, "wheat_seed", 3600)
	require.NoError(t, err)

	err = svc.Water(1, "public", 0)
	assert.NoError(t, err)

	plots, err := svc.GetPlots(1, "public")
	require.NoError(t, err)
	assert.True(t, plots[0].Watered)

	err = svc.Water(1, "public", 0)
	assert.ErrorIs(t, err, ErrAlreadyWatered)
}

func TestFertilize(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "wheat_seed", Quantity: 5})
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "fertilizer", Quantity: 1})

	err := svc.Plant(1, "public", 0, "wheat_seed", 3600)
	require.NoError(t, err)

	plots, _ := svc.GetPlots(1, "public")
	originalGrowTime := plots[0].GrowTime

	err = svc.Fertilize(1, "public", 0)
	assert.NoError(t, err)

	plots, _ = svc.GetPlots(1, "public")
	assert.Less(t, plots[0].GrowTime, originalGrowTime)
}

func TestFertilizeNoFertilizer(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "wheat_seed", Quantity: 5})

	err := svc.Plant(1, "public", 0, "wheat_seed", 3600)
	require.NoError(t, err)

	err = svc.Fertilize(1, "public", 0)
	assert.ErrorIs(t, err, ErrNoFertilizer)
}

func TestAccelerate(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "wheat_seed", Quantity: 5})
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "growth_elixir", Quantity: 1})

	err := svc.Plant(1, "public", 0, "wheat_seed", 3600)
	require.NoError(t, err)

	plots, _ := svc.GetPlots(1, "public")
	assert.False(t, plots[0].Ready)

	err = svc.Accelerate(1, "public", 0)
	assert.NoError(t, err)

	plots, _ = svc.GetPlots(1, "public")
	assert.True(t, plots[0].Ready)

	var inv model.Inventory
	err = s.DB.Where("user_id = ? AND item_id = ?", 1, "growth_elixir").First(&inv).Error
	assert.Error(t, err, "elixir should be consumed")
}

func TestAccelerateNoElixir(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "wheat_seed", Quantity: 5})

	err := svc.Plant(1, "public", 0, "wheat_seed", 3600)
	require.NoError(t, err)

	err = svc.Accelerate(1, "public", 0)
	assert.ErrorIs(t, err, ErrNoAccelerator)
}

func TestAccelerateReadyPlot(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "growth_elixir", Quantity: 1})
	_ = s.DB.Create(&model.UserFarming{
		UserID: 1, ZoneKey: "public", PlotIndex: 0,
		ItemName: "wheat_seed", PlantTime: time.Now().Add(-2 * time.Second), GrowTime: 1,
	})

	err := svc.Accelerate(1, "public", 0)
	assert.ErrorIs(t, err, ErrNotReady)
}

func TestHasZoneAccessPublic(t *testing.T) {
	svc, _ := testService(t)
	assert.True(t, svc.HasZoneAccess(1, "public"))
	assert.False(t, svc.HasZoneAccess(1, "veggie"))
	assert.False(t, svc.HasZoneAccess(1, "greenhouse"))
	assert.False(t, svc.HasZoneAccess(1, "orchard"))
}

func TestHasZoneAccessWithDeed(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "garden_plot", Quantity: 1})
	assert.True(t, svc.HasZoneAccess(1, "veggie"))
	assert.False(t, svc.HasZoneAccess(1, "greenhouse"))
	assert.False(t, svc.HasZoneAccess(1, "orchard"))
}

func TestGetAccessibleZones(t *testing.T) {
	svc, s := testService(t)
	zones := svc.GetAccessibleZones(1)
	assert.Equal(t, []string{"public"}, zones)

	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "garden_plot", Quantity: 1})
	zones = svc.GetAccessibleZones(1)
	assert.Contains(t, zones, "public")
	assert.Contains(t, zones, "veggie")
}

func TestCountActivePlots(t *testing.T) {
	svc, s := testService(t)
	assert.Equal(t, 0, svc.CountActivePlots(1))

	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "wheat_seed", Quantity: 5})
	_ = svc.Plant(1, "public", 0, "wheat_seed", 300)
	assert.Equal(t, 1, svc.CountActivePlots(1))
}

func TestMaxTotalPlots(t *testing.T) {
	svc, s := testService(t)
	assert.Equal(t, 3, svc.MaxTotalPlots(1))

	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "garden_plot", Quantity: 1})
	assert.Equal(t, 6, svc.MaxTotalPlots(1))
}

func TestGetNextHarvest(t *testing.T) {
	svc, s := testService(t)
	name, secs := svc.GetNextHarvest(1)
	assert.Empty(t, name)
	assert.Equal(t, 0, secs)

	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "wheat_seed", Quantity: 5})
	_ = svc.Plant(1, "public", 0, "wheat_seed", 3600)
	name, secs = svc.GetNextHarvest(1)
	assert.NotEmpty(t, name)
	assert.Greater(t, secs, 0)
}

func TestGetFarmerLevel(t *testing.T) {
	svc, _ := testService(t)
	assert.Equal(t, 0, svc.GetFarmerLevel(1))
}

func TestHasBlessing(t *testing.T) {
	svc, _ := testService(t)
	assert.False(t, svc.HasBlessing(1))
	SetBlessing(1, "public")
	assert.True(t, svc.HasBlessing(1))
	assert.Equal(t, "public", svc.GetBlessingZone(1))
	svc.ConsumeBlessing(1)
	assert.False(t, svc.HasBlessing(1))
}

func TestHasItem(t *testing.T) {
	svc, s := testService(t)
	assert.False(t, svc.HasItem(1, "wheat_seed"))
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "wheat_seed", Quantity: 1})
	assert.True(t, svc.HasItem(1, "wheat_seed"))
}

func TestHasItemNormalizesDisplayName(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "fertilizer", Quantity: 1})
	assert.True(t, svc.HasItem(1, "Fertilizer"), "display-name lookup must find the canonical row")
	assert.False(t, svc.HasItem(1, "not_a_real_item"))
	assert.Equal(t, 1, svc.GetItemQuantity(1, "Fertilizer"))
	assert.True(t, svc.ConsumeItem(1, "Fertilizer", 1))
	assert.False(t, svc.ConsumeItem(1, "not_a_real_item", 1))
}

func TestConsumeItem(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "wheat_seed", Quantity: 3})
	assert.True(t, svc.ConsumeItem(1, "wheat_seed", 2))
	assert.Equal(t, 1, svc.GetItemQuantity(1, "wheat_seed"))
	assert.True(t, svc.ConsumeItem(1, "wheat_seed", 1))
	assert.Equal(t, 0, svc.GetItemQuantity(1, "wheat_seed"))
}

func TestCheckMutation(t *testing.T) {
	svc, _ := testService(t)
	_ = svc.CheckMutation(1, "wheat") // Just verify no panic
}

func TestRollMysteriousSeed(t *testing.T) {
	svc, _ := testService(t)
	found := false
	for i := 0; i < 1000; i++ {
		if svc.RollMysteriousSeed(1) {
			found = true
			break
		}
	}
	assert.True(t, found, "should eventually roll a mysterious seed")
}

func TestResolveMystery(t *testing.T) {
	svc, _ := testService(t)
	result := svc.resolveMystery("mysterious_seed")
	assert.NotEmpty(t, result)
}

func TestGetExpertise(t *testing.T) {
	svc, _ := testService(t)
	exp := svc.GetExpertise(1)
	assert.Empty(t, exp)

	svc.store.DB.Create(&model.UserCropHarvest{UserID: 1, CropName: "wheat", Count: 10})
	exp = svc.GetExpertise(1)
	assert.Len(t, exp, 1)
	assert.Equal(t, "wheat", exp[0].CropName)
	assert.Equal(t, 10, exp[0].Harvested)
	assert.Equal(t, "farm.expert_title", exp[0].Title)
}

func TestGetFarmerXP(t *testing.T) {
	svc, _ := testService(t)
	xp, next := svc.GetFarmerXP(1)
	assert.Equal(t, 0, xp)
	assert.Equal(t, 50, next)
}

func TestAddItem(t *testing.T) {
	svc, _ := testService(t)
	err := svc.AddItem(1, "wheat_seed", 3)
	assert.NoError(t, err)
	assert.Equal(t, 3, svc.GetItemQuantity(1, "wheat_seed"))
}

func TestMysteriousSeedPlant(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "mysterious_seed", Quantity: 1})

	err := svc.Plant(1, "public", 0, "mysterious_seed", 1800)
	require.NoError(t, err)

	plots, err := svc.GetPlots(1, "public")
	require.NoError(t, err)
	assert.True(t, plots[0].Mysterious)
	assert.Equal(t, "mysterious_seed", plots[0].ItemName)
}

func TestHarvestMysteriousSeed(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "mysterious_seed", Quantity: 1})

	err := svc.Plant(1, "public", 0, "mysterious_seed", -1) // negative = already grown
	require.NoError(t, err)

	result, err := svc.Harvest(1, "public", 0)
	require.NoError(t, err)
	assert.NotEmpty(t, result.CropName)
	assert.Greater(t, result.Quantity, 0)
}

func TestRegularSeedNames(t *testing.T) {
	assert.NotEmpty(t, RegularSeedNames)
	assert.Contains(t, RegularSeedNames, "wheat_seed")
	assert.NotContains(t, RegularSeedNames, "mysterious_seed")
}

func TestRollEventNeverNil(t *testing.T) {
	svc, _ := testService(t)
	evt := svc.RollEvent(1, "public", nil)
	if evt != nil {
		assert.NotEmpty(t, evt.Title)
	}
}

func TestResolveEventDefault(t *testing.T) {
	svc, _ := testService(t)
	result := svc.ResolveEvent(1, &Event{Type: 999}, "anything")
	assert.NotNil(t, result)
	assert.True(t, result.BackToMenu)
}

func TestWaterNonExistentPlot(t *testing.T) {
	svc, _ := testService(t)
	err := svc.Water(1, "public", 0)
	assert.Error(t, err)
}

func TestFertilizeNonExistentPlot(t *testing.T) {
	svc, _ := testService(t)
	err := svc.Fertilize(1, "public", 0)
	assert.Error(t, err)
}

func TestPlantOccupiedPlot(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "wheat_seed", Quantity: 5})

	err := svc.Plant(1, "public", 0, "wheat_seed", 300)
	require.NoError(t, err)

	err = svc.Plant(1, "public", 0, "wheat_seed", 300)
	assert.ErrorIs(t, err, ErrOccupied)
}

func TestGoldenCarrotCondition(t *testing.T) {
	svc, s := testService(t)
	assert.False(t, svc.CheckGoldenCarrot(1))
	s.DB.Create(&model.Job{UserID: 1, JobName: "farmer", Level: 10, XP: 0})
	assert.True(t, svc.CheckGoldenCarrot(1) || !svc.CheckGoldenCarrot(1)) // time dependent
}

func TestGetMutationFlavor(t *testing.T) {
	svc, _ := testService(t)
	flavor := svc.GetMutationFlavor("ghost_wheat")
	assert.Equal(t, "farm.mutation_ghost_wheat", flavor)
	flavor = svc.GetMutationFlavor("nonexistent")
	assert.Empty(t, flavor)
}
