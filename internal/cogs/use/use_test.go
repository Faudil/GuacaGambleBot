package use

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/i18n"
	invsvc "guacagamblebot/internal/service/inventory"
	usesvc "guacagamblebot/internal/service/use"
	"guacagamblebot/internal/store"
)

func TestMain(m *testing.M) {
	_ = i18n.Load("../../../locales")
	os.Exit(m.Run())
}

func testCog(t *testing.T) *Cog {
	t.Helper()
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "use_cog.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	s := store.New(d, cfg)
	return &Cog{store: s, cfg: cfg, svc: usesvc.New(s, cfg), inv: invsvc.New(s, cfg)}
}

func TestUsableMenuOnlyShowsUsableItems(t *testing.T) {
	c := testCog(t)
	require.NoError(t, c.store.AddItemRaw(c.store.DB, 1, "beer", 2))
	require.NoError(t, c.store.AddItemRaw(c.store.DB, 1, "coal", 5))
	require.NoError(t, c.store.AddItemRaw(c.store.DB, 1, "scratch_ticket", 1))

	embed, comps, err := c.usableMenu("en", 1)
	require.NoError(t, err)
	assert.Equal(t, "🧰 Item Used", embed.Title)

	require.Len(t, comps, 1)
	row, ok := comps[0].(discordgo.ActionsRow)
	require.True(t, ok)
	require.Len(t, row.Components, 1)
	sel, ok := row.Components[0].(discordgo.SelectMenu)
	require.True(t, ok)

	assert.Len(t, sel.Options, 2, "only usable items must appear")
	byValue := make(map[string]discordgo.SelectMenuOption, len(sel.Options))
	for _, opt := range sel.Options {
		byValue[opt.Value] = opt
	}

	beer, ok := byValue["beer"]
	require.True(t, ok, "beer must be in the menu")
	assert.Contains(t, beer.Label, "x2")
	assert.Equal(t, "🍺", beer.Emoji.Name)
	assert.NotContains(t, byValue, "coal", "non-usable items must be excluded")
}

func TestUsableMenuEmptyInventory(t *testing.T) {
	c := testCog(t)
	require.NoError(t, c.store.AddItemRaw(c.store.DB, 1, "coal", 5))

	_, _, err := c.usableMenu("en", 1)
	assert.ErrorIs(t, err, errNoUsableItems)
}

func TestUsableMenuIgnoresEquippedItems(t *testing.T) {
	c := testCog(t)
	require.NoError(t, c.store.AddItemRaw(c.store.DB, 1, "beer", 1))
	_, err := c.store.CreateEquipment(1, "diamond_sword", "mighty blade", "⚔️", "rare", "weapon", 1, 0, 2, 0, 0, 0, nil, "")
	require.NoError(t, err)

	_, comps, err := c.usableMenu("en", 1)
	require.NoError(t, err)
	row, ok := comps[0].(discordgo.ActionsRow)
	require.True(t, ok)
	sel, ok := row.Components[0].(discordgo.SelectMenu)
	require.True(t, ok)
	assert.Len(t, sel.Options, 1)
	assert.Equal(t, "beer", sel.Options[0].Value)
}
