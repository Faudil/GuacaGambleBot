package store

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
)

func (s *Store) SaveDelveSession(session *model.DelveSession) error {
	return s.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"guild_id", "channel_id", "floor", "zone", "hp", "max_hp", "mana", "max_mana",
			"torches", "keys", "gold", "inventory", "deployed_pets", "flags",
			"status_effects", "rooms_cleared", "seed", "message_id", "status",
			"auto_rescued", "auto_rescue_pet", "died_at", "updated_at",
		}),
	}).Create(session).Error
}

func (s *Store) GetDelveSession(userID int64) (*model.DelveSession, error) {
	var session model.DelveSession
	err := s.DB.Where("user_id = ?", userID).First(&session).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &session, err
}

func (s *Store) DeleteDelveSession(userID int64) error {
	return s.DB.Where("user_id = ?", userID).Delete(&model.DelveSession{}).Error
}

func (s *Store) AddDelveFlag(userID int64, flagID string, metadata string) error {
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "flag_id"}},
		DoNothing: true,
	}).Create(&model.UserDelveFlag{
		UserID:   userID,
		FlagID:   flagID,
		Metadata: metadata,
		EarnedAt: time.Now(),
	}).Error
}

func (s *Store) GetDelveFlags(userID int64) ([]model.UserDelveFlag, error) {
	var flags []model.UserDelveFlag
	err := s.DB.Where("user_id = ?", userID).Order("earned_at asc").Find(&flags).Error
	return flags, err
}

func (s *Store) HasDelveFlag(userID int64, flagID string) (bool, error) {
	var count int64
	err := s.DB.Model(&model.UserDelveFlag{}).
		Where("user_id = ? AND flag_id = ?", userID, flagID).
		Count(&count).Error
	return count > 0, err
}

func (s *Store) SaveDelveRunHistory(history *model.DelveRunHistory) error {
	return s.DB.Create(history).Error
}

func (s *Store) GetDelveRunHistory(userID int64) ([]model.DelveRunHistory, error) {
	var history []model.DelveRunHistory
	err := s.DB.Where("user_id = ?", userID).Order("run_date desc").Find(&history).Error
	return history, err
}

func (s *Store) SaveDelveGauntletScore(score *model.DelveGauntletScore) error {
	return s.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "week_start"}},
		DoUpdates: clause.Assignments(map[string]any{
			"floor": gorm.Expr("GREATEST(floor, ?)", score.Floor),
			"score": gorm.Expr("GREATEST(score, ?)", score.Score),
		}),
	}).Create(score).Error
}

func (s *Store) GetDelveGauntletLeaderboard(guildID int64, weekStart string, limit int) ([]model.DelveGauntletScore, error) {
	var scores []model.DelveGauntletScore
	err := s.DB.Where("guild_id = ? AND week_start = ?", guildID, weekStart).
		Order("score desc").Limit(limit).Find(&scores).Error
	return scores, err
}

func (s *Store) GetDelveGauntletScore(userID int64, weekStart string) (*model.DelveGauntletScore, error) {
	var score model.DelveGauntletScore
	err := s.DB.Where("user_id = ? AND week_start = ?", userID, weekStart).First(&score).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &score, err
}

func (s *Store) GetFallenPlayersOnFloor(guildID, floor int64, limit int) ([]model.DelveSession, error) {
	var sessions []model.DelveSession
	err := s.DB.Where("guild_id = ? AND floor = ? AND status = 'fallen'", guildID, floor).
		Order("RANDOM()").Limit(limit).Find(&sessions).Error
	return sessions, err
}

func (s *Store) AddItemRaw(db *gorm.DB, userID int64, itemID string, quantity int) error {
	if quantity <= 0 {
		return nil
	}
	inv := &model.Inventory{UserID: userID, ItemID: itemID, Quantity: quantity}
	// Newly granted tools start with a full durability bar. The upsert below
	// only increments quantity, so buying an extra tool never resets the
	// durability of the tool currently in use.
	if it := items.Get(itemID); it != nil && it.Durability > 0 {
		inv.Durability = it.Durability
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
		DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + ?", quantity)}),
	}).Create(inv).Error
}
