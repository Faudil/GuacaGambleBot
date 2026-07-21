package fishing

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
	"guacagamblebot/internal/store"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "f.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	svc := New(s, cfg)
	_ = s.DB.Create(&model.Item{Name: "Vieille botte", Price: 1, Description: "An old boot"})
	_ = s.DB.Create(&model.Item{Name: "Truite", Price: 10, Description: "Trout"})
	_ = s.DB.Create(&model.Item{Name: "Saumon", Price: 10, Description: "Salmon"})
	_ = s.DB.Create(&model.Item{Name: "Sardine", Price: 15, Description: "Sardine"})
	_ = s.DB.Create(&model.Item{Name: "Carpe", Price: 25, Description: "Carp"})
	_ = s.DB.Create(&model.Item{Name: "Poisson-Globe", Price: 50, Description: "Pufferfish"})
	_ = s.DB.Create(&model.Item{Name: "Espadon", Price: 150, Description: "Swordfish"})
	_ = s.DB.Create(&model.Item{Name: "Requin", Price: 100, Description: "Shark"})
	_ = s.DB.Create(&model.Item{Name: "Baleine", Price: 300, Description: "Whale"})
	_ = s.DB.Create(&model.Item{Name: "Tentacule de Kraken", Price: 500, Description: "Kraken"})
	return svc, s
}

func TestCastLine(t *testing.T) {
	svc, _ := testService(t)
	res, err := svc.CastLine(1, "pond")
	require.NoError(t, err)
	assert.NotEmpty(t, res.ItemName)
	assert.Greater(t, res.XP, 0)
}

func TestCastLineLimit(t *testing.T) {
	svc, _ := testService(t)
	for i := 0; i < 10; i++ {
		_, err := svc.CastLine(1, "pond")
		require.NoError(t, err)
	}
	_, err := svc.CastLine(1, "pond")
	assert.ErrorIs(t, err, ErrLimit)
}

func TestBiomeSpecificLoot(t *testing.T) {
	svc, _ := testService(t)
	oceanItems := map[string]bool{
		"Poisson-Globe": true, "Espadon": true, "Requin": true,
		"Baleine": true, "Tentacule de Kraken": true,
	}
	for i := 0; i < 5; i++ {
		res, err := svc.CastLine(99, "ocean")
		require.NoError(t, err)
		assert.True(t, oceanItems[res.ItemName], "unexpected ocean item: "+res.ItemName)
	}
}

func TestCheckCooldown(t *testing.T) {
	svc, _ := testService(t)
	d, err := svc.CheckCooldown(1)
	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), d)
}
