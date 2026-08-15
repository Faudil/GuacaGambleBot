package delve

import (
	"fmt"
	"math/rand"
	"strings"

	"guacagamblebot/internal/i18n"
)

type prefixDef struct {
	ID      string
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
	ID        string
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
	ID          string
	Name        string
	Description string
	MinRar      Rarity
}

var prefixes = []prefixDef{
	{ID: "flaming", Name: "Flaming", Emoji: "🔥", StatSTR: 3, MinRar: Uncommon},
	{ID: "shadow", Name: "Shadow", Emoji: "🌑", StatDEX: 3, MinRar: Uncommon},
	{ID: "sturdy", Name: "Sturdy", Emoji: "🛡️", StatVIT: 4, MinRar: Common},
	{ID: "pristine", Name: "Pristine", Emoji: "✨", StatLUK: 2, StatDEX: 2, MinRar: Rare},
	{ID: "frozen", Name: "Frozen", Emoji: "❄️", StatINT: 3, MinRar: Uncommon},
	{ID: "stormforged", Name: "Stormforged", Emoji: "⚡", StatSTR: 2, StatDEX: 2, MinRar: Rare},
	{ID: "arcane", Name: "Arcane", Emoji: "🔮", StatINT: 5, MinRar: Rare},
	{ID: "vampiric", Name: "Vampiric", Emoji: "🩸", StatSTR: 2, StatVIT: 3, MinRar: Rare},
	{ID: "blessed", Name: "Blessed", Emoji: "✨", StatLUK: 4, StatVIT: 2, MinRar: Epic},
	{ID: "cursed", Name: "Cursed", Emoji: "💀", StatSTR: 6, MinRar: Epic, StatVIT: -2},
	{ID: "molten", Name: "Molten", Emoji: "🌋", StatSTR: 4, StatINT: 2, MinRar: Rare},
	{ID: "venomous", Name: "Venomous", Emoji: "🐍", StatDEX: 4, MinRar: Uncommon},
	{ID: "thunderous", Name: "Thunderous", Emoji: "🌩️", StatSTR: 3, StatDEX: 3, MinRar: Rare},
	{ID: "soulbound", Name: "Soulbound", Emoji: "💜", StatINT: 3, StatLUK: 3, MinRar: Epic},
	{ID: "radiant", Name: "Radiant", Emoji: "☀️", StatVIT: 3, StatLUK: 3, MinRar: Epic},
	{ID: "ancient", Name: "Ancient", Emoji: "📜", StatINT: 4, StatSTR: 2, MinRar: Epic},
	{ID: "bone", Name: "Bone", Emoji: "🦴", StatSTR: 2, MinRar: Common},
	{ID: "iron", Name: "Iron", Emoji: "⛓️", StatVIT: 2, MinRar: Common},
	{ID: "crystal", Name: "Crystal", Emoji: "💎", StatINT: 3, StatLUK: 1, MinRar: Uncommon},
	{ID: "wardens", Name: "Warden's", Emoji: "⚜️", StatVIT: 3, StatSTR: 2, MinRar: Rare},
}

