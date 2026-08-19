package inventory

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	invsvc "guacagamblebot/internal/service/inventory"
	"guacagamblebot/internal/store"
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
		Entries: []invsvc.InvEntry{{ItemName: "Coal", Quantity: 3, Item: &items.Item{ID: "coal", Name: "Coal", Category: items.Mining}}},
		Current: 3,
		Limit:   100,
		UserID:  1,
	}
	themes := presentThemes(res.Entries)

	c.buildEmbed("en", res, "Bob", themes[0], true)
	comps := c.buildComponents("en", 1, 1, themes, themes[0], true)
	assert.Len(t, comps, 2, "owner view must have the theme select row and the nav row")

	c.buildEmbed("en", res, "Bob", themes[0], false)
	comps = c.buildComponents("en", 1, 2, themes, themes[0], false)
	require.Len(t, comps, 2, "foreign view still paginates")
	row, ok := comps[1].(discordgo.ActionsRow)
	require.True(t, ok)
	assert.Len(t, row.Components, 3, "foreign view nav row must not include the sell button")
}

func TestBuildComponentsNavRowHasSellButtonForOwner(t *testing.T) {
	require.NoError(t, i18n.Load("../../../locales"))
	c := &Cog{}
	res := &invsvc.InvResult{
		Entries: []invsvc.InvEntry{{ItemName: "Coal", Quantity: 3}},
		Current: 3,
		Limit:   100,
		UserID:  42,
	}
	themes := presentThemes(res.Entries)
	comps := c.buildComponents("en", 42, 42, themes, themes[0], true)
	require.Len(t, comps, 2)

	row, ok := comps[1].(discordgo.ActionsRow)
	require.True(t, ok)
	require.Len(t, row.Components, 4)

	var sellBtn discordgo.Button
	found := false
	for _, comp := range row.Components {
		if btn, ok := comp.(discordgo.Button); ok && btn.Label == "💰 Sell items" {
			sellBtn = btn
			found = true
		}
	}
	require.True(t, found, "nav row must include the sell button")
	ownerID, ok := components.OwnerID(sellBtn.CustomID)
	assert.True(t, ok, "sell button must carry an owner id")
	assert.Equal(t, int64(42), ownerID)
}

func TestBuildComponentsForeignViewHasNoSellButton(t *testing.T) {
	c := &Cog{}
	res := &invsvc.InvResult{
		Entries: []invsvc.InvEntry{{ItemName: "Coal", Quantity: 3}},
		Current: 3,
		Limit:   100,
		UserID:  42,
	}
	themes := presentThemes(res.Entries)
	comps := c.buildComponents("en", 7, 42, themes, themes[0], false)
	require.Len(t, comps, 2)

	row, ok := comps[1].(discordgo.ActionsRow)
	require.True(t, ok)
	assert.Len(t, row.Components, 3, "foreign view nav row must have prev/page/next only")
}

func TestSellButtonEncodesOwner(t *testing.T) {
	btn := discordgo.Button{Label: "💰 Sell items",
		CustomID: components.EncodeOwner(42, "inventory", "sell"), Style: discordgo.SecondaryButton}
	ownerID, ok := components.OwnerID(btn.CustomID)
	assert.True(t, ok, "sell button must carry an owner id")
	assert.Equal(t, int64(42), ownerID)
}

func TestBuildEmbedCountsFooter(t *testing.T) {
	c := &Cog{}
	res := &invsvc.InvResult{Current: 7, Limit: 100, UserID: 1}

	embed := c.buildEmbed("en", res, "Bob", "other", false)
	assert.Contains(t, embed.Footer.Text, "7/100")
	assert.NotContains(t, embed.Footer.Text, "!use", "foreign view footer must not show the use hint")
}

