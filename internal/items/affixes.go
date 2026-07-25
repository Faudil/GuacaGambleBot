package items

import "math/rand"

type AffixDef struct {
	ID     string
	Name   string
	Stat   string   // "str", "dex", "int", "vit", "luk"
	MinVal int
	MaxVal int
	MinRar Rarity   // minimum rarity this affix can appear at
	Slots  []string // nil = all slots
	Weight int      // relative roll weight
}

var affixCountByRarity = map[Rarity]int{
	RarityCommon:    0,
	RarityUncommon:  1,
	RarityRare:      2,
	RarityEpic:      3,
	RarityLegendary: 4,
}

// AffixPool is the full pool of rollable affixes.
var AffixPool = []AffixDef{
	// --- Common affixes (all slots) ---
	{ID: "of_the_bear", Name: "of the Bear", Stat: "str", MinVal: 1, MaxVal: 3, MinRar: RarityCommon, Weight: 10},
	{ID: "of_the_cat", Name: "of the Cat", Stat: "dex", MinVal: 1, MaxVal: 3, MinRar: RarityCommon, Weight: 10},
	{ID: "of_the_owl", Name: "of the Owl", Stat: "int", MinVal: 1, MaxVal: 3, MinRar: RarityCommon, Weight: 10},
	{ID: "of_the_turtle", Name: "of the Turtle", Stat: "vit", MinVal: 1, MaxVal: 3, MinRar: RarityCommon, Weight: 10},
	{ID: "of_the_rabbit", Name: "of the Rabbit", Stat: "luk", MinVal: 1, MaxVal: 3, MinRar: RarityCommon, Weight: 10},

	// --- Uncommon affixes ---
	{ID: "of_power", Name: "of Power", Stat: "str", MinVal: 3, MaxVal: 6, MinRar: RarityUncommon, Weight: 8},
	{ID: "of_swiftness", Name: "of Swiftness", Stat: "dex", MinVal: 3, MaxVal: 6, MinRar: RarityUncommon, Weight: 8},
	{ID: "of_wisdom", Name: "of Wisdom", Stat: "int", MinVal: 3, MaxVal: 6, MinRar: RarityUncommon, Weight: 8},
	{ID: "of_endurance", Name: "of Endurance", Stat: "vit", MinVal: 3, MaxVal: 6, MinRar: RarityUncommon, Weight: 8},
	{ID: "of_fortune", Name: "of Fortune", Stat: "luk", MinVal: 3, MaxVal: 6, MinRar: RarityUncommon, Weight: 8},
	{ID: "of_accuracy", Name: "of Accuracy", Stat: "dex", MinVal: 2, MaxVal: 5, MinRar: RarityUncommon, Slots: []string{"weapon"}, Weight: 6},
	{ID: "of_protection", Name: "of Protection", Stat: "vit", MinVal: 2, MaxVal: 5, MinRar: RarityUncommon, Slots: []string{"armor"}, Weight: 6},
	{ID: "of_the_scholar", Name: "of the Scholar", Stat: "int", MinVal: 2, MaxVal: 5, MinRar: RarityUncommon, Slots: []string{"accessory"}, Weight: 6},
	{ID: "of_gambling", Name: "of Gambling", Stat: "luk", MinVal: 2, MaxVal: 5, MinRar: RarityUncommon, Slots: []string{"accessory", "trinket"}, Weight: 6},

	// --- Rare affixes ---
	{ID: "of_the_giant", Name: "of the Giant", Stat: "str", MinVal: 5, MaxVal: 10, MinRar: RarityRare, Weight: 5},
	{ID: "of_the_wind", Name: "of the Wind", Stat: "dex", MinVal: 5, MaxVal: 10, MinRar: RarityRare, Weight: 5},
	{ID: "of_the_archmage", Name: "of the Archmage", Stat: "int", MinVal: 5, MaxVal: 10, MinRar: RarityRare, Weight: 5},
	{ID: "of_the_mountain", Name: "of the Mountain", Stat: "vit", MinVal: 5, MaxVal: 10, MinRar: RarityRare, Weight: 5},
	{ID: "of_the_stars", Name: "of the Stars", Stat: "luk", MinVal: 5, MaxVal: 10, MinRar: RarityRare, Weight: 5},
	{ID: "of_the_dragon", Name: "of the Dragon", Stat: "str", MinVal: 3, MaxVal: 7, MinRar: RarityRare, Slots: []string{"weapon"}, Weight: 4},
	{ID: "of_the_warden", Name: "of the Warden", Stat: "vit", MinVal: 3, MaxVal: 7, MinRar: RarityRare, Slots: []string{"armor"}, Weight: 4},
	{ID: "of_the_seer", Name: "of the Seer", Stat: "int", MinVal: 3, MaxVal: 7, MinRar: RarityRare, Slots: []string{"accessory", "trinket"}, Weight: 4},

	// --- Epic affixes ---
	{ID: "of_annihilation", Name: "of Annihilation", Stat: "str", MinVal: 8, MaxVal: 15, MinRar: RarityEpic, Slots: []string{"weapon"}, Weight: 3},
	{ID: "of_shadows", Name: "of Shadows", Stat: "dex", MinVal: 8, MaxVal: 15, MinRar: RarityEpic, Slots: []string{"armor", "trinket"}, Weight: 3},
	{ID: "of_the_void", Name: "of the Void", Stat: "int", MinVal: 8, MaxVal: 15, MinRar: RarityEpic, Slots: []string{"accessory", "trinket"}, Weight: 3},
	{ID: "of_immortality", Name: "of Immortality", Stat: "vit", MinVal: 8, MaxVal: 15, MinRar: RarityEpic, Weight: 3},
	{ID: "of_miracles", Name: "of Miracles", Stat: "luk", MinVal: 8, MaxVal: 15, MinRar: RarityEpic, Weight: 3},

	// --- Legendary affixes ---
	{ID: "of_the_titan", Name: "of the Titan", Stat: "str", MinVal: 12, MaxVal: 20, MinRar: RarityLegendary, Weight: 1},
	{ID: "of_eternity", Name: "of Eternity", Stat: "vit", MinVal: 12, MaxVal: 20, MinRar: RarityLegendary, Weight: 1},
	{ID: "of_the_chosen", Name: "of the Chosen", Stat: "str", MinVal: 5, MaxVal: 10, MinRar: RarityLegendary, Slots: []string{"weapon", "armor"}, Weight: 1},
	{ID: "of_the_chosen", Name: "of the Chosen", Stat: "dex", MinVal: 5, MaxVal: 10, MinRar: RarityLegendary, Slots: []string{"weapon", "armor"}, Weight: 1},
}

