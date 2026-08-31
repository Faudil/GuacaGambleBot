package housing

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"gorm.io/gorm"

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
	MaxSanctuaryTier int
}

// MaxFurnitureSlots caps the total furniture a house can hold.
const MaxFurnitureSlots = 10

// SlotsAt returns how many furniture slots the house offers at a given level:
// one extra slot per level above 1, never exceeding the cap.
func (ht *HouseType) SlotsAt(level int) int {
	if level < 1 {
		level = 1
	}
	total := ht.FurnitureSlots + (level - 1)
	if total > MaxFurnitureSlots {
		return MaxFurnitureSlots
	}
	return total
}

var Houses = map[string]*HouseType{
	"cardboard_box": {
		ID: "cardboard_box", Price: 50, MaxLevel: 1, IncomePerHour: 1,
		InventoryBonus: 100, PetSlotsBonus: 0, BankCapacity: 600, FurnitureSlots: 0, Color: 0xB9936C,
		Buffs:            []string{"+100 Inventory Slots", "$600 Bank Cap"},
		MaxSanctuaryTier: 1,
	},
	"wooden_shack": {
		ID: "wooden_shack", Price: 500, MaxLevel: 3, IncomePerHour: 5,
		InventoryBonus: 250, PetSlotsBonus: 1, BankCapacity: 1000, CraftingDiscount: 0.05, FurnitureSlots: 2, Color: 0xA1887F,
		Buffs:            []string{"+250 Inventory Slots", "+1 Pet Slot", "$1,000 Bank Cap", "5% Crafting Discount"},
		MaxSanctuaryTier: 2,
	},
	"brick_house": {
		ID: "brick_house", Price: 5000, MaxLevel: 5, IncomePerHour: 10,
		InventoryBonus: 500, PetSlotsBonus: 3, BankCapacity: 5000, CraftingDiscount: 0.10, FurnitureSlots: 4, Color: 0xD32F2F,
		Buffs:            []string{"+500 Inventory Slots", "+3 Pet Slots", "$5,000 Bank Cap", "10% Crafting Discount"},
		MaxSanctuaryTier: 3,
	},
	"mansion": {
		ID: "mansion", Price: 50000, MaxLevel: 10, IncomePerHour: 25,
		InventoryBonus: 1000, PetSlotsBonus: 5, BankCapacity: 10000, CraftingDiscount: 0.20, FurnitureSlots: 6, Color: 0x1E88E5,
		Buffs:            []string{"+1000 Inventory Slots", "+5 Pet Slots", "$10,000 Bank Cap", "20% Crafting Discount"},
		MaxSanctuaryTier: 8,
	},
	"gilded_palace": {
		ID: "gilded_palace", Price: 1000000, MaxLevel: 20, IncomePerHour: 50,
		InventoryBonus: 2000, PetSlotsBonus: 7, BankCapacity: 100000, CraftingDiscount: 0.30, FurnitureSlots: 7, Color: 0xFFB300,
		Buffs:            []string{"+2000 Inventory Slots", "+7 Pet Slots", "$100,000 Bank Cap", "30% Crafting Discount"},
		MaxSanctuaryTier: 10,
	},
}

