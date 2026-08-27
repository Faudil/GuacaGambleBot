package mining

import (
	"encoding/json"
	"errors"
	"math"
	"math/rand"
	"time"

	"gorm.io/gorm"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
	jobssvc "guacagamblebot/internal/service/jobs"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
)

var ErrMineLimit = errors.New("mining daily limit reached")

type MineItem struct {
	Name  string
	Value int
	Count int
}

// oreTable is the canonical mining ores sorted by value (catalog Price).
var oreTable = []MineItem{
	{Name: "pebble", Value: 1},
	{Name: "coal", Value: 5},
	{Name: "iron_ore", Value: 10},
	{Name: "copper_ore", Value: 15},
	{Name: "silver_ore", Value: 25},
	{Name: "gold_nugget", Value: 50},
	{Name: "platinum", Value: 75},
	{Name: "emerald", Value: 100},
	{Name: "rough_diamond", Value: 300},
	{Name: "ancient_alloy", Value: 500},
	{Name: "kethari_crystal", Value: 1000},
	{Name: "primordial_geode", Value: 2000},
	{Name: "resonance_core", Value: 5000},
}

var oreMinDepth = map[string]int{
	"pebble":           1,
	"coal":             1,
	"iron_ore":         2,
	"copper_ore":       3,
	"silver_ore":       6,
	"gold_nugget":      9,
	"platinum":         12,
	"emerald":          15,
	"rough_diamond":    20,
	"ancient_alloy":    24,
	"kethari_crystal":  30,
	"primordial_geode": 30,
	"resonance_core":   30,
}

// oreValueCurve returns the expected ore value at depth.
// Continuous exponential curve: ~5@d1, ~20@d4, ~63@d7, ~161@d9, ~368@d11, ~600@d12, ~2200@d15, >5000@d24.
func oreValueCurve(depth int) int {
	if depth < 1 {
		depth = 1
	}
	v := 5 * math.Pow(1.55, float64(depth-1))
	if v > 6000 {
		v = 6000
	}
	return int(v)
}

// eligibleOres returns ores whose depth gate is satisfied.
func eligibleOres(depth int) []MineItem {
	var out []MineItem
	for _, o := range oreTable {
		if oreMinDepth[o.Name] <= depth {
			out = append(out, o)
		}
	}
	if len(out) == 0 {
		return oreTable[:2]
	}
	return out
}

// rollOre procedurally picks an ore for depth with noise + rare tier shifts + quantity variance.
// Ores whose min depth is not yet reached are never eligible (e.g. diamond <20).
func rollOre(depth int) MineItem {
	target := float64(oreValueCurve(depth)) * (0.45 + rand.Float64()*1.15)
	// Rare tier shifts for overlap: jackpot up, poor vein down
	shift := rand.Float64()
	if shift < 0.03 {
		target *= 1.8
	} else if shift < 0.11 {
		target *= 0.55
	}
	eligible := eligibleOres(depth)
	best := eligible[0]
	bestDiff := math.Abs(float64(best.Value) - target)
	for _, o := range eligible[1:] {
		diff := math.Abs(float64(o.Value) - target)
		if diff < bestDiff {
			best = o
			bestDiff = diff
		}
	}
	count := 1
	if best.Value < 100 {
		count = 1 + rand.Intn(3)
	} else if best.Value < 300 {
		count = 1 + rand.Intn(2)
	}
	return MineItem{Name: best.Name, Value: best.Value, Count: count}
}

// lootAtDepth remains for backward compat / tests: returns nearby candidates around the curve.
func lootAtDepth(depth int) []MineItem {
	it := rollOre(depth)
	// Build a small candidate slice around the rolled tier for legacy callers.
	idx := 0
	for i, o := range oreTable {
		if o.Name == it.Name {
			idx = i
			break
		}
	}
	var out []MineItem
	for d := -1; d <= 1; d++ {
		j := idx + d
		if j < 0 || j >= len(oreTable) {
			continue
		}
		if oreMinDepth[oreTable[j].Name] > depth {
			continue
		}
		out = append(out, MineItem{Name: oreTable[j].Name, Value: oreTable[j].Value})
	}
	if len(out) == 0 {
		out = append(out, MineItem{Name: it.Name, Value: it.Value})
	}
	return out
}

const (
	dailyDescendLimit = 15

	steelPickaxeItem     = "steel_pickaxe"
	diamondDrillItem     = "diamond_drill"
	ghostVeilBuff        = "ghostly_veil"
	ancientCoreShardItem = "ancient_core_shard"
	ancientAlloyItem     = "ancient_alloy"

	loreKethari  = "mine_lore_kethari"
	loreEngine   = "mine_lore_engine"
	loreFracture = "mine_lore_fracture"
	loreKing     = "mine_lore_king"

	// NPCCharterEventID marks a narrative event that offers a mine charter
	// (contract) through an NPC encounter, in place of choosing one up front.
	NPCCharterEventID     = "npc_charter"
	charterOfferMinDepth  = 2
	charterOfferChancePct = 20
)

type ToolInfo struct {
	ItemID        string
	MinLevel      int
	LootTierBonus int
	RiskReduction int
	Durability    int
}

var miningTools = []ToolInfo{
	{ItemID: "", MinLevel: 1, LootTierBonus: 0, RiskReduction: 0, Durability: 0},
	{ItemID: steelPickaxeItem, MinLevel: 5, LootTierBonus: 1, RiskReduction: 5, Durability: 25},
	{ItemID: diamondDrillItem, MinLevel: 10, LootTierBonus: 2, RiskReduction: 10, Durability: 50},
}

func GetToolInfo(itemID string) ToolInfo {
	for _, t := range miningTools {
		if t.ItemID == itemID {
			return t
		}
	}
	return miningTools[0]
}

func (s *Service) OwnedTools(userID int64, level int) []ToolInfo {
	all := AvailableTools(level)
	var owned []ToolInfo
	for _, t := range all {
		if t.ItemID == "" {
			owned = append(owned, t)
			continue
		}
		has, _ := s.HasItem(userID, t.ItemID)
		if has {
			owned = append(owned, t)
		}
	}
	return owned
}

func LockedTools(level int) []ToolInfo {
	var out []ToolInfo
	for _, t := range miningTools {
		if t.MinLevel > level && t.ItemID != "" {
			out = append(out, t)
		}
	}
	return out
}

func AvailableTools(level int) []ToolInfo {
	var out []ToolInfo
	for _, t := range miningTools {
		if t.MinLevel <= level {
			out = append(out, t)
		}
	}
	return out
}

type LoreFragment struct {
	ID    string
	Title string
	Depth int
}

func (t ToolInfo) LocaleNameKey() string {
	switch t.ItemID {
	case steelPickaxeItem:
		return "mining.tool_name_steel"
	case diamondDrillItem:
		return "mining.tool_name_diamond"
	default:
		return "mining.tool_name_iron"
	}
}

func (t ToolInfo) LocaleDescKey() string {
	switch t.ItemID {
	case steelPickaxeItem:
		return "mining.tool_desc_steel"
	case diamondDrillItem:
		return "mining.tool_desc_diamond"
	default:
		return "mining.tool_desc_iron"
	}
}

func (t ToolInfo) Emoji() string {
	switch t.ItemID {
	case steelPickaxeItem:
		return "⛏️"
	case diamondDrillItem:
		return "🔧"
	default:
		return "🪨"
	}
}

var MiningLore = []LoreFragment{
	{ID: loreKethari, Title: "The Kethari — Children of Stone", Depth: 13},
	{ID: loreEngine, Title: "The Resonance Engine", Depth: 21},
	{ID: loreFracture, Title: "The Great Fracture", Depth: 33},
	{ID: loreKing, Title: "The Sleeping King", Depth: 42},
}

func DepthFlavorKey(depth int) string {
	switch {
	case depth <= 2:
		return "mining.flavor_tier1"
	case depth <= 4:
		return "mining.flavor_tier2"
	case depth <= 6:
		return "mining.flavor_tier3"
	case depth <= 9:
		return "mining.flavor_tier4"
	case depth <= 12:
		return "mining.flavor_tier5"
	case depth <= 15:
		return "mining.flavor_tier6"
	default:
		return "mining.flavor_tier7"
	}
}

