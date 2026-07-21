package fishing

import (
	"errors"
	"math/rand"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type FishItem struct {
	Name  string
	Value int
}

var PondFish = []FishItem{
	{"old_boot", 1},
	{"trout", 10},
	{"salmon", 10},
}

var RiverFish = []FishItem{
	{"salmon", 10},
	{"sardine", 15},
	{"carp", 25},
	{"pufferfish", 50},
}

var OceanFish = []FishItem{
	{"pufferfish", 50},
	{"swordfish", 150},
	{"shark", 100},
	{"whale", 300},
	{"kraken_tentacle", 500},
}

type CastResult struct {
	ItemName string
	Value    int
	XP       int
	Reaction float64
}

var (
	ErrCooldown = errors.New("on cooldown")
	ErrLimit    = errors.New("daily limit reached")
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) CheckCooldown(userID int64) (time.Duration, error) {
	var cd model.Cooldown
	err := s.store.DB.Where("user_id = ? AND activity_name = ?", userID, "fish").First(&cd).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	elapsed := time.Since(cd.LastUsed)
	cooldown := 5 * time.Minute
	if elapsed >= cooldown {
		return 0, nil
	}
	return cooldown - elapsed, nil
}

func (s *Service) SetCooldown(userID int64) error {
	return s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "activity_name"}},
		DoUpdates: clause.Assignments(map[string]any{"last_used": time.Now()}),
	}).Create(&model.Cooldown{UserID: userID, ActivityName: "fish", LastUsed: time.Now()}).Error
}

func (s *Service) CastLine(userID int64, biome string) (*CastResult, error) {
	ok, _, err := s.store.CheckGameLimit(userID, "fish", 10)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrLimit
	}

	var pool []FishItem
	switch biome {
	case "pond":
		pool = PondFish
	case "river":
		pool = RiverFish
	case "ocean":
		pool = OceanFish
	default:
		pool = PondFish
	}

	roll := rand.Float64()
	reaction := rand.Float64()*2.0 + 0.3
	isPerfect := reaction < 1.0
	var item FishItem
	if isPerfect && roll < 0.20 {
		item = pool[len(pool)-1]
	} else if roll < 0.60 {
		item = pool[rand.Intn(len(pool)-1)+1]
	} else {
		item = pool[0]
	}

	xp := 10 + rand.Intn(11)

	if err := s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
		DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + 1")}),
	}).Create(&model.Inventory{UserID: userID, ItemID: item.Name, Quantity: 1}).Error; err != nil {
		return nil, err
	}

	if err := achievement.IncrementStat(s.store.DB, userID, "items_fished", 1); err != nil {
		return nil, err
	}

	if err := s.store.RecordActivity(userID, "items_fished", 1); err != nil {
		return nil, err
	}

	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "fisher").First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			job = model.Job{UserID: userID, JobName: "fisher", Level: 1, XP: xp}
			if err := s.store.DB.Create(&job).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		job.XP += xp
		next := jobXPForLevel(job.Level)
		if job.XP >= next {
			job.XP -= next
			job.Level++
		}
		if err := s.store.DB.Model(&model.Job{}).Where("user_id = ? AND job_name = ?", userID, "fisher").
			Updates(map[string]any{"xp": job.XP, "level": job.Level}).Error; err != nil {
			return nil, err
		}
	}

	if err := s.store.IncrementGameLimit(userID, "fish"); err != nil {
		return nil, err
	}
	if err := s.SetCooldown(userID); err != nil {
		return nil, err
	}

	return &CastResult{ItemName: item.Name, Value: item.Value, XP: xp, Reaction: reaction}, nil
}

func jobXPForLevel(level int) int {
	return 50 + level*25
}
