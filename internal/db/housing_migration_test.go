package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"guacagamblebot/internal/config"
)

func gormOpen(path string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
}

func openTest(t *testing.T, path string) {
	cfg := &config.Config{DBPath: path}
	if _, err := Open(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateCustomHousingShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prod.db")
	g, err := gormOpen(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Exec(`CREATE TABLE "user_housing" (
	user_id INTEGER NOT NULL,
	house_type TEXT NOT NULL,
	level INTEGER DEFAULT 1,
	last_collected DATETIME,
	is_active INTEGER DEFAULT 0,
	custom_name TEXT,
	custom_color TEXT,
	under_construction TEXT,
	finish_time DATETIME,
	stored_items TEXT DEFAULT '{}',
	PRIMARY KEY (user_id, house_type)
)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := g.Exec(`INSERT INTO user_housing (user_id, house_type, level, is_active, stored_items) VALUES (1,'wooden_shack',1,1,'{}'),(2,'cardboard_box',1,1,'{}'),(3,'cabin',1,0,'{}'),(4,'cardboard_box',1,1,'{}')`).Error; err != nil {
		t.Fatal(err)
	}
	sqlDB, err := g.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}

	openTest(t, path)         // first migrate: must repair
	g2, err := gormOpen(path) // second migrate must be a no-op, data intact
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := g2.Raw("SELECT count(*) FROM user_housing").Row().Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("expected 4 rows, got %d", count)
	}
	var ht string
	if err := g2.Raw("SELECT house_type FROM user_housing WHERE user_id = 3").Row().Scan(&ht); err != nil {
		t.Fatal(err)
	}
	if ht != "cabin" {
		t.Fatalf("expected cabin, got %s", ht)
	}
}

func TestMigrateLegacyHousingShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	g, err := gormOpen(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Exec(`CREATE TABLE "user_housing" (user_id INTEGER PRIMARY KEY AUTOINCREMENT, house_type text, level integer DEFAULT 1, last_collected datetime, custom_name text, custom_color text, under_construction text, finish_time datetime, stored_items text DEFAULT '{}')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := g.Exec(`INSERT INTO user_housing (user_id, house_type, level, stored_items) VALUES (1,'wooden_shack',2,'{}'),(2,NULL,1,'{}')`).Error; err != nil {
		t.Fatal(err)
	}
	sqlDB, err := g.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}

	openTest(t, path)
	g2, err := gormOpen(path)
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := g2.Raw("SELECT count(*) FROM user_housing").Row().Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows, got %d", count)
	}
	var ht string
	if err := g2.Raw("SELECT house_type FROM user_housing WHERE user_id = 2").Row().Scan(&ht); err != nil {
		t.Fatal(err)
	}
	if ht != "cardboard_box" {
		t.Fatalf("expected cardboard_box default, got %s", ht)
	}
}

func TestMigrateProdDBCopy(t *testing.T) {
	// Regression check against a snapshot of the production DB. Skips when the
	// snapshot isn't present (CI / other machines).
	if _, err := os.Stat("/tmp/guacarepro/guacabot_go.db"); err != nil {
		t.Skip("prod DB snapshot not present")
	}
	openTest(t, "/tmp/guacarepro/guacabot_go.db")
}
