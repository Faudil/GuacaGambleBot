package delve

import (
	"fmt"
	"math/rand"
	"strings"
)

type prefixDef struct {
	Name    string
	Emoji   string
	StatSTR int
	StatDEX int
	StatINT int
	StatVIT int
	StatLUK int
	MinRar  Rarity
}

type baseDef struct {
	Name      string
	Emoji     string
	EquipSlot string
	StatSTR   int
	StatDEX   int
	StatINT   int
	StatVIT   int
	StatLUK   int
	MinRar    Rarity
}

type suffixDef struct {
	Name        string
	Description string
	MinRar      Rarity
}

var prefixes = []prefixDef{
	{Name: "Flaming", Emoji: "🔥", StatSTR: 3, MinRar: Uncommon},
	{Name: "Shadow", Emoji: "🌑", StatDEX: 3, MinRar: Uncommon},
	{Name: "Sturdy", Emoji: "🛡️", StatVIT: 4, MinRar: Common},
	{Name: "Pristine", Emoji: "✨", StatLUK: 2, StatDEX: 2, MinRar: Rare},
	{Name: "Frozen", Emoji: "❄️", StatINT: 3, MinRar: Uncommon},
	{Name: "Stormforged", Emoji: "⚡", StatSTR: 2, StatDEX: 2, MinRar: Rare},
	{Name: "Arcane", Emoji: "🔮", StatINT: 5, MinRar: Rare},
	{Name: "Vampiric", Emoji: "🩸", StatSTR: 2, StatVIT: 3, MinRar: Rare},
	{Name: "Blessed", Emoji: "✨", StatLUK: 4, StatVIT: 2, MinRar: Epic},
	{Name: "Cursed", Emoji: "💀", StatSTR: 6, MinRar: Epic, StatVIT: -2},
	{Name: "Molten", Emoji: "🌋", StatSTR: 4, StatINT: 2, MinRar: Rare},
	{Name: "Venomous", Emoji: "🐍", StatDEX: 4, MinRar: Uncommon},
	{Name: "Thunderous", Emoji: "🌩️", StatSTR: 3, StatDEX: 3, MinRar: Rare},
	{Name: "Soulbound", Emoji: "💜", StatINT: 3, StatLUK: 3, MinRar: Epic},
	{Name: "Radiant", Emoji: "☀️", StatVIT: 3, StatLUK: 3, MinRar: Epic},
	{Name: "Ancient", Emoji: "📜", StatINT: 4, StatSTR: 2, MinRar: Epic},
	{Name: "Bone", Emoji: "🦴", StatSTR: 2, MinRar: Common},
	{Name: "Iron", Emoji: "⛓️", StatVIT: 2, MinRar: Common},
	{Name: "Crystal", Emoji: "💎", StatINT: 3, StatLUK: 1, MinRar: Uncommon},
	{Name: "Warden's", Emoji: "⚜️", StatVIT: 3, StatSTR: 2, MinRar: Rare},
}

var bases = []baseDef{
	{Name: "Longsword", Emoji: "⚔️", EquipSlot: "weapon", StatSTR: 2, MinRar: Common},
	{Name: "Dagger", Emoji: "🗡️", EquipSlot: "weapon", StatDEX: 2, MinRar: Common},
	{Name: "Staff", Emoji: "🪄", EquipSlot: "weapon", StatINT: 2, MinRar: Common},
	{Name: "Battle Axe", Emoji: "🪓", EquipSlot: "weapon", StatSTR: 4, MinRar: Uncommon},
	{Name: "Wand", Emoji: "✨", EquipSlot: "weapon", StatINT: 3, MinRar: Uncommon},
	{Name: "Shortbow", Emoji: "🏹", EquipSlot: "weapon", StatDEX: 3, MinRar: Uncommon},
	{Name: "Leather Cap", Emoji: "🧢", EquipSlot: "armor", StatVIT: 1, MinRar: Common},
	{Name: "Chainmail", Emoji: "🛡️", EquipSlot: "armor", StatVIT: 3, MinRar: Uncommon},
	{Name: "Robe", Emoji: "👘", EquipSlot: "armor", StatINT: 2, MinRar: Common},
	{Name: "Plate Armor", Emoji: "🪖", EquipSlot: "armor", StatVIT: 5, StatSTR: 1, MinRar: Rare},
	{Name: "Amethyst Ring", Emoji: "💍", EquipSlot: "accessory", StatINT: 2, StatLUK: 1, MinRar: Common},
	{Name: "Silver Pendant", Emoji: "📿", EquipSlot: "accessory", StatLUK: 3, MinRar: Uncommon},
	{Name: "Emerald Brooch", Emoji: "💚", EquipSlot: "accessory", StatDEX: 2, StatLUK: 2, MinRar: Rare},
	{Name: "Ruby Band", Emoji: "❤️", EquipSlot: "accessory", StatSTR: 2, StatVIT: 2, MinRar: Rare},
	{Name: "Shadow Cloak", Emoji: "🌙", EquipSlot: "armor", StatDEX: 3, MinRar: Uncommon},
	{Name: "Bone Amulet", Emoji: "🦷", EquipSlot: "accessory", StatSTR: 2, StatLUK: 1, MinRar: Uncommon},
	{Name: "Crystal Shield", Emoji: "💠", EquipSlot: "armor", StatVIT: 4, StatINT: 2, MinRar: Rare},
	{Name: "Obsidian Dagger", Emoji: "🗡️", EquipSlot: "weapon", StatDEX: 4, StatSTR: 1, MinRar: Rare},
	{Name: "Spirit Mask", Emoji: "🎭", EquipSlot: "armor", StatINT: 3, StatLUK: 2, MinRar: Epic},
	{Name: "Arcane Orb", Emoji: "🔮", EquipSlot: "accessory", StatINT: 4, MinRar: Epic},
}

