package items

// SetTier defines the bonus granted at a given piece threshold.
type SetTier struct {
	Pieces  int
	StatSTR int
	StatDEX int
	StatINT int
	StatVIT int
	StatLUK int
	Desc    string // e.g. "2pc: +5 STR, +3 VIT"
}

// SetDef defines a named equipment set.
type SetDef struct {
	ID      string
	Name    string
	Emoji   string
	Bonuses []SetTier
}

// SetsByName indexes set definitions by their ID.
var SetsByName = map[string]SetDef{
	"dragon_slayer": {
		ID:    "dragon_slayer",
		Name:  "Dragon Slayer",
		Emoji: "🐉",
		Bonuses: []SetTier{
			{Pieces: 2, StatSTR: 5, StatVIT: 3, Desc: "2pc: +5 STR, +3 VIT"},
			{Pieces: 4, StatSTR: 10, StatVIT: 5, StatLUK: 3, Desc: "4pc: +10 STR, +5 VIT, +3 LUK"},
		},
	},
	"shadow_stalker": {
		ID:    "shadow_stalker",
		Name:  "Shadow Stalker",
		Emoji: "🌑",
		Bonuses: []SetTier{
			{Pieces: 2, StatDEX: 4, StatLUK: 3, Desc: "2pc: +4 DEX, +3 LUK"},
			{Pieces: 4, StatDEX: 8, StatLUK: 6, Desc: "4pc: +8 DEX, +6 LUK"},
		},
	},
	"arcane_weaver": {
		ID:    "arcane_weaver",
		Name:  "Arcane Weaver",
		Emoji: "🔮",
		Bonuses: []SetTier{
			{Pieces: 2, StatINT: 5, StatDEX: 3, Desc: "2pc: +5 INT, +3 DEX"},
			{Pieces: 4, StatINT: 10, StatDEX: 5, Desc: "4pc: +10 INT, +5 DEX"},
		},
	},
	"rift_walker": {
		ID:    "rift_walker",
		Name:  "Rift Walker",
		Emoji: "🔮",
		Bonuses: []SetTier{
			{Pieces: 2, StatSTR: 8, StatVIT: 8, StatLUK: 5, Desc: "2pc: +8 STR, +8 VIT, +5 LUK"},
			{Pieces: 4, StatSTR: 15, StatDEX: 15, StatINT: 15, StatVIT: 15, StatLUK: 10, Desc: "4pc: +15 ALL, +10 LUK"},
		},
	},
}

// PieceCountBySet returns the number of equipped items per set ID.
type EquippedSetInfo struct {
	SetID      string
	SetName    string
	SetEmoji   string
	Pieces     int
	ActiveTier *SetTier
}

// CalculateSetBonuses aggregates set bonuses from a list of equipped set IDs.
// Returns total stat bonuses and per-set info for display.
func CalculateSetBonuses(equippedSets []string) (str, dex, intt, vit, luk int, infos []EquippedSetInfo) {
	counts := map[string]int{}
	for _, sid := range equippedSets {
		if sid == "" {
			continue
		}
		counts[sid]++
	}
	for sid, cnt := range counts {
		set, ok := SetsByName[sid]
		if !ok {
			continue
		}
		info := EquippedSetInfo{SetID: sid, SetName: set.Name, SetEmoji: set.Emoji, Pieces: cnt}
		for _, tier := range set.Bonuses {
			if cnt >= tier.Pieces {
				if info.ActiveTier == nil || tier.Pieces > info.ActiveTier.Pieces {
					info.ActiveTier = &SetTier{
						Pieces:  tier.Pieces,
						StatSTR: tier.StatSTR,
						StatDEX: tier.StatDEX,
						StatINT: tier.StatINT,
						StatVIT: tier.StatVIT,
						StatLUK: tier.StatLUK,
						Desc:    tier.Desc,
					}
				}
				str += tier.StatSTR
				dex += tier.StatDEX
				intt += tier.StatINT
				vit += tier.StatVIT
				luk += tier.StatLUK
			}
		}
		infos = append(infos, info)
	}
	return
}
