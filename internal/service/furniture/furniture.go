package furniture

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	housingsvc "guacagamblebot/internal/service/housing"
	"guacagamblebot/internal/store"
)

// Effect is a passive bonus granted by a furniture while it is placed in the
// user's active house. Stat is the key other systems query via EffectValue.
type Effect struct {
	Stat        string
	Value       float64
	Description string
}

type FurnitureDef struct {
	ID              string
	Name            string
	Emoji           string
	Description     string
	CostMoney       int
	CostItems       map[string]int
	SlotType        string
	Slots           int
	Effects         []Effect
	UnlocksResearch []string
}

var FurnitureDefs = map[string]*FurnitureDef{
	"workbench": {
		ID: "workbench", Name: "Workbench", Emoji: "🪚",
		Description:     "A sturdy workbench for crafting tools and weapons.",
		CostMoney:       2000,
		SlotType:        "floor",
		Slots:           1,
		CostItems:       map[string]int{"pebble": 10, "iron_ore": 5},
		Effects:         []Effect{{Stat: "craft_cost", Value: 0.10, Description: "-10% crafting ingredient costs"}},
		UnlocksResearch: []string{"tool_crafting"},
	},
	"enchanting_table": {
		ID: "enchanting_table", Name: "Enchanting Table", Emoji: "🔮",
		Description:     "Mystical energies swirl around this arcane table.",
		CostMoney:       3000,
		SlotType:        "table",
		Slots:           2,
		CostItems:       map[string]int{"silver_ore": 5, "pufferfish": 3},
		Effects:         []Effect{{Stat: "pet_heal", Value: 0.10, Description: "+10% pet heal discount"}},
		UnlocksResearch: []string{"scroll_magic"},
	},
	"magnetic_coil": {
		ID: "magnetic_coil", Name: "Magnetic Coil", Emoji: "🧲",
		Description:     "A complex electromagnetic research device.",
		CostMoney:       5000,
		SlotType:        "table",
		Slots:           1,
		CostItems:       map[string]int{"copper_ore": 10, "iron_ore": 5},
		Effects:         []Effect{{Stat: "dig_luck", Value: 0.05, Description: "+5% rare dig finds"}},
		UnlocksResearch: []string{"magnetism"},
	},
	"gambling_parlor": {
		ID: "gambling_parlor", Name: "Gambling Parlor", Emoji: "🎰",
		Description:     "Everything you need for probability manipulation.",
		CostMoney:       4000,
		SlotType:        "floor",
		Slots:           2,
		CostItems:       map[string]int{"coal": 10, "gold_nugget": 5},
		Effects:         []Effect{{Stat: "casino_mega", Value: 1, Description: "Unlocks the Mega Slots machine and raises daily casino limits (10 → 15)"}},
		UnlocksResearch: []string{"game_theory"},
	},
	"greenhouse_kit": {
		ID: "greenhouse_kit", Name: "Greenhouse Kit", Emoji: "🌱",
		Description:     "Advanced horticultural equipment for serious farming.",
		CostMoney:       6000,
		SlotType:        "floor",
		Slots:           2,
		CostItems:       map[string]int{"rotten_plant": 20, "wheat": 10, "gold_nugget": 5},
		Effects:         []Effect{{Stat: "farm_yield", Value: 0.10, Description: "+10% farm harvest yield"}},
		UnlocksResearch: []string{"advanced_botany"},
	},
	"genetics_lab": {
		ID: "genetics_lab", Name: "Genetics Lab", Emoji: "🧬",
		Description:     "A cutting-edge laboratory for DNA analysis and engineering.",
		CostMoney:       10000,
		SlotType:        "floor",
		Slots:           3,
		CostItems:       map[string]int{"pure_dna": 5, "bone_dust": 20, "emerald": 2},
		Effects:         []Effect{{Stat: "pet_xp", Value: 0.10, Description: "+10% pet XP"}},
		UnlocksResearch: []string{"dna_research"},
	},
	"forge": {
		ID: "forge", Name: "Forge", Emoji: "🔨",
		Description:     "A roaring forge for smithing weapons and armor.",
		CostMoney:       3000,
		SlotType:        "floor",
		Slots:           2,
		CostItems:       map[string]int{"iron_ore": 10, "coal": 5},
		Effects:         []Effect{{Stat: "equip_quality", Value: 0.05, Description: "+5% chance to upgrade crafted rarity"}},
		UnlocksResearch: []string{"equip_common", "equip_uncommon", "equip_rare", "set_dragon_slayer", "set_shadow_stalker"},
	},
	"arcane_forge": {
		ID: "arcane_forge", Name: "Arcane Forge", Emoji: "🔮",
		Description:     "An enchanted forge infused with arcane energies for legendary crafting.",
		CostMoney:       10000,
		SlotType:        "floor",
		Slots:           3,
		CostItems:       map[string]int{"platinum": 5, "rough_diamond": 3},
		Effects:         []Effect{{Stat: "equip_legendary", Value: 0.02, Description: "+2% legendary craft chance"}},
		UnlocksResearch: []string{"equip_epic", "equip_legendary", "set_arcane_weaver"},
	},
}

// ErrNoFurnitureSlots is returned when the active house has no furniture slots
// at all (e.g. a cardboard box), so the player must upgrade houses first.
var ErrNoFurnitureSlots = errors.New("this house has no furniture slots")

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

// ActiveHouseType resolves the user's active house type, or "" when none.
func ActiveHouseType(s *store.Store, userID int64) string {
	var h model.UserHousing
	if err := s.DB.Where("user_id = ? AND is_active = ?", userID, true).First(&h).Error; err != nil {
		return ""
	}
	return h.HouseType
}

// HasFurniture reports whether the user has the given furniture placed in their
// active house. It is the gate other services use for feature-unlock furniture.
func HasFurniture(s *store.Store, userID int64, furnitureID string) bool {
	houseType := ActiveHouseType(s, userID)
	if houseType == "" {
		return false
	}
	var count int64
	s.DB.Model(&model.UserFurniture{}).
		Where("user_id = ? AND house_type = ? AND furniture_id = ?", userID, houseType, furnitureID).
		Count(&count)
	return count > 0
}

// EffectValue returns the summed value of the given effect stat granted by the
// furniture placed in the user's active house.
func EffectValue(s *store.Store, userID int64, stat string) float64 {
	houseType := ActiveHouseType(s, userID)
	if houseType == "" {
		return 0
	}
	var placed []model.UserFurniture
	if err := s.DB.Where("user_id = ? AND house_type = ?", userID, houseType).Find(&placed).Error; err != nil {
		return 0
	}
	var total float64
	for _, p := range placed {
		if fd := FurnitureDefs[p.FurnitureID]; fd != nil {
			for _, e := range fd.Effects {
				if e.Stat == stat {
					total += e.Value
				}
			}
		}
	}
	return total
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

// GetUsedSlots returns the total size of the furniture placed in the active
// house.
func (s *Service) GetUsedSlots(userID int64) int {
	houseType, err := s.activeHouseType(userID)
	if err != nil {
		return 0
	}
	var placed []model.UserFurniture
	if err := s.store.DB.Where("user_id = ? AND house_type = ?", userID, houseType).Find(&placed).Error; err != nil {
		return 0
	}
	used := 0
	for _, p := range placed {
		if fd := FurnitureDefs[p.FurnitureID]; fd != nil {
			used += fd.Slots
		} else {
			used++
		}
	}
	return used
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
	return ht.SlotsAt(h.Level)
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
	if maxSlots == 0 {
		return ErrNoFurnitureSlots
	}
	if used+fd.Slots > maxSlots {
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
