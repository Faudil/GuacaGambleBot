package archeology

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
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "a.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	s := store.New(d, cfg)
	svc := New(s, cfg)
	return svc, s
}

func TestNewGameSafe(t *testing.T) {
	svc, _ := testService(t)
	state, err := svc.NewGame(1, "safe")
	require.NoError(t, err)
	assert.Equal(t, "safe", state.PermitType)
	assert.Equal(t, 50, state.Depth)
	assert.Equal(t, 100, state.Integrity)
	assert.Equal(t, 5, state.Actions)
}

func TestNewGameFailleNoMoney(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.NewGame(1, "faille")
	assert.ErrorIs(t, err, ErrNoMoney)
}

func TestNewGameFaille(t *testing.T) {
	svc, s := testService(t)
	_, err := s.UpdateBalance(1, 500)
	require.NoError(t, err)
	state, err := svc.NewGame(1, "faille")
	require.NoError(t, err)
	assert.Equal(t, "faille", state.PermitType)
}

func TestApplyAction(t *testing.T) {
	svc, _ := testService(t)
	state, _ := svc.NewGame(1, "safe")
	outcome := svc.ApplyAction(state, ActionBrush)
	assert.Equal(t, 48, outcome.State.Depth)
	assert.Equal(t, 100, outcome.State.Integrity)
	assert.Equal(t, 4, outcome.State.Actions)
	assert.False(t, outcome.Damaged)
}

func TestResolveDisaster(t *testing.T) {
	state := &GameState{Integrity: 0, Depth: 30, Actions: 2}
	res := (&Service{}).Resolve(state)
	assert.Equal(t, "bone_dust", res.ItemName)
}

func TestResolveDamaged(t *testing.T) {
	state := &GameState{Integrity: 30, Depth: 0, Actions: 3}
	res := (&Service{}).Resolve(state)
	assert.Equal(t, "damaged_fossil", res.ItemName)
}

func TestResolvePerfect(t *testing.T) {
	state := &GameState{Integrity: 100, Depth: 0, Actions: 3}
	res := (&Service{}).Resolve(state)
	assert.Equal(t, "pure_dna", res.ItemName)
}

func TestReanimateInvalidRarity(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.Reanimate(1, "invalid")
	assert.Error(t, err)
}

func TestReanimateNoParts(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.Reanimate(1, "commun")
	assert.Error(t, err)
}
