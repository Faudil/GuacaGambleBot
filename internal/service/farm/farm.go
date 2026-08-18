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
	charsvc "guacagamblebot/internal/service/character"
	furnituresvc "guacagamblebot/internal/service/furniture"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
)

type Crop struct {
	Name        string
	Value       int
	GrowTimeSec int
	SeedName    string
}

type Seed struct {
	Name        string
	Price       int
	Crop        Crop
	GrowTimeSec int
}

var Crops = []Crop{
	{Name: "wheat", Value: 5, GrowTimeSec: 300, SeedName: "wheat_seed"},
	{Name: "oat", Value: 8, GrowTimeSec: 600, SeedName: "oat_seed"},
	{Name: "corn", Value: 12, GrowTimeSec: 1800, SeedName: "corn_seed"},
	{Name: "carrot", Value: 15, GrowTimeSec: 900, SeedName: "carrot_seed"},
	{Name: "potato", Value: 20, GrowTimeSec: 3600, SeedName: "potato_seed"},
	{Name: "tomato", Value: 25, GrowTimeSec: 7200, SeedName: "tomato_seed"},
	{Name: "pumpkin", Value: 40, GrowTimeSec: 14400, SeedName: "pumpkin_seed"},
	{Name: "coffee_bean", Value: 60, GrowTimeSec: 28800, SeedName: "coffee_seed"},
	{Name: "cocoa_bean", Value: 75, GrowTimeSec: 43200, SeedName: "cocoa_seed"},
	{Name: "strawberry", Value: 90, GrowTimeSec: 64800, SeedName: "strawberry_seed"},
	{Name: "golden_apple", Value: 150, GrowTimeSec: 86400, SeedName: "golden_apple_seed"},
	{Name: "star_fruit", Value: 250, GrowTimeSec: 172800, SeedName: "star_fruit_seed"},
}

var Seeds = []Seed{
	{Name: "wheat_seed", Price: 2, GrowTimeSec: 300, Crop: cropByName("wheat")},
	{Name: "oat_seed", Price: 3, GrowTimeSec: 600, Crop: cropByName("oat")},
	{Name: "corn_seed", Price: 5, GrowTimeSec: 1800, Crop: cropByName("corn")},
	{Name: "carrot_seed", Price: 7, GrowTimeSec: 900, Crop: cropByName("carrot")},
	{Name: "potato_seed", Price: 8, GrowTimeSec: 3600, Crop: cropByName("potato")},
	{Name: "tomato_seed", Price: 10, GrowTimeSec: 7200, Crop: cropByName("tomato")},
	{Name: "pumpkin_seed", Price: 15, GrowTimeSec: 14400, Crop: cropByName("pumpkin")},
	{Name: "coffee_seed", Price: 25, GrowTimeSec: 28800, Crop: cropByName("coffee_bean")},
	{Name: "cocoa_seed", Price: 30, GrowTimeSec: 43200, Crop: cropByName("cocoa_bean")},
	{Name: "strawberry_seed", Price: 40, GrowTimeSec: 64800, Crop: cropByName("strawberry")},
	{Name: "golden_apple_seed", Price: 75, GrowTimeSec: 86400, Crop: cropByName("golden_apple")},
	{Name: "star_fruit_seed", Price: 125, GrowTimeSec: 172800, Crop: cropByName("star_fruit")},
}

var RegularSeedNames = func() []string {
	var names []string
	for _, s := range Seeds {
		names = append(names, s.Name)
	}
	return names
}()

const PlotsPerZone = 3

func cropByName(name string) Crop {
	for _, c := range Crops {
		if c.Name == name {
			return c
		}
	}
	return Crop{}
}

type PlotInfo struct {
	PlotIndex  int
	ItemName   string
	PlantTime  time.Time
	GrowTime   int
	Ready      bool
	Progress   int
	Watered    bool
	Mutated    bool
	Mysterious bool
}

type HarvestResult struct {
	CropName  string
	Quantity  int
	XP        int
	Value     int
	Mutated   bool
	LeveledUp bool
	NewLevel  int
}

var (
	ErrNoSeed         = errors.New("you don't have that seed")
	ErrOccupied       = errors.New("plot is occupied")
	ErrNotReady       = errors.New("crop is not ready yet")
	ErrAlreadyWatered = errors.New("this plot has already been watered")
	ErrNoFertilizer   = errors.New("you don't have fertilizer")
	ErrNoAccelerator  = errors.New("you don't have a growth elixir")
	ErrCooldown       = errors.New("please wait before using the farm again")
	ErrNoCrop         = errors.New("you don't have that crop")
	ErrNotProcessable = errors.New("that item can't be processed into seeds")
)

type Service struct {
	store  *store.Store
	cfg    *config.Config
	npcSvc *npcsvc.Service
}

