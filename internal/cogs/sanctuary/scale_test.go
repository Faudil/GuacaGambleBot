package sanctuary_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"

	sanctuarycog "guacagamblebot/internal/cogs/sanctuary"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

// bodyRT records the body of the last HTTP request the router sent to
// Discord, so the response payload can be asserted offline (see the
// equivalent harness in internal/cogs/farm/farm_test.go).
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

type selectMenu struct {
	CustomID string `json:"custom_id"`
	Options  []any  `json:"options"`
	Type     int    `json:"type"`
}

type actionRow struct {
	Components []json.RawMessage `json:"components"`
}

type respPayload struct {
	Data struct {
		Components []actionRow `json:"components"`
	} `json:"data"`
	Components []actionRow `json:"components"`
}

func (r *bodyRT) payload(t *testing.T) respPayload {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	var out respPayload
	require.NoError(t, json.Unmarshal(r.body, &out))
	if len(out.Data.Components) == 0 && len(out.Components) > 0 {
		out.Data.Components = out.Components
	}
	return out
}

func newSanctuaryBot(t *testing.T) (*interaction.Router, *store.Store, *bodyRT) {
	require.NoError(t, i18n.Load("../../../locales"))
	d := testutil.NewDB(t)
	cfg := &config.Config{StartingBalance: 100}
	st := store.New(d, cfg)

	rt := &bodyRT{}
	s, err := discordgo.New("test")
	require.NoError(t, err)
	s.Client = &http.Client{Transport: rt}

	bot := &interaction.Bot{Session: s, DB: d, Prefix: "!"}
	r := interaction.NewRouter(bot, st)
	sanctuarycog.Register(r, st, cfg)
	require.NoError(t, st.DB.Create(&model.ServerSetting{ServerID: 100, Language: "en", Enabled: true}).Error)
	return r, st, rt
}

func seedPets(t *testing.T, st *store.Store, userID int64, count int, inSanctuary bool) {
	t.Helper()
	for i := 0; i < count; i++ {
		require.NoError(t, st.DB.Create(&model.UserPet{
			UserID:      userID,
			PetType:     "Licorne",
			Nickname:    "Pet" + strconv.Itoa(i),
			Level:       1,
			InSanctuary: inSanctuary,
		}).Error)
	}
}

// seedNamedPet inserts a single pet with a specific nickname and species, for
// tests asserting search behaviour.
func seedNamedPet(t *testing.T, st *store.Store, userID int64, nickname, species string, inSanctuary bool) int64 {
	t.Helper()
	pet := model.UserPet{
		UserID:      userID,
		PetType:     species,
		Nickname:    nickname,
		Level:       1,
		InSanctuary: inSanctuary,
	}
	require.NoError(t, st.DB.Create(&pet).Error)
	return pet.ID
}

func dispatch(r *interaction.Router, userID int64, customID string, values ...string) {
	data := discordgo.MessageComponentInteractionData{CustomID: customID}
	if len(values) > 0 {
		data.Values = values
	}
	r.DispatchInteraction(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionMessageComponent,
		GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: strconv.FormatInt(userID, 10)}},
		Data:   data,
	}})
}

// A sanctuary holding a hundred pets must never make the bot send Discord an
// invalid payload: select menus are hard-capped at 25 options and action
// rows at 5 buttons, or Discord rejects the whole interaction response and
// the user sees nothing at all.
func TestSanctuaryHandlesOneHundredPets(t *testing.T) {
	userID := int64(500)
	r, st, rt := newSanctuaryBot(t)

	require.NoError(t, st.DB.Create(&model.UserSanctuary{UserID: userID, Tier: 10}).Error)
	require.NoError(t, st.DB.Create(&model.User{UserID: userID, Balance: 1000}).Error)
	seedPets(t, st, userID, 100, true)

	// Opening the main panel must not panic or blow past Discord's 5-button row limit.
	r.DispatchInteraction(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionApplicationCommand, GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: strconv.FormatInt(userID, 10)}},
		Data:   discordgo.ApplicationCommandInteractionData{Name: "sanctuary"},
	}})
	panel := rt.payload(t)
	for _, row := range panel.Data.Components {
		require.LessOrEqual(t, len(row.Components), 5, "action row must not exceed Discord's 5-button limit")
	}

	// Opening the Recall picker with 100 sanctuary pets must cap at 25 options
	// per page, with a nav row (Prev / Search / Next) alongside the menu.
	time.Sleep(600 * time.Millisecond) // clear the router's per-user rate limit
	dispatch(r, userID, components.EncodeOwner(userID, "sanctuary", "recall"))
	picker := rt.payload(t)
	require.Len(t, picker.Data.Components, 2, "expected a select-menu row plus a nav row")
	var sel selectMenu
	require.NoError(t, json.Unmarshal(picker.Data.Components[0].Components[0], &sel))
	require.LessOrEqual(t, len(sel.Options), 25, "select menu must not exceed Discord's 25-option limit")
	require.NotEmpty(t, sel.Options)
	require.Len(t, picker.Data.Components[1].Components, 3, "nav row must have Prev/Search/Next")

	// Picking a pet from that menu must actually recall it, not just redraw the same menu.
	var pet model.UserPet
	require.NoError(t, st.DB.Where("user_id = ? AND in_sanctuary = ?", userID, true).First(&pet).Error)
	time.Sleep(600 * time.Millisecond) // clear the router's per-user rate limit
	dispatch(r, userID, components.EncodeOwner(userID, "sanctuary", "recall_pick"), strconv.FormatInt(pet.ID, 10))

	var refreshed model.UserPet
	require.NoError(t, st.DB.First(&refreshed, pet.ID).Error)
	require.False(t, refreshed.InSanctuary, "picking a pet from the recall menu must actually recall it")
}

