package pets

import (
	"encoding/json"
	"errors"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) DB() *gorm.DB { return s.store.DB }

const (
	MaxPetLevel    = 50
	BasePetSlots   = 3
	MaxBond        = 100
	SkillInterval  = 10
	BondFeedAmount = 1
)

// ─── Pet CRUD ──────────────────────────────────────────────────

func (s *Service) GetPets(userID int64) ([]model.UserPet, error) {
	var pets []model.UserPet
	err := s.store.DB.Where("user_id = ?", userID).Find(&pets).Error
	return pets, err
}

func (s *Service) GetPetByID(petID int64) (*model.UserPet, error) {
	var pet model.UserPet
	err := s.store.DB.First(&pet, petID).Error
	if err != nil {
		return nil, err
	}
	return &pet, nil
}

func (s *Service) GetActivePet(userID int64) (*model.UserPet, error) {
	var pet model.UserPet
	err := s.store.DB.Where("user_id = ? AND is_active = ?", userID, true).First(&pet).Error
	if err != nil {
		return nil, err
	}
	return &pet, nil
}

func (s *Service) SetActivePet(userID int64, petID int64) error {
	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.UserPet{}).
			Where("user_id = ? AND is_active = ?", userID, true).
			Update("is_active", false).Error; err != nil {
			return err
		}
		return tx.Model(&model.UserPet{}).
			Where("id = ? AND user_id = ?", petID, userID).
			Update("is_active", true).Error
	})
}

func (s *Service) CreatePet(userID int64, petType string) (*model.UserPet, error) {
	pt, ok := PetTypes[petType]
	if !ok {
		return nil, nil
	}

	history := []model.PetHistoryEntry{
		{Time: time.Now(), Event: "hatched", Detail: pt.Emoji + " A wild **" + petType + "** emerged from its egg! It looks at you with curious eyes."},
	}
	hJSON, _ := json.Marshal(history)

	pet := model.UserPet{
		UserID:      userID,
		PetType:     petType,
		Nickname:    petType,
		MaxHP:       pt.MaxHP,
		HP:          pt.MaxHP,
		Atk:         pt.Atk,
		Defense:     pt.Defense,
		Speed:       pt.Speed,
		DGE:         pt.DGE,
		ACC:         pt.ACC,
		CritC:       pt.CritC,
		CritD:       pt.CritD,
		Bonus:       pt.Bonus,
		Elo:         1000,
		Personality: RandomPersonality(),
		History:     string(hJSON),
	}
	err := s.store.DB.Create(&pet).Error
	if err != nil {
		return nil, err
	}
	return &pet, nil
}

func (s *Service) UpdatePet(pet *model.UserPet) error {
	return s.store.DB.Save(pet).Error
}

func (s *Service) DeletePet(petID int64) error {
	return s.store.DB.Delete(&model.UserPet{}, petID).Error
}

func (s *Service) TransferPet(petID int64, newOwnerID int64) error {
	return s.store.DB.Model(&model.UserPet{}).
		Where("id = ?", petID).
		Update("user_id", newOwnerID).Error
}

func (s *Service) HealPet(pet *model.UserPet, cost int) error {
	if _, err := s.store.UpdateBalance(pet.UserID, -cost); err != nil {
		return err
	}
	pet.HP = pet.MaxHP
	return s.UpdatePet(pet)
}

// ─── Pet Capacity ──────────────────────────────────────────────

func (s *Service) PetCount(userID int64) int {
	var count int64
	s.store.DB.Model(&model.UserPet{}).Where("user_id = ?", userID).Count(&count)
	return int(count)
}

func (s *Service) ActivePetCount(userID int64) int {
	var count int64
	s.store.DB.Model(&model.UserPet{}).Where("user_id = ? AND in_sanctuary = ?", userID, false).Count(&count)
	return int(count)
}

func (s *Service) SanctuaryPetCount(userID int64) int {
	var count int64
	s.store.DB.Model(&model.UserPet{}).Where("user_id = ? AND in_sanctuary = ?", userID, true).Count(&count)
	return int(count)
}

func (s *Service) MaxPetSlots(userID int64) int {
	var u model.User
	if err := s.store.DB.Where("user_id = ?", userID).First(&u).Error; err != nil {
		return BasePetSlots
	}
	return BasePetSlots + u.ExtraPetSlots
}

func (s *Service) CanCreatePet(userID int64) bool {
	return s.ActivePetCount(userID) < s.MaxPetSlots(userID)
}

func (s *Service) GetSanctuaryPets(userID int64) ([]model.UserPet, error) {
	var pets []model.UserPet
	err := s.store.DB.Where("user_id = ? AND in_sanctuary = ?", userID, true).Find(&pets).Error
	return pets, err
}

func (s *Service) GetActiveRosterPets(userID int64) ([]model.UserPet, error) {
	var pets []model.UserPet
	err := s.store.DB.Where("user_id = ? AND in_sanctuary = ?", userID, false).Find(&pets).Error
	return pets, err
}

