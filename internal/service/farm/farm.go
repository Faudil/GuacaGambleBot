package farm

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type Crop struct {
	Name          string
	Value         int
	GrowTimeSec   int
	SeedName      string
}

type Seed struct {
	Name        string
	Price       int
	Crop        Crop
	GrowTimeSec int
}

var Crops = []Crop{
	{Name: "Blé", Value: 5, GrowTimeSec: 300, SeedName: "Graine de Blé"},
	{Name: "Avoine", Value: 8, GrowTimeSec: 600, SeedName: "Graine d'Avoine"},
	{Name: "Maïs", Value: 12, GrowTimeSec: 1800, SeedName: "Graine de Maïs"},
	{Name: "Patate", Value: 20, GrowTimeSec: 3600, SeedName: "Graine de Patate"},
	{Name: "Tomate", Value: 25, GrowTimeSec: 7200, SeedName: "Graine de Tomate"},
	{Name: "Citrouille", Value: 40, GrowTimeSec: 14400, SeedName: "Graine de Citrouille"},
	{Name: "Grain de Café", Value: 60, GrowTimeSec: 28800, SeedName: "Graine de Café"},
	{Name: "Fève de Cacao", Value: 75, GrowTimeSec: 43200, SeedName: "Graine de Cacao"},
	{Name: "Fraise", Value: 90, GrowTimeSec: 64800, SeedName: "Graine de Fraise"},
	{Name: "Pomme Dorée", Value: 150, GrowTimeSec: 86400, SeedName: "Pépin de Pomme Dorée"},
	{Name: "Fruit Étoile", Value: 250, GrowTimeSec: 172800, SeedName: "Pépin de Fruit Étoile"},
}

var Seeds = []Seed{
	{Name: "Graine de Blé", Price: 2, GrowTimeSec: 300, Crop: cropByName("Blé")},
	{Name: "Graine d'Avoine", Price: 3, GrowTimeSec: 600, Crop: cropByName("Avoine")},
	{Name: "Graine de Maïs", Price: 5, GrowTimeSec: 1800, Crop: cropByName("Maïs")},
	{Name: "Graine de Patate", Price: 8, GrowTimeSec: 3600, Crop: cropByName("Patate")},
	{Name: "Graine de Tomate", Price: 10, GrowTimeSec: 7200, Crop: cropByName("Tomate")},
	{Name: "Graine de Citrouille", Price: 15, GrowTimeSec: 14400, Crop: cropByName("Citrouille")},
	{Name: "Graine de Café", Price: 25, GrowTimeSec: 28800, Crop: cropByName("Grain de Café")},
	{Name: "Graine de Cacao", Price: 30, GrowTimeSec: 43200, Crop: cropByName("Fève de Cacao")},
	{Name: "Graine de Fraise", Price: 40, GrowTimeSec: 64800, Crop: cropByName("Fraise")},
	{Name: "Pépin de Pomme Dorée", Price: 75, GrowTimeSec: 86400, Crop: cropByName("Pomme Dorée")},
	{Name: "Pépin de Fruit Étoile", Price: 125, GrowTimeSec: 172800, Crop: cropByName("Fruit Étoile")},
}

func cropByName(name string) Crop {
	for _, c := range Crops {
		if c.Name == name {
			return c
		}
	}
	return Crop{}
}

type PlotInfo struct {
	PlotIndex int
	ItemName  string
	PlantTime time.Time
	GrowTime  int
	Ready     bool
	Progress  int
}

type HarvestResult struct {
	CropName string
	Quantity int
	XP       int
}

var (
	ErrNoSeed   = errors.New("you don't have that seed")
	ErrOccupied = errors.New("plot is occupied")
	ErrNotReady = errors.New("crop is not ready yet")
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) GetPlots(userID int64, zoneKey string) ([]PlotInfo, error) {
	var plots []model.UserFarming
	if err := s.store.DB.Where("user_id = ? AND zone_key = ?", userID, zoneKey).Find(&plots).Error; err != nil {
		return nil, err
	}
	plotMap := make(map[int]model.UserFarming)
	for _, p := range plots {
		plotMap[p.PlotIndex] = p
	}
	var out []PlotInfo
	for i := 0; i < 3; i++ {
		p, ok := plotMap[i]
		if !ok {
			out = append(out, PlotInfo{PlotIndex: i})
			continue
		}
		elapsed := time.Since(p.PlantTime).Seconds()
		ready := elapsed >= float64(p.GrowTime)
		progress := 0
		if p.GrowTime > 0 {
			progress = int((elapsed / float64(p.GrowTime)) * 100)
			if progress > 100 {
				progress = 100
			}
		}
		out = append(out, PlotInfo{
			PlotIndex: p.PlotIndex,
			ItemName:  p.ItemName,
			PlantTime: p.PlantTime,
			GrowTime:  p.GrowTime,
			Ready:     ready,
			Progress:  progress,
		})
	}
	return out, nil
}

