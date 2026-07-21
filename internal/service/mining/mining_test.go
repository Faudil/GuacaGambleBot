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
	items := []string{"Caillou", "Charbon", "Minerai de Fer", "Minerai de Cuivre",
		"Minerai d'argent", "Pépite d'Or", "Platine", "Emeraude", "Diamant Brut"}
	for _, name := range items {
		_ = s.DB.Create(&model.Item{Name: name, Price: 1})
	}
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
	bag := []BagEntry{{Name: "Caillou", Count: 3}, {Name: "Charbon", Count: 1}}
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
