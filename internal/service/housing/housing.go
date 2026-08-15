package housing

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type HouseType struct {
	ID               string
	Price            int
	MaxLevel         int
	IncomePerHour    int
	InventoryBonus   int
	PetSlotsBonus    int
	BankCapacity     int
	CraftingDiscount float64
	FurnitureSlots   int
	Color            int
	Buffs            []string
}

var Houses = map[string]*HouseType{
	"cardboard_box": {
		ID: "cardboard_box", Price: 500, MaxLevel: 1, IncomePerHour: 5,
		InventoryBonus: 5, PetSlotsBonus: 1, BankCapacity: 500, FurnitureSlots: 1, Color: 0xB9936C,
		Buffs: []string{"+5 Inventory Slots", "+1 Pet Slot", "$500 Bank Cap"},
	},
	"wooden_shack": {
		ID: "wooden_shack", Price: 5000, MaxLevel: 3, IncomePerHour: 50,
		InventoryBonus: 20, PetSlotsBonus: 2, BankCapacity: 10000, CraftingDiscount: 0.05, FurnitureSlots: 2, Color: 0xA1887F,
		Buffs: []string{"+20 Inventory Slots", "+2 Pet Slots", "$10,000 Bank Cap", "5% Crafting Discount"},
	},
	"brick_house": {
		ID: "brick_house", Price: 25000, MaxLevel: 5, IncomePerHour: 250,
		InventoryBonus: 50, PetSlotsBonus: 5, BankCapacity: 2000, CraftingDiscount: 0.10, FurnitureSlots: 4, Color: 0xD32F2F,
		Buffs: []string{"+50 Inventory Slots", "+5 Pet Slots", "$2,000 Bank Cap", "10% Crafting Discount"},
	},
	"mansion": {
		ID: "mansion", Price: 100000, MaxLevel: 10, IncomePerHour: 1000,
		InventoryBonus: 100, PetSlotsBonus: 10, BankCapacity: 5000, CraftingDiscount: 0.20, FurnitureSlots: 6, Color: 0x1E88E5,
		Buffs: []string{"+100 Inventory Slots", "+10 Pet Slots", "$250,000 Bank Cap", "20% Crafting Discount"},
	},
	"gilded_palace": {
		ID: "gilded_palace", Price: 500000, MaxLevel: 20, IncomePerHour: 5000,
		InventoryBonus: 250, PetSlotsBonus: 25, BankCapacity: 1000000, CraftingDiscount: 0.30, FurnitureSlots: 10, Color: 0xFFB300,
		Buffs: []string{"+250 Inventory Slots", "+25 Pet Slots", "$1,000,000 Bank Cap", "30% Crafting Discount"},
	},
}

var BaseProduction = map[string]map[string]float64{
	"cardboard_box": {"wheat": 0.1},
	"wooden_shack":  {"wheat": 0.5, "oat": 0.2},
	"brick_house":   {"iron_ore": 0.5, "coal": 1.0},
	"mansion":       {"silver_ore": 0.5, "gold_nugget": 0.2},
	"gilded_palace": {"platinum": 0.2, "emerald": 0.1},
}

type UpgradeDef struct {
	ID        string
	Name      string
	Branch    string
	CostMoney int
	CostItems map[string]int
	TimeHours int
	Requires  string
	BonusDesc string
}

