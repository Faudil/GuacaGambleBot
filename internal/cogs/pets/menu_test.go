package pets_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/require"

	petscog "guacagamblebot/internal/cogs/pets"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	petsvc "guacagamblebot/internal/service/pets"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

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

// A pet species whose emoji Discord will not accept must never cost the player
// their whole /pet menu: Discord rejects the entire response with 400 Invalid
// Form Body, which the cogs discard, so the menu just never arrives. Regression
// test for the "🐺🐺🐺" Cerbère that broke /pet for its only owner.
func TestPetMenuSurvivesIllegalSpeciesEmoji(t *testing.T) {
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
	petscog.Register(r, st, cfg)
	require.NoError(t, st.DB.Create(&model.ServerSetting{ServerID: 100, Language: "fr", Enabled: true}).Error)

	// Re-arm the exact defect independently of the content table, so this keeps
	// testing the guard even after the species emoji were corrected.
	cerb := petsvc.PetTypes["Cerbère"]
	require.NotNil(t, cerb)
	restore := cerb.Emoji
	cerb.Emoji = "🐺🐺🐺"
	t.Cleanup(func() { cerb.Emoji = restore })

	uid := int64(171078222522482690)
	require.NoError(t, st.DB.Create(&model.User{UserID: uid, Balance: 1000}).Error)
	// "Cerbère" is the species that carried the illegal emoji.
	for _, n := range []string{"Cerbère", "Winnie"} {
		require.NoError(t, st.DB.Create(&model.UserPet{
			UserID: uid, PetType: "Cerbère", Nickname: n,
			Level: 1, HP: 50, MaxHP: 50,
		}).Error)
	}

	r.DispatchInteraction(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionApplicationCommand, GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: strconv.FormatInt(uid, 10)}},
		Data:   discordgo.ApplicationCommandInteractionData{Name: "pet"},
	}})

	rt.mu.Lock()
	raw := append([]byte(nil), rt.body...)
	rt.mu.Unlock()
	require.NotEmpty(t, raw, "the menu must actually be sent")

	var payload struct {
		Data struct {
			Components []struct {
				Components []struct {
					Options []struct {
						Label string `json:"label"`
						Emoji *struct {
							Name string `json:"name"`
							ID   string `json:"id"`
						} `json:"emoji"`
					} `json:"options"`
				} `json:"components"`
			} `json:"components"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &payload))

	seen := 0
	for _, row := range payload.Data.Components {
		for _, c := range row.Components {
			for _, o := range c.Options {
				seen++
				if o.Emoji != nil && o.Emoji.ID == "" {
					require.True(t, isOneEmoji(o.Emoji.Name),
						"option %q would make Discord reject the whole response: emoji %q", o.Label, o.Emoji.Name)
				}
			}
		}
	}
	require.Equal(t, 2, seen, "both pets must be listed")
}

func isOneEmoji(s string) bool {
	if s == "" {
		return false
	}
	base := 0
	for _, r := range s {
		switch {
		case r == 0x200D:
			return true
		case r == 0xFE0F || r == 0xFE0E:
		case r >= 0x1F3FB && r <= 0x1F3FF:
		default:
			base++
		}
	}
	return base == 1
}