var BaseProduction = map[string]map[string]float64{
	"cardboard_box": {"wheat": 0.1},
	"wooden_shack":  {"wheat": 0.2, "oat": 0.2},
	"brick_house":   {"iron_ore": 0.1, "coal": 0.2},
	"mansion":       {"silver_ore": 0.1, "gold_nugget": 0.2},
	"gilded_palace": {"platinum": 0.1, "emerald": 0.01},
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

var (
	ErrInvalidHouseType = errors.New("invalid house type")
	ErrNotEnoughMoney   = errors.New("not enough money")
	ErrAlreadyOwned     = errors.New("already owned")
)

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

// GetHousing returns the user's currently active house.
func (s *Service) GetHousing(userID int64) (*model.UserHousing, error) {
	var h model.UserHousing
	err := s.store.DB.Where("user_id = ? AND is_active = ?", userID, true).First(&h).Error
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// BankCapacity returns the maximum amount the user can hold in the bank:
// the active house's BankCapacity (default 500 when the user owns no house),
// boosted by the merchant upgrades (merchant_office +20%, merchant_vault x2).
func (s *Service) BankCapacity(userID int64) (int, error) {
	h, err := s.GetHousing(userID)
	if err != nil {
		return 500, nil
	}
	ht := Houses[h.HouseType]
	if ht == nil {
		return 500, nil
	}
	cap := ht.BankCapacity
	var upgrades []model.UserHousingUpgrade
	if err := s.store.DB.Where("user_id = ?", userID).Find(&upgrades).Error; err != nil {
		return 0, err
	}
	upgMap := map[string]bool{}
	for _, u := range upgrades {
		upgMap[u.UpgradeID] = true
	}
	if upgMap["merchant_office"] {
		cap = int(math.Floor(float64(cap) * 1.2))
	}
	if upgMap["merchant_vault"] {
		cap *= 2
	}
	return cap, nil
}

// ListHouses returns every house the user owns.
func (s *Service) ListHouses(userID int64) ([]model.UserHousing, error) {
	var houses []model.UserHousing
	err := s.store.DB.Where("user_id = ?", userID).Order("house_type").Find(&houses).Error
	return houses, err
}

// HasHouse reports whether the user already owns the given house type.
func (s *Service) HasHouse(userID int64, houseType string) bool {
	var count int64
	s.store.DB.Model(&model.UserHousing{}).Where("user_id = ? AND house_type = ?", userID, houseType).Count(&count)
	return count > 0
}

func (s *Service) recalcHousingBonuses(userID int64) {
	var houses []model.UserHousing
	if err := s.store.DB.Where("user_id = ?", userID).Find(&houses).Error; err != nil {
		return
	}
	totalInv, totalPet := 0, 0
	for _, h := range houses {
		if ht := Houses[h.HouseType]; ht != nil {
			totalInv += ht.InventoryBonus
			totalPet += ht.PetSlotsBonus
		}
	}
	s.store.DB.Model(&model.User{}).Where("user_id = ?", userID).
		Updates(map[string]any{
			"extra_inv_slots": totalInv,
			"extra_pet_slots": totalPet,
		})
}

func (s *Service) applyHousingBonuses(userID int64, ht *HouseType) {
	s.recalcHousingBonuses(userID)
}

// BuyHouse purchases a new house type and moves in immediately. Each house
// type can only be owned once.
func (s *Service) BuyHouse(userID int64, houseType string) error {
	ht, ok := Houses[houseType]
	if !ok {
		return ErrInvalidHouseType
	}
	if s.HasHouse(userID, houseType) {
		return ErrAlreadyOwned
	}
	now := time.Now()
	err := s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.store.UpdateBalanceTx(tx, userID, 0); err != nil {
			return err
		}
		var bal int
		if err := tx.Model(&model.User{}).Where("user_id = ?", userID).Pluck("balance", &bal).Error; err != nil {
			return err
		}
		if bal < ht.Price {
			return ErrNotEnoughMoney
		}
		if err := s.store.UpdateBalanceTx(tx, userID, -ht.Price); err != nil {
			return err
		}
		if err := tx.Model(&model.UserHousing{}).Where("user_id = ?", userID).
			Update("is_active", false).Error; err != nil {
			return err
		}
		return tx.Create(&model.UserHousing{
			UserID: userID, HouseType: houseType, Level: 1, LastCollected: &now, IsActive: true,
			StoredItems: "{}",
		}).Error
	})
	if err != nil {
		return err
	}
	s.recalcHousingBonuses(userID)
	return nil
}

// SwitchHouse makes another owned house the active one.
func (s *Service) SwitchHouse(userID int64, houseType string) error {
	if _, ok := Houses[houseType]; !ok {
		return ErrInvalidHouseType
	}
	if !s.HasHouse(userID, houseType) {
		return ErrAlreadyOwned
	}
	err := s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.UserHousing{}).Where("user_id = ?", userID).
			Update("is_active", false).Error; err != nil {
			return err
		}
		return tx.Model(&model.UserHousing{}).Where("user_id = ? AND house_type = ?", userID, houseType).
			Update("is_active", true).Error
	})
	if err != nil {
		return err
	}
	s.recalcHousingBonuses(userID)
	return nil
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
	if err := s.store.DB.Model(&model.UserHousing{}).Where("user_id = ? AND is_active = ?", userID, true).
		UpdateColumn("level", gorm.Expr("level + 1")).Error; err != nil {
		return err
	}
	s.recalcHousingBonuses(userID)
	return nil
}

func (s *Service) Rename(userID int64, name string) error {
	return s.store.DB.Model(&model.UserHousing{}).Where("user_id = ? AND is_active = ?", userID, true).
		Update("custom_name", name).Error
}

func (s *Service) SetColor(userID int64, hex string) error {
	return s.store.DB.Model(&model.UserHousing{}).Where("user_id = ? AND is_active = ?", userID, true).
		Update("custom_color", hex).Error
}

// CollectResult is what a house has accrued since its last collection. Items
// maps item ID to quantity; callers localize the IDs for display.
type CollectResult struct {
	Income int
	Items  map[string]int
}

// pendingCollect computes the income and resources accrued since the house was
// last collected, without banking either. Shared by Collect and GetCollectInfo
// so the preview always matches what collecting actually grants.
func (s *Service) pendingCollect(userID int64) (*CollectResult, error) {
	h, err := s.GetHousing(userID)
	if err != nil {
		return nil, err
	}
	ht := Houses[h.HouseType]
	if ht == nil {
		return nil, fmt.Errorf("unknown house")
	}
	hours := time.Since(*h.LastCollected).Hours()

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

	prodMult := 1.0
	if upgMap["industrial_workshop"] {
		prodMult = 2.0
	}
	produced := map[string]int{}
	for itemID, rate := range BaseProduction[h.HouseType] {
		if qty := int(math.Floor(hours * rate * prodMult)); qty > 0 {
			produced[itemID] = qty
		}
	}
	return &CollectResult{Income: income, Items: produced}, nil
}

