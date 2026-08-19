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

// riftDropIDs are the Veil Rift legendary set pieces granted by raids. Their
// stats, slots and minimum levels live in the central catalog
// (internal/items/catalog.go), which is the single source of truth: a raid
// drop and a shop purchase of the same piece are identical.
var riftDropIDs = []string{
	"rift_blade", "dechirure_scythe", "rift_cowl", "rift_warden_aegis", "rift_band", "rift_eye",
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
	it := items.Get(riftDropIDs[rand.Intn(len(riftDropIDs))])
	if it == nil {
		return nil
	}

	affixDefs := items.RollAffixes(items.RarityLegendary, it.EquipSlot)
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
		userID, it.ID, it.Name, it.Emoji,
		"legendary", it.EquipSlot, it.MinLevel,
		it.StatSTR, it.StatDEX, it.StatINT, it.StatVIT, it.StatLUK,
		affixes, it.SetID)
	if err != nil {
		return nil
	}

	statsParts := []string{}
	if it.StatSTR > 0 {
		statsParts = append(statsParts, fmt.Sprintf("STR+%d", it.StatSTR))
	}
	if it.StatDEX > 0 {
		statsParts = append(statsParts, fmt.Sprintf("DEX+%d", it.StatDEX))
	}
	if it.StatINT > 0 {
		statsParts = append(statsParts, fmt.Sprintf("INT+%d", it.StatINT))
	}
	if it.StatVIT > 0 {
		statsParts = append(statsParts, fmt.Sprintf("VIT+%d", it.StatVIT))
	}
	if it.StatLUK > 0 {
		statsParts = append(statsParts, fmt.Sprintf("LUK+%d", it.StatLUK))
	}

	return &LegendaryDrop{
		UserID:   userID,
		ItemName: it.Name,
		Emoji:    it.Emoji,
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
