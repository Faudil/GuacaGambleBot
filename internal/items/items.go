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
	EquipSlot   string // "weapon", "armor", "accessory", "trinket", or ""
	MinLevel    int    // minimum character level to equip; 0 = derived from rarity
	StatSTR     int
	StatDEX     int
	StatINT     int
	StatVIT     int
	StatLUK     int
	SetID       string // set identifier for set items, empty if none
	SetName     string // human-readable set name

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

var all = []Item{
	// --- Mining ---
	{ID: "pebble", Name: "Pebble", Emoji: "🪨", Price: 1, Description: "A useless little rock. Totally worthless.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "coal", Name: "Coal", Emoji: "🪨", Price: 5, Description: "Great for keeping warm.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "iron_ore", Name: "Iron Ore", Emoji: "⛏️", Price: 10, Description: "Useful for forging sturdy tools.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "copper_ore", Name: "Copper Ore", Emoji: "⛏️", Price: 15, Description: "Useful for forging sturdy tools.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "silver_ore", Name: "Silver Ore", Emoji: "⛏️", Price: 25, Description: "Useful for forging sturdy tools.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "gold_nugget", Name: "Gold Nugget", Emoji: "✨", Price: 50, Description: "Shiny! Merchants love this.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "platinum", Name: "Platinum", Emoji: "✨", Price: 75, Description: "Shiny! Merchants love this.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "emerald", Name: "Emerald", Emoji: "💚", Price: 100, Description: "AMAZING! Worth a fortune!", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "rough_diamond", Name: "Rough Diamond", Emoji: "💎", Price: 300, Description: "AMAZING! Worth a fortune!", EffectType: "resource", Droppable: true, Category: Mining},

	// --- Fishing ---
	{ID: "old_boot", Name: "Old Boot", Emoji: "🥾", Price: 1, Description: "An old boot. Worthless.", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "trout", Name: "Trout", Emoji: "🐟", Price: 10, Description: "Freshwater fish.", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "salmon", Name: "Salmon", Emoji: "🐟", Price: 10, Description: "Perfect for sushi.", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "sardine", Name: "Sardine", Emoji: "🐟", Price: 15, Description: "A small saltwater fish.", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "carp", Name: "Carp", Emoji: "🐟", Price: 25, Description: "The best freshwater fish.", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "pufferfish", Name: "Pufferfish", Emoji: "🐡", Price: 50, Description: "Watch out, it stings!", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "swordfish", Name: "Swordfish", Emoji: "🐟", Price: 150, Description: "A majestic fighting fish.", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "shark", Name: "Shark", Emoji: "🦈", Price: 100, Description: "AMAZING! Worth a fortune!", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "whale", Name: "Whale", Emoji: "🐋", Price: 300, Description: "AMAZING! Worth a fortune!", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "kraken_tentacle", Name: "Kraken Tentacle", Emoji: "🦑", Price: 500, Description: "YOU CAUGHT A MONSTER?!", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "magma_carp", Name: "Magma Carp", Emoji: "🐟", Price: 200, Description: "A fiery fish from the depths.", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "lava_serpent", Name: "Lava Serpent", Emoji: "🐍", Price: 500, Description: "A legendary beast of molten rock.", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "worm", Name: "Worm", Emoji: "🪱", Price: 5, Description: "Common bait. Good for small catches.", EffectType: "bait", Droppable: true, Category: Fishing},
	{ID: "crayfish", Name: "Crayfish", Emoji: "🦞", Price: 25, Description: "Rare bait. Attracts stronger fish.", EffectType: "bait", Droppable: true, Category: Fishing},
	{ID: "golden_lure", Name: "Golden Lure", Emoji: "👑", Price: 100, Description: "Legendary bait. Draws the deadliest creatures.", EffectType: "bait", Droppable: true, Category: Fishing},
	{ID: "mutagen", Name: "Mutagen", Emoji: "🧪", Price: 100, Description: "A glowing mutagenic substance from a mutated fish.", EffectType: "resource", Droppable: true, Category: Fishing},

	// --- Farming ---
	{ID: "rotten_plant", Name: "Rotten Plant", Emoji: "🌿", Price: 0, Description: "You mismanaged your farm...", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "wheat", Name: "Wheat", Emoji: "🌾", Price: 5, Description: "Essential for making bread.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "oat", Name: "Oat", Emoji: "🌾", Price: 8, Description: "Perfect for breakfast.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "corn", Name: "Corn", Emoji: "🌽", Price: 12, Description: "Also makes popcorn!", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "potato", Name: "Potato", Emoji: "🥔", Price: 20, Description: "You can make vodka from it…", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "tomato", Name: "Tomato", Emoji: "🍅", Price: 25, Description: "Fruit or vegetable? The debate continues.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "pumpkin", Name: "Pumpkin", Emoji: "🎃", Price: 40, Description: "Perfect for Halloween.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "coffee_bean", Name: "Coffee Bean", Emoji: "🫘", Price: 60, Description: "The black gold of the morning.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "cocoa_bean", Name: "Cocoa Bean", Emoji: "🫘", Price: 75, Description: "The main ingredient for happiness (chocolate).", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "strawberry", Name: "Strawberry", Emoji: "🍓", Price: 90, Description: "Red, sweet and juicy.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "golden_apple", Name: "Golden Apple", Emoji: "🍎", Price: 150, Description: "It shines with a magical glow.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "star_fruit", Name: "Star Fruit", Emoji: "⭐", Price: 250, Description: "A cosmic fruit from another dimension.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "carrot", Name: "Carrot", Emoji: "🥕", Price: 15, Description: "A crunchy orange root.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "ghost_wheat", Name: "Ghost Wheat", Emoji: "🌾", Price: 50, Description: "Translucent wheat that shimmers in moonlight.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "prismatic_corn", Name: "Prismatic Corn", Emoji: "🌽", Price: 48, Description: "Kernels shift through impossible colors.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "golden_potato", Name: "Golden Potato", Emoji: "🥔", Price: 160, Description: "A potato encased in solid gold!", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "blood_tomato", Name: "Blood Tomato", Emoji: "🍅", Price: 150, Description: "Deep crimson, pulsing with warmth...", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "cursed_pumpkin", Name: "Cursed Pumpkin", Emoji: "🎃", Price: 400, Description: "It whispers to you at night.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "golden_carrot", Name: "Golden Carrot", Emoji: "🥕", Price: 1000, Description: "A carrot of pure gold. Legendary.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "nova_fruit", Name: "Nova Fruit", Emoji: "⭐", Price: 5000, Description: "A miniature sun cradled in your hands.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "mysterious_seed", Name: "Mysterious Seed", Emoji: "🔮", Price: 0, Description: "An unknown seed pulsing with energy.", EffectType: "resource", Droppable: false, Category: Farming},

	// --- Archeology ---
	{ID: "bone_dust", Name: "Bone Dust", Emoji: "🦴", Price: 1, Description: "Bone dust from a completely destroyed fossil.", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "damaged_fossil", Name: "Damaged Fossil", Emoji: "🦴", Price: 50, Description: "A poorly extracted fossil that lost its value.", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "common_fossil", Name: "Common Fossil", Emoji: "🦴", Price: 150, Description: "An intact fossil of a common animal.", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "rare_fossil", Name: "Rare Fossil", Emoji: "🦴", Price: 300, Description: "An intact fossil of a rare animal.", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "epic_fossil", Name: "Epic Fossil", Emoji: "🦴", Price: 500, Description: "An intact fossil of an epic animal.", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "legendary_fragment", Name: "Legendary Fragment", Emoji: "🦖", Price: 1000, Description: "A legendary T-Rex fragment!", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "pure_dna", Name: "Pure DNA", Emoji: "🧬", Price: 3000, Description: "Perfectly preserved dinosaur DNA. Amazing!", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "shadow_fossil", Name: "Shadow Fossil", Emoji: "🖤", Price: 5000, Description: "A fossil of pure darkness. It seems to absorb light.", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "cursed_artifact", Name: "Cursed Artifact", Emoji: "🔮", Price: 800, Description: "An ancient relic pulsing with dark energy.", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "purified_relic", Name: "Purified Relic", Emoji: "✨", Price: 1500, Description: "A relic cleansed of its dark curse.", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "fossilized_egg", Name: "Fossilized Egg", Emoji: "🥚", Price: 500, Description: "A stone egg with ancient patterns. Who knows what's inside?", EffectType: "consumable", Droppable: false, Category: Special},
	{ID: "coelacanth_egg", Name: "Coelacanth Egg", Emoji: "🥚", Price: 2500, Description: "An egg that seems prehistoric. It pulses faintly.", EffectType: "consumable", Droppable: false, Category: Special},
	{ID: "excavator_hat", Name: "Excavator's Hat", Emoji: "🎩", Price: 0, Description: "A worn expedition hat. It carries centuries of stories.", EffectType: "collectible", Droppable: false, Category: Special},
	{ID: "journal_page_1", Name: "Journal Page #1", Emoji: "📄", Price: 1, Description: "A torn page from an ancient excavator's journal.", EffectType: "resource", Droppable: false, Category: Archeology},
	{ID: "journal_page_2", Name: "Journal Page #2", Emoji: "📄", Price: 1, Description: "A torn page from an ancient excavator's journal.", EffectType: "resource", Droppable: false, Category: Archeology},
	{ID: "journal_page_3", Name: "Journal Page #3", Emoji: "📄", Price: 1, Description: "A torn page from an ancient excavator's journal.", EffectType: "resource", Droppable: false, Category: Archeology},
	{ID: "journal_page_4", Name: "Journal Page #4", Emoji: "📄", Price: 1, Description: "A torn page from an ancient excavator's journal.", EffectType: "resource", Droppable: false, Category: Archeology},
	{ID: "journal_page_5", Name: "Journal Page #5", Emoji: "📄", Price: 1, Description: "A torn page from an ancient excavator's journal.", EffectType: "resource", Droppable: false, Category: Archeology},
	{ID: "journal_page_6", Name: "Journal Page #6", Emoji: "📄", Price: 1, Description: "A torn page from an ancient excavator's journal.", EffectType: "resource", Droppable: false, Category: Archeology},
	{ID: "journal_page_7", Name: "Journal Page #7", Emoji: "📄", Price: 1, Description: "A torn page from an ancient excavator's journal.", EffectType: "resource", Droppable: false, Category: Archeology},
	{ID: "journal_page_8", Name: "Journal Page #8", Emoji: "📄", Price: 1, Description: "A torn page from an ancient excavator's journal.", EffectType: "resource", Droppable: false, Category: Archeology},

	// --- Seeds (Materials) ---
	{ID: "wheat_seed", Name: "Wheat Seed", Emoji: "🌱", Price: 2, Description: "Plant to get wheat (5 min).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "oat_seed", Name: "Oat Seed", Emoji: "🌱", Price: 3, Description: "Plant to get oat (10 min).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "corn_seed", Name: "Corn Seed", Emoji: "🌱", Price: 5, Description: "Plant to get corn (30 min).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "potato_seed", Name: "Potato Seed", Emoji: "🌱", Price: 8, Description: "Plant to get potato (1h).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "tomato_seed", Name: "Tomato Seed", Emoji: "🌱", Price: 10, Description: "Plant to get tomato (2h).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "pumpkin_seed", Name: "Pumpkin Seed", Emoji: "🌱", Price: 15, Description: "Plant to get pumpkin (4h).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "coffee_seed", Name: "Coffee Seed", Emoji: "🌱", Price: 25, Description: "Plant to get coffee (8h).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "cocoa_seed", Name: "Cocoa Seed", Emoji: "🌱", Price: 30, Description: "Plant to get cocoa (12h).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "strawberry_seed", Name: "Strawberry Seed", Emoji: "🌱", Price: 40, Description: "Plant to get strawberry (18h).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "golden_apple_seed", Name: "Golden Apple Seed", Emoji: "🌱", Price: 75, Description: "Plant to get golden apple (24h).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "star_fruit_seed", Name: "Star Fruit Seed", Emoji: "🌱", Price: 125, Description: "Plant to get star fruit (48h).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "carrot_seed", Name: "Carrot Seed", Emoji: "🌱", Price: 7, Description: "Plant to get carrot (15 min).", EffectType: "resource", Droppable: true, Category: Materials},

	// --- Tools ---
	{ID: "beer", Name: "Beer", Emoji: "🍺", Price: 50, Description: "The miner's drink! Resets !mine cooldown.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "coffee", Name: "Coffee", Emoji: "☕", Price: 50, Description: "Wakes you up. Resets !daily cooldown.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "bow", Name: "Bow", Emoji: "🏹", Price: 300, Description: "Helps with hunting! Resets !hunt cooldown.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "fertilizer", Name: "Fertilizer", Emoji: "🧪", Price: 200, Description: "Accelerates crop growth! Reduces remaining grow time by 30%.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "growth_elixir", Name: "Growth Elixir", Emoji: "⚡", Price: 250, Description: "Instantly matures a growing crop.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "hook", Name: "Hook", Emoji: "🪝", Price: 200, Description: "Attracts fish! Resets !fish cooldown.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "forget_potion", Name: "Forget Potion", Emoji: "🧪", Price: 2500, Description: "Resets your pet to level 10.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "skill_scroll", Name: "Skill Scroll", Emoji: "📜", Price: 5000, Description: "Resets all your active pet's learned skills.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "bond_treat", Name: "Bond Treat", Emoji: "🍬", Price: 150, Description: "Increases your active pet's bond level by 5.", EffectType: "consumable", Droppable: false, Category: Food},

	// --- Pet Food (crafted, +1 stat, +1 bond) ---
	{ID: "warrior_stew", Name: "Warrior's Stew", Emoji: "🍲", Price: 100, Description: "A hearty meat stew that fortifies your pet. +1 ATK.", EffectType: "consumable", Droppable: false, Category: Food},
	{ID: "stonebread", Name: "Stonebread", Emoji: "🍞", Price: 100, Description: "A dense, mineral-rich loaf. +1 DEF.", EffectType: "consumable", Droppable: false, Category: Food},
	{ID: "zephyr_berries", Name: "Zephyr Berries", Emoji: "🫐", Price: 120, Description: "Wild berries crackling with wind energy. +1 SPD.", EffectType: "consumable", Droppable: false, Category: Food},
	{ID: "hunters_soup", Name: "Hunter's Eye Soup", Emoji: "🥘", Price: 120, Description: "A sharp-eyed hunter's secret recipe. +1 ACC.", EffectType: "consumable", Droppable: false, Category: Food},

	// --- Pet Potions (crafted, +2 stat, +1 bond) ---
	{ID: "berserker_elixir", Name: "Berserker Elixir", Emoji: "⚗️", Price: 400, Description: "A volatile brew that awakens raw fury. +2 ATK.", EffectType: "consumable", Droppable: false, Category: Food},
	{ID: "adamant_tonic", Name: "Adamant Tonic", Emoji: "🧪", Price: 400, Description: "Turns the drinker's hide to living stone. +2 DEF.", EffectType: "consumable", Droppable: false, Category: Food},
	{ID: "gale_draught", Name: "Gale Draught", Emoji: "💨", Price: 450, Description: "Tastes like a storm on the tongue. +2 SPD.", EffectType: "consumable", Droppable: false, Category: Food},
	{ID: "oracles_insight", Name: "Oracle's Insight", Emoji: "👁️", Price: 450, Description: "A single sip of foresight. +2 ACC.", EffectType: "consumable", Droppable: false, Category: Food},
	{ID: "personality_mirror", Name: "Personality Mirror", Emoji: "🪞", Price: 7500, Description: "Mysteriously changes your pet's personality.", EffectType: "consumable", Droppable: false, Category: Special},
	{ID: "artifact_shard", Name: "Artifact Shard", Emoji: "💠", Price: 7500, Description: "Resets your pet artifact, re-rolling all 3 stats.", EffectType: "consumable", Droppable: false, Category: Special},
	{ID: "fortune_cookie", Name: "Fortune Cookie", Emoji: "🥠", Price: 20, Description: "A delicious cookie with a premonitory message.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "rusty_magnet", Name: "Rusty Magnet", Emoji: "🧲", Price: 30, Description: "Use it to find some pocket change.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "magnet", Name: "Magnet", Emoji: "🧲", Price: 50, Description: "Use it to find money on the ground.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "electric_magnet", Name: "Electric Magnet", Emoji: "🧲", Price: 500, Description: "Use it to find lots of money!", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "rigged_coin", Name: "Rigged Coin", Emoji: "🪙", Price: 200, Description: "Boosts your luck. Raises coinflip odds to 75%.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "casino_token", Name: "Casino Token", Emoji: "🎰", Price: 50, Description: "Gives you another chance at !casino.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "vip_ticket", Name: "VIP Ticket", Emoji: "🎟️", Price: 100, Description: "Resets your casino and coinflip limits.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "identity_scroll", Name: "Identity Scroll", Emoji: "📜", Price: 500, Description: "Randomly changes your server nickname.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "thieves_glove", Name: "Thieve's Glove", Emoji: "🧤", Price: 20, Description: "Use !rob @target with these gloves.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "scratch_ticket", Name: "Scratch Ticket", Emoji: "🎰", Price: 100, Description: "Scratch to win up to $1000 instantly!", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "data_disk", Name: "Data Disk", Emoji: "💾", Price: 50, Description: "A corrupted memory disk from a lost era.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "old_journal", Name: "Old Journal", Emoji: "📖", Price: 30, Description: "A dusty notebook handwritten by an unknown author.", EffectType: "consumable", Droppable: false, Category: Tools},

	// --- Mining Tools & Artifacts ---
	{ID: "steel_pickaxe", Name: "Steel Pickaxe", Emoji: "⛏️", Price: 1500, Description: "A sturdy steel pickaxe. Grants +1 loot tier and -5% collapse risk per descend. Breaks after 25 digs.", EffectType: "mining_tool", Droppable: false, Category: Tools, Durability: 25},
	{ID: "diamond_drill", Name: "Diamond Drill", Emoji: "🔧", Price: 5000, Description: "A diamond-tipped drill. Grants +2 loot tiers and -10% collapse risk per descend. Breaks after 50 digs.", EffectType: "mining_tool", Droppable: false, Category: Tools, Durability: 50},
	{ID: "ancient_alloy", Name: "Ancient Alloy", Emoji: "🔩", Price: 500, Description: "A fragment of metal from a forgotten forge. It hums with residual heat.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "ancient_core_shard", Name: "Ancient Core Shard", Emoji: "💠", Price: 0, Description: "A crystalline shard from the Heart of the Mountain. It pulses with a faint light.", EffectType: "collectible", Droppable: false, Category: Special},
	{ID: "kethari_crystal", Name: "Kethari Crystal", Emoji: "🔮", Price: 1000, Description: "A prismatic crystal forged by the Kethari. It hums with stored resonance.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "primordial_geode", Name: "Primordial Geode", Emoji: "🪨", Price: 2000, Description: "A geode from the dawn of the mountain. Who knows what lies within?", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "resonance_core", Name: "Resonance Core", Emoji: "⚡", Price: 5000, Description: "A perfectly preserved Kethari power source. It radiates ancient energy.", EffectType: "resource", Droppable: true, Category: Mining},

	// --- Veil Raid ---
	{ID: "veil_key", Name: "Veil Key", Emoji: "🔮", Price: 5000, Description: "Opens a dimensional tear into the Veil Rift. Required to create or join a raid.", EffectType: "consumable", Droppable: false, Category: Special},
	{ID: "dimensional_shard", Name: "Dimensional Shard", Emoji: "💠", Price: 100, Description: "Fragments of broken reality. Used as raid currency for exclusive gear and cosmetics.", EffectType: "collectible", Droppable: false, Category: Special},

	// --- Eggs ---
	{ID: "forest_egg", Name: "Forest Egg", Emoji: "🥚", Price: 4000, Description: "A forest-embraced egg. Type !hatch to open it!", EffectType: "consumable", Droppable: false, Category: Special},
	{ID: "cave_egg", Name: "Cave Egg", Emoji: "🥚", Price: 6000, Description: "A cavern-dark egg. Type !hatch to open it!", EffectType: "consumable", Droppable: false, Category: Special},
	{ID: "desert_egg", Name: "Desert Egg", Emoji: "🥚", Price: 8000, Description: "A sun-scorched egg. Type !hatch to open it!", EffectType: "consumable", Droppable: false, Category: Special},
	{ID: "mountain_egg", Name: "Mountain Egg", Emoji: "🥚", Price: 10000, Description: "A peak-born egg. Type !hatch to open it!", EffectType: "consumable", Droppable: false, Category: Special},
	{ID: "ocean_egg", Name: "Ocean Egg", Emoji: "🥚", Price: 12000, Description: "A deep-sea egg. Type !hatch to open it!", EffectType: "consumable", Droppable: false, Category: Special},
	{ID: "tundra_egg", Name: "Tundra Egg", Emoji: "🥚", Price: 14000, Description: "An ice-wrapped egg. Type !hatch to open it!", EffectType: "consumable", Droppable: false, Category: Special},
	{ID: "volcano_egg", Name: "Volcano Egg", Emoji: "🥚", Price: 16000, Description: "A magma-infused egg. Type !hatch to open it!", EffectType: "consumable", Droppable: false, Category: Special},
	{ID: "boss_trophy", Name: "Boss Trophy", Emoji: "🏆", Price: 10000, Description: "A legendary trophy for defeating the boss.", EffectType: "collectible", Droppable: false, Category: Special},
	{ID: "mastery_medallion", Name: "Mastery Medallion", Emoji: "🏅", Price: 50000, Description: "Awarded only to those who master every journal path.", EffectType: "collectible", Droppable: false, Category: Special},
	{ID: "garden_plot", Name: "Garden Plot", Emoji: "🌿", Price: 500, Description: "A patch of fertile soil for growing vegetables.", EffectType: "permanent", Droppable: false, Category: Special},
	{ID: "tropical_greenhouse", Name: "Tropical Greenhouse", Emoji: "🌿", Price: 1000, Description: "A heated glass structure for coffee and cocoa.", EffectType: "permanent", Droppable: false, Category: Special},
	{ID: "enchanted_orchard", Name: "Enchanted Orchard", Emoji: "🌿", Price: 10000, Description: "A magical floating island. Only legendary fruits grow here.", EffectType: "permanent", Droppable: false, Category: Special},
	{ID: "scarecrow_charm", Name: "Scarecrow Charm", Emoji: "🧿", Price: 0, Description: "A small charm shaped like a winking scarecrow.", EffectType: "collectible", Droppable: false, Category: Special},

	// --- Equipment: Tier 1 (Lv 1, Common) ---
	{ID: "stick", Name: "Wooden Stick", Emoji: "🪵", Price: 50, Description: "A sturdy branch. Decent for a start. (+2 STR)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityCommon, MinLevel: 1, EquipSlot: "weapon", StatSTR: 2},
	{ID: "leather_armor", Name: "Leather Armor", Emoji: "🦺", Price: 300, Description: "Basic protection. (+2 VIT)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityCommon, MinLevel: 1, EquipSlot: "armor", StatVIT: 2},

	// --- Equipment: Tier 2 (Lv 5, Uncommon) ---
	{ID: "iron_pickaxe", Name: "Iron Pickaxe", Emoji: "⛏️", Price: 500, Description: "A miner's best friend. (+4 STR)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityUncommon, MinLevel: 5, EquipSlot: "weapon", StatSTR: 4},
	{ID: "fishing_rod", Name: "Fishing Rod", Emoji: "🎣", Price: 500, Description: "A quality rod. (+4 DEX)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityUncommon, MinLevel: 5, EquipSlot: "weapon", StatDEX: 4},
	{ID: "lucky_charm", Name: "Lucky Charm", Emoji: "🍀", Price: 400, Description: "A four-leaf clover. (+3 LUK)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityUncommon, MinLevel: 5, EquipSlot: "accessory", StatLUK: 3},
	{ID: "miner_helmet", Name: "Miner's Helmet", Emoji: "⛑️", Price: 800, Description: "Thick steel helmet. (+3 VIT, +1 STR)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityUncommon, MinLevel: 5, EquipSlot: "armor", StatVIT: 3, StatSTR: 1},
	// Tutorial finale reward — "The Signal" quest completion.
	{ID: "zenith_blade", Name: "Zenith Blade", Emoji: "⚔️", Price: 0, Description: "Forged from the Vault Guardian's core. It hums with the Generator's light. (+5 STR, +2 VIT)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityRare, MinLevel: 5, EquipSlot: "weapon", StatSTR: 5, StatVIT: 2, ShopExcluded: true},

	// --- Equipment: Tier 3 (Lv 10, Rare) ---
	{ID: "hunters_bow", Name: "Hunter's Bow", Emoji: "🏹", Price: 1200, Description: "A precise bow. (+6 STR)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityRare, MinLevel: 10, EquipSlot: "weapon", StatSTR: 6},
	{ID: "crystal_staff", Name: "Crystal Staff", Emoji: "🔮", Price: 2500, Description: "Pulsing with arcane energy. (+6 INT)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityRare, MinLevel: 10, EquipSlot: "weapon", StatINT: 6},
	{ID: "hunter_cloak", Name: "Hunter's Cloak", Emoji: "🧥", Price: 1500, Description: "Warm and agile. (+4 VIT, +3 DEX)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityRare, MinLevel: 10, EquipSlot: "armor", StatVIT: 4, StatDEX: 3},
	{ID: "golden_ring", Name: "Golden Ring", Emoji: "💍", Price: 2000, Description: "Glows with fortune. (+4 LUK, +1 STR)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityRare, MinLevel: 10, EquipSlot: "accessory", StatLUK: 4, StatSTR: 1},

	// --- Equipment: Tier 4 (Lv 15, Epic) ---
	{ID: "enchanted_robe", Name: "Enchanted Robe", Emoji: "👘", Price: 3000, Description: "Woven with wisdom. (+6 INT, +3 VIT)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityEpic, MinLevel: 15, EquipSlot: "armor", StatINT: 6, StatVIT: 3},
	{ID: "ancient_amulet", Name: "Ancient Amulet", Emoji: "📿", Price: 3000, Description: "Humming with ancient magic. (+4 INT, +4 LUK)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityEpic, MinLevel: 15, EquipSlot: "accessory", StatINT: 4, StatLUK: 4},

	// --- Dragon Slayer Set (Lv 20, Legendary) ---
	{ID: "dragon_slayer_sword", Name: "Dragon Slayer Sword", Emoji: "🗡️", Price: 0, Description: "A blade forged from a dragon's fang. Part of the Dragon Slayer set.", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityLegendary, MinLevel: 20, EquipSlot: "weapon", StatSTR: 12, StatVIT: 4, SetID: "dragon_slayer", SetName: "Dragon Slayer"},
	{ID: "dragon_slayer_armor", Name: "Dragon Slayer Armor", Emoji: "🛡️", Price: 0, Description: "Scale mail of an ancient wyrm. Part of the Dragon Slayer set.", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityLegendary, MinLevel: 20, EquipSlot: "armor", StatVIT: 8, StatSTR: 4, SetID: "dragon_slayer", SetName: "Dragon Slayer"},
	{ID: "dragon_slayer_ring", Name: "Dragon Slayer Ring", Emoji: "💍", Price: 0, Description: "A band of dragon bone and gold. Part of the Dragon Slayer set.", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityLegendary, MinLevel: 20, EquipSlot: "accessory", StatLUK: 4, StatSTR: 6, SetID: "dragon_slayer", SetName: "Dragon Slayer"},
	{ID: "dragon_slayer_talisman", Name: "Dragon Slayer Talisman", Emoji: "📿", Price: 0, Description: "A tooth of the first dragon ever slain. Part of the Dragon Slayer set.", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityLegendary, MinLevel: 20, EquipSlot: "trinket", StatSTR: 6, StatVIT: 5, StatLUK: 3, SetID: "dragon_slayer", SetName: "Dragon Slayer"},

	// --- Shadow Stalker Set (Lv 20, Legendary) ---
	{ID: "shadow_stalker_blade", Name: "Shadow Stalker Blade", Emoji: "🗡️", Price: 0, Description: "A dagger that drinks the light. Part of the Shadow Stalker set.", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityLegendary, MinLevel: 20, EquipSlot: "weapon", StatDEX: 12, StatLUK: 4, SetID: "shadow_stalker", SetName: "Shadow Stalker"},
	{ID: "shadow_stalker_cloak", Name: "Shadow Stalker Cloak", Emoji: "🧥", Price: 0, Description: "Woven from midnight silk. Part of the Shadow Stalker set.", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityLegendary, MinLevel: 20, EquipSlot: "armor", StatDEX: 6, StatVIT: 6, SetID: "shadow_stalker", SetName: "Shadow Stalker"},
	{ID: "shadow_stalker_amulet", Name: "Shadow Stalker Amulet", Emoji: "📿", Price: 0, Description: "A dark gem that whispers secrets. Part of the Shadow Stalker set.", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityLegendary, MinLevel: 20, EquipSlot: "accessory", StatLUK: 8, StatDEX: 4, SetID: "shadow_stalker", SetName: "Shadow Stalker"},
	{ID: "shadow_stalker_charm", Name: "Shadow Stalker Charm", Emoji: "🍀", Price: 0, Description: "Luck woven from shadow itself. Part of the Shadow Stalker set.", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityLegendary, MinLevel: 20, EquipSlot: "trinket", StatDEX: 5, StatLUK: 6, SetID: "shadow_stalker", SetName: "Shadow Stalker"},

	// --- Arcane Weaver Set (Lv 20, Legendary) ---
	{ID: "arcane_weaver_staff", Name: "Arcane Weaver Staff", Emoji: "🪄", Price: 0, Description: "A conduit of raw magic. Part of the Arcane Weaver set.", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityLegendary, MinLevel: 20, EquipSlot: "weapon", StatINT: 12, StatDEX: 4, SetID: "arcane_weaver", SetName: "Arcane Weaver"},
	{ID: "arcane_weaver_robe", Name: "Arcane Weaver Robe", Emoji: "👘", Price: 0, Description: "Ethereal fabric humming with power. Part of the Arcane Weaver set.", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityLegendary, MinLevel: 20, EquipSlot: "armor", StatINT: 8, StatVIT: 4, SetID: "arcane_weaver", SetName: "Arcane Weaver"},
	{ID: "arcane_weaver_crown", Name: "Arcane Weaver Crown", Emoji: "👑", Price: 0, Description: "A circlet of crystallized thought. Part of the Arcane Weaver set.", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityLegendary, MinLevel: 20, EquipSlot: "accessory", StatINT: 8, StatLUK: 4, SetID: "arcane_weaver", SetName: "Arcane Weaver"},
	{ID: "arcane_weaver_orb", Name: "Arcane Weaver Orb", Emoji: "🔮", Price: 0, Description: "A sphere of pure arcane energy. Part of the Arcane Weaver set.", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityLegendary, MinLevel: 20, EquipSlot: "trinket", StatINT: 8, StatDEX: 4, SetID: "arcane_weaver", SetName: "Arcane Weaver"},

	// --- Criminality items ---
	{ID: "mask_of_malveillance", Name: "Mask of Malveillance", Emoji: "🎭", Price: 50000, Description: "An ancient mask that awakens the underworld. Those who wear it gain access to the shadows.", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityLegendary, MinLevel: 25, EquipSlot: "trinket", StatDEX: 5, StatLUK: 5, ShopExcluded: true},
	{ID: "hounds_cloak", Name: "Hound's Cloak", Emoji: "🧥", Price: 1, Description: "The ceremonial cloak of the Iron Lodge. A symbol of the hunt. (+6 VIT, +2 STR)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityEpic, MinLevel: 15, EquipSlot: "armor", StatVIT: 6, StatSTR: 2, ShopExcluded: true},
	{ID: "shadow_cowl", Name: "Shadow Cowl", Emoji: "🕶️", Price: 1, Description: "A dark hood worn by those who walk the Silent Path. (+4 DEX, +2 VIT)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityRare, MinLevel: 10, EquipSlot: "armor", StatDEX: 4, StatVIT: 2, ShopExcluded: true},

	// --- Bounty Hunter gear (Sheriff Vance) ---
	{ID: "iron_shackles", Name: "Iron Shackles", Emoji: "⛓️", Price: 100, Description: "Rusty but reliable manacles. (+3 VIT, +1 STR)", EffectType: "equipment", Droppable: false, Category: Equipment, EquipSlot: "trinket", StatVIT: 3, StatSTR: 1, ShopExcluded: true},
	{ID: "wanted_poster", Name: "Wanted Poster", Emoji: "📜", Price: 500, Description: "Every outlaw's face eventually ends up here. (+4 LUK)", EffectType: "equipment", Droppable: false, Category: Equipment, EquipSlot: "trinket", StatLUK: 4, ShopExcluded: true},
	{ID: "reinforced_badge", Name: "Reinforced Badge", Emoji: "⭐", Price: 3000, Description: "A badge that has stopped a knife or two. (+6 VIT, +2 STR)", EffectType: "equipment", Droppable: false, Category: Equipment, EquipSlot: "accessory", StatVIT: 6, StatSTR: 2, ShopExcluded: true},
	{ID: "bounty_scope", Name: "Bounty Scope", Emoji: "🔭", Price: 12000, Description: "It never loses a trail. (+6 DEX, +4 LUK)", EffectType: "equipment", Droppable: false, Category: Equipment, EquipSlot: "trinket", StatDEX: 6, StatLUK: 4, ShopExcluded: true},
	{ID: "lawbringer_seal", Name: "Lawbringer's Seal", Emoji: "📯", Price: 40000, Description: "The authority of the Iron Lodge itself. (+8 VIT, +4 STR, +2 LUK)", EffectType: "equipment", Droppable: false, Category: Equipment, EquipSlot: "trinket", StatVIT: 8, StatSTR: 4, StatLUK: 2, ShopExcluded: true},

	// --- Thieves' Guild tools (The Whisper) ---
	{ID: "lockpick_set", Name: "Lockpick Set", Emoji: "🔓", Price: 100, Description: "Every lock has a secret. This finds it. (+3 DEX, +1 LUK)", EffectType: "equipment", Droppable: false, Category: Equipment, EquipSlot: "trinket", StatDEX: 3, StatLUK: 1, ShopExcluded: true},
	{ID: "smoke_pellet", Name: "Smoke Pellet", Emoji: "💨", Price: 500, Description: "A quick getaway in a tiny ball. (+4 DEX)", EffectType: "equipment", Droppable: false, Category: Equipment, EquipSlot: "trinket", StatDEX: 4, ShopExcluded: true},
	{ID: "shadow_cloak", Name: "Shadow Cloak", Emoji: "🌑", Price: 3000, Description: "Woven from the night itself. (+6 VIT, +4 DEX)", EffectType: "equipment", Droppable: false, Category: Equipment, EquipSlot: "armor", StatVIT: 6, StatDEX: 4, ShopExcluded: true},
	{ID: "silent_steps", Name: "Silenced Footsteps", Emoji: "👣", Price: 12000, Description: "Boots that never make a sound. (+6 DEX, +4 VIT)", EffectType: "equipment", Droppable: false, Category: Equipment, EquipSlot: "accessory", StatDEX: 6, StatVIT: 4, ShopExcluded: true},
	{ID: "master_key", Name: "Master Key", Emoji: "🗝️", Price: 40000, Description: "It opens doors that don't exist. (+8 LUK, +4 DEX, +2 INT)", EffectType: "equipment", Droppable: false, Category: Equipment, EquipSlot: "trinket", StatLUK: 8, StatDEX: 4, StatINT: 2, ShopExcluded: true},

	// --- Trinkets (Boss League rewards, Lv 15 Epic) ---
	{ID: "spark_shard", Name: "Spark Shard", Emoji: "⚡", Price: 0, Description: "A crackling fragment of Vezir's lightning. (+5 STR, +2 VIT)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityEpic, MinLevel: 15, EquipSlot: "trinket", StatSTR: 5, StatVIT: 2, ShopExcluded: true},
	{ID: "stone_heart", Name: "Stone Heart", Emoji: "🪨", Price: 0, Description: "Tal'Rok's core, dense with resolve. (+4 DEX, +4 VIT)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityEpic, MinLevel: 15, EquipSlot: "trinket", StatDEX: 4, StatVIT: 4, ShopExcluded: true},
	{ID: "storm_core", Name: "Storm Core", Emoji: "🌪️", Price: 0, Description: "Kael's fury, captured in crystal. (+5 DEX, +3 LUK)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityEpic, MinLevel: 15, EquipSlot: "trinket", StatDEX: 5, StatLUK: 3, ShopExcluded: true},
	{ID: "abyss_pearl", Name: "Abyss Pearl", Emoji: "🫧", Price: 0, Description: "Vorgath's gift from the deep. (+6 INT, +3 LUK)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityEpic, MinLevel: 15, EquipSlot: "trinket", StatINT: 6, StatLUK: 3, ShopExcluded: true},
	{ID: "phoenix_crest", Name: "Phoenix Crest", Emoji: "🔥", Price: 0, Description: "Solaris' flame, now yours. (+5 STR, +4 INT, +3 VIT, +3 LUK)", EffectType: "equipment", Droppable: false, Category: Equipment, Rarity: RarityEpic, MinLevel: 15, EquipSlot: "trinket", StatSTR: 5, StatINT: 4, StatVIT: 3, StatLUK: 3, ShopExcluded: true},

	// --- Veil Rift Legendary Set (Lv 25) ---
	{ID: "rift_blade", Name: "Rift-Tempered Blade", Emoji: "⚔️", Price: 8000, Description: "A blade forged in the space between worlds. Part of the Rift Walker set.", EffectType: "equipment", Droppable: false, Category: Equipment, MinLevel: 25, EquipSlot: "weapon", StatSTR: 16, StatDEX: 10, SetID: "rift_walker", SetName: "Rift Walker"},
	{ID: "dechirure_scythe", Name: "Scythe of the Sundered Veil", Emoji: "🜁", Price: 8500, Description: "Forged from a shard of Déchirure's own form. Part of the Rift Walker set.", EffectType: "equipment", Droppable: false, Category: Equipment, MinLevel: 25, EquipSlot: "weapon", StatSTR: 14, StatINT: 14, SetID: "rift_walker", SetName: "Rift Walker"},
	{ID: "rift_cowl", Name: "Cowl of the Veil Walker", Emoji: "👑", Price: 7500, Description: "Woven from threads of fractured reality. Part of the Rift Walker set.", EffectType: "equipment", Droppable: false, Category: Equipment, MinLevel: 25, EquipSlot: "armor", StatVIT: 14, StatDEX: 6, SetID: "rift_walker", SetName: "Rift Walker"},
	{ID: "rift_warden_aegis", Name: "Aegis of the Rift Warden", Emoji: "🛡️", Price: 7800, Description: "Forged from crystallized dimensional tears. Part of the Rift Walker set.", EffectType: "equipment", Droppable: false, Category: Equipment, MinLevel: 25, EquipSlot: "armor", StatVIT: 14, StatSTR: 8, SetID: "rift_walker", SetName: "Rift Walker"},
	{ID: "rift_band", Name: "Band of Dimensional Passage", Emoji: "💍", Price: 7000, Description: "A ring that hums with a thousand planes. Part of the Rift Walker set.", EffectType: "equipment", Droppable: false, Category: Equipment, MinLevel: 25, EquipSlot: "accessory", StatLUK: 10, StatSTR: 3, StatDEX: 3, StatINT: 3, StatVIT: 3, SetID: "rift_walker", SetName: "Rift Walker"},
	{ID: "rift_eye", Name: "Eye of the Rift", Emoji: "👁️", Price: 7200, Description: "It sees what should not be seen. Part of the Rift Walker set.", EffectType: "equipment", Droppable: false, Category: Equipment, MinLevel: 25, EquipSlot: "trinket", StatINT: 10, StatVIT: 8, SetID: "rift_walker", SetName: "Rift Walker"},
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
