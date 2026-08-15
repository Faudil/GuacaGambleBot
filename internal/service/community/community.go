package community

import (
	"math"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type BuildingDef struct {
	Key       string
	MaxLevel  int
	CostFunc  func(level int) map[string]int
	BonusFunc func(level int) map[string]any
}

var Buildings = map[string]*BuildingDef{
	"market": {
		Key: "market", MaxLevel: 10,
		CostFunc: func(level int) map[string]int {
			switch level {
			case 1:
				return map[string]int{"money": 10000, "Pebble": 200}
			case 2:
				return map[string]int{"money": 50000, "Pebble": 1000, "Wood": 200}
			case 3:
				return map[string]int{"money": 150000, "Wood": 500, "Stone": 500}
			case 4:
				return map[string]int{"money": 500000, "Stone": 1500, "Iron Ore": 500}
			case 5:
				return map[string]int{"money": 1000000, "Iron Ore": 2000, "Gold Ore": 200}
			case 6:
				return map[string]int{"money": 3000000, "Gold Ore": 1000, "Diamond": 50}
			case 7:
				return map[string]int{"money": 5000000, "Diamond": 200, "Ruby": 200}
			case 8:
				return map[string]int{"money": 8000000, "Ruby": 500, "Emerald": 500}
			case 9:
				return map[string]int{"money": 15000000, "Emerald": 1000, "Ancient Relic": 50}
			case 10:
				return map[string]int{"money": 30000000, "Ancient Relic": 200, "Meteorite": 10}
			}
			return map[string]int{"money": 10000, "Pebble": 200}
		},
		BonusFunc: func(level int) map[string]any {
			if level == 0 {
				return nil
			}
			discount := math.Min(20, float64(level*2))
			return map[string]any{"shop_discount": discount}
		},
	},
	"bank": {
		Key: "bank", MaxLevel: 10,
		CostFunc: func(level int) map[string]int {
			switch level {
			case 1:
				return map[string]int{"money": 15000, "Coal": 100}
			case 2:
				return map[string]int{"money": 60000, "Iron Ore": 100}
			case 3:
				return map[string]int{"money": 200000, "Gold Ore": 100}
			case 4:
				return map[string]int{"money": 700000, "Gold Ore": 500}
			case 5:
				return map[string]int{"money": 2000000, "Diamond": 100}
			case 6:
				return map[string]int{"money": 5000000, "Ruby": 100}
			case 7:
				return map[string]int{"money": 10000000, "Emerald": 100}
			case 8:
				return map[string]int{"money": 20000000, "Sapphire": 100}
			case 9:
				return map[string]int{"money": 50000000, "Star Fragment": 10}
			case 10:
				return map[string]int{"money": 100000000, "Star Fragment": 50}
			}
			return map[string]int{"money": 15000, "Coal": 100}
		},
		BonusFunc: func(level int) map[string]any {
			if level == 0 {
				return nil
			}
			payout := math.Min(50, float64(level*5))
			return map[string]any{"job_payout": payout}
		},
	},
	"statue": {
		Key: "statue", MaxLevel: 5,
		CostFunc: func(level int) map[string]int {
			baseMoney := 10000 * int(math.Pow(5, float64(level-1)))
			basePebble := 500 * level
			return map[string]int{"money": baseMoney, "Pebble": basePebble}
		},
		BonusFunc: func(level int) map[string]any {
			if level == 0 {
				return nil
			}
			return map[string]any{"glory_bonus": level * 10}
		},
	},
	"hospital": {
		Key: "hospital", MaxLevel: 10,
		CostFunc: func(level int) map[string]int {
			switch level {
			case 1:
				return map[string]int{"money": 8000, "Wheat": 200, "Carrot": 100}
			case 2:
				return map[string]int{"money": 40000, "Potato": 300, "Tomato": 200}
			case 3:
				return map[string]int{"money": 150000, "Strawberry": 400, "Golden Apple": 50}
			case 4:
				return map[string]int{"money": 500000, "Golden Apple": 200, "Golden Potato": 100}
			case 5:
				return map[string]int{"money": 1500000, "Golden Potato": 300, "Blood Tomato": 150}
			case 6:
				return map[string]int{"money": 4000000, "Golden Carrot": 150, "Golden Apple": 400}
			case 7:
				return map[string]int{"money": 10000000, "Golden Carrot": 300, "Blood Tomato": 300}
			case 8:
				return map[string]int{"money": 25000000, "Golden Carrot": 500, "Ghost Wheat": 500}
			case 9:
				return map[string]int{"money": 50000000, "Ghost Wheat": 800, "Golden Carrot": 600}
			case 10:
				return map[string]int{"money": 100000000, "Ghost Wheat": 1500, "Golden Carrot": 1000}
			}
			return map[string]int{"money": 8000, "Wheat": 200, "Carrot": 100}
		},
		BonusFunc: func(level int) map[string]any {
			if level == 0 {
				return nil
			}
			discount := math.Min(100, float64(level*10))
			return map[string]any{"heal_discount": discount}
		},
	},
}

type BuildingInfo struct {
	Key      string
	Level    int
	MaxLevel int
	Costs    map[string]int
	Bonuses  map[string]any
}

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) GetProjectLevel(serverID int64, projectID string) (int, error) {
	var p model.ServerProject
	err := s.store.DB.Where("server_id = ? AND project_id = ?", serverID, projectID).First(&p).Error
	if err != nil {
		return 0, nil
	}
	return p.Level, nil
}