var UpgradesTree = map[string]*UpgradeDef{
	"merchant_office":     {ID: "merchant_office", Name: "Bureau de Négociant", Branch: "merchant", CostMoney: 5000, CostItems: map[string]int{"coal": 20, "copper_ore": 10}, TimeHours: 4, BonusDesc: "+20% Capacité Banque, +15% Revenus"},
	"merchant_vault":      {ID: "merchant_vault", Name: "Chambre Forte", Branch: "merchant", CostMoney: 25000, CostItems: map[string]int{"gold_nugget": 5, "silver_ore": 20}, TimeHours: 24, Requires: "merchant_office", BonusDesc: "Capacité Banque doublée"},
	"industrial_workshop": {ID: "industrial_workshop", Name: "Atelier Industriel", Branch: "industrial", CostMoney: 4000, CostItems: map[string]int{"pebble": 100, "iron_ore": 20}, TimeHours: 6, BonusDesc: "Production de ressources x2"},
	"industrial_drill":    {ID: "industrial_drill", Name: "Foreuse Automatique", Branch: "industrial", CostMoney: 30000, CostItems: map[string]int{"platinum": 2, "iron_ore": 100}, TimeHours: 48, Requires: "industrial_workshop", BonusDesc: "Donne parfois des minerais rares"},
	"mystic_altar":        {ID: "mystic_altar", Name: "Autel Mystique", Branch: "mystic", CostMoney: 7500, CostItems: map[string]int{"pufferfish": 5, "rotten_plant": 20}, TimeHours: 12, BonusDesc: "-5% Coûts de Craft, Régénération Pet"},
	"mystic_laboratory":   {ID: "mystic_laboratory", Name: "Laboratoire d'Alchimie", Branch: "mystic", CostMoney: 50000, CostItems: map[string]int{"emerald": 5, "forget_potion": 1}, TimeHours: 72, Requires: "mystic_altar", BonusDesc: "-15% Coûts de Craft, Chance XP Pet augmentée"},
}

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) GetHousing(userID int64) (*model.UserHousing, error) {
	var h model.UserHousing
	err := s.store.DB.Where("user_id = ?", userID).First(&h).Error
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func (s *Service) BuyHouse(userID int64, houseType string) error {
	ht, ok := Houses[houseType]
	if !ok {
		return fmt.Errorf("invalid house type")
	}
	bal, err := s.store.GetBalance(userID)
	if err != nil {
		return err
	}
	if bal < ht.Price {
		return fmt.Errorf("not enough money")
	}
	if _, err := s.store.UpdateBalance(userID, -ht.Price); err != nil {
		return err
	}
	now := time.Now()
	return s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{"house_type": houseType, "level": 1, "last_collected": now}),
	}).Create(&model.UserHousing{
		UserID: userID, HouseType: houseType, Level: 1, LastCollected: &now,
		StoredItems: "{}",
	}).Error
}

func (s *Service) UpgradeLevel(userID int64) error {
	h, err := s.GetHousing(userID)
	if err != nil {
		return err
	}
	ht := Houses[h.HouseType]
	if ht == nil {
		return fmt.Errorf("unknown house")
	}
	if h.Level >= ht.MaxLevel {
		return fmt.Errorf("max level")
	}
	cost := int(math.Floor(float64(ht.Price) * 0.5 * math.Pow(1.5, float64(h.Level-1))))
	bal, err := s.store.GetBalance(userID)
	if err != nil {
		return err
	}
	if bal < cost {
		return fmt.Errorf("not enough money")
	}
	if _, err := s.store.UpdateBalance(userID, -cost); err != nil {
		return err
	}
	return s.store.DB.Model(&model.UserHousing{}).Where("user_id = ?", userID).
		UpdateColumn("level", gorm.Expr("level + 1")).Error
}

func (s *Service) Rename(userID int64, name string) error {
	return s.store.DB.Model(&model.UserHousing{}).Where("user_id = ?", userID).
		Update("custom_name", name).Error
}

func (s *Service) SetColor(userID int64, hex string) error {
	return s.store.DB.Model(&model.UserHousing{}).Where("user_id = ?", userID).
		Update("custom_color", hex).Error
}

