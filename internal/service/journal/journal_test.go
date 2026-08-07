package journal

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

func testStore(t *testing.T) *store.Store {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	return store.New(d, &config.Config{StartingBalance: 100, DailyAmount: 50})
}

// TestStepCompletesAndRewards verifies that reaching a milestone's stat target
// advances the path and grants the reward exactly once.
func TestStepCompletesAndRewards(t *testing.T) {
	s := testStore(t)
	New(s)

	require.NoError(t, s.DB.Create(&model.UserStat{UserID: 1, ItemsMined: 25}).Error)
	require.NoError(t, s.RecordActivity(1, "items_mined", 1))

	var entry model.UserJournalEntry
	require.NoError(t, s.DB.Where("user_id = ? AND path_id = ?", 1, "prospector").First(&entry).Error)
	assert.Equal(t, 1, entry.StepIndex, "first prospector step should be done")

	bal, _ := s.GetBalance(1)
	assert.Equal(t, 150, bal, "reward money granted once")
}

// TestCascadeCompletesSeveralSteps verifies several consecutive steps complete
// in a single check when their milestones are all already met.
func TestCascadeCompletesSeveralSteps(t *testing.T) {
	s := testStore(t)
	New(s)

	require.NoError(t, s.DB.Create(&model.UserStat{UserID: 1, ItemsMined: 40, ItemsFished: 40}).Error)
	require.NoError(t, s.RecordActivity(1, "items_mined", 1))

	var entry model.UserJournalEntry
	require.NoError(t, s.DB.Where("user_id = ? AND path_id = ?", 1, "prospector").First(&entry).Error)
	assert.Equal(t, 2, entry.StepIndex, "mine 25 and fish 25 both complete in one pass")

	bal, _ := s.GetBalance(1)
	assert.Equal(t, 200, bal, "both rewards granted")
}

// TestNoRewardTwice verifies a step already completed does not re-issue rewards.
func TestNoRewardTwice(t *testing.T) {
	s := testStore(t)
	New(s)

	require.NoError(t, s.DB.Create(&model.UserStat{UserID: 1, ItemsMined: 25}).Error)
	require.NoError(t, s.RecordActivity(1, "items_mined", 1))
	require.NoError(t, s.RecordActivity(1, "items_mined", 1))

	bal, _ := s.GetBalance(1)
	assert.Equal(t, 150, bal, "reward granted once despite repeated activity")
}

// TestViewProgress verifies progress bars and ranks are derived from stats.
func TestViewProgress(t *testing.T) {
	s := testStore(t)
	svc := New(s)

	v, err := svc.View(1)
	require.NoError(t, err)
	require.Len(t, v.Paths, 8)

	var prospector *PathView
	for i := range v.Paths {
		if v.Paths[i].PathID == "prospector" {
			prospector = &v.Paths[i]
		}
	}
	require.NotNil(t, prospector)
	assert.Equal(t, 0, prospector.Rank, "no progress yet")
	assert.False(t, prospector.Steps[0].Done)

	require.NoError(t, s.DB.Create(&model.UserStat{UserID: 1, ItemsMined: 10}).Error)
	v, err = svc.View(1)
	require.NoError(t, err)
	for i := range v.Paths {
		if v.Paths[i].PathID == "prospector" {
			prospector = &v.Paths[i]
		}
	}
	assert.Equal(t, 10, prospector.Steps[0].Progress)
	assert.Equal(t, 25, prospector.Steps[0].Target)
	assert.False(t, prospector.Steps[0].Done)
}

// TestTrack verifies pinning a path toggles the tracked flag.
func TestTrack(t *testing.T) {
	s := testStore(t)
	svc := New(s)

	require.NoError(t, svc.Track(1, "champion"))

	var entry model.UserJournalEntry
	require.NoError(t, s.DB.Where("user_id = ? AND path_id = ?", 1, "champion").First(&entry).Error)
	assert.True(t, entry.Tracked)

	require.NoError(t, svc.Track(1, "champion"))
	require.NoError(t, s.DB.Where("user_id = ? AND path_id = ?", 1, "champion").First(&entry).Error)
	assert.False(t, entry.Tracked)
}

// TestLeaderboard verifies ordering by completed steps then progress.
func TestLeaderboard(t *testing.T) {
	s := testStore(t)
	svc := New(s)

	require.NoError(t, s.DB.Create(&model.UserStat{UserID: 1, ItemsMined: 25}).Error)
	require.NoError(t, s.DB.Create(&model.UserStat{UserID: 2, ItemsMined: 5}).Error)
	require.NoError(t, s.RecordActivity(1, "items_mined", 1))
	require.NoError(t, s.RecordActivity(2, "items_mined", 1))

	entries, err := svc.Leaderboard("prospector", 5)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, int64(1), entries[0].UserID, "user with more steps ranks first")
	assert.Equal(t, int64(2), entries[1].UserID)
}

// TestRankFor maps completion fractions to ranks.
func TestRankFor(t *testing.T) {
	assert.Equal(t, 0, RankFor(0, 8))
	assert.Equal(t, 1, RankFor(1, 8))
	assert.Equal(t, 2, RankFor(2, 8))
	assert.Equal(t, 3, RankFor(4, 8))
	assert.Equal(t, 4, RankFor(7, 8))
}

