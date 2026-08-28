package interaction

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func noop(b *Bot, i *discordgo.InteractionCreate) {}

func TestCatalogGroupsAliases(t *testing.T) {
	r := NewRouter(&Bot{Prefix: "!"}, nil)
	r.Categorize("economy", func() {
		// "eco" carries the canonical command's key, so it is an alias of it.
		r.Slash("economy", "cmd.economy.desc", noop)
		r.Slash("eco", "cmd.economy.desc", noop)
		r.Prefix("economy", func(b *Bot, s *discordgo.Session, m *discordgo.Message) {})
		r.Slash("daily", "cmd.daily.desc", noop)
	})

	groups := r.Catalog("economy")
	assert.Len(t, groups, 2, "aliases should fold into their canonical command")
	assert.Equal(t, "economy", groups[0].Name)
	assert.Equal(t, []string{"eco"}, groups[0].Aliases)
	assert.True(t, groups[0].HasPrefix, "economy also has a prefix command")
	assert.Equal(t, "daily", groups[1].Name)
	assert.False(t, groups[1].HasPrefix)
}

// The canonical name is the one matching the description key, whatever order
// the cog registered its aliases in.
func TestCatalogCanonicalFromKeyRegardlessOfOrder(t *testing.T) {
	r := NewRouter(&Bot{Prefix: "!"}, nil)
	r.Categorize("economy", func() {
		r.Slash("bal", "cmd.balance.desc", noop)
		r.Slash("balance", "cmd.balance.desc", noop)
	})
	groups := r.Catalog("economy")
	assert.Len(t, groups, 1)
	assert.Equal(t, "balance", groups[0].Name)
	assert.Equal(t, []string{"bal"}, groups[0].Aliases)
}

func TestCategorizeRestoresPreviousCategory(t *testing.T) {
	r := NewRouter(&Bot{Prefix: "!"}, nil)
	r.Categorize("casino", func() { r.Slash("slots", "cmd.slots.desc", noop) })
	r.Slash("stray", "cmd.stray.desc", noop)

	assert.Equal(t, []string{"casino"}, r.Categories())
	var stray Command
	for _, c := range r.CommandList() {
		if c.Name == "stray" {
			stray = c
		}
	}
	assert.Empty(t, stray.Category, "registrations outside Categorize stay uncategorised")
}
