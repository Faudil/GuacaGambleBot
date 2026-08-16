package db

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestMigrateExistingHousingUpgradesOldSchema simulates a database created with
// the legacy single-house schema (user_id as the only primary key) and verifies
// that Migrate rebuilds the table with the composite key and backfills is_active.
func TestMigrateExistingHousingUpgradesOldSchema(t *testing.T) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "old.db")), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, d.Exec(`CREATE TABLE user_housing (
		user_id INTEGER PRIMARY KEY,
		house_type TEXT,
		level INTEGER DEFAULT 1,
		last_collected DATETIME,
		custom_name TEXT,
		custom_color TEXT,
		under_construction TEXT,
		finish_time DATETIME,
		stored_items TEXT DEFAULT '{}'
	)`).Error)
	now := time.Now()
	require.NoError(t, d.Exec(`INSERT INTO user_housing (user_id, house_type, level, last_collected, stored_items) VALUES (?, ?, ?, ?, ?)`,
		1, "brick_house", 3, now, "{}").Error)

	require.NoError(t, Migrate(d))

	var count int64
	require.NoError(t, d.Table("user_housing").Count(&count).Error)
	assert.Equal(t, int64(1), count)

	var active int64
	require.NoError(t, d.Table("user_housing").Where("user_id = ?", 1).Where("is_active = ?", true).Count(&active).Error)
	assert.Equal(t, int64(1), active)
}
