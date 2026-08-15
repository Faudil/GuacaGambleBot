package pets

import (
	"math/rand"
	"sort"

	"gorm.io/gorm/clause"

	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/model"
)

const (
	ArtifactPVPWinXP        = 100
	ArtifactPVPLossXP       = 50
	ArtifactExpeditionXP    = 120
	ArtifactHuntXP          = 80
	ArtifactDelveCombatXP   = 100
	ArtifactDelveCompleteXP = 300
	ArtifactDailyBonusXP    = 100
	ArtifactAutoBattleXP    = 5
	ArtifactMaxLevel        = 10
)

type ArtifactStatDef struct {
	ID         string
	Name       string
	Emoji      string
	PerLevelFn func(bp *battle.BattlePet, level int)
	MaxEffect  string
}

var ArtifactStats = []ArtifactStatDef{
	{ID: "impact", Name: "Impact", Emoji: "💥", MaxEffect: "+20% damage", PerLevelFn: func(bp *battle.BattlePet, level int) {
		bp.Atk = int(float64(bp.Atk) * (1.0 + 0.02*float64(level)))
	}},
	{ID: "piercing", Name: "Piercing", Emoji: "🗡️", MaxEffect: "Ignore 30% def", PerLevelFn: func(bp *battle.BattlePet, level int) {
		bp.PerkInt["artifact_piercing"] = level
	}},
	{ID: "resilience", Name: "Resilience", Emoji: "🛡️", MaxEffect: "-20% dmg taken", PerLevelFn: func(bp *battle.BattlePet, level int) {
		bp.Defense = int(float64(bp.Defense) * (1.0 + 0.02*float64(level)))
	}},
	{ID: "vampirism", Name: "Vampirism", Emoji: "🩸", MaxEffect: "20% lifesteal", PerLevelFn: func(bp *battle.BattlePet, level int) {
		bp.PerkInt["artifact_vampirism"] = level * 2
	}},
	{ID: "haste", Name: "Haste", Emoji: "⚡", MaxEffect: "+30 speed", PerLevelFn: func(bp *battle.BattlePet, level int) {
		bp.Speed += 3 * level
	}},
	{ID: "precision", Name: "Precision", Emoji: "🎯", MaxEffect: "+20 accuracy", PerLevelFn: func(bp *battle.BattlePet, level int) {
		bp.ACC += 2 * level
	}},
	{ID: "fortune", Name: "Fortune", Emoji: "🍀", MaxEffect: "+10% crit chance", PerLevelFn: func(bp *battle.BattlePet, level int) {
		bp.CritC += level
	}},
	{ID: "might", Name: "Might", Emoji: "💪", MaxEffect: "+30% crit dmg", PerLevelFn: func(bp *battle.BattlePet, level int) {
		bp.CritD += 0.03 * float64(level)
	}},
	{ID: "warding", Name: "Warding", Emoji: "🔮", MaxEffect: "+20% status resist", PerLevelFn: func(bp *battle.BattlePet, level int) {
		bp.PerkInt["artifact_warding"] = level * 2
	}},
	{ID: "rejuvenation", Name: "Rejuvenation", Emoji: "💚", MaxEffect: "10% HP regen/turn", PerLevelFn: func(bp *battle.BattlePet, level int) {
		bp.PerkInt["artifact_rejuvenation"] = level
	}},
}

var byArtifactStatID = func() map[string]*ArtifactStatDef {
	m := make(map[string]*ArtifactStatDef, len(ArtifactStats))
	for i := range ArtifactStats {
		m[ArtifactStats[i].ID] = &ArtifactStats[i]
	}
	return m
}()

func GetArtifactStat(id string) *ArtifactStatDef {
	return byArtifactStatID[id]
}

func artifactXPForLevel(level int) int {
	return level * level * 10
}

func ArtifactXPForLevel(level int) int {
	return artifactXPForLevel(level)
}

func TotalArtifactXPToMax() int {
	total := 0
	for lvl := 1; lvl < ArtifactMaxLevel; lvl++ {
		total += artifactXPForLevel(lvl)
	}
	return total
}

func (s *Service) GetArtifact(userID int64) (*model.UserPetArtifact, error) {
	var a model.UserPetArtifact
	err := s.store.DB.Where("user_id = ?", userID).First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Service) CreateArtifact(userID int64) (*model.UserPetArtifact, error) {
	stats := rand.Perm(len(ArtifactStats))
	a := &model.UserPetArtifact{
		UserID:   userID,
		Level:    1,
		XP:       0,
		Stat1:    ArtifactStats[stats[0]].ID,
		Stat1Lvl: 1,
		Stat2:    ArtifactStats[stats[1]].ID,
		Stat2Lvl: 1,
		Stat3:    ArtifactStats[stats[2]].ID,
		Stat3Lvl: 1,
	}
	err := s.store.DB.Create(a).Error
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Service) EnsureArtifact(userID int64) (*model.UserPetArtifact, error) {
	a, err := s.GetArtifact(userID)
	if err == nil {
		return a, nil
	}
	return s.CreateArtifact(userID)
}