var bases = []baseDef{
	{ID: "longsword", Name: "Longsword", Emoji: "⚔️", EquipSlot: "weapon", StatSTR: 2, MinRar: Common},
	{ID: "dagger", Name: "Dagger", Emoji: "🗡️", EquipSlot: "weapon", StatDEX: 2, MinRar: Common},
	{ID: "staff", Name: "Staff", Emoji: "🪄", EquipSlot: "weapon", StatINT: 2, MinRar: Common},
	{ID: "battle_axe", Name: "Battle Axe", Emoji: "🪓", EquipSlot: "weapon", StatSTR: 4, MinRar: Uncommon},
	{ID: "wand", Name: "Wand", Emoji: "✨", EquipSlot: "weapon", StatINT: 3, MinRar: Uncommon},
	{ID: "shortbow", Name: "Shortbow", Emoji: "🏹", EquipSlot: "weapon", StatDEX: 3, MinRar: Uncommon},
	{ID: "leather_cap", Name: "Leather Cap", Emoji: "🧢", EquipSlot: "armor", StatVIT: 1, MinRar: Common},
	{ID: "chainmail", Name: "Chainmail", Emoji: "🛡️", EquipSlot: "armor", StatVIT: 3, MinRar: Uncommon},
	{ID: "robe", Name: "Robe", Emoji: "👘", EquipSlot: "armor", StatINT: 2, MinRar: Common},
	{ID: "plate_armor", Name: "Plate Armor", Emoji: "🪖", EquipSlot: "armor", StatVIT: 5, StatSTR: 1, MinRar: Rare},
	{ID: "amethyst_ring", Name: "Amethyst Ring", Emoji: "💍", EquipSlot: "accessory", StatINT: 2, StatLUK: 1, MinRar: Common},
	{ID: "silver_pendant", Name: "Silver Pendant", Emoji: "📿", EquipSlot: "accessory", StatLUK: 3, MinRar: Uncommon},
	{ID: "emerald_brooch", Name: "Emerald Brooch", Emoji: "💚", EquipSlot: "accessory", StatDEX: 2, StatLUK: 2, MinRar: Rare},
	{ID: "ruby_band", Name: "Ruby Band", Emoji: "❤️", EquipSlot: "accessory", StatSTR: 2, StatVIT: 2, MinRar: Rare},
	{ID: "shadow_cloak", Name: "Shadow Cloak", Emoji: "🌙", EquipSlot: "armor", StatDEX: 3, MinRar: Uncommon},
	{ID: "bone_amulet", Name: "Bone Amulet", Emoji: "🦷", EquipSlot: "accessory", StatSTR: 2, StatLUK: 1, MinRar: Uncommon},
	{ID: "crystal_shield", Name: "Crystal Shield", Emoji: "💠", EquipSlot: "armor", StatVIT: 4, StatINT: 2, MinRar: Rare},
	{ID: "obsidian_dagger", Name: "Obsidian Dagger", Emoji: "🗡️", EquipSlot: "weapon", StatDEX: 4, StatSTR: 1, MinRar: Rare},
	{ID: "spirit_mask", Name: "Spirit Mask", Emoji: "🎭", EquipSlot: "armor", StatINT: 3, StatLUK: 2, MinRar: Epic},
	{ID: "arcane_orb", Name: "Arcane Orb", Emoji: "🔮", EquipSlot: "accessory", StatINT: 4, MinRar: Epic},
}

var suffixes = []suffixDef{
	{ID: "the_vengeful_warden", Name: "the Vengeful Warden", Description: "Forged in the tears of a betrayed guardian.", MinRar: Rare},
	{ID: "the_sunken_court", Name: "the Sunken Court", Description: "Retrieved from the depths of a drowned kingdom.", MinRar: Uncommon},
	{ID: "the_mad_jester", Name: "the Mad Jester", Description: "Cackles with chaotic energy.", MinRar: Rare},
	{ID: "the_forgotten_king", Name: "the Forgotten King", Description: "A relic of a ruler erased from history.", MinRar: Epic},
	{ID: "the_silent_watcher", Name: "the Silent Watcher", Description: "Feels like it's always watching.", MinRar: Common},
	{ID: "the_hollow_priest", Name: "the Hollow Priest", Description: "Hums with a mournful prayer.", MinRar: Uncommon},
	{ID: "the_bone_collector", Name: "the Bone Collector", Description: "Craves the heat of battle.", MinRar: Common},
	{ID: "the_ashen_remnant", Name: "the Ashen Remnant", Description: "Still warm from the fire that consumed its home.", MinRar: Rare},
	{ID: "the_endless_depths", Name: "the Endless Depths", Description: "Seems to drink the light around it.", MinRar: Epic},
	{ID: "the_shattered_oath", Name: "the Shattered Oath", Description: "Remembers a promise that was broken.", MinRar: Uncommon},
	{ID: "the_dying_star", Name: "the Dying Star", Description: "Pulsing with fading celestial light.", MinRar: Legendary},
	{ID: "the_first_flame", Name: "the First Flame", Description: "The very first ember of creation.", MinRar: Legendary},
	{ID: "the_iron_pact", Name: "the Iron Pact", Description: "Bound by an unbreakable contract.", MinRar: Rare},
	{ID: "deep_roots", Name: "Deep Roots", Description: "Grown in the darkest soil.", MinRar: Common},
	{ID: "the_frostbound_heart", Name: "the Frostbound Heart", Description: "Never thaws.", MinRar: Rare},
	{ID: "the_abyss_gazer", Name: "the Abyss Gazer", Description: "Looking into it, it looks back.", MinRar: Epic},
	{ID: "the_lost_pages", Name: "the Lost Pages", Description: "A story cut short.", MinRar: Uncommon},
	{ID: "the_crimson_tide", Name: "the Crimson Tide", Description: "Still damp with the memory of battle.", MinRar: Uncommon},
	{ID: "thunders_call", Name: "Thunder's Call", Description: "Hums with static electricity.", MinRar: Rare},
	{ID: "the_golden_dawn", Name: "the Golden Dawn", Description: "Radiates hope and warmth.", MinRar: Epic},
}

