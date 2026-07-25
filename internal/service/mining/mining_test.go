package mining

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
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "m.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	s := store.New(d, cfg)
	svc := New(s, cfg)
	return svc, s
}

func TestDescend(t *testing.T) {
	svc, _ := testService(t)
	res, err := svc.Descend(1, 1, nil, BranchCareful, "", 0)
	require.NoError(t, err)
	if !res.Collapsed {
		assert.NotNil(t, res.Item)
		assert.NotEmpty(t, res.Bag)
	}
}

func TestDescendCollapse(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 1, XP: 0})
	bag := []BagEntry{}
	collapsed := false
	for i := 0; i < 100; i++ {
		res, err := svc.Descend(1, 40, bag, BranchAggressive, "", 0)
		require.NoError(t, err)
		if res.Collapsed {
			collapsed = true
			break
		}
		bag = res.Bag
	}
	assert.True(t, collapsed, "should have collapsed at high depth")
}

func TestDescendCanGoAnyDepth(t *testing.T) {
	svc, _ := testService(t)
	// Even at level 1, Descend processes normally at any depth
	for depth := 1; depth <= 50; depth++ {
		res, err := svc.Descend(1, depth, nil, BranchCareful, "", 0)
		require.NoError(t, err)
		// Result is valid: either collapsed or got a result
		if res.Collapsed {
			break
		}
	}
}

func TestDescendRestGivesNoLoot(t *testing.T) {
	svc, _ := testService(t)
	res, err := svc.Descend(1, 1, nil, BranchRest, "", 0)
	require.NoError(t, err)
	assert.False(t, res.Collapsed, "rest should not collapse")
	assert.Nil(t, res.Item, "rest should give no item")
}

func TestDescendLevelReducesRisk(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 50, XP: 0})
	_ = s.DB.Create(&model.Job{UserID: 2, JobName: "miner", Level: 1, XP: 0})

	collapseL50 := 0
	collapseL1 := 0
	trials := 40

	for i := 0; i < trials; i++ {
		r1, err := svc.Descend(1, 15, nil, BranchCareful, "", 0)  // Level 50
		require.NoError(t, err)
		r2, err := svc.Descend(2, 15, nil, BranchCareful, "", 0)  // Level 1
		require.NoError(t, err)
		if r1.Collapsed {
			collapseL50++
		}
		if r2.Collapsed {
			collapseL1++
		}
	}

	// Level 50: (14*5) - 10 - 75 = 70 - 85 = -15 -> 0 -> floor 5%
	// Level 1:  (14*5) - 10 - 1  = 70 - 11 = 59%
	t.Logf("Level 50 collapses: %d/%d, Level 1 collapses: %d/%d", collapseL50, trials, collapseL1, trials)
	assert.Greater(t, collapseL1, collapseL50,
		"level 1 should collapse far more often than level 50")
}

func TestDescendHiddenChamber(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 1, XP: 0})
	found := false
	for i := 0; i < 500; i++ {
		bag := []BagEntry{}
		res, err := svc.Descend(1, 40, bag, BranchAggressive, "", 0)
		require.NoError(t, err)
		if res.Event != nil && res.Event.Type == "hidden_chamber" {
			found = true
			assert.False(t, res.Collapsed, "hidden chamber should prevent collapse")
			break
		}
	}
	assert.True(t, found, "should have found a hidden chamber eventually")
}

func TestLeaveMine(t *testing.T) {
	svc, _ := testService(t)
	bag := []BagEntry{{Name: "pebble", Count: 3}, {Name: "coal", Count: 1}}
	res, err := svc.LeaveMine(1, bag, "")
	require.NoError(t, err)
	assert.Equal(t, bag, res.Bag)
	assert.Greater(t, res.XP, 0)
}

func TestLeaveMineEmpty(t *testing.T) {
	svc, _ := testService(t)
	res, err := svc.LeaveMine(1, nil, "")
	require.NoError(t, err)
	assert.Empty(t, res.Bag)
	assert.Equal(t, 0, res.XP)
}

func TestLeaveMineAddsCharacterXP(t *testing.T) {
	svc, s := testService(t)
	bag := []BagEntry{{Name: "pebble", Count: 3}}
	_, err := svc.LeaveMine(1, bag, "")
	require.NoError(t, err)

	c, err := s.GetCharacter(1)
	require.NoError(t, err)
	assert.Greater(t, c.XP, 0)
}

func TestMineReinforceBuffPreventsCollapse(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 1, XP: 0})
	_ = s.SetActiveBuff(1, "reinforce")

	res, err := svc.Descend(1, 40, nil, BranchAggressive, "", 0)
	require.NoError(t, err)
	assert.False(t, res.Collapsed, "reinforce should prevent collapse on first descend")

	has, _ := s.HasActiveBuff(1, "reinforce")
	assert.False(t, has, "reinforce should be consumed after one descend")
}

