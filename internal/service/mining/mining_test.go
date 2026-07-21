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
	res, err := svc.Descend(1, 1, nil, 0)
	require.NoError(t, err)
	if !res.Collapsed {
		assert.NotNil(t, res.Item)
		assert.NotEmpty(t, res.Bag)
	}
}

func TestDescendCollapse(t *testing.T) {
	svc, _ := testService(t)
	bag := []BagEntry{}
	collapsed := false
	for i := 0; i < 100; i++ {
		res, err := svc.Descend(1, 30, bag, 0)
		require.NoError(t, err)
		if res.Collapsed {
			collapsed = true
			break
		}
		bag = res.Bag
	}
	assert.True(t, collapsed, "should have collapsed at high risk")
}

func TestLeaveMine(t *testing.T) {
	svc, _ := testService(t)
	bag := []BagEntry{{Name: "pebble", Count: 3}, {Name: "coal", Count: 1}}
	res, err := svc.LeaveMine(1, bag)
	require.NoError(t, err)
	assert.Equal(t, bag, res.Bag)
	assert.Greater(t, res.XP, 0)
}

func TestLeaveMineEmpty(t *testing.T) {
	svc, _ := testService(t)
	res, err := svc.LeaveMine(1, nil)
	require.NoError(t, err)
	assert.Empty(t, res.Bag)
	assert.Equal(t, 0, res.XP)
}

func TestLeaveMineAddsCharacterXP(t *testing.T) {
	svc, s := testService(t)
	bag := []BagEntry{{Name: "pebble", Count: 3}}
	_, err := svc.LeaveMine(1, bag)
	require.NoError(t, err)

	c, err := s.GetCharacter(1)
	require.NoError(t, err)
	assert.Greater(t, c.XP, 0)
}

func TestMineReinforceBuffPreventsCollapse(t *testing.T) {
	svc, s := testService(t)
	_ = s.SetActiveBuff(1, "reinforce")

	// Descend at a depth where collapse would normally happen
	collapsed := false
	res, err := svc.Descend(1, 50, nil, 0)
	require.NoError(t, err)
	if res.Collapsed {
		collapsed = true
	}
	// reinforce should prevent collapse, even at very high depth
	assert.False(t, collapsed, "reinforce should prevent collapse")

	has, _ := s.HasActiveBuff(1, "reinforce")
	assert.False(t, has, "reinforce should be consumed after one descend")
}

func TestMineScavengerBuffDoublesItems(t *testing.T) {
	svc, s := testService(t)
	_ = s.SetActiveBuff(1, "scavenger")

	bag := []BagEntry{{Name: "pebble", Count: 2}, {Name: "coal", Count: 1}}
	res, err := svc.LeaveMine(1, bag)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(res.Bag), 2)
	// scavenger adds 50% more to each entry
	for _, e := range res.Bag {
		if e.Name == "pebble" {
			assert.Greater(t, e.Count, 2)
		}
	}

	has, _ := s.HasActiveBuff(1, "scavenger")
	assert.False(t, has, "scavenger should be consumed after leave")
}
