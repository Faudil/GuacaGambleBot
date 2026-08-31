package components

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestIsSingleEmoji(t *testing.T) {
	good := []string{"🐺", "🐿️", "👍🏽", "👩‍🚀", "⚡"}
	for _, e := range good {
		if !isSingleEmoji(e) {
			t.Errorf("%q should be a single emoji", e)
		}
	}
	bad := []string{"", "🐺🐺🐺", "🐍⚡", "🐢🌳", "🐿️❄️"}
	for _, e := range bad {
		if isSingleEmoji(e) {
			t.Errorf("%q should be rejected", e)
		}
	}
}

// A multi-emoji species (the "🐺🐺🐺" Cerbère) made Discord reject the whole
// interaction response with 400 Invalid Form Body, so /pet and the sanctuary
// pickers silently sent nothing to the one player who owned one. The emoji must
// end up in the label instead, where multi-emoji strings are legal.
func TestSanitizeComponentsDemotesIllegalEmojiIntoLabel(t *testing.T) {
	rows := []discordgo.MessageComponent{
		ActionRow(discordgo.SelectMenu{
			CustomID: "pets::pet::1",
			Options: []discordgo.SelectMenuOption{
				{Label: "Cerbère", Value: "8", Emoji: &discordgo.ComponentEmoji{Name: "🐺🐺🐺"}},
				{Label: "Winnie", Value: "2", Emoji: &discordgo.ComponentEmoji{Name: "🐻"}},
			},
		}),
		ActionRow(discordgo.Button{Label: "Hatch", Emoji: &discordgo.ComponentEmoji{Name: "🥚🥚"}}),
	}

	got := SanitizeComponents(rows)

	menu := got[0].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	if menu.Options[0].Emoji != nil {
		t.Errorf("illegal emoji must be cleared, got %v", menu.Options[0].Emoji)
	}
	if menu.Options[0].Label != "🐺🐺🐺 Cerbère" {
		t.Errorf("emoji must move into the label, got %q", menu.Options[0].Label)
	}
	if menu.Options[1].Emoji == nil || menu.Options[1].Emoji.Name != "🐻" {
		t.Error("a legal emoji must be left alone")
	}
	btn := got[1].(discordgo.ActionsRow).Components[0].(discordgo.Button)
	if btn.Emoji != nil || btn.Label != "🥚🥚 Hatch" {
		t.Errorf("button emoji not demoted: %v %q", btn.Emoji, btn.Label)
	}
}

// Custom (uploaded) emoji carry an ID and a plain-text name; they must survive.
func TestSanitizeComponentsKeepsCustomEmoji(t *testing.T) {
	rows := []discordgo.MessageComponent{
		ActionRow(discordgo.Button{Label: "x", Emoji: &discordgo.ComponentEmoji{Name: "guaca", ID: "123"}}),
	}
	btn := SanitizeComponents(rows)[0].(discordgo.ActionsRow).Components[0].(discordgo.Button)
	if btn.Emoji == nil || btn.Emoji.ID != "123" {
		t.Error("custom emoji must be preserved")
	}
}
