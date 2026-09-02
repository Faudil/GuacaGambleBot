// Package glossary powers the /glossary command: for any item or equipment
// base in the catalog, it derives a human-readable list of where it comes
// from by reading the real source-of-truth tables (crafting recipes, mining
// depths, fishing pools, hunt loot, the item's catalog Source) instead of
// duplicating that data, so the text can never drift out of sync with the
// actual game rules.
package glossary

import (
	"fmt"
	"sort"
	"strings"

	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/items"
	craftingsvc "guacagamblebot/internal/service/crafting"
	"guacagamblebot/internal/service/fishing"
	"guacagamblebot/internal/service/hunt"
	"guacagamblebot/internal/service/mining"
	"guacagamblebot/internal/store"
)

type Service struct {
	store *store.Store
}

func New(s *store.Store) *Service {
	return &Service{store: s}
}

// Categories lists the catalog categories shown in the glossary, in display
// order.
var Categories = []items.Category{
	items.Mining,
	items.Fishing,
	items.Farming,
	items.Archeology,
	items.Food,
	items.Tools,
	items.Materials,
	items.Equipment,
	items.Special,
}

// Discovered returns the set of item/equipment-base IDs the user has ever
// obtained.
func (s *Service) Discovered(userID int64) (map[string]bool, error) {
	return s.store.DiscoveredItemIDs(userID)
}

// CategoryProgress returns how many of the given category's catalog items the
// user has discovered, out of how many total.
func (s *Service) CategoryProgress(userID int64, cat items.Category) (int, int, error) {
	discovered, err := s.store.DiscoveredItemIDs(userID)
	if err != nil {
		return 0, 0, err
	}
	entries := items.ItemsByCategory(cat)
	count := 0
	for _, it := range entries {
		if discovered[it.ID] {
			count++
		}
	}
	return count, len(entries), nil
}

// AllProgress returns per-category and total discovery progress for the user.
func (s *Service) AllProgress(userID int64) (map[items.Category]struct{ D, T int }, int, int, error) {
	out := make(map[items.Category]struct{ D, T int })
	totalD, totalT := 0, 0
	for _, cat := range Categories {
		d, t, err := s.CategoryProgress(userID, cat)
		if err != nil {
			return nil, 0, 0, err
		}
		out[cat] = struct{ D, T int }{d, t}
		totalD += d
		totalT += t
	}
	return out, totalD, totalT, nil
}

// AcquisitionSources returns human-readable strings describing every known
// way to obtain itemID, derived live from the game's actual loot/recipe
// tables. Never empty for a real catalog item — falls back to the item's
// coarse catalog Source when no specific activity table references it.
func AcquisitionSources(itemID, lang string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(s string) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}

	if depth, ok := mining.OreMinDepth(itemID); ok {
		add(i18n.T("glossary.source_mining", lang, map[string]any{"depth": depth}))
	}

	for _, fish := range fishing.FishPool {
		if fish.ItemID == itemID {
			add(i18n.T("glossary.source_fishing", lang, map[string]any{"biome": fish.Biome}))
		}
	}

	for _, zone := range hunt.Zones {
		for _, loot := range zone.LootTable {
			if loot.Item == itemID {
				add(i18n.T("glossary.source_hunt", lang, map[string]any{"zone": zone.Key}))
				break
			}
		}
	}

	for _, recipe := range craftingsvc.Recipes {
		if recipe.Result != itemID {
			continue
		}
		var ings []string
		for ing, qty := range recipe.Ingredients {
			ings = append(ings, fmt.Sprintf("%dx %s", qty, items.LocalizedName(ing, lang)))
		}
		sort.Strings(ings)
		add(i18n.T("glossary.source_craft", lang, map[string]any{"ingredients": strings.Join(ings, ", ")}))
	}

	if it := items.Get(itemID); it != nil {
		switch it.Source {
		case items.SourceShop:
			add(i18n.T("glossary.source_shop", lang))
		case items.SourceQuest:
			add(i18n.T("glossary.source_quest", lang))
		case items.SourceBoss:
			add(i18n.T("glossary.source_boss", lang))
		case items.SourceCriminality:
			add(i18n.T("glossary.source_criminality", lang))
		case items.SourceVeil:
			add(i18n.T("glossary.source_veil", lang))
		}
	}

	if len(out) == 0 {
		add(i18n.T("glossary.source_unknown", lang))
	}
	return out
}