func New(s *store.Store, cfg *config.Config, npcSvc *npcsvc.Service) *Service {
	return &Service{store: s, cfg: cfg, npcSvc: npcSvc}
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
	for i := 0; i < PlotsPerZone; i++ {
		p, ok := plotMap[i]
		if !ok {
			out = append(out, PlotInfo{PlotIndex: i})
			continue
		}
		var itemName string
		if p.Mysterious {
			itemName = "mysterious_seed"
		} else {
			itemName = p.ItemName
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
			PlotIndex:  p.PlotIndex,
			ItemName:   itemName,
			PlantTime:  p.PlantTime,
			GrowTime:   p.GrowTime,
			Ready:      ready,
			Progress:   progress,
			Watered:    p.Watered,
			Mutated:    p.Mutated,
			Mysterious: p.Mysterious,
		})
	}
	return out, nil
}

func (s *Service) Plant(userID int64, zoneKey string, plotIndex int, seedName string, growTime int) error {
	var existing model.UserFarming
	if err := s.store.DB.Where("user_id = ? AND zone_key = ? AND plot_index = ?", userID, zoneKey, plotIndex).First(&existing).Error; err == nil {
		return ErrOccupied
	}

	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, seedName).First(&inv).Error; err != nil {
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

	mysterious := seedName == "mysterious_seed"

	level := s.getFarmerLevel(userID)
	reduction := float64(level) * 0.01
	if reduction > 0.5 {
		reduction = 0.5
	}
	finalGrowTime := int(float64(growTime) * (1 - reduction))
	if charsvc.HasPassive(s.store, userID, "perk_green_thumb") {
		finalGrowTime = finalGrowTime * 9 / 10
	}

	if err := s.store.DB.Create(&model.UserFarming{
		UserID:     userID,
		ZoneKey:    zoneKey,
		PlotIndex:  plotIndex,
		ItemName:   seedName,
		PlantTime:  time.Now(),
		GrowTime:   finalGrowTime,
		Mysterious: mysterious,
	}).Error; err != nil {
		return err
	}

	if err := s.store.IncrementGameLimit(userID, "farm"); err != nil {
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

	free, err := s.store.FreeSlots(s.store.DB, userID)
	if err != nil {
		return nil, err
	}
	if free <= 0 {
		return nil, store.ErrInventoryFull
	}

	var cropName string
	var mutation bool

	if plot.Mysterious {
		cropName = s.resolveMystery(plot.ItemName)
		mutation = false
	} else if plot.Mutated {
		cropName = plot.ItemName
		mutation = true
	} else {
		crop := cropBySeedName(plot.ItemName)
		cropName = crop.Name

		if mutated := s.CheckMutation(userID, cropName); mutated != "" {
			cropName = mutated
			mutation = true
		}
	}

	crop := findCrop(cropName)

	level := s.getFarmerLevel(userID)
	quantity := 1

	if plot.Watered {
		if rand.Float64() < 0.35 {
			quantity++
		}
	}

	intBonus := charsvc.GetINTBonus(s.store, userID)
	doubleChance := float64(level)*0.02 + (intBonus-1.0)*0.5
	if rand.Float64() < doubleChance {
		quantity++
	}

	if charsvc.HasBuff(s.store, userID, "scavenger") {
		quantity += quantity / 2
		charsvc.ConsumeBuff(s.store, userID, "scavenger")
	}

	lukBonus := charsvc.GetLUKBonus(s.store, userID)
	if lukBonus > 0 && rand.Float64() < lukBonus*0.1 {
		quantity++
	}

	// A Greenhouse Kit placed in the active house boosts the harvest yield.
	if rand.Float64() < furnituresvc.EffectValue(s.store, userID, "farm_yield") {
		quantity++
	}

	if mutation {
		quantity *= 2
		if quantity < 2 {
			quantity = 2
		}
	}

	var value int
	if mutation {
		value = crop.Value * quantity
	} else {
		value = crop.Value * quantity
	}

	if err := s.store.AddItemRaw(s.store.DB, userID, cropName, quantity); err != nil {
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

	var ch model.UserCropHarvest
	if err := s.store.DB.Where("user_id = ? AND crop_name = ?", userID, cropName).First(&ch).Error; err != nil {
		s.store.DB.Create(&model.UserCropHarvest{UserID: userID, CropName: cropName, Count: quantity})
	} else {
		s.store.DB.Model(&ch).UpdateColumn("count", gorm.Expr("count + ?", quantity))
	}

	s.store.DB.Delete(&plot)

	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "farmer").First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			job = model.Job{UserID: userID, JobName: "farmer", Level: 1, XP: xp}
			levelUpJob(&job)
			s.store.DB.Create(&job)
		}
	} else {
		job.XP += xp
		levelUpJob(&job)
		s.store.DB.Model(&model.Job{}).Where("user_id = ? AND job_name = ?", userID, "farmer").
			Updates(map[string]any{"xp": job.XP, "level": job.Level})
	}

	leveled, lvl := charsvc.AddXP(s.store, userID, xp)

	if err := s.store.IncrementGameLimit(userID, "farm"); err != nil {
		return nil, err
	}

	if mutation {
		s.npcSvc.AddActivityReputation(userID, "farming", 3)
	} else {
		s.npcSvc.AddActivityReputation(userID, "farming", 1)
	}

	return &HarvestResult{CropName: cropName, Quantity: quantity, XP: xp, Value: value, Mutated: mutation, LeveledUp: leveled, NewLevel: lvl}, nil
}