func TestPresentThemesKeepsDisplayOrder(t *testing.T) {
	res := &invsvc.InvResult{
		Entries: []invsvc.InvEntry{
			{ItemName: "Wheat", Quantity: 1, Item: &items.Item{ID: "wheat", Name: "Wheat", Category: "farming"}},
			{ItemName: "Coal", Quantity: 1, Item: &items.Item{ID: "coal", Name: "Coal", Category: "mining"}},
			{ItemName: "Mystery", Quantity: 1, Item: &items.Item{ID: "mystery", Name: "Mystery", Category: "unknown_cat"}},
			{ItemName: "Salmon", Quantity: 1, Item: &items.Item{ID: "salmon", Name: "Salmon", Category: "fishing"}},
			{ItemName: "Delve Blade", Quantity: 1, Item: &items.Item{ID: "blade", Name: "Delve Blade", Category: "delve"}},
		},
	}
	themes := presentThemes(res.Entries)
	assert.Equal(t, []string{"mining", "fishing", "farming", "equipment", "other"}, themes)
}

func TestBuildCategoryFieldsOnlyRendersSelectedTheme(t *testing.T) {
	res := &invsvc.InvResult{
		Entries: []invsvc.InvEntry{
			{ItemName: "Wheat", Quantity: 4, Item: &items.Item{ID: "wheat", Name: "Wheat", Category: "farming"}},
			{ItemName: "Coal", Quantity: 3, Item: &items.Item{ID: "coal", Name: "Coal", Category: "mining"}},
		},
	}

	fields := buildCategoryFields(res, "mining", "en")
	require.Len(t, fields, 1)
	assert.Equal(t, "⛏️ Mining", fields[0].Name)
	assert.Contains(t, fields[0].Value, "Coal")
	assert.NotContains(t, fields[0].Value, "Wheat")
}

func TestBuildComponentsThemeSelectCarriesTarget(t *testing.T) {
	res := &invsvc.InvResult{
		Entries: []invsvc.InvEntry{
			{ItemName: "Wheat", Quantity: 1, Item: &items.Item{ID: "wheat", Name: "Wheat", Category: "farming"}},
			{ItemName: "Coal", Quantity: 1, Item: &items.Item{ID: "coal", Name: "Coal", Category: "mining"}},
		},
	}
	themes := presentThemes(res.Entries)
	comps := (&Cog{}).buildComponents("en", 9, 42, themes, "mining", false)

	row, ok := comps[0].(discordgo.ActionsRow)
	require.True(t, ok)
	sel, ok := row.Components[0].(discordgo.SelectMenu)
	require.True(t, ok)

	ownerID, ok := components.OwnerID(sel.CustomID)
	assert.True(t, ok)
	assert.Equal(t, int64(9), ownerID)

	_, _, rest := components.Decode(sel.CustomID)
	assert.Equal(t, "42", rest[0], "select must carry the target user id")
	require.Len(t, sel.Options, 2)
	assert.True(t, sel.Options[0].Default, "current theme must be marked default")
}

type countRT struct {
	mu        sync.Mutex
	callbacks int
}

func (r *countRT) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.Path, "/callback") {
		r.mu.Lock()
		r.callbacks++
		r.mu.Unlock()
	}
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"id":"1"}`)),
		Header:     make(http.Header),
	}, nil
}

func newInventoryBot(t *testing.T) (*interaction.Router, *store.Store, *countRT) {
	require.NoError(t, i18n.Load("../../../locales"))
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "inv.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	st := store.New(d, cfg)
	require.NoError(t, st.DB.Create(&model.User{UserID: 200}).Error)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: 200, ItemID: "wheat", Quantity: 3}).Error)

	rt := &countRT{}
	s, err := discordgo.New("test")
	require.NoError(t, err)
	s.Client = &http.Client{Transport: rt}

	bot := &interaction.Bot{Session: s, DB: d, Prefix: "!"}
	r := interaction.NewRouter(bot, st)
	Register(r, st, cfg)
	return r, st, rt
}

func TestInventorySlashWithUserOptionDoesNotPanic(t *testing.T) {
	r, _, rt := newInventoryBot(t)
	r.DispatchInteraction(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: "200", Username: "tester"}},
		Data: discordgo.ApplicationCommandInteractionData{
			Name: "inventory",
			Options: []*discordgo.ApplicationCommandInteractionDataOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Value: "300"},
			},
			Resolved: &discordgo.ApplicationCommandInteractionDataResolved{
				Users: map[string]*discordgo.User{"300": {ID: "300", Username: "buddy"}},
			},
		},
	}})
	assert.Equal(t, 1, rt.callbacks, "slash inventory with a user option must respond without panicking")
}
