package interaction

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/components"
)

// deferRT is a mock HTTP transport that records the method/path and body of
// every request so the deferred-response pipeline can be asserted offline.
type deferRT struct {
	mu     sync.Mutex
	calls  []string
	bodies []string
}

func (r *deferRT) RoundTrip(req *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	r.mu.Lock()
	r.calls = append(r.calls, req.Method+" "+req.URL.Path)
	r.bodies = append(r.bodies, string(body))
	r.mu.Unlock()
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"id":"1"}`)),
		Header:     make(http.Header),
	}, nil
}

func (r *deferRT) snapshot() ([]string, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...), append([]string(nil), r.bodies...)
}

func newDeferTestSession(t *testing.T) (*DeferringSession, *deferRT) {
	t.Helper()
	s, err := discordgo.New("test")
	require.NoError(t, err)
	rt := &deferRT{}
	s.Client = &http.Client{Transport: rt}
	return NewDeferringSession(s), rt
}

func deferTestInteraction() *discordgo.Interaction {
	return &discordgo.Interaction{ID: "42", Token: "tok"}
}

func TestDeferringPassThroughWhenNotDeferred(t *testing.T) {
	ds, rt := newDeferTestSession(t)
	err := ds.InteractionRespond(deferTestInteraction(), &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "hi"},
	})
	require.NoError(t, err)
	calls, _ := rt.snapshot()
	require.Len(t, calls, 1, "non-deferred interactions must be sent directly")
	assert.Contains(t, calls[0], "/callback")
}

func TestDeferringUpdateBecomesEdit(t *testing.T) {
	ds, rt := newDeferTestSession(t)
	i := deferTestInteraction()
	ds.deferInteraction(i)

	err := ds.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{{Title: "room"}},
			Components: []discordgo.MessageComponent{components.ActionRow(components.Button("b", "d::a", discordgo.PrimaryButton))},
		},
	})
	require.NoError(t, err)

	calls, bodies := rt.snapshot()
	require.Len(t, calls, 1, "the deferred ack is the router's job; the handler's reply is the edit")
	assert.Contains(t, calls[0], "PATCH")
	assert.Contains(t, calls[0], "/messages/@original")

	var edit map[string]any
	require.NoError(t, json.Unmarshal([]byte(bodies[0]), &edit))
	require.Contains(t, edit, "embeds")
	require.Contains(t, edit, "components")
}

func TestDeferringChannelMessageBecomesFollowup(t *testing.T) {
	ds, rt := newDeferTestSession(t)
	i := deferTestInteraction()
	ds.deferInteraction(i)

	err := ds.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "oops",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	require.NoError(t, err)

	calls, bodies := rt.snapshot()
	require.Len(t, calls, 1, "the deferred ack is the router's job; the handler's reply is the follow-up")
	assert.Contains(t, calls[0], "POST")
	assert.Contains(t, calls[0], "/webhooks/")

	var follow map[string]any
	require.NoError(t, json.Unmarshal([]byte(bodies[0]), &follow))
	assert.Equal(t, "oops", follow["content"])
	assert.Equal(t, float64(discordgo.MessageFlagsEphemeral), follow["flags"])
}

func TestDeferringDeferredResponseIsNoop(t *testing.T) {
	ds, rt := newDeferTestSession(t)
	i := deferTestInteraction()
	ds.deferInteraction(i)

	err := ds.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	require.NoError(t, err)

	calls, _ := rt.snapshot()
	assert.Empty(t, calls, "the router already deferred; the handler's deferral must not re-ack")
}

func TestDeferringModalOnDeferredIsDropped(t *testing.T) {
	ds, rt := newDeferTestSession(t)
	i := deferTestInteraction()
	ds.deferInteraction(i)

	err := ds.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{CustomID: "d::m", Title: "t"},
	})
	require.NoError(t, err)

	calls, _ := rt.snapshot()
	assert.Empty(t, calls, "a modal can never follow a deferred ack; the reply must be dropped, not sent")
}

func TestIsModalOpener(t *testing.T) {
	assert.True(t, isModalOpener("delve", "puzzle_solve"))
	assert.True(t, isModalOpener("economy", "give"))
	assert.True(t, isModalOpener("market", "sellitem"))
	assert.False(t, isModalOpener("delve", "fight"))
	assert.False(t, isModalOpener("npc", "gift_alchemist"), "gift no longer opens a modal")
	assert.False(t, isModalOpener("npc", "chat_alchemist"))
}

func TestRouterFastSlashRespondsDirectly(t *testing.T) {
	rt := &deferRT{}
	s, err := discordgo.New("test")
	require.NoError(t, err)
	s.Client = &http.Client{Transport: rt}

	r := NewRouter(&Bot{Session: s, Prefix: "!"}, nil)
	r.Slash("ping", "d", func(b *Bot, i *discordgo.InteractionCreate) {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "pong"},
		})
	})

	r.onInteraction(s, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:     "42",
		Type:   discordgo.InteractionApplicationCommand,
		Token:  "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: "200"}},
		Data:   discordgo.ApplicationCommandInteractionData{Name: "ping"},
	}})

	calls, bodies := rt.snapshot()
	require.Len(t, calls, 1, "a fast handler must respond directly, without a deferred ack")
	assert.Contains(t, calls[0], "/callback")
	var ack map[string]any
	require.NoError(t, json.Unmarshal([]byte(bodies[0]), &ack))
	assert.Equal(t, float64(discordgo.InteractionResponseChannelMessageWithSource), ack["type"])
}

func TestRouterSlowSlashDefersAndFollowsUp(t *testing.T) {
	rt := &deferRT{}
	s, err := discordgo.New("test")
	require.NoError(t, err)
	s.Client = &http.Client{Transport: rt}

	r := NewRouter(&Bot{Session: s, Prefix: "!"}, nil)
	r.directRespondWindow = 5 * time.Millisecond
	r.Slash("slow", "d", func(b *Bot, i *discordgo.InteractionCreate) {
		time.Sleep(20 * time.Millisecond)
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "pong"},
		})
	})

	r.onInteraction(s, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:     "42",
		Type:   discordgo.InteractionApplicationCommand,
		Token:  "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: "200"}},
		Data:   discordgo.ApplicationCommandInteractionData{Name: "slow"},
	}})

	calls, bodies := rt.snapshot()
	require.Len(t, calls, 2, "a slow handler must get the deferred ack + follow-up")
	assert.Contains(t, calls[0], "/callback")
	assert.Contains(t, calls[1], "/webhooks/")
	var ack map[string]any
	require.NoError(t, json.Unmarshal([]byte(bodies[0]), &ack))
	assert.Equal(t, float64(discordgo.InteractionResponseDeferredChannelMessageWithSource), ack["type"])
}

func TestRouterFastComponentEditsDirectly(t *testing.T) {
	rt := &deferRT{}
	s, err := discordgo.New("test")
	require.NoError(t, err)
	s.Client = &http.Client{Transport: rt}

	r := NewRouter(&Bot{Session: s, Prefix: "!"}, nil)
	r.Component("test", "run", func(b *Bot, i *discordgo.InteractionCreate) {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{{Title: "view"}},
			},
		})
	})

	r.onInteraction(s, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:     "43",
		Type:   discordgo.InteractionMessageComponent,
		Token:  "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: "200"}},
		Data:   discordgo.MessageComponentInteractionData{CustomID: "test::run::200"},
	}})

	calls, _ := rt.snapshot()
	require.Len(t, calls, 1, "a fast component handler must respond directly, without a deferred ack")
	assert.Contains(t, calls[0], "/callback")
}

func TestRouterSlowComponentDefersAndEdits(t *testing.T) {
	rt := &deferRT{}
	s, err := discordgo.New("test")
	require.NoError(t, err)
	s.Client = &http.Client{Transport: rt}

	r := NewRouter(&Bot{Session: s, Prefix: "!"}, nil)
	r.directRespondWindow = 5 * time.Millisecond
	r.Component("test", "run", func(b *Bot, i *discordgo.InteractionCreate) {
		time.Sleep(20 * time.Millisecond)
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{{Title: "view"}},
			},
		})
	})

	r.onInteraction(s, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:     "43",
		Type:   discordgo.InteractionMessageComponent,
		Token:  "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: "200"}},
		Data:   discordgo.MessageComponentInteractionData{CustomID: "test::run::200"},
	}})

	calls, _ := rt.snapshot()
	require.Len(t, calls, 2, "a slow component handler must get the deferred ack + edit")
	assert.Contains(t, calls[0], "/callback")
	assert.Contains(t, calls[1], "PATCH")
	assert.Contains(t, calls[1], "/messages/@original")
}

func TestRouterRecoversHandlerPanicInDispatchGoroutine(t *testing.T) {
	rt := &deferRT{}
	s, err := discordgo.New("test")
	require.NoError(t, err)
	s.Client = &http.Client{Transport: rt}

	r := NewRouter(&Bot{Session: s, Prefix: "!"}, nil)
	r.Slash("boom", "d", func(b *Bot, i *discordgo.InteractionCreate) {
		panic("handler exploded")
	})

	require.NotPanics(t, func() {
		r.onInteraction(s, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
			ID:     "42",
			Type:   discordgo.InteractionApplicationCommand,
			Token:  "tok",
			Member: &discordgo.Member{User: &discordgo.User{ID: "200"}},
			Data:   discordgo.ApplicationCommandInteractionData{Name: "boom"},
		}})
	})

	calls, bodies := rt.snapshot()
	require.Len(t, calls, 1, "the panic must be recovered and the user informed")
	assert.Contains(t, calls[0], "/callback")
	var resp map[string]any
	require.NoError(t, json.Unmarshal([]byte(bodies[0]), &resp))
	assert.NotEqual(t, float64(discordgo.InteractionResponseDeferredChannelMessageWithSource), resp["type"],
		"a panicked handler must never leave the interaction deferred")
}

func TestRouterDoesNotDeferModalOpener(t *testing.T) {
	rt := &deferRT{}
	s, err := discordgo.New("test")
	require.NoError(t, err)
	s.Client = &http.Client{Transport: rt}

	r := NewRouter(&Bot{Session: s, Prefix: "!"}, nil)
	r.Component("delve", "puzzle_solve", func(b *Bot, i *discordgo.InteractionCreate) {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{CustomID: "delve::puzzle_answer", Title: "riddle"},
		})
	})

	r.onInteraction(s, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:     "44",
		Type:   discordgo.InteractionMessageComponent,
		Token:  "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: "200"}},
		Data:   discordgo.MessageComponentInteractionData{CustomID: "delve::puzzle_solve::200"},
	}})

	calls, bodies := rt.snapshot()
	require.Len(t, calls, 1, "modal openers must respond directly, without a deferred ack")
	assert.Contains(t, calls[0], "/callback")
	var modal map[string]any
	require.NoError(t, json.Unmarshal([]byte(bodies[0]), &modal))
	assert.Equal(t, float64(discordgo.InteractionResponseModal), modal["type"])
}

func TestRouterMarketSellItemOpensModal(t *testing.T) {
	rt := &deferRT{}
	s, err := discordgo.New("test")
	require.NoError(t, err)
	s.Client = &http.Client{Transport: rt}

	r := NewRouter(&Bot{Session: s, Prefix: "!"}, nil)
	r.Component("market", "sellitem", func(b *Bot, i *discordgo.InteractionCreate) {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{CustomID: "market::order::sell::coal", Title: "sell coal"},
		})
	})

	r.onInteraction(s, &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:     "45",
		Type:   discordgo.InteractionMessageComponent,
		Token:  "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: "200"}},
		Data:   discordgo.MessageComponentInteractionData{CustomID: "market::sellitem::1", Values: []string{"coal"}},
	}})

	calls, bodies := rt.snapshot()
	require.Len(t, calls, 1, "the sell item select must open the amount modal directly, without a deferred ack")
	assert.Contains(t, calls[0], "/callback")
	var modal map[string]any
	require.NoError(t, json.Unmarshal([]byte(bodies[0]), &modal))
	assert.Equal(t, float64(discordgo.InteractionResponseModal), modal["type"])
	data, ok := modal["data"].(map[string]any)
	require.True(t, ok, "modal payload must carry a data object")
	assert.Equal(t, "market::order::sell::coal", data["custom_id"])
}
