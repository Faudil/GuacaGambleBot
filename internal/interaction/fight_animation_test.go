package interaction

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"

	"guacagamblebot/internal/battle"
)

func TestAnimateFightAlwaysFinishes(t *testing.T) {
	ResetAnimationBudget()
	oldDelay := fightFrameDelay
	fightFrameDelay = time.Millisecond
	defer func() { fightFrameDelay = oldDelay }()

	finished := false
	var journalSeen []string
	turns := []battle.BattleTurn{
		{Pet1HP: 100, Pet2HP: 50, Msg: "a"},
		{Pet1HP: 90, Pet2HP: 40, Msg: "b"},
	}
	AnimateFight(
		turns,
		func(j []string, t battle.BattleTurn) *discordgo.MessageEmbed {
			journalSeen = append(journalSeen, t.Msg)
			return &discordgo.MessageEmbed{}
		},
		func(*discordgo.MessageEmbed, []discordgo.MessageComponent) {},
		func(j []string) { finished = true },
	)
	assert.True(t, finished, "finish must always run")
	assert.Equal(t, []string{"a", "b"}, journalSeen)
}

func TestAnimateFightBudgetCollapse(t *testing.T) {
	SetAnimationBudget(1)
	oldDelay := fightFrameDelay
	fightFrameDelay = time.Millisecond
	defer func() { fightFrameDelay = oldDelay }()

	finished := false
	var journalSeen []string
	turns := []battle.BattleTurn{
		{Pet1HP: 100, Pet2HP: 50, Msg: "a"},
		{Pet1HP: 90, Pet2HP: 40, Msg: "b"},
	}
	AnimateFight(
		turns,
		func(j []string, t battle.BattleTurn) *discordgo.MessageEmbed {
			journalSeen = append(journalSeen, t.Msg)
			return &discordgo.MessageEmbed{}
		},
		func(*discordgo.MessageEmbed, []discordgo.MessageComponent) {},
		func(j []string) { finished = true },
	)
	assert.True(t, finished, "finish must run even when the budget collapses the animation")
	assert.Equal(t, []string{"a"}, journalSeen, "only the first frame fits in the budget")
	ResetAnimationBudget()
}

func TestAnimateFightCapsTurns(t *testing.T) {
	// Generous budget so the turn cap (12) is hit before the per-second
	// animation budget would cut the animation short.
	SetAnimationBudget(30)
	oldDelay := fightFrameDelay
	fightFrameDelay = time.Millisecond
	defer func() { fightFrameDelay = oldDelay }()

	turns := make([]battle.BattleTurn, maxFightTurns+5)
	for i := range turns {
		turns[i] = battle.BattleTurn{Pet1HP: 100, Pet2HP: 100 - i, Msg: "x"}
	}
	edits := 0
	AnimateFight(
		turns,
		func(j []string, t battle.BattleTurn) *discordgo.MessageEmbed {
			return &discordgo.MessageEmbed{}
		},
		func(*discordgo.MessageEmbed, []discordgo.MessageComponent) { edits++ },
		func(j []string) {},
	)
	assert.Equal(t, maxFightTurns, edits, "animation is capped to the last maxFightTurns turns")
}
