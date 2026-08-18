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

func TestAddItemRawNormalizesDisplayName(t *testing.T) {
	s := newStore(t)
	// The daily shop used to grant items keyed by their display name. The store
	// must resolve the key and save under the canonical id so canonical-ID
	// lookups (crafting, farm, use) can find it.
	require.NoError(t, s.AddItemRaw(s.DB, 1, "Fertilizer", 1))

	var inv model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "fertilizer").First(&inv).Error)
	assert.Equal(t, 1, inv.Quantity)

	var orphan model.Inventory
	err := s.DB.Where("user_id = ? AND item_id = ?", 1, "Fertilizer").First(&orphan).Error
	assert.Error(t, err, "no display-name-keyed row may be created")
}

func TestAddItemRawRejectsUnknownItem(t *testing.T) {
	s := newStore(t)
	err := s.AddItemRaw(s.DB, 1, "not_a_real_item", 1)
	assert.ErrorIs(t, err, ErrUnknownItem)
}

func TestHasItemNormalizesDisplayName(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "fertilizer", 2))

	has, err := s.HasItem(1, "Fertilizer", 1)
	require.NoError(t, err)
	assert.True(t, has, "display-name lookup must find the canonical row")

	has, err = s.HasItem(1, "Fertilizer", 3)
	require.NoError(t, err)
	assert.False(t, has, "quantity check still applies")

	has, err = s.HasItem(1, "not_a_real_item", 1)
	require.NoError(t, err)
	assert.False(t, has)
}

func TestRemoveInventoryItemNormalizesDisplayName(t *testing.T) {
	s := newStore(t)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "coal", 5))

	require.NoError(t, s.RemoveInventoryItem(1, "Coal", 2))

	var inv model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "coal").First(&inv).Error)
	assert.Equal(t, 3, inv.Quantity)
}