func DepthColor(depth int) int {
	switch {
	case depth <= 3:
		return 0x4A90D9
	case depth <= 6:
		return 0x7B5EA7
	case depth <= 9:
		return 0x8B4513
	case depth <= 12:
		return 0x4A0E4E
	case depth <= 15:
		return 0x2C1810
	case depth <= 20:
		return 0x1A0A2E
	case depth <= 30:
		return 0x0A0000
	default:
		return 0x000000
	}
}

func LoreAtDepth(depth int) *LoreFragment {
	for _, l := range MiningLore {
		if l.Depth == depth {
			return &l
		}
	}
	return nil
}

type BagEntry struct {
	Name  string
	Count int
}

type MiningEvent struct {
	Type  string
	Items []BagEntry
	Buff  string
}

type EventRarity string

const (
	EventCommon    EventRarity = "common"
	EventRare      EventRarity = "rare"
	EventLegendary EventRarity = "legendary"
)

type EventStage int

const (
	StageShallow EventStage = 3
	StageDepth   EventStage = 9
	StageDeep    EventStage = 17
)

type NarrativeOption struct {
	Label  string
	Desc   string
	Effect *EventEffect
}

type EventEffect struct {
	Items      []BagEntry
	RiskMod    int
	RiskTurns  int
	ForceLeave bool
	LoreID     string
	DepthGain  int
	RemoveItem string
	// RequireItem gates the option behind an owned inventory item (e.g. a
	// magnet). Options whose requirement is not met are hidden from the embed.
	RequireItem string
	// ConsumeItem removes one unit of the inventory item when the option is
	// picked. The player is expected to own it (ownership is checked by the
	// cog before consuming).
	ConsumeItem string
	// RepairTool restores this many durability points to the active tool.
	RepairTool int
	Message    string
	MsgArgs    map[string]any
}

// ─── Contracts (in-run mine charters) ─────────────────────────────────────

type ContractType string

const (
	ContractReachDepth  ContractType = "reach_depth"
	ContractCollectGems ContractType = "collect_gems"
	ContractDigCount    ContractType = "dig_count"
	ContractEventCount  ContractType = "event_count"
)

type Contract struct {
	Type          ContractType `json:"type"`
	Target        int          `json:"target"`
	RewardCredits int          `json:"reward_credits"`
	RewardXP      int          `json:"reward_xp"`
}

func RollContracts() []Contract {
	pool := []Contract{
		{Type: ContractReachDepth, Target: 12, RewardCredits: 150, RewardXP: 20},
		{Type: ContractReachDepth, Target: 18, RewardCredits: 250, RewardXP: 35},
		{Type: ContractReachDepth, Target: 25, RewardCredits: 400, RewardXP: 50},
		{Type: ContractDigCount, Target: 8, RewardCredits: 120, RewardXP: 15},
		{Type: ContractDigCount, Target: 12, RewardCredits: 200, RewardXP: 25},
		{Type: ContractCollectGems, Target: 3, RewardCredits: 180, RewardXP: 20},
		{Type: ContractCollectGems, Target: 5, RewardCredits: 300, RewardXP: 35},
		{Type: ContractEventCount, Target: 2, RewardCredits: 150, RewardXP: 15},
		{Type: ContractEventCount, Target: 3, RewardCredits: 220, RewardXP: 25},
	}
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	out := make([]Contract, 0, 3)
	for i := 0; i < 3 && i < len(pool); i++ {
		out = append(out, pool[i])
	}
	return out
}

func ContractCompleted(c *Contract, depth int, bag []BagEntry, digs int, events int) bool {
	if c == nil {
		return false
	}
	switch c.Type {
	case ContractReachDepth:
		return depth >= c.Target
	case ContractDigCount:
		return digs >= c.Target
	case ContractCollectGems:
		cnt := 0
		for _, e := range bag {
			// gems = value >= 100
			price := 0
			for _, o := range oreTable {
				if o.Name == e.Name {
					price = o.Value
					break
				}
			}
			if price >= 100 {
				cnt += e.Count
			}
		}
		return cnt >= c.Target
	case ContractEventCount:
		return events >= c.Target
	}
	return false
}

func ContractProgress(c *Contract, depth int, bag []BagEntry, digs int, events int) (int, int) {
	if c == nil {
		return 0, 0
	}
	switch c.Type {
	case ContractReachDepth:
		if depth > c.Target {
			depth = c.Target
		}
		return depth, c.Target
	case ContractDigCount:
		if digs > c.Target {
			digs = c.Target
		}
		return digs, c.Target
	case ContractCollectGems:
		cnt := 0
		for _, e := range bag {
			price := 0
			for _, o := range oreTable {
				if o.Name == e.Name {
					price = o.Value
					break
				}
			}
			if price >= 100 {
				cnt += e.Count
			}
		}
		if cnt > c.Target {
			cnt = c.Target
		}
		return cnt, c.Target
	case ContractEventCount:
		if events > c.Target {
			events = c.Target
		}
		return events, c.Target
	}
	return 0, c.Target
}

type EventDef struct {
	ID       string
	Stage    EventStage
	Rarity   EventRarity
	MinDepth int
	Options  []NarrativeOption
}

type NarrativeEvent struct {
	ID      string
	Stage   EventStage
	Rarity  EventRarity
	Options []NarrativeOption
}

type DescendResult struct {
	Item           *MineItem
	Collapsed      bool
	Bag            []BagEntry
	Event          *MiningEvent
	NarrativeEvent *NarrativeEvent
	LoreID         string
	ToolBroke      bool
}

type LeaveResult struct {
	XP             int
	Bag            []BagEntry
	Unlocks        []*achievement.Achievement
	ToolID         string
	LeveledUp      bool
	NewLevel       int
	ContractDone   bool
	ContractReward *Contract
}

