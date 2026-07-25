package pets

import (
	"math/rand"
	"time"

	"guacagamblebot/internal/battle"
)

type WeeklyModifierDef struct {
	ID          string
	Name        string
	Emoji       string
	Description string
	BattleApply func(p1, p2 *battle.BattlePet)
	Boosted     []string
	Nerfed      []string
}

var WeeklyModifiers = []WeeklyModifierDef{
	{
		ID: "burning_sun", Name: "Burning Sun", Emoji: "☀️",
		Description: "Fire dmg +40%. All pets start with 1 burn turn.",
		BattleApply: func(p1, p2 *battle.BattlePet) {
			p1.PerkInt["mod_burning_sun"] = 1
			p2.PerkInt["mod_burning_sun"] = 1
			p1.PerkInt["mod_start_burn"] = 1
			p2.PerkInt["mod_start_burn"] = 1
		},
		Boosted: []string{"impact", "rejuvenation"},
		Nerfed:  []string{"resilience", "warding"},
	},
	{
		ID: "heavy_rain", Name: "Heavy Rain", Emoji: "🌧️",
		Description: "Dodge -50%. All speed -10%.",
		BattleApply: func(p1, p2 *battle.BattlePet) {
			p1.PerkInt["mod_heavy_rain"] = 1
			p2.PerkInt["mod_heavy_rain"] = 1
		},
		Boosted: []string{"piercing", "precision"},
		Nerfed:  []string{"fortune", "haste"},
	},
	{
		ID: "starlight", Name: "Starlight", Emoji: "✨",
		Description: "Crit chance +15%. Special effects always crit.",
		BattleApply: func(p1, p2 *battle.BattlePet) {
			p1.PerkInt["mod_starlight"] = 1
			p2.PerkInt["mod_starlight"] = 1
		},
		Boosted: []string{"fortune", "might"},
		Nerfed:  []string{"resilience", "vampirism"},
	},
	{
		ID: "iron_will", Name: "Iron Will", Emoji: "🛡️",
		Description: "Defense +30%. Attack -15%.",
		BattleApply: func(p1, p2 *battle.BattlePet) {
			p1.PerkInt["mod_iron_will"] = 1
			p2.PerkInt["mod_iron_will"] = 1
			p1.Defense = int(float64(p1.Defense) * 1.30)
			p2.Defense = int(float64(p2.Defense) * 1.30)
		},
		Boosted: []string{"resilience", "warding"},
		Nerfed:  []string{"impact", "might"},
	},
	{
		ID: "blood_moon", Name: "Blood Moon", Emoji: "🌕",
		Description: "Lifesteal +50%. No passive HP regen.",
		BattleApply: func(p1, p2 *battle.BattlePet) {
			p1.PerkInt["mod_blood_moon"] = 1
			p2.PerkInt["mod_blood_moon"] = 1
		},
		Boosted: []string{"vampirism", "impact"},
		Nerfed:  []string{"rejuvenation", "resilience"},
	},
	{
		ID: "thunderstorm", Name: "Thunderstorm", Emoji: "⛈️",
		Description: "Speed +20%. Slow pets (<15 spd) take +20% dmg.",
		BattleApply: func(p1, p2 *battle.BattlePet) {
			p1.PerkInt["mod_thunderstorm"] = 1
			p2.PerkInt["mod_thunderstorm"] = 1
		},
		Boosted: []string{"haste", "piercing"},
		Nerfed:  []string{"resilience", "warding"},
	},
	{
		ID: "shadow_realm", Name: "Shadow Realm", Emoji: "🌑",
		Description: "Accuracy -25%. Dodging heals 8% HP.",
		BattleApply: func(p1, p2 *battle.BattlePet) {
			p1.PerkInt["mod_shadow_realm"] = 1
			p2.PerkInt["mod_shadow_realm"] = 1
		},
		Boosted: []string{"rejuvenation", "precision"},
		Nerfed:  []string{"fortune", "might"},
	},
	{
		ID: "rampage", Name: "Rampage", Emoji: "💢",
		Description: "When <40% HP: +30% dmg, -20% def.",
		BattleApply: func(p1, p2 *battle.BattlePet) {
			p1.PerkInt["mod_rampage"] = 1
			p2.PerkInt["mod_rampage"] = 1
		},
		Boosted: []string{"might", "impact"},
		Nerfed:  []string{"warding", "rejuvenation"},
	},
	{
		ID: "frost_aura", Name: "Frost Aura", Emoji: "❄️",
		Description: "+15 def for all. Status effects -1 turn.",
		BattleApply: func(p1, p2 *battle.BattlePet) {
			p1.Defense += 15
			p2.Defense += 15
			p1.PerkInt["mod_frost_aura"] = 1
			p2.PerkInt["mod_frost_aura"] = 1
		},
		Boosted: []string{"resilience", "warding"},
		Nerfed:  []string{"vampirism", "haste"},
	},
	{
		ID: "chaos", Name: "Chaos", Emoji: "🌀",
		Description: "Each turn, a random stat fluctuates ±20%.",
		BattleApply: func(p1, p2 *battle.BattlePet) {
			p1.PerkInt["mod_chaos"] = 1
			p2.PerkInt["mod_chaos"] = 1
		},
		Boosted: []string{"impact", "fortune"},
		Nerfed:  []string{"resilience", "warding"},
	},
}

var byModID = func() map[string]*WeeklyModifierDef {
	m := make(map[string]*WeeklyModifierDef, len(WeeklyModifiers))
	for i := range WeeklyModifiers {
		m[WeeklyModifiers[i].ID] = &WeeklyModifiers[i]
	}
	return m
}()

func GetModifierDef(id string) *WeeklyModifierDef {
	return byModID[id]
}

func RandomModifier() *WeeklyModifierDef {
	return &WeeklyModifiers[rand.Intn(len(WeeklyModifiers))]
}

func ApplyModifierToBattle(p1, p2 *battle.BattlePet, modID string) {
	mod := GetModifierDef(modID)
	if mod == nil {
		return
	}
	mod.BattleApply(p1, p2)
}

func IsBoostedStat(modID, statID string) bool {
	mod := GetModifierDef(modID)
	if mod == nil {
		return false
	}
	for _, s := range mod.Boosted {
		if s == statID {
			return true
		}
	}
	return false
}

func IsNerfedStat(modID, statID string) bool {
	mod := GetModifierDef(modID)
	if mod == nil {
		return false
	}
	for _, s := range mod.Nerfed {
		if s == statID {
			return true
		}
	}
	return false
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}
