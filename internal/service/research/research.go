package research

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	furnituresvc "guacagamblebot/internal/service/furniture"
	"guacagamblebot/internal/store"
)

type ResearchDef struct {
	ID                string
	Name              string
	Description       string
	TimeHours         int
	CostMoney         int
	CostItems         map[string]int
	RequiredFurniture string
	UnlocksRecipes    []string
	BonusDesc         string
}

var ResearchDefs = map[string]*ResearchDef{
	"tool_crafting": {
		ID: "tool_crafting", Name: "Tool Crafting", Description: "Learn to craft basic tools and weapons.",
		TimeHours: 4, CostMoney: 1000, CostItems: map[string]int{"coal": 5},
		RequiredFurniture: "workbench",
		UnlocksRecipes:    []string{"bow", "rusty_magnet", "hook"},
		BonusDesc:         "Unlocks: Bow, Rusty Magnet, Hook",
	},
	"scroll_magic": {
		ID: "scroll_magic", Name: "Scroll Magic", Description: "Arcane knowledge to craft magical scrolls and potions.",
		TimeHours: 6, CostMoney: 2000, CostItems: map[string]int{"rotten_plant": 10, "silver_ore": 2},
		RequiredFurniture: "enchanting_table",
		UnlocksRecipes:    []string{"identity_scroll", "forget_potion"},
		BonusDesc:         "Unlocks: Identity Scroll, Forget Potion",
	},
	"magnetism": {
		ID: "magnetism", Name: "Magnetism", Description: "Electromagnetic theory for advanced magnet crafting.",
		TimeHours: 8, CostMoney: 3000, CostItems: map[string]int{"platinum": 5},
		RequiredFurniture: "magnetic_coil",
		UnlocksRecipes:    []string{"magnet", "electric_magnet"},
		BonusDesc:         "Unlocks: Magnet, Electric Magnet",
	},
	"game_theory": {
		ID: "game_theory", Name: "Game Theory", Description: "Probability manipulation and rigging techniques.",
		TimeHours: 12, CostMoney: 5000, CostItems: map[string]int{"gold_nugget": 5, "rough_diamond": 3},
		RequiredFurniture: "gambling_parlor",
		UnlocksRecipes:    []string{"rigged_coin", "casino_token", "vip_ticket"},
		BonusDesc:         "Unlocks: Rigged Coin, Casino Token, VIP Ticket",
	},
	"advanced_botany": {
		ID: "advanced_botany", Name: "Advanced Botany", Description: "Fertilizers, greenhouses, and magical orchards.",
		TimeHours: 16, CostMoney: 4000, CostItems: map[string]int{"rotten_plant": 10, "emerald": 5},
		RequiredFurniture: "greenhouse_kit",
		UnlocksRecipes:    []string{"fertilizer", "garden_plot", "tropical_greenhouse", "enchanted_orchard"},
		BonusDesc:         "Unlocks: Fertilizer, Garden Plot, Tropical Greenhouse, Enchanted Orchard",
	},
	"dna_research": {
		ID: "dna_research", Name: "DNA Research", Description: "Genetic engineering to create new life forms.",
		TimeHours: 24, CostMoney: 8000, CostItems: map[string]int{"pure_dna": 3, "golden_apple": 2, "emerald": 1},
		RequiredFurniture: "genetics_lab",
		UnlocksRecipes:    []string{"volcano_egg"},
		BonusDesc:         "Unlocks: Volcano Egg",
	},
	"pet_nutrition": {
		ID: "pet_nutrition", Name: "Pet Nutrition", Description: "Master alchemical feeding to brew potions that empower your pets.",
		TimeHours: 12, CostMoney: 6000, CostItems: map[string]int{"golden_apple": 2, "mutagen": 2},
		RequiredFurniture: "enchanting_table",
		UnlocksRecipes:    []string{"berserker_elixir", "adamant_tonic", "gale_draught", "oracles_insight"},
		BonusDesc:         "Unlocks: Pet potion crafting (Berserker Elixir, Adamant Tonic, Gale Draught, Oracle's Insight)",
	},

	// --- Equipment Rarity Research ---
	"equip_common": {
		ID: "equip_common", Name: "Common Equipment", Description: "Learn to forge basic Common equipment.",
		TimeHours: 2, CostMoney: 500, CostItems: map[string]int{"coal": 5, "pebble": 3},
		RequiredFurniture: "forge",
		UnlocksRecipes:    []string{"craft_stick", "craft_leather_armor"},
		BonusDesc:         "Unlocks: Common equipment crafting",
	},
	"equip_uncommon": {
		ID: "equip_uncommon", Name: "Uncommon Equipment", Description: "Improve your forging to craft Uncommon gear.",
		TimeHours: 4, CostMoney: 1500, CostItems: map[string]int{"iron_ore": 10},
		RequiredFurniture: "forge",
		UnlocksRecipes:    []string{"craft_iron_pickaxe", "craft_lucky_charm", "craft_fishing_rod", "craft_miner_helmet"},
		BonusDesc:         "Unlocks: Uncommon equipment crafting",
	},
	"equip_rare": {
		ID: "equip_rare", Name: "Rare Equipment", Description: "Master rare metalworking for Rare equipment.",
		TimeHours: 8, CostMoney: 4000, CostItems: map[string]int{"silver_ore": 5, "emerald": 3},
		RequiredFurniture: "forge",
		UnlocksRecipes:    []string{"craft_hunters_bow", "craft_hunter_cloak", "craft_golden_ring", "craft_crystal_staff"},
		BonusDesc:         "Unlocks: Rare equipment crafting",
	},
	"equip_epic": {
		ID: "equip_epic", Name: "Epic Equipment", Description: "Unlock the secrets of Epic gear forging.",
		TimeHours: 16, CostMoney: 8000, CostItems: map[string]int{"platinum": 5, "rough_diamond": 3},
		RequiredFurniture: "arcane_forge",
		UnlocksRecipes:    []string{"craft_enchanted_robe", "craft_ancient_amulet"},
		BonusDesc:         "Unlocks: Epic equipment crafting",
	},
	"equip_legendary": {
		ID: "equip_legendary", Name: "Legendary Equipment", Description: "Attain the pinnacle of crafting — Legendary gear.",
		TimeHours: 24, CostMoney: 15000, CostItems: map[string]int{"rough_diamond": 5, "pure_dna": 3},
		RequiredFurniture: "arcane_forge",
		UnlocksRecipes:    []string{},
		BonusDesc:         "Unlocks: Legendary equipment crafting (coming soon)",
	},

	// --- Set Research ---
	"set_dragon_slayer": {
		ID: "set_dragon_slayer", Name: "Dragon Slayer Set", Description: "Study the remains of ancient dragons to craft their slayer's gear.",
		TimeHours: 12, CostMoney: 5000, CostItems: map[string]int{"rough_diamond": 3, "gold_nugget": 5},
		RequiredFurniture: "forge",
		UnlocksRecipes:    []string{"craft_dragon_slayer_sword", "craft_dragon_slayer_armor", "craft_dragon_slayer_ring", "craft_dragon_slayer_talisman"},
		BonusDesc:         "Unlocks: Dragon Slayer set crafting",
	},
	"set_shadow_stalker": {
		ID: "set_shadow_stalker", Name: "Shadow Stalker Set", Description: "Harness shadow essence to forge the Stalker's arsenal.",
		TimeHours: 12, CostMoney: 5000, CostItems: map[string]int{"platinum": 5, "emerald": 3},
		RequiredFurniture: "forge",
		UnlocksRecipes:    []string{"craft_shadow_stalker_blade", "craft_shadow_stalker_cloak", "craft_shadow_stalker_amulet", "craft_shadow_stalker_charm"},
		BonusDesc:         "Unlocks: Shadow Stalker set crafting",
	},
	"set_arcane_weaver": {
		ID: "set_arcane_weaver", Name: "Arcane Weaver Set", Description: "Channel pure arcane energy to weave the Weaver's regalia.",
		TimeHours: 12, CostMoney: 5000, CostItems: map[string]int{"platinum": 5, "rough_diamond": 3, "emerald": 2},
		RequiredFurniture: "arcane_forge",
		UnlocksRecipes:    []string{"craft_arcane_weaver_staff", "craft_arcane_weaver_robe", "craft_arcane_weaver_crown", "craft_arcane_weaver_orb"},
		BonusDesc:         "Unlocks: Arcane Weaver set crafting",
	},
}

