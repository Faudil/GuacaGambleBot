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

func TestGenerateExpedition(t *testing.T) {
	svc, _ := testService(t)
	res := svc.Generate("Dragon", 10, 2, "fr")
	assert.NotNil(t, res)
	assert.Greater(t, res.XP, 0)
	assert.NotEmpty(t, res.Log)
}

func TestStartAndGetActive(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.User{UserID: 1}).Error)
	require.NoError(t, st.DB.Create(&model.UserPet{ID: 1, UserID: 1, PetType: "Dragon", Nickname: "Draco"}).Error)

	res := svc.Generate("Dragon", 10, 2, "fr")
	exp, err := svc.Start(1, 1, 2, res)
	require.NoError(t, err)
	require.NotNil(t, exp)
	assert.Equal(t, int64(1), exp.UserID)
	assert.Equal(t, int64(1), exp.PetID)
	assert.False(t, exp.IsClaimed)

	active, err := svc.GetActive(1)
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, exp.ID, active.ID)
}

func TestClaim(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.User{UserID: 1}).Error)
	require.NoError(t, st.DB.Create(&model.UserPet{ID: 1, UserID: 1, PetType: "Dragon", Nickname: "Draco"}).Error)

	res := svc.Generate("Dragon", 10, 1, "fr")
	exp, err := svc.Start(1, 1, 1, res)
	require.NoError(t, err)

	_, _, err = svc.Claim(exp)
	require.NoError(t, err)

	_, err = svc.GetActive(1)
	assert.Error(t, err)
}
