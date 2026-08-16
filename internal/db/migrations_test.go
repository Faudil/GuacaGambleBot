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
	require.NoError(t, d.AutoMigrate(&model.UserQuest{}, &model.UserQuestData{}, &model.UserStat{}, &model.DataMigration{}))
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

func TestMigrateTutorialRewindSkippedHunt(t *testing.T) {
	d := migrationTestDB(t)

	// Skipped players: past the hunt block (step >= 11), no hunt wins.
	require.NoError(t, d.Create(&model.UserQuest{UserID: 1, QuestID: "tutorial", Status: "ACTIVE"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 1, QuestID: "tutorial", StepIndex: 12, ProgressValue: 1, CustomData: `{"target_stat":"items_digged"}`}).Error)
	require.NoError(t, d.Create(&model.UserQuest{UserID: 2, QuestID: "tutorial", Status: "ACTIVE"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 2, QuestID: "tutorial", StepIndex: 16}).Error)

	// Legit player: past the hunt block with hunt wins — must not move.
	require.NoError(t, d.Create(&model.UserQuest{UserID: 3, QuestID: "tutorial", Status: "ACTIVE"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 3, QuestID: "tutorial", StepIndex: 12}).Error)
	require.NoError(t, d.Create(&model.UserStat{UserID: 3, PveWins: 2}).Error)

	// Pre-hunt players and early steps — must not move.
	require.NoError(t, d.Create(&model.UserQuest{UserID: 4, QuestID: "tutorial", Status: "ACTIVE"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 4, QuestID: "tutorial", StepIndex: 6}).Error)
	require.NoError(t, d.Create(&model.UserQuest{UserID: 5, QuestID: "tutorial", Status: "ACTIVE"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 5, QuestID: "tutorial", StepIndex: 3}).Error)

	// Completed quest and unrelated quest — must not move.
	require.NoError(t, d.Create(&model.UserQuest{UserID: 100, QuestID: "tutorial", Status: "COMPLETED"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 100, QuestID: "tutorial", StepIndex: 12}).Error)
	require.NoError(t, d.Create(&model.UserQuest{UserID: 200, QuestID: "boss_league", Status: "ACTIVE"}).Error)
	require.NoError(t, d.Create(&model.UserQuestData{UserID: 200, QuestID: "boss_league", StepIndex: 12}).Error)

	require.NoError(t, migrateTutorialRewindSkippedHunt(d))

	for _, tc := range []struct {
		uid  int64
		want int
	}{
		{1, 7}, {2, 7}, {3, 12}, {4, 6}, {5, 3}, {100, 12}, {200, 12},
	} {
		qid := "tutorial"
		if tc.uid == 200 {
			qid = "boss_league"
		}
		var uqd model.UserQuestData
		require.NoError(t, d.Where("user_id = ? AND quest_id = ?", tc.uid, qid).First(&uqd).Error)
		assert.Equal(t, tc.want, uqd.StepIndex, "user %d", tc.uid)
	}

	// Rewound rows have progress and custom data reset.
	var rewound model.UserQuestData
	require.NoError(t, d.Where("user_id = 1 AND quest_id = 'tutorial'").First(&rewound).Error)
	assert.Equal(t, 0, rewound.ProgressValue)
	assert.Equal(t, "{}", rewound.CustomData)
}

func TestRunDataMigrationsIdempotent(t *testing.T) {
	d := migrationTestDB(t)

	// First run applies the tutorial migrations and records the markers.
	require.NoError(t, runDataMigrations(d))
	for _, id := range []string{"tutorial_step_reorder", "tutorial_rewind_skipped_hunt"} {
		var count int64
		require.NoError(t, d.Model(&model.DataMigration{}).Where("id = ?", id).Count(&count).Error)
		assert.Equal(t, int64(1), count, "marker %s", id)
	}

	// Second run must be a no-op (markers present, nothing re-applied).
	require.NoError(t, runDataMigrations(d))
	for _, id := range []string{"tutorial_step_reorder", "tutorial_rewind_skipped_hunt"} {
		var count int64
		require.NoError(t, d.Model(&model.DataMigration{}).Where("id = ?", id).Count(&count).Error)
		assert.Equal(t, int64(1), count, "marker %s", id)
	}
}
