package components

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const sep = "::"

// Encode builds a custom_id from its parts, e.g. ("economy", "balance") ->
// "economy::balance". The custom_id namespace is shared by buttons and modals.
func Encode(parts ...string) string {
	return strings.Join(parts, sep)
}

// Decode splits a custom_id back into its domain, action and any extra parts.
func Decode(customID string) (domain, action string, rest []string) {
	parts := strings.Split(customID, sep)
	if len(parts) == 0 {
		return "", "", nil
	}
	domain = parts[0]
	if len(parts) > 1 {
		action = parts[1]
	}
	if len(parts) > 2 {
		rest = parts[2:]
	}
	return
}

// EncodeOwner appends the embed owner's user id as the final element of a
// custom_id, so the router can verify that only the user who created the
// message may interact with its components.
func EncodeOwner(ownerID int64, parts ...string) string {
	return Encode(append(parts, strconv.FormatInt(ownerID, 10))...)
}

// OwnerID extracts the owner user id previously appended by EncodeOwner. It
// returns false when the custom_id carries no trailing owner id.
func OwnerID(customID string) (int64, bool) {
	_, _, rest := Decode(customID)
	if len(rest) == 0 {
		return 0, false
	}
	id, err := strconv.ParseInt(rest[len(rest)-1], 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// Embed builds a basic message embed.
func Embed(title, description string, color int) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{Title: title, Description: description, Color: color}
}

// Field builds an embed field.
func Field(name, value string, inline bool) *discordgo.MessageEmbedField {
	return &discordgo.MessageEmbedField{Name: name, Value: value, Inline: inline}
}

// Button builds a button component with a custom_id.
func Button(label, customID string, style discordgo.ButtonStyle) discordgo.MessageComponent {
	return discordgo.Button{Label: label, CustomID: customID, Style: style}
}

// ButtonDisabled builds a button component with an explicit disabled state.
func ButtonDisabled(label, customID string, style discordgo.ButtonStyle, disabled bool) discordgo.MessageComponent {
	return discordgo.Button{Label: label, CustomID: customID, Style: style, Disabled: disabled}
}

// ActionRow wraps one or more components in an action row.
func ActionRow(components ...discordgo.MessageComponent) discordgo.MessageComponent {
	return discordgo.ActionsRow{Components: components}
}

// TextInput builds a modal text input.
func TextInput(customID, label string, required bool, placeholder string, style discordgo.TextInputStyle, minLen, maxLen int) discordgo.MessageComponent {
	return discordgo.TextInput{
		CustomID:    customID,
		Label:       label,
		Required:    required,
		Placeholder: placeholder,
		Style:       style,
		MinLength:   minLen,
		MaxLength:   maxLen,
	}
}

// ModalResponse wraps text inputs into a modal interaction response payload.
func ModalResponse(customID, title string, inputs ...discordgo.MessageComponent) *discordgo.InteractionResponseData {
	rows := make([]discordgo.MessageComponent, 0, len(inputs))
	for _, in := range inputs {
		rows = append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{in}})
	}
	return &discordgo.InteractionResponseData{
		CustomID:   customID,
		Title:      title,
		Components: rows,
	}
}

// InteractionResponse is a small helper to build a message response carrying an
// embed and optional components.
func InteractionResponse(responseType discordgo.InteractionResponseType, embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent) *discordgo.InteractionResponse {
	data := &discordgo.InteractionResponseData{}
	if embed != nil {
		data.Embeds = []*discordgo.MessageEmbed{embed}
	}
	if comps != nil {
		data.Components = comps
	}
	return &discordgo.InteractionResponse{Type: responseType, Data: data}
}

// WebhookEditResponse builds a WebhookEdit from an embed and optional
// components, useful for editing interaction responses mid-animation.
func WebhookEditResponse(embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent) *discordgo.WebhookEdit {
	data := &discordgo.WebhookEdit{}
	if embed != nil {
		embeds := []*discordgo.MessageEmbed{embed}
		data.Embeds = &embeds
	}
	if comps != nil {
		data.Components = &comps
	}
	return data
}