// eventPool holds all narrative events by rarity × stage.
// Add new events here to extend the system.
var eventPool = func() []EventDef {
	o := func(l, d string, e *EventEffect) NarrativeOption {
		return NarrativeOption{Label: l, Desc: d, Effect: e}
	}
	ef := func(msg string) *EventEffect { return &EventEffect{Message: msg} }
	efr := func(msg string, riskMod, turns int) *EventEffect {
		return &EventEffect{Message: msg, RiskMod: riskMod, RiskTurns: turns}
	}
	efi := func(msg string, items ...BagEntry) *EventEffect {
		return &EventEffect{Message: msg, Items: items}
	}
	return []EventDef{
		// ═══ SHALLOW — COMMON ═══
		{ID: "collapse", Stage: StageShallow, Rarity: EventCommon, MinDepth: 3,
			Options: []NarrativeOption{
				o("mining.ev_collapse_o1", "mining.ev_collapse_o1d", ef("mining.ev_collapse_r1")),
				o("mining.ev_collapse_o2", "mining.ev_collapse_o2d", efr("mining.ev_collapse_r2", -15, 3)),
			}},
		{ID: "lantern", Stage: StageShallow, Rarity: EventCommon, MinDepth: 4,
			Options: []NarrativeOption{
				o("mining.ev_lantern_o1", "mining.ev_lantern_o1d", efr("mining.ev_lantern_r1", -5, 10)),
				o("mining.ev_lantern_o2", "mining.ev_lantern_o2d", efi("mining.ev_lantern_r2", BagEntry{Name: "coal", Count: 2})),
				o("mining.ev_lantern_o3", "mining.ev_lantern_o3d", ef("mining.ev_lantern_r3")),
			}},
		{ID: "camp", Stage: StageShallow, Rarity: EventCommon, MinDepth: 5,
			Options: []NarrativeOption{
				o("mining.ev_camp_o1", "mining.ev_camp_o1d", efi("mining.ev_camp_r1", BagEntry{Name: "coal", Count: 1}, BagEntry{Name: "iron_ore", Count: 1})),
				o("mining.ev_camp_o2", "mining.ev_camp_o2d", efr("mining.ev_camp_r2", -20, 2)),
				o("mining.ev_camp_o3", "mining.ev_camp_o3d", ef("mining.ev_camp_r3")),
			}},
		{ID: "cart", Stage: StageShallow, Rarity: EventCommon, MinDepth: 5,
			Options: []NarrativeOption{
				o("mining.ev_cart_o1", "mining.ev_cart_o1d", efi("mining.ev_cart_r1", BagEntry{Name: "coal", Count: 3})),
				o("mining.ev_cart_o2", "mining.ev_cart_o2d", ef("mining.ev_cart_r2")),
			}},
		{ID: "fungus", Stage: StageShallow, Rarity: EventCommon, MinDepth: 5,
			Options: []NarrativeOption{
				o("mining.ev_fungus_o1", "mining.ev_fungus_o1d", efr("mining.ev_fungus_r1", -10, 3)),
				o("mining.ev_fungus_o2", "mining.ev_fungus_o2d", ef("mining.ev_fungus_r2")),
			}},

		// ═══ SHALLOW — RARE ═══
		{ID: "collapse_rare", Stage: StageShallow, Rarity: EventRare, MinDepth: 3,
			Options: []NarrativeOption{
				o("mining.ev_collapse_rare_o1", "mining.ev_collapse_rare_o1d",
					&EventEffect{RiskMod: -10, RiskTurns: 5, Items: []BagEntry{{Name: "iron_ore", Count: 2}}, Message: "mining.ev_collapse_rare_r1"}),
				o("mining.ev_collapse_rare_o2", "mining.ev_collapse_rare_o2d", ef("mining.ev_collapse_rare_r2")),
			}},
		{ID: "camp_rare", Stage: StageShallow, Rarity: EventRare, MinDepth: 5,
			Options: []NarrativeOption{
				o("mining.ev_camp_rare_o1", "mining.ev_camp_rare_o1d",
					&EventEffect{RiskMod: 10, RiskTurns: 5, Items: []BagEntry{{Name: "gold_nugget", Count: 1}, {Name: "copper_ore", Count: 2}}, Message: "mining.ev_camp_rare_r1"}),
				o("mining.ev_camp_rare_o2", "mining.ev_camp_rare_o2d", efr("mining.ev_camp_rare_r2", -30, 2)),
			}},
		{ID: "cart_rare", Stage: StageShallow, Rarity: EventRare, MinDepth: 5,
			Options: []NarrativeOption{
				o("mining.ev_cart_rare_o1", "mining.ev_cart_rare_o1d",
					&EventEffect{RiskMod: 10, RiskTurns: 5, Items: []BagEntry{{Name: "iron_ore", Count: 3}, {Name: "gold_nugget", Count: 1}}, Message: "mining.ev_cart_rare_r1"}),
				o("mining.ev_cart_rare_o2", "mining.ev_cart_rare_o2d",
					&EventEffect{RiskMod: 15, RiskTurns: 5, Items: []BagEntry{{Name: "ancient_alloy", Count: 1}}, Message: "mining.ev_cart_rare_r2"}),
			}},

		// ═══ SHALLOW — LEGENDARY ═══
		{ID: "collapse_legendary", Stage: StageShallow, Rarity: EventLegendary, MinDepth: 3,
			Options: []NarrativeOption{
				o("mining.ev_collapse_leg_o1", "mining.ev_collapse_leg_o1d",
					&EventEffect{RiskMod: 15, RiskTurns: 5, Items: []BagEntry{{Name: "ancient_alloy", Count: 1}}, Message: "mining.ev_collapse_leg_r1"}),
				o("mining.ev_collapse_leg_o2", "mining.ev_collapse_leg_o2d",
					&EventEffect{DepthGain: 2, RiskMod: 15, RiskTurns: 10, Items: []BagEntry{{Name: "kethari_crystal", Count: 1}}, Message: "mining.ev_collapse_leg_r2"}),
			}},
		{ID: "lantern_legendary", Stage: StageShallow, Rarity: EventLegendary, MinDepth: 4,
			Options: []NarrativeOption{
				o("mining.ev_lantern_leg_o1", "mining.ev_lantern_leg_o1d",
					&EventEffect{RiskMod: -10, RiskTurns: 10, Message: "mining.ev_lantern_leg_r1"}),
				o("mining.ev_lantern_leg_o2", "mining.ev_lantern_leg_o2d",
					&EventEffect{RiskMod: 15, RiskTurns: 10, Items: []BagEntry{{Name: "kethari_crystal", Count: 2}, {Name: "ancient_core_shard", Count: 1}}, Message: "mining.ev_lantern_leg_r2"}),
			}},

		// ═══ DEPTH — COMMON ═══
		{ID: "glow", Stage: StageDepth, Rarity: EventCommon, MinDepth: 9,
			Options: []NarrativeOption{
				o("mining.ev_glow_o1", "mining.ev_glow_o1d", &EventEffect{Message: "mining.ev_glow_r1", RiskMod: 5, RiskTurns: 3}),
				o("mining.ev_glow_o2", "mining.ev_glow_o2d", ef("mining.ev_glow_r2")),
			}},
		{ID: "river", Stage: StageDepth, Rarity: EventCommon, MinDepth: 9,
			Options: []NarrativeOption{
				o("mining.ev_river_o1", "mining.ev_river_o1d", ef("mining.ev_river_r1")),
				o("mining.ev_river_o2", "mining.ev_river_o2d", &EventEffect{DepthGain: 1, RiskMod: 10, RiskTurns: 2, Message: "mining.ev_river_r2"}),
			}},
		{ID: "gem", Stage: StageDepth, Rarity: EventCommon, MinDepth: 10,
			Options: []NarrativeOption{
				o("mining.ev_gem_o1", "mining.ev_gem_o1d",
					&EventEffect{Items: []BagEntry{{Name: "emerald", Count: 1}, {Name: "rough_diamond", Count: 1}, {Name: "gold_nugget", Count: 2}}, ForceLeave: true, Message: "mining.ev_gem_r1"}),
				o("mining.ev_gem_o2", "mining.ev_gem_o2d",
					&EventEffect{Items: []BagEntry{{Name: "emerald", Count: 1}, {Name: "gold_nugget", Count: 1}}, RiskMod: 15, RiskTurns: 3, Message: "mining.ev_gem_r2"}),
			}},
		{ID: "chasm", Stage: StageDepth, Rarity: EventCommon, MinDepth: 11,
			Options: []NarrativeOption{
				o("mining.ev_chasm_o1", "mining.ev_chasm_o1d", ef("mining.ev_chasm_r1")),
				o("mining.ev_chasm_o2", "mining.ev_chasm_o2d", &EventEffect{DepthGain: 2, RiskMod: 20, RiskTurns: 2, Message: "mining.ev_chasm_r2"}),
			}},

		// ═══ DEPTH — RARE ═══
		{ID: "glow_rare", Stage: StageDepth, Rarity: EventRare, MinDepth: 9,
			Options: []NarrativeOption{
				o("mining.ev_glow_rare_o1", "mining.ev_glow_rare_o1d",
					&EventEffect{RiskMod: 10, RiskTurns: 5, Items: []BagEntry{{Name: "kethari_crystal", Count: 1}}, Message: "mining.ev_glow_rare_r1"}),
				o("mining.ev_glow_rare_o2", "mining.ev_glow_rare_o2d", efr("mining.ev_glow_rare_r2", -10, 5)),
			}},
		{ID: "gem_rare", Stage: StageDepth, Rarity: EventRare, MinDepth: 10,
			Options: []NarrativeOption{
				o("mining.ev_gem_rare_o1", "mining.ev_gem_rare_o1d",
					&EventEffect{Items: []BagEntry{{Name: "emerald", Count: 2}, {Name: "rough_diamond", Count: 2}, {Name: "ancient_alloy", Count: 1}}, ForceLeave: true, Message: "mining.ev_gem_rare_r1"}),
				o("mining.ev_gem_rare_o2", "mining.ev_gem_rare_o2d",
					&EventEffect{Items: []BagEntry{{Name: "emerald", Count: 2}, {Name: "platinum", Count: 2}}, RiskMod: 20, RiskTurns: 3, Message: "mining.ev_gem_rare_r2"}),
			}},
		{ID: "river_rare", Stage: StageDepth, Rarity: EventRare, MinDepth: 9,
			Options: []NarrativeOption{
				o("mining.ev_river_rare_o1", "mining.ev_river_rare_o1d", efr("mining.ev_river_rare_r1", -20, 5)),
				o("mining.ev_river_rare_o2", "mining.ev_river_rare_o2d",
					&EventEffect{RiskMod: 10, RiskTurns: 5, Items: []BagEntry{{Name: "gold_nugget", Count: 3}, {Name: "emerald", Count: 1}}, Message: "mining.ev_river_rare_r2"}),
			}},

		// ═══ DEPTH — LEGENDARY ═══
		{ID: "glow_legendary", Stage: StageDepth, Rarity: EventLegendary, MinDepth: 9,
			Options: []NarrativeOption{
				o("mining.ev_glow_leg_o1", "mining.ev_glow_leg_o1d",
					&EventEffect{RiskMod: 15, RiskTurns: 10, Items: []BagEntry{{Name: "kethari_crystal", Count: 2}, {Name: "ancient_alloy", Count: 1}}, Message: "mining.ev_glow_leg_r1"}),
				o("mining.ev_glow_leg_o2", "mining.ev_glow_leg_o2d",
					&EventEffect{RiskMod: 15, RiskTurns: 10, Items: []BagEntry{{Name: "resonance_core", Count: 1}}, Message: "mining.ev_glow_leg_r2"}),
			}},
		{ID: "chasm_legendary", Stage: StageDepth, Rarity: EventLegendary, MinDepth: 11,
			Options: []NarrativeOption{
				o("mining.ev_chasm_leg_o1", "mining.ev_chasm_leg_o1d",
					&EventEffect{RiskMod: 15, RiskTurns: 10, Items: []BagEntry{{Name: "rough_diamond", Count: 3}, {Name: "kethari_crystal", Count: 1}}, Message: "mining.ev_chasm_leg_r1"}),
				o("mining.ev_chasm_leg_o2", "mining.ev_chasm_leg_o2d",
					&EventEffect{DepthGain: 3, Items: []BagEntry{{Name: "primordial_geode", Count: 1}}, RiskMod: 15, RiskTurns: 5, Message: "mining.ev_chasm_leg_r2"}),
			}},

		// ═══ DEEP — COMMON ═══
		{ID: "crystal", Stage: StageDeep, Rarity: EventCommon, MinDepth: 15,
			Options: []NarrativeOption{
				o("mining.ev_crystal_o1", "mining.ev_crystal_o1d", efr("mining.ev_crystal_r1", -10, 3)),
				o("mining.ev_crystal_o2", "mining.ev_crystal_o2d", ef("mining.ev_crystal_r2")),
			}},
		{ID: "shrine", Stage: StageDeep, Rarity: EventCommon, MinDepth: 17,
			Options: []NarrativeOption{
				o("mining.ev_shrine_o1", "mining.ev_shrine_o1d",
					&EventEffect{RiskMod: -15, RiskTurns: 5, RemoveItem: "random", Message: "mining.ev_shrine_r1"}),
				o("mining.ev_shrine_o2", "mining.ev_shrine_o2d", ef("mining.ev_shrine_r2")),
				o("mining.ev_shrine_o3", "mining.ev_shrine_o3d", ef("mining.ev_shrine_r3")),
			}},
		{ID: "void", Stage: StageDeep, Rarity: EventCommon, MinDepth: 18,
			Options: []NarrativeOption{
				o("mining.ev_void_o1", "mining.ev_void_o1d", ef("mining.ev_void_r1")),
				o("mining.ev_void_o2", "mining.ev_void_o2d", &EventEffect{RiskMod: 15, RiskTurns: 3, Message: "mining.ev_void_r2"}),
			}},

		// ═══ DEEP — RARE ═══
		{ID: "magnet_hoard", Stage: StageDeep, Rarity: EventRare, MinDepth: 15,
			Options: []NarrativeOption{
				o("mining.ev_magnet_hoard_o1", "mining.ev_magnet_hoard_o1d",
					&EventEffect{RequireItem: "rusty_magnet", ConsumeItem: "rusty_magnet", Message: "mining.ev_magnet_hoard_r1"}),
				o("mining.ev_magnet_hoard_o2", "mining.ev_magnet_hoard_o2d",
					&EventEffect{RequireItem: "magnet", ConsumeItem: "magnet", Message: "mining.ev_magnet_hoard_r2"}),
				o("mining.ev_magnet_hoard_o3", "mining.ev_magnet_hoard_o3d",
					&EventEffect{RequireItem: "electric_magnet", ConsumeItem: "electric_magnet", Message: "mining.ev_magnet_hoard_r3"}),
				o("mining.ev_magnet_hoard_o4", "mining.ev_magnet_hoard_o4d", ef("mining.ev_magnet_hoard_r4")),
			}},
		{ID: "crystal_rare", Stage: StageDeep, Rarity: EventRare, MinDepth: 15,
			Options: []NarrativeOption{
				o("mining.ev_crystal_rare_o1", "mining.ev_crystal_rare_o1d",
					&EventEffect{RiskMod: 10, RiskTurns: 5, Items: []BagEntry{{Name: "kethari_crystal", Count: 1}}, Message: "mining.ev_crystal_rare_r1"}),
				o("mining.ev_crystal_rare_o2", "mining.ev_crystal_rare_o2d",
					&EventEffect{RiskMod: 15, RiskTurns: 5, Items: []BagEntry{{Name: "ancient_alloy", Count: 1}}, Message: "mining.ev_crystal_rare_r2"}),
			}},
		{ID: "void_rare", Stage: StageDeep, Rarity: EventRare, MinDepth: 18,
			Options: []NarrativeOption{
				o("mining.ev_void_rare_o1", "mining.ev_void_rare_o1d", ef("mining.ev_void_rare_r1")),
				o("mining.ev_void_rare_o2", "mining.ev_void_rare_o2d",
					&EventEffect{RiskMod: -25, RiskTurns: 5, LoreID: "mine_lore_kethari", Message: "mining.ev_void_rare_r2"}),
			}},
		{ID: "guardian_rare", Stage: StageDeep, Rarity: EventRare, MinDepth: 19,
			Options: []NarrativeOption{
				o("mining.ev_guardian_rare_o1", "mining.ev_guardian_rare_o1d",
					&EventEffect{RiskMod: 15, RiskTurns: 10, Items: []BagEntry{{Name: "ancient_core_shard", Count: 1}, {Name: "ancient_alloy", Count: 3}}, Message: "mining.ev_guardian_rare_r1"}),
				o("mining.ev_guardian_rare_o2", "mining.ev_guardian_rare_o2d",
					&EventEffect{RiskMod: -30, RiskTurns: 10, Message: "mining.ev_guardian_rare_r2"}),
			}},

		// ═══ DEEP — LEGENDARY ═══
		{ID: "shrine_legendary", Stage: StageDeep, Rarity: EventLegendary, MinDepth: 17,
			Options: []NarrativeOption{
				o("mining.ev_shrine_leg_o1", "mining.ev_shrine_leg_o1d",
					&EventEffect{Items: []BagEntry{{Name: "resonance_core", Count: 1}, {Name: "kethari_crystal", Count: 2}}, RiskMod: 15, RiskTurns: 10, Message: "mining.ev_shrine_leg_r1"}),
				o("mining.ev_shrine_leg_o2", "mining.ev_shrine_leg_o2d",
					&EventEffect{RiskMod: 10, RiskTurns: 10, LoreID: "mine_lore_engine", Items: []BagEntry{{Name: "ancient_core_shard", Count: 1}}, Message: "mining.ev_shrine_leg_r2"}),
				o("mining.ev_shrine_leg_o3", "mining.ev_shrine_leg_o3d",
					&EventEffect{RiskMod: 20, RiskTurns: 5, Items: []BagEntry{{Name: "primordial_geode", Count: 1}}, Message: "mining.ev_shrine_leg_r3"}),
			}},
		{ID: "crystal_legendary", Stage: StageDeep, Rarity: EventLegendary, MinDepth: 15,
			Options: []NarrativeOption{
				o("mining.ev_crystal_leg_o1", "mining.ev_crystal_leg_o1d",
					&EventEffect{RiskMod: -15, RiskTurns: 10, Message: "mining.ev_crystal_leg_r1"}),
				o("mining.ev_crystal_leg_o2", "mining.ev_crystal_leg_o2d",
					&EventEffect{RiskMod: 15, RiskTurns: 10, Items: []BagEntry{{Name: "kethari_crystal", Count: 3}, {Name: "ancient_alloy", Count: 2}, {Name: "resonance_core", Count: 1}}, Message: "mining.ev_crystal_leg_r2"}),
			}},

		// ═══ NEW — SHALLOW COMMON ═══
		{ID: "spider_nest", Stage: StageShallow, Rarity: EventCommon, MinDepth: 3,
			Options: []NarrativeOption{
				o("mining.ev_spider_o1", "mining.ev_spider_o1d", &EventEffect{RiskMod: -10, RiskTurns: 3, Message: "mining.ev_spider_r1"}),
				o("mining.ev_spider_o2", "mining.ev_spider_o2d", &EventEffect{RemoveItem: "random", RiskMod: 10, RiskTurns: 3, Message: "mining.ev_spider_r2"}),
			}},
		{ID: "ore_spring", Stage: StageShallow, Rarity: EventCommon, MinDepth: 4,
			Options: []NarrativeOption{
				o("mining.ev_ore_spring_o1", "mining.ev_ore_spring_o1d", &EventEffect{Items: []BagEntry{{Name: "copper_ore", Count: 2}, {Name: "silver_ore", Count: 1}}, Message: "mining.ev_ore_spring_r1"}),
				o("mining.ev_ore_spring_o2", "mining.ev_ore_spring_o2d", &EventEffect{Items: []BagEntry{{Name: "coal", Count: 3}}, RiskMod: -10, RiskTurns: 2, Message: "mining.ev_ore_spring_r2"}),
			}},
		{ID: "dugout", Stage: StageShallow, Rarity: EventCommon, MinDepth: 4,
			Options: []NarrativeOption{
				o("mining.ev_dugout_o1", "mining.ev_dugout_o1d", &EventEffect{Items: []BagEntry{{Name: "iron_ore", Count: 2}}, RepairTool: 5, Message: "mining.ev_dugout_r1"}),
				o("mining.ev_dugout_o2", "mining.ev_dugout_o2d", efr("mining.ev_dugout_r2", -15, 3)),
			}},

		// ═══ NEW — SHALLOW RARE ═══
		{ID: "old_dynamite", Stage: StageShallow, Rarity: EventRare, MinDepth: 5,
			Options: []NarrativeOption{
				o("mining.ev_dynamite_o1", "mining.ev_dynamite_o1d", &EventEffect{RiskMod: -10, RiskTurns: 5, Items: []BagEntry{{Name: "iron_ore", Count: 2}}, Message: "mining.ev_dynamite_r1"}),
				o("mining.ev_dynamite_o2", "mining.ev_dynamite_o2d", &EventEffect{DepthGain: 2, RiskMod: 20, RiskTurns: 3, Items: []BagEntry{{Name: "gold_nugget", Count: 2}}, Message: "mining.ev_dynamite_r2"}),
			}},
		{ID: "goblin_trader", Stage: StageShallow, Rarity: EventRare, MinDepth: 5,
			Options: []NarrativeOption{
				o("mining.ev_goblin_o1", "mining.ev_goblin_o1d", &EventEffect{Message: "mining.ev_goblin_r1"}),
				o("mining.ev_goblin_o2", "mining.ev_goblin_o2d", ef("mining.ev_goblin_r2")),
			}},

		// ═══ NEW — DEPTH COMMON ═══
		{ID: "geode_cluster", Stage: StageDepth, Rarity: EventCommon, MinDepth: 10,
			Options: []NarrativeOption{
				o("mining.ev_geode_o1", "mining.ev_geode_o1d", &EventEffect{Items: []BagEntry{{Name: "emerald", Count: 1}, {Name: "platinum", Count: 1}}, Message: "mining.ev_geode_r1"}),
				o("mining.ev_geode_o2", "mining.ev_geode_o2d", &EventEffect{RiskMod: 10, RiskTurns: 3, Items: []BagEntry{{Name: "rough_diamond", Count: 1}}, Message: "mining.ev_geode_r2"}),
			}},
		{ID: "fossil_bed", Stage: StageDepth, Rarity: EventCommon, MinDepth: 11,
			Options: []NarrativeOption{
				o("mining.ev_fossil_o1", "mining.ev_fossil_o1d", &EventEffect{Items: []BagEntry{{Name: "gold_nugget", Count: 3}, {Name: "platinum", Count: 1}}, Message: "mining.ev_fossil_r1"}),
				o("mining.ev_fossil_o2", "mining.ev_fossil_o2d", &EventEffect{Items: []BagEntry{{Name: "ancient_alloy", Count: 1}}, LoreID: "mine_lore_fracture", Message: "mining.ev_fossil_r2"}),
			}},

		// ═══ NEW — DEPTH RARE ═══
		{ID: "kethari_pump", Stage: StageDepth, Rarity: EventRare, MinDepth: 10,
			Options: []NarrativeOption{
				o("mining.ev_pump_o1", "mining.ev_pump_o1d", &EventEffect{RiskMod: -20, RiskTurns: 5, Message: "mining.ev_pump_r1"}),
				o("mining.ev_pump_o2", "mining.ev_pump_o2d", &EventEffect{Items: []BagEntry{{Name: "ancient_alloy", Count: 1}, {Name: "kethari_crystal", Count: 1}}, RiskMod: 15, RiskTurns: 5, Message: "mining.ev_pump_r2"}),
			}},
		{ID: "mushroom_forest", Stage: StageDepth, Rarity: EventRare, MinDepth: 10,
			Options: []NarrativeOption{
				o("mining.ev_mushroom_o1", "mining.ev_mushroom_o1d", &EventEffect{RiskMod: -15, RiskTurns: 3, Items: []BagEntry{{Name: "emerald", Count: 1}}, Message: "mining.ev_mushroom_r1"}),
				o("mining.ev_mushroom_o2", "mining.ev_mushroom_o2d", &EventEffect{Items: []BagEntry{{Name: "gold_nugget", Count: 2}}, RemoveItem: "random", Message: "mining.ev_mushroom_r2"}),
			}},

		// ═══ NEW — DEEP COMMON ═══
		{ID: "gas_pocket", Stage: StageDeep, Rarity: EventCommon, MinDepth: 16,
			Options: []NarrativeOption{
				o("mining.ev_gas_o1", "mining.ev_gas_o1d", efr("mining.ev_gas_r1", -10, 3)),
				o("mining.ev_gas_o2", "mining.ev_gas_o2d", &EventEffect{RiskMod: 20, RiskTurns: 5, Message: "mining.ev_gas_r2"}),
			}},

		// ═══ NEW — DEEP RARE ═══
		{ID: "dice_ghost", Stage: StageDeep, Rarity: EventRare, MinDepth: 17,
			Options: []NarrativeOption{
				o("mining.ev_dice_o1", "mining.ev_dice_o1d", &EventEffect{Message: "mining.ev_dice_r1"}),
				o("mining.ev_dice_o2", "mining.ev_dice_o2d", efr("mining.ev_dice_r2", -15, 5)),
			}},
		{ID: "anvil", Stage: StageDeep, Rarity: EventRare, MinDepth: 16,
			Options: []NarrativeOption{
				o("mining.ev_anvil_o1", "mining.ev_anvil_o1d", &EventEffect{RepairTool: 15, Message: "mining.ev_anvil_r1"}),
				o("mining.ev_anvil_o2", "mining.ev_anvil_o2d", &EventEffect{Items: []BagEntry{{Name: "ancient_alloy", Count: 2}}, Message: "mining.ev_anvil_r2"}),
			}},

		// ═══ NEW — DEEP LEGENDARY ═══
		{ID: "vault_lock", Stage: StageDeep, Rarity: EventLegendary, MinDepth: 18,
			Options: []NarrativeOption{
				o("mining.ev_vault_o1", "mining.ev_vault_o1d", &EventEffect{Items: []BagEntry{{Name: "resonance_core", Count: 1}}, Message: "mining.ev_vault_r1"}),
				o("mining.ev_vault_o2", "mining.ev_vault_o2d", &EventEffect{RiskMod: 20, RiskTurns: 5, Items: []BagEntry{{Name: "kethari_crystal", Count: 2}}, Message: "mining.ev_vault_r2"}),
				o("mining.ev_vault_o3", "mining.ev_vault_o3d", ef("mining.ev_vault_r3")),
			}},
		{ID: "rift_tear", Stage: StageDeep, Rarity: EventLegendary, MinDepth: 19,
			Options: []NarrativeOption{
				o("mining.ev_rift_o1", "mining.ev_rift_o1d", &EventEffect{RiskMod: 15, RiskTurns: 10, Items: []BagEntry{{Name: "resonance_core", Count: 1}, {Name: "primordial_geode", Count: 1}}, Message: "mining.ev_rift_r1"}),
				o("mining.ev_rift_o2", "mining.ev_rift_o2d", &EventEffect{RiskMod: -20, RiskTurns: 5, LoreID: "mine_lore_king", Message: "mining.ev_rift_r2"}),
			}},
	}
}()

