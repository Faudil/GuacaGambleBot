package db

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/model"
)

func migrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "migrate.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, d.AutoMigrate(&model.UserQuest{}, &model.UserQuestData{}, &model.DataMigration{}))
	return d
}

func TestMigrateTutorialSteps(t *testing.T) {
	d := migrationTestDB(t)

	// ACTIVE tutorial players at various old step indices.
	oldSteps := map[int64]int{1: 0, 2: 7, 3: 12, 4: 13, 5: 19, 6: 30}
	for uid, step := range oldSteps {
		require.NoError(t, d.Create(&model.UserQuest{UserID: uid, QuestID: "tutorial", Status: "ACTIVE"}).Error)
		require.NoError(t, d.Create(&model.UserQuestData{UserID: uid, QuestID: "tutorial", StepIndex: step}).Error)
	}
	// A completed quest must not be remapped.
	require.NoError(t, d.Create(&model.UserQuest{UserID: 100, QuestID: "tutorial", Status: "COMPLETED"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 100, QuestID: "tutorial", StepIndex: 7}).Error)
	// An unrelated quest must not be touched.
	require.NoError(t, d.Create(&model.UserQuest{UserID: 200, QuestID: "boss_league", Status: "ACTIVE"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 200, QuestID: "boss_league", StepIndex: 7}).Error)

	require.NoError(t, migrateTutorialSteps(d))

	expected := map[int64]int{1: 0, 2: 12, 3: 7, 4: 8, 5: 14, 6: 31}
	for uid, want := range expected {
		var uqd model.UserQuestData
		require.NoError(t, d.Where("user_id = ? AND quest_id = 'tutorial'", uid).First(&uqd).Error)
		assert.Equal(t, want, uqd.StepIndex, "user %d", uid)
	}
	var completed model.UserQuestData
	require.NoError(t, d.Where("user_id = 100 AND quest_id = 'tutorial'").First(&completed).Error)
	assert.Equal(t, 7, completed.StepIndex, "completed quest must not be remapped")

	var other model.UserQuestData
	require.NoError(t, d.Where("user_id = 200 AND quest_id = 'boss_league'").First(&other).Error)
	assert.Equal(t, 7, other.StepIndex, "other quests must not be remapped")
}

func TestRunDataMigrationsIdempotent(t *testing.T) {
	d := migrationTestDB(t)

	// First run applies the tutorial migration and records the marker.
	require.NoError(t, runDataMigrations(d))
	var count int64
	require.NoError(t, d.Model(&model.DataMigration{}).Where("id = ?", "tutorial_step_reorder").Count(&count).Error)
	assert.Equal(t, int64(1), count)

	// Second run must be a no-op (marker present, nothing re-applied).
	require.NoError(t, runDataMigrations(d))
	require.NoError(t, d.Model(&model.DataMigration{}).Where("id = ?", "tutorial_step_reorder").Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
