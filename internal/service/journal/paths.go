// Package journal implements the player journal: open-ended progression paths
// made of steps (milestones). Each path lives in its own file and is registered
// in init() — adding a new path means writing one file, nothing else.
package journal

import (
	"guacagamblebot/internal/store"
)

// Check reports a step's progress against its target. done is true when the
// milestone is reached. Checks read live state from the store (stats, tables),
// so progress is always derived, never cached.
type Check func(s *store.Store, userID int64) (progress, target int, done bool)

// Reward is granted once when a step is completed.
type Reward struct {
	Money   int
	Crowns  int
	ItemIDs []string
}

// Step is a single milestone on a path. Steps are completed in order.
type Step struct {
	TextKey string // i18n key describing the milestone
	Check   Check
	Reward  Reward
}

// Path is one progression track (Prospector, High Roller, ...).
type Path struct {
	ID       string
	Emoji    string
	TitleKey string
	DescKey  string
	Steps    []Step
}

// paths is the registry of all journal paths.
var paths = map[string]*Path{}

// GetPaths returns every registered path.
func GetPaths() []*Path {
	out := make([]*Path, 0, len(paths))
	for _, p := range paths {
		out = append(out, p)
	}
	return out
}

// GetPath returns a path by ID, or nil.
func GetPath(id string) *Path {
	return paths[id]
}
