package items

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
}

var all = []Item{
	// --- Mining ---
	{ID: "pebble",        Name: "Pebble",          Emoji: "🪨", Price: 1,   Description: "A useless little rock. Totally worthless.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "coal",          Name: "Coal",            Emoji: "🪨", Price: 5,   Description: "Great for keeping warm.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "iron_ore",      Name: "Iron Ore",        Emoji: "⛏️", Price: 10,  Description: "Useful for forging sturdy tools.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "copper_ore",    Name: "Copper Ore",      Emoji: "⛏️", Price: 15,  Description: "Useful for forging sturdy tools.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "silver_ore",    Name: "Silver Ore",      Emoji: "⛏️", Price: 25,  Description: "Useful for forging sturdy tools.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "gold_nugget",   Name: "Gold Nugget",     Emoji: "✨", Price: 50,  Description: "Shiny! Merchants love this.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "platinum",      Name: "Platinum",        Emoji: "✨", Price: 75,  Description: "Shiny! Merchants love this.", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "emerald",       Name: "Emerald",         Emoji: "💚", Price: 100, Description: "AMAZING! Worth a fortune!", EffectType: "resource", Droppable: true, Category: Mining},
	{ID: "rough_diamond", Name: "Rough Diamond",   Emoji: "💎", Price: 300, Description: "AMAZING! Worth a fortune!", EffectType: "resource", Droppable: true, Category: Mining},

	// --- Fishing ---
	{ID: "old_boot",         Name: "Old Boot",           Emoji: "🥾", Price: 1,   Description: "An old boot. Worthless.", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "trout",            Name: "Trout",              Emoji: "🐟", Price: 10,  Description: "Freshwater fish.", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "salmon",           Name: "Salmon",             Emoji: "🐟", Price: 10,  Description: "Perfect for sushi.", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "sardine",          Name: "Sardine",            Emoji: "🐟", Price: 15,  Description: "A small saltwater fish.", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "carp",             Name: "Carp",               Emoji: "🐟", Price: 25,  Description: "The best freshwater fish.", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "pufferfish",       Name: "Pufferfish",         Emoji: "🐡", Price: 50,  Description: "Watch out, it stings!", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "swordfish",        Name: "Swordfish",          Emoji: "🐟", Price: 150, Description: "A majestic fighting fish.", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "shark",            Name: "Shark",              Emoji: "🦈", Price: 100, Description: "AMAZING! Worth a fortune!", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "whale",            Name: "Whale",              Emoji: "🐋", Price: 300, Description: "AMAZING! Worth a fortune!", EffectType: "resource", Droppable: true, Category: Fishing},
	{ID: "kraken_tentacle",  Name: "Kraken Tentacle",    Emoji: "🦑", Price: 500, Description: "YOU CAUGHT A MONSTER?!", EffectType: "resource", Droppable: true, Category: Fishing},

	// --- Farming ---
	{ID: "rotten_plant",  Name: "Rotten Plant",  Emoji: "🌿", Price: 0,   Description: "You mismanaged your farm...", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "wheat",         Name: "Wheat",         Emoji: "🌾", Price: 5,   Description: "Essential for making bread.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "oat",           Name: "Oat",           Emoji: "🌾", Price: 8,   Description: "Perfect for breakfast.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "corn",          Name: "Corn",          Emoji: "🌽", Price: 12,  Description: "Also makes popcorn!", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "potato",        Name: "Potato",        Emoji: "🥔", Price: 20,  Description: "You can make vodka from it…", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "tomato",        Name: "Tomato",        Emoji: "🍅", Price: 25,  Description: "Fruit or vegetable? The debate continues.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "pumpkin",       Name: "Pumpkin",       Emoji: "🎃", Price: 40,  Description: "Perfect for Halloween.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "coffee_bean",   Name: "Coffee Bean",   Emoji: "🫘", Price: 60,  Description: "The black gold of the morning.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "cocoa_bean",    Name: "Cocoa Bean",    Emoji: "🫘", Price: 75,  Description: "The main ingredient for happiness (chocolate).", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "strawberry",    Name: "Strawberry",    Emoji: "🍓", Price: 90,  Description: "Red, sweet and juicy.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "golden_apple",  Name: "Golden Apple",  Emoji: "🍎", Price: 150, Description: "It shines with a magical glow.", EffectType: "resource", Droppable: true, Category: Farming},
	{ID: "star_fruit",    Name: "Star Fruit",    Emoji: "⭐", Price: 250, Description: "A cosmic fruit from another dimension.", EffectType: "resource", Droppable: true, Category: Farming},

	// --- Archeology ---
	{ID: "bone_dust",           Name: "Bone Dust",           Emoji: "🦴", Price: 1,    Description: "Bone dust from a completely destroyed fossil.", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "damaged_fossil",      Name: "Damaged Fossil",      Emoji: "🦴", Price: 50,   Description: "A poorly extracted fossil that lost its value.", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "common_fossil",       Name: "Common Fossil",       Emoji: "🦴", Price: 150,  Description: "An intact fossil of a common animal.", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "rare_fossil",         Name: "Rare Fossil",         Emoji: "🦴", Price: 300,  Description: "An intact fossil of a rare animal.", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "epic_fossil",         Name: "Epic Fossil",         Emoji: "🦴", Price: 500,  Description: "An intact fossil of an epic animal.", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "legendary_fragment",  Name: "Legendary Fragment",  Emoji: "🦖", Price: 1000, Description: "A legendary T-Rex fragment!", EffectType: "resource", Droppable: true, Category: Archeology},
	{ID: "pure_dna",            Name: "Pure DNA",            Emoji: "🧬", Price: 3000, Description: "Perfectly preserved dinosaur DNA. Amazing!", EffectType: "resource", Droppable: true, Category: Archeology},

	// --- Seeds (Materials) ---
	{ID: "wheat_seed",        Name: "Wheat Seed",        Emoji: "🌱", Price: 2,   Description: "Plant to get wheat (5 min).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "oat_seed",          Name: "Oat Seed",          Emoji: "🌱", Price: 3,   Description: "Plant to get oat (10 min).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "corn_seed",         Name: "Corn Seed",         Emoji: "🌱", Price: 5,   Description: "Plant to get corn (30 min).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "potato_seed",       Name: "Potato Seed",       Emoji: "🌱", Price: 8,   Description: "Plant to get potato (1h).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "tomato_seed",       Name: "Tomato Seed",       Emoji: "🌱", Price: 10,  Description: "Plant to get tomato (2h).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "pumpkin_seed",      Name: "Pumpkin Seed",     Emoji: "🌱", Price: 15,  Description: "Plant to get pumpkin (4h).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "coffee_seed",       Name: "Coffee Seed",      Emoji: "🌱", Price: 25,  Description: "Plant to get coffee (8h).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "cocoa_seed",        Name: "Cocoa Seed",       Emoji: "🌱", Price: 30,  Description: "Plant to get cocoa (12h).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "strawberry_seed",   Name: "Strawberry Seed",  Emoji: "🌱", Price: 40,  Description: "Plant to get strawberry (18h).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "golden_apple_seed", Name: "Golden Apple Seed", Emoji: "🌱", Price: 75,  Description: "Plant to get golden apple (24h).", EffectType: "resource", Droppable: true, Category: Materials},
	{ID: "star_fruit_seed",   Name: "Star Fruit Seed",  Emoji: "🌱", Price: 125, Description: "Plant to get star fruit (48h).", EffectType: "resource", Droppable: true, Category: Materials},

	// --- Tools ---
	{ID: "beer",            Name: "Beer",              Emoji: "🍺", Price: 50,   Description: "The miner's drink! Resets !mine cooldown.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "coffee",          Name: "Coffee",            Emoji: "☕", Price: 50,   Description: "Wakes you up. Resets !daily cooldown.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "bow",             Name: "Bow",               Emoji: "🏹", Price: 300,  Description: "Helps with hunting! Resets !hunt cooldown.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "fertilizer",      Name: "Fertilizer",        Emoji: "🧪", Price: 200,  Description: "Accelerates crop growth! Resets !farm cooldown.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "hook",            Name: "Hook",              Emoji: "🪝", Price: 200,  Description: "Attracts fish! Resets !fish cooldown.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "forget_potion",   Name: "Forget Potion",     Emoji: "🧪", Price: 2500, Description: "Resets your pet to level 10.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "fortune_cookie",  Name: "Fortune Cookie",    Emoji: "🥠", Price: 20,   Description: "A delicious cookie with a premonitory message.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "rusty_magnet",    Name: "Rusty Magnet",      Emoji: "🧲", Price: 30,   Description: "Use it to find some pocket change.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "magnet",          Name: "Magnet",            Emoji: "🧲", Price: 50,   Description: "Use it to find money on the ground.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "electric_magnet", Name: "Electric Magnet",   Emoji: "🧲", Price: 500,  Description: "Use it to find lots of money!", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "rigged_coin",     Name: "Rigged Coin",       Emoji: "🪙", Price: 200,  Description: "Boosts your luck. Raises coinflip odds to 75%.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "casino_token",    Name: "Casino Token",      Emoji: "🎰", Price: 50,   Description: "Gives you another chance at !casino.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "vip_ticket",      Name: "VIP Ticket",        Emoji: "🎟️", Price: 100,  Description: "Resets your casino and coinflip limits.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "identity_scroll", Name: "Identity Scroll",   Emoji: "📜", Price: 500,  Description: "Randomly changes your server nickname.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "thieves_glove",   Name: "Thieve's Glove",    Emoji: "🧤", Price: 20,   Description: "Use !rob @target with these gloves.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "scratch_ticket",  Name: "Scratch Ticket",    Emoji: "🎰", Price: 100,  Description: "Scratch to win up to $1000 instantly!", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "data_disk",       Name: "Data Disk",         Emoji: "💾", Price: 50,   Description: "A corrupted Zenith memory disk.", EffectType: "consumable", Droppable: false, Category: Tools},
	{ID: "old_journal",     Name: "Old Journal",       Emoji: "📖", Price: 30,   Description: "A dusty notebook written by a survivor.", EffectType: "consumable", Droppable: false, Category: Tools},

	// --- Special ---
	{ID: "mystery_egg",      Name: "Mystery Egg",       Emoji: "🥚", Price: 6000,  Description: "A trembling egg... Type !hatch to open it!", EffectType: "consumable", Droppable: false, Category: Special},
	{ID: "season_egg",       Name: "Season Egg",        Emoji: "🥚", Price: 12000, Description: "A trembling egg... Type !hatch to open it!", EffectType: "consumable", Droppable: false, Category: Special},
	{ID: "boss_trophy",      Name: "Boss Trophy",       Emoji: "🏆", Price: 10000, Description: "A legendary trophy for defeating the boss.", EffectType: "collectible", Droppable: false, Category: Special},
	{ID: "garden_plot",      Name: "Garden Plot",       Emoji: "🌿", Price: 500,   Description: "A patch of fertile soil for growing vegetables.", EffectType: "permanent", Droppable: false, Category: Special},
	{ID: "tropical_greenhouse", Name: "Tropical Greenhouse", Emoji: "🌿", Price: 1000, Description: "A heated glass structure for coffee and cocoa.", EffectType: "permanent", Droppable: false, Category: Special},
	{ID: "enchanted_orchard",   Name: "Enchanted Orchard",   Emoji: "🌿", Price: 10000, Description: "A magical floating island. Only legendary fruits grow here.", EffectType: "permanent", Droppable: false, Category: Special},
}

var byID = func() map[string]*Item {
	m := make(map[string]*Item, len(all))
	for i := range all {
		m[all[i].ID] = &all[i]
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

func Get(nameOrID string) *Item {
	if it, ok := byID[nameOrID]; ok {
		return it
	}
	return byName[nameOrID]
}

func AllItems() []Item {
	return all
}

func ItemsByCategory(cat Category) []Item {
	return byCategory[cat]
}