func getStage(depth int) EventStage {
	switch {
	case depth >= 17:
		return StageDeep
	case depth >= 9:
		return StageDepth
	default:
		return StageShallow
	}
}

func rollRarity() EventRarity {
	r := rand.Intn(100)
	switch {
	case r < 60:
		return EventCommon
	case r < 90:
		return EventRare
	default:
		return EventLegendary
	}
}

func rollEventChance(depth int) int {
	switch {
	case depth >= 17:
		chance := 30 + (depth-17)*2
		if chance > 50 {
			return 50
		}
		return chance
	case depth >= 9:
		chance := 20 + (depth-9)*3
		if chance > 45 {
			return 45
		}
		return chance
	default:
		return 20
	}
}

func pickNarrativeEvent(depth int) *NarrativeEvent {
	stage := getStage(depth)
	rarity := rollRarity()

	for _, r := range []EventRarity{rarity, EventCommon, EventRare, EventLegendary} {
		var candidates []EventDef
		for _, e := range eventPool {
			if e.Stage == stage && e.Rarity == r && depth >= e.MinDepth {
				candidates = append(candidates, e)
			}
		}
		if len(candidates) > 0 {
			ev := candidates[rand.Intn(len(candidates))]
			return &NarrativeEvent{
				ID:      ev.ID,
				Stage:   ev.Stage,
				Rarity:  ev.Rarity,
				Options: ev.Options,
			}
		}
	}
	return nil
}

