package farm_test

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	farmcog "guacagamblebot/internal/cogs/farm"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
	"guacagamblebot/internal/universe/hoakhaven"
)

// bodyRT is a mock HTTP transport that records the body of the last request so
// the interaction response payload can be asserted offline. The last request is
// the handler's real reply: the router first acknowledges with a deferred
// response, then the handler's response is translated into an edit/follow-up.
type bodyRT struct {
	mu   sync.Mutex
	body []byte
}

func (r *bodyRT) RoundTrip(req *http.Request) (*http.Response, error) {
	b, _ := io.ReadAll(req.Body)
	r.mu.Lock()
	r.body = b
	r.mu.Unlock()
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"id":"1"}`)),
		Header:     make(http.Header),
	}, nil
}

func (r *bodyRT) payload(t *testing.T) resp {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	var out resp
	require.NoError(t, json.Unmarshal(r.body, &out))
	// Post-deferral edits carry embeds/components at the top level (WebhookEdit)
	// instead of under "data" (InteractionResponseData); merge them so callers
	// can assert on either shape.
	if len(out.Data.Components) == 0 && len(out.Components) > 0 {
		out.Data.Components = out.Components
	}
	if len(out.Data.Embeds) == 0 && len(out.Embeds) > 0 {
		out.Data.Embeds = out.Embeds
	}
	return out
}

type btn struct {
	CustomID string `json:"custom_id"`
	Label    string `json:"label"`
	Style    int    `json:"style"`
	Disabled bool   `json:"disabled"`
}

type resp struct {
	Data struct {
		Embeds []struct {
			Description string `json:"description"`
		} `json:"embeds"`
		Components []struct {
			Components []btn `json:"components"`
		} `json:"components"`
	} `json:"data"`
	Embeds []struct {
		Description string `json:"description"`
	} `json:"embeds"`
	Components []struct {
		Components []btn `json:"components"`
	} `json:"components"`
}

func newFarmBot(t *testing.T) (*interaction.Router, *store.Store, *bodyRT) {
	require.NoError(t, i18n.Load("../../../locales"))
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "farm.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	st := store.New(d, cfg)
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	require.NotNil(t, def)

	rt := &bodyRT{}
	s, err := discordgo.New("test")
	require.NoError(t, err)
	s.Client = &http.Client{Transport: rt}

	bot := &interaction.Bot{Session: s, DB: d, Prefix: "!"}
	r := interaction.NewRouter(bot, st)
	farmcog.Register(r, st, cfg)
	require.NoError(t, st.DB.Create(&model.ServerSetting{ServerID: 100, Language: "en", Enabled: true}).Error)
	return r, st, rt
}

func plantGrowingCrop(t *testing.T, st *store.Store, userID int64, seed string, growTime int) {
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: userID, ItemID: seed, Quantity: 5}).Error)
	require.NoError(t, st.DB.Create(&model.UserFarming{
		UserID:    userID,
		ZoneKey:   "public",
		PlotIndex: 0,
		ItemName:  seed,
		PlantTime: time.Now(),
		GrowTime:  growTime,
	}).Error)
}

func inspectPlot(t *testing.T, r *interaction.Router, userID int64, customID string) {
	r.DispatchInteraction(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionMessageComponent,
		GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: userIDStr(userID)}},
		Data:   discordgo.MessageComponentInteractionData{CustomID: customID},
	}})
}

func userIDStr(id int64) string {
	return strconv.FormatInt(id, 10)
}

func TestInspectShowsFertilizeButtonWhenOwned(t *testing.T) {
	r, st, rt := newFarmBot(t)
	plantGrowingCrop(t, st, 200, "wheat_seed", 3600)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: 200, ItemID: "fertilizer", Quantity: 1}).Error)

	inspectPlot(t, r, 200, components.EncodeOwner(200, "farm", "inspect", "public", "0"))

	p := rt.payload(t)
	var fert *btn
	for _, row := range p.Data.Components {
		for _, b := range row.Components {
			if strings.HasPrefix(b.CustomID, "farm::fertilize") {
				fert = &b
			}
		}
	}
	require.NotNil(t, fert, "fertilize button must be present")
	assert.False(t, fert.Disabled, "fertilize button must be enabled when the user owns fertilizer")
	assert.Contains(t, fert.Label, "Fertilize")
	assert.Equal(t, 4, fert.Style, "fertilize button must use the danger style when enabled")
}

func TestInspectShowsLockedFertilizeButtonWithoutItem(t *testing.T) {
	r, st, rt := newFarmBot(t)
	plantGrowingCrop(t, st, 200, "wheat_seed", 3600)

	inspectPlot(t, r, 200, components.EncodeOwner(200, "farm", "inspect", "public", "0"))

	p := rt.payload(t)
	var fert *btn
	for _, row := range p.Data.Components {
		for _, b := range row.Components {
			if strings.HasPrefix(b.CustomID, "farm::fertilize") {
				fert = &b
			}
		}
	}
	require.NotNil(t, fert, "fertilize button must be present even without the item")
	assert.True(t, fert.Disabled, "fertilize button must be disabled when the user lacks fertilizer")
	assert.Contains(t, fert.Label, "need Fertilizer")
	assert.Equal(t, 2, fert.Style, "locked fertilize button must use the secondary style")
	assert.Contains(t, p.Data.Embeds[0].Description, "Fertilizer", "inspect view must hint how to obtain fertilizer")
}

func TestInspectNoFertilizeButtonWhenReady(t *testing.T) {
	r, st, rt := newFarmBot(t)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: 200, ItemID: "wheat_seed", Quantity: 5}).Error)
	require.NoError(t, st.DB.Create(&model.UserFarming{
		UserID:    200,
		ZoneKey:   "public",
		PlotIndex: 0,
		ItemName:  "wheat_seed",
		PlantTime: time.Now().Add(-2 * time.Second),
		GrowTime:  1,
	}).Error)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: 200, ItemID: "fertilizer", Quantity: 1}).Error)

	inspectPlot(t, r, 200, components.EncodeOwner(200, "farm", "inspect", "public", "0"))

	p := rt.payload(t)
	for _, row := range p.Data.Components {
		for _, b := range row.Components {
			assert.NotContains(t, b.CustomID, "farm::fertilize", "ready plots must not offer fertilize")
		}
	}
}
