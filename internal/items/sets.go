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
	// Procedural marks sets whose pieces are generated at runtime (e.g. the
	// delve zone sets, rolled by AssignSetName) instead of being fixed catalog
	// items. Procedural sets must not be checked for static catalog pieces.
	Procedural bool
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