// ─── Leveling ──────────────────────────────────────────────────

type LevelResult struct {
	Leveled         bool
	NewLevel        int
	SkillPointGained bool
}

func (s *Service) AddXP(pet *model.UserPet, amount int) *LevelResult {
	if pet.Level >= MaxPetLevel {
		pet.XP = 0
		return &LevelResult{Leveled: false, NewLevel: pet.Level}
	}
	rMult := RarityXP[petTypeRarity(pet.PetType)]

	pet.XP += amount
	res := &LevelResult{NewLevel: pet.Level}

	for pet.XP >= xpForLevel(pet.Level, rMult) && pet.Level < MaxPetLevel {
		pet.XP -= xpForLevel(pet.Level, rMult)
		pet.Level++

		pet.MaxHP += 2
		pet.HP += 2
		pet.Atk += 1
		if pet.Level%2 == 0 {
			pet.Defense += 1
		}
		if pet.Level%5 == 0 {
			pet.Speed += 1
			pet.DGE += 1
			pet.ACC += 1
		}
		if pet.Level%SkillInterval == 0 {
			pet.SkillPoints++
			res.SkillPointGained = true
		}
		if pet.Level == 10 || pet.Level == 20 || pet.Level == 30 || pet.Level == 40 || pet.Level == 50 {
			pet.Elo = 1000
		}

		s.RecordHistory(pet, "leveled", "**"+pet.Nickname+"** reached level **"+itoa(pet.Level)+"**!")
		res.Leveled = true
		res.NewLevel = pet.Level
	}
	// Auto-start boss_league quest when any pet reaches level 20
	if pet.Level >= 20 {
		var existing model.UserQuest
		err := s.store.DB.Where("user_id = ? AND quest_id = ?", pet.UserID, "boss_league").First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = s.store.CreateQuest(pet.UserID, "boss_league")
		}
	}
	return res
}

func xpForLevel(level int, rMult float64) int {
	return int(float64(level*level) * rMult * 15)
}

// ─── Bond System ───────────────────────────────────────────────

func (s *Service) AddBond(pet *model.UserPet, amount int) int {
	if pet.BondLevel >= MaxBond {
		return pet.BondLevel
	}
	prev := pet.BondLevel
	pet.BondLevel += amount
	if pet.BondLevel > MaxBond {
		pet.BondLevel = MaxBond
	}
	for _, milestone := range []int{25, 50, 75, 100} {
		if prev < milestone && pet.BondLevel >= milestone {
			s.RecordHistory(pet, "bonded",
				milestoneText(pet.Nickname, milestone))
		}
	}
	return pet.BondLevel
}

func milestoneText(name string, level int) string {
	switch level {
	case 25:
		return "💕 **" + name + "** starts to trust you. A quiet understanding forms between you."
	case 50:
		return "❤️ **" + name + "** has become more than a pet — you're partners now. It responds to your mood."
	case 75:
		return "💖 The bond with **" + name + "** is unbreakable. It would follow you anywhere."
	case 100:
		return "✨ **" + name + "** is part of your soul. No words are needed — you understand each other completely."
	}
	return ""
}

func (s *Service) BondStatBonus(pet *model.UserPet) (float64, int) {
	// Bond gives a small stat multiplier (up to 15% at max bond)
	mult := 1.0 + float64(pet.BondLevel)/MaxBond*0.15
	return mult, pet.BondLevel / 10 // bonus bond tier for display
}

// ─── History ───────────────────────────────────────────────────

func (s *Service) RecordHistory(pet *model.UserPet, event, detail string) {
	entries := make([]model.PetHistoryEntry, 0)
	if pet.History != "" && pet.History != "[]" {
		_ = json.Unmarshal([]byte(pet.History), &entries)
	}
	entries = append(entries, model.PetHistoryEntry{
		Time:   time.Now(),
		Event:  event,
		Detail: detail,
	})
	// Keep last 20 entries max
	if len(entries) > 20 {
		entries = entries[len(entries)-20:]
	}
	hJSON, _ := json.Marshal(entries)
	pet.History = string(hJSON)
}

func (s *Service) GetHistory(pet *model.UserPet) []model.PetHistoryEntry {
	entries := make([]model.PetHistoryEntry, 0)
	if pet.History != "" && pet.History != "[]" {
		_ = json.Unmarshal([]byte(pet.History), &entries)
	}
	return entries
}

// ─── Skills ────────────────────────────────────────────────────

func (s *Service) SelectSkill(petID int64, slot int, skillID string) error {
	return s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "pet_id"}, {Name: "slot"}},
		DoUpdates: clause.Assignments(map[string]any{"skill_id": skillID}),
	}).Create(&model.UserPetSkill{
		PetID:   petID,
		Slot:    slot,
		SkillID: skillID,
	}).Error
}

func (s *Service) ResetSkills(petID int64) error {
	return s.store.DB.Where("pet_id = ?", petID).
		Delete(&model.UserPetSkill{}).Error
}

