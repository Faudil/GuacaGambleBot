package crafting

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
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "craft.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestGetCrafterLevelDefault(t *testing.T) {
	svc, _ := testService(t)
	assert.Equal(t, 1, svc.GetCrafterLevel(1))
}

func TestGetCrafterLevelExisting(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.Job{UserID: 1, JobName: "crafter", Level: 5, XP: 100}).Error)
	assert.Equal(t, 5, svc.GetCrafterLevel(1))
}

func TestCraftNoRecipe(t *testing.T) {
	svc, _ := testService(t)
	err := svc.Craft(1, "nonexistent_recipe", 1)
	assert.ErrorIs(t, err, ErrNoRecipe)
}

func TestCraftLevelTooLow(t *testing.T) {
	svc, _ := testService(t)
	err := svc.Craft(1, "œuf mystère", 1)
	assert.ErrorIs(t, err, ErrNoLevel)
}

func TestCraftMissingIngredients(t *testing.T) {
	svc, _ := testService(t)
	err := svc.Craft(1, "bière", 1)
	assert.ErrorIs(t, err, ErrNoIngredients)
}

func craftSetup(t *testing.T, st *store.Store, userID int64) model.Item {
	require.NoError(t, st.DB.Create(&model.Job{UserID: userID, JobName: "crafter", Level: 2, XP: 0}).Error)
	ble := model.Item{Name: "blé", Price: 2, Description: "", EffectType: "resource"}
	require.NoError(t, st.DB.Create(&ble).Error)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: userID, ItemID: ble.ID, Quantity: 10}).Error)
	return ble
}

func itemID(t *testing.T, st *store.Store, name string) int64 {
	var it model.Item
	require.NoError(t, st.DB.Where("name = ?", name).First(&it).Error)
	return it.ID
}

func TestCraftSuccess(t *testing.T) {
	svc, st := testService(t)
	ble := craftSetup(t, st, 1)

	err := svc.Craft(1, "bière", 1)
	require.NoError(t, err)

	biereID := itemID(t, st, "bière")
	var inv model.Inventory
	require.NoError(t, st.DB.Where("user_id = ? AND item_id = ?", 1, biereID).First(&inv).Error)
	assert.Equal(t, 1, inv.Quantity)

	var invBle model.Inventory
	require.NoError(t, st.DB.Where("user_id = ? AND item_id = ?", 1, ble.ID).First(&invBle).Error)
	assert.Equal(t, 7, invBle.Quantity)
}

func TestCraftAddsXP(t *testing.T) {
	svc, st := testService(t)
	craftSetup(t, st, 2)

	require.NoError(t, svc.Craft(2, "bière", 1))

	var job model.Job
	st.DB.Where("user_id = ? AND job_name = ?", 2, "crafter").First(&job)
	assert.Equal(t, 10, job.XP)
}

func TestCraftMultiple(t *testing.T) {
	svc, st := testService(t)
	ble := craftSetup(t, st, 3)

	require.NoError(t, svc.Craft(3, "bière", 3))

	biereID := itemID(t, st, "bière")
	var inv model.Inventory
	st.DB.Where("user_id = ? AND item_id = ?", 3, biereID).First(&inv)
	assert.Equal(t, 3, inv.Quantity)

	var invBle model.Inventory
	st.DB.Where("user_id = ? AND item_id = ?", 3, ble.ID).First(&invBle)
	assert.Equal(t, 1, invBle.Quantity)
}

func TestLevelUpCheckNoJob(t *testing.T) {
	svc, _ := testService(t)
	didLevel, level := svc.LevelUpCheck(1)
	assert.False(t, didLevel)
	assert.Equal(t, 1, level)
}

func TestLevelUpCheckNotEnoughXP(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.Job{UserID: 1, JobName: "crafter", Level: 3, XP: 50}).Error)

	didLevel, level := svc.LevelUpCheck(1)
	assert.False(t, didLevel)
	assert.Equal(t, 3, level)
}

func TestLevelUpCheckSuccess(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.Job{UserID: 1, JobName: "crafter", Level: 3, XP: 300}).Error)

	didLevel, level := svc.LevelUpCheck(1)
	assert.True(t, didLevel)
	assert.Equal(t, 4, level)

	var job model.Job
	st.DB.Where("user_id = ? AND job_name = ?", 1, "crafter").First(&job)
	assert.Equal(t, 4, job.Level)
	assert.Equal(t, 0, job.XP) // 300 - 300 = 0
}
