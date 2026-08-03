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
	_, _, err := svc.Craft(1, "nonexistent_recipe", 1)
	assert.ErrorIs(t, err, ErrNoRecipe)
}

func TestCraftLevelTooLow(t *testing.T) {
	svc, _ := testService(t)
	_, _, err := svc.Craft(1, "volcano_egg", 1)
	assert.ErrorIs(t, err, ErrNoLevel)
}

func TestCraftMissingIngredients(t *testing.T) {
	svc, _ := testService(t)
	_, _, err := svc.Craft(1, "beer", 1)
	assert.ErrorIs(t, err, ErrNoIngredients)
}

func craftSetup(t *testing.T, st *store.Store, userID int64) string {
	require.NoError(t, st.DB.Create(&model.Job{UserID: userID, JobName: "crafter", Level: 2, XP: 0}).Error)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: userID, ItemID: "wheat", Quantity: 10}).Error)
	return "wheat"
}

func TestCraftSuccess(t *testing.T) {
	svc, st := testService(t)
	wheatID := craftSetup(t, st, 1)

	_, _, err := svc.Craft(1, "beer", 1)
	require.NoError(t, err)

	var inv model.Inventory
	require.NoError(t, st.DB.Where("user_id = ? AND item_id = ?", 1, "beer").First(&inv).Error)
	assert.Equal(t, 1, inv.Quantity)

	var invBle model.Inventory
	require.NoError(t, st.DB.Where("user_id = ? AND item_id = ?", 1, wheatID).First(&invBle).Error)
	assert.Equal(t, 7, invBle.Quantity)
}

func TestCraftAddsXP(t *testing.T) {
	svc, st := testService(t)
	craftSetup(t, st, 2)

	_, _, _ = svc.Craft(2, "beer", 1)

	var job model.Job
	st.DB.Where("user_id = ? AND job_name = ?", 2, "crafter").First(&job)
	assert.Equal(t, 10, job.XP)
}

func TestCraftMultiple(t *testing.T) {
	svc, st := testService(t)
	wheatID := craftSetup(t, st, 3)

	_, _, _ = svc.Craft(3, "beer", 3)

	var inv model.Inventory
	st.DB.Where("user_id = ? AND item_id = ?", 3, "beer").First(&inv)
	assert.Equal(t, 3, inv.Quantity)

	var invBle model.Inventory
	st.DB.Where("user_id = ? AND item_id = ?", 3, wheatID).First(&invBle)
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

func TestCraftAddsCharacterXP(t *testing.T) {
	svc, st := testService(t)
	craftSetup(t, st, 7)
	_, _, _ = svc.Craft(7, "beer", 1)

	c, err := st.GetCharacter(7)
	require.NoError(t, err)
	assert.Greater(t, c.XP, 0)
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