type navBtn struct {
	Label    string `json:"label"`
	Disabled bool   `json:"disabled"`
	CustomID string `json:"custom_id"`
}

// A large sanctuary must let a player page through pets beyond the first 25,
// and search must be able to find a specific pet by name or species from
// anywhere in the list without paging to it manually.
func TestSanctuaryPickerPaginationAndSearch(t *testing.T) {
	userID := int64(600)
	r, st, rt := newSanctuaryBot(t)

	require.NoError(t, st.DB.Create(&model.UserSanctuary{UserID: userID, Tier: 10}).Error)
	seedPets(t, st, userID, 30, true) // page 0: 25 pets, page 1: 5 pets
	needleID := seedNamedPet(t, st, userID, "Zelda", "Renard", true)

	// Page 0: Prev disabled, Next enabled, full page of 25.
	dispatch(r, userID, components.EncodeOwner(userID, "sanctuary", "recall"))
	p0 := rt.payload(t)
	var sel0 selectMenu
	require.NoError(t, json.Unmarshal(p0.Data.Components[0].Components[0], &sel0))
	require.Len(t, sel0.Options, 25)
	var prev0, next0 navBtn
	require.NoError(t, json.Unmarshal(p0.Data.Components[1].Components[0], &prev0))
	require.NoError(t, json.Unmarshal(p0.Data.Components[1].Components[2], &next0))
	require.True(t, prev0.Disabled, "Prev must be disabled on the first page")
	require.False(t, next0.Disabled, "Next must be enabled when more pages remain")

	// Follow the Next button to page 1: the remaining 6 pets (5 seeded + Zelda).
	time.Sleep(600 * time.Millisecond)
	dispatch(r, userID, next0.CustomID)
	p1 := rt.payload(t)
	var sel1 selectMenu
	require.NoError(t, json.Unmarshal(p1.Data.Components[0].Components[0], &sel1))
	require.Len(t, sel1.Options, 6)
	var prev1, next1 navBtn
	require.NoError(t, json.Unmarshal(p1.Data.Components[1].Components[0], &prev1))
	require.NoError(t, json.Unmarshal(p1.Data.Components[1].Components[2], &next1))
	require.False(t, prev1.Disabled, "Prev must be enabled once past the first page")
	require.True(t, next1.Disabled, "Next must be disabled on the last page")

	// Search for "Zelda" (by name) must surface her regardless of which page she'd
	// otherwise land on, without needing to page there manually.
	time.Sleep(600 * time.Millisecond)
	r.DispatchInteraction(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionModalSubmit,
		GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: strconv.FormatInt(userID, 10)}},
		Data: discordgo.ModalSubmitInteractionData{
			CustomID: components.EncodeOwner(userID, "sanctuary", "pet_search_submit", "recall"),
			Components: []discordgo.MessageComponent{
				&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.TextInput{CustomID: "query", Value: "zelda"},
				}},
			},
		},
	}})
	searchResult := rt.payload(t)
	var selSearch selectMenu
	require.NoError(t, json.Unmarshal(searchResult.Data.Components[0].Components[0], &selSearch))
	require.Len(t, selSearch.Options, 1, "searching by name must narrow to the matching pet")

	// Search by species ("renard") must find the same pet.
	time.Sleep(600 * time.Millisecond)
	r.DispatchInteraction(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionModalSubmit,
		GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: strconv.FormatInt(userID, 10)}},
		Data: discordgo.ModalSubmitInteractionData{
			CustomID: components.EncodeOwner(userID, "sanctuary", "pet_search_submit", "recall"),
			Components: []discordgo.MessageComponent{
				&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.TextInput{CustomID: "query", Value: "renard"},
				}},
			},
		},
	}})
	speciesResult := rt.payload(t)
	var selSpecies selectMenu
	require.NoError(t, json.Unmarshal(speciesResult.Data.Components[0].Components[0], &selSpecies))
	require.Len(t, selSpecies.Options, 1, "searching by species must narrow to matching pets")
	require.Equal(t, strconv.FormatInt(needleID, 10), selSpecies.Options[0].(map[string]any)["value"])
}
