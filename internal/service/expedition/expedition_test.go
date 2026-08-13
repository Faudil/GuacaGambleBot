package expedition

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
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "e.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func testPet() *model.UserPet {
	return &model.UserPet{
		ID: 1, UserID: 1, PetType: "Dragon", Nickname: "Draco",
		Level: 10, MaxHP: 200, HP: 200, Atk: 40, Defense: 30, Speed: 30,
		DGE: 20, ACC: 30, CritC: 15, CritD: 2.0, SpcC: 10,
	}
}

func TestGenerateExpedition(t *testing.T) {
	svc, _ := testService(t)
	pet := testPet()
	res := svc.Generate(pet, 2)
	assert.NotNil(t, res)
	assert.Greater(t, res.XP, 0)
	assert.NotEmpty(t, res.Log)
	assert.LessOrEqual(t, res.PetHP, pet.HP)
	assert.GreaterOrEqual(t, res.PetHP, 0)
}

func TestStartAndGetActive(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.User{UserID: 1}).Error)
	require.NoError(t, st.DB.Create(testPet()).Error)

	res := svc.Generate(testPet(), 2)
	exp, err := svc.Start(1, 1, 2, res)
	require.NoError(t, err)
	require.NotNil(t, exp)
	assert.Equal(t, int64(1), exp.UserID)
	assert.Equal(t, int64(1), exp.PetID)
	assert.False(t, exp.IsClaimed)

	// HP loss from combat events must be persisted on the pet.
	var pet model.UserPet
	require.NoError(t, st.DB.First(&pet, 1).Error)
	assert.Equal(t, res.PetHP, pet.HP)
	assert.True(t, pet.OnExpedition)

	active, err := svc.GetActive(1)
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, exp.ID, active.ID)
}

func TestClaim(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.User{UserID: 1}).Error)
	require.NoError(t, st.DB.Create(testPet()).Error)

	res := svc.Generate(testPet(), 1)
	exp, err := svc.Start(1, 1, 1, res)
	require.NoError(t, err)

	_, _, err = svc.Claim(exp)
	require.NoError(t, err)

	_, err = svc.GetActive(1)
	assert.Error(t, err)
}
