package achievements

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	achievementsvc "guacagamblebot/internal/service/achievements"
	"guacagamblebot/internal/store"
)

func TestMain(m *testing.M) {
	if err := i18n.Load("../../../locales"); err != nil {
		fmt.Println("could not load locales:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func testStore(t *testing.T) *store.Store {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	return store.New(d, &config.Config{StartingBalance: 100, DailyAmount: 50})
}

func testCog(t *testing.T) *Cog {
	st := testStore(t)
	return &Cog{store: st, cfg: &config.Config{}, svc: achievementsvc.New(st, &config.Config{})}
}

// realViews mirrors the production payload: every registered achievement, with
// its real emoji, glory and localized name.
func realViews(unlocked bool) []achievementsvc.View {
	all := achievement.All()
	views := make([]achievementsvc.View, 0, len(all))
	for _, a := range all {
		views = append(views, achievementsvc.View{
			ID:       a.ID,
			Emoji:    a.Emoji,
			Glory:    a.Glory,
			Unlocked: unlocked,
		})
	}
	return views
}

func TestListViewStaysWithinEmbedLimit(t *testing.T) {
	c := testCog(t)
	views := realViews(false)
	require.Len(t, views, 100, "fixture must reproduce the full achievements registry")

	for _, lang := range []string{"en", "fr"} {
		for page := 1; page <= 4; page++ {
			embed, _ := c.listView(lang, 42, views, page)
			require.LessOrEqual(t, len(embed.Description), 4096,
				"%s: page %d exceeds Discord's embed description limit", lang, page)
			require.NotEmpty(t, embed.Description)
		}
	}
}

func TestListViewPagination(t *testing.T) {
	c := testCog(t)
	views := realViews(false)

	embed1, comps1 := c.listView("en", 42, views, 1)
	lines1 := strings.Split(embed1.Description, "\n")
	assert.Len(t, lines1, pageSize, "page 1 must hold a full page")
	assert.Equal(t, "📄 1/4", navPageLabel(t, comps1))

	embed4, comps4 := c.listView("en", 42, views, 4)
	lines4 := strings.Split(embed4.Description, "\n")
	assert.Len(t, lines4, len(views)-3*pageSize, "page 4 must hold the remaining achievements")
	assert.Equal(t, "📄 4/4", navPageLabel(t, comps4))

	// Out-of-range pages are clamped to the nearest valid page.
	clampedLow, _ := c.listView("en", 42, views, 0)
	assert.Equal(t, embed1.Description, clampedLow.Description, "page 0 must clamp to page 1")
	clampedHigh, _ := c.listView("en", 42, views, 99)
	assert.Equal(t, embed4.Description, clampedHigh.Description, "page 99 must clamp to the last page")
}

func TestListViewEmpty(t *testing.T) {
	c := testCog(t)
	embed, comps := c.listView("en", 42, nil, 1)
	assert.Equal(t, "You haven't unlocked any achievements yet.", embed.Description)
	nav := navRow(t, comps)
	require.Len(t, nav.Components, 3)
	assert.True(t, nav.Components[0].(discordgo.Button).Disabled, "prev must be disabled on page 1")
	assert.True(t, nav.Components[2].(discordgo.Button).Disabled, "next must be disabled on a single page")
}

func TestListComponentsOwnerGated(t *testing.T) {
	c := testCog(t)
	comps := c.listComponents("en", 42, 95, 2)

	prev := navRow(t, comps).Components[0].(discordgo.Button)
	next := navRow(t, comps).Components[2].(discordgo.Button)
	assert.False(t, prev.Disabled, "prev must be enabled past page 1")
	assert.False(t, next.Disabled, "next must be enabled before the last page")
	assert.Equal(t, "achievements::nav::prev::2::42", prev.CustomID)
	assert.Equal(t, "achievements::nav::next::2::42", next.CustomID)

	last := c.listComponents("en", 42, 95, 4)
	lastNext := navRow(t, last).Components[2].(discordgo.Button)
	assert.True(t, lastNext.Disabled, "next must be disabled on the last page")

	// The show button resets to page one.
	showRow := comps[0].(discordgo.ActionsRow)
	showBtn := showRow.Components[0].(discordgo.Button)
	assert.Equal(t, "achievements::show::42", showBtn.CustomID)
}

// navRow extracts the second action row (prev/page/next) from a component list.
func navRow(t *testing.T, comps []discordgo.MessageComponent) discordgo.ActionsRow {
	t.Helper()
	require.Len(t, comps, 2)
	row, ok := comps[1].(discordgo.ActionsRow)
	require.True(t, ok, "second row must be the nav row")
	return row
}

func navPageLabel(t *testing.T, comps []discordgo.MessageComponent) string {
	t.Helper()
	pageBtn := navRow(t, comps).Components[1].(discordgo.Button)
	return pageBtn.Label
}

// mockRT records every request like the interaction package's deferRT.
type mockRT struct {
	calls  []string
	bodies []string
}

func (r *mockRT) RoundTrip(req *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	r.calls = append(r.calls, req.Method+" "+req.URL.Path)
	r.bodies = append(r.bodies, string(body))
	return &http.Response{
		StatusCode: 200,
		Body:       http.NoBody,
		Header:     make(http.Header),
	}, nil
}

func componentInteraction(customID, userID string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:    "1",
		Token: "tok",
		Type:  discordgo.InteractionMessageComponent,
		Member: &discordgo.Member{
			User: &discordgo.User{ID: userID},
		},
		Data: discordgo.MessageComponentInteractionData{CustomID: customID},
	}}
}

