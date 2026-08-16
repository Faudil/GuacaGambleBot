package delve

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

func TestVaultKeyRoom(t *testing.T) {
	room := VaultKeyRoom("en")

	assert.Equal(t, RoomVaultKey, room.Type)
	assert.NotEmpty(t, room.Description)

	var actions []string
	for _, b := range room.Buttons {
		actions = append(actions, b.Action)
	}
	assert.Contains(t, actions, "key_take")
	assert.Contains(t, actions, "floor_leave")

	// The key action must be the primary one.
	assert.Equal(t, "key_take", room.Buttons[0].Action)
	assert.Equal(t, discordgo.SuccessButton, room.Buttons[0].Style)
}