var suffixes = []suffixDef{
	{Name: "the Vengeful Warden", Description: "Forged in the tears of a betrayed guardian.", MinRar: Rare},
	{Name: "the Sunken Court", Description: "Retrieved from the depths of a drowned kingdom.", MinRar: Uncommon},
	{Name: "the Mad Jester", Description: "Cackles with chaotic energy.", MinRar: Rare},
	{Name: "the Forgotten King", Description: "A relic of a ruler erased from history.", MinRar: Epic},
	{Name: "the Silent Watcher", Description: "Feels like it's always watching.", MinRar: Common},
	{Name: "the Hollow Priest", Description: "Hums with a mournful prayer.", MinRar: Uncommon},
	{Name: "the Bone Collector", Description: "Craves the heat of battle.", MinRar: Common},
	{Name: "the Ashen Remnant", Description: "Still warm from the fire that consumed its home.", MinRar: Rare},
	{Name: "the Endless Depths", Description: "Seems to drink the light around it.", MinRar: Epic},
	{Name: "the Shattered Oath", Description: "Remembers a promise that was broken.", MinRar: Uncommon},
	{Name: "the Dying Star", Description: "Pulsing with fading celestial light.", MinRar: Legendary},
	{Name: "the First Flame", Description: "The very first ember of creation.", MinRar: Legendary},
	{Name: "the Iron Pact", Description: "Bound by an unbreakable contract.", MinRar: Rare},
	{Name: "Deep Roots", Description: "Grown in the darkest soil.", MinRar: Common},
	{Name: "the Frostbound Heart", Description: "Never thaws.", MinRar: Rare},
	{Name: "the Abyss Gazer", Description: "Looking into it, it looks back.", MinRar: Epic},
	{Name: "the Lost Pages", Description: "A story cut short.", MinRar: Uncommon},
	{Name: "the Crimson Tide", Description: "Still damp with the memory of battle.", MinRar: Uncommon},
	{Name: "Thunder's Call", Description: "Hums with static electricity.", MinRar: Rare},
	{Name: "the Golden Dawn", Description: "Radiates hope and warmth.", MinRar: Epic},
}

type LootResult struct {
	Item  DelveItem
	Value int
}

func rollRarity(floor int, lukBonus float64) Rarity {
	r := rand.Float64() - lukBonus
	switch {
	case r < 0.05:
		return Legendary
	case r < 0.15:
		return Epic
	case r < 0.30:
		return Rare
	case r < 0.60:
		return Uncommon
	default:
		return Common
	}
}

