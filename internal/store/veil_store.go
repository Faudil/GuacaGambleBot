package store

import (
	"encoding/json"
	"time"

	"gorm.io/gorm/clause"

	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
)

func (s *Store) HasItem(userID int64, itemID string, qty int) (bool, error) {
	canonical := items.Canonical(itemID)
	if canonical == "" {
		return false, nil
	}
	var inv model.Inventory
	err := s.DB.Where("user_id = ? AND item_id = ?", userID, canonical).First(&inv).Error
	if err != nil {
		return false, nil
	}
	return inv.Quantity >= qty, nil
}

func (s *Store) CreateVeilRaid(raid *model.VeilRaid) error {
	return s.DB.Create(raid).Error
}

func (s *Store) GetVeilRaid(raidID int64) (*model.VeilRaid, error) {
	var raid model.VeilRaid
	err := s.DB.Where("id = ?", raidID).First(&raid).Error
	return &raid, err
}

func (s *Store) GetActiveVeilRaidByUser(userID int64) (*model.VeilRaid, error) {
	var raids []model.VeilRaid
	err := s.DB.Where("status NOT IN ('completed','failed')").Find(&raids).Error
	if err != nil {
		return nil, err
	}
	for _, r := range raids {
		var ids []int64
		UnmarshalJSON(r.ParticipantIDs, &ids)
		for _, pid := range ids {
			if pid == userID {
				return &r, nil
			}
		}
	}
	return nil, nil
}

func (s *Store) GetFormingVeilRaids(guildID int64) ([]model.VeilRaid, error) {
	var raids []model.VeilRaid
	err := s.DB.Where("guild_id = ? AND status = 'forming'", guildID).Order("created_at desc").Find(&raids).Error
	return raids, err
}

func (s *Store) SaveVeilRaid(raid *model.VeilRaid) error {
	raid.UpdatedAt = time.Now()
	return s.DB.Save(raid).Error
}

func (s *Store) DeleteVeilRaid(raidID int64) error {
	return s.DB.Where("id = ?", raidID).Delete(&model.VeilRaid{}).Error
}

func (s *Store) GetVeilRaidLockout(userID int64, weekStart string) (*model.VeilRaidLockout, error) {
	var lockout model.VeilRaidLockout
	err := s.DB.Where("user_id = ? AND week_start = ?", userID, weekStart).First(&lockout).Error
	return &lockout, err
}

func (s *Store) UpsertVeilRaidLockout(lockout *model.VeilRaidLockout) error {
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "week_start"}},
		DoUpdates: clause.AssignmentColumns([]string{"cleared", "helped_at"}),
	}).Create(lockout).Error
}

func (s *Store) SaveVeilRaidHallOfFame(entry *model.VeilRaidHallOfFame) error {
	return s.DB.Create(entry).Error
}

func (s *Store) GetVeilRaidHallOfFame(guildID int64, weekStart string) ([]model.VeilRaidHallOfFame, error) {
	var entries []model.VeilRaidHallOfFame
	err := s.DB.Where("guild_id = ? AND week_start = ?", guildID, weekStart).
		Order("clear_time asc").Find(&entries).Error
	return entries, err
}

func UnmarshalJSON(data string, target any) {
	if data == "" {
		return
	}
	_ = json.Unmarshal([]byte(data), target)
}
