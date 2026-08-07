package pets

import "guacagamblebot/internal/model"

// FeedItemDef describes what feeding an item does to a pet.
type FeedItemDef struct {
	ItemID      string
	Stat        string
	Amount      int
	Bond        int
	CountsToCap bool // if false, the item bypasses the food capacity (e.g. bond treats)
}

// RawFeedItems maps raw gathered ingredients that pets can eat directly.
// Raw food grants no bond — the price of skipping the crafting step.
var RawFeedItems = map[string]string{
	"sardine": "speed",
	"trout":   "speed",
	"salmon":  "speed",
	"carp":    "speed",
	"wheat":   "acc",
	"carrot":  "acc",
	"potato":  "acc",
	"corn":    "acc",
	"pebble":  "defense",
	"coal":    "defense",
}

// CraftedFeedItems maps crafted foods (tier 1) and potions (tier 2)
// to their effects. Foods grant +1 stat, potions +2.
var CraftedFeedItems = map[string]FeedItemDef{
	// --- Tier 1: Food ---
	"warrior_stew":   {ItemID: "warrior_stew", Stat: "atk", Amount: 1, Bond: 1, CountsToCap: true},
	"stonebread":     {ItemID: "stonebread", Stat: "defense", Amount: 1, Bond: 1, CountsToCap: true},
	"zephyr_berries": {ItemID: "zephyr_berries", Stat: "speed", Amount: 1, Bond: 1, CountsToCap: true},
	"hunters_soup":   {ItemID: "hunters_soup", Stat: "acc", Amount: 1, Bond: 1, CountsToCap: true},
	// --- Tier 2: Potions ---
	"berserker_elixir": {ItemID: "berserker_elixir", Stat: "atk", Amount: 2, Bond: 1, CountsToCap: true},
	"adamant_tonic":    {ItemID: "adamant_tonic", Stat: "defense", Amount: 2, Bond: 1, CountsToCap: true},
	"gale_draught":     {ItemID: "gale_draught", Stat: "speed", Amount: 2, Bond: 1, CountsToCap: true},
	"oracles_insight":  {ItemID: "oracles_insight", Stat: "acc", Amount: 2, Bond: 1, CountsToCap: true},
}

// BondTreatItem is a special feedable that grants bond without consuming food capacity.
const BondTreatItem = "bond_treat"

// GetFeedItemDef returns the feeding definition for an item, or nil if it cannot be fed.
func GetFeedItemDef(itemID string) *FeedItemDef {
	if def, ok := CraftedFeedItems[itemID]; ok {
		return &def
	}
	if stat, ok := RawFeedItems[itemID]; ok {
		return &FeedItemDef{ItemID: itemID, Stat: stat, Amount: 1, Bond: 0, CountsToCap: true}
	}
	if itemID == BondTreatItem {
		return &FeedItemDef{ItemID: BondTreatItem, Bond: 5, CountsToCap: false}
	}
	return nil
}

// MaxFoodCapacity returns how many capacity-counting meals a pet can eat in total.
func MaxFoodCapacity(pet *model.UserPet) int {
	return RarityFoodCapacity[petTypeRarity(pet.PetType)] * pet.Level
}

// IsFull reports whether the pet has reached its lifetime food capacity.
func IsFull(pet *model.UserPet) bool {
	return pet.FoodEaten >= MaxFoodCapacity(pet)
}