func GenerateLoot(zone string, floor int, lukBonus float64) *LootResult {
	rar := rollRarity(floor, lukBonus)
	rarityMod := map[Rarity]int{Common: 1, Uncommon: 2, Rare: 3, Epic: 5, Legendary: 10}

	var candidates []baseDef
	for _, b := range bases {
		if b.MinRar <= rar {
			candidates = append(candidates, b)
		}
	}
	base := candidates[rand.Intn(len(candidates))]

	item := DelveItem{
		Emoji:     base.Emoji,
		EquipSlot: base.EquipSlot,
		Rarity:    rar,
		Quantity:  1,
	}

	nameParts := []string{}
	statSTR := base.StatSTR
	statDEX := base.StatDEX
	statINT := base.StatINT
	statVIT := base.StatVIT
	statLUK := base.StatLUK

	if rar >= Uncommon && rand.Float64() < 0.6 {
		var candidates []prefixDef
		for _, p := range prefixes {
			if p.MinRar <= rar {
				candidates = append(candidates, p)
			}
		}
		pref := candidates[rand.Intn(len(candidates))]
		nameParts = append(nameParts, pref.Name)
		statSTR += pref.StatSTR
		statDEX += pref.StatDEX
		statINT += pref.StatINT
		statVIT += pref.StatVIT
		statLUK += pref.StatLUK
		item.Emoji = pref.Emoji
	}
	nameParts = append(nameParts, base.Name)

	desc := ""
	if rar >= Uncommon && rand.Float64() < 0.5 {
		var candidates []suffixDef
		for _, s := range suffixes {
			if s.MinRar <= rar {
				candidates = append(candidates, s)
			}
		}
		suf := candidates[rand.Intn(len(candidates))]
		nameParts = append(nameParts, suf.Name)
		desc = suf.Description
	}

	item.Name = strings.Join(nameParts, " ")
	item.ID = fmt.Sprintf("delve_%s", strings.ToLower(strings.ReplaceAll(item.Name, " ", "_")))
	item.Description = desc
	item.StatSTR = statSTR * rarityMod[rar] / 2
	item.StatDEX = statDEX * rarityMod[rar] / 2
	item.StatINT = statINT * rarityMod[rar] / 2
	item.StatVIT = statVIT * rarityMod[rar] / 2
	item.StatLUK = statLUK * rarityMod[rar] / 2

	if item.StatSTR < 0 {
		item.IsCursed = true
	}
	if rar >= Epic && rand.Float64() < 0.2 {
		item.IsSoulbound = true
	}

	value := rarityMod[rar] * 25
	return &LootResult{Item: item, Value: value}
}

func LootRewardText(item DelveItem) string {
	rarEmoji := RarityEmoji[item.Rarity]
	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("%s **%s** %s\n", rarEmoji, item.Name, item.Emoji))
	sb.WriteString(fmt.Sprintf("`%s` · Slot: %s\n", item.Rarity.String(), item.EquipSlot))
	var parts []string
	if item.StatSTR > 0 {
		parts = append(parts, fmt.Sprintf("STR+%d", item.StatSTR))
	}
	if item.StatDEX > 0 {
		parts = append(parts, fmt.Sprintf("DEX+%d", item.StatDEX))
	}
	if item.StatINT > 0 {
		parts = append(parts, fmt.Sprintf("INT+%d", item.StatINT))
	}
	if item.StatVIT > 0 {
		parts = append(parts, fmt.Sprintf("VIT+%d", item.StatVIT))
	}
	if item.StatLUK > 0 {
		parts = append(parts, fmt.Sprintf("LUK+%d", item.StatLUK))
	}
	sb.WriteString(fmt.Sprintf("├ Stats: %s\n", strings.Join(parts, " · ")))
	if item.Description != "" {
		sb.WriteString(fmt.Sprintf("「%s」\n", item.Description))
	}
	if item.IsCursed {
		sb.WriteString("⚠️ **Cursed!** Negative stat effect active.\n")
	}
	if item.IsSoulbound {
		sb.WriteString("💜 **Soulbound** — will be recorded in your chronicle.\n")
	}
	return sb.String()
}

func GoldReward(zone string, floor int) int {
	base := map[string]int{
		"crypt": 10, "fungal_wilds": 20, "forge_district": 35, "abyss": 50,
	}
	return base[zone] + floor*5 + rand.Intn(10)
}

func MaybeDropVeilKey(zone string, floor int) *DelveItem {
	if floor < 7 {
		return nil
	}
	chance := 0.02
	if floor >= 10 {
		chance = 0.05
	}
	if zone == "abyss" {
		chance = 0.08
	}
	if rand.Float64() < chance {
		return &DelveItem{
			ID: "veil_key", Name: "Veil Key", Emoji: "🔮", Rarity: Epic, Quantity: 1,
			Description: "A shimmering key that tears the fabric of reality. Required to enter the Veil Rift.",
		}
	}
	return nil
}

func MaybeDropKey(zone string, floor int) *DelveItem {
	if rand.Intn(100) >= KeyDropChance(floor) {
		return nil
	}
	return &DelveItem{
		ID: "dungeon_key", Name: "Dungeon Key", Emoji: "🔑", Rarity: Common, Quantity: 1,
		Description: "A rusted iron key. It might open something in the depths.",
	}
}

var zoneSetNames = map[string]struct {
	SetID   string
	SetName string
	Zone    string
}{
	"crypt":          {"crypt_lord_regalia", "Crypt Lord's Regalia", "crypt"},
	"fungal_wilds":   {"fungal_raiment", "Fungal Raiment", "fungal_wilds"},
	"forge_district": {"forge_master_arsenal", "Forge Master's Arsenal", "forge_district"},
}

func AssignSetName(item *DelveItem, zone string) {
	if item.Rarity < Rare {
		return
	}
	set, ok := zoneSetNames[zone]
	if !ok {
		return
	}
	if rand.Intn(100) >= 15 {
		return
	}
	item.SetName = set.SetName
	item.ID = fmt.Sprintf("%s_%s", set.SetID, item.ID)
}
