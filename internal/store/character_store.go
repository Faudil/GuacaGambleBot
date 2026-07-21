package store

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/model"
)

// XPForCharacterLevel returns the XP required to reach the next level.
func XPForCharacterLevel(level int) int {
	return int(float64(level) * 100 * 1.5)
}

// EnsureCharacter creates a character row with default values if missing.
func (s *Store) EnsureCharacter(userID int64) (*model.UserCharacter, error) {
	var c model.UserCharacter
	err := s.DB.Where("user_id = ?", userID).
		Attrs(model.UserCharacter{UserID: userID, STR: 5, DEX: 5, INT: 5, VIT: 5, LUK: 5}).
		FirstOrCreate(&c).Error
	return &c, err
}

// GetCharacter returns the current character state.
func (s *Store) GetCharacter(userID int64) (*model.UserCharacter, error) {
	return s.EnsureCharacter(userID)
}

// AddCharacterXP awards XP and handles level-ups. Returns whether the player
// leveled up and the new level. Crowns are awarded on each level-up.
func (s *Store) AddCharacterXP(userID int64, amount int) (leveledUp bool, newLevel int, err error) {
	c, err := s.EnsureCharacter(userID)
	if err != nil {
		return false, 0, err
	}

	c.XP += amount
	leveled := false
	for c.XP >= XPForCharacterLevel(c.Level) && c.Level < 100 {
		c.XP -= XPForCharacterLevel(c.Level)
		c.Level++
		c.SkillPoints += 2
		leveled = true
	}

	if err := s.DB.Model(&model.UserCharacter{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"level":        c.Level,
			"xp":           c.XP,
			"skill_points": c.SkillPoints,
		}).Error; err != nil {
		return false, 0, err
	}

	if leveled {
		_, _ = s.AdjustColumn(userID, "crowns", 1)
	}

	return leveled, c.Level, nil
}

// AllocateStat adds one point to a stat, consuming a skill point.
func (s *Store) AllocateStat(userID int64, stat string) error {
	c, err := s.EnsureCharacter(userID)
	if err != nil {
		return err
	}
	if c.SkillPoints <= 0 {
		return nil
	}
	allowed := map[string]string{"str": "str", "dex": "dex", "int": "int", "vit": "vit", "luk": "luk"}
	col, ok := allowed[stat]
	if !ok {
		return nil
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.UserCharacter{}).
			Where("user_id = ? AND skill_points > 0", userID).
			UpdateColumn("skill_points", gorm.Expr("skill_points - 1")).Error; err != nil {
			return err
		}
		return tx.Model(&model.UserCharacter{}).
			Where("user_id = ?", userID).
			UpdateColumn(col, gorm.Expr(col+" + 1")).Error
	})
}

// GetEquipment returns the player's equipped items as a slot→itemID map.
func (s *Store) GetEquipment(userID int64) (map[string]string, error) {
	var rows []model.CharacterEquipment
	if err := s.DB.Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[r.Slot] = r.ItemID
	}
	return out, nil
}

// EquipItem equips an item to a slot, unequipping any previous item in that
// slot first. The item is not consumed from inventory.
func (s *Store) EquipItem(userID int64, slot, itemID string) error {
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "slot"}},
		DoUpdates: clause.Assignments(map[string]any{"item_id": itemID}),
	}).Create(&model.CharacterEquipment{
		UserID: userID, Slot: slot, ItemID: itemID,
	}).Error
}

// UnequipSlot removes whatever is equipped in the given slot.
func (s *Store) UnequipSlot(userID int64, slot string) error {
	return s.DB.Where("user_id = ? AND slot = ?", userID, slot).
		Delete(&model.CharacterEquipment{}).Error
}

// SetActiveBuff records a buff that was just activated.
func (s *Store) SetActiveBuff(userID int64, skillID string) error {
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "skill_id"}},
		DoUpdates: clause.Assignments(map[string]any{"created_at": time.Now()}),
	}).Create(&model.ActiveBuff{
		UserID: userID, SkillID: skillID, CreatedAt: time.Now(),
	}).Error
}

// ConsumeActiveBuff removes a buff if it exists and returns whether it did.
func (s *Store) ConsumeActiveBuff(userID int64, skillID string) (bool, error) {
	res := s.DB.Where("user_id = ? AND skill_id = ?", userID, skillID).
		Delete(&model.ActiveBuff{})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// HasActiveBuff checks whether a buff is currently active.
func (s *Store) HasActiveBuff(userID int64, skillID string) (bool, error) {
	var count int64
	err := s.DB.Model(&model.ActiveBuff{}).
		Where("user_id = ? AND skill_id = ?", userID, skillID).
		Count(&count).Error
	return count > 0, err
}

// CheckSkillAvailability checks whether a skill can be used (under daily limit
// and off cooldown). Returns a human-readable reason when unavailable.
func (s *Store) CheckSkillAvailability(userID int64, skillID string, dailyLimit int, cooldownMins int) (available bool, reason string, err error) {
	ok, remaining, err := s.CheckGameLimit(userID, "skill_"+skillID, dailyLimit)
	if err != nil {
		return false, "", err
	}
	if !ok {
		return false, "daily limit reached", nil
	}
	if remaining <= 0 {
		return false, "daily limit reached", nil
	}
	ready, err := s.CheckCooldown(userID, "skill_"+skillID, time.Duration(cooldownMins)*time.Minute)
	if err != nil {
		return false, "", err
	}
	if !ready {
		return false, "on cooldown", nil
	}
	return true, "", nil
}

// UseSkill records a skill use (sets cooldown and increments daily limit).
func (s *Store) UseSkill(userID int64, skillID string) error {
	if err := s.SetCooldown(userID, "skill_"+skillID); err != nil {
		return err
	}
	return s.IncrementGameLimit(userID, "skill_"+skillID)
}
