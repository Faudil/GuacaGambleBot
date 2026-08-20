package testutil

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/db"
)

var (
	templateOnce sync.Once
	templatePath string
	templateErr  error
)

// NewDB returns a freshly opened, fully migrated SQLite database. The full
// migration (AutoMigrate of every model plus data migrations) runs exactly
// once per test binary on a template file; every test then gets a cheap file
// copy of that template, keeping tests isolated while removing the dominant
// per-test cost. Durability pragmas are relaxed since tests are disposable.
func NewDB(t *testing.T) *gorm.DB {
	t.Helper()
	templateOnce.Do(func() {
		f, err := os.CreateTemp("", "gamblebot-migrated-*.db")
		if err != nil {
			templateErr = err
			return
		}
		templatePath = f.Name()
		if err := f.Close(); err != nil {
			templateErr = err
			return
		}
		tmpl, err := gorm.Open(sqlite.Open(templatePath), &gorm.Config{})
		if err != nil {
			templateErr = err
			return
		}
		templateErr = db.Migrate(tmpl)
		if sqlDB, err := tmpl.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, templateErr)

	path := filepath.Join(t.TempDir(), "test.db")
	src, err := os.Open(templatePath)
	require.NoError(t, err)
	defer src.Close()
	dst, err := os.Create(path)
	require.NoError(t, err)
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		require.NoError(t, err)
	}
	require.NoError(t, dst.Close())

	d, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, d.Exec("PRAGMA journal_mode=MEMORY").Error)
	require.NoError(t, d.Exec("PRAGMA synchronous=OFF").Error)
	t.Cleanup(func() {
		if sqlDB, err := d.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return d
}
