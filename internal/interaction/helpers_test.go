package interaction

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestUserIDFromMember(t *testing.T) {
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Member: &discordgo.Member{User: &discordgo.User{ID: "42"}},
	}}
	assert.Equal(t, "42", UserID(i))
}

func TestUserIDFromUser(t *testing.T) {
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		User: &discordgo.User{ID: "7"},
	}}
	assert.Equal(t, "7", UserID(i))
}

func TestUserIDEmpty(t *testing.T) {
	assert.Equal(t, "", UserID(&discordgo.InteractionCreate{}))
}

func TestModalValues(t *testing.T) {
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionModalSubmit,
		Data: discordgo.ModalSubmitInteractionData{
			CustomID: "economy::give_submit",
			Components: []discordgo.MessageComponent{
				&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.TextInput{CustomID: "recipient", Value: "123"},
					&discordgo.TextInput{CustomID: "amount", Value: "50"},
				}},
			},
		},
	}}
	vals := ModalValues(i)
	assert.Equal(t, "123", vals["recipient"])
	assert.Equal(t, "50", vals["amount"])
}

func TestModalValuesIgnoresNonTextInputs(t *testing.T) {
	i := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type: discordgo.InteractionModalSubmit,
		Data: discordgo.ModalSubmitInteractionData{
			Components: []discordgo.MessageComponent{
				&discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					&discordgo.Button{Label: "x", CustomID: "btn"},
				}},
			},
		},
	}}
	assert.Empty(t, ModalValues(i))
}