func lookupEventDef(eventID string) *EventDef {
	for _, e := range eventPool {
		if e.ID == eventID {
			return &e
		}
	}
	return nil
}

func (s *Service) ApplyEventOption(eventID string, optionIdx int, depth int, bag []BagEntry) *EventEffect {
	edef := lookupEventDef(eventID)
	if edef == nil || optionIdx < 0 || optionIdx >= len(edef.Options) {
		return &EventEffect{Message: "mining.event_none"}
	}
	eff := *edef.Options[optionIdx].Effect

	// Shrine common "touch" mystery effect
	if eventID == "shrine" && optionIdx == 1 {
		r := rand.Intn(100)
		switch {
		case r < 30:
			eff = EventEffect{RiskMod: -15, RiskTurns: 5, Message: "mining.ev_shrine_r2a"}
		case r < 60:
			eff = EventEffect{RiskMod: 20, RiskTurns: 5, Message: "mining.ev_shrine_r2b"}
		case r < 80:
			eff = EventEffect{Message: "mining.ev_shrine_r2c"}
		default:
			eff = EventEffect{Message: "mining.ev_shrine_r2d"}
		}
	}

	// Glow common "investigate" 50/50 reward — procedural ore
	if eventID == "glow" && optionIdx == 0 && rand.Intn(100) < 50 {
		it := rollOre(depth)
		eff.Items = append(eff.Items, BagEntry{Name: it.Name, Count: it.Count})
		eff.Message = "mining.ev_glow_r1g"
	}

	// Goblin trader: 45% win ore, else lose
	if eventID == "goblin_trader" && optionIdx == 0 {
		if rand.Intn(100) < 45 {
			if rand.Intn(100) < 50 {
				eff.Items = []BagEntry{{Name: "gold_nugget", Count: 2}, {Name: "emerald", Count: 1}}
			} else {
				eff.Items = []BagEntry{{Name: "silver_ore", Count: 3}, {Name: "platinum", Count: 1}}
			}
			eff.Message = "mining.ev_goblin_r1_win"
		} else {
			eff.Items = nil
			eff.Message = "mining.ev_goblin_r1_lose"
		}
	}

	// Dice ghost: 50/50 ore or lose
	if eventID == "dice_ghost" && optionIdx == 0 {
		if rand.Intn(100) < 50 {
			eff.Items = []BagEntry{{Name: "gold_nugget", Count: 2}, {Name: "emerald", Count: 1}}
			eff.Message = "mining.ev_dice_r1_win"
		} else {
			eff.Items = nil
			eff.RiskMod = 10
			eff.RiskTurns = 3
			eff.Message = "mining.ev_dice_r1_lose"
		}
	}

	// Vault lock: 40% success, else lose
	if eventID == "vault_lock" && optionIdx == 0 {
		if rand.Intn(100) < 40 {
			eff.Message = "mining.ev_vault_r1_win"
		} else {
			eff.Items = nil
			eff.Message = "mining.ev_vault_r1_lose"
		}
	}

	return &eff
}

