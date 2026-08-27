package components

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// FightLabels are the localized labels used by the retro battle screen.
type FightLabels struct {
	VS  string // "VS" separator label
	HP  string // "HP" / "PV"
	Lvl string // "Lvl" / "Niv"
}

// FightLabelsFor returns the localized fight labels for the given language.
// The VS label is provided by the caller since each fight type uses its own
// locale key.
func FightLabelsFor(lang, vs string) FightLabels {
	labels := FightLabels{VS: vs, HP: "HP", Lvl: "Lvl"}
	if lang == "fr" {
		labels.HP = "PV"
		labels.Lvl = "Niv"
	}
	return labels
}

// FightFrameEmbed builds the retro battle screen embed: title, the stacked
// markdown body and the border color matching the latest journal action.
func FightFrameEmbed(title string, p1, p2 DisplayPet, labels FightLabels, journal []string) *discordgo.MessageEmbed {
	body := RenderFightFrame(p1, p2, labels, journal)
	return Embed(title, body, FrameColor(journal))
}

// DisplayPet is the display data of one combatant for the retro battle frame.
type DisplayPet struct {
	Name  string
	Emoji string
	Level int
	HP    int
	MaxHP int
	Owner string // optional owner display name (PvP only)
	IsKO  bool   // pet is knocked out: bar empty, name crossed by a skull
}

// HPBar renders a 10-block colored HP bar using emoji squares so the colors
// render on every Discord client (desktop, web and mobile, which does not
// support ANSI colors): green above 50%, yellow from 25% to 50%, red below
// 25%, all dark when knocked out.
func HPBar(hp, maxHP int) string {
	const blocks = 10
	if maxHP <= 0 {
		maxHP = 1
	}
	filled := hp * blocks / maxHP
	if filled < 0 {
		filled = 0
	}
	if filled > blocks {
		filled = blocks
	}
	fill := "🟩"
	switch {
	case hp <= 0:
		fill = "⬛"
	case hp*100/maxHP < 25:
		fill = "🟥"
	case hp*100/maxHP < 50:
		fill = "🟨"
	}
	return strings.Repeat(fill, filled) + strings.Repeat("⬛", blocks-filled)
}

// RenderFightFrame builds the retro RPG battle screen as stacked markdown:
// each combatant gets its own block (owner, bold name, HP bar), a VS line
// separates them, then a divider and the combat journal. Emojis have variable
// width on Discord, so the layout never relies on side-by-side alignment —
// that makes it render correctly on every client.
func RenderFightFrame(p1, p2 DisplayPet, labels FightLabels, journal []string) string {
	lines := []string{}

	if p1.Owner != "" {
		lines = append(lines, "*"+p1.Owner+"*")
	}
	lines = append(lines, petLabel(p1, labels.Lvl))
	lines = append(lines, hpLine(p1, labels.HP))

	lines = append(lines, "")
	lines = append(lines, "⚡ **"+labels.VS+"** ⚡")
	lines = append(lines, "")

	if p2.Owner != "" {
		lines = append(lines, "*"+p2.Owner+"*")
	}
	lines = append(lines, petLabel(p2, labels.Lvl))
	lines = append(lines, hpLine(p2, labels.HP))

	lines = append(lines, "")
	lines = append(lines, strings.Repeat("─", 40))

	// Journal lines keep their **bold** markers: markdown is rendered outside
	// code blocks, so attacker names and values show up bold.
	lines = append(lines, journal...)

	return strings.Join(lines, "\n")
}

func petLabel(p DisplayPet, lvlLabel string) string {
	name := p.Emoji + " **" + p.Name + "**"
	if p.IsKO {
		name = "💀 **" + p.Name + "**"
	}
	return name + " — " + lvlLabel + " " + strconv.Itoa(p.Level)
}

func hpLine(p DisplayPet, hpLabel string) string {
	return hpLabel + " " + HPBar(p.HP, p.MaxHP) + " **" + fmt.Sprintf("%d/%d", p.HP, p.MaxHP) + "**"
}

// FrameColor returns the embed border colour matching the latest journal
// action, so a fight's border tracks what just happened: danger on a crit,
// muted on a dodge, arcane on a stun, success on a heal, warning on a
// damage-over-time tick, info on an ordinary hit, and idle before the first
// blow lands.
func FrameColor(journal []string) int {
	if len(journal) == 0 {
		return ColorIdle
	}
	last := journal[len(journal)-1]
	switch {
	case strings.Contains(last, "CRITICAL HIT"):
		return ColorDanger
	case strings.Contains(last, "dodges"):
		return ColorMuted
	case strings.Contains(last, "stunned"):
		return ColorArcane
	case strings.Contains(last, "heals"), strings.Contains(last, "regenerates"),
		strings.Contains(last, "drains"), strings.Contains(last, "leeches"):
		return ColorSuccess
	case strings.Contains(last, "poison"), strings.Contains(last, "bleed"),
		strings.HasPrefix(last, "🔥"), strings.Contains(last, "burns"):
		return ColorWarning
	}
	return ColorInfo
}
