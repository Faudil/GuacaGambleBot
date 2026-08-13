package components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHPBarFull(t *testing.T) {
	bar := HPBar(200, 200)
	assert.Equal(t, 10, strings.Count(bar, "🟩"))
	assert.Equal(t, 0, strings.Count(bar, "⬛"))
	assert.Equal(t, 10, len([]rune(bar)))
}

func TestHPBarLow(t *testing.T) {
	bar := HPBar(30, 200)
	assert.Equal(t, 1, strings.Count(bar, "🟥"))
	assert.Equal(t, 9, strings.Count(bar, "⬛"))
}

func TestHPBarMid(t *testing.T) {
	bar := HPBar(70, 200)
	assert.Equal(t, 3, strings.Count(bar, "🟨"))
	assert.Equal(t, 7, strings.Count(bar, "⬛"))
}

func TestHPBarKO(t *testing.T) {
	bar := HPBar(0, 200)
	assert.Equal(t, 10, strings.Count(bar, "⬛"))
	assert.Equal(t, 0, strings.Count(bar, "🟩"))
}

func TestRenderFightFrame(t *testing.T) {
	p1 := DisplayPet{Name: "Draco", Emoji: "🐉", Level: 15, HP: 150, MaxHP: 200, Owner: "Alice"}
	p2 := DisplayPet{Name: "Kael", Emoji: "🦅", Level: 30, HP: 300, MaxHP: 420, Owner: "Bob"}
	labels := FightLabels{VS: "VS", HP: "HP", Lvl: "Lvl"}
	journal := []string{
		"💥 **CRITICAL HIT!** ⚔️ 🦅 **Kael** deals **85** damage",
		"💨 🐉 **Draco** dodges Kael's attack!",
	}

	body := RenderFightFrame(p1, p2, labels, journal)
	assert.Contains(t, body, "Alice")
	assert.Contains(t, body, "Bob")
	assert.Contains(t, body, "**Draco**")
	assert.Contains(t, body, "**Kael**")
	assert.Contains(t, body, "150/200")
	assert.Contains(t, body, "300/420")
	assert.Contains(t, body, "**VS**")
	assert.Contains(t, body, "🟩") // colored bars present
	// No ANSI escapes anywhere (mobile-safe); markdown bold markers are kept
	// so names and values render bold outside code blocks.
	assert.NotContains(t, body, "\x1b")
	assert.Contains(t, body, "**")
	// Each combatant is stacked, not side-by-side.
	assert.True(t, strings.Index(body, "Draco") < strings.Index(body, "Kael"))
	assert.True(t, strings.Index(body, "150/200") < strings.Index(body, "300/420"))
}

func TestFightLabelsFor(t *testing.T) {
	en := FightLabelsFor("en", "VS")
	assert.Equal(t, FightLabels{VS: "VS", HP: "HP", Lvl: "Lvl"}, en)

	fr := FightLabelsFor("fr", "VS")
	assert.Equal(t, FightLabels{VS: "VS", HP: "PV", Lvl: "Niv"}, fr)
}

func TestFightFrameEmbed(t *testing.T) {
	p1 := DisplayPet{Name: "Draco", Emoji: "🐉", Level: 15, HP: 150, MaxHP: 200}
	p2 := DisplayPet{Name: "Kael", Emoji: "🦅", Level: 30, HP: 300, MaxHP: 420}
	emb := FightFrameEmbed("⚔️ ARENA", p1, p2, FightLabels{VS: "VS", HP: "HP", Lvl: "Lvl"},
		[]string{"💥 **CRITICAL HIT!**"})
	assert.Equal(t, "⚔️ ARENA", emb.Title)
	assert.NotContains(t, emb.Description, "```", "markdown must render, so no code fence")
	assert.Contains(t, emb.Description, "🟩")
	assert.NotContains(t, emb.Description, "\x1b")
	assert.Equal(t, 0xFF0000, emb.Color)
}

func TestFrameColor(t *testing.T) {
	assert.Equal(t, 0xFF0000, FrameColor([]string{"💥 **CRITICAL HIT!**"}))
	assert.Equal(t, 0x95A5A6, FrameColor([]string{"💨 **X** dodges Y's attack!"}))
	assert.Equal(t, 0x9B59B6, FrameColor([]string{"💫 **X** is stunned (2 turns left)"}))
	assert.Equal(t, 0x2ECC71, FrameColor([]string{"💚 **X** regenerates **6** HP."}))
	assert.Equal(t, 0xE67E22, FrameColor([]string{"🔥 **X** burns and loses **16** HP."}))
	assert.Equal(t, 0x3498DB, FrameColor([]string{"⚔️ 🐉 **X** deals **32** damage"}))
	assert.Equal(t, 0x2C3E50, FrameColor(nil))
}
