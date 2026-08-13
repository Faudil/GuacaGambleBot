package items

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetExistingItemByID(t *testing.T) {
	item := Get("coal")
	require.NotNil(t, item)
	assert.Equal(t, "coal", item.ID)
	assert.Equal(t, "Coal", item.Name)
	assert.Equal(t, Mining, item.Category)
	assert.True(t, item.Price > 0, "coal should have a price > 0")
}

func TestGetExistingItemByName(t *testing.T) {
	item := Get("Coal")
	require.NotNil(t, item)
	assert.Equal(t, "coal", item.ID)
}

func TestGetMissingItem(t *testing.T) {
	assert.Nil(t, Get("nonexistent_item_xyz"))
}

func TestAllItemsNotEmpty(t *testing.T) {
	all := AllItems()
	assert.Greater(t, len(all), 50, "should have at least 50 items")
}

func TestItemsByCategory(t *testing.T) {
	mining := ItemsByCategory("mining")
	assert.Greater(t, len(mining), 5, "mining category should have items")

	unknown := ItemsByCategory("unknown_cat")
	assert.Empty(t, unknown)
}

func TestItemHasRequiredFields(t *testing.T) {
	for _, item := range AllItems() {
		assert.NotEmpty(t, item.Name, "item must have a name")
		assert.NotEmpty(t, item.Category, "item %q must have a category", item.Name)
		assert.NotEmpty(t, item.Emoji, "item %q must have an emoji", item.Name)
		assert.NotEmpty(t, item.EffectType, "item %q must have an effect type", item.Name)
	}
}

func TestGearHasMinLevel(t *testing.T) {
	for _, item := range AllItems() {
		if item.EquipSlot == "" {
			continue
		}
		assert.Greater(t, item.MinLevel, 0, "gear %q must have a min level", item.Name)
		assert.NotEmpty(t, item.Rarity, "gear %q must have a rarity", item.Name)
	}
}

func TestSetItemsAreLegendary(t *testing.T) {
	for _, id := range []string{
		"dragon_slayer_sword", "shadow_stalker_blade", "arcane_weaver_staff",
	} {
		it := Get(id)
		require.NotNil(t, it)
		assert.Equal(t, RarityLegendary, it.Rarity, "%s should be legendary", id)
		assert.Equal(t, 20, it.MinLevel, "%s should require level 20", id)
	}
}

func TestRiftGearRequiresLevel25(t *testing.T) {
	it := Get("rift_blade")
	require.NotNil(t, it)
	assert.Equal(t, 25, it.MinLevel)
	assert.Equal(t, RarityLegendary, it.Rarity)
}

func TestMinLevelForRarity(t *testing.T) {
	assert.Equal(t, 1, MinLevelForRarity(RarityCommon))
	assert.Equal(t, 5, MinLevelForRarity(RarityUncommon))
	assert.Equal(t, 10, MinLevelForRarity(RarityRare))
	assert.Equal(t, 15, MinLevelForRarity(RarityEpic))
	assert.Equal(t, 20, MinLevelForRarity(RarityLegendary))
}
