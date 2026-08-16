package mining

import (
	"encoding/json"
	"errors"
	"math/rand"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
)

var ErrMineLimit = errors.New("mining daily limit reached")

type MineItem struct {
	Name  string
	Value int
}

func lootAtDepth(depth int) []MineItem {
	switch {
	case depth <= 1:
		return []MineItem{{"pebble", 1}, {"coal", 5}}
	case depth <= 3:
		return []MineItem{{"coal", 5}, {"iron_ore", 10}, {"copper_ore", 15}}
	case depth == 4:
		return []MineItem{{"copper_ore", 15}, {"silver_ore", 25}, {"gold_nugget", 50}}
	case depth == 5:
		return []MineItem{{"silver_ore", 25}, {"gold_nugget", 50}, {"emerald", 100}}
	case depth <= 7:
		return []MineItem{{"gold_nugget", 50}, {"platinum", 75}, {"emerald", 100}}
	case depth <= 9:
		return []MineItem{{"platinum", 75}, {"emerald", 100}, {"rough_diamond", 300}}
	case depth <= 14:
		return []MineItem{{"emerald", 100}, {"rough_diamond", 300}, {"ancient_alloy", 500}}
	case depth <= 19:
		return []MineItem{{"rough_diamond", 300}, {"ancient_alloy", 500}, {"kethari_crystal", 1000}}
	case depth <= 24:
		return []MineItem{{"ancient_alloy", 500}, {"kethari_crystal", 1000}, {"primordial_geode", 2000}}
	case depth <= 29:
		return []MineItem{{"kethari_crystal", 1000}, {"primordial_geode", 2000}, {"resonance_core", 5000}}
	default:
		return []MineItem{{"primordial_geode", 2000}, {"resonance_core", 5000}}
	}
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
)

type ToolInfo struct {
	ItemID        string
	MinLevel      int
	LootTierBonus int
	RiskReduction int
}

var miningTools = []ToolInfo{
	{ItemID: "", MinLevel: 1, LootTierBonus: 0, RiskReduction: 0},
	{ItemID: steelPickaxeItem, MinLevel: 5, LootTierBonus: 1, RiskReduction: 5},
	{ItemID: diamondDrillItem, MinLevel: 10, LootTierBonus: 2, RiskReduction: 10},
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
	Message    string
	MsgArgs    map[string]any
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
}

