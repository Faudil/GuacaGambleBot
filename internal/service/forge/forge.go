package forge

import (
	"encoding/json"
	"errors"
	"math/rand"
	"strings"

	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	furnituresvc "guacagamblebot/internal/service/furniture"
	"guacagamblebot/internal/store"
)

var (
	ErrNoForge          = errors.New("you need a Forge placed in your house to fuse equipment")
	ErrNeedArcaneForge  = errors.New("epic to legendary fusion requires the Arcane Forge")
	ErrResearchRequired = errors.New("the fusion research for this rarity is not completed")
	ErrNeedFive         = errors.New("fusion requires exactly 5 items of the same rarity")
	ErrNotOwned         = errors.New("you don't own that item")
	ErrEquippedItem     = errors.New("the item must be unequipped first")
	ErrWrongRarity      = errors.New("all fused items must share the same rarity")
	ErrNoItems          = errors.New("no unequipped items of that rarity")
	ErrUnknownItem      = errors.New("item not found")
)

// FuseCount is the number of items consumed by one fusion.
const FuseCount = 5

// RarityTiers lists the rarities in fusion order: fusing 5 items of one tier
// produces one random item of the next tier.
var RarityTiers = []items.Rarity{
	items.RarityCommon,
	items.RarityUncommon,
	items.RarityRare,
	items.RarityEpic,
	items.RarityLegendary,
}

// NextRarity returns the rarity produced by fusing items of the given rarity.
// The second return is false when the rarity cannot be fused (legendary).
func NextRarity(from items.Rarity) (items.Rarity, bool) {
	for i, r := range RarityTiers {
		if r == from && i < len(RarityTiers)-1 {
			return RarityTiers[i+1], true
		}
	}
	return "", false
}

// RequiredFurniture returns the furniture ID whose presence in the active
// house gates fusion of the given rarity. Epic → legendary requires the
// Arcane Forge, everything below needs the regular Forge.
func RequiredFurniture(from items.Rarity) string {
	if from == items.RarityEpic {
		return "arcane_forge"
	}
	return "forge"
}

// ResearchFor returns the research that must be completed to fuse items of the
// given rarity. Each fusion tier has its own research (fusion_common …).
func ResearchFor(from items.Rarity) string {
	switch from {
	case items.RarityCommon:
		return "fusion_common"
	case items.RarityUncommon:
		return "fusion_uncommon"
	case items.RarityRare:
		return "fusion_rare"
	case items.RarityEpic:
		return "fusion_epic"
	}
	return ""
}

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

// CanFuse reports whether the user may fuse items of the given rarity: the
// forge furniture must be placed in their active house and the tier's fusion
// research must be completed.
func (s *Service) CanFuse(userID int64, from items.Rarity) error {
	if _, ok := NextRarity(from); !ok {
		return ErrNoForge
	}
	if !furnituresvc.HasFurniture(s.store, userID, RequiredFurniture(from)) {
		if from == items.RarityEpic {
			return ErrNeedArcaneForge
		}
		return ErrNoForge
	}
	if !s.isResearchCompleted(userID, ResearchFor(from)) {
		return ErrResearchRequired
	}
	return nil
}

// isResearchCompleted reports whether the user finished the given research.
// Fusion research is started and paid for through the house research UI.
func (s *Service) isResearchCompleted(userID int64, researchID string) bool {
	if researchID == "" {
		return true
	}
	var r model.UserResearch
	err := s.store.DB.Where("user_id = ? AND research_id = ? AND completed = ?",
		userID, researchID, true).First(&r).Error
	return err == nil
}

// UnequippedCount returns how many unequipped equipment instances the user
// owns at the given rarity.
func (s *Service) UnequippedCount(userID int64, rarity items.Rarity) int {
	var count int64
	s.store.DB.Model(&model.UserEquipment{}).
		Where("user_id = ? AND is_equipped = ? AND rarity = ?", userID, false, string(rarity)).
		Count(&count)
	return int(count)
}

