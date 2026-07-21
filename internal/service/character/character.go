package character

import (
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

// Service holds the Character cog business logic.
type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

// ProfileResult holds the full profile data shown to the player.
type ProfileResult struct {
	Wallet        int
	Bank          int
	Crowns        int
	AchCount      int
	Level         int
	XP            int
	XPNext        int
	SkillPoints   int
	STR           int
	DEX           int
	INT           int
	VIT           int
	LUK           int
	EquipSTR      int
	EquipDEX      int
	EquipINT      int
	EquipVIT      int
	EquipLUK      int
	TotalJobLevel int
}

// Profile returns the full character profile.
func (s *Service) Profile(userID int64) (*ProfileResult, error) {
	wallet, err := s.store.GetBalance(userID)
	if err != nil {
		return nil, err
	}
	_, bank, err := s.store.GetBankData(userID)
	if err != nil {
		return nil, err
	}
	var u model.User
	if err := s.store.DB.Where("user_id = ?", userID).First(&u).Error; err != nil {
		return nil, err
	}
	var achCount int64
	if err := s.store.DB.Model(&model.UserAchievement{}).
		Where("user_id = ?", userID).Count(&achCount).Error; err != nil {
		return nil, err
	}
	c, err := s.store.EnsureCharacter(userID)
	if err != nil {
		return nil, err
	}
	eq, _ := s.store.GetEquipment(userID)

	var jobTotal int64
	s.store.DB.Model(&model.Job{}).Select("COALESCE(SUM(level), 0)").Where("user_id = ?", userID).Scan(&jobTotal)

	eqSTR, eqDEX, eqINT, eqVIT, eqLUK := equipBonuses(eq)

	return &ProfileResult{
		Wallet:        wallet,
		Bank:          bank,
		Crowns:        u.Crowns,
		AchCount:      int(achCount),
		Level:         c.Level,
		XP:            c.XP,
		XPNext:        store.XPForCharacterLevel(c.Level),
		SkillPoints:   c.SkillPoints,
		STR:           c.STR,
		DEX:           c.DEX,
		INT:           c.INT,
		VIT:           c.VIT,
		LUK:           c.LUK,
		EquipSTR:      eqSTR,
		EquipDEX:      eqDEX,
		EquipINT:      eqINT,
		EquipVIT:      eqVIT,
		EquipLUK:      eqLUK,
		TotalJobLevel: int(jobTotal),
	}, nil
}

func equipBonuses(eq map[string]string) (str, dex, intt, vit, luk int) {
	for _, itemID := range eq {
		it := items.Get(itemID)
		if it == nil {
			continue
		}
		str += it.StatSTR
		dex += it.StatDEX
		intt += it.StatINT
		vit += it.StatVIT
		luk += it.StatLUK
	}
	return
}

// EffectiveStats holds base plus equipment stat bonuses.
type EffectiveStats struct {
	BaseSTR, BaseDEX, BaseINT, BaseVIT, BaseLUK int
	BonSTR, BonDEX, BonINT, BonVIT, BonLUK      int
}

// TotalSTR returns the effective STR (base + equipment).
func (e *EffectiveStats) TotalSTR() int { return e.BaseSTR + e.BonSTR }
func (e *EffectiveStats) TotalDEX() int { return e.BaseDEX + e.BonDEX }
func (e *EffectiveStats) TotalINT() int { return e.BaseINT + e.BonINT }
func (e *EffectiveStats) TotalVIT() int { return e.BaseVIT + e.BonVIT }
func (e *EffectiveStats) TotalLUK() int { return e.BaseLUK + e.BonLUK }

// GetEffectiveStats returns base + equipment stats for a user.
func GetEffectiveStats(s *store.Store, userID int64) (*EffectiveStats, error) {
	c, err := s.EnsureCharacter(userID)
	if err != nil {
		return nil, err
	}
	eq, _ := s.GetEquipment(userID)
	bonSTR, bonDEX, bonINT, bonVIT, bonLUK := equipBonuses(eq)
	return &EffectiveStats{
		BaseSTR: c.STR, BaseDEX: c.DEX, BaseINT: c.INT, BaseVIT: c.VIT, BaseLUK: c.LUK,
		BonSTR: bonSTR, BonDEX: bonDEX, BonINT: bonINT, BonVIT: bonVIT, BonLUK: bonLUK,
	}, nil
}

// Bonus multipliers for other services to use.

// GetSTRBonus returns the gather-quantity multiplier based on STR.
func GetSTRBonus(s *store.Store, userID int64) float64 {
	es, err := GetEffectiveStats(s, userID)
	if err != nil {
		return 1.0
	}
	return 1.0 + float64(es.TotalSTR())*0.01
}

// GetDEXBonus returns the fishing-reaction-time multiplier based on DEX.
func GetDEXBonus(s *store.Store, userID int64) float64 {
	es, err := GetEffectiveStats(s, userID)
	if err != nil {
		return 1.0
	}
	return 1.0 + float64(es.TotalDEX())*0.01
}

// GetINTBonus returns the crafting/XP multiplier based on INT.
func GetINTBonus(s *store.Store, userID int64) float64 {
	es, err := GetEffectiveStats(s, userID)
	if err != nil {
		return 1.0
	}
	return 1.0 + float64(es.TotalINT())*0.02
}

// GetVITReduction returns a risk/collapse reduction factor (0-1) based on VIT.
func GetVITReduction(s *store.Store, userID int64) float64 {
	es, err := GetEffectiveStats(s, userID)
	if err != nil {
		return 0
	}
	return float64(es.TotalVIT()) * 0.002
}

// GetLUKBonus returns a rare-drop multiplier based on LUK.
func GetLUKBonus(s *store.Store, userID int64) float64 {
	es, err := GetEffectiveStats(s, userID)
	if err != nil {
		return 0
	}
	return float64(es.TotalLUK()) * 0.02
}

// Buff helpers.

// HasBuff checks whether a skill buff is active.
func HasBuff(s *store.Store, userID int64, skillID string) bool {
	ok, _ := s.HasActiveBuff(userID, skillID)
	return ok
}

// ConsumeBuff consumes one trigger of a skill buff and returns true if it was active.
func ConsumeBuff(s *store.Store, userID int64, skillID string) bool {
	ok, _ := s.ConsumeActiveBuff(userID, skillID)
	return ok
}

// AddXP awards character XP and handles level-up crowns. Returns whether the
// player leveled up.
func AddXP(s *store.Store, userID int64, amount int) (bool, int) {
	if amount <= 0 {
		return false, 0
	}
	leveled, lvl, _ := s.AddCharacterXP(userID, amount)
	return leveled, lvl
}

// Service methods for equipment.

func (s *Service) EquipItem(userID int64, slot, itemID string) error {
	return s.store.EquipItem(userID, slot, itemID)
}

func (s *Service) UnequipSlot(userID int64, slot string) error {
	return s.store.UnequipSlot(userID, slot)
}

func (s *Service) GetEquipment(userID int64) (map[string]string, error) {
	return s.store.GetEquipment(userID)
}

// Service methods for skills.

func (s *Service) ActivateSkill(userID int64, skillID string) error {
	return s.store.UseSkill(userID, skillID)
}

func (s *Service) SetBuff(userID int64, skillID string) error {
	return s.store.SetActiveBuff(userID, skillID)
}

// SkillStatus describes a single skill's current status for a user.
type SkillStatus struct {
	Skill
	Available bool
	Reason    string // "available", "locked", "on cooldown", "daily limit reached"
	UsesLeft  int
}

func (s *Service) GetSkills(userID int64) ([]SkillStatus, error) {
	c, err := s.store.EnsureCharacter(userID)
	if err != nil {
		return nil, err
	}
	all := AllSkills()
	statuses := make([]SkillStatus, 0, len(all))
	for _, sk := range all {
		st := SkillStatus{Skill: sk}
		if c.Level < sk.UnlockLevel {
			st.Available = false
			st.Reason = "locked"
			statuses = append(statuses, st)
			continue
		}
		ok, remaining, err := s.store.CheckGameLimit(userID, "skill_"+sk.ID, sk.DailyLimit)
		if err != nil || !ok || remaining <= 0 {
			st.Available = false
			if remaining <= 0 {
				st.Reason = "daily limit reached"
			} else {
				st.Reason = "on cooldown"
			}
			statuses = append(statuses, st)
			continue
		}
		ready, err := s.store.CheckCooldown(userID, "skill_"+sk.ID, sk.CooldownDur())
		if err != nil || !ready {
			st.Available = false
			st.Reason = "on cooldown"
			statuses = append(statuses, st)
			continue
		}
		st.Available = true
		st.Reason = "available"
		st.UsesLeft = remaining
		statuses = append(statuses, st)
	}
	return statuses, nil
}
