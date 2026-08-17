package furniture

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	housingsvc "guacagamblebot/internal/service/housing"
	"guacagamblebot/internal/store"
)

type FurnitureDef struct {
	ID              string
	Name            string
	Emoji           string
	Description     string
	CostMoney       int
	CostItems       map[string]int
	SlotType        string
	UnlocksResearch []string
}

var FurnitureDefs = map[string]*FurnitureDef{
	"workbench": {
		ID: "workbench", Name: "Workbench", Emoji: "🪚",
		Description:     "A sturdy workbench for crafting tools and weapons.",
		CostMoney:       2000,
		SlotType:        "floor",
		CostItems:       map[string]int{"pebble": 10, "iron_ore": 5},
		UnlocksResearch: []string{"tool_crafting"},
	},
	"enchanting_table": {
		ID: "enchanting_table", Name: "Enchanting Table", Emoji: "🔮",
		Description:     "Mystical energies swirl around this arcane table.",
		CostMoney:       3000,
		SlotType:        "table",
		CostItems:       map[string]int{"silver_ore": 5, "pufferfish": 3},
		UnlocksResearch: []string{"scroll_magic"},
	},
	"magnetic_coil": {
		ID: "magnetic_coil", Name: "Magnetic Coil", Emoji: "🧲",
		Description:     "A complex electromagnetic research device.",
		CostMoney:       5000,
		SlotType:        "table",
		CostItems:       map[string]int{"copper_ore": 10, "iron_ore": 5},
		UnlocksResearch: []string{"magnetism"},
	},
	"gambling_parlor": {
		ID: "gambling_parlor", Name: "Gambling Parlor", Emoji: "🎰",
		Description:     "Everything you need for probability manipulation.",
		CostMoney:       4000,
		SlotType:        "floor",
		CostItems:       map[string]int{"coal": 10, "gold_nugget": 5},
		UnlocksResearch: []string{"game_theory"},
	},
	"greenhouse_kit": {
		ID: "greenhouse_kit", Name: "Greenhouse Kit", Emoji: "🌱",
		Description:     "Advanced horticultural equipment for serious farming.",
		CostMoney:       6000,
		SlotType:        "floor",
		CostItems:       map[string]int{"rotten_plant": 20, "wheat": 10, "gold_nugget": 5},
		UnlocksResearch: []string{"advanced_botany"},
	},
	"genetics_lab": {
		ID: "genetics_lab", Name: "Genetics Lab", Emoji: "🧬",
		Description:     "A cutting-edge laboratory for DNA analysis and engineering.",
		CostMoney:       10000,
		SlotType:        "floor",
		CostItems:       map[string]int{"pure_dna": 5, "bone_dust": 20, "emerald": 2},
		UnlocksResearch: []string{"dna_research"},
	},
	"forge": {
		ID: "forge", Name: "Forge", Emoji: "🔨",
		Description:     "A roaring forge for smithing weapons and armor.",
		CostMoney:       3000,
		SlotType:        "floor",
		CostItems:       map[string]int{"iron_ore": 10, "coal": 5},
		UnlocksResearch: []string{"equip_common", "equip_uncommon", "equip_rare", "set_dragon_slayer", "set_shadow_stalker"},
	},
	"arcane_forge": {
		ID: "arcane_forge", Name: "Arcane Forge", Emoji: "🔮",
		Description:     "An enchanted forge infused with arcane energies for legendary crafting.",
		CostMoney:       10000,
		SlotType:        "floor",
		CostItems:       map[string]int{"platinum": 5, "rough_diamond": 3},
		UnlocksResearch: []string{"equip_epic", "equip_legendary", "set_arcane_weaver"},
	},
}

type Service struct {
	store *store.Store
	cfg   *config.Config
	hsvc  *housingsvc.Service
}

func New(s *store.Store, cfg *config.Config, hsvc *housingsvc.Service) *Service {
	return &Service{store: s, cfg: cfg, hsvc: hsvc}
}

func (s *Service) activeHouseType(userID int64) (string, error) {
	h, err := s.hsvc.GetHousing(userID)
	if err != nil {
		return "", err
	}
	return h.HouseType, nil
}

func (s *Service) GetPlaced(userID int64) ([]model.UserFurniture, error) {
	houseType, err := s.activeHouseType(userID)
	if err != nil {
		return nil, err
	}
	var placed []model.UserFurniture
	err = s.store.DB.Where("user_id = ? AND house_type = ?", userID, houseType).Find(&placed).Error
	return placed, err
}

func (s *Service) GetUsedSlots(userID int64) int {
	houseType, err := s.activeHouseType(userID)
	if err != nil {
		return 0
	}
	var count int64
	s.store.DB.Model(&model.UserFurniture{}).Where("user_id = ? AND house_type = ?", userID, houseType).Count(&count)
	return int(count)
}

func (s *Service) GetMaxSlots(userID int64) int {
	h, err := s.hsvc.GetHousing(userID)
	if err != nil {
		return 0
	}
	ht := housingsvc.Houses[h.HouseType]
	if ht == nil {
		return 0
	}
	return ht.FurnitureSlots
}

func (s *Service) IsPlaced(userID int64, furnitureID string) bool {
	houseType, err := s.activeHouseType(userID)
	if err != nil {
		return false
	}
	var count int64
	s.store.DB.Model(&model.UserFurniture{}).Where("user_id = ? AND house_type = ? AND furniture_id = ?", userID, houseType, furnitureID).Count(&count)
	return count > 0
}

func (s *Service) Place(userID int64, furnitureID string) error {
	fd, ok := FurnitureDefs[furnitureID]
	if !ok {
		return fmt.Errorf("unknown furniture")
	}
	houseType, err := s.activeHouseType(userID)
	if err != nil {
		return fmt.Errorf("you don't own a house")
	}
	used := s.GetUsedSlots(userID)
	maxSlots := s.GetMaxSlots(userID)
	if used >= maxSlots {
		return fmt.Errorf("no free slots (%d/%d)", used, maxSlots)
	}
	var existing int64
	s.store.DB.Model(&model.UserFurniture{}).Where("user_id = ? AND house_type = ? AND furniture_id = ?", userID, houseType, furnitureID).Count(&existing)
	if existing > 0 {
		return fmt.Errorf("already placed")
	}
	bal, err := s.store.GetBalance(userID)
	if err != nil {
		return err
	}
	if bal < fd.CostMoney {
		return fmt.Errorf("not enough money")
	}

	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("user_id = ?", userID).
			UpdateColumn("balance", gorm.Expr("balance - ?", fd.CostMoney)).Error; err != nil {
			return err
		}
		for itemID, qty := range fd.CostItems {
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
		return tx.Create(&model.UserFurniture{
			UserID: userID, HouseType: houseType, FurnitureID: furnitureID, PlacedAt: time.Now(),
		}).Error
	})
}

func (s *Service) Remove(userID int64, furnitureID string) error {
	houseType, err := s.activeHouseType(userID)
	if err != nil {
		return fmt.Errorf("not placed")
	}
	res := s.store.DB.Where("user_id = ? AND house_type = ? AND furniture_id = ?", userID, houseType, furnitureID).Delete(&model.UserFurniture{})
	if res.RowsAffected == 0 {
		return fmt.Errorf("not placed")
	}
	return nil
}
