package hunt

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
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "h.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	s := store.New(d, cfg)
	svc := New(s, cfg)
	return svc, s
}

func addActivePet(t *testing.T, s *store.Store, userID int64) {
	t.Helper()
	_ = s.DB.Create(&model.UserPet{
		UserID:   userID,
		PetType:  "Chien",
		Nickname: "Buddy",
		Level:    5,
		XP:       0,
		MaxHP:    100,
		HP:       100,
		Atk:      15,
		Defense:  8,
		Speed:    10,
		IsActive: true,
	})
}

func TestExecuteHuntNoPet(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.ExecuteHunt(1, "easy")
	assert.ErrorIs(t, err, ErrNoPet)
}

func TestExecuteHuntEasy(t *testing.T) {
	svc, s := testService(t)
	addActivePet(t, s, 1)
	res, err := svc.ExecuteHunt(1, "easy")
	require.NoError(t, err)
	assert.True(t, res.PlayerWon || res.EnemyWon)
	assert.NotEmpty(t, res.Log)
	assert.Greater(t, res.XP, 0)
}

func TestExecuteHuntInvalidZone(t *testing.T) {
	svc, s := testService(t)
	addActivePet(t, s, 1)
	_, err := svc.ExecuteHunt(1, "nonexistent")
	assert.Error(t, err)
}

func TestNewEnemy(t *testing.T) {
	e := NewEnemy("easy")
	assert.NotEmpty(t, e.Name)
	assert.Greater(t, e.HP, 0)
	assert.Greater(t, e.Level, 0)
}