// Fuse consumes five unequipped items of the given rarity and creates one
// random piece of the next rarity in their place.
func (s *Service) Fuse(userID int64, from items.Rarity, equipIDs []uint) (*model.UserEquipment, error) {
	if len(equipIDs) != FuseCount {
		return nil, ErrNeedFive
	}
	if err := s.CanFuse(userID, from); err != nil {
		return nil, err
	}

	var rows []model.UserEquipment
	if err := s.store.DB.Where("id IN ?", equipIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) != FuseCount {
		return nil, ErrNeedFive
	}
	for _, r := range rows {
		if r.UserID != userID {
			return nil, ErrNotOwned
		}
		if r.IsEquipped {
			return nil, ErrEquippedItem
		}
		if string(r.Rarity) != string(from) {
			return nil, ErrWrongRarity
		}
	}

	to, _ := NextRarity(from)
	piece := generateFusedItem(to)

	var created *model.UserEquipment
	err := s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id IN ?", equipIDs).Delete(&model.UserEquipment{}).Error; err != nil {
			return err
		}
		eq, err := s.store.CreateEquipmentTx(tx, userID, piece.ID, piece.Name, piece.Emoji,
			string(to), piece.EquipSlot, piece.MinLevel,
			piece.StatSTR, piece.StatDEX, piece.StatINT, piece.StatVIT, piece.StatLUK,
			piece.Affixes, "")
		if err != nil {
			return err
		}
		created = eq
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

// Scrap destroys one unequipped equipment instance and returns the resources
// recovered from it, keyed by item ID.
func (s *Service) Scrap(userID int64, equipID uint) (map[string]int, error) {
	var target model.UserEquipment
	if err := s.store.DB.First(&target, equipID).Error; err != nil {
		return nil, ErrUnknownItem
	}
	if target.UserID != userID {
		return nil, ErrNotOwned
	}
	if target.IsEquipped {
		return nil, ErrEquippedItem
	}

	pool := scrapPools[items.Rarity(target.Rarity)]
	if len(pool) == 0 {
		pool = scrapPools[items.RarityCommon]
	}
	rewards := make(map[string]int, len(pool))
	for _, e := range pool {
		if e.Max <= e.Min {
			rewards[e.ItemID] = e.Min
		} else {
			rewards[e.ItemID] = e.Min + rand.Intn(e.Max-e.Min+1)
		}
	}

	err := s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", equipID).Delete(&model.UserEquipment{}).Error; err != nil {
			return err
		}
		for itemID, qty := range rewards {
			if err := s.store.AddItemRaw(tx, userID, itemID, qty); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rewards, nil
}

// scrapEntry is one resource type a scrap can yield, with a random quantity
// range. Scrapping is the resource sink for old equipment: every entry of the
// rarity's pool is granted, so higher rarities always recover more.
type scrapEntry struct {
	ItemID string
	Min    int
	Max    int
}

var scrapPools = map[items.Rarity][]scrapEntry{
	items.RarityCommon: {
		{ItemID: "pebble", Min: 2, Max: 3},
		{ItemID: "coal", Min: 1, Max: 2},
		{ItemID: "iron_ore", Min: 1, Max: 2},
	},
	items.RarityUncommon: {
		{ItemID: "iron_ore", Min: 2, Max: 3},
		{ItemID: "copper_ore", Min: 1, Max: 2},
		{ItemID: "silver_ore", Min: 1, Max: 2},
	},
	items.RarityRare: {
		{ItemID: "silver_ore", Min: 2, Max: 3},
		{ItemID: "gold_nugget", Min: 1, Max: 2},
		{ItemID: "emerald", Min: 1, Max: 2},
	},
	items.RarityEpic: {
		{ItemID: "gold_nugget", Min: 2, Max: 3},
		{ItemID: "emerald", Min: 1, Max: 2},
		{ItemID: "rough_diamond", Min: 1, Max: 2},
		{ItemID: "platinum", Min: 1, Max: 2},
	},
	items.RarityLegendary: {
		{ItemID: "platinum", Min: 2, Max: 3},
		{ItemID: "rough_diamond", Min: 1, Max: 2},
		{ItemID: "ancient_alloy", Min: 1, Max: 2},
		{ItemID: "kethari_crystal", Min: 1, Max: 2},
	},
}

