package store

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/model"
)

// SaveMiningSession upserts an in-progress mining expedition so a bot restart
// cannot lose the player's depth, effects or loot.
func (s *Store) SaveMiningSession(session *model.MiningSession) error {
	return s.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"depth", "tool_id", "ghost_veil_turns", "risk_mod", "risk_turns",
			"bag", "contract", "updated_at",
		}),
	}).Create(session).Error
}

// GetMiningSession returns the persisted expedition for userID, or nil when
// there is none.
func (s *Store) GetMiningSession(userID int64) (*model.MiningSession, error) {
	var session model.MiningSession
	err := s.DB.Where("user_id = ?", userID).First(&session).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &session, err
}

// DeleteMiningSession removes the persisted expedition for userID.
func (s *Store) DeleteMiningSession(userID int64) error {
	return s.DB.Where("user_id = ?", userID).Delete(&model.MiningSession{}).Error
}
