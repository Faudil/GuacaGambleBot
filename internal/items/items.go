package items

import "sync"

type Rarity string

const (
	RarityCommon    Rarity = "common"
	RarityUncommon  Rarity = "uncommon"
	RarityRare      Rarity = "rare"
	RarityEpic      Rarity = "epic"
	RarityLegendary Rarity = "legendary"
)

// Rank returns the tier order of the rarity (Common=0 .. Legendary=4). Rarity
// values are stored as strings, whose lexical order differs from the tier
// order, so comparisons must go through Rank.
func (r Rarity) Rank() int {
	switch r {
	case RarityCommon:
		return 0
	case RarityUncommon:
		return 1
	case RarityRare:
		return 2
	case RarityEpic:
		return 3
	case RarityLegendary:
		return 4
	}
	return -1
}

func (r Rarity) String() string {
	return string(r)
}

func rarityForPrice(price int) Rarity {
	return RarityForPrice(price)
}

// RarityForPrice maps a price to the rarity tier the catalog assigns to items.
func RarityForPrice(price int) Rarity {
	switch {
	case price >= 5000:
		return RarityLegendary
	case price >= 1000:
		return RarityEpic
	case price >= 200:
		return RarityRare
	case price >= 50:
		return RarityUncommon
	default:
		return RarityCommon
	}
}

type Category string

const (
	Mining     Category = "mining"
	Fishing    Category = "fishing"
	Farming    Category = "farming"
	Archeology Category = "archeology"
	Food       Category = "food"
	Tools      Category = "tools"
	Materials  Category = "materials"
	Special    Category = "special"
	Equipment  Category = "equipment"
	Delve      Category = "delve"
)

type Item struct {
	ID          string
	Name        string
	Emoji       string
	Price       int
	Description string
	EffectType  string
	Droppable   bool
	Category    Category
	Rarity      Rarity
	EquipSlot   string // "weapon", "armor", "jewelry", "trinket", or "" (see slots.go)
	MinLevel    int    // minimum character level to equip; 0 = derived from rarity
	StatSTR     int
	StatDEX     int
	StatINT     int
	StatVIT     int
	StatLUK     int
	SetID       string // set identifier for set items, empty if none
	SetName     string // human-readable set name
	Source      string // primary acquisition source; see the constants in catalog.go

	// ShopExcluded marks items that are obtained only through their dedicated
	// activity (criminality, boss leagues, etc.) and must never be offered by
	// the daily shop nor sold on the market or to the vendor.
	ShopExcluded bool

	// Durability is the number of uses a tool lasts before breaking. Zero means
	// the item does not wear down (plain resources, consumables, ...).
	Durability int
}

// MinLevelForRarity maps a rarity tier to the minimum character level required
// to equip gear of that tier.
func MinLevelForRarity(r Rarity) int {
	switch r {
	case RarityLegendary:
		return 20
	case RarityEpic:
		return 15
	case RarityRare:
		return 10
	case RarityUncommon:
		return 5
	default:
		return 1
	}
}

var byID = func() map[string]*Item {
	m := make(map[string]*Item, len(all))
	for i := range all {
		m[all[i].ID] = &all[i]
	}
	// Legacy alias for the pre-ASCII item ID.
	if it, ok := m["dechirure_scythe"]; ok {
		m["déchirure_scythe"] = it
	}
	return m
}()

var byName = func() map[string]*Item {
	m := make(map[string]*Item, len(all))
	for i := range all {
		m[all[i].Name] = &all[i]
	}
	return m
}()

var byCategory = func() map[Category][]Item {
	m := make(map[Category][]Item)
	for _, it := range all {
		m[it.Category] = append(m[it.Category], it)
	}
	return m
}()

func init() {
	for i := range all {
		if all[i].Rarity == "" {
			all[i].Rarity = rarityForPrice(all[i].Price)
		}
		if all[i].EquipSlot != "" && all[i].MinLevel == 0 {
			all[i].MinLevel = MinLevelForRarity(all[i].Rarity)
		}
	}
}

var (
	mu           sync.RWMutex
	dynamicItems []Item
)

// RegisterDynamic adds a procedurally generated item to the runtime catalog.
// It is safe for concurrent use.
func RegisterDynamic(item Item) {
	mu.Lock()
	defer mu.Unlock()
	item.Droppable = false
	item.Category = Delve
	if item.Rarity == "" {
		item.Rarity = RarityForPrice(item.Price)
	}
	dynamicItems = append(dynamicItems, item)
}

// GetDynamicByID returns a dynamic item by its ID, or nil.
func GetDynamicByID(id string) *Item {
	mu.RLock()
	defer mu.RUnlock()
	for i := range dynamicItems {
		if dynamicItems[i].ID == id {
			return &dynamicItems[i]
		}
	}
	return nil
}

func Get(nameOrID string) *Item {
	if it, ok := byID[nameOrID]; ok {
		return it
	}
	if it := GetDynamicByID(nameOrID); it != nil {
		return it
	}
	return byName[nameOrID]
}

// Canonical resolves a display name or an id to the canonical item ID. It
// returns "" when the key does not match any known item. Every inventory
// read/write helper normalizes keys through this function so a display name
// (e.g. "Fertilizer") can never be stored or looked up under a different key
// than the canonical id (e.g. "fertilizer").
func Canonical(nameOrID string) string {
	if it := Get(nameOrID); it != nil {
		return it.ID
	}
	return ""
}

func AllItems() []Item {
	out := make([]Item, len(all))
	copy(out, all)
	return out
}

// DisplayName resolves a name or ID to the canonical English display name.
// If the item is not found, the input is returned unchanged.
func DisplayName(nameOrID string) string {
	it := Get(nameOrID)
	if it == nil {
		return nameOrID
	}
	return it.Name
}

func ItemsByCategory(cat Category) []Item {
	return byCategory[cat]
}

// IsSellable reports whether the item can be sold at all. Any item with a
// positive base price can be sold to the vendor at any time, unless it is
// shop-excluded or a set piece (award-only items are never merchantable);
// only the weekly rotation decides whether it sells at the dynamic market
// price instead.
func (it *Item) IsSellable() bool {
	return it.Price > 0 && !it.ShopExcluded && it.SetID == ""
}

// IsLegendaryDrop reports whether the item is a legendary-tier reward that
// must be earned by playing its activity. Legendary drops never appear in the
// market rotation nor the daily shop: buying them would override the need to
// play the activity that awards them.
func (it *Item) IsLegendaryDrop() bool {
	return it.Droppable && it.Rarity == RarityLegendary
}

func (it *Item) IsMarketable() bool {
	if it.Price <= 0 {
		return false
	}
	if it.IsLegendaryDrop() {
		return false
	}
	switch it.Category {
	case Mining, Fishing, Farming, Archeology, Tools, Food:
		return true
	}
	return false
}

func MarketableItems() []Item {
	result := make([]Item, 0, len(all))
	for _, it := range all {
		if it.IsMarketable() {
			result = append(result, it)
		}
	}
	return result
}
