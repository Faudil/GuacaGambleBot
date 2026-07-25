package npcs

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	invsvc "guacagamblebot/internal/service/inventory"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
)

type DailyRepCap struct {
	Flat      int
	PerLevel  int
}

func GetDailyRepCap() DailyRepCap {
	return DailyRepCap{Flat: 500, PerLevel: 0}
}

type NPCInfo struct {
	ID          string
	Name        string
	Emoji       string
	Color       int
	Level       int
	Reputation  int
	NextLevel   int
	RankName    string
	Data        *universe.NPCData
}

type Service struct {
	store    *store.Store
	cfg      *config.Config
	universe *universe.Definition
	inv      *invsvc.Service
}

func New(s *store.Store, cfg *config.Config, def *universe.Definition, inv *invsvc.Service) *Service {
	return &Service{store: s, cfg: cfg, universe: def, inv: inv}
}

func (s *Service) GetNPCData(id string) *universe.NPCData {
	return s.universe.NPCs[id]
}

func (s *Service) GetAllNPCMeta() []*universe.NPCData {
	out := make([]*universe.NPCData, 0, len(s.universe.NPCs))
	for _, n := range s.universe.NPCs {
		out = append(out, n)
	}
	return out
}

func (s *Service) GetReputation(userID int64, npcID string) (*model.UserNPCReputation, error) {
	var r model.UserNPCReputation
	if err := s.store.DB.Where("user_id = ? AND npc_id = ?", userID, npcID).First(&r).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r = model.UserNPCReputation{UserID: userID, NPCID: npcID}
			if err := s.store.DB.Create(&r).Error; err != nil {
				return nil, err
			}
			return &r, nil
		}
		return nil, err
	}
	return &r, nil
}

func (s *Service) GetAllReputations(userID int64) ([]model.UserNPCReputation, error) {
	var reps []model.UserNPCReputation
	if err := s.store.DB.Where("user_id = ?", userID).Find(&reps).Error; err != nil {
		return nil, err
	}
	return reps, nil
}

func (s *Service) AddReputation(userID int64, npcID string, points int) (int, error) {
	cap := GetDailyRepCap()
	today := time.Now().Format("2006-01-02")
	var daily model.UserNPCDailyRep
	err := s.store.DB.Where("user_id = ? AND npc_id = ? AND date_str = ?", userID, npcID, today).First(&daily).Error
	if err != nil {
		daily = model.UserNPCDailyRep{UserID: userID, NPCID: npcID, DateStr: today, Amount: 0}
	}
	remaining := cap.Flat - daily.Amount
	if remaining <= 0 {
		return 0, nil
	}
	if points > remaining {
		points = remaining
	}
	if err := s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "npc_id"}, {Name: "date_str"}},
		DoUpdates: clause.Assignments(map[string]any{"amount": gorm.Expr("amount + ?", points)}),
	}).Create(&model.UserNPCDailyRep{
		UserID: userID, NPCID: npcID, DateStr: today, Amount: points,
	}).Error; err != nil {
		return 0, err
	}
	var rep model.UserNPCReputation
	if err := s.store.DB.Where("user_id = ? AND npc_id = ?", userID, npcID).First(&rep).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			rep = model.UserNPCReputation{UserID: userID, NPCID: npcID}
			if err := s.store.DB.Create(&rep).Error; err != nil {
				return 0, err
			}
		} else {
			return 0, err
		}
	}
	newRep := rep.Reputation + points
	nextLevel := rep.Level
	for newRep >= 100*nextLevel {
		newRep -= 100 * nextLevel
		nextLevel++
	}
	if err := s.store.DB.Model(&model.UserNPCReputation{}).
		Where("user_id = ? AND npc_id = ?", userID, npcID).
		Updates(map[string]any{"reputation": newRep, "level": nextLevel}).Error; err != nil {
		return 0, err
	}
	return points, nil
}

func (s *Service) RankUp(userID int64, npcID string) error {
	rep, err := s.GetReputation(userID, npcID)
	if err != nil {
		return err
	}
	if rep.Reputation < 100*rep.Level {
		return fmt.Errorf("not enough reputation")
	}
	newLevel := rep.Level + 1
	return s.store.DB.Model(&model.UserNPCReputation{}).
		Where("user_id = ? AND npc_id = ?", userID, npcID).
		Updates(map[string]any{"level": newLevel, "reputation": 0}).Error
}

func RankName(level int) string {
	names := []string{"Inconnu", "Connaissance", "Associé", "Ami", "Partenaire"}
	idx := level - 1
	if idx >= len(names) {
		idx = len(names) - 1
	}
	if idx < 0 {
		idx = 0
	}
	return names[idx]
}

type NPCBonus struct {
	ShopDiscount          float64
	GamblePayout          float64
	XPBoost               float64
	MiningRiskReduction   int
	FarmingSpeedBoost     float64
	FishingTimeBonus      float64
}

func (s *Service) GetBonuses(userID int64) *NPCBonus {
	b := &NPCBonus{}
	for _, npc := range s.universe.NPCs {
		rep, err := s.GetReputation(userID, npc.ID)
		if err != nil || rep == nil {
			continue
		}
		lvl := rep.Level
		switch npc.ID {
		case "gamblebot":
			b.ShopDiscount = maxFloat(b.ShopDiscount, float64(lvl)*2.0)
		case "thorek":
			b.MiningRiskReduction = maxInt(b.MiningRiskReduction, lvl*2)
		case "elara":
			b.FarmingSpeedBoost = maxFloat(b.FarmingSpeedBoost, float64(lvl)*2.0)
		case "irian":
			b.FishingTimeBonus = maxFloat(b.FishingTimeBonus, float64(lvl)*0.5)
		}
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
