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

// pathView returns the view for a single path ID.
func pathView(t *testing.T, v *JournalView, id string) *PathView {
	for i := range v.Paths {
		if v.Paths[i].PathID == id {
			return &v.Paths[i]
		}
	}
	t.Fatalf("path %s not found", id)
	return nil
}

// TestStepHiddenUntilDiscover verifies the first step stays a mystery until
// its Discover check passes, then surfaces with progress.
func TestStepHiddenUntilDiscover(t *testing.T) {
	s := testStore(t)
	svc := New(s)

	// No activity yet: the prospector rumor has not been heard.
	v, err := svc.View(1)
	require.NoError(t, err)
	pv := pathView(t, v, "prospector")
	assert.False(t, pv.HasRumor, "rumor hidden before any mining")
	assert.False(t, pv.Steps[0].Discovered)
	assert.Equal(t, 0, pv.Steps[0].Progress, "no progress leaks from a hidden rumor")

	// Below the discover threshold: still a mystery.
	require.NoError(t, s.DB.Create(&model.UserStat{UserID: 1, ItemsMined: 3}).Error)
	v, err = svc.View(1)
	require.NoError(t, err)
	pv = pathView(t, v, "prospector")
	assert.False(t, pv.HasRumor, "still hidden at 3 ores mined")

	// Threshold crossed: the rumor surfaces with progress.
	require.NoError(t, s.DB.Model(&model.UserStat{}).Where("user_id = ?", 1).Update("items_mined", 5).Error)
	v, err = svc.View(1)
	require.NoError(t, err)
	pv = pathView(t, v, "prospector")
	assert.True(t, pv.HasRumor, "rumor surfaced at 5 ores mined")
	assert.True(t, pv.Steps[0].Discovered)
	assert.Equal(t, 5, pv.Steps[0].Progress)
	assert.Equal(t, 25, pv.Steps[0].Target)
}

// TestNextRumorSurfacesOnCompletion verifies the following step surfaces
// automatically once the previous one is done (no Discover gate).
func TestNextRumorSurfacesOnCompletion(t *testing.T) {
	s := testStore(t)
	svc := New(s)

	require.NoError(t, s.DB.Create(&model.UserStat{UserID: 1, ItemsMined: 25}).Error)
	require.NoError(t, s.RecordActivity(1, "items_mined", 1))

	v, err := svc.View(1)
	require.NoError(t, err)
	pv := pathView(t, v, "prospector")
	assert.Equal(t, 1, pv.Completed, "first step done")
	assert.True(t, pv.Steps[0].Done)
	assert.True(t, pv.HasRumor, "second step surfaces without a discover gate")
	assert.True(t, pv.Steps[1].Discovered)
	assert.Equal(t, 25, pv.Steps[1].Target)
}

