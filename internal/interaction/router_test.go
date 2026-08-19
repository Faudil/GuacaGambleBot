package interaction

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func newTestInteraction(t discordgo.InteractionType, data discordgo.InteractionData) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: t, Data: data,
		GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: "200"}},
	}}
}

func newTestInteractionAs(t discordgo.InteractionType, data discordgo.InteractionData, userID string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: t, Data: data,
		GuildID: "100", Token: "tok",
		Member: &discordgo.Member{User: &discordgo.User{ID: userID}},
	}}
}

func TestSlashDispatch(t *testing.T) {
	called := false
	r := NewRouter(&Bot{Prefix: "!"}, nil)
	r.Slash("economy", "desc", func(b *Bot, i *discordgo.InteractionCreate) { called = true })

	r.onInteraction(&discordgo.Session{}, newTestInteraction(
		discordgo.InteractionApplicationCommand,
		discordgo.ApplicationCommandInteractionData{Name: "economy"},
	))
	assert.True(t, called, "registered slash handler should be called")

	called = false
	r.onInteraction(&discordgo.Session{}, newTestInteraction(
		discordgo.InteractionApplicationCommand,
		discordgo.ApplicationCommandInteractionData{Name: "unknown"},
	))
	assert.False(t, called, "unregistered slash handler must not be called")
}

func TestPrefixDispatch(t *testing.T) {
	called := false
	r := NewRouter(&Bot{Prefix: "!"}, nil)
	r.Prefix("economy", func(b *Bot, s *discordgo.Session, m *discordgo.Message) { called = true })

	r.onMessage(&discordgo.Session{}, &discordgo.MessageCreate{Message: &discordgo.Message{Content: "!economy", Author: &discordgo.User{ID: "1"}}})
	assert.True(t, called, "prefix !economy should dispatch")

	called = false
	r.onMessage(&discordgo.Session{}, &discordgo.MessageCreate{Message: &discordgo.Message{Content: "!economy", Author: &discordgo.User{Bot: true}}})
	assert.False(t, called, "bot messages must be ignored")

	called = false
	r.onMessage(&discordgo.Session{}, &discordgo.MessageCreate{Message: &discordgo.Message{Content: "economy", Author: &discordgo.User{ID: "1"}}})
	assert.False(t, called, "non-prefixed messages must be ignored")

	called = false
	r.onMessage(&discordgo.Session{}, &discordgo.MessageCreate{Message: &discordgo.Message{Content: "!nope", Author: &discordgo.User{ID: "1"}}})
	assert.False(t, called, "unknown prefix command must be ignored")
}

func TestComponentDispatch(t *testing.T) {
	called := false
	r := NewRouter(&Bot{Prefix: "!"}, nil)
	r.Component("economy", "balance", func(b *Bot, i *discordgo.InteractionCreate) { called = true })

	r.onInteraction(&discordgo.Session{}, newTestInteraction(
		discordgo.InteractionMessageComponent,
		discordgo.MessageComponentInteractionData{CustomID: "economy::balance::200"},
	))
	assert.True(t, called, "registered component handler should be called")

	called = false
	r.onInteraction(&discordgo.Session{}, newTestInteraction(
		discordgo.InteractionMessageComponent,
		discordgo.MessageComponentInteractionData{CustomID: "economy::daily"},
	))
	assert.False(t, called, "unregistered component action must not be called")
}

func TestModalDispatch(t *testing.T) {
	called := false
	r := NewRouter(&Bot{Prefix: "!"}, nil)
	r.Modal("economy", "give_submit", func(b *Bot, i *discordgo.InteractionCreate) { called = true })

	r.onInteraction(&discordgo.Session{}, newTestInteraction(
		discordgo.InteractionModalSubmit,
		discordgo.ModalSubmitInteractionData{CustomID: "economy::give_submit"},
	))
	assert.True(t, called, "registered modal handler should be called")
}

func TestSlashDefsRegistered(t *testing.T) {
	r := NewRouter(&Bot{Prefix: "!"}, nil)
	r.Slash("economy", "Économie", nil)
	assert.Len(t, r.slashDefs, 1)
	assert.Equal(t, "economy", r.slashDefs[0].Name)
}

func TestOwnerGatedComponentDispatch(t *testing.T) {
	called := false
	r := NewRouter(&Bot{Prefix: "!"}, nil)
	r.Component("farm", "menu", func(b *Bot, i *discordgo.InteractionCreate) { called = true })

	// The embed owner (custom_id carries their id) may interact.
	r.onInteraction(&discordgo.Session{}, newTestInteractionAs(
		discordgo.InteractionMessageComponent,
		discordgo.MessageComponentInteractionData{CustomID: "farm::menu::200"},
		"200",
	))
	assert.True(t, called, "owner clicking their own menu must be allowed")

	// A different user clicking the same message must be rejected.
	called = false
	r.onInteraction(&discordgo.Session{}, newTestInteractionAs(
		discordgo.InteractionMessageComponent,
		discordgo.MessageComponentInteractionData{CustomID: "farm::menu::300"},
		"400",
	))
	assert.False(t, called, "non-owner clicking a personal menu must be rejected")

	// A legacy custom_id without an owner id must be rejected (fail closed).
	called = false
	r.onInteraction(&discordgo.Session{}, newTestInteractionAs(
		discordgo.InteractionMessageComponent,
		discordgo.MessageComponentInteractionData{CustomID: "farm::menu"},
		"500",
	))
	assert.False(t, called, "custom_id without owner id must be rejected")

	// Non-gated domains remain open to anyone.
	called = false
	r.Component("market", "nav", func(b *Bot, i *discordgo.InteractionCreate) { called = true })
	r.onInteraction(&discordgo.Session{}, newTestInteractionAs(
		discordgo.InteractionMessageComponent,
		discordgo.MessageComponentInteractionData{CustomID: "market::nav::prev::0::all"},
		"600",
	))
	assert.True(t, called, "non-gated domain must dispatch without an owner id")

	// Exempt actions inside gated domains stay open.
	called = false
	r.Component("pets", "battle_accept", func(b *Bot, i *discordgo.InteractionCreate) { called = true })
	r.onInteraction(&discordgo.Session{}, newTestInteractionAs(
		discordgo.InteractionMessageComponent,
		discordgo.MessageComponentInteractionData{CustomID: "pets::battle_accept::123::200::456"},
		"700",
	))
	assert.True(t, called, "exempt action must dispatch for any user")

	// House embeds are personal menus: only the owner may operate them.
	called = false
	r.Component("house", "show", func(b *Bot, i *discordgo.InteractionCreate) { called = true })
	r.onInteraction(&discordgo.Session{}, newTestInteractionAs(
		discordgo.InteractionMessageComponent,
		discordgo.MessageComponentInteractionData{CustomID: "house::show::800"},
		"800",
	))
	assert.True(t, called, "owner clicking their own house embed must be allowed")

	called = false
	r.onInteraction(&discordgo.Session{}, newTestInteractionAs(
		discordgo.InteractionMessageComponent,
		discordgo.MessageComponentInteractionData{CustomID: "house::show::800"},
		"801",
	))
	assert.False(t, called, "non-owner clicking a house embed must be rejected")

	called = false
	r.onInteraction(&discordgo.Session{}, newTestInteractionAs(
		discordgo.InteractionMessageComponent,
		discordgo.MessageComponentInteractionData{CustomID: "house::show"},
		"802",
	))
	assert.False(t, called, "house custom_id without owner id must be rejected (fail closed)")
}
