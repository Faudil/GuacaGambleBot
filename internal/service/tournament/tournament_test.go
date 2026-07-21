package tournament

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
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "t.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestSimulateMatch(t *testing.T) {
	svc, _ := testService(t)
	p1 := &TournamentPlayer{
		UserID: 1,
		Pet: &model.UserPet{
			ID: 1, PetType: "Dragon", Nickname: "Draco",
			Level: 10, MaxHP: 130, HP: 130, Atk: 35, Defense: 20, Speed: 20,
			DGE: 15, ACC: 25, CritC: 10, CritD: 1.2, SpcC: 5,
		},
	}
	p2 := &TournamentPlayer{
		UserID: 2,
		Pet: &model.UserPet{
			ID: 2, PetType: "Souris", Nickname: "Mini",
			Level: 10, MaxHP: 25, HP: 25, Atk: 12, Defense: 3, Speed: 25,
			DGE: 25, ACC: 5, CritC: 10, CritD: 1.5, SpcC: 0,
		},
	}

	result := svc.SimulateMatch(p1, p2)
	assert.NotNil(t, result)
	// Dragon should beat Mouse
	assert.Equal(t, int64(1), result.WinnerID)
}

func TestShufflePlayers(t *testing.T) {
	players := []TournamentPlayer{
		{UserID: 1}, {UserID: 2}, {UserID: 3}, {UserID: 4},
	}
	original := make([]TournamentPlayer, len(players))
	copy(original, players)
	ShufflePlayers(players)
	// Shuffle should not change length
	assert.Equal(t, len(original), len(players))
}
