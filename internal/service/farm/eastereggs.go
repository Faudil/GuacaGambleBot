package farm

import (
	"math/rand"
	"time"

	"guacagamblebot/internal/model"
)

type Mutation struct {
	BaseCrop   string
	MutatedID  string
	Chance     float64
	Multiplier int
	FlavorKey  string
}

var Mutations = []Mutation{
	{BaseCrop: "wheat", MutatedID: "ghost_wheat", Chance: 0.015, Multiplier: 5, FlavorKey: "farm.mutation_ghost_wheat"},
	{BaseCrop: "corn", MutatedID: "prismatic_corn", Chance: 0.012, Multiplier: 4, FlavorKey: "farm.mutation_prismatic_corn"},
	{BaseCrop: "potato", MutatedID: "golden_potato", Chance: 0.010, Multiplier: 8, FlavorKey: "farm.mutation_golden_potato"},
	{BaseCrop: "tomato", MutatedID: "blood_tomato", Chance: 0.010, Multiplier: 6, FlavorKey: "farm.mutation_blood_tomato"},
	{BaseCrop: "pumpkin", MutatedID: "cursed_pumpkin", Chance: 0.008, Multiplier: 10, FlavorKey: "farm.mutation_cursed_pumpkin"},
	{BaseCrop: "star_fruit", MutatedID: "nova_fruit", Chance: 0.003, Multiplier: 20, FlavorKey: "farm.mutation_nova_fruit"},
}

func (s *Service) CheckMutation(userID int64, cropName string) string {
	level := s.getFarmerLevel(userID)
	for _, m := range Mutations {
		if m.BaseCrop != cropName {
			continue
		}
		chance := m.Chance + float64(level)*0.001
		if chance > 0.10 {
			chance = 0.10
		}
		if rand.Float64() < chance {
			return m.MutatedID
		}
	}
	return ""
}

func (s *Service) GetMutationFlavor(cropID string) string {
	for _, m := range Mutations {
		if m.MutatedID == cropID {
			return m.FlavorKey
		}
	}
	return ""
}

func (s *Service) CheckGoldenCarrot(userID int64) bool {
	level := s.getFarmerLevel(userID)
	if level < 5 {
		return false
	}
	hour := time.Now().UTC().Hour()
	if hour < 4 || hour >= 6 {
		return false
	}
	return true
}

type MysteryCrop struct {
	CropID      string
	Weight      int
	GrowTimeSec int
}

var mysteryPool = []MysteryCrop{
	{CropID: "wheat", Weight: 20, GrowTimeSec: 300},
	{CropID: "corn", Weight: 15, GrowTimeSec: 1800},
	{CropID: "carrot", Weight: 15, GrowTimeSec: 900},
	{CropID: "potato", Weight: 12, GrowTimeSec: 3600},
	{CropID: "tomato", Weight: 10, GrowTimeSec: 7200},
	{CropID: "pumpkin", Weight: 8, GrowTimeSec: 14400},
	{CropID: "coffee_bean", Weight: 5, GrowTimeSec: 28800},
	{CropID: "cocoa_bean", Weight: 4, GrowTimeSec: 43200},
	{CropID: "strawberry", Weight: 3, GrowTimeSec: 64800},
	{CropID: "golden_apple", Weight: 2, GrowTimeSec: 86400},
	{CropID: "star_fruit", Weight: 1, GrowTimeSec: 172800},
	{CropID: "ghost_wheat", Weight: 2, GrowTimeSec: 600},
	{CropID: "golden_potato", Weight: 1, GrowTimeSec: 7200},
	{CropID: "cursed_pumpkin", Weight: 1, GrowTimeSec: 28800},
	{CropID: "nova_fruit", Weight: 1, GrowTimeSec: 345600},
}

func (s *Service) RollMysteriousSeed(userID int64) bool {
	level := s.getFarmerLevel(userID)
	chance := 0.03 + float64(level)*0.002
	if chance > 0.12 {
		chance = 0.12
	}
	return rand.Float64() < chance
}

func (s *Service) resolveMystery(seedName string) string {
	_ = seedName
	totalWeight := 0
	for _, m := range mysteryPool {
		totalWeight += m.Weight
	}
	roll := rand.Intn(totalWeight)
	cumulative := 0
	for _, m := range mysteryPool {
		cumulative += m.Weight
		if roll < cumulative {
			return m.CropID
		}
	}
	return "wheat"
}

func (s *Service) GetScarecrowChance() float64 {
	return 0.01
}

type CropExpertise struct {
	CropName  string
	Harvested int
	Title     string
}

func (s *Service) GetExpertise(userID int64) []CropExpertise {
	var harvests []model.UserCropHarvest
	s.store.DB.Where("user_id = ?", userID).Find(&harvests)
	harvestMap := make(map[string]int)
	for _, h := range harvests {
		harvestMap[h.CropName] = h.Count
	}
	var results []CropExpertise
	for _, c := range Crops {
		val := harvestMap[c.Name]
		if val == 0 {
			continue
		}
		title := ""
		if val >= 100 {
			title = "farm.legend_title"
		} else if val >= 50 {
			title = "farm.master_title"
		} else if val >= 10 {
			title = "farm.expert_title"
		}
		results = append(results, CropExpertise{
			CropName:  c.Name,
			Harvested: val,
			Title:     title,
		})
	}
	return results
}
