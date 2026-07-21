package items

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetExistingItem(t *testing.T) {
	item := Get("charbon")
	require.NotNil(t, item)
	assert.Equal(t, "charbon", item.Name)
	assert.Equal(t, Mining, item.Category)
	assert.True(t, item.Price > 0, "charbon should have a price > 0")
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