func (s *Service) Collect(userID int64) (*CollectResult, error) {
	res, err := s.pendingCollect(userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	tx := s.store.DB.Begin()
	if res.Income > 0 {
		if err := tx.Model(&model.User{}).Where("user_id = ?", userID).
			UpdateColumn("balance", gorm.Expr("balance + ?", res.Income)).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	tx.Model(&model.UserHousing{}).Where("user_id = ? AND is_active = ?", userID, true).
		Update("last_collected", now)
	tx.Commit()

	return res, nil
}

func (s *Service) GetCollectInfo(userID int64) (*CollectResult, error) {
	return s.pendingCollect(userID)
}

// HasUpgrade reports whether the user already owns the given house upgrade.
func (s *Service) HasUpgrade(userID int64, upgradeID string) (bool, error) {
	var count int64
	err := s.store.DB.Model(&model.UserHousingUpgrade{}).
		Where("user_id = ? AND upgrade_id = ?", userID, upgradeID).Count(&count).Error
	return count > 0, err
}

// StartConstruction begins building a house upgrade: the player pays the
// money and item cost immediately and the upgrade finishes after TimeHours.
// Only one construction can run at a time.
func (s *Service) StartConstruction(userID int64, upgradeID string) error {
	upg, ok := UpgradesTree[upgradeID]
	if !ok {
		return fmt.Errorf("unknown upgrade")
	}
	h, err := s.GetHousing(userID)
	if err != nil {
		return err
	}
	if h.UnderConstruction != nil && *h.UnderConstruction != "" {
		return fmt.Errorf("construction already in progress")
	}
	owned, err := s.HasUpgrade(userID, upgradeID)
	if err != nil {
		return err
	}
	if owned {
		return fmt.Errorf("already owned")
	}
	if upg.Requires != "" {
		hasReq, err := s.HasUpgrade(userID, upg.Requires)
		if err != nil {
			return err
		}
		if !hasReq {
			return fmt.Errorf("requires %s", upg.Requires)
		}
	}
	bal, err := s.store.GetBalance(userID)
	if err != nil {
		return err
	}
	if bal < upg.CostMoney {
		return fmt.Errorf("not enough money")
	}
	finish := time.Now().Add(time.Duration(upg.TimeHours) * time.Hour)

	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		// Validate all items are available first, in deterministic order.
		itemIDs := make([]string, 0, len(upg.CostItems))
		for itemID := range upg.CostItems {
			itemIDs = append(itemIDs, itemID)
		}
		sort.Strings(itemIDs)
		for _, itemID := range itemIDs {
			var inv model.Inventory
			if err := tx.Where("user_id = ? AND item_id = ? AND quantity >= ?", userID, itemID, upg.CostItems[itemID]).First(&inv).Error; err != nil {
				return fmt.Errorf("missing %s x%d", itemID, upg.CostItems[itemID])
			}
		}
		if err := tx.Model(&model.User{}).Where("user_id = ?", userID).
			UpdateColumn("balance", gorm.Expr("balance - ?", upg.CostMoney)).Error; err != nil {
			return err
		}
		for _, itemID := range itemIDs {
			if err := tx.Model(&model.Inventory{}).
				Where("user_id = ? AND item_id = ?", userID, itemID).
				UpdateColumn("quantity", gorm.Expr("quantity - ?", upg.CostItems[itemID])).Error; err != nil {
				return err
			}
		}
		return tx.Model(&model.UserHousing{}).Where("user_id = ? AND is_active = ?", userID, true).
			Updates(map[string]any{"under_construction": upgradeID, "finish_time": finish}).Error
	})
}

// CompleteConstruction finalizes a finished house construction and grants the
// upgrade to the user.
func (s *Service) CompleteConstruction(userID int64) error {
	var h model.UserHousing
	if err := s.store.DB.Where("user_id = ? AND is_active = ?", userID, true).First(&h).Error; err != nil {
		return err
	}
	if h.UnderConstruction == nil || *h.UnderConstruction == "" {
		return fmt.Errorf("no construction")
	}
	if h.FinishTime == nil || time.Now().Before(*h.FinishTime) {
		return fmt.Errorf("construction not finished")
	}
	owned, err := s.HasUpgrade(userID, *h.UnderConstruction)
	if err != nil {
		return err
	}
	if owned {
		return fmt.Errorf("already owned")
	}
	upg := model.UserHousingUpgrade{UserID: h.UserID, UpgradeID: *h.UnderConstruction}
	if err := s.store.DB.Create(&upg).Error; err != nil {
		return err
	}
	return s.store.DB.Model(&model.UserHousing{}).Where("user_id = ? AND is_active = ?", userID, true).
		Updates(map[string]any{"under_construction": nil, "finish_time": nil}).Error
}

func (s *Service) GetStoredItems(userID int64) (map[string]int, error) {
	var h model.UserHousing
	if err := s.store.DB.Where("user_id = ? AND is_active = ?", userID, true).First(&h).Error; err != nil {
		return nil, err
	}
	var items map[string]int
	if err := json.Unmarshal([]byte(h.StoredItems), &items); err != nil {
		return map[string]int{}, nil
	}
	return items, nil
}
