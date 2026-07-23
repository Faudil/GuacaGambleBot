package mining

import (
	"errors"
	"math/rand"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
	"guacagamblebot/internal/store"
)

var ErrMineLimit = errors.New("mining daily limit reached")

type MineItem struct {
	Name  string
	Value int
}

var DepthLoot = [][]MineItem{
	{},
	{{"pebble", 1}, {"coal", 5}},
	{{"coal", 5}, {"iron_ore", 10}, {"copper_ore", 15}},
	{{"copper_ore", 15}, {"iron_ore", 10}, {"gold_nugget", 50}},
	{{"copper_ore", 15}, {"silver_ore", 25}, {"gold_nugget", 50}},
	{{"silver_ore", 25}, {"gold_nugget", 50}},
	{{"silver_ore", 25}, {"gold_nugget", 50}},
	{{"gold_nugget", 50}, {"platinum", 75}, {"emerald", 100}},
	{{"gold_nugget", 50}, {"platinum", 75}, {"emerald", 100}},
	{{"gold_nugget", 50}, {"platinum", 75}, {"emerald", 100}},
	{{"platinum", 75}, {"emerald", 100}, {"rough_diamond", 300}},
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
	ok, _, err := s.store.CheckGameLimit(userID, "mine_descend", 50)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrMineLimit
	}

	risk := (depth - 1) * 5
	risk -= riskReduc

	vitReduc := int(charsvc.GetVITReduction(s.store, userID) * 100)
	risk -= vitReduc

	if charsvc.HasBuff(s.store, userID, "reinforce") {
		risk = 0
		charsvc.ConsumeBuff(s.store, userID, "reinforce")
	}

	if risk < 0 {
		risk = 0
	}

	_ = s.store.IncrementGameLimit(userID, "mine_descend")

	roll := rand.Intn(100) + 1
	if roll <= risk {
		return &DescendResult{Collapsed: true, Bag: bag}, nil
	}

	lvl := depth
	if lvl >= len(DepthLoot) {
		lvl = len(DepthLoot) - 1
	}
	pool := DepthLoot[lvl]

	if charsvc.HasBuff(s.store, userID, "midas_touch") {
		var filtered []MineItem
		for _, it := range pool {
			if it.Value >= 50 {
				filtered = append(filtered, it)
			}
		}
		if len(filtered) > 0 {
			pool = filtered
			charsvc.ConsumeBuff(s.store, userID, "midas_touch")
		}
	} else if charsvc.HasBuff(s.store, userID, "nose_for_treasure") {
		var filtered []MineItem
		for _, it := range pool {
			if it.Value >= 100 {
				filtered = append(filtered, it)
			}
		}
		if len(filtered) > 0 {
			pool = filtered
		}
		// always consume, even if pool empty — buff is used up
		charsvc.ConsumeBuff(s.store, userID, "nose_for_treasure")
	}

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
	strMult := charsvc.GetSTRBonus(s.store, userID)
	if strMult > 1.0 {
		for i := range bag {
			bag[i].Count = int(float64(bag[i].Count) * strMult)
		}
	}

	if charsvc.HasBuff(s.store, userID, "scavenger") {
		for i := range bag {
			bag[i].Count += bag[i].Count / 2
		}
		charsvc.ConsumeBuff(s.store, userID, "scavenger")
	}

	totalXP := len(bag) * 10
	for _, e := range bag {
		totalXP += e.Count * 5
	}

	for _, e := range bag {
		if err := s.store.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
			DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + ?", e.Count)}),
		}).Create(&model.Inventory{UserID: userID, ItemID: e.Name, Quantity: e.Count}).Error; err != nil {
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

	charsvc.AddXP(s.store, userID, totalXP)

	unlocks, err := achievement.CheckAndUnlock(s.store.DB, userID)
	if err != nil {
		return nil, err
	}

	return &LeaveResult{XP: totalXP, Bag: bag, Unlocks: unlocks}, nil
}

func jobXPForLevel(level int) int {
	return 50 + level*25
}
