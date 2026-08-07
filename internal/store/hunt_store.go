package store

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/model"
)

// HasUnlockedZone reports whether a user has unlocked the given hunt zone.
func (s *Store) HasUnlockedZone(userID int64, zoneKey string) (bool, error) {
	var count int64
	err := s.DB.Model(&model.UserHuntUnlock{}).
		Where("user_id = ? AND zone_key = ?", userID, zoneKey).
		Count(&count).Error
	return count > 0, err
}

// UnlockZone grants permanent access to a hunt zone for a user.
func (s *Store) UnlockZone(userID int64, zoneKey string) error {
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "zone_key"}},
		DoNothing: true,
	}).Create(&model.UserHuntUnlock{UserID: userID, ZoneKey: zoneKey}).Error
}

// IncrementZoneWins records one more hunt win in a zone and returns the new
// all-time win count.
func (s *Store) IncrementZoneWins(userID int64, zoneKey string) (int, error) {
	return s.incrementZoneStat(userID, zoneKey, "wins")
}

// IncrementZoneBossKills records one more zone boss kill and returns the new
// all-time boss kill count.
func (s *Store) IncrementZoneBossKills(userID int64, zoneKey string) (int, error) {
	return s.incrementZoneStat(userID, zoneKey, "boss_kills")
}

func (s *Store) incrementZoneStat(userID int64, zoneKey, column string) (int, error) {
	wins, bossKills := 0, 0
	if column == "wins" {
		wins = 1
	} else if column == "boss_kills" {
		bossKills = 1
	}
	err := s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "zone_key"}},
		DoUpdates: clause.Assignments(map[string]any{column: gorm.Expr(column + " + 1")}),
	}).Create(&model.UserHuntZoneStat{
		UserID:    userID,
		ZoneKey:   zoneKey,
		Wins:      wins,
		BossKills: bossKills,
	}).Error
	if err != nil {
		return 0, err
	}
	var stat model.UserHuntZoneStat
	if err := s.DB.Where("user_id = ? AND zone_key = ?", userID, zoneKey).First(&stat).Error; err != nil {
		return 0, err
	}
	switch column {
	case "wins":
		return stat.Wins, nil
	case "boss_kills":
		return stat.BossKills, nil
	}
	return 0, nil
}

// GetZoneProgress returns the all-time hunt win count per zone for a user.
func (s *Store) GetZoneProgress(userID int64) (map[string]int, error) {
	var stats []model.UserHuntZoneStat
	if err := s.DB.Where("user_id = ?", userID).Find(&stats).Error; err != nil {
		return nil, err
	}
	m := make(map[string]int, len(stats))
	for _, st := range stats {
		m[st.ZoneKey] = st.Wins
	}
	return m, nil
}