type LeaveResult struct {
	XP        int
	Bag       []BagEntry
	Unlocks   []*achievement.Achievement
	ToolID    string
	LeveledUp bool
	NewLevel  int
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
				o("mining.ev_lantern_o1", "mining.ev_lantern_o1d", efr("mining.ev_lantern_r1", -5, 99)),
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
					efi("mining.ev_camp_rare_r1", BagEntry{Name: "gold_nugget", Count: 1}, BagEntry{Name: "copper_ore", Count: 2})),
				o("mining.ev_camp_rare_o2", "mining.ev_camp_rare_o2d", efr("mining.ev_camp_rare_r2", -30, 3)),
			}},
		{ID: "cart_rare", Stage: StageShallow, Rarity: EventRare, MinDepth: 5,
			Options: []NarrativeOption{
				o("mining.ev_cart_rare_o1", "mining.ev_cart_rare_o1d",
					efi("mining.ev_cart_rare_r1", BagEntry{Name: "iron_ore", Count: 3}, BagEntry{Name: "gold_nugget", Count: 1})),
				o("mining.ev_cart_rare_o2", "mining.ev_cart_rare_o2d",
					&EventEffect{RiskMod: -15, RiskTurns: 5, Items: []BagEntry{{Name: "ancient_alloy", Count: 1}}, Message: "mining.ev_cart_rare_r2"}),
			}},

		// ═══ SHALLOW — LEGENDARY ═══
		{ID: "collapse_legendary", Stage: StageShallow, Rarity: EventLegendary, MinDepth: 3,
			Options: []NarrativeOption{
				o("mining.ev_collapse_leg_o1", "mining.ev_collapse_leg_o1d",
					&EventEffect{RiskMod: -20, RiskTurns: 5, Items: []BagEntry{{Name: "ancient_alloy", Count: 1}}, Message: "mining.ev_collapse_leg_r1"}),
				o("mining.ev_collapse_leg_o2", "mining.ev_collapse_leg_o2d",
					&EventEffect{DepthGain: 2, Items: []BagEntry{{Name: "kethari_crystal", Count: 1}}, Message: "mining.ev_collapse_leg_r2"}),
			}},
		{ID: "lantern_legendary", Stage: StageShallow, Rarity: EventLegendary, MinDepth: 4,
			Options: []NarrativeOption{
				o("mining.ev_lantern_leg_o1", "mining.ev_lantern_leg_o1d",
					&EventEffect{RiskMod: -10, RiskTurns: 99, Items: []BagEntry{{Name: "ancient_core_shard", Count: 1}}, Message: "mining.ev_lantern_leg_r1"}),
				o("mining.ev_lantern_leg_o2", "mining.ev_lantern_leg_o2d",
					&EventEffect{Items: []BagEntry{{Name: "kethari_crystal", Count: 2}}, Message: "mining.ev_lantern_leg_r2"}),
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
					efi("mining.ev_glow_rare_r1", BagEntry{Name: "kethari_crystal", Count: 1})),
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
					&EventEffect{Items: []BagEntry{{Name: "gold_nugget", Count: 3}, {Name: "emerald", Count: 1}}, Message: "mining.ev_river_rare_r2"}),
			}},

		// ═══ DEPTH — LEGENDARY ═══
		{ID: "glow_legendary", Stage: StageDepth, Rarity: EventLegendary, MinDepth: 9,
			Options: []NarrativeOption{
				o("mining.ev_glow_leg_o1", "mining.ev_glow_leg_o1d",
					&EventEffect{Items: []BagEntry{{Name: "kethari_crystal", Count: 2}, {Name: "ancient_alloy", Count: 1}}, Message: "mining.ev_glow_leg_r1"}),
				o("mining.ev_glow_leg_o2", "mining.ev_glow_leg_o2d",
					&EventEffect{RiskMod: -15, RiskTurns: 99, Items: []BagEntry{{Name: "resonance_core", Count: 1}}, Message: "mining.ev_glow_leg_r2"}),
			}},
		{ID: "chasm_legendary", Stage: StageDepth, Rarity: EventLegendary, MinDepth: 11,
			Options: []NarrativeOption{
				o("mining.ev_chasm_leg_o1", "mining.ev_chasm_leg_o1d",
					&EventEffect{Items: []BagEntry{{Name: "rough_diamond", Count: 3}, {Name: "kethari_crystal", Count: 1}}, Message: "mining.ev_chasm_leg_r1"}),
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
		{ID: "crystal_rare", Stage: StageDeep, Rarity: EventRare, MinDepth: 15,
			Options: []NarrativeOption{
				o("mining.ev_crystal_rare_o1", "mining.ev_crystal_rare_o1d",
					efi("mining.ev_crystal_rare_r1", BagEntry{Name: "kethari_crystal", Count: 1})),
				o("mining.ev_crystal_rare_o2", "mining.ev_crystal_rare_o2d",
					&EventEffect{RiskMod: -20, RiskTurns: 5, Items: []BagEntry{{Name: "ancient_alloy", Count: 1}}, Message: "mining.ev_crystal_rare_r2"}),
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
					&EventEffect{Items: []BagEntry{{Name: "ancient_core_shard", Count: 1}}, Message: "mining.ev_guardian_rare_r1"}),
				o("mining.ev_guardian_rare_o2", "mining.ev_guardian_rare_o2d",
					&EventEffect{RiskMod: -30, RiskTurns: 99, Items: []BagEntry{{Name: "ancient_alloy", Count: 3}}, Message: "mining.ev_guardian_rare_r2"}),
			}},

		// ═══ DEEP — LEGENDARY ═══
		{ID: "shrine_legendary", Stage: StageDeep, Rarity: EventLegendary, MinDepth: 17,
			Options: []NarrativeOption{
				o("mining.ev_shrine_leg_o1", "mining.ev_shrine_leg_o1d",
					&EventEffect{Items: []BagEntry{{Name: "resonance_core", Count: 1}, {Name: "kethari_crystal", Count: 2}}, RiskMod: -15, RiskTurns: 10, Message: "mining.ev_shrine_leg_r1"}),
				o("mining.ev_shrine_leg_o2", "mining.ev_shrine_leg_o2d",
					&EventEffect{LoreID: "mine_lore_engine", Items: []BagEntry{{Name: "ancient_core_shard", Count: 1}}, Message: "mining.ev_shrine_leg_r2"}),
				o("mining.ev_shrine_leg_o3", "mining.ev_shrine_leg_o3d",
					&EventEffect{RiskMod: -40, RiskTurns: 5, Items: []BagEntry{{Name: "primordial_geode", Count: 1}}, Message: "mining.ev_shrine_leg_r3"}),
			}},
		{ID: "crystal_legendary", Stage: StageDeep, Rarity: EventLegendary, MinDepth: 15,
			Options: []NarrativeOption{
				o("mining.ev_crystal_leg_o1", "mining.ev_crystal_leg_o1d",
					&EventEffect{RiskMod: -15, RiskTurns: 99, Items: []BagEntry{{Name: "resonance_core", Count: 1}}, Message: "mining.ev_crystal_leg_r1"}),
				o("mining.ev_crystal_leg_o2", "mining.ev_crystal_leg_o2d",
					&EventEffect{Items: []BagEntry{{Name: "kethari_crystal", Count: 3}, {Name: "ancient_alloy", Count: 2}}, Message: "mining.ev_crystal_leg_r2"}),
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

	// Glow common "investigate" 50/50 reward
	if eventID == "glow" && optionIdx == 0 && rand.Intn(100) < 50 {
		pool := lootAtDepth(depth)
		if len(pool) > 0 {
			it := pool[rand.Intn(len(pool))]
			eff.Items = append(eff.Items, BagEntry{Name: it.Name, Count: 1})
			eff.Message = "mining.ev_glow_r1g"
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
	return s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
		DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + ?", quantity)}),
	}).Create(&model.Inventory{UserID: userID, ItemID: itemID, Quantity: quantity}).Error
}

func (s *Service) HasItem(userID int64, itemID string) (bool, error) {
	var inv model.Inventory
	err := s.store.DB.Where("user_id = ? AND item_id = ? AND quantity > 0", userID, itemID).First(&inv).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) ConsumeItem(userID int64, itemID string) error {
	res := s.store.DB.Model(&model.Inventory{}).
		Where("user_id = ? AND item_id = ? AND quantity > 0", userID, itemID).
		UpdateColumn("quantity", gorm.Expr("quantity - 1"))
	if res.Error != nil {
		return res.Error
	}
	return s.store.DB.Where("user_id = ? AND item_id = ? AND quantity <= 0", userID, itemID).
		Delete(&model.Inventory{}).Error
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

func (s *Service) Descend(userID int64, depth int, bag []BagEntry, toolID string, ghostVeilTurns int) (*DescendResult, error) {
	ml, err := s.GetMinerLevel(userID)
	if err != nil {
		return nil, err
	}

	ti := GetToolInfo(toolID)
	if ti.MinLevel > ml {
		toolID = ""
		ti = GetToolInfo("")
	}

	risk := (depth-1)*5 - ti.RiskReduction
	levelReduc := int(float64(ml) * 1.5)
	risk -= levelReduc
	vitReduc := int(charsvc.GetVITReduction(s.store, userID) * 100)
	risk -= vitReduc
	if charsvc.HasPassive(s.store, userID, "perk_collapse_resist") {
		risk -= 5
	}

	if ghostVeilTurns > 0 {
		risk -= 10
	}
	if charsvc.HasBuff(s.store, userID, "reinforce") {
		risk = 0
		charsvc.ConsumeBuff(s.store, userID, "reinforce")
	}
	if risk < 0 {
		risk = 0
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

	var easterEgg *MiningEvent
	eRoll := rand.Intn(100) + 1
	if depth >= 10 && eRoll <= 1 {
		_ = s.GrantItem(userID, ancientAlloyItem, 1)
		easterEgg = &MiningEvent{Type: "ancient_forge", Items: []BagEntry{{Name: ancientAlloyItem, Count: 1}}}
		s.npcSvc.AddActivityReputation(userID, "mining", 5)
	} else if depth >= 6 && eRoll <= 3 {
		easterEgg = &MiningEvent{Type: "ghost_miner", Buff: ghostVeilBuff}
	} else if depth >= 7 && eRoll <= 6 && !hiddenChamber {
		extra := lootAtDepth(depth)
		if len(extra) > 0 {
			it := extra[rand.Intn(len(extra))]
			easterEgg = &MiningEvent{Type: "whispering_runes", Items: []BagEntry{{Name: it.Name, Count: 1}}}
			found := false
			for i, e := range bag {
				if e.Name == it.Name {
					bag[i].Count++
					found = true
					break
				}
			}
			if !found {
				bag = append(bag, BagEntry{Name: it.Name, Count: 1})
			}
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
		pool := lootAtDepth(depth)
		for i := 0; i < bonusCount && len(pool) > 0; i++ {
			it := pool[rand.Intn(len(pool))]
			found := false
			for j, e := range chamberItems {
				if e.Name == it.Name {
					chamberItems[j].Count++
					found = true
					break
				}
			}
			if !found {
				chamberItems = append(chamberItems, BagEntry{Name: it.Name, Count: 1})
			}
			found2 := false
			for j, e := range bag {
				if e.Name == it.Name {
					bag[j].Count++
					found2 = true
					break
				}
			}
			if !found2 {
				bag = append(bag, BagEntry{Name: it.Name, Count: 1})
			}
		}
		easterEgg = &MiningEvent{Type: "hidden_chamber", Items: chamberItems}
		s.npcSvc.AddActivityReputation(userID, "mining", 3)
		return &DescendResult{Item: nil, Bag: bag, Event: easterEgg, LoreID: loreID}, nil
	}

	lvl := depth + ti.LootTierBonus
	if lvl < 1 {
		lvl = 1
	}
	pool := lootAtDepth(lvl)

	if charsvc.HasBuff(s.store, userID, "midas_touch") {
		var filtered []MineItem
		for _, it := range pool {
			if it.Value >= 50 {
				filtered = append(filtered, it)
			}
		}
		if len(filtered) > 0 {
			pool = filtered
			charsvc.ConsumeBuff(s.store, userID, "midas_touch")
		}
	} else if charsvc.HasBuff(s.store, userID, "nose_for_treasure") {
		var filtered []MineItem
		for _, it := range pool {
			if it.Value >= 100 {
				filtered = append(filtered, it)
			}
		}
		if len(filtered) > 0 {
			pool = filtered
		}
		charsvc.ConsumeBuff(s.store, userID, "nose_for_treasure")
	}

	item := pool[rand.Intn(len(pool))]
	found := false
	for i, e := range bag {
		if e.Name == item.Name {
			bag[i].Count++
			found = true
			break
		}
	}
	if !found {
		bag = append(bag, BagEntry{Name: item.Name, Count: 1})
	}

	var nEvent *NarrativeEvent
	if rand.Intn(100) < rollEventChance(depth) {
		nEvent = pickNarrativeEvent(depth)
	}

	if item.Value >= 50 {
		s.npcSvc.AddActivityReputation(userID, "mining", 3)
	} else {
		s.npcSvc.AddActivityReputation(userID, "mining", 1)
	}

	return &DescendResult{Item: &item, Bag: bag, Event: easterEgg, NarrativeEvent: nEvent, LoreID: loreID}, nil
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
		if err := s.store.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
			DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + ?", e.Count)}),
		}).Create(&model.Inventory{UserID: userID, ItemID: e.Name, Quantity: e.Count}).Error; err != nil {
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
		next := jobXPForLevel(job.Level)
		for job.XP >= next {
			job.XP -= next
			job.Level++
			next = jobXPForLevel(job.Level)
		}
		if err := s.store.DB.Model(&model.Job{}).Where("user_id = ? AND job_name = ?", userID, "miner").
			Updates(map[string]any{"xp": job.XP, "level": job.Level}).Error; err != nil {
			return nil, err
		}
	}

	if toolID != "" {
		_ = s.ConsumeItem(userID, toolID)
	}

	leveled, lvl := charsvc.AddXP(s.store, userID, totalXP)

	unlocks, err := achievement.CheckAndUnlock(s.store.DB, userID)
	if err != nil {
		return nil, err
	}

	return &LeaveResult{XP: totalXP, Bag: bag, Unlocks: unlocks, ToolID: toolID, LeveledUp: leveled, NewLevel: lvl}, nil
}

func jobXPForLevel(level int) int {
	return 50 + level*25
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
	Depth          int
	ToolID         string
	GhostVeilTurns int
	RiskMod        int
	RiskTurns      int
	Bag            []BagEntry
}

// SaveSession persists an in-progress expedition for the user.
func (s *Service) SaveSession(userID int64, ps *PersistedSession) error {
	bagJSON, err := json.Marshal(ps.Bag)
	if err != nil {
		return err
	}
	return s.store.SaveMiningSession(&model.MiningSession{
		UserID:         userID,
		Depth:          ps.Depth,
		ToolID:         ps.ToolID,
		GhostVeilTurns: ps.GhostVeilTurns,
		RiskMod:        ps.RiskMod,
		RiskTurns:      ps.RiskTurns,
		Bag:            string(bagJSON),
		UpdatedAt:      time.Now(),
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
	return &PersistedSession{
		Depth:          m.Depth,
		ToolID:         m.ToolID,
		GhostVeilTurns: m.GhostVeilTurns,
		RiskMod:        m.RiskMod,
		RiskTurns:      m.RiskTurns,
		Bag:            bag,
	}, nil
}

// DeleteSession removes the persisted expedition for the user.
func (s *Service) DeleteSession(userID int64) error {
	return s.store.DeleteMiningSession(userID)
}