type LootResult struct {
	Item  DelveItem
	Value int
}

// minLevelForRarity maps a delve loot rarity to the minimum character level
// required to equip the generated piece.
func minLevelForRarity(r Rarity) int {
	switch r {
	case Legendary:
		return 20
	case Epic:
		return 15
	case Rare:
		return 10
	case Uncommon:
		return 5
	default:
		return 1
	}
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
		item.PrefixID = pref.ID
	}
	nameParts = append(nameParts, base.Name)
	item.BaseID = base.ID

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
		item.SuffixID = suf.ID
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

func DelveItemName(item DelveItem, lang string) string {
	part := func(ns, id, fallback string) string {
		key := "delve.loot." + ns + "." + id
		tr := i18n.T(key, lang)
		if tr == key {
			return fallback
		}
		return tr
	}
	if item.BaseID == "" {
		key := "delve.loot." + item.ID
		tr := i18n.T(key, lang)
		if tr == key {
			return item.Name
		}
		return tr
	}
	var prefix, suffix string
	if item.PrefixID != "" {
		prefix = part("prefix", item.PrefixID, "")
	}
	if item.SuffixID != "" {
		suffix = part("suffix", item.SuffixID, "")
	}
	format := i18n.T("delve.loot.name_format", lang)
	name := strings.ReplaceAll(format, "{prefix}", prefix)
	name = strings.ReplaceAll(name, "{base}", part("base", item.BaseID, item.Name))
	name = strings.ReplaceAll(name, "{suffix}", suffix)
	return strings.Join(strings.Fields(name), " ")
}

func RarityName(r Rarity, lang string) string {
	key := "delve.loot.rarity." + strings.ToLower(r.String())
	tr := i18n.T(key, lang)
	if tr == key {
		return r.String()
	}
	return tr
}

func LootRewardText(item DelveItem, lang string) string {
	rarEmoji := RarityEmoji[item.Rarity]
	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("%s **%s** %s\n", rarEmoji, DelveItemName(item, lang), item.Emoji))
	slot := i18n.T("delve.loot.slot."+item.EquipSlot, lang)
	if slot == "delve.loot.slot."+item.EquipSlot {
		slot = item.EquipSlot
	}
	sb.WriteString(fmt.Sprintf("`%s` · %s %s\n", RarityName(item.Rarity, lang), i18n.T("delve.loot.slot_label", lang), slot))
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
	sb.WriteString(i18n.T("delve.loot.stats_line", lang, map[string]any{"stats": strings.Join(parts, " · ")}) + "\n")
	if item.SuffixID != "" {
		sb.WriteString(fmt.Sprintf("「%s」\n", i18n.T("delve.loot.suffix_desc."+item.SuffixID, lang)))
	} else if item.Description != "" {
		sb.WriteString(fmt.Sprintf("「%s」\n", item.Description))
	}
	if item.IsCursed {
		sb.WriteString(i18n.T("delve.loot.cursed_line", lang) + "\n")
	}
	if item.IsSoulbound {
		sb.WriteString(i18n.T("delve.loot.soulbound_line", lang) + "\n")
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
