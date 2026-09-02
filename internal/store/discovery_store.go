package store

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/model"
)

// recordItemDiscovery marks an item or equipment base as discovered by the
// user, for the /glossary command. Idempotent: repeat grants of an item the
// user already found are a no-op. Called from the two chokepoints where an
// item ever enters a player's possession: AddItemRaw (stackable items) and
// CreateEquipmentTx (gear).
func (s *Store) recordItemDiscovery(db *gorm.DB, userID int64, itemID string) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
		DoNothing: true,
	}).Create(&model.UserItemDiscovery{
		UserID:       userID,
		ItemID:       itemID,
		DiscoveredAt: time.Now(),
	}).Error
}

// DiscoveredItemIDs returns the set of item/equipment-base IDs the user has
// ever obtained.
func (s *Store) DiscoveredItemIDs(userID int64) (map[string]bool, error) {
	var rows []model.UserItemDiscovery
	if err := s.DB.Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[r.ItemID] = true
	}
	return out, nil
}

// ItemDiscoveredAt returns when the user first obtained the given item, and
// whether it has been discovered at all.
func (s *Store) ItemDiscoveredAt(userID int64, itemID string) (time.Time, bool) {
	var row model.UserItemDiscovery
	if err := s.DB.Where("user_id = ? AND item_id = ?", userID, itemID).First(&row).Error; err != nil {
		return time.Time{}, false
	}
	return row.DiscoveredAt, true
}
