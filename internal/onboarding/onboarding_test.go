package onboarding

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
	"guacagamblebot/internal/store"
)

// recordRT is a mock HTTP transport that answers 200 so discordgo's
// InteractionRespond succeeds offline.
type recordRT struct {
	mu    sync.Mutex
	calls []string
}

func (r *recordRT) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.calls = append(r.calls, req.Method+" "+req.URL.Path)
	r.mu.Unlock()
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"id":"1"}`)),
		Header:     make(http.Header),
	}, nil
}

func (r *recordRT) callbackCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, c := range r.calls {
		if strings.Contains(c, "/callback") {
			n++
		}
	}
	return n
}

func newTestCog(t *testing.T) (*Cog, *store.Store, *interaction.Router, *recordRT) {
	require.NoError(t, i18n.Load("../../locales"))
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "onboarding.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{Prefix: "!"}
	st := store.New(d, cfg)

	rt := &recordRT{}
	s, err := discordgo.New("test")
	require.NoError(t, err)
	s.Client = &http.Client{Transport: rt}

	bot := &interaction.Bot{Session: s, DB: d, Prefix: "!"}
	r := interaction.NewRouter(bot, st)
	c := &Cog{store: st, cfg: cfg}
	Register(r, st, cfg)
	return c, st, r, rt
}

func TestMenuContainsExpectedComponents(t *testing.T) {
	c, _, _, _ := newTestCog(t)
	embed, comps := c.menu("en", nil)
	assert.Equal(t, "Welcome to GuacaGambleBot! 🎰", embed.Title)
	require.Len(t, comps, 4)

	row0 := comps[0].(discordgo.ActionsRow)
	chSel := row0.Components[0].(discordgo.SelectMenu)
	assert.Equal(t, discordgo.ChannelSelectMenu, chSel.MenuType)
	assert.Equal(t, components.Encode("onboarding", "channel"), chSel.CustomID)

	row1 := comps[1].(discordgo.ActionsRow)
	uniSel := row1.Components[0].(discordgo.SelectMenu)
	assert.Equal(t, components.Encode("onboarding", "universe"), uniSel.CustomID)

	row2 := comps[2].(discordgo.ActionsRow)
	langSel := row2.Components[0].(discordgo.SelectMenu)
	assert.Len(t, langSel.Options, 2)

	row3 := comps[3].(discordgo.ActionsRow)
	finishBtn := row3.Components[2].(discordgo.Button)
	assert.Equal(t, components.Encode("onboarding", "finish"), finishBtn.CustomID)
}

func TestSetupSlashOpensMenu(t *testing.T) {
	_, _, r, rt := newTestCog(t)
	r.DispatchInteraction(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: "200"}},
		Data:   discordgo.ApplicationCommandInteractionData{Name: "setup"},
	}})
	assert.Equal(t, 1, rt.callbackCount(), "setup slash should respond with the menu")
}

func TestChannelSelectPersistsChannel(t *testing.T) {
	_, st, r, rt := newTestCog(t)
	r.DispatchInteraction(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionMessageComponent,
		GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: "200"}},
		Data: discordgo.MessageComponentInteractionData{
			CustomID: components.Encode("onboarding", "channel"),
			Values:   []string{"123"},
		},
	}})
	assert.Equal(t, 1, rt.callbackCount())
	ss, err := st.GetServerSetting(100)
	require.NoError(t, err)
	require.NotNil(t, ss)
	assert.Equal(t, int64(123), ss.ChannelID)
}

func TestFinishRequiresChannel(t *testing.T) {
	_, _, r, rt := newTestCog(t)
	r.DispatchInteraction(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionMessageComponent,
		GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: "200"}},
		Data:   discordgo.MessageComponentInteractionData{CustomID: components.Encode("onboarding", "finish")},
	}})
	assert.Equal(t, 1, rt.callbackCount(), "finish without a channel must still respond (error)")
}
