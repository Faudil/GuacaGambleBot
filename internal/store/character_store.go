package store

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/items"
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
// Deprecated: use GetEquipped instead.
func (s *Store) GetEquipment(userID int64) (map[string]string, error) {
	var rows []model.UserEquipment
	if err := s.DB.Where("user_id = ? AND is_equipped = ?", userID, true).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[r.EquipSlot] = r.BaseID
	}
	return out, nil
}

// GetEquipped returns all currently equipped UserEquipment instances for a user.
func (s *Store) GetEquipped(userID int64) ([]model.UserEquipment, error) {
	var rows []model.UserEquipment
	err := s.DB.Where("user_id = ? AND is_equipped = ?", userID, true).Find(&rows).Error
	return rows, err
}

// GetUnequipped returns all unequipped UserEquipment instances for a user.
func (s *Store) GetUnequipped(userID int64) ([]model.UserEquipment, error) {
	var rows []model.UserEquipment
	err := s.DB.Where("user_id = ? AND is_equipped = ?", userID, false).Find(&rows).Error
	return rows, err
}

// GetUnequippedBySlot returns unequipped items for a specific equipment slot.
func (s *Store) GetUnequippedBySlot(userID int64, slot string) ([]model.UserEquipment, error) {
	var rows []model.UserEquipment
	err := s.DB.Where("user_id = ? AND is_equipped = ? AND equip_slot = ?", userID, false, slot).Find(&rows).Error
	return rows, err
}

// GetAllUserEquipment returns all UserEquipment instances for a user (equipped + unequipped).
func (s *Store) GetAllUserEquipment(userID int64) ([]model.UserEquipment, error) {
	var rows []model.UserEquipment
	err := s.DB.Where("user_id = ?", userID).Find(&rows).Error
	return rows, err
}

// CreateEquipment creates a new UserEquipment instance and returns it.
func (s *Store) CreateEquipment(userID int64, baseID, name, emoji, rarity, equipSlot string,
	statSTR, statDEX, statINT, statVIT, statLUK int,
	affixes []byte, setID string) (*model.UserEquipment, error) {
	affixStr := "[]"
	if len(affixes) > 0 {
		affixStr = string(affixes)
	}
	eq := model.UserEquipment{
		UserID:     userID,
		BaseID:     baseID,
		Name:       name,
		Emoji:      emoji,
		Rarity:     rarity,
		EquipSlot:  equipSlot,
		StatSTR:    statSTR,
		StatDEX:    statDEX,
		StatINT:    statINT,
		StatVIT:    statVIT,
		StatLUK:    statLUK,
		Affixes:    affixStr,
		SetID:      setID,
		IsEquipped: false,
	}
	if err := s.DB.Create(&eq).Error; err != nil {
		return nil, err
	}
	return &eq, nil
}

// EquipInstance equips a UserEquipment instance. If another item was equipped
// in the same slot, it is unequipped first.
func (s *Store) EquipInstance(userID int64, equipID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var target model.UserEquipment
		if err := tx.First(&target, equipID).Error; err != nil {
			return err
		}
		if target.UserID != userID {
			return nil
		}
		if err := tx.Model(&model.UserEquipment{}).
			Where("user_id = ? AND equip_slot = ? AND is_equipped = ?", userID, target.EquipSlot, true).
			Update("is_equipped", false).Error; err != nil {
			return err
		}
		return tx.Model(&model.UserEquipment{}).
			Where("id = ?", equipID).
			Update("is_equipped", true).Error
	})
}

// UnequipSlot unequips whatever item is in the given slot.
func (s *Store) UnequipSlot(userID int64, slot string) error {
	return s.DB.Model(&model.UserEquipment{}).
		Where("user_id = ? AND equip_slot = ? AND is_equipped = ?", userID, slot, true).
		Update("is_equipped", false).Error
}

// CreateEquipmentFromAffixes is a convenience that accepts parsed affixes and
// tallies the total stats from base + affixes before creating the row.
func (s *Store) CreateEquipmentFromAffixes(userID int64, baseID, name, emoji string,
	rarity string, equipSlot string,
	baseSTR, baseDEX, baseINT, baseVIT, baseLUK int,
	affixes []items.AppliedAffix, setID string) (*model.UserEquipment, error) {

	totalSTR, totalDEX, totalINT, totalVIT, totalLUK := baseSTR, baseDEX, baseINT, baseVIT, baseLUK
	for _, a := range affixes {
		switch a.Stat {
		case "str":
			totalSTR += a.Value
		case "dex":
			totalDEX += a.Value
		case "int":
			totalINT += a.Value
		case "vit":
			totalVIT += a.Value
		case "luk":
			totalLUK += a.Value
		}
	}
	data, _ := json.Marshal(affixes)
	return s.CreateEquipment(userID, baseID, name, emoji, rarity, equipSlot,
		totalSTR, totalDEX, totalINT, totalVIT, totalLUK, data, setID)
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
