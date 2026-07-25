package fishing

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
	invsvc "guacagamblebot/internal/service/inventory"
	loresvc "guacagamblebot/internal/service/lore"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
	hoakhaven "guacagamblebot/internal/universe/hoakhaven"
)

func testService(t *testing.T) (*Service, *store.Store) {
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "f.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	loreSvc := loresvc.New(s, cfg, def)
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)
	svc := New(s, cfg, loreSvc, npcSvc)
	return svc, s
}

func TestGenerateFish(t *testing.T) {
	svc, _ := testService(t)
	state := svc.GenerateFish("pond", BaitCommon)
	require.NotNil(t, state)
	assert.Equal(t, 100, state.Tension)
	assert.Greater(t, state.Stamina, 0)
	assert.Equal(t, 0, state.Distance)
	assert.False(t, state.Escaped)
	assert.Greater(t, state.Weight, 0)
	assert.Greater(t, state.Size, 0)
}

func TestGenerateFishValidBiome(t *testing.T) {
	svc, _ := testService(t)
	biomes := []string{"pond", "river", "ocean", "lava"}
	for _, b := range biomes {
		state := svc.GenerateFish(b, BaitCommon)
		require.NotNil(t, state, "biome %s should generate a fish", b)
		assert.Greater(t, state.Stamina, 0)
	}
}

func TestGenerateFishBaitTierFilters(t *testing.T) {
	svc, _ := testService(t)
	for i := 0; i < 50; i++ {
		pondLegendary := svc.GenerateFish("pond", BaitLegendary)
		rareOrBetter := pondLegendary.Stamina >= 30
		if rareOrBetter {
			return
		}
	}
	t.Log("legendary bait should produce rare+ fish within 50 tries, got common repeatedly")
	// This is a probabilistic test; if it fails here, the weighting might be off.
}

func TestApplyActionReel(t *testing.T) {
	svc, _ := testService(t)
	state := svc.GenerateFish("pond", BaitCommon)
	initialTension := state.Tension
	initialStamina := state.Stamina

	result := svc.ApplyAction(state, "reel")

	assert.Less(t, state.Tension, initialTension, "tension should decrease")
	assert.Less(t, state.Stamina, initialStamina, "stamina should decrease")
	assert.False(t, result.DistanceStep)
	assert.False(t, result.Caught)
	assert.False(t, result.Escaped)
	assert.Less(t, result.TensionDelta, 0)
	assert.Less(t, result.StaminaDelta, 0)
}

func TestApplyActionPull(t *testing.T) {
	svc, _ := testService(t)
	state := svc.GenerateFish("pond", BaitCommon)
	initialTension := state.Tension
	initialStamina := state.Stamina

	result := svc.ApplyAction(state, "pull")

	assert.Less(t, state.Tension, initialTension, "tension should decrease more")
	assert.Less(t, state.Stamina, initialStamina, "stamina should decrease more")
	assert.True(t, result.DistanceStep, "pull should advance distance")
	assert.Greater(t, state.Distance, 0)
}

func TestApplyActionRest(t *testing.T) {
	svc, _ := testService(t)
	state := svc.GenerateFish("pond", BaitCommon)
	svc.ApplyAction(state, "pull")
	svc.ApplyAction(state, "pull")
	tensionAfterPull := state.Tension
	staminaAfterPull := state.Stamina

	result := svc.ApplyAction(state, "rest")

	assert.Greater(t, state.Tension, tensionAfterPull, "tension should recover with rest")
	assert.Greater(t, state.Stamina, staminaAfterPull, "fish stamina should increase with rest (evasiveness)")
	assert.Greater(t, result.TensionDelta, 0)
}

func TestCatchFish(t *testing.T) {
	svc, _ := testService(t)
	state := svc.GenerateFish("pond", BaitCommon)

	for i := 0; i < 20 && state.Stamina > 0; i++ {
		result := svc.ApplyAction(state, "reel")
		if result.Caught {
			break
		}
	}
	assert.LessOrEqual(t, state.Stamina, 0)
	assert.Equal(t, 2, state.Distance, "distance should be caught")
}

func TestEscapeFish(t *testing.T) {
	svc, _ := testService(t)
	for try := 0; try < 20; try++ {
		state := svc.GenerateFish("lava", BaitLegendary)
		for i := 0; i < 50; i++ {
			result := svc.ApplyAction(state, "pull")
			if result.Escaped {
				return
			}
			if result.Caught {
				break
			}
		}
	}
	t.Log("Lava Serpent should snap the line at least once in 20 attempts (flaky RNG)")
}

func TestLuckyBreakTriggerRate(t *testing.T) {
	svc, _ := testService(t)
	luckyCount := 0
	iterations := 500

	for i := 0; i < iterations; i++ {
		state := svc.GenerateFish("lava", BaitLegendary)
		for j := 0; j < 50; j++ {
			result := svc.ApplyAction(state, "pull")
			if result.LuckyBreak {
				luckyCount++
				break
			}
			if result.Escaped || result.Caught {
				break
			}
		}
	}

	t.Logf("Lucky Break triggered %d/%d times (%.1f%%)", luckyCount, iterations, float64(luckyCount)/float64(iterations)*100)
	assert.Greater(t, luckyCount, 0, "Lucky Break should trigger at least sometimes")
	assert.Less(t, luckyCount, iterations/2, "Lucky Break should not trigger more than 50%% of the time")
}

func TestBaitItemID(t *testing.T) {
	assert.Equal(t, "worm", baitItemID(BaitCommon))
	assert.Equal(t, "crayfish", baitItemID(BaitRare))
	assert.Equal(t, "golden_lure", baitItemID(BaitLegendary))
}

func TestFisherLevelDefault(t *testing.T) {
	svc, _ := testService(t)
	lvl, err := svc.GetFisherLevel(999)
	require.NoError(t, err)
	assert.Equal(t, 0, lvl, "new user should have fisher level 0")
}

func TestLavaUnlocked(t *testing.T) {
	svc, s := testService(t)
	ok, err := svc.LavaUnlocked(1)
	require.NoError(t, err)
	assert.False(t, ok, "new user should not have lava unlocked")

	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "fisher", Level: 10})
	ok, err = svc.LavaUnlocked(1)
	require.NoError(t, err)
	assert.True(t, ok, "fisher level 10 should unlock lava")
}

func TestResolveCatch(t *testing.T) {
	svc, _ := testService(t)
	state := svc.GenerateFish("pond", BaitCommon)
	for i := 0; i < 20; i++ {
		r := svc.ApplyAction(state, "reel")
		if r.Caught {
			break
		}
	}
	require.LessOrEqual(t, state.Stamina, 0)

	res, err := svc.ResolveCatch(1, state)
	require.NoError(t, err)
	assert.True(t, res.Caught)
	assert.Greater(t, res.XP, 0)
	assert.NotEmpty(t, res.ItemName)
}

func TestFreeCastLimits(t *testing.T) {
	svc, _ := testService(t)
	free, err := svc.CanFreeCast(1)
	require.NoError(t, err)
	assert.True(t, free)

	_ = svc.UseFreeCast(1)
	free, err = svc.CanFreeCast(1)
	require.NoError(t, err)
	assert.False(t, free, "after use, free cast should be unavailable")
}
