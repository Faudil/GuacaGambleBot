package interaction

import (
	"sync"
	"time"
)

// AnimationBudgetPerSecond caps how many animated battle frame edits the bot
// sends per second across all servers and all concurrent fights. Discord's
// global limit is 50 requests/second, so animation traffic is capped well
// below that to leave headroom for commands and other traffic. When the budget
// is exhausted, battle animations collapse to their final frame instead of
// hitting Discord's rate limits (429s).
const AnimationBudgetPerSecond = 10

const animationWindow = time.Second

var (
	animationMu     sync.Mutex
	animationTimes  []time.Time
	animationBudget = AnimationBudgetPerSecond
)

// CanAnimateEdit reports whether a new animated frame edit may be sent right
// now. It is safe for concurrent use by all fight animations across the bot.
func CanAnimateEdit() bool {
	animationMu.Lock()
	defer animationMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-animationWindow)
	kept := animationTimes[:0]
	for _, t := range animationTimes {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	animationTimes = kept
	if len(animationTimes) >= animationBudget {
		return false
	}
	animationTimes = append(animationTimes, now)
	return true
}

// SetAnimationBudget overrides the per-second animation edit budget and clears
// the current window. Values below 1 are clamped to 1. Intended for tests.
func SetAnimationBudget(n int) {
	animationMu.Lock()
	defer animationMu.Unlock()
	animationBudget = n
	if animationBudget < 1 {
		animationBudget = 1
	}
	animationTimes = animationTimes[:0]
}

// ResetAnimationBudget restores the default budget and clears the window.
func ResetAnimationBudget() {
	animationMu.Lock()
	defer animationMu.Unlock()
	animationBudget = AnimationBudgetPerSecond
	animationTimes = animationTimes[:0]
}
