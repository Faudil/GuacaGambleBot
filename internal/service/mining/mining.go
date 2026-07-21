package mining

import (
	"errors"
	"math/rand"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type MineItem struct {
	Name  string
	Value int
}

var DepthLoot = [][]MineItem{
	{},
	{{"Caillou", 1}, {"Charbon", 5}},
	{{"Charbon", 5}, {"Minerai de Fer", 10}, {"Minerai de Cuivre", 15}},
	{{"Minerai de Cuivre", 15}, {"Minerai de Fer", 10}, {"Pépite d'Or", 50}},
	{{"Minerai de Cuivre", 15}, {"Minerai d'argent", 25}, {"Pépite d'Or", 50}},
	{{"Minerai d'argent", 25}, {"Pépite d'Or", 50}},
	{{"Minerai d'argent", 25}, {"Pépite d'Or", 50}},
	{{"Pépite d'Or", 50}, {"Platine", 75}, {"Emeraude", 100}},
	{{"Pépite d'Or", 50}, {"Platine", 75}, {"Emeraude", 100}},
	{{"Pépite d'Or", 50}, {"Platine", 75}, {"Emeraude", 100}},
	{{"Platine", 75}, {"Emeraude", 100}, {"Diamant Brut", 300}},
}

type BagEntry struct {
	Name  string
	Count int
}

type DescendResult struct {
	Item      *MineItem
	Collapsed bool
	Bag       []BagEntry
}

type LeaveResult struct {
	XP     int
	Bag    []BagEntry
	Unlocks []*achievement.Achievement
}

var (
	ErrCollapsed = errors.New("mine collapsed")
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) Descend(userID int64, depth int, bag []BagEntry, riskReduc int) (*DescendResult, error) {
	risk := (depth - 1) * 5
	risk -= riskReduc
	if risk < 0 {
		risk = 0
	}

	roll := rand.Intn(100) + 1
	if roll <= risk {
		return &DescendResult{Collapsed: true, Bag: bag}, nil
	}

	lvl := depth
	if lvl >= len(DepthLoot) {
		lvl = len(DepthLoot) - 1
	}
	pool := DepthLoot[lvl]
	item := pool[rand.Intn(len(pool))]

	found := false
	for i, e := range bag {
		if e.Name == item.Name {
			bag[i].Count++
			found = true
			break
		}
	}
	if !found {
		bag = append(bag, BagEntry{Name: item.Name, Count: 1})
	}

	return &DescendResult{Item: &item, Bag: bag}, nil
}

func (s *Service) LeaveMine(userID int64, bag []BagEntry) (*LeaveResult, error) {
	totalXP := len(bag) * 10
	for _, e := range bag {
		totalXP += e.Count * 5
	}

	for _, e := range bag {
		var dbItem model.Item
		if err := s.store.DB.Where("name = ?", e.Name).First(&dbItem).Error; err != nil {
			return nil, err
		}
		if err := s.store.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
			DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + ?", e.Count)}),
		}).Create(&model.Inventory{UserID: userID, ItemID: dbItem.ID, Quantity: e.Count}).Error; err != nil {
			return nil, err
		}
	}

	if len(bag) > 0 {
		if err := achievement.IncrementStat(s.store.DB, userID, "items_mined", len(bag)); err != nil {
			return nil, err
		}
		if err := s.store.RecordActivity(userID, "items_mined", len(bag)); err != nil {
			return nil, err
		}
	}

	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "miner").First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			job = model.Job{UserID: userID, JobName: "miner", Level: 1, XP: totalXP}
			if err := s.store.DB.Create(&job).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		job.XP += totalXP
		next := jobXPForLevel(job.Level)
		if job.XP >= next {
			job.XP -= next
			job.Level++
		}
		if err := s.store.DB.Model(&model.Job{}).Where("user_id = ? AND job_name = ?", userID, "miner").
			Updates(map[string]any{"xp": job.XP, "level": job.Level}).Error; err != nil {
			return nil, err
		}
	}

	unlocks, err := achievement.CheckAndUnlock(s.store.DB, userID)
	if err != nil {
		return nil, err
	}

	return &LeaveResult{XP: totalXP, Bag: bag, Unlocks: unlocks}, nil
}

func jobXPForLevel(level int) int {
	return 50 + level*25
}