// TestHunterAllZonesBoss verifies the "slay every zone boss" closure completes
// only when all seven zones have at least one boss kill.
func TestHunterAllZonesBoss(t *testing.T) {
	s := testStore(t)
	New(s)

	require.NoError(t, s.DB.Create(&model.UserStat{UserID: 1, PveWins: 25}).Error)
	for _, z := range []string{"forest", "cave", "desert", "mountain", "ocean", "tundra"} {
		require.NoError(t, s.DB.Create(&model.UserHuntUnlock{UserID: 1, ZoneKey: z}).Error)
		require.NoError(t, s.DB.Create(&model.UserHuntZoneStat{UserID: 1, ZoneKey: z, BossKills: 5}).Error)
	}
	require.NoError(t, s.RecordActivity(1, "pve_wins", 1))

	var entry model.UserJournalEntry
	require.NoError(t, s.DB.Where("user_id = ? AND path_id = ?", 1, "hunter").First(&entry).Error)
	assert.Equal(t, 5, entry.StepIndex, "six zones are not enough for the all-zones step")

	// Slay the volcano boss too: all zones beaten -> steps cascade to 7 done.
	require.NoError(t, s.DB.Create(&model.UserHuntZoneStat{UserID: 1, ZoneKey: "volcano", BossKills: 5}).Error)
	require.NoError(t, s.RecordActivity(1, "pve_wins", 1))

	require.NoError(t, s.DB.Where("user_id = ? AND path_id = ?", 1, "hunter").First(&entry).Error)
	assert.Equal(t, 7, entry.StepIndex, "all zones beaten advances past the all-zones step")
}

// TestHistorianLore verifies lore-fragment milestones drive the path forward.
func TestHistorianLore(t *testing.T) {
	s := testStore(t)
	New(s)

	for i := 0; i < 5; i++ {
		require.NoError(t, s.DB.Create(&model.UserLoreEntry{UserID: 1, LoreID: "lore_" + string(rune('a'+i))}).Error)
	}
	require.NoError(t, s.RecordActivity(1, "items_mined", 1))

	var entry model.UserJournalEntry
	require.NoError(t, s.DB.Where("user_id = ? AND path_id = ?", 1, "historian").First(&entry).Error)
	assert.Equal(t, 1, entry.StepIndex, "five lore fragments complete the first historian step")
}

// completeAllPaths forces every journal path to its final step.
func completeAllPaths(t *testing.T, s *store.Store, userID int64) {
	for _, p := range GetPaths() {
		require.NoError(t, s.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "path_id"}},
			DoUpdates: clause.Assignments(map[string]any{"step_index": len(p.Steps)}),
		}).Create(&model.UserJournalEntry{UserID: userID, PathID: p.ID, StepIndex: len(p.Steps)}).Error)
	}
}

// TestMasteryUnlock verifies the secret Mastery legend unlocks once, with its
// reward, when every path is completed.
func TestMasteryUnlock(t *testing.T) {
	s := testStore(t)
	svc := New(s)

	completeAllPaths(t, s, 1)

	v, err := svc.View(1)
	require.NoError(t, err)
	assert.True(t, v.MasteryNew, "first View reveals the newly earned Mastery")
	assert.True(t, v.MasteryUnlocked)

	var m int64
	require.NoError(t, s.DB.Model(&model.UserJournalMastery{}).Where("user_id = ?", 1).Count(&m).Error)
	assert.Equal(t, int64(1), m, "mastery row created")

	var medallion model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "mastery_medallion").First(&medallion).Error)
	assert.Equal(t, 1, medallion.Quantity, "medallion granted")

	var crowns int
	require.NoError(t, s.DB.Raw("SELECT crowns FROM users WHERE user_id = ?", 1).Scan(&crowns).Error)
	assert.Equal(t, 100, crowns, "crowns granted")

	var achCount int64
	require.NoError(t, s.DB.Model(&model.UserAchievement{}).Where("user_id = ? AND achievement_id = ?", 1, "journal_mastery").Count(&achCount).Error)
	assert.Equal(t, int64(1), achCount, "hidden mastery achievement recorded")

	// Second View must not re-grant anything.
	v, err = svc.View(1)
	require.NoError(t, err)
	assert.False(t, v.MasteryNew, "unlock happens exactly once")
	assert.True(t, v.MasteryUnlocked)

	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "mastery_medallion").First(&medallion).Error)
	assert.Equal(t, 1, medallion.Quantity, "medallion not granted twice")

	require.NoError(t, s.DB.Raw("SELECT crowns FROM users WHERE user_id = ?", 1).Scan(&crowns).Error)
	assert.Equal(t, 100, crowns, "crowns not granted twice")
}

// TestMasteryNotUnlocked verifies an incomplete path prevents the unlock.
func TestMasteryNotUnlocked(t *testing.T) {
	s := testStore(t)
	svc := New(s)

	completeAllPaths(t, s, 1)
	// Leave one path incomplete.
	require.NoError(t, s.DB.Model(&model.UserJournalEntry{}).
		Where("user_id = ? AND path_id = ?", 1, "hunter").
		Update("step_index", 0).Error)

	v, err := svc.View(1)
	require.NoError(t, err)
	assert.False(t, v.MasteryNew)
	assert.False(t, v.MasteryUnlocked)

	var m int64
	require.NoError(t, s.DB.Model(&model.UserJournalMastery{}).Where("user_id = ?", 1).Count(&m).Error)
	assert.Equal(t, int64(0), m, "no mastery row while a path is incomplete")
}
