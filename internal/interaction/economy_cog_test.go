package interaction_test

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

	"guacagamblebot/internal/cogs/economy"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

// recordRT is a mock HTTP transport that records every request and answers 200
// so discordgo's InteractionRespond / FollowupMessageCreate succeed offline.
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

func newEconomyBot(t *testing.T) (*interaction.Router, *interaction.Bot, *store.Store, *recordRT) {
	require.NoError(t, i18n.Load("../../locales"))
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cog.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	st := store.New(d, cfg)

	rt := &recordRT{}
	s, err := discordgo.New("test")
	require.NoError(t, err)
	s.Client = &http.Client{Transport: rt}

	bot := &interaction.Bot{Session: s, DB: d, Prefix: "!"}
	r := interaction.NewRouter(bot, st)
	economy.Register(r, st, cfg)
	return r, bot, st, rt
}

func setBalance(t *testing.T, st *store.Store, id int64, bal int) {
	require.NoError(t, st.DB.Create(&model.User{UserID: id}).Error)
	require.NoError(t, st.DB.Model(&model.User{}).Where("user_id = ?", id).Update("balance", bal).Error)
}

func TestEconomySlashOpensMenu(t *testing.T) {
	r, _, _, rt := newEconomyBot(t)
	r.DispatchInteraction(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionApplicationCommand,
		GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: "200"}},
		Data:   discordgo.ApplicationCommandInteractionData{Name: "economy"},
	}})
	assert.Equal(t, 1, rt.callbackCount(), "slash handler should respond with the menu")
}

func TestEconomyBalanceButton(t *testing.T) {
	r, _, _, rt := newEconomyBot(t)
	r.DispatchInteraction(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionMessageComponent,
		GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: "200"}},
		Data:   discordgo.MessageComponentInteractionData{CustomID: components.EncodeOwner(200, "economy", "balance")},
	}})
	assert.Equal(t, 1, rt.callbackCount(), "balance button should respond")
}

func TestEconomyDailyButton(t *testing.T) {
	r, _, st, rt := newEconomyBot(t)
	setBalance(t, st, 200, 1000)
	r.DispatchInteraction(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionMessageComponent,
		GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: "200"}},
		Data:   discordgo.MessageComponentInteractionData{CustomID: components.EncodeOwner(200, "economy", "daily")},
	}})
	assert.Equal(t, 1, rt.callbackCount(), "daily button should respond")
	bal, err := st.GetBalance(200)
	require.NoError(t, err)
	assert.Equal(t, 1050, bal, "daily should credit the salary")
}

func TestEconomyGiveModal(t *testing.T) {
	r, _, st, rt := newEconomyBot(t)
	setBalance(t, st, 200, 1000)
	setBalance(t, st, 300, 0)
	b0, _ := st.GetBalance(200)
	t.Logf("sender before give = %d", b0)
	gi := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionModalSubmit,
		GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: "200"}},
		Data: discordgo.ModalSubmitInteractionData{
			CustomID: components.Encode("economy", "give_submit"),
			Components: []discordgo.MessageComponent{
				&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.TextInput{CustomID: "recipient", Value: "<@300>"},
					&discordgo.TextInput{CustomID: "amount", Value: "50"},
				}},
			},
		},
	}}
	t.Logf("modal values = %v", interaction.ModalValues(gi))
	r.DispatchInteraction(gi)
	assert.Equal(t, 1, rt.callbackCount(), "give modal should respond")
	sb, err := st.GetBalance(200)
	require.NoError(t, err)
	rb, err := st.GetBalance(300)
	require.NoError(t, err)
	assert.Equal(t, 950, sb)
	assert.Equal(t, 50, rb)
}
