package character

import "time"

// Skill defines an active ability with a daily usage limit and cooldown.
type Skill struct {
	ID           string
	Name         string
	Description  string
	Emoji        string
	UnlockLevel  int
	DailyLimit   int
	CooldownMins int
}

// CooldownDur returns the cooldown as a time.Duration.
func (s Skill) CooldownDur() time.Duration {
	return time.Duration(s.CooldownMins) * time.Minute
}

var allSkills = []Skill{
	{
		ID: "overclock", Name: "Overclock",
		Description: "Grants 3 extra fishing, hunting, and mining actions.",
		Emoji:       "⚡", UnlockLevel: 3, DailyLimit: 3, CooldownMins: 120,
	},
	{
		ID: "pet_bond", Name: "Pet Bond",
		Description: "Active pet gains +25% all stats for its next battle or hunt.",
		Emoji:       "❤️", UnlockLevel: 6, DailyLimit: 3, CooldownMins: 240,
	},
	{
		ID: "bulwark", Name: "Bulwark",
		Description: "Active pet heals to full HP and is immune for the first round of its next battle.",
		Emoji:       "🛡️", UnlockLevel: 10, DailyLimit: 2, CooldownMins: 240,
	},
	{
		ID: "scavenger", Name: "Scavenger",
		Description: "Next gathering action yields +50% items.",
		Emoji:       "🎒", UnlockLevel: 12, DailyLimit: 3, CooldownMins: 180,
	},
	{
		ID: "reinforce", Name: "Reinforce",
		Description: "Next gathering has zero fail / collapse / escape risk.",
		Emoji:       "🏗️", UnlockLevel: 15, DailyLimit: 1, CooldownMins: 1440,
	},
	{
		ID: "nose_for_treasure", Name: "Nose for Treasure",
		Description: "Next gathering guarantees at least an Epic drop.",
		Emoji:       "🔍", UnlockLevel: 18, DailyLimit: 2, CooldownMins: 480,
	},
	{
		ID: "quick_learner", Name: "Quick Learner",
		Description: "Next activity gives 2x character XP.",
		Emoji:       "📚", UnlockLevel: 20, DailyLimit: 2, CooldownMins: 360,
	},
	{
		ID: "lucky_break", Name: "Lucky Break",
		Description: "Next slots or coinflip game: 2x win chance. Also deflects one bullet in Russian Roulette.",
		Emoji:       "🍀", UnlockLevel: 22, DailyLimit: 2, CooldownMins: 360,
	},
	{
		ID: "second_wind", Name: "Second Wind",
		Description: "Reset one daily game limit (extra casino, coinflip, etc. today).",
		Emoji:       "💨", UnlockLevel: 25, DailyLimit: 1, CooldownMins: 720,
	},
	{
		ID: "jackpot_fever", Name: "Jackpot Fever",
		Description: "Your next casino win pays out at 3x.",
		Emoji:       "🎰", UnlockLevel: 30, DailyLimit: 1, CooldownMins: 1440,
	},
	{
		ID: "golden_touch", Name: "Golden Touch",
		Description: "Next market sale gives 2x profit.",
		Emoji:       "✨", UnlockLevel: 32, DailyLimit: 2, CooldownMins: 240,
	},
	{
		ID: "efficiency", Name: "Efficiency",
		Description: "Next craft uses 50%% fewer materials.",
		Emoji:       "🔧", UnlockLevel: 35, DailyLimit: 2, CooldownMins: 240,
	},
	{
		ID: "insider_trading", Name: "Insider Trading",
		Description: "Market prices locked at maximum for 5 minutes.",
		Emoji:       "📊", UnlockLevel: 38, DailyLimit: 1, CooldownMins: 720,
	},
	{
		ID: "perfect_forge", Name: "Perfect Forge",
		Description: "Next craft is guaranteed top quality.",
		Emoji:       "⚒️", UnlockLevel: 42, DailyLimit: 1, CooldownMins: 480,
	},
	{
		ID: "midas_touch", Name: "Midas Touch",
		Description: "Next gathering yields only rare items.",
		Emoji:       "👑", UnlockLevel: 48, DailyLimit: 1, CooldownMins: 2880,
	},
}

var skillByID = func() map[string]Skill {
	m := make(map[string]Skill, len(allSkills))
	for _, s := range allSkills {
		m[s.ID] = s
	}
	return m
}()

// AllSkills returns every skill definition.
func AllSkills() []Skill {
	out := make([]Skill, len(allSkills))
	copy(out, allSkills)
	return out
}

// GetSkill returns a single skill by ID.
func GetSkill(id string) (Skill, bool) {
	s, ok := skillByID[id]
	return s, ok
}
