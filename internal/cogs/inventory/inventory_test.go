package inventory

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/components"
	invsvc "guacagamblebot/internal/service/inventory"
)

func TestResolveTargetDefaultsToSelf(t *testing.T) {
	id, ok := resolveTarget([]string{"!inv"}, 42)
	assert.True(t, ok)
	assert.Equal(t, int64(42), id)

	id, ok = resolveTarget([]string{"!inv", "<@!123>"}, 42)
	assert.True(t, ok)
	assert.Equal(t, int64(123), id)
}

func TestResolveTargetParsesMentionAndRawID(t *testing.T) {
	for _, arg := range []string{"<@!123>", "<@123>", "@123", "123"} {
		id, ok := resolveTarget([]string{"!inv", arg}, 42)
		assert.True(t, ok, "must parse %q", arg)
		assert.Equal(t, int64(123), id, "parsing %q", arg)
	}
}

func TestResolveTargetRejectsGarbage(t *testing.T) {
	_, ok := resolveTarget([]string{"!inv", "not-a-user"}, 42)
	assert.False(t, ok)
}

func TestBuildEmbedSellOnlyForOwner(t *testing.T) {
	c := &Cog{}
	res := &invsvc.InvResult{
		Entries: []invsvc.InvEntry{{ItemName: "Coal", Quantity: 3}},
		Current: 3,
		Limit:   100,
		UserID:  1,
	}

	_, comps := c.buildEmbed("en", res, "Bob", true)
	assert.NotEmpty(t, comps, "owner view must include the sell button")

	_, comps = c.buildEmbed("en", res, "Bob", false)
	assert.Empty(t, comps, "foreign view must not include the sell button")
}

func TestSellButtonEncodesOwner(t *testing.T) {
	comps := sellButton(42, "en")
	assert.Len(t, comps, 1)

	row, ok := comps[0].(discordgo.ActionsRow)
	require.True(t, ok)
	require.Len(t, row.Components, 1)
	btn, ok := row.Components[0].(discordgo.Button)
	require.True(t, ok)

	ownerID, ok := components.OwnerID(btn.CustomID)
	assert.True(t, ok, "sell button must carry an owner id")
	assert.Equal(t, int64(42), ownerID)
}

func TestBuildEmbedCountsFooter(t *testing.T) {
	c := &Cog{}
	res := &invsvc.InvResult{Current: 7, Limit: 100, UserID: 1}

	embed, _ := c.buildEmbed("en", res, "Bob", false)
	assert.Contains(t, embed.Footer.Text, "7/100")
	assert.NotContains(t, embed.Footer.Text, "!use", "foreign view footer must not show the use hint")
}
