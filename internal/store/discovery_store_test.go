package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddItemRawRecordsDiscovery(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "coal", 1))

	discovered, err := s.DiscoveredItemIDs(1)
	require.NoError(t, err)
	assert.True(t, discovered["coal"])

	_, ok := s.ItemDiscoveredAt(1, "coal")
	assert.True(t, ok)
}

func TestAddItemRawDiscoveryIsIdempotent(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "coal", 1))
	first, _ := s.ItemDiscoveredAt(1, "coal")

	require.NoError(t, s.AddItemRaw(s.DB, 1, "coal", 5))
	second, _ := s.ItemDiscoveredAt(1, "coal")

	assert.Equal(t, first, second, "re-granting an already-discovered item must not overwrite the first discovery date")
}

func TestAddItemRawSkipsDiscoveryOnUnknownItem(t *testing.T) {
	s := newStore(t)
	require.Error(t, s.AddItemRaw(s.DB, 1, "not_a_real_item", 1))

	discovered, err := s.DiscoveredItemIDs(1)
	require.NoError(t, err)
	assert.Empty(t, discovered)
}

func TestCreateEquipmentRecordsDiscovery(t *testing.T) {
	s := newStore(t)
	_, err := s.CreateEquipment(1, "stick", "Stick", "🪵", "common", "weapon", 1, 0, 0, 0, 0, 0, []byte("[]"), "")
	require.NoError(t, err)

	discovered, err := s.DiscoveredItemIDs(1)
	require.NoError(t, err)
	assert.True(t, discovered["stick"])
}

func TestDiscoveredItemIDsIsPerUser(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "coal", 1))

	discovered, err := s.DiscoveredItemIDs(2)
	require.NoError(t, err)
	assert.Empty(t, discovered, "another user must not see item 1's discoveries")
}