func (s *Service) Water(userID int64, zoneKey string, plotIndex int) error {
	var plot model.UserFarming
	if err := s.store.DB.Where("user_id = ? AND zone_key = ? AND plot_index = ?", userID, zoneKey, plotIndex).First(&plot).Error; err != nil {
		return fmt.Errorf("no crop found")
	}
	if plot.Watered {
		return ErrAlreadyWatered
	}

	elapsed := time.Since(plot.PlantTime).Seconds()
	if elapsed >= float64(plot.GrowTime) {
		return ErrNotReady
	}

	if err := s.store.DB.Model(&plot).UpdateColumn("watered", true).Error; err != nil {
		return err
	}
	return nil
}

func (s *Service) Fertilize(userID int64, zoneKey string, plotIndex int) error {
	var plot model.UserFarming
	if err := s.store.DB.Where("user_id = ? AND zone_key = ? AND plot_index = ?", userID, zoneKey, plotIndex).First(&plot).Error; err != nil {
		return fmt.Errorf("no crop found")
	}

	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, "fertilizer").First(&inv).Error; err != nil {
		return ErrNoFertilizer
	}
	if inv.Quantity < 1 {
		return ErrNoFertilizer
	}
	if inv.Quantity <= 1 {
		s.store.DB.Delete(&inv)
	} else {
		s.store.DB.Model(&inv).UpdateColumn("quantity", gorm.Expr("quantity - 1"))
	}

	reduction := int(float64(plot.GrowTime) * 0.3)
	newGrowTime := plot.GrowTime - reduction
	if newGrowTime < 60 {
		newGrowTime = 60
	}
	if err := s.store.DB.Model(&plot).UpdateColumn("grow_time", newGrowTime).Error; err != nil {
		return err
	}
	return nil
}

// Accelerate consumes a Growth Elixir to instantly mature the crop on the given
// plot. The grow time is set to the elapsed time so the plot becomes harvestable
// immediately.
func (s *Service) Accelerate(userID int64, zoneKey string, plotIndex int) error {
	var plot model.UserFarming
	if err := s.store.DB.Where("user_id = ? AND zone_key = ? AND plot_index = ?", userID, zoneKey, plotIndex).First(&plot).Error; err != nil {
		return fmt.Errorf("no crop found")
	}

	if time.Since(plot.PlantTime).Seconds() >= float64(plot.GrowTime) {
		return ErrNotReady
	}

	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, "growth_elixir").First(&inv).Error; err != nil {
		return ErrNoAccelerator
	}
	if inv.Quantity < 1 {
		return ErrNoAccelerator
	}
	if inv.Quantity <= 1 {
		s.store.DB.Delete(&inv)
	} else {
		s.store.DB.Model(&inv).UpdateColumn("quantity", gorm.Expr("quantity - 1"))
	}

	elapsed := int(time.Since(plot.PlantTime).Seconds())
	if err := s.store.DB.Model(&plot).UpdateColumn("grow_time", elapsed).Error; err != nil {
		return err
	}
	return nil
}

func (s *Service) GetAccessibleZones(userID int64) []string {
	zones := []string{"public"}
	var invs []model.Inventory
	s.store.DB.Where("user_id = ? AND item_id IN (?)", userID, []string{"garden_plot", "tropical_greenhouse", "enchanted_orchard"}).Find(&invs)
	owned := map[string]bool{}
	for _, inv := range invs {
		owned[inv.ItemID] = inv.Quantity > 0
	}
	if owned["garden_plot"] {
		zones = append(zones, "veggie")
	}
	if owned["tropical_greenhouse"] {
		zones = append(zones, "greenhouse")
	}
	if owned["enchanted_orchard"] {
		zones = append(zones, "orchard")
	}
	return zones
}

func (s *Service) HasZoneAccess(userID int64, zoneKey string) bool {
	if zoneKey == "public" {
		return true
	}
	deedMap := map[string]string{
		"veggie":     "garden_plot",
		"greenhouse": "tropical_greenhouse",
		"orchard":    "enchanted_orchard",
	}
	deed, ok := deedMap[zoneKey]
	if !ok {
		return false
	}
	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, deed).First(&inv).Error; err != nil {
		return false
	}
	return inv.Quantity > 0
}

