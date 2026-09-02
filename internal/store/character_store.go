package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
)

var (
	ErrNoSkillPoints = errors.New("no skill points available")
	ErrInvalidStat   = errors.New("invalid stat")
	ErrNoPerkPoints  = errors.New("no perk points available")
	ErrLevelTooLow   = errors.New("level too low to equip this item")
)

// XPForCharacterLevel returns the XP required to reach the next level.
// Levels 1-20 grow linearly; past level 20 requirements grow exponentially so
// late levels become long-term goals.
func XPForCharacterLevel(level int) int {
	if level <= 20 {
		return 300 * level
	}
	return int(6000 * math.Pow(1.06, float64(level-20)))
}

// hasPassive reports whether the character row's passive list contains id.
func hasPassive(c *model.UserCharacter, id string) bool {
	var list []string
	if err := json.Unmarshal([]byte(c.Passives), &list); err != nil {
		return false
	}
	for _, p := range list {
		if p == id {
			return true
		}
	}
	return false
}

// HasPassive reports whether a user owns the given passive perk.
func (s *Store) HasPassive(userID int64, id string) bool {
	c, err := s.GetCharacter(userID)
	if err != nil {
		return false
	}
	return hasPassive(c, id)
}

// DecrementPerkPoints consumes one pending perk choice. It returns an error
// when the user has none left.
func (s *Store) DecrementPerkPoints(userID int64) error {
	res := s.DB.Model(&model.UserCharacter{}).
		Where("user_id = ? AND perk_points > 0", userID).
		UpdateColumn("perk_points", gorm.Expr("perk_points - 1"))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNoPerkPoints
	}
	return nil
}

// AddPassive appends a passive perk ID to the user's passive list.
func (s *Store) AddPassive(userID int64, id string) error {
	c, err := s.GetCharacter(userID)
	if err != nil {
		return err
	}
	var list []string
	_ = json.Unmarshal([]byte(c.Passives), &list)
	for _, p := range list {
		if p == id {
			return nil
		}
	}
	list = append(list, id)
	data, _ := json.Marshal(list)
	return s.DB.Model(&model.UserCharacter{}).
		Where("user_id = ?", userID).
		Update("passives", string(data)).Error
}

// GetPerkPoints returns the user's pending perk choices.
func (s *Store) GetPerkPoints(userID int64) (int, error) {
	c, err := s.GetCharacter(userID)
	if err != nil {
		return 0, err
	}
	return c.PerkPoints, nil
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
// leveled up and the new level. Crowns are awarded on each level-up. An active
// quick_learner buff doubles the XP amount and is consumed.
func (s *Store) AddCharacterXP(userID int64, amount int) (leveledUp bool, newLevel int, err error) {
	c, err := s.EnsureCharacter(userID)
	if err != nil {
		return false, 0, err
	}

	if ok, _ := s.ConsumeActiveBuff(userID, "quick_learner"); ok {
		amount *= 2
	}
	if hasPassive(c, "perk_xp_boost") {
		amount = amount * 105 / 100
	}

	c.XP += amount
	leveled := false
	for c.XP >= XPForCharacterLevel(c.Level) && c.Level < 100 {
		c.XP -= XPForCharacterLevel(c.Level)
		c.Level++
		c.SkillPoints += 2
		c.PerkPoints++
		leveled = true
	}

	if err := s.DB.Model(&model.UserCharacter{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"level":        c.Level,
			"xp":           c.XP,
			"skill_points": c.SkillPoints,
			"perk_points":  c.PerkPoints,
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
		return ErrNoSkillPoints
	}
	allowed := map[string]string{"str": "str", "dex": "dex", "int": "int", "vit": "vit", "luk": "luk"}
	col, ok := allowed[stat]
	if !ok {
		return ErrInvalidStat
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

// AddStatPoints directly adds n points to a stat without consuming skill
// points (used by level-up perks).
func (s *Store) AddStatPoints(userID int64, stat string, n int) error {
	allowed := map[string]string{"str": "str", "dex": "dex", "int": "int", "vit": "vit", "luk": "luk"}
	col, ok := allowed[stat]
	if !ok {
		return ErrInvalidStat
	}
	return s.DB.Model(&model.UserCharacter{}).
		Where("user_id = ?", userID).
		UpdateColumn(col, gorm.Expr(col+" + ?", n)).Error
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
// minLevel is the minimum character level required to equip the piece.
func (s *Store) CreateEquipment(userID int64, baseID, name, emoji, rarity, equipSlot string,
	minLevel int,
	statSTR, statDEX, statINT, statVIT, statLUK int,
	affixes []byte, setID string) (*model.UserEquipment, error) {
	return s.CreateEquipmentTx(s.DB, userID, baseID, name, emoji, rarity, equipSlot, minLevel,
		statSTR, statDEX, statINT, statVIT, statLUK, affixes, setID)
}

// CreateEquipmentTx creates a new UserEquipment instance inside an existing
// transaction. Callers running within an outer s.DB.Transaction must use this
// instead of CreateEquipment: the transaction holds the pool's single SQLite
// connection, so a nested pool query would deadlock (maxOpenConns is 1 in
// production).
func (s *Store) CreateEquipmentTx(tx *gorm.DB, userID int64, baseID, name, emoji, rarity, equipSlot string,
	minLevel int,
	statSTR, statDEX, statINT, statVIT, statLUK int,
	affixes []byte, setID string) (*model.UserEquipment, error) {
	if minLevel <= 0 {
		minLevel = 1
	}
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
		MinLevel:   minLevel,
		StatSTR:    statSTR,
		StatDEX:    statDEX,
		StatINT:    statINT,
		StatVIT:    statVIT,
		StatLUK:    statLUK,
		Affixes:    affixStr,
		SetID:      setID,
		IsEquipped: false,
	}
	if err := tx.Create(&eq).Error; err != nil {
		return nil, err
	}
	if err := s.recordItemDiscovery(tx, userID, baseID); err != nil {
		return nil, err
	}
	return &eq, nil
}

// EquipInstance equips a UserEquipment instance. If another item was equipped
// in the same slot, it is unequipped first. Items above the character's level
// are rejected.
func (s *Store) EquipInstance(userID int64, equipID uint) error {
	var target model.UserEquipment
	if err := s.DB.First(&target, equipID).Error; err != nil {
		return err
	}
	if target.UserID != userID {
		return nil
	}
	c, err := s.EnsureCharacter(userID)
	if err != nil {
		return err
	}
	if c.Level < target.MinLevel {
		return fmt.Errorf("%w: requires level %d (you are %d)", ErrLevelTooLow, target.MinLevel, c.Level)
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
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
	rarity string, equipSlot string, minLevel int,
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
	return s.CreateEquipment(userID, baseID, name, emoji, rarity, equipSlot, minLevel,
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
