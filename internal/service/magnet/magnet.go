package magnet

import "math/rand"

// PoolEntry is a weighted ore entry in a magnet pull table.
type PoolEntry struct {
	ItemID string
	Weight int
}

// Tier is the quality tier of a magnet item.
type Tier int

const (
	TierRusty Tier = iota + 1
	TierBasic
	TierElectric
)

// TierOf resolves a magnet item ID to its tier, if it is one.
func TierOf(itemID string) (Tier, bool) {
	switch itemID {
	case "rusty_magnet":
		return TierRusty, true
	case "magnet":
		return TierBasic, true
	case "electric_magnet":
		return TierElectric, true
	}
	return 0, false
}

// standalonePools are the modest ore tables used when a magnet is used with
// the /use command: the payout sits just above the craft cost.
var standalonePools = map[Tier][]PoolEntry{
	TierRusty: {
		{ItemID: "coal", Weight: 1},
		{ItemID: "iron_ore", Weight: 3},
		{ItemID: "copper_ore", Weight: 3},
		{ItemID: "silver_ore", Weight: 2},
		{ItemID: "gold_nugget", Weight: 1},
	},
	TierBasic: {
		{ItemID: "silver_ore", Weight: 4},
		{ItemID: "gold_nugget", Weight: 2},
		{ItemID: "platinum", Weight: 1},
		{ItemID: "emerald", Weight: 1},
		{ItemID: "rough_diamond", Weight: 1},
	},
	TierElectric: {
		{ItemID: "platinum", Weight: 3},
		{ItemID: "emerald", Weight: 2},
		{ItemID: "rough_diamond", Weight: 1},
		{ItemID: "ancient_alloy", Weight: 1},
		{ItemID: "kethari_crystal", Weight: 1},
	},
}

// eventPools are the richer ore tables granted when a magnet is spent inside a
// gameplay event (mine, farm, delve). The jackpot ores only drop here.
var eventPools = map[Tier][]PoolEntry{
	TierRusty: {
		{ItemID: "coal", Weight: 2},
		{ItemID: "iron_ore", Weight: 3},
		{ItemID: "copper_ore", Weight: 3},
		{ItemID: "silver_ore", Weight: 2},
		{ItemID: "gold_nugget", Weight: 2},
	},
	TierBasic: {
		{ItemID: "silver_ore", Weight: 3},
		{ItemID: "gold_nugget", Weight: 3},
		{ItemID: "platinum", Weight: 2},
		{ItemID: "emerald", Weight: 2},
		{ItemID: "rough_diamond", Weight: 1},
	},
	TierElectric: {
		{ItemID: "platinum", Weight: 3},
		{ItemID: "emerald", Weight: 2},
		{ItemID: "rough_diamond", Weight: 2},
		{ItemID: "ancient_alloy", Weight: 2},
		{ItemID: "kethari_crystal", Weight: 1},
		{ItemID: "primordial_geode", Weight: 1},
	},
}

func standaloneCount(t Tier) int {
	switch t {
	case TierRusty:
		return 2
	case TierBasic:
		return 2
	default:
		return 3
	}
}

func eventCount(t Tier) int {
	switch t {
	case TierRusty:
		return 3
	case TierBasic:
		return 4
	default:
		return 5
	}
}

func pick(pool []PoolEntry) string {
	total := 0
	for _, e := range pool {
		total += e.Weight
	}
	r := rand.Intn(total)
	for _, e := range pool {
		if r < e.Weight {
			return e.ItemID
		}
		r -= e.Weight
	}
	return pool[len(pool)-1].ItemID
}

// Pull draws the standalone ore haul for the given magnet item. It returns
// nil when itemID is not a magnet.
func Pull(itemID string) []string {
	t, ok := TierOf(itemID)
	if !ok {
		return nil
	}
	pool := standalonePools[t]
	out := make([]string, 0, standaloneCount(t))
	for i := 0; i < standaloneCount(t); i++ {
		out = append(out, pick(pool))
	}
	return out
}

// EventPull draws the richer ore haul granted when a magnet is spent inside a
// gameplay event (mine, farm, delve). It returns nil when itemID is not a
// magnet.
func EventPull(itemID string) []string {
	t, ok := TierOf(itemID)
	if !ok {
		return nil
	}
	pool := eventPools[t]
	out := make([]string, 0, eventCount(t))
	for i := 0; i < eventCount(t); i++ {
		out = append(out, pick(pool))
	}
	return out
}

// ItemIDs lists every magnet item ID in ascending order of power.
func ItemIDs() []string {
	return []string{"rusty_magnet", "magnet", "electric_magnet"}
}

// BestOwned returns the strongest magnet the player owns according to the
// owns callback, or "" when they own none.
func BestOwned(owns func(itemID string) bool) string {
	ids := ItemIDs()
	for i := len(ids) - 1; i >= 0; i-- {
		if owns(ids[i]) {
			return ids[i]
		}
	}
	return ""
}
