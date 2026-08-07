package achievement

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/db"
	"guacagamblebot/internal/model"
)

func testDB(t *testing.T) *gorm.DB {
	d, err := gorm.Open(sqlite.Open(t.TempDir()+"/a.db"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	return d
}

func TestIncrementAndUnlock(t *testing.T) {
	d := testDB(t)
	// User with balance 2000 should unlock eco_1k.
	require.NoError(t, d.Create(&model.User{UserID: 1, Balance: 2000}).Error)

	require.NoError(t, IncrementStat(d, 1, "daily_uses", 1))

	unlocks, err := CheckAndUnlock(d, 1)
	require.NoError(t, err)

	ids := map[string]bool{}
	for _, a := range unlocks {
		ids[a.ID] = true
	}
	assert.True(t, ids["eco_1k"], "eco_1k should unlock at balance 2000")
	assert.False(t, ids["eco_10k"], "eco_10k should not unlock at balance 2000")
}

func TestDailyAchievementProgression(t *testing.T) {
	d := testDB(t)
	require.NoError(t, d.Create(&model.User{UserID: 2, Balance: 100}).Error)

	// First daily claim -> daily_1 unlocks, daily_10 does not.
	require.NoError(t, IncrementStat(d, 2, "daily_uses", 1))
	unlocks, err := CheckAndUnlock(d, 2)
	require.NoError(t, err)
	assertContains(t, unlocks, "daily_1")
	assertNotContains(t, unlocks, "daily_10")

	// Re-evaluating without new claims must not re-unlock.
	unlocks, err = CheckAndUnlock(d, 2)
	require.NoError(t, err)
	assert.Empty(t, unlocks)
}

func assertContains(t *testing.T, list []*Achievement, id string) {
	t.Helper()
	for _, a := range list {
		if a.ID == id {
			return
		}
	}
	t.Errorf("expected achievement %q in unlocks", id)
}

func assertNotContains(t *testing.T, list []*Achievement, id string) {
	t.Helper()
	for _, a := range list {
		if a.ID == id {
			t.Errorf("did not expect achievement %q in unlocks", id)
			return
		}
	}
}

func TestHuntZoneUnlockAchievements(t *testing.T) {
	d := testDB(t)
	require.NoError(t, d.Create(&model.User{UserID: 3, Balance: 100}).Error)

	// No unlocks yet: none of the zone achievements fire.
	unlocks, err := CheckAndUnlock(d, 3)
	require.NoError(t, err)
	assertNotContains(t, unlocks, "hunt_unlock_mountain")
	assertNotContains(t, unlocks, "hunt_unlock_volcano")

	// Unlock mountain -> only the mountain achievement fires.
	require.NoError(t, d.Create(&model.UserHuntUnlock{UserID: 3, ZoneKey: "mountain"}).Error)
	unlocks, err = CheckAndUnlock(d, 3)
	require.NoError(t, err)
	assertContains(t, unlocks, "hunt_unlock_mountain")
	assertNotContains(t, unlocks, "hunt_unlock_ocean")

	// Unlock volcano -> the volcano achievement fires as well.
	require.NoError(t, d.Create(&model.UserHuntUnlock{UserID: 3, ZoneKey: "volcano"}).Error)
	unlocks, err = CheckAndUnlock(d, 3)
	require.NoError(t, err)
	assertContains(t, unlocks, "hunt_unlock_volcano")
}

func TestHuntBossKillAchievements(t *testing.T) {
	d := testDB(t)
	require.NoError(t, d.Create(&model.User{UserID: 4, Balance: 100}).Error)

	// One forest boss kill -> first-boss achievement only.
	require.NoError(t, d.Create(&model.UserHuntZoneStat{UserID: 4, ZoneKey: "forest", Wins: 1, BossKills: 1}).Error)
	unlocks, err := CheckAndUnlock(d, 4)
	require.NoError(t, err)
	assertContains(t, unlocks, "hunt_boss_forest")
	assertNotContains(t, unlocks, "hunt_boss_10")
	assertNotContains(t, unlocks, "hunt_boss_cave")

	// Total of 10 boss kills across zones -> hunt_boss_10 fires.
	require.NoError(t, d.Create(&model.UserHuntZoneStat{UserID: 4, ZoneKey: "cave", BossKills: 4}).Error)
	require.NoError(t, d.Create(&model.UserHuntZoneStat{UserID: 4, ZoneKey: "desert", BossKills: 5}).Error)
	unlocks, err = CheckAndUnlock(d, 4)
	require.NoError(t, err)
	assertContains(t, unlocks, "hunt_boss_10")
	assertNotContains(t, unlocks, "hunt_boss_50")

	// 100 total -> the top milestone fires.
	require.NoError(t, d.Model(&model.UserHuntZoneStat{}).
		Where("user_id = ? AND zone_key = ?", int64(4), "forest").Update("boss_kills", 91).Error)
	unlocks, err = CheckAndUnlock(d, 4)
	require.NoError(t, err)
	assertContains(t, unlocks, "hunt_boss_100")
}