func TestShowAndNavDispatch(t *testing.T) {
	c := testCog(t)
	st := c.store

	raw, err := discordgo.New("test")
	require.NoError(t, err)
	rt := &mockRT{}
	raw.Client = &http.Client{Transport: rt}

	r := interaction.NewRouter(&interaction.Bot{Session: raw, Prefix: "!"}, st)
	Register(r, st, &config.Config{})

	// Opening the list (Show) renders page 1.
	r.DispatchInteraction(componentInteraction("achievements::show::42", "42"))
	require.Len(t, rt.calls, 1, "a fast handler must answer directly")
	body := rt.bodies[0]
	var resp struct {
		Type int `json:"type"`
		Data struct {
			Content string `json:"content"`
			Embeds  []struct {
				Description string `json:"description"`
			} `json:"embeds"`
			Components []json.RawMessage `json:"components"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &resp))
	assert.Equal(t, int(discordgo.InteractionResponseUpdateMessage), resp.Type)
	require.Len(t, resp.Data.Embeds, 1)
	require.LessOrEqual(t, len(resp.Data.Embeds[0].Description), 4096,
		"the dispatched Show response must fit Discord's embed limit")

	// Browsing next lands on page 2 (nav buttons carry the page + owner).
	// A fresh user id keeps each dispatch outside the 500ms rate limiter.
	r.DispatchInteraction(componentInteraction("achievements::nav::next::1::43", "43"))
	require.Len(t, rt.calls, 2)
	require.NoError(t, json.Unmarshal([]byte(rt.bodies[1]), &resp))
	require.Len(t, resp.Data.Embeds, 1)
	require.Contains(t, string(rt.bodies[1]), "achievements::nav::prev::2::43")
	require.Contains(t, string(rt.bodies[1]), "achievements::nav::next::2::43")

	// Another user clicking the owner's buttons is rejected.
	r.DispatchInteraction(componentInteraction("achievements::nav::next::2::44", "45"))
	require.Len(t, rt.calls, 3)
	require.NoError(t, json.Unmarshal([]byte(rt.bodies[2]), &resp))
	require.Contains(t, resp.Data.Content, "<@44>",
		"the clicker must be told the menu belongs to its owner")
}

func TestServiceSortsByGlory(t *testing.T) {
	st := testStore(t)
	require.NoError(t, st.DB.Create(&model.UserAchievement{UserID: 42, AchievementID: "daily_1"}).Error)

	svc := achievementsvc.New(st, &config.Config{})
	views, err := svc.List(42)
	require.NoError(t, err)
	require.Greater(t, len(views), 1)

	for i := 1; i < len(views); i++ {
		prev, cur := views[i-1], views[i]
		require.Falsef(t, cur.Glory > prev.Glory,
			"glory must be descending at %d (%s %d before %s %d)", i, cur.ID, cur.Glory, prev.ID, prev.Glory)
		require.Falsef(t, cur.Glory == prev.Glory && cur.ID <= prev.ID,
			"ids must be ascending for equal glory at %d (%q before %q)", i, cur.ID, prev.ID)
	}
}
