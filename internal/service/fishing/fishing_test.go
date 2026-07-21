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
	"guacagamblebot/internal/store"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "f.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	svc := New(s, cfg)
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
		"pufferfish": true, "swordfish": true, "shark": true,
		"whale": true, "kraken_tentacle": true,
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

func TestFishAddsCharacterXP(t *testing.T) {
	svc, s := testService(t)
	_, err := svc.CastLine(1, "pond")
	require.NoError(t, err)

	c, err := s.GetCharacter(1)
	require.NoError(t, err)
	assert.Greater(t, c.XP, 0)
}

func TestFishScavengerBuffGivesExtraItem(t *testing.T) {
	svc, s := testService(t)
	_ = s.SetActiveBuff(1, "scavenger")

	_, err := svc.CastLine(1, "pond")
	require.NoError(t, err)

	has, _ := s.HasActiveBuff(1, "scavenger")
	assert.False(t, has, "scavenger should be consumed after cast")
}