type Service struct {
	store *store.Store
	cfg   *config.Config
	fsvc  *furnituresvc.Service
}

func New(s *store.Store, cfg *config.Config, fsvc *furnituresvc.Service) *Service {
	return &Service{store: s, cfg: cfg, fsvc: fsvc}
}

func (s *Service) Start(userID int64, researchID string) error {
	rd, ok := ResearchDefs[researchID]
	if !ok {
		return fmt.Errorf("unknown research")
	}
	if !s.fsvc.IsPlaced(userID, rd.RequiredFurniture) {
		return fmt.Errorf("requires %s furniture", rd.RequiredFurniture)
	}
	var existing model.UserResearch
	err := s.store.DB.Where("user_id = ? AND research_id = ?", userID, researchID).First(&existing).Error
	if err == nil {
		if existing.Completed {
			return fmt.Errorf("already completed")
		}
		return fmt.Errorf("already in progress")
	}
	bal, err := s.store.GetBalance(userID)
	if err != nil {
		return err
	}
	if bal < rd.CostMoney {
		return fmt.Errorf("not enough money")
	}
	now := time.Now()
	finish := now.Add(time.Duration(rd.TimeHours) * time.Hour)

	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("user_id = ?", userID).
			UpdateColumn("balance", gorm.Expr("balance - ?", rd.CostMoney)).Error; err != nil {
			return err
		}
		for itemID, qty := range rd.CostItems {
			var inv model.Inventory
			if err := tx.Where("user_id = ? AND item_id = ? AND quantity >= ?", userID, itemID, qty).First(&inv).Error; err != nil {
				return fmt.Errorf("missing %s x%d", itemID, qty)
			}
			if err := tx.Model(&model.Inventory{}).
				Where("user_id = ? AND item_id = ?", userID, itemID).
				UpdateColumn("quantity", gorm.Expr("quantity - ?", qty)).Error; err != nil {
				return err
			}
		}
		return tx.Create(&model.UserResearch{
			UserID: userID, ResearchID: researchID,
			StartTime: &now, FinishTime: &finish, Completed: false,
		}).Error
	})
}