// RollAffixes picks random affixes for a given rarity and slot.
// Returns at most the number allowed by the rarity, with no duplicate stats.
func RollAffixes(rarity Rarity, slot string) []AffixDef {
	count := affixCountByRarity[rarity]
	if count <= 0 {
		return nil
	}

	var pool []AffixDef
	for _, a := range AffixPool {
		if a.MinRar > rarity {
			continue
		}
		if a.Slots != nil {
			ok := false
			for _, s := range a.Slots {
				if s == slot {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		pool = append(pool, a)
	}

	if len(pool) == 0 {
		return nil
	}

	usedStats := map[string]bool{}
	result := make([]AffixDef, 0, count)
	perm := rand.Perm(len(pool))
	for _, idx := range perm {
		a := pool[idx]
		if usedStats[a.Stat] {
			continue
		}
		usedStats[a.Stat] = true
		result = append(result, a)
		if len(result) >= count {
			break
		}
	}
	return result
}

// RollAffixValue rolls a random value within the affix's range.
func RollAffixValue(a AffixDef) int {
	if a.MaxVal <= a.MinVal {
		return a.MinVal
	}
	return a.MinVal + rand.Intn(a.MaxVal-a.MinVal+1)
}

type AppliedAffix struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Stat  string `json:"stat"`
	Value int    `json:"value"`
}