func (s *Service) AddArtifactXP(userID int64, amount int) (*model.UserPetArtifact, bool, error) {
	a, err := s.EnsureArtifact(userID)
	if err != nil {
		return nil, false, err
	}
	if a.Level >= ArtifactMaxLevel {
		a.XP = 0
		return a, false, s.store.DB.Save(a).Error
	}
	a.XP += amount
	leveled := false
	for a.XP >= artifactXPForLevel(a.Level) && a.Level < ArtifactMaxLevel {
		a.XP -= artifactXPForLevel(a.Level)
		a.Level++
		a.UnspentPoints++
		leveled = true
	}
	if err := s.store.DB.Save(a).Error; err != nil {
		return nil, false, err
	}
	return a, leveled, nil
}

func (s *Service) LevelArtifactStat(userID int64, statPos int) (*model.UserPetArtifact, error) {
	a, err := s.GetArtifact(userID)
	if err != nil {
		return nil, err
	}
	if a.UnspentPoints <= 0 {
		return a, nil
	}
	switch statPos {
	case 0:
		a.Stat1Lvl++
	case 1:
		a.Stat2Lvl++
	case 2:
		a.Stat3Lvl++
	}
	a.UnspentPoints--
	err = s.store.DB.Save(a).Error
	return a, err
}

func (s *Service) ResetArtifact(userID int64) (*model.UserPetArtifact, error) {
	stats := rand.Perm(len(ArtifactStats))
	a := &model.UserPetArtifact{
		UserID:   userID,
		Level:    1,
		XP:       0,
		Stat1:    ArtifactStats[stats[0]].ID,
		Stat1Lvl: 1,
		Stat2:    ArtifactStats[stats[1]].ID,
		Stat2Lvl: 1,
		Stat3:    ArtifactStats[stats[2]].ID,
		Stat3Lvl: 1,
	}
	err := s.store.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"level": 1, "xp": 0, "unspent_points": 0,
			"stat1": a.Stat1, "stat1_lvl": 1,
			"stat2": a.Stat2, "stat2_lvl": 1,
			"stat3": a.Stat3, "stat3_lvl": 1,
		}),
	}).Create(a).Error
	return a, err
}

func (s *Service) ApplyArtifactToBattle(userID int64, bp *battle.BattlePet, modID string) {
	a, err := s.GetArtifact(userID)
	if err != nil {
		return
	}
	statIDs := []string{a.Stat1, a.Stat2, a.Stat3}
	statLvls := []int{a.Stat1Lvl, a.Stat2Lvl, a.Stat3Lvl}

	for i, statID := range statIDs {
		level := statLvls[i]
		if level < 1 {
			level = 1
		}

		multiplier := 1.0
		if IsBoostedStat(modID, statID) {
			multiplier = 2.0
		} else if IsNerfedStat(modID, statID) {
			multiplier = 0.5
		}
		scaledLevel := int(float64(level) * multiplier)
		if scaledLevel < 1 {
			scaledLevel = 1
		}

		def := GetArtifactStat(statID)
		if def != nil {
			def.PerLevelFn(bp, scaledLevel)
		}
	}
}

func (s *Service) ArtifactStatNames(a *model.UserPetArtifact) []string {
	return []string{a.Stat1, a.Stat2, a.Stat3}
}

func (s *Service) ArtifactStatLevels(a *model.UserPetArtifact) []int {
	return []int{a.Stat1Lvl, a.Stat2Lvl, a.Stat3Lvl}
}

func (s *Service) ArtifactSortedStatSlots(a *model.UserPetArtifact) []int {
	type slot struct {
		idx int
		lvl int
	}
	slots := make([]slot, 3)
	for i, lvl := range s.ArtifactStatLevels(a) {
		slots[i] = slot{idx: i, lvl: lvl}
	}
	sort.Slice(slots, func(i, j int) bool {
		if slots[i].lvl != slots[j].lvl {
			return slots[i].lvl > slots[j].lvl
		}
		return slots[i].idx < slots[j].idx
	})
	result := make([]int, 3)
	for i, s := range slots {
		result[i] = s.idx
	}
	return result
}
