package delve

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateRoomLootEquipmentShare checks that room loot rolls yield
// equipment at the configured share (55%) and that every non-equipment roll
// grants exactly one of gold, a heal or a misc item.
func TestGenerateRoomLootEquipmentShare(t *testing.T) {
	const iterations = 20000
	equipment := 0
	for i := 0; i < iterations; i++ {
		loot := GenerateRoomLoot("crypt", 1, 0)
		require.NotNil(t, loot)
		if loot.Item.EquipSlot != "" {
			equipment++
			continue
		}
		nonEquip := 0
		if loot.Gold > 0 {
			nonEquip++
		}
		if loot.Heal > 0 {
			nonEquip++
		}
		if loot.Item.ID != "" {
			require.Empty(t, loot.Item.EquipSlot, "misc items must not be equipment")
			nonEquip++
		}
		assert.Equal(t, 1, nonEquip, "a non-equipment roll must grant exactly one of gold/heal/item")
	}

	share := float64(equipment) / float64(iterations)
	assert.InDelta(t, 0.55, share, 0.03,
		"equipment share must be ~55%% (got %.3f)", share)
}

// TestMiscLootItemsAreStackableNonEquipment ensures the fallback item pool
// only produces non-equipment, stackable finds.
func TestMiscLootItemsAreStackableNonEquipment(t *testing.T) {
	for i := 0; i < 200; i++ {
		it := randomMiscItem()
		assert.NotEmpty(t, it.ID)
		assert.Empty(t, it.EquipSlot, "misc loot must not be equipment")
		assert.GreaterOrEqual(t, it.Quantity, 1)
	}
}