// TestRankUpQueuesSighting verifies the first rank queues only a rank-up
// scene, the first rank 2 queues the Chronicler sighting scene (DM preferred)
// exactly once, and later rank-ups queue the lighter rank-up scene again.
func TestRankUpQueuesSighting(t *testing.T) {
	s := testStore(t)
	New(s)

	// First rank (0 -> 1): just a rank-up scene, no reveal yet.
	require.NoError(t, s.DB.Create(&model.UserStat{UserID: 1, ItemsMined: 25}).Error)
	require.NoError(t, s.RecordActivity(1, "items_mined", 1))

	sc, ok := s.PopJournalScene(1)
	require.True(t, ok, "rank-up scene queued on first rank")
	assert.Equal(t, "journal.rankup", sc.Key)
	assert.False(t, sc.DM, "rank-up is not a DM")
	_, ok = s.PopJournalScene(1)
	assert.False(t, ok, "no sighting at first rank")

	// Reaching rank 2 (4 completed prospector steps) reveals the Chronicler.
	require.NoError(t, s.DB.Model(&model.UserStat{}).Where("user_id = ?", 1).
		Update("items_fished", 25).Error)
	require.NoError(t, s.RecordActivity(1, "items_fished", 1))
	require.NoError(t, s.DB.Model(&model.UserStat{}).Where("user_id = ?", 1).
		Update("items_farmed", 25).Error)
	require.NoError(t, s.RecordActivity(1, "items_farmed", 1))
	require.NoError(t, s.DB.Model(&model.UserStat{}).Where("user_id = ?", 1).
		Update("pve_wins", 10).Error)
	require.NoError(t, s.RecordActivity(1, "pve_wins", 1))

	sc, ok = s.PopJournalScene(1)
	require.True(t, ok, "sighting scene queued on rank 2")
	assert.Equal(t, "journal.chronicler.sighting", sc.Key)
	assert.True(t, sc.DM, "sighting prefers DM delivery")

	// A later rank-up (2 -> 3) queues a lighter rankup scene instead.
	require.NoError(t, s.DB.Model(&model.UserStat{}).Where("user_id = ?", 1).
		Update("items_farmed", 100).Error)
	require.NoError(t, s.RecordActivity(1, "items_farmed", 1))

	sc, ok = s.PopJournalScene(1)
	require.True(t, ok)
	assert.Equal(t, "journal.rankup", sc.Key)
	assert.False(t, sc.DM)

	// No more scenes queued by a repeated check.
	_, ok = s.PopJournalScene(1)
	assert.False(t, ok)
}

// TestSceneLinePopsQueued verifies queued scenes surface first.
func TestSceneLinePopsQueued(t *testing.T) {
	s := testStore(t)
	New(s)

	s.PushJournalScene(1, store.JournalScene{Key: "journal.rankup", Params: map[string]any{"rank": 2}})
	text, _ := SceneLine(s, 1, "mining", "en")
	assert.Contains(t, text, "rank")
	_, ok := s.PopJournalScene(1)
	assert.False(t, ok, "queued scene consumed")
}

// TestSceneLineAmbientBeforeMeeting verifies eerie sightings appear before the
// Chronicler is met and are throttled by cooldown.
func TestSceneLineAmbientBeforeMeeting(t *testing.T) {
	s := testStore(t)
	New(s)
	oldRoll := sceneRoll
	sceneRoll = func(int) bool { return true }
	defer func() { sceneRoll = oldRoll }()

	text, dm := SceneLine(s, 1, "mining", "en")
	assert.NotEmpty(t, text, "ambient sighting surfaces for unmet players")
	assert.False(t, dm)

	// Cooldown throttles the next sighting.
	text, _ = SceneLine(s, 1, "mining", "en")
	assert.Empty(t, text)
}

// TestSceneLineRecognitionAfterMeeting verifies recognition lines appear only
// for players ranked in the domain's path.
func TestSceneLineRecognitionAfterMeeting(t *testing.T) {
	s := testStore(t)
	New(s)
	oldRoll := sceneRoll
	sceneRoll = func(int) bool { return true }
	defer func() { sceneRoll = oldRoll }()

	// Meet the Chronicler.
	require.NoError(t, s.DB.Create(&model.UserNPCSecret{
		UserID: 1, NPCID: ChroniclerID, SecretID: ChroniclerIntroSecret, Seen: true,
	}).Error)

	// No rank yet: no recognition anywhere.
	text, _ := SceneLine(s, 1, "mining", "en")
	assert.Empty(t, text, "no recognition without a rank in the domain path")

	// Rank 1 in prospector: recognition in mining.
	require.NoError(t, s.DB.Create(&model.UserJournalEntry{UserID: 1, PathID: "prospector", StepIndex: 1}).Error)
	text, _ = SceneLine(s, 1, "mining", "en")
	assert.NotEmpty(t, text, "recognition surfaces for a ranked player")
	assert.Contains(t, text, "recognition")

	// Unrelated domain stays silent.
	text, _ = SceneLine(s, 1, "lotto", "en")
	assert.Empty(t, text, "no recognition in an unrelated domain")
}