func (s *Service) Collect(userID int64) (int, []string, error) {
	h, err := s.GetHousing(userID)
	if err != nil {
		return 0, nil, err
	}
	ht := Houses[h.HouseType]
	if ht == nil {
		return 0, nil, fmt.Errorf("unknown house")
	}
	now := time.Now()
	elapsed := now.Sub(*h.LastCollected)
	hours := elapsed.Hours()

	baseIncome := float64(ht.IncomePerHour) * (1 + float64(h.Level-1)*0.1)
	income := int(math.Floor(hours * baseIncome))

	var upgrades []model.UserHousingUpgrade
	s.store.DB.Where("user_id = ?", userID).Find(&upgrades)
	upgMap := map[string]bool{}
	for _, u := range upgrades {
		upgMap[u.UpgradeID] = true
	}
	if upgMap["merchant_office"] {
		income = int(math.Floor(float64(income) * 1.15))
	}

	prod := BaseProduction[h.HouseType]
	prodMult := 1.0
	if upgMap["industrial_workshop"] {
		prodMult = 2.0
	}
	var items []string
	for itemID, rate := range prod {
		qty := int(math.Floor(hours * rate * prodMult))
		if qty > 0 {
			items = append(items, fmt.Sprintf("%s:%d", itemID, qty))
		}
	}

	tx := s.store.DB.Begin()
	if income > 0 {
		if err := tx.Model(&model.User{}).Where("user_id = ?", userID).
			UpdateColumn("balance", gorm.Expr("balance + ?", income)).Error; err != nil {
			tx.Rollback()
			return 0, nil, err
		}
	}
	tx.Model(&model.UserHousing{}).Where("user_id = ?", userID).
		Update("last_collected", now)
	tx.Commit()

	return income, items, nil
}

type CollectResult struct {
	Income int
	Items  []string
}

func (s *Service) GetCollectInfo(userID int64) (*CollectResult, error) {
	h, err := s.GetHousing(userID)
	if err != nil {
		return nil, err
	}
	ht := Houses[h.HouseType]
	if ht == nil {
		return nil, fmt.Errorf("unknown house")
	}
	now := time.Now()
	elapsed := now.Sub(*h.LastCollected)
	hours := elapsed.Hours()

	baseIncome := float64(ht.IncomePerHour) * (1 + float64(h.Level-1)*0.1)
	income := int(math.Floor(hours * baseIncome))

	var upgrades []model.UserHousingUpgrade
	s.store.DB.Where("user_id = ?", userID).Find(&upgrades)
	upgMap := map[string]bool{}
	for _, u := range upgrades {
		upgMap[u.UpgradeID] = true
	}
	if upgMap["merchant_office"] {
		income = int(math.Floor(float64(income) * 1.15))
	}

	prod := BaseProduction[h.HouseType]
	prodMult := 1.0
	if upgMap["industrial_workshop"] {
		prodMult = 2.0
	}
	var items []string
	for itemID, rate := range prod {
		qty := int(math.Floor(hours * rate * prodMult))
		if qty > 0 {
			items = append(items, fmt.Sprintf("%s:%d", itemID, qty))
		}
	}
	return &CollectResult{Income: income, Items: items}, nil
}

func (s *Service) StartConstruction(userID, upgradeID string, timeHours int) error {
	finish := time.Now().Add(time.Duration(timeHours) * time.Hour)
	return s.store.DB.Model(&model.UserHousing{}).Where("user_id = ?", userID).
		Updates(map[string]any{"under_construction": upgradeID, "finish_time": finish}).Error
}

func (s *Service) CompleteConstruction(userID string) error {
	var upg model.UserHousingUpgrade
	var h model.UserHousing
	s.store.DB.Where("user_id = ?", userID).First(&h)
	if h.UnderConstruction == nil {
		return fmt.Errorf("no construction")
	}
	upg = model.UserHousingUpgrade{UserID: h.UserID, UpgradeID: *h.UnderConstruction}
	if err := s.store.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&upg).Error; err != nil {
		return err
	}
	return s.store.DB.Model(&model.UserHousing{}).Where("user_id = ?", userID).
		Updates(map[string]any{"under_construction": nil, "finish_time": nil}).Error
}

func (s *Service) GetStoredItems(userID int64) (map[string]int, error) {
	var h model.UserHousing
	if err := s.store.DB.Where("user_id = ?", userID).First(&h).Error; err != nil {
		return nil, err
	}
	var items map[string]int
	if err := json.Unmarshal([]byte(h.StoredItems), &items); err != nil {
		return map[string]int{}, nil
	}
	return items, nil
}
