package boss

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
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "b.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestGetStage(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.User{UserID: 1}).Error)
	stage, err := svc.GetStage(1)
	require.NoError(t, err)
	assert.Equal(t, 0, stage)
}

func TestSetStage(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.User{UserID: 1}).Error)
	require.NoError(t, svc.SetStage(1, 2))
	stage, err := svc.GetStage(1)
	require.NoError(t, err)
	assert.Equal(t, 2, stage)
}

func TestBossLeagueLength(t *testing.T) {
	assert.Len(t, BossLeague, 5)
}

func TestCreateBossPet(t *testing.T) {
	svc, _ := testService(t)
	boss := BossLeague[0]
	bp := svc.CreateBossPet(boss)
	assert.NotNil(t, bp)
	assert.Equal(t, boss.HP, bp.MaxHP)
	assert.Equal(t, boss.Atk, bp.Atk)
}
