package interaction

import (
	"github.com/bwmarrin/discordgo"
)

// UserID returns the Discord ID (snowflake string) of the user who triggered
// the interaction.
func UserID(i *discordgo.InteractionCreate) string {
	if i == nil || i.Interaction == nil {
		return ""
	}
	if i.User != nil {
		return i.User.ID
	}
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	return ""
}

// ModalValues extracts the submitted text-input values from a modal, keyed by
// their custom_id.
func ModalValues(i *discordgo.InteractionCreate) map[string]string {
	out := map[string]string{}
	data := i.ModalSubmitData()
	for _, comp := range data.Components {
		row, ok := comp.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, c := range row.Components {
			if ti, ok := c.(*discordgo.TextInput); ok {
				out[ti.CustomID] = ti.Value
			}
		}
	}
	return out
}
