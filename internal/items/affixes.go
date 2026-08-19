package items

import "math/rand"

type AffixDef struct {
	ID     string
	Name   string
	Stat   string // "str", "dex", "int", "vit", "luk"
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
