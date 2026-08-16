package community

import (
	"errors"
	"math"
	"sort"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

var (
	// ErrBuildingNotFound is returned when the requested project does not exist.
	ErrBuildingNotFound = errors.New("building not found")
	// ErrMaxLevel is returned when the project is already at its maximum level.
	ErrMaxLevel = errors.New("building at max level")
	// ErrResourceNotNeeded is returned when the project does not need the resource.
	ErrResourceNotNeeded = errors.New("resource not needed")
	// ErrResourceFull is returned when the quota for the resource is already met.
	ErrResourceFull = errors.New("resource quota already met")
	// ErrNotEnoughMoney is returned when the player cannot afford the contribution.
	ErrNotEnoughMoney = errors.New("not enough money")
	// ErrNotEnoughItems is returned when the player does not own enough of the item.
	ErrNotEnoughItems = errors.New("not enough items")
	// ErrInvalidAmount is returned when the requested contribution is <= 0.
	ErrInvalidAmount = errors.New("invalid amount")
)

// MoneyKey is the pseudo-resource used for coin contributions.
const MoneyKey = "money"

// BuildingOrder lists projects in their display order.
var BuildingOrder = []string{"market", "bank", "hospital", "statue"}

// BuildingDef defines a community project: its costs (keyed by canonical item
// ID, or MoneyKey for coins) and the bonuses it grants once leveled.
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
				return map[string]int{"money": 10000, "pebble": 200}
			case 2:
				return map[string]int{"money": 50000, "pebble": 1000, "oat": 200}
			case 3:
				return map[string]int{"money": 150000, "oat": 500, "primordial_geode": 500}
			case 4:
				return map[string]int{"money": 500000, "primordial_geode": 1500, "iron_ore": 500}
			case 5:
				return map[string]int{"money": 1000000, "iron_ore": 2000, "gold_nugget": 200}
			case 6:
				return map[string]int{"money": 3000000, "gold_nugget": 1000, "rough_diamond": 50}
			case 7:
				return map[string]int{"money": 5000000, "rough_diamond": 200, "kethari_crystal": 200}
			case 8:
				return map[string]int{"money": 8000000, "kethari_crystal": 500, "emerald": 500}
			case 9:
				return map[string]int{"money": 15000000, "emerald": 1000, "purified_relic": 50}
			case 10:
				return map[string]int{"money": 30000000, "purified_relic": 200, "resonance_core": 10}
			}
			return map[string]int{"money": 10000, "pebble": 200}
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
				return map[string]int{"money": 15000, "coal": 100}
			case 2:
				return map[string]int{"money": 60000, "iron_ore": 100}
			case 3:
				return map[string]int{"money": 200000, "gold_nugget": 100}
			case 4:
				return map[string]int{"money": 700000, "gold_nugget": 500}
			case 5:
				return map[string]int{"money": 2000000, "rough_diamond": 100}
			case 6:
				return map[string]int{"money": 5000000, "kethari_crystal": 100}
			case 7:
				return map[string]int{"money": 10000000, "emerald": 100}
			case 8:
				return map[string]int{"money": 20000000, "platinum": 100}
			case 9:
				return map[string]int{"money": 50000000, "star_fruit": 10}
			case 10:
				return map[string]int{"money": 100000000, "star_fruit": 50}
			}
			return map[string]int{"money": 15000, "coal": 100}
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
			return map[string]int{"money": baseMoney, "pebble": basePebble}
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
				return map[string]int{"money": 8000, "wheat": 200, "carrot": 100}
			case 2:
				return map[string]int{"money": 40000, "potato": 300, "tomato": 200}
			case 3:
				return map[string]int{"money": 150000, "strawberry": 400, "golden_apple": 50}
			case 4:
				return map[string]int{"money": 500000, "golden_apple": 200, "golden_potato": 100}
			case 5:
				return map[string]int{"money": 1500000, "golden_potato": 300, "blood_tomato": 150}
			case 6:
				return map[string]int{"money": 4000000, "golden_carrot": 150, "golden_apple": 400}
			case 7:
				return map[string]int{"money": 10000000, "golden_carrot": 300, "blood_tomato": 300}
			case 8:
				return map[string]int{"money": 25000000, "golden_carrot": 500, "ghost_wheat": 500}
			case 9:
				return map[string]int{"money": 50000000, "ghost_wheat": 800, "golden_carrot": 600}
			case 10:
				return map[string]int{"money": 100000000, "ghost_wheat": 1500, "golden_carrot": 1000}
			}
			return map[string]int{"money": 8000, "wheat": 200, "carrot": 100}
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

