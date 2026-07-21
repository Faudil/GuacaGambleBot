package pets

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
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "p.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestCreatePet(t *testing.T) {
	svc, _ := testService(t)
	pet, err := svc.CreatePet(1, "Dragon")
	require.NoError(t, err)
	require.NotNil(t, pet)
	assert.Equal(t, "Dragon", pet.PetType)
	assert.Equal(t, 130, pet.MaxHP)
	assert.Equal(t, 35, pet.Atk)
}

func TestGetPets(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.CreatePet(1, "Escargot")
	require.NoError(t, err)
	_, err = svc.CreatePet(1, "Souris")
	require.NoError(t, err)
	pets, err := svc.GetPets(1)
	require.NoError(t, err)
	assert.Len(t, pets, 2)
}

func TestAddXPLevelUp(t *testing.T) {
	svc, _ := testService(t)
	pet, err := svc.CreatePet(1, "Escargot")
	require.NoError(t, err)
	res := svc.AddXP(pet, 1000)
	assert.True(t, res.Leveled)
	assert.Greater(t, pet.Level, 1)
}

func TestUpdateElo(t *testing.T) {
	svc, _ := testService(t)
	p1, _ := svc.CreatePet(1, "Dragon")
	p2, _ := svc.CreatePet(2, "Souris")
	p1.Level = 10
	p2.Level = 10
	p1.Elo = 1000
	p2.Elo = 1000
	d1, d2 := svc.UpdateElo(p1, p2, 1.0)
	assert.NotEqual(t, 0, d1)
	assert.NotEqual(t, 0, d2)
	assert.Greater(t, p1.Elo, 1000)
	assert.Less(t, p2.Elo, 1000)
}

func TestRollGacha(t *testing.T) {
	name := RollGacha("")
	assert.NotEmpty(t, name)
	_, ok := PetTypes[name]
	assert.True(t, ok)
}

func TestRollGachaLegendary(t *testing.T) {
	name := RollGacha(RarityLegendary)
	pt, ok := PetTypes[name]
	require.True(t, ok)
	assert.Equal(t, RarityLegendary, pt.Rarity)
}