func (s *Service) GetPetSkills(petID int64) ([]model.UserPetSkill, error) {
	var skills []model.UserPetSkill
	err := s.store.DB.Where("pet_id = ?", petID).Find(&skills).Error
	return skills, err
}

func (s *Service) SpendSkillPoint(pet *model.UserPet) error {
	if pet.SkillPoints <= 0 {
		return nil
	}
	pet.SkillPoints--
	return s.UpdatePet(pet)
}

func (s *Service) RerollPersonality(pet *model.UserPet) error {
	old := pet.Personality
	pet.Personality = RandomPersonality()
	s.RecordHistory(pet, "personality_change",
		"🌀 **"+pet.Nickname+"** underwent a mysterious transformation... "+
			"It's now **"+PersonalityTraits[pet.Personality].Name+"**! (was "+PersonalityTraits[old].Name+")")
	return s.UpdatePet(pet)
}

// ─── Feeding ───────────────────────────────────────────────────

func (s *Service) FeedPet(pet *model.UserPet, stat string, amount int) error {
	rCap := RarityFoodCapacity[petTypeRarity(pet.PetType)]
	if pet.FoodEaten >= rCap*pet.Level {
		return nil
	}
	switch stat {
	case "max_hp":
		pet.MaxHP += amount
		pet.HP += amount
	case "atk":
		pet.Atk += amount
	case "defense":
		pet.Defense += amount
	case "speed":
		pet.Speed += amount
	case "dge":
		pet.DGE += amount
	case "acc":
		pet.ACC += amount
	case "crit_c":
		pet.CritC += amount
	case "crit_d":
		pet.CritD += float64(amount)
	}
	pet.FoodEaten++
	s.AddBond(pet, BondFeedAmount)
	return s.UpdatePet(pet)
}

// ─── Forget / Prestige ─────────────────────────────────────────

func (s *Service) ForgetXP(pet *model.UserPet) bool {
	if pet.Level < 20 {
		return false
	}
	pt, ok := PetTypes[pet.PetType]
	if !ok {
		return false
	}
	pet.XP = 0
	pet.Level = 10
	pet.MaxHP = pt.MaxHP + 20
	pet.HP = pet.MaxHP
	pet.Atk = pt.Atk + 10
	pet.Defense = pt.Defense + 5
	pet.CritC = pt.CritC
	pet.CritD = pt.CritD
	pet.ACC = pt.ACC
	pet.DGE = pt.DGE
	pet.Speed = pt.Speed
	pet.FoodEaten = 0
	pet.SkillPoints = 0
	_ = s.ResetSkills(pet.ID)
	s.RecordHistory(pet, "forgot", "🌀 **"+pet.Nickname+"** went through a mysterious transformation, losing its memories but keeping its bond.")
	return true
}

// ─── ELO ───────────────────────────────────────────────────────

func (s *Service) UpdateElo(p1, p2 *model.UserPet, result float64) (int, int) {
	if p1.Level < 5 || p2.Level < 5 {
		return 0, 0
	}
	K := 32.0
	eS := 1.0 / (1.0 + math.Pow(10, float64(p2.Elo-p1.Elo)/400))
	eO := 1.0 / (1.0 + math.Pow(10, float64(p1.Elo-p2.Elo)/400))
	scoreO := 1.0 - result
	dS := int(K * (result - eS))
	dO := int(K * (scoreO - eO))
	p1.Elo += dS
	p2.Elo += dO
	if p1.Elo < 0 {
		p1.Elo = 0
	}
	if p2.Elo < 0 {
		p2.Elo = 0
	}
	return dS, dO
}

func (s *Service) GetServerElo(petID, serverID int64) (int, error) {
	var elo model.ServerPetElo
	err := s.store.DB.Where("pet_id = ? AND server_id = ?", petID, serverID).First(&elo).Error
	if err != nil {
		return 1000, nil
	}
	return elo.Elo, nil
}

func (s *Service) UpdateServerElo(petID, serverID int64, elo int) error {
	return s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "pet_id"}, {Name: "server_id"}},
		DoUpdates: clause.Assignments(map[string]any{"elo": elo}),
	}).Create(&model.ServerPetElo{PetID: petID, ServerID: serverID, Elo: elo}).Error
}

// ─── Achievement Helpers ───────────────────────────────────────

func (s *Service) CheckAndUnlock(userID int64) ([]*achievement.Achievement, error) {
	return achievement.CheckAndUnlock(s.store.DB, userID)
}

func (s *Service) IncrementStat(userID int64, stat string, amount int) error {
	return achievement.IncrementStat(s.store.DB, userID, stat, amount)
}

// ─── Helpers ───────────────────────────────────────────────────

func petTypeRarity(petType string) string {
	if pt, ok := PetTypes[petType]; ok {
		return pt.Rarity
	}
	return RarityCommon
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	if neg {
		out = "-" + out
	}
	return out
}
