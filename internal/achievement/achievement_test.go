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