func (s *Service) GetAllProjects(serverID int64) ([]BuildingInfo, error) {
	var out []BuildingInfo
	for key, b := range Buildings {
		lvl, _ := s.GetProjectLevel(serverID, key)
		bonuses := b.BonusFunc(lvl)
		costs := b.CostFunc(lvl + 1)
		out = append(out, BuildingInfo{
			Key: key, Level: lvl, MaxLevel: b.MaxLevel,
			Costs: costs, Bonuses: bonuses,
		})
	}
	return out, nil
}

func (s *Service) Invest(serverID, userID int64, buildingName, resKey string, amount int) (bool, error) {
	b, ok := Buildings[buildingName]
	if !ok {
		return false, nil
	}
	currentLevel, _ := s.GetProjectLevel(serverID, buildingName)
	if currentLevel >= b.MaxLevel {
		return false, nil
	}
	targetLevel := currentLevel + 1
	costs := b.CostFunc(targetLevel)
	required, ok := costs[resKey]
	if !ok {
		return false, nil
	}

	var contrib model.ServerProjectContribution
	err := s.store.DB.Where("server_id = ? AND project_id = ? AND resource_type = ?",
		serverID, buildingName, resKey).First(&contrib).Error
	current := 0
	if err == nil {
		current = contrib.AmountContributed
	}
	if current >= required {
		return false, nil
	}
	investAmount := amount
	if current+investAmount > required {
		investAmount = required - current
	}

	if resKey == "money" {
		bal, err := s.store.GetBalance(userID)
		if err != nil {
			return false, err
		}
		if bal < investAmount {
			return false, nil
		}
		if _, err := s.store.UpdateBalance(userID, -investAmount); err != nil {
			return false, err
		}
	}

	if err := s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "server_id"}, {Name: "project_id"}, {Name: "resource_type"}},
		DoUpdates: clause.Assignments(map[string]any{"amount_contributed": gorm.Expr("amount_contributed + ?", investAmount)}),
	}).Create(&model.ServerProjectContribution{
		ServerID: serverID, ProjectID: buildingName, ResourceType: resKey, AmountContributed: investAmount,
	}).Error; err != nil {
		return false, err
	}

	var ucs model.UserCommunityStat
	s.store.DB.Where("user_id = ? AND server_id = ?", userID, serverID).First(&ucs)
	if resKey == "money" {
		ucs.TotalMoneyInvested += investAmount
	} else {
		ucs.TotalItemsInvested += investAmount
	}
	s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "server_id"}},
		DoUpdates: clause.Assignments(map[string]any{"total_money_invested": ucs.TotalMoneyInvested, "total_items_invested": ucs.TotalItemsInvested}),
	}).Create(&ucs)

	var contributions []model.ServerProjectContribution
	s.store.DB.Where("server_id = ? AND project_id = ?", serverID, buildingName).Find(&contributions)
	allDone := true
	for res, req := range costs {
		found := false
		for _, c := range contributions {
			if c.ResourceType == res && c.AmountContributed >= req {
				found = true
				break
			}
		}
		if !found {
			allDone = false
			break
		}
	}

	if allDone {
		newLevel := currentLevel + 1
		s.store.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "server_id"}, {Name: "project_id"}},
			DoUpdates: clause.Assignments(map[string]any{"level": newLevel}),
		}).Create(&model.ServerProject{ServerID: serverID, ProjectID: buildingName, Level: newLevel})
		s.store.DB.Where("server_id = ? AND project_id = ?", serverID, buildingName).Delete(&model.ServerProjectContribution{})
		return true, nil
	}

	return false, nil
}

func (s *Service) GetUserStats(userID, serverID int64) (*model.UserCommunityStat, error) {
	var ucs model.UserCommunityStat
	err := s.store.DB.Where("user_id = ? AND server_id = ?", userID, serverID).First(&ucs).Error
	if err != nil {
		return &model.UserCommunityStat{UserID: userID, ServerID: serverID}, nil
	}
	return &ucs, nil
}