// ResourceProgress tracks how much of a resource has been contributed towards
// the next level of a project.
type ResourceProgress struct {
	Resource    string // canonical item ID, or MoneyKey for coins
	Contributed int
	Required    int
}

// Full reports whether the resource quota for the next level is met.
func (p ResourceProgress) Full() bool {
	return p.Required > 0 && p.Contributed >= p.Required
}

// BuildingInfo is the read model of a project used by the UI.
type BuildingInfo struct {
	Key      string
	Level    int
	MaxLevel int
	Costs    map[string]int
	Bonuses  map[string]any
	Progress []ResourceProgress
}

// InvestResult reports what a successful investment accomplished.
type InvestResult struct {
	Invested  int  // amount actually contributed (capped at the remaining quota)
	LeveledUp bool // whether this contribution completed the level
	NewLevel  int
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

// GetAllProjects returns every project in display order, each with its current
// level, next-level costs and live contribution progress.
func (s *Service) GetAllProjects(serverID int64) ([]BuildingInfo, error) {
	var contribs []model.ServerProjectContribution
	s.store.DB.Where("server_id = ?", serverID).Find(&contribs)
	byProject := map[string]map[string]int{}
	for _, c := range contribs {
		m := byProject[c.ProjectID]
		if m == nil {
			m = map[string]int{}
			byProject[c.ProjectID] = m
		}
		m[c.ResourceType] = c.AmountContributed
	}

	out := make([]BuildingInfo, 0, len(BuildingOrder))
	for _, key := range BuildingOrder {
		b, ok := Buildings[key]
		if !ok {
			continue
		}
		lvl, _ := s.GetProjectLevel(serverID, key)
		costs := b.CostFunc(lvl + 1)
		progress := make([]ResourceProgress, 0, len(costs))
		for res, required := range costs {
			progress = append(progress, ResourceProgress{
				Resource: res, Contributed: byProject[key][res], Required: required,
			})
		}
		sort.Slice(progress, func(i, j int) bool {
			if progress[i].Resource == MoneyKey {
				return true
			}
			if progress[j].Resource == MoneyKey {
				return false
			}
			return progress[i].Resource < progress[j].Resource
		})
		out = append(out, BuildingInfo{
			Key: key, Level: lvl, MaxLevel: b.MaxLevel,
			Costs: costs, Bonuses: b.BonusFunc(lvl), Progress: progress,
		})
	}
	return out, nil
}

// GetProjectInfo returns the full read model for a single project, or nil when
// the project does not exist.
func (s *Service) GetProjectInfo(serverID int64, projectID string) (*BuildingInfo, error) {
	all, err := s.GetAllProjects(serverID)
	if err != nil {
		return nil, err
	}
	for i := range all {
		if all[i].Key == projectID {
			return &all[i], nil
		}
	}
	return nil, nil
}

// Invest contributes amount of resource resKey to buildingName. Item and coin
// contributions are deducted from the player atomically with the contribution
// and stat bookkeeping. It returns the amount actually invested and whether the
// contribution completed the level.
func (s *Service) Invest(serverID, userID int64, buildingName, resKey string, amount int) (*InvestResult, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	b, ok := Buildings[buildingName]
	if !ok {
		return nil, ErrBuildingNotFound
	}
	currentLevel, _ := s.GetProjectLevel(serverID, buildingName)
	if currentLevel >= b.MaxLevel {
		return nil, ErrMaxLevel
	}
	targetLevel := currentLevel + 1
	costs := b.CostFunc(targetLevel)
	required, ok := costs[resKey]
	if !ok {
		return nil, ErrResourceNotNeeded
	}

	var contrib model.ServerProjectContribution
	if err := s.store.DB.Where("server_id = ? AND project_id = ? AND resource_type = ?",
		serverID, buildingName, resKey).First(&contrib).Error; err == nil {
		if contrib.AmountContributed >= required {
			return nil, ErrResourceFull
		}
	}
	current := contrib.AmountContributed
	investAmount := amount
	if current+investAmount > required {
		investAmount = required - current
	}

	res := &InvestResult{Invested: investAmount, NewLevel: targetLevel}
	err := s.store.DB.Transaction(func(tx *gorm.DB) error {
		if resKey == MoneyKey {
			var bal int
			if err := tx.Model(&model.User{}).
				Where("user_id = ?", userID).Pluck("balance", &bal).Error; err != nil {
				return err
			}
			if bal < investAmount {
				return ErrNotEnoughMoney
			}
			if err := s.store.UpdateBalanceTx(tx, userID, -investAmount); err != nil {
				return err
			}
		} else {
			has, err := hasItemTx(tx, userID, resKey, investAmount)
			if err != nil {
				return err
			}
			if !has {
				return ErrNotEnoughItems
			}
			if err := tx.Model(&model.Inventory{}).
				Where("user_id = ? AND item_id = ?", userID, resKey).
				UpdateColumn("quantity", gorm.Expr("quantity - ?", investAmount)).Error; err != nil {
				return err
			}
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "server_id"}, {Name: "project_id"}, {Name: "resource_type"}},
			DoUpdates: clause.Assignments(map[string]any{"amount_contributed": gorm.Expr("amount_contributed + ?", investAmount)}),
		}).Create(&model.ServerProjectContribution{
			ServerID: serverID, ProjectID: buildingName, ResourceType: resKey, AmountContributed: investAmount,
		}).Error; err != nil {
			return err
		}

		var ucs model.UserCommunityStat
		tx.Where("user_id = ? AND server_id = ?", userID, serverID).First(&ucs)
		ucs.UserID = userID
		ucs.ServerID = serverID
		if resKey == MoneyKey {
			ucs.TotalMoneyInvested += investAmount
		} else {
			ucs.TotalItemsInvested += investAmount
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "server_id"}},
			DoUpdates: clause.Assignments(map[string]any{"total_money_invested": ucs.TotalMoneyInvested, "total_items_invested": ucs.TotalItemsInvested}),
		}).Create(&ucs).Error; err != nil {
			return err
		}

		var contributions []model.ServerProjectContribution
		if err := tx.Where("server_id = ? AND project_id = ?", serverID, buildingName).Find(&contributions).Error; err != nil {
			return err
		}
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
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "server_id"}, {Name: "project_id"}},
				DoUpdates: clause.Assignments(map[string]any{"level": targetLevel}),
			}).Create(&model.ServerProject{ServerID: serverID, ProjectID: buildingName, Level: targetLevel}).Error; err != nil {
				return err
			}
			if err := tx.Where("server_id = ? AND project_id = ?", serverID, buildingName).Delete(&model.ServerProjectContribution{}).Error; err != nil {
				return err
			}
			res.LeveledUp = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func hasItemTx(tx *gorm.DB, userID int64, itemID string, qty int) (bool, error) {
	var inv model.Inventory
	err := tx.Where("user_id = ? AND item_id = ?", userID, itemID).First(&inv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return inv.Quantity >= qty, nil
}

func (s *Service) GetUserStats(userID, serverID int64) (*model.UserCommunityStat, error) {
	var ucs model.UserCommunityStat
	err := s.store.DB.Where("user_id = ? AND server_id = ?", userID, serverID).First(&ucs).Error
	if err != nil {
		return &model.UserCommunityStat{UserID: userID, ServerID: serverID}, nil
	}
	return &ucs, nil
}

// TopContributor is a single entry of the per-server contribution leaderboard.
type TopContributor struct {
	UserID int64
	Total  int
}

// GetTopContributors returns the limit users with the highest total
// contribution (money + items) on the server, most valuable first.
func (s *Service) GetTopContributors(serverID int64, limit int) ([]TopContributor, error) {
	var rows []TopContributor
	err := s.store.DB.Model(&model.UserCommunityStat{}).
		Select("user_id, total_money_invested + total_items_invested AS total").
		Where("server_id = ?", serverID).
		Where("total_money_invested + total_items_invested > 0").
		Order("total_money_invested + total_items_invested DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}