func (s *Service) Plant(userID int64, zoneKey string, plotIndex int, seedName string, growTime int) error {
	var existing model.UserFarming
	if err := s.store.DB.Where("user_id = ? AND zone_key = ? AND plot_index = ?", userID, zoneKey, plotIndex).First(&existing).Error; err == nil {
		return ErrOccupied
	}

	var dbItem model.Item
	if err := s.store.DB.Where("name = ?", seedName).First(&dbItem).Error; err != nil {
		return ErrNoSeed
	}
	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, dbItem.ID).First(&inv).Error; err != nil {
		return ErrNoSeed
	}
	if inv.Quantity < 1 {
		return ErrNoSeed
	}

	if inv.Quantity <= 1 {
		s.store.DB.Delete(&inv)
	} else {
		s.store.DB.Model(&inv).UpdateColumn("quantity", gorm.Expr("quantity - 1"))
	}

	level := s.getFarmerLevel(userID)
	reduction := float64(level) * 0.01
	if reduction > 0.5 {
		reduction = 0.5
	}
	finalGrowTime := int(float64(growTime) * (1 - reduction))

	if err := s.store.DB.Create(&model.UserFarming{
		UserID:    userID,
		ZoneKey:   zoneKey,
		PlotIndex: plotIndex,
		ItemName:  seedName,
		PlantTime: time.Now(),
		GrowTime:  finalGrowTime,
	}).Error; err != nil {
		return err
	}
	return nil
}

func (s *Service) Harvest(userID int64, zoneKey string, plotIndex int) (*HarvestResult, error) {
	var plot model.UserFarming
	if err := s.store.DB.Where("user_id = ? AND zone_key = ? AND plot_index = ?", userID, zoneKey, plotIndex).First(&plot).Error; err != nil {
		return nil, fmt.Errorf("no crop found")
	}

	elapsed := time.Since(plot.PlantTime).Seconds()
	if elapsed < float64(plot.GrowTime) {
		return nil, ErrNotReady
	}

	crop := cropBySeedName(plot.ItemName)
	if crop.Name == "" {
		return nil, fmt.Errorf("unknown crop")
	}

	level := s.getFarmerLevel(userID)
	quantity := 1
	if rand.Float64() < float64(level)*0.02 {
		quantity++
	}

	var dbItem model.Item
	if err := s.store.DB.Where("name = ?", crop.Name).First(&dbItem).Error; err != nil {
		return nil, err
	}
	if err := s.store.DB.Model(&model.Inventory{}).
		Where("user_id = ? AND item_id = ?", userID, dbItem.ID).
		UpdateColumn("quantity", gorm.Expr("quantity + ?", quantity)).Error; err != nil {
		return nil, err
	}

	xp := (crop.Value * 8 / 10) + 10
	xp *= quantity

	if err := achievement.IncrementStat(s.store.DB, userID, "items_farmed", quantity); err != nil {
		return nil, err
	}
	if err := s.store.RecordActivity(userID, "items_farmed", quantity); err != nil {
		return nil, err
	}

	s.store.DB.Delete(&plot)

	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "farmer").First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			job = model.Job{UserID: userID, JobName: "farmer", Level: 1, XP: xp}
			s.store.DB.Create(&job)
		}
	} else {
		job.XP += xp
		next := 50 + job.Level*25
		if job.XP >= next {
			job.XP -= next
			job.Level++
		}
		s.store.DB.Model(&model.Job{}).Where("user_id = ? AND job_name = ?", userID, "farmer").
			Updates(map[string]any{"xp": job.XP, "level": job.Level})
	}

	return &HarvestResult{CropName: crop.Name, Quantity: quantity, XP: xp}, nil
}

func (s *Service) getFarmerLevel(userID int64) int {
	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "farmer").First(&job).Error; err != nil {
		return 0
	}
	return job.Level
}

func cropBySeedName(seedName string) Crop {
	for _, s := range Seeds {
		if s.Name == seedName {
			return s.Crop
		}
	}
	return Crop{}
}
