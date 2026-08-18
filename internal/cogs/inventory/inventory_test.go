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
