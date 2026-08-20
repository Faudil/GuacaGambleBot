package delve

import (
	"fmt"
	"math/rand"
	"strings"

	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/items"
)

type LootResult struct {
	Item  DelveItem
	Gold  int
	Heal  int
	Value int
}

// equipmentDropChance is the share of delve room loot rolls that yield
// equipment; the rest become a non-equipment reward (gold, misc items or
// a small heal).
const equipmentDropChance = 0.55

// GenerateRoomLoot rolls a cleared room's reward. Equipment is rarer than
// the other finds: 55% of rolls produce gear (GenerateLoot), the rest are
// gold, misc items or a small heal.
func GenerateRoomLoot(zone string, floor int, lukBonus float64) *LootResult {
	if rand.Float64() < equipmentDropChance {
		return GenerateLoot(zone, floor, lukBonus)
	}
	return generateNonEquipmentLoot(zone, floor)
}

func generateNonEquipmentLoot(zone string, floor int) *LootResult {
	switch r := rand.Intn(100); {
	case r < 35: // gold find
		return &LootResult{Gold: GoldReward(zone, floor) * 3}
	case r < 65: // misc items
		return &LootResult{Item: randomMiscItem()}
	default: // small heal
		heal := 15 + floor*2
		if heal > 40 {
			heal = 40
		}
		return &LootResult{Heal: heal}
	}
}

// randomMiscItem rolls a stackable non-equipment find (depth shards or
// ancient coins).
func randomMiscItem() DelveItem {
	switch rand.Intn(2) {
	case 0:
		return DelveItem{
			ID: "depth_shard", Name: "Depth Shard", Emoji: "💎", Rarity: Rare, Quantity: 2 + rand.Intn(3),
		}
	default:
		return DelveItem{
			ID: "ancient_coins", Name: "Ancient Coins", Emoji: "🪙", Rarity: Uncommon, Quantity: 3 + rand.Intn(4),
			Description: "Coins minted long before the Undercroft swallowed the city.",
		}
	}
}

// minLevelForRarity maps a delve loot rarity to the minimum character level
// required to equip the generated piece. Levels are aligned with the delve
// curve (RecommendedLevel(floor) = 1 + 2*(floor-1)) so gear is usable close
// to the depth where it drops.
func minLevelForRarity(r Rarity) int {
	switch r {
	case Legendary:
		return 15
	case Epic:
		return 10
	case Rare:
		return 5
	case Uncommon:
		return 3
	default:
		return 1
	}
}

// floorRarityBonus shifts the rarity roll toward higher tiers as the player
// descends. It grows from 0 on floor 1 up to a 0.15 cap around floor 13, so
// early floors mostly yield Common/Uncommon while the Abyss rewards Epics
// and Legendaries.
func floorRarityBonus(floor int) float64 {
	b := float64(floor-1) * 0.0125
	if b > 0.15 {
		return 0.15
	}
	return b
}

func rollRarity(floor int, lukBonus float64) Rarity {
	r := rand.Float64() - lukBonus - floorRarityBonus(floor)
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

	var candidates []items.DelveBase
	for _, b := range items.DelveBases {
		if b.MinRar.Rank() <= rar.Rank() {
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
		var candidates []items.DelvePrefix
		for _, p := range items.DelvePrefixes {
			if p.MinRar.Rank() <= rar.Rank() {
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
		var candidates []items.DelveSuffix
		for _, s := range items.DelveSuffixes {
			if s.MinRar.Rank() <= rar.Rank() {
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
	if item.EquipSlot == "" {
		return sb.String()
	}
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

// AssignSetName tags a delve drop with its zone's set (Rare/Epic only —
// Legendaries are standalone). Set pieces are mid-game gear: collecting and
// equipping them grants set bonuses (see items.CalculateSetBonuses).
func AssignSetName(item *DelveItem, zone string) {
	if item.Rarity < Rare || item.Rarity >= Legendary {
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