type Service struct {
	store  *store.Store
	cfg    *config.Config
	npcSvc *npcsvc.Service
}

func New(s *store.Store, cfg *config.Config, npcSvc *npcsvc.Service) *Service {
	return &Service{store: s, cfg: cfg, npcSvc: npcSvc}
}

func (s *Service) GetMinerLevel(userID int64) (int, error) {
	var job model.Job
	err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "miner").First(&job).Error
	if err == gorm.ErrRecordNotFound {
		return 1, nil
	}
	if err != nil {
		return 1, err
	}
	return job.Level, nil
}

func (s *Service) HasLore(userID int64, loreID string) (bool, error) {
	var count int64
	err := s.store.DB.Model(&model.UserLoreEntry{}).
		Where("user_id = ? AND lore_id = ?", userID, loreID).
		Count(&count).Error
	return count > 0, err
}

func (s *Service) GrantLore(userID int64, loreID string) error {
	return s.store.DB.Create(&model.UserLoreEntry{
		UserID: userID,
		LoreID: loreID,
	}).Error
}

func (s *Service) GrantItem(userID int64, itemID string, quantity int) error {
	return s.store.AddItemRaw(s.store.DB, userID, itemID, quantity)
}

func (s *Service) HasItem(userID int64, itemID string) (bool, error) {
	canonical := items.Canonical(itemID)
	if canonical == "" {
		return false, nil
	}
	var inv model.Inventory
	err := s.store.DB.Where("user_id = ? AND item_id = ? AND quantity > 0", userID, canonical).First(&inv).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// toolDurabilityMax returns how many digs a tool lasts, or 0 for the base tool.
func toolDurabilityMax(itemID string) int {
	if it := items.Get(itemID); it != nil {
		return it.Durability
	}
	return 0
}

// ToolDurability returns the remaining digs of the user's active tool stack.
// Zero means the tool is absent or the base (free) tool is in use.
func (s *Service) ToolDurability(userID int64, toolID string) int {
	if toolID == "" {
		return 0
	}
	max := toolDurabilityMax(toolID)
	if max <= 0 {
		return 0
	}
	var inv model.Inventory
	err := s.store.DB.Where("user_id = ? AND item_id = ? AND quantity > 0", userID, toolID).
		First(&inv).Error
	if err != nil {
		return 0
	}
	if inv.Durability <= 0 {
		return max
	}
	return inv.Durability
}

// ConsumeToolDurability uses one dig of the active tool. When the tool breaks,
// a single unit is removed from the stack (or the row when it was the last
// one) and broke is reported so the session can fall back to the base tool.
// Legacy rows with zero durability are lazily initialized to a full tool.
func (s *Service) ConsumeToolDurability(userID int64, toolID string) (bool, error) {
	if toolID == "" {
		return false, nil
	}
	max := toolDurabilityMax(toolID)
	if max <= 0 {
		return false, nil
	}
	var inv model.Inventory
	err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, toolID).First(&inv).Error
	if err == gorm.ErrRecordNotFound {
		// The tool is gone (sold/traded mid-session): fall back to base tool.
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if inv.Quantity <= 0 {
		return true, nil
	}

	durability := inv.Durability
	if durability <= 0 {
		durability = max
	}
	durability--

	broke := durability <= 0
	if !broke {
		err = s.store.DB.Model(&model.Inventory{}).
			Where("user_id = ? AND item_id = ?", userID, toolID).
			UpdateColumn("durability", durability).Error
		return false, err
	}

	// The active tool shattered: consume one unit and start a fresh one, or
	// delete the row when the stack is empty.
	if inv.Quantity > 1 {
		err = s.store.DB.Model(&model.Inventory{}).
			Where("user_id = ? AND item_id = ?", userID, toolID).
			Updates(map[string]any{"quantity": inv.Quantity - 1, "durability": max}).Error
	} else {
		err = s.store.DB.Where("user_id = ? AND item_id = ?", userID, toolID).
			Delete(&model.Inventory{}).Error
	}
	return true, err
}

// EnterMine reserves one daily expedition entry for the user. Returns
// ErrMineLimit when the player has already started today's quota of
// expeditions. Digs within an expedition do not consume quota.
func (s *Service) EnterMine(userID int64) error {
	ok, _, err := s.store.CheckGameLimit(userID, "mine_descend", dailyDescendLimit)
	if err != nil {
		return err
	}
	if !ok {
		return ErrMineLimit
	}
	return s.store.IncrementGameLimit(userID, "mine_descend")
}

// RemainingEntries returns how many expedition entries the user still has
// available today.
func (s *Service) RemainingEntries(userID int64) (int, error) {
	_, remaining, err := s.store.CheckGameLimit(userID, "mine_descend", dailyDescendLimit)
	return remaining, err
}

// RiskFor returns the collapse chance (0-100) the next dig would have for the
// user at the given risk level, tool and active effects. riskUnits accumulates
// in half-steps: a normal descend (going one depth deeper) adds 2 units (the
// full +5% per depth), while staying in place to prospect again adds only 1
// unit (half that risk increase). Descend uses the same math so the displayed
// risk always matches the actual roll.
func (s *Service) RiskFor(userID int64, riskUnits int, toolID string, ghostVeilTurns, riskMod int) int {
	ml, err := s.GetMinerLevel(userID)
	if err != nil {
		ml = 1
	}

	ti := GetToolInfo(toolID)
	if ti.MinLevel > ml {
		ti = GetToolInfo("")
	}

	risk := riskUnits*5/2 - ti.RiskReduction + riskMod
	risk -= int(float64(ml) * 1.5)
	risk -= int(charsvc.GetVITReduction(s.store, userID) * 100)
	if charsvc.HasPassive(s.store, userID, "perk_collapse_resist") {
		risk -= 5
	}

	if ghostVeilTurns > 0 {
		risk -= 10
	}
	if charsvc.HasBuff(s.store, userID, "reinforce") {
		risk = 0
	}
	if risk < 0 {
		risk = 0
	}
	return risk
}

func (s *Service) Descend(userID int64, depth, riskUnits int, bag []BagEntry, toolID string, ghostVeilTurns, riskMod int, hasContract, charterDeclined bool) (*DescendResult, error) {
	free, err := s.store.FreeSlots(s.store.DB, userID)
	if err != nil {
		return nil, err
	}
	if free <= 0 {
		return nil, store.ErrInventoryFull
	}

	ml, err := s.GetMinerLevel(userID)
	if err != nil {
		return nil, err
	}

	ti := GetToolInfo(toolID)
	if ti.MinLevel > ml {
		toolID = ""
		ti = GetToolInfo("")
	}

	risk := s.RiskFor(userID, riskUnits, toolID, ghostVeilTurns, riskMod)
	if charsvc.HasBuff(s.store, userID, "reinforce") {
		charsvc.ConsumeBuff(s.store, userID, "reinforce")
	}

	roll := rand.Intn(100) + 1
	hiddenChamber := false
	if roll <= risk && rand.Intn(100) < 10 {
		hiddenChamber = true
		risk = 0
		roll = risk + 1
	}

	if roll <= risk {
		return &DescendResult{Collapsed: true, Bag: bag}, nil
	}

	// A successful dig wears the tool down. If it shatters here the session
	// falls back to the base tool for the rest of the expedition.
	toolBroke, err := s.ConsumeToolDurability(userID, toolID)
	if err != nil {
		return nil, err
	}

	var easterEgg *MiningEvent
	eRoll := rand.Intn(100) + 1
	if depth >= 10 && eRoll <= 1 {
		_ = s.GrantItem(userID, ancientAlloyItem, 1)
		easterEgg = &MiningEvent{Type: "ancient_forge", Items: []BagEntry{{Name: ancientAlloyItem, Count: 1}}}
		s.npcSvc.AddActivityReputation(userID, "mining", 5)
	} else if depth >= 6 && eRoll <= 3 {
		easterEgg = &MiningEvent{Type: "ghost_miner", Buff: ghostVeilBuff}
	} else if depth >= 7 && eRoll <= 6 && !hiddenChamber {
		it := rollOre(depth)
		easterEgg = &MiningEvent{Type: "whispering_runes", Items: []BagEntry{{Name: it.Name, Count: it.Count}}}
		found := false
		for i, e := range bag {
			if e.Name == it.Name {
				bag[i].Count += it.Count
				found = true
				break
			}
		}
		if !found {
			bag = append(bag, BagEntry{Name: it.Name, Count: it.Count})
		}
	}

	var loreID string
	if lf := LoreAtDepth(depth); lf != nil {
		has, err := s.HasLore(userID, lf.ID)
		if err == nil && !has {
			_ = s.GrantLore(userID, lf.ID)
			if lf.ID == loreKethari {
				_ = s.GrantItem(userID, ancientCoreShardItem, 1)
			}
			loreID = lf.ID
		}
	}

	if hiddenChamber {
		bonusCount := 2 + rand.Intn(4)
		var chamberItems []BagEntry
		for i := 0; i < bonusCount; i++ {
			it := rollOre(depth)
			found := false
			for j, e := range chamberItems {
				if e.Name == it.Name {
					chamberItems[j].Count += it.Count
					found = true
					break
				}
			}
			if !found {
				chamberItems = append(chamberItems, BagEntry{Name: it.Name, Count: it.Count})
			}
			found2 := false
			for j, e := range bag {
				if e.Name == it.Name {
					bag[j].Count += it.Count
					found2 = true
					break
				}
			}
			if !found2 {
				bag = append(bag, BagEntry{Name: it.Name, Count: it.Count})
			}
		}
		easterEgg = &MiningEvent{Type: "hidden_chamber", Items: chamberItems}
		s.npcSvc.AddActivityReputation(userID, "mining", 3)
		return &DescendResult{Item: nil, Bag: bag, Event: easterEgg, LoreID: loreID, ToolBroke: toolBroke}, nil
	}

	lvl := depth + ti.LootTierBonus
	if lvl < 1 {
		lvl = 1
	}
	item := rollOre(lvl)
	// Midas / nose buffs: re-roll until threshold if needed
	if charsvc.HasBuff(s.store, userID, "midas_touch") {
		if item.Value < 50 {
			for tries := 0; tries < 5; tries++ {
				cand := rollOre(lvl)
				if cand.Value >= 50 {
					item = cand
					break
				}
			}
		}
		charsvc.ConsumeBuff(s.store, userID, "midas_touch")
	} else if charsvc.HasBuff(s.store, userID, "nose_for_treasure") {
		if item.Value < 100 {
			for tries := 0; tries < 5; tries++ {
				cand := rollOre(lvl)
				if cand.Value >= 100 {
					item = cand
					break
				}
			}
		}
		charsvc.ConsumeBuff(s.store, userID, "nose_for_treasure")
	}

	found := false
	for i, e := range bag {
		if e.Name == item.Name {
			bag[i].Count += item.Count
			found = true
			break
		}
	}
	if !found {
		bag = append(bag, BagEntry{Name: item.Name, Count: item.Count})
	}

	var nEvent *NarrativeEvent
	offerCharter := !hasContract && !charterDeclined && depth >= charterOfferMinDepth &&
		rand.Intn(100) < charterOfferChancePct
	if offerCharter {
		nEvent = &NarrativeEvent{ID: NPCCharterEventID, Stage: getStage(depth), Rarity: EventCommon}
	} else if rand.Intn(100) < rollEventChance(depth) {
		nEvent = pickNarrativeEvent(depth)
	}

	if item.Value >= 50 {
		s.npcSvc.AddActivityReputation(userID, "mining", 3)
	} else {
		s.npcSvc.AddActivityReputation(userID, "mining", 1)
	}

	return &DescendResult{Item: &item, Bag: bag, Event: easterEgg, NarrativeEvent: nEvent, LoreID: loreID, ToolBroke: toolBroke}, nil
}

func (s *Service) LeaveMine(userID int64, bag []BagEntry, toolID string) (*LeaveResult, error) {
	strMult := charsvc.GetSTRBonus(s.store, userID)
	if charsvc.HasPassive(s.store, userID, "perk_mine_yield") {
		strMult += 0.05
	}
	if strMult > 1.0 {
		for i := range bag {
			bag[i].Count = int(float64(bag[i].Count) * strMult)
		}
	}

	if charsvc.HasBuff(s.store, userID, "scavenger") {
		for i := range bag {
			bag[i].Count += bag[i].Count / 2
		}
		charsvc.ConsumeBuff(s.store, userID, "scavenger")
	}

	totalXP := len(bag) * 10
	for _, e := range bag {
		totalXP += e.Count * 5
	}

	for _, e := range bag {
		if err := s.store.AddItemRaw(s.store.DB, userID, e.Name, e.Count); err != nil {
			return nil, err
		}
	}

	if len(bag) > 0 {
		if err := achievement.IncrementStat(s.store.DB, userID, "items_mined", len(bag)); err != nil {
			return nil, err
		}
		if err := s.store.RecordActivity(userID, "items_mined", len(bag)); err != nil {
			return nil, err
		}
	}

	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "miner").First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			job = model.Job{UserID: userID, JobName: "miner", Level: 1, XP: totalXP}
			if err := s.store.DB.Create(&job).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		job.XP += totalXP
		next := jobssvc.XPForLevel(job.Level)
		for job.XP >= next {
			job.XP -= next
			job.Level++
			next = jobssvc.XPForLevel(job.Level)
		}
		if err := s.store.DB.Model(&model.Job{}).Where("user_id = ? AND job_name = ?", userID, "miner").
			Updates(map[string]any{"xp": job.XP, "level": job.Level}).Error; err != nil {
			return nil, err
		}
	}

	leveled, lvl := charsvc.AddXP(s.store, userID, totalXP)

	unlocks, err := achievement.CheckAndUnlock(s.store.DB, userID)
	if err != nil {
		return nil, err
	}

	return &LeaveResult{XP: totalXP, Bag: bag, Unlocks: unlocks, ToolID: toolID, LeveledUp: leveled, NewLevel: lvl}, nil
}