func (s *Service) Complete(userID int64, researchID string) error {
	var r model.UserResearch
	if err := s.store.DB.Where("user_id = ? AND research_id = ?", userID, researchID).First(&r).Error; err != nil {
		return fmt.Errorf("research not found")
	}
	if r.Completed {
		return fmt.Errorf("already completed")
	}
	if r.FinishTime != nil && time.Now().Before(*r.FinishTime) {
		return fmt.Errorf("research still in progress")
	}
	return s.store.DB.Model(&model.UserResearch{}).
		Where("user_id = ? AND research_id = ?", userID, researchID).
		Update("completed", true).Error
}

func (s *Service) IsCompleted(userID int64, researchID string) bool {
	var r model.UserResearch
	err := s.store.DB.Where("user_id = ? AND research_id = ? AND completed = ?", userID, researchID, true).First(&r).Error
	return err == nil
}

func (s *Service) GetActive(userID int64) ([]model.UserResearch, error) {
	var list []model.UserResearch
	err := s.store.DB.Where("user_id = ? AND completed = ?", userID, false).Find(&list).Error
	return list, err
}

func (s *Service) GetCompleted(userID int64) ([]model.UserResearch, error) {
	var list []model.UserResearch
	err := s.store.DB.Where("user_id = ? AND completed = ?", userID, true).Find(&list).Error
	return list, err
}
