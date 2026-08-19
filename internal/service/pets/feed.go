package pets

// FeedItemDef describes what feeding an item does to a pet.
type FeedItemDef struct {
	ItemID string
	Stat   string
	Amount float64
	Bond   int
}

// RawFeedItems maps raw gathered ingredients that pets can eat directly.
// Raw food grants no bond — the price of skipping the crafting step.
var RawFeedItems = map[string]string{
	"sardine":        "speed",
	"trout":          "speed",
	"carp":           "speed",
	"salmon":         "max_hp",
	"wheat":          "acc",
	"oat":            "acc",
	"corn":           "acc",
	"carrot":         "defense",
	"potato":         "defense",
	"tomato":         "atk",
	"pumpkin":        "atk",
	"coffee_bean":    "atk",
	"cocoa_bean":     "atk",
	"coffee":         "speed",
	"strawberry":     "speed",
	"star_fruit":     "speed",
	"nova_fruit":     "max_hp",
	"golden_apple":   "max_hp",
	"ghost_wheat":    "acc",
	"prismatic_corn": "acc",
	"golden_potato":  "defense",
	"golden_carrot":  "defense",
	"blood_tomato":   "atk",
	"cursed_pumpkin": "atk",
}

// CraftedFeedItems maps crafted foods (tier 1) and potions (tier 2)
// to their effects. Foods grant +1 stat, potions +2.
var CraftedFeedItems = map[string]FeedItemDef{
	// --- Tier 1: Food ---
	"warrior_stew":   {ItemID: "warrior_stew", Stat: "atk", Amount: 1, Bond: 2},
	"stonebread":     {ItemID: "stonebread", Stat: "defense", Amount: 1, Bond: 2},
	"zephyr_berries": {ItemID: "zephyr_berries", Stat: "speed", Amount: 1, Bond: 2},
	"hunters_soup":   {ItemID: "hunters_soup", Stat: "acc", Amount: 1, Bond: 2},
	"lucky_roast":    {ItemID: "lucky_roast", Stat: "crit_c", Amount: 1, Bond: 2},
	"thunder_steak":  {ItemID: "thunder_steak", Stat: "crit_d", Amount: 0.1, Bond: 2},
	"heart_stew":     {ItemID: "heart_stew", Stat: "max_hp", Amount: 2, Bond: 2},
	"dragon_chili":   {ItemID: "dragon_chili", Stat: "atk", Amount: 1, Bond: 2},
	"iron_loaf":      {ItemID: "iron_loaf", Stat: "defense", Amount: 1, Bond: 2},
	"storm_porridge": {ItemID: "storm_porridge", Stat: "speed", Amount: 1, Bond: 2},
	"falcon_pie":     {ItemID: "falcon_pie", Stat: "acc", Amount: 1, Bond: 2},
	"clover_salad":   {ItemID: "clover_salad", Stat: "crit_c", Amount: 1, Bond: 2},
	"volcano_ribs":   {ItemID: "volcano_ribs", Stat: "crit_d", Amount: 0.1, Bond: 2},
	"giant_noodles":  {ItemID: "giant_noodles", Stat: "max_hp", Amount: 2, Bond: 2},
	// --- Tier 2: Potions ---
	"berserker_elixir":   {ItemID: "berserker_elixir", Stat: "atk", Amount: 2, Bond: 3},
	"adamant_tonic":      {ItemID: "adamant_tonic", Stat: "defense", Amount: 2, Bond: 3},
	"gale_draught":       {ItemID: "gale_draught", Stat: "speed", Amount: 2, Bond: 3},
	"oracles_insight":    {ItemID: "oracles_insight", Stat: "acc", Amount: 2, Bond: 3},
	"fatalist_elixir":    {ItemID: "fatalist_elixir", Stat: "crit_c", Amount: 2, Bond: 3},
	"ruin_tonic":         {ItemID: "ruin_tonic", Stat: "crit_d", Amount: 0.2, Bond: 3},
	"vitality_elixir":    {ItemID: "vitality_elixir", Stat: "max_hp", Amount: 4, Bond: 3},
	"skull_elixir":       {ItemID: "skull_elixir", Stat: "atk", Amount: 2, Bond: 3},
	"bastion_tonic":      {ItemID: "bastion_tonic", Stat: "defense", Amount: 2, Bond: 3},
	"tempest_draught":    {ItemID: "tempest_draught", Stat: "speed", Amount: 2, Bond: 3},
	"seer_elixir":        {ItemID: "seer_elixir", Stat: "acc", Amount: 2, Bond: 3},
	"gamblers_tonic":     {ItemID: "gamblers_tonic", Stat: "crit_c", Amount: 2, Bond: 3},
	"annihilator_elixir": {ItemID: "annihilator_elixir", Stat: "crit_d", Amount: 0.2, Bond: 3},
	"colossus_draught":   {ItemID: "colossus_draught", Stat: "max_hp", Amount: 4, Bond: 3},
}

// BondTreatItem is a special feedable that grants bond without consuming food capacity.
const BondTreatItem = "bond_treat"

// GetFeedItemDef returns the feeding definition for an item, or nil if it cannot be fed.
func GetFeedItemDef(itemID string) *FeedItemDef {
	if def, ok := CraftedFeedItems[itemID]; ok {
		return &def
	}
	if stat, ok := RawFeedItems[itemID]; ok {
		return &FeedItemDef{ItemID: itemID, Stat: stat, Amount: 1, Bond: 0}
	}
	if itemID == BondTreatItem {
		return &FeedItemDef{ItemID: BondTreatItem, Bond: 5}
	}
	return nil
}
