package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEquipInstanceEnforcesMinLevel(t *testing.T) {
	s := newStore(t)

	// Give user 1 a legendary-tier item (level 20) while they are level 1.
	eq, err := s.CreateEquipment(1, "dragon_slayer_sword", "Dragon Slayer Sword", "🗡️",
		"legendary", "weapon", 20, 12, 0, 0, 4, 0, []byte("[]"), "dragon_slayer")
	require.NoError(t, err)

	err = s.EquipInstance(1, eq.ID)
	assert.ErrorIs(t, err, ErrLevelTooLow)

	// A level 1 item equips fine.
	eq2, err := s.CreateEquipment(1, "stick", "Wooden Stick", "🪵",
		"common", "weapon", 1, 2, 0, 0, 0, 0, []byte("[]"), "")
	require.NoError(t, err)
	require.NoError(t, s.EquipInstance(1, eq2.ID))

	equipped, err := s.GetEquipped(1)
	require.NoError(t, err)
	require.Len(t, equipped, 1)
	assert.Equal(t, "stick", equipped[0].BaseID)
}

func TestCreateEquipmentSnapshotsMinLevel(t *testing.T) {
	s := newStore(t)
	eq, err := s.CreateEquipment(1, "spark_shard", "Spark Shard", "⚡",
		"epic", "trinket", 15, 5, 0, 0, 2, 0, []byte("[]"), "")
	require.NoError(t, err)
	assert.Equal(t, 15, eq.MinLevel)

	// Zero min level is clamped to 1.
	eq2, err := s.CreateEquipment(1, "x", "X", "❌", "common", "trinket", 0, 0, 0, 0, 0, 0, []byte("[]"), "")
	require.NoError(t, err)
	assert.Equal(t, 1, eq2.MinLevel)
}
