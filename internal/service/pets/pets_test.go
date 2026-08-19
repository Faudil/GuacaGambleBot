package pets

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/store"
)

func TestAllPetTypesHaveDamageType(t *testing.T) {
	for name, pt := range PetTypes {
		_, ok := battle.PetDamageType(name)
		assert.True(t, ok, "pet %q (%s) has no battle damage type", name, pt.Rarity)
	}
}

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

func TestGetFeedItemDef(t *testing.T) {
	for _, id := range []string{"oat", "coffee", "coffee_bean", "tomato", "pumpkin", "golden_apple", "nova_fruit"} {
		assert.NotNil(t, GetFeedItemDef(id), "expected %s to be feedable", id)
	}
	for _, id := range []string{"pebble", "coal", "iron_ore", "wheat_seed"} {
		assert.Nil(t, GetFeedItemDef(id), "expected %s to NOT be feedable", id)
	}
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
	name := RollGacha("", "forest")
	assert.NotEmpty(t, name)
	_, ok := PetTypes[name]
	assert.True(t, ok)
}

func TestRollGachaLegendary(t *testing.T) {
	name := RollGacha(RarityLegendary, "forest")
	pt, ok := PetTypes[name]
	require.True(t, ok)
	assert.Equal(t, RarityLegendary, pt.Rarity)
}

func TestCreatePetAutoActivatesFirst(t *testing.T) {
	svc, _ := testService(t)
	first, err := svc.CreatePet(1, "Dragon")
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.True(t, first.IsActive, "first pet should be auto-activated")

	second, err := svc.CreatePet(1, "Escargot")
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.False(t, second.IsActive, "second pet must not steal activation")

	active, err := svc.GetActivePet(1)
	require.NoError(t, err)
	assert.Equal(t, first.ID, active.ID)
}

func TestSetActivePetExclusive(t *testing.T) {
	svc, _ := testService(t)
	a, err := svc.CreatePet(1, "Dragon")
	require.NoError(t, err)
	b, err := svc.CreatePet(1, "Escargot")
	require.NoError(t, err)
	require.True(t, a.IsActive)
	require.False(t, b.IsActive)

	require.NoError(t, svc.SetActivePet(1, b.ID, 0))

	active, err := svc.GetActivePet(1)
	require.NoError(t, err)
	assert.Equal(t, b.ID, active.ID, "switching activation must be exclusive")

	pets, err := svc.GetPets(1)
	require.NoError(t, err)
	for _, p := range pets {
		if p.ID == b.ID {
			assert.True(t, p.IsActive)
		} else {
			assert.False(t, p.IsActive)
		}
	}
}

func TestHealCost(t *testing.T) {
	assert.Equal(t, 1, HealCost(1, 0))
	assert.Equal(t, 1, HealCost(2, 0))
	assert.Equal(t, 50, HealCost(100, 0))
	assert.Equal(t, 25, HealCost(100, 50))
	assert.Equal(t, 45, HealCost(100, 10))
	assert.Equal(t, 0, HealCost(100, 100), "100% discount must make the heal free")
	assert.Equal(t, 1, HealCost(100, 99))
}

func TestHealPetGuards(t *testing.T) {
	svc, s := testService(t)
	pet, err := svc.CreatePet(1, "Dragon")
	require.NoError(t, err)

	err = svc.HealPet(pet, 10)
	assert.ErrorIs(t, err, ErrPetAlreadyFullHP, "full HP must be rejected")

	pet.HP = 10
	_, err = s.UpdateBalance(1, 100)
	require.NoError(t, err)
	require.NoError(t, svc.UpdatePet(pet))

	require.NoError(t, svc.HealPet(pet, 10))
	assert.Equal(t, pet.MaxHP, pet.HP)

	bal, err := s.GetBalance(1)
	require.NoError(t, err)
	assert.Equal(t, 190, bal, "heal must deduct exactly the paid cost")

	pet.HP = 5
	require.NoError(t, svc.UpdatePet(pet))
	err = svc.HealPet(pet, 100000)
	assert.ErrorIs(t, err, ErrInsufficientFunds)
	assert.Equal(t, 5, pet.HP, "HP must not change on failure")

	pet.HP = 5
	require.NoError(t, svc.UpdatePet(pet))
	require.NoError(t, svc.HealPet(pet, 0), "free heal (100% discount) must work")
	assert.Equal(t, pet.MaxHP, pet.HP)
}
