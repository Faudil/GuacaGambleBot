package interaction

import (
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/battle"
)

// Fight pacing shared by every animated battle (boss, PvP and hunt).
const (
	maxFightTurns   = 12
	maxJournalLines = 5
)

// fightFrameDelay paces the animation; a var so tests can shorten it.
var fightFrameDelay = 900 * time.Millisecond

// FightEdit sends one battle frame edit. The caller decides whether it edits
// an interaction response or a channel message (prefix path).
type FightEdit func(embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent)

// AnimateFight replays a battle turn by turn with live HP bars. It paces the
// frames, keeps a rolling journal of the last few actions, caps the animation
// to a bounded number of turns, and collapses straight to the finish when the
// global animation budget is exhausted (rate-limit safety). finish always
// runs, receiving the final journal so callers can render the result frame.
func AnimateFight(turns []battle.BattleTurn,
	buildFrame func(journal []string, t battle.BattleTurn) *discordgo.MessageEmbed,
	edit FightEdit, finish func(journal []string)) {

	if len(turns) > maxFightTurns {
		turns = turns[len(turns)-maxFightTurns:]
	}

	time.Sleep(fightFrameDelay)
	journal := make([]string, 0, maxJournalLines)
	for _, t := range turns {
		// If the global animation budget is exhausted, jump straight to the
		// result instead of risking Discord rate limits.
		if !CanAnimateEdit() {
			break
		}
		journal = append(journal, t.Msg)
		if len(journal) > maxJournalLines {
			journal = journal[len(journal)-maxJournalLines:]
		}
		edit(buildFrame(journal, t), nil)
		time.Sleep(fightFrameDelay)
	}
	finish(journal)
}
