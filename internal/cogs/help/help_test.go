package help

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

func testCog(t *testing.T) *Cog {
	t.Helper()
	require.NoError(t, i18n.Load(filepath.Join("..", "..", "..", "locales")))
	cfg := &config.Config{Prefix: "!"}
	st := store.New(testutil.NewDB(t), cfg)
	r := interaction.NewRouter(&interaction.Bot{Session: &discordgo.Session{}, Prefix: "!"}, st)
	r.Categorize(CatCasino, func() {
		r.Slash("casino", "cmd.casino.desc", func(b *interaction.Bot, i *discordgo.InteractionCreate) {})
		r.Slash("cas", "cmd.casino.desc", func(b *interaction.Bot, i *discordgo.InteractionCreate) {})
		r.Prefix("casino", func(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {})
	})
	return &Cog{store: st, cfg: cfg, router: r}
}

func TestOverviewListsOnlyPopulatedCategories(t *testing.T) {
	c := testCog(t)
	embed, comps := c.overview("en")

	require.Len(t, embed.Fields, 1, "only the category with commands should appear")
	assert.Contains(t, embed.Fields[0].Name, "Casino")
	assert.Contains(t, embed.Fields[0].Value, "`/casino`")
	assert.NotContains(t, embed.Fields[0].Value, "`/cas`", "aliases stay folded into their command")
	assert.Contains(t, embed.Footer.Text, "1 command")

	row, ok := comps[0].(discordgo.ActionsRow)
	require.True(t, ok)
	menu, ok := row.Components[0].(discordgo.SelectMenu)
	require.True(t, ok)
	assert.Len(t, menu.Options, 1, "empty categories are not offered in the picker")
}

func TestCategoryViewShowsAliasesPrefixAndDescription(t *testing.T) {
	c := testCog(t)
	embed, _ := c.categoryView("en", CatCasino)

	assert.Contains(t, embed.Description, "`/casino`")
	assert.Contains(t, embed.Description, "`!casino`", "prefix form is shown when one exists")
	assert.Contains(t, embed.Description, "(cas)", "aliases are listed")
	assert.Contains(t, embed.Description, i18n.T("cmd.casino.desc", "en"))
}

func TestCategoryViewIsLocalised(t *testing.T) {
	c := testCog(t)
	embed, _ := c.categoryView("fr", CatCasino)
	assert.Contains(t, embed.Description, i18n.T("cmd.casino.desc", "fr"))
	assert.NotContains(t, embed.Description, i18n.T("cmd.casino.desc", "en"))
}

// An unknown or stale category id must not render a blank screen.
func TestUnknownCategoryFallsBackToOverview(t *testing.T) {
	c := testCog(t)
	embed, _ := c.categoryView("en", "does-not-exist")
	assert.Equal(t, i18n.T("help.title", "en"), embed.Title)
}

func TestEveryDeclaredCategoryHasALocalisedName(t *testing.T) {
	require.NoError(t, i18n.Load(filepath.Join("..", "..", "..", "locales")))
	for _, cat := range categories {
		for _, lang := range i18n.Languages() {
			key := "help.cat." + cat.Key
			got := i18n.T(key, lang)
			assert.NotEqual(t, key, got, "category %q has no %s name", cat.Key, lang)
			assert.False(t, strings.TrimSpace(got) == "", "category %q has an empty %s name", cat.Key, lang)
		}
	}
}
