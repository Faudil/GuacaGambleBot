package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/model"
)

func TestMiningSessionStore(t *testing.T) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "m.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	s := New(d, &config.Config{StartingBalance: 100})

	got, err := s.GetMiningSession(42)
	require.NoError(t, err)
	assert.Nil(t, got, "no session should exist initially")

	now := time.Now()
	err = s.SaveMiningSession(&model.MiningSession{
		UserID:         42,
		Depth:          7,
		ToolID:         "steel_pickaxe",
		GhostVeilTurns: 2,
		RiskMod:        -5,
		RiskTurns:      3,
		Bag:            `[{"Name":"coal","Count":3}]`,
		UpdatedAt:      now,
	})
	require.NoError(t, err)

	got, err = s.GetMiningSession(42)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int64(42), got.UserID)
	assert.Equal(t, 7, got.Depth)
	assert.Equal(t, "steel_pickaxe", got.ToolID)
	assert.Equal(t, 2, got.GhostVeilTurns)
	assert.Equal(t, -5, got.RiskMod)
	assert.Equal(t, 3, got.RiskTurns)
	assert.Equal(t, `[{"Name":"coal","Count":3}]`, got.Bag)
	assert.WithinDuration(t, now, got.UpdatedAt, time.Second)

	// Upsert updates the existing row instead of creating a duplicate.
	err = s.SaveMiningSession(&model.MiningSession{
		UserID:         42,
		Depth:          8,
		ToolID:         "steel_pickaxe",
		GhostVeilTurns: 1,
		RiskMod:        -5,
		RiskTurns:      2,
		Bag:            `[{"Name":"coal","Count":4}]`,
		UpdatedAt:      now.Add(time.Minute),
	})
	require.NoError(t, err)
	var count int64
	require.NoError(t, s.DB.Model(&model.MiningSession{}).Where("user_id = ?", 42).Count(&count).Error)
	assert.Equal(t, int64(1), count, "upsert must not duplicate the row")
	got, err = s.GetMiningSession(42)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 8, got.Depth)
	assert.Equal(t, `[{"Name":"coal","Count":4}]`, got.Bag)

	require.NoError(t, s.DeleteMiningSession(42))
	got, err = s.GetMiningSession(42)
	require.NoError(t, err)
	assert.Nil(t, got)
}