// fusedPiece describes a procedurally generated piece of fused gear.
type fusedPiece struct {
	ID, Name, Emoji, EquipSlot                  string
	MinLevel                                    int
	StatSTR, StatDEX, StatINT, StatVIT, StatLUK int
	Affixes                                     []byte
}

// rarityMod scales base stats by the fused rarity, matching the delve loot
// curve so higher-tier fusions produce meaningfully stronger gear.
var rarityMod = map[items.Rarity]int{
	items.RarityCommon:    1,
	items.RarityUncommon:  2,
	items.RarityRare:      3,
	items.RarityEpic:      5,
	items.RarityLegendary: 10,
}

// generateFusedItem rolls a random piece of gear at exactly the target rarity:
// a random delve base (any slot), optional prefix and suffix, rolled affixes
// and a min level derived from the rarity. Set, quest and award-only items are
// never produced.
func generateFusedItem(target items.Rarity) *fusedPiece {
	var bases []items.DelveBase
	for _, b := range items.DelveBases {
		if b.MinRar.Rank() <= target.Rank() {
			bases = append(bases, b)
		}
	}
	base := bases[rand.Intn(len(bases))]

	statSTR, statDEX := base.StatSTR, base.StatDEX
	statINT, statVIT, statLUK := base.StatINT, base.StatVIT, base.StatLUK
	nameParts := []string{}
	emoji := base.Emoji

	if target.Rank() >= items.RarityUncommon.Rank() && rand.Float64() < 0.6 {
		var prefixes []items.DelvePrefix
		for _, p := range items.DelvePrefixes {
			if p.MinRar.Rank() <= target.Rank() {
				prefixes = append(prefixes, p)
			}
		}
		pref := prefixes[rand.Intn(len(prefixes))]
		nameParts = append(nameParts, pref.Name)
		statSTR += pref.StatSTR
		statDEX += pref.StatDEX
		statINT += pref.StatINT
		statVIT += pref.StatVIT
		statLUK += pref.StatLUK
		emoji = pref.Emoji
	}
	nameParts = append(nameParts, base.Name)

	if target.Rank() >= items.RarityUncommon.Rank() && rand.Float64() < 0.5 {
		var suffixes []items.DelveSuffix
		for _, sf := range items.DelveSuffixes {
			if sf.MinRar.Rank() <= target.Rank() {
				suffixes = append(suffixes, sf)
			}
		}
		suf := suffixes[rand.Intn(len(suffixes))]
		nameParts = append(nameParts, suf.Name)
	}

	name := strings.Join(nameParts, " ")
	mod := rarityMod[target]

	piece := &fusedPiece{
		ID:        "fused_" + strings.ToLower(strings.ReplaceAll(name, " ", "_")),
		Name:      name,
		Emoji:     emoji,
		EquipSlot: base.EquipSlot,
		MinLevel:  items.MinLevelForRarity(target),
		StatSTR:   statSTR * mod / 2,
		StatDEX:   statDEX * mod / 2,
		StatINT:   statINT * mod / 2,
		StatVIT:   statVIT * mod / 2,
		StatLUK:   statLUK * mod / 2,
	}

	affixes := items.RollAffixes(target, base.EquipSlot)
	applied := make([]items.AppliedAffix, 0, len(affixes))
	for _, a := range affixes {
		applied = append(applied, items.AppliedAffix{
			ID: a.ID, Name: a.Name, Stat: a.Stat, Value: items.RollAffixValue(a),
		})
	}
	piece.Affixes, _ = json.Marshal(applied)
	return piece
}
