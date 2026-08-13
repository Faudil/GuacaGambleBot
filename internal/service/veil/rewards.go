package veil

import (
	"fmt"
	"math/rand"
	"strings"

	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
)

type LegendaryDrop struct {
	UserID   int64
	ItemName string
	Emoji    string
	StatsStr string
	Affixes  []items.AppliedAffix
}

var legendaryPool = []struct {
	ID      string
	Name    string
	Emoji   string
	Slot    string
	StatSTR int
	StatDEX int
	StatINT int
	StatVIT int
	StatLUK int
}{
	{ID: "rift_blade", Name: "Rift-Tempered Blade", Emoji: "⚔️", Slot: "weapon", StatSTR: 15, StatDEX: 10},
	{ID: "dechirure_scythe", Name: "Scythe of the Sundered Veil", Emoji: "🜁", Slot: "weapon", StatSTR: 12, StatINT: 12},
	{ID: "rift_cowl", Name: "Cowl of the Veil Walker", Emoji: "👑", Slot: "armor", StatVIT: 12, StatDEX: 8},
	{ID: "rift_warden_aegis", Name: "Aegis of the Rift Warden", Emoji: "🛡️", Slot: "armor", StatVIT: 15, StatSTR: 8},
	{ID: "rift_band", Name: "Band of Dimensional Passage", Emoji: "💍", Slot: "accessory", StatLUK: 10, StatSTR: 3, StatDEX: 3, StatINT: 3, StatVIT: 3},
	{ID: "rift_eye", Name: "Eye of the Rift", Emoji: "👁️", Slot: "trinket", StatINT: 12, StatVIT: 8},
}

func GenerateRewards(raid *model.VeilRaid, lang string) (shards map[int64]int, chronicles map[int64]string, drops []LegendaryDrop) {
	shards = map[int64]int{}
	chronicles = map[int64]string{}
	drops = []LegendaryDrop{}

	uniqueFragments := rand.Intn(4) + 4
	scrambledPools := generateScrambledPool(uniqueFragments, lang)

	svc := &Service{}
	states := svc.GetPlayerStates(raid)

	for _, ps := range states {
		if ps.Status != "active" {
			continue
		}

		shardCount := 50 + rand.Intn(100)
		shards[ps.UserID] = shardCount

		svc.store.AddItemRaw(svc.store.DB, ps.UserID, "dimensional_shard", shardCount)

		drop := rollLegendaryForPlayer(ps.UserID)
		if drop != nil {
			drops = append(drops, *drop)
		}

		fragment := scrambledPools[ps.UserID%int64(len(scrambledPools))]
		chronicles[ps.UserID] = i18n.T("veil.rewards.chronicle_fmt", lang, map[string]any{
			"fragment": fragment,
			"others":   len(states) - 1,
			"moment":   randomHeroicMoment(ps, lang),
		})
	}
	return
}

func rollLegendaryForPlayer(userID int64) *LegendaryDrop {
	entry := legendaryPool[rand.Intn(len(legendaryPool))]

	affixDefs := items.RollAffixes(items.RarityLegendary, entry.Slot)
	affixes := make([]items.AppliedAffix, len(affixDefs))
	for i, a := range affixDefs {
		affixes[i] = items.AppliedAffix{
			ID:    a.ID,
			Name:  a.Name,
			Stat:  a.Stat,
			Value: items.RollAffixValue(a),
		}
	}

	svc := &Service{}
	_, err := svc.store.CreateEquipmentFromAffixes(
		userID, entry.ID, entry.Name, entry.Emoji,
		"legendary", entry.Slot, 25,
		entry.StatSTR, entry.StatDEX, entry.StatINT, entry.StatVIT, entry.StatLUK,
		affixes, "rift_walker")
	if err != nil {
		return nil
	}

	statsParts := []string{}
	if entry.StatSTR > 0 {
		statsParts = append(statsParts, fmt.Sprintf("STR+%d", entry.StatSTR))
	}
	if entry.StatDEX > 0 {
		statsParts = append(statsParts, fmt.Sprintf("DEX+%d", entry.StatDEX))
	}
	if entry.StatINT > 0 {
		statsParts = append(statsParts, fmt.Sprintf("INT+%d", entry.StatINT))
	}
	if entry.StatVIT > 0 {
		statsParts = append(statsParts, fmt.Sprintf("VIT+%d", entry.StatVIT))
	}
	if entry.StatLUK > 0 {
		statsParts = append(statsParts, fmt.Sprintf("LUK+%d", entry.StatLUK))
	}

	return &LegendaryDrop{
		UserID:   userID,
		ItemName: entry.Name,
		Emoji:    entry.Emoji,
		StatsStr: strings.Join(statsParts, ", "),
		Affixes:  affixes,
	}
}

func generateScrambledPool(count int, lang string) []string {
	base := make([]string, 7)
	for i := 0; i < 7; i++ {
		base[i] = i18n.T(fmt.Sprintf("veil.chronicle.frag_%d", i), lang)
	}
	rand.Shuffle(len(base), func(i, j int) { base[i], base[j] = base[j], base[i] })
	if count > len(base) {
		count = len(base)
	}
	return base[:count]
}

func randomHeroicMoment(ps model.VeilPlayerState, lang string) string {
	return i18n.T(fmt.Sprintf("veil.chronicle.heroic_%d", rand.Intn(5)), lang)
}

var _ = fmt.Sprintf