func LoreDisplayName(loreID string) string {
	for _, l := range MiningLore {
		if l.ID == loreID {
			return l.Title
		}
	}
	return loreID
}

func GhostVeilBuffID() string        { return ghostVeilBuff }
func LoreKethariID() string          { return loreKethari }
func AncientCoreShardItemID() string { return ancientCoreShardItem }

// StaleSessionTimeout is how long a persisted expedition may stay idle before
// it is considered abandoned. Abandoned bags are auto-granted so loot is never
// silently lost across restarts.
const StaleSessionTimeout = 2 * time.Hour

// PersistedSession is the session state that survives bot restarts. The bag is
// serialized to JSON inside the DB row.
type PersistedSession struct {
	Depth           int
	ToolID          string
	GhostVeilTurns  int
	RiskMod         int
	RiskTurns       int
	Bag             []BagEntry
	Contract        *Contract
	EventCount      int
	CharterDeclined bool
	RiskUnits       int
}

// RepairTool restores durability to the active tool.
func (s *Service) RepairTool(userID int64, toolID string, amount int) error {
	if toolID == "" || amount <= 0 {
		return nil
	}
	max := toolDurabilityMax(toolID)
	if max <= 0 {
		return nil
	}
	var inv model.Inventory
	err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, toolID).First(&inv).Error
	if err != nil {
		return nil
	}
	dur := inv.Durability
	if dur <= 0 {
		dur = max
	}
	dur += amount
	if dur > max {
		dur = max
	}
	return s.store.DB.Model(&model.Inventory{}).Where("user_id = ? AND item_id = ?", userID, toolID).UpdateColumn("durability", dur).Error
}