// SafeEmoji builds a ComponentEmoji from a Unicode emoji string, returning nil
// when the string is not exactly one emoji. Discord rejects the entire
// interaction response with 400 Invalid Form Body if a component emoji name
// holds more than one emoji (e.g. "🐺🐺🐺"), so a single bad entry in a content
// table silently blanks the whole menu for every user who owns that entry.
func SafeEmoji(name string) *discordgo.ComponentEmoji {
	if !isSingleEmoji(name) {
		return nil
	}
	return &discordgo.ComponentEmoji{Name: name}
}

// Discord's label limits, which the emoji fallback below must respect.
const (
	maxOptionLabel = 100
	maxButtonLabel = 80
)

// SanitizeComponents rewrites any component emoji Discord would reject into the
// component's label, where multi-emoji strings are perfectly legal. Emoji reach
// components from content tables (pet species, items, NPCs, lore, rarities), so
// one bad entry would otherwise take down the whole message with a 400 rather
// than just looking odd. Moving it into the label keeps the emoji visible.
//
// This runs at the session boundary, so it protects every component the bot
// sends, including call sites that build discordgo structs directly.
func SanitizeComponents(rows []discordgo.MessageComponent) []discordgo.MessageComponent {
	if rows == nil {
		return nil
	}
	out := make([]discordgo.MessageComponent, len(rows))
	for i, row := range rows {
		out[i] = sanitizeComponent(row)
	}
	return out
}

func sanitizeComponent(c discordgo.MessageComponent) discordgo.MessageComponent {
	switch v := c.(type) {
	case discordgo.ActionsRow:
		v.Components = SanitizeComponents(v.Components)
		return v
	case *discordgo.ActionsRow:
		if v == nil {
			return c
		}
		row := *v
		row.Components = SanitizeComponents(row.Components)
		return &row
	case discordgo.Button:
		v.Label, v.Emoji = demoteEmoji(v.Label, v.Emoji, maxButtonLabel)
		return v
	case *discordgo.Button:
		if v == nil {
			return c
		}
		b := *v
		b.Label, b.Emoji = demoteEmoji(b.Label, b.Emoji, maxButtonLabel)
		return &b
	case discordgo.SelectMenu:
		v.Options = sanitizeOptions(v.Options)
		return v
	case *discordgo.SelectMenu:
		if v == nil {
			return c
		}
		m := *v
		m.Options = sanitizeOptions(m.Options)
		return &m
	default:
		return c
	}
}

func sanitizeOptions(opts []discordgo.SelectMenuOption) []discordgo.SelectMenuOption {
	if opts == nil {
		return nil
	}
	out := make([]discordgo.SelectMenuOption, len(opts))
	for i, o := range opts {
		o.Label, o.Emoji = demoteEmoji(o.Label, o.Emoji, maxOptionLabel)
		out[i] = o
	}
	return out
}

// demoteEmoji leaves a legal emoji alone; an illegal one is prefixed onto the
// label and cleared from the emoji field. Custom emoji (which carry an ID) are
// never touched — their Name is a plain identifier, not a Unicode emoji.
func demoteEmoji(label string, e *discordgo.ComponentEmoji, maxLabel int) (string, *discordgo.ComponentEmoji) {
	if e == nil || e.ID != "" || isSingleEmoji(e.Name) {
		return label, e
	}
	if e.Name != "" {
		label = strings.TrimSpace(e.Name + " " + label)
	}
	if len([]rune(label)) > maxLabel {
		label = string([]rune(label)[:maxLabel])
	}
	return label, nil
}

// isSingleEmoji reports whether name is one emoji: a single rune, or a single
// ZWJ/variation-selector/skin-tone sequence built around one base rune.
func isSingleEmoji(name string) bool {
	if name == "" {
		return false
	}
	base := 0
	for _, r := range name {
		switch {
		case r == 0x200D: // zero-width joiner: continues the same emoji
			return true
		case r == 0xFE0F || r == 0xFE0E: // variation selector
		case r >= 0x1F3FB && r <= 0x1F3FF: // skin-tone modifier
		default:
			base++
		}
	}
	return base == 1
}
