package components

import (
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
