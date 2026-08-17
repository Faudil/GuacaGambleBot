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

	// AutoMigrate cannot change primary keys on an existing table; the data
	// migration must rebuild it with the composite key or buying a second house
	// would hit a user_id unique-constraint error.
	assert.Equal(t, []string{"user_id", "house_type"}, requirePKs(t, d, "user_housing"))
}

// TestMigrateLegacyFurnitureScopesToActiveHouse simulates the pre-house-scoped
// user_furniture schema and verifies Migrate rebuilds it with house_type in the
// primary key, assigning rows to the user's active house.
func TestMigrateLegacyFurnitureScopesToActiveHouse(t *testing.T) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "furn.db")), &gorm.Config{})
	require.NoError(t, err)
	now := time.Now()

	require.NoError(t, d.Exec(`CREATE TABLE user_housing (
		user_id INTEGER PRIMARY KEY,
		house_type TEXT,
		level INTEGER DEFAULT 1,
		last_collected DATETIME,
		is_active INTEGER DEFAULT 0,
		stored_items TEXT DEFAULT '{}'
	)`).Error)
	require.NoError(t, d.Exec(`INSERT INTO user_housing (user_id, house_type, is_active) VALUES (1, 'brick_house', 1), (2, 'mansion', 1)`).Error)

	require.NoError(t, d.Exec(`CREATE TABLE user_furniture (
		user_id INTEGER NOT NULL,
		furniture_id TEXT NOT NULL,
		placed_at DATETIME,
		PRIMARY KEY (user_id, furniture_id)
	)`).Error)
	require.NoError(t, d.Exec(`INSERT INTO user_furniture (user_id, furniture_id, placed_at) VALUES (1, 'genetics_lab', ?), (1, 'forge', ?), (2, 'magnetic_coil', ?)`, now, now, now).Error)

	require.NoError(t, Migrate(d))

	assert.Equal(t, []string{"user_id", "house_type", "furniture_id"}, requirePKs(t, d, "user_furniture"))

	var rows []struct {
		UserID      int64
		HouseType   string
		FurnitureID string
	}
	require.NoError(t, d.Table("user_furniture").Order("user_id, furniture_id").Scan(&rows).Error)
	require.Len(t, rows, 3)
	expected := []struct {
		UserID      int64
		HouseType   string
		FurnitureID string
	}{
		{1, "brick_house", "forge"},
		{1, "brick_house", "genetics_lab"},
		{2, "mansion", "magnetic_coil"},
	}
	for i, r := range rows {
		assert.Equal(t, expected[i], r, "legacy furniture must be scoped to the user's active house")
	}
}

func requirePKs(t *testing.T, d *gorm.DB, table string) []string {
	t.Helper()
	cols, err := tableColumns(d, table)
	require.NoError(t, err)
	var pks []string
	for _, c := range cols {
		if c.PK > 0 {
			pks = append(pks, c.Name)
		}
	}
	return pks
}