func TestMineScavengerBuffDoublesItems(t *testing.T) {
	svc, s := testService(t)
	_ = s.SetActiveBuff(1, "scavenger")

	bag := []BagEntry{{Name: "pebble", Count: 2}, {Name: "coal", Count: 1}}
	res, err := svc.LeaveMine(1, bag, "")
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(res.Bag), 2)
	for _, e := range res.Bag {
		if e.Name == "pebble" {
			assert.Greater(t, e.Count, 2)
		}
	}

	has, _ := s.HasActiveBuff(1, "scavenger")
	assert.False(t, has, "scavenger should be consumed after leave")
}

func TestGetMinerLevelDefault(t *testing.T) {
	svc, _ := testService(t)
	lvl, err := svc.GetMinerLevel(999)
	require.NoError(t, err)
	assert.Equal(t, 1, lvl, "new users should have miner level 1")
}

func TestAvailableToolsByLevel(t *testing.T) {
	t1 := AvailableTools(1)
	assert.Len(t, t1, 1)
	assert.Equal(t, "", t1[0].ItemID)

	t5 := AvailableTools(5)
	assert.Len(t, t5, 2)

	t10 := AvailableTools(10)
	assert.Len(t, t10, 3)
}

func TestLockedToolsByLevel(t *testing.T) {
	l1 := LockedTools(1)
	assert.Len(t, l1, 2)

	l5 := LockedTools(5)
	assert.Len(t, l5, 1)

	l10 := LockedTools(10)
	assert.Len(t, l10, 0)
}

func TestLoreAtDepth(t *testing.T) {
	assert.NotNil(t, LoreAtDepth(13))
	assert.NotNil(t, LoreAtDepth(21))
	assert.NotNil(t, LoreAtDepth(33))
	assert.NotNil(t, LoreAtDepth(42))
	assert.Nil(t, LoreAtDepth(1))
}

func TestLootAtDepth(t *testing.T) {
	assert.Len(t, lootAtDepth(1), 2)
	assert.Contains(t, lootAtDepth(1), MineItem{"pebble", 1})
	assert.Len(t, lootAtDepth(5), 3)
	assert.Len(t, lootAtDepth(15), 3)     // kethari_crystal tier
	assert.Len(t, lootAtDepth(25), 3)     // primordial_geode tier
	assert.Len(t, lootAtDepth(35), 2)     // resonance_core tier
}

func TestRiskDecreaseWithLevel(t *testing.T) {
	svc, s := testService(t)
	s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 1, XP: 0})
	s.DB.Create(&model.Job{UserID: 2, JobName: "miner", Level: 20, XP: 0})

	// Verify miner levels are correctly stored
	var j1, j2 model.Job
	s.DB.Where("user_id = ? AND job_name = ?", 1, "miner").First(&j1)
	s.DB.Where("user_id = ? AND job_name = ?", 2, "miner").First(&j2)
	t.Logf("User 1 miner level: %d, User 2 miner level: %d", j1.Level, j2.Level)

	l1, _ := svc.GetMinerLevel(1)
	l2, _ := svc.GetMinerLevel(2)
	t.Logf("GetMinerLevel(1) = %d, GetMinerLevel(2) = %d", l1, l2)
	assert.Equal(t, 1, l1)
	assert.Equal(t, 20, l2)

	// At very high depth, level 1 always collapses but level 20 has slight advantage
	// At depth 15 careful:
	// Level 1: (14*5) - 10 - 1 = 59% risk
	// Level 20: (14*5) - 10 - 30 = 30% risk
	collapseL1 := 0
	collapseL20 := 0
	trials := 40

	for i := 0; i < trials; i++ {
		r1, err := svc.Descend(1, 15, nil, BranchCareful, "", 0)  // Level 1
		require.NoError(t, err)
		r2, err := svc.Descend(2, 15, nil, BranchCareful, "", 0)  // Level 20
		require.NoError(t, err)
		if r1.Collapsed {
			collapseL1++
		}
		if r2.Collapsed {
			collapseL20++
		}
	}

	t.Logf("Level 1 collapses: %d/%d (%.0f%%), Level 20 collapses: %d/%d (%.0f%%)",
		collapseL1, trials, float64(collapseL1)/float64(trials)*100,
		collapseL20, trials, float64(collapseL20)/float64(trials)*100)
	assert.Greater(t, collapseL1, collapseL20,
		"lower level miner should collapse more often at same depth")
}