// SaveSession persists an in-progress expedition for the user.
func (s *Service) SaveSession(userID int64, ps *PersistedSession) error {
	bagJSON, err := json.Marshal(ps.Bag)
	if err != nil {
		return err
	}
	contractJSON := ""
	if ps.Contract != nil {
		b, err := json.Marshal(ps.Contract)
		if err == nil {
			contractJSON = string(b)
		}
	}
	return s.store.SaveMiningSession(&model.MiningSession{
		UserID:          userID,
		Depth:           ps.Depth,
		ToolID:          ps.ToolID,
		GhostVeilTurns:  ps.GhostVeilTurns,
		RiskMod:         ps.RiskMod,
		RiskTurns:       ps.RiskTurns,
		Bag:             string(bagJSON),
		Contract:        contractJSON,
		CharterDeclined: ps.CharterDeclined,
		RiskUnits:       ps.RiskUnits,
		UpdatedAt:       time.Now(),
	})
}

// LoadSession restores a persisted expedition, or returns nil when there is
// none. A session idle for longer than StaleSessionTimeout is considered
// abandoned: its loot is granted through LeaveMine (STR/scavenger bonuses, XP
// and tool consumption all apply) and the row is removed.
func (s *Service) LoadSession(userID int64) (*PersistedSession, error) {
	m, err := s.store.GetMiningSession(userID)
	if err != nil || m == nil {
		return nil, err
	}
	if time.Since(m.UpdatedAt) > StaleSessionTimeout {
		// Delete the row first so two concurrent loads cannot double-grant
		// the abandoned bag.
		if err := s.store.DeleteMiningSession(userID); err != nil {
			return nil, err
		}
		var bag []BagEntry
		if m.Bag != "" {
			if err := json.Unmarshal([]byte(m.Bag), &bag); err != nil {
				return nil, err
			}
		}
		var contract *Contract
		if m.Contract != "" {
			var c Contract
			if err := json.Unmarshal([]byte(m.Contract), &c); err == nil {
				contract = &c
			}
		}
		// Grant contract reward if completed on stale auto-grant
		if contract != nil && ContractCompleted(contract, m.Depth, bag, m.Depth-1, 0) {
			_, _ = s.store.UpdateBalance(userID, contract.RewardCredits)
			_, _ = charsvc.AddXP(s.store, userID, contract.RewardXP)
		}
		if _, err := s.LeaveMine(userID, bag, m.ToolID); err != nil {
			return nil, err
		}
		return nil, nil
	}
	var bag []BagEntry
	if m.Bag != "" {
		if err := json.Unmarshal([]byte(m.Bag), &bag); err != nil {
			return nil, err
		}
	}
	var contract *Contract
	if m.Contract != "" {
		var c Contract
		if err := json.Unmarshal([]byte(m.Contract), &c); err == nil {
			contract = &c
		}
	}
	riskUnits := m.RiskUnits
	if riskUnits == 0 && m.Depth > 1 {
		// Backfills sessions persisted before risk_units existed, so an
		// in-progress expedition doesn't suddenly look risk-free.
		riskUnits = 2 * (m.Depth - 1)
	}
	return &PersistedSession{
		Depth:           m.Depth,
		ToolID:          m.ToolID,
		GhostVeilTurns:  m.GhostVeilTurns,
		RiskMod:         m.RiskMod,
		RiskTurns:       m.RiskTurns,
		Bag:             bag,
		Contract:        contract,
		EventCount:      0,
		CharterDeclined: m.CharterDeclined,
		RiskUnits:       riskUnits,
	}, nil
}

// DeleteSession removes the persisted expedition for the user.
func (s *Service) DeleteSession(userID int64) error {
	return s.store.DeleteMiningSession(userID)
}
