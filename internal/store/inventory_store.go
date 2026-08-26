package store

import (
	"errors"

	"gorm.io/gorm"

	"guacagamblebot/internal/model"
)

// ErrInventoryFull is returned when granting an item would exceed the user's
// inventory capacity.
var ErrInventoryFull = errors.New("inventory full")

// BaseInventoryLimit is the inventory capacity every player starts with.
const BaseInventoryLimit = 300

// houseInventoryBonus mirrors housing.Houses InventoryBonus to compute
// inventory capacity as the sum of all owned houses without importing
// the housing service (avoiding an import cycle).
var houseInventoryBonus = map[string]int{
	"cardboard_box": 100,
	"wooden_shack":  250,
	"brick_house":   500,
	"mansion":       1000,
	"gilded_palace": 2000,
}

// InventoryUsed returns how many slots the user currently occupies: one slot
// per quantity unit of stackable items plus one per equipment instance.
func (s *Store) InventoryUsed(db *gorm.DB, userID int64) (int, error) {
	var used int64
	if err := db.Table("inventory").
		Where("user_id = ? AND quantity > 0", userID).
		Select("COALESCE(SUM(quantity), 0)").
		Scan(&used).Error; err != nil {
		return 0, err
	}
	var equip int64
	if err := db.Model(&model.UserEquipment{}).
		Where("user_id = ?", userID).
		Count(&equip).Error; err != nil {
		return 0, err
	}
	return int(used + equip), nil
}

// InventoryLimit returns the user's inventory capacity: the base limit plus
// the sum of inventory bonuses from all houses the user owns. When the user
// owns no house it falls back to the cached extra_inv_slots (or base limit
// for brand-new players).
func (s *Store) InventoryLimit(db *gorm.DB, userID int64) (int, error) {
	var houses []model.UserHousing
	if err := db.Where("user_id = ?", userID).Find(&houses).Error; err == nil && len(houses) > 0 {
		total := 0
		for _, h := range houses {
			if b, ok := houseInventoryBonus[h.HouseType]; ok {
				total += b
			}
		}
		return BaseInventoryLimit + total, nil
	}
	var u model.User
	if err := db.Where("user_id = ?", userID).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return BaseInventoryLimit, nil
		}
		return 0, err
	}
	return BaseInventoryLimit + u.ExtraInvSlots, nil
}

// FreeSlots returns how many inventory slots remain before the user hits the
// capacity limit.
func (s *Store) FreeSlots(db *gorm.DB, userID int64) (int, error) {
	used, err := s.InventoryUsed(db, userID)
	if err != nil {
		return 0, err
	}
	limit, err := s.InventoryLimit(db, userID)
	if err != nil {
		return 0, err
	}
	return limit - used, nil
}