func (s *Service) CountActivePlots(userID int64) int {
	var count int64
	s.store.DB.Model(&model.UserFarming{}).Where("user_id = ?", userID).Count(&count)
	return int(count)
}

func (s *Service) MaxTotalPlots(userID int64) int {
	zones := s.GetAccessibleZones(userID)
	return len(zones) * PlotsPerZone
}

func (s *Service) GetNextHarvest(userID int64) (string, int) {
	var plots []model.UserFarming
	if err := s.store.DB.Where("user_id = ?", userID).Find(&plots).Error; err != nil || len(plots) == 0 {
		return "", 0
	}
	var closest *model.UserFarming
	var closestRemaining int
	for i := range plots {
		elapsed := time.Since(plots[i].PlantTime).Seconds()
		remaining := int(float64(plots[i].GrowTime) - elapsed)
		if remaining <= 0 {
			continue
		}
		if closest == nil || remaining < closestRemaining {
			closest = &plots[i]
			closestRemaining = remaining
		}
	}
	if closest == nil {
		return "", 0
	}
	var name string
	if closest.Mysterious {
		name = "mysterious_seed"
	} else {
		name = closest.ItemName
	}
	return name, closestRemaining
}

func (s *Service) GetFarmerLevel(userID int64) int {
	return s.getFarmerLevel(userID)
}

// levelUpJob applies as many level-ups as the job's XP warrants.
func levelUpJob(job *model.Job) {
	next := 50 + job.Level*25
	for job.XP >= next {
		job.XP -= next
		job.Level++
		next = 50 + job.Level*25
	}
}

func (s *Service) GetFarmerXP(userID int64) (int, int) {
	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "farmer").First(&job).Error; err != nil {
		return 0, 50
	}
	next := 50 + job.Level*25
	return job.XP, next
}

func (s *Service) HasBlessing(userID int64) bool {
	_, ok := blessings[userID]
	return ok
}

func (s *Service) GetBlessingZone(userID int64) string {
	if b, ok := blessings[userID]; ok {
		return b.ZoneKey
	}
	return ""
}

func (s *Service) ConsumeBlessing(userID int64) {
	delete(blessings, userID)
}

func (s *Service) HasItem(userID int64, itemID string) bool {
	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, itemID).First(&inv).Error; err != nil {
		return false
	}
	return inv.Quantity > 0
}

func (s *Service) GetItemQuantity(userID int64, itemID string) int {
	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, itemID).First(&inv).Error; err != nil {
		return 0
	}
	return inv.Quantity
}

func (s *Service) ConsumeItem(userID int64, itemID string, quantity int) bool {
	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, itemID).First(&inv).Error; err != nil {
		return false
	}
	if inv.Quantity < quantity {
		return false
	}
	if inv.Quantity <= quantity {
		s.store.DB.Delete(&inv)
	} else {
		s.store.DB.Model(&inv).UpdateColumn("quantity", gorm.Expr("quantity - ?", quantity))
	}
	return true
}

func (s *Service) AddItem(userID int64, itemID string, quantity int) error {
	return s.store.AddItemRaw(s.store.DB, userID, itemID, quantity)
}

// SeedForCrop returns the seed item ID that produces the given crop.
func SeedForCrop(cropName string) (string, bool) {
	for _, sd := range Seeds {
		if sd.Crop.Name == cropName {
			return sd.Name, true
		}
	}
	return "", false
}

// ConvertToSeeds processes one fruit into 2-4 seeds of the same plant.
func (s *Service) ConvertToSeeds(userID int64, cropName string) (seedID string, qty int, err error) {
	ok := false
	for _, c := range Crops {
		if c.Name == cropName {
			ok = true
			break
		}
	}
	if !ok {
		return "", 0, ErrNotProcessable
	}

	seedID, hasSeed := SeedForCrop(cropName)
	if !hasSeed {
		return "", 0, ErrNotProcessable
	}

	if !s.ConsumeItem(userID, cropName, 1) {
		return "", 0, ErrNoCrop
	}

	qty = 2 + rand.Intn(3)
	if err := s.AddItem(userID, seedID, qty); err != nil {
		return "", 0, err
	}
	return seedID, qty, nil
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

func findCrop(name string) Crop {
	for _, c := range Crops {
		if c.Name == name {
			return c
		}
	}
	return Crop{Name: name, Value: 0}
}

var blessings = map[int64]struct {
	ZoneKey string
}{}

func SetBlessing(userID int64, zoneKey string) {
	blessings[userID] = struct{ ZoneKey string }{ZoneKey: zoneKey}
}
