package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/model"
)

func TestInventoryLimitBase(t *testing.T) {
	s := newStore(t)
	limit, err := s.InventoryLimit(s.DB, 1)
	require.NoError(t, err)
	assert.Equal(t, BaseInventoryLimit, limit)
}

func TestInventoryLimitHonorsExtraSlots(t *testing.T) {
	s := newStore(t)
	_, err := s.GetBalance(1)
	require.NoError(t, err)
	require.NoError(t, s.DB.Model(&model.User{}).Where("user_id = ?", 1).
		Update("extra_inv_slots", 50).Error)

	limit, err := s.InventoryLimit(s.DB, 1)
	require.NoError(t, err)
	assert.Equal(t, BaseInventoryLimit+50, limit)
}

func TestInventoryUsedCountsUnitsAndEquipment(t *testing.T) {
	s := newStore(t)
	_, err := s.GetBalance(1)
	require.NoError(t, err)
	require.NoError(t, s.DB.Create(&model.Inventory{UserID: 1, ItemID: "coal", Quantity: 30}).Error)
	require.NoError(t, s.DB.Create(&model.Inventory{UserID: 1, ItemID: "beer", Quantity: 5}).Error)
	_, err = s.CreateEquipment(1, "stick", "Stick", "🪵", "common", "weapon", 1, 0, 0, 0, 0, 0, []byte("[]"), "")
	require.NoError(t, err)

	used, err := s.InventoryUsed(s.DB, 1)
	require.NoError(t, err)
	assert.Equal(t, 36, used)

	free, err := s.FreeSlots(s.DB, 1)
	require.NoError(t, err)
	assert.Equal(t, BaseInventoryLimit-36, free)
}

func TestAddItemRawIgnoresNonPositiveQuantity(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "coal", 0))
	require.NoError(t, s.AddItemRaw(s.DB, 1, "coal", -1))
	used, err := s.InventoryUsed(s.DB, 1)
	require.NoError(t, err)
	assert.Equal(t, 0, used)
}

func TestAddItemRawAllowsOverflow(t *testing.T) {
	s := newStore(t)
	_, err := s.GetBalance(1)
	require.NoError(t, err)
	// A big drop may legitimately overflow the capacity; it must be granted.
	require.NoError(t, s.AddItemRaw(s.DB, 1, "coal", BaseInventoryLimit+25))
	used, err := s.InventoryUsed(s.DB, 1)
	require.NoError(t, err)
	assert.Equal(t, BaseInventoryLimit+25, used)
}
