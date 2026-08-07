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
	charsvc "guacagamblebot/internal/service/character"
	invsvc "guacagamblebot/internal/service/inventory"
	loresvc "guacagamblebot/internal/service/lore"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
)

type BaitTier string

const (
	BaitCommon    BaitTier = "common"
	BaitRare      BaitTier = "rare"
	BaitLegendary BaitTier = "legendary"
)

type FishSpecies struct {
	Name        string
	ItemID      string
	Biome       string
	BaitTier    BaitTier
	Strength    int
	Evasiveness int
	Stamina     int
	MinWeight   int
	MaxWeight   int
	MinSize     int
	MaxSize     int
	Secret      string // "" normal, "ghost_carp", "cosmic_jellyfish"
}

var FishPool = []FishSpecies{
	// Pond
	{Name: "Old Boot", ItemID: "old_boot", Biome: "pond", BaitTier: BaitCommon, Strength: 0, Evasiveness: 0, Stamina: 15, MinWeight: 1, MaxWeight: 2, MinSize: 5, MaxSize: 10},
	{Name: "Common Carp", ItemID: "carp", Biome: "pond", BaitTier: BaitCommon, Strength: 1, Evasiveness: 1, Stamina: 25, MinWeight: 2, MaxWeight: 5, MinSize: 10, MaxSize: 20},
	{Name: "Goldfish", ItemID: "golden_apple", Biome: "pond", BaitTier: BaitRare, Strength: 2, Evasiveness: 2, Stamina: 30, MinWeight: 1, MaxWeight: 3, MinSize: 5, MaxSize: 10},
	{Name: "River Eel", ItemID: "sardine", Biome: "pond", BaitTier: BaitLegendary, Strength: 3, Evasiveness: 8, Stamina: 40, MinWeight: 3, MaxWeight: 8, MinSize: 20, MaxSize: 40},
	// River
	{Name: "Trout", ItemID: "trout", Biome: "river", BaitTier: BaitCommon, Strength: 2, Evasiveness: 2, Stamina: 30, MinWeight: 1, MaxWeight: 4, MinSize: 10, MaxSize: 20},
	{Name: "Salmon", ItemID: "salmon", Biome: "river", BaitTier: BaitCommon, Strength: 3, Evasiveness: 3, Stamina: 35, MinWeight: 3, MaxWeight: 10, MinSize: 15, MaxSize: 30},
	{Name: "Pike", ItemID: "pufferfish", Biome: "river", BaitTier: BaitRare, Strength: 5, Evasiveness: 3, Stamina: 45, MinWeight: 5, MaxWeight: 15, MinSize: 20, MaxSize: 40},
	{Name: "Giant Catfish", ItemID: "whale", Biome: "river", BaitTier: BaitLegendary, Strength: 6, Evasiveness: 5, Stamina: 60, MinWeight: 20, MaxWeight: 60, MinSize: 30, MaxSize: 60},
	// Ocean
	{Name: "Cod", ItemID: "sardine", Biome: "ocean", BaitTier: BaitCommon, Strength: 3, Evasiveness: 2, Stamina: 35, MinWeight: 2, MaxWeight: 8, MinSize: 15, MaxSize: 25},
	{Name: "Swordfish", ItemID: "swordfish", Biome: "ocean", BaitTier: BaitCommon, Strength: 5, Evasiveness: 3, Stamina: 50, MinWeight: 50, MaxWeight: 200, MinSize: 60, MaxSize: 120},
	{Name: "Shark", ItemID: "shark", Biome: "ocean", BaitTier: BaitRare, Strength: 6, Evasiveness: 4, Stamina: 55, MinWeight: 100, MaxWeight: 500, MinSize: 80, MaxSize: 200},
	{Name: "Whale", ItemID: "whale", Biome: "ocean", BaitTier: BaitLegendary, Strength: 7, Evasiveness: 6, Stamina: 75, MinWeight: 1000, MaxWeight: 10000, MinSize: 200, MaxSize: 600},
	// Lava Pool
	{Name: "Lava Guppy", ItemID: "trout", Biome: "lava", BaitTier: BaitCommon, Strength: 4, Evasiveness: 3, Stamina: 35, MinWeight: 1, MaxWeight: 3, MinSize: 5, MaxSize: 10},
	{Name: "Magma Carp", ItemID: "magma_carp", Biome: "lava", BaitTier: BaitRare, Strength: 6, Evasiveness: 5, Stamina: 55, MinWeight: 10, MaxWeight: 50, MinSize: 20, MaxSize: 50},
	{Name: "Lava Serpent", ItemID: "lava_serpent", Biome: "lava", BaitTier: BaitLegendary, Strength: 8, Evasiveness: 7, Stamina: 85, MinWeight: 50, MaxWeight: 200, MinSize: 100, MaxSize: 300},
	// Legendary global
	{Name: "Legendary Kraken", ItemID: "kraken_tentacle", Biome: "any", BaitTier: BaitLegendary, Strength: 10, Evasiveness: 10, Stamina: 100, MinWeight: 5000, MaxWeight: 50000, MinSize: 500, MaxSize: 2000},
	// Secret: Ghost Carp (pond + legendary + night only)
	{Name: "Ghost Carp", ItemID: "carp", Biome: "pond", BaitTier: BaitLegendary, Strength: 4, Evasiveness: 6, Stamina: 50, MinWeight: 2, MaxWeight: 8, MinSize: 10, MaxSize: 25, Secret: "ghost_carp"},
	// Secret: Cosmic Jellyfish (any biome + legendary, very rare)
	{Name: "Cosmic Jellyfish", ItemID: "star_fruit", Biome: "any", BaitTier: BaitLegendary, Strength: 9, Evasiveness: 9, Stamina: 70, MinWeight: 10, MaxWeight: 50, MinSize: 30, MaxSize: 80, Secret: "cosmic_jellyfish"},
}

type FishFightState struct {
	Species    FishSpecies
	Tension    int
	Stamina    int
	Distance   int
	LuckyBreak bool
	Escaped    bool
	Weight     int
	Size       int
	Golden     bool
	Mutated    bool
	Mood       string
	Quiet      bool
}

type FightActionResult struct {
	TensionDelta int
	StaminaDelta int
	DistanceStep bool
	Caught       bool
	Escaped      bool
	LuckyBreak   bool
	Mood         string
}

type FightResolve struct {
	ItemName    string
	ItemID      string
	Value       int
	Weight      int
	Size        int
	XP          int
	Caught      bool
	Golden      bool
	Mutated     bool
	Secret      string
	LoreID      string
	LoreName    string
	BottleMsg   string
	LeveledUp   bool
	NewLevel    int
}

var (
	ErrCooldown   = errors.New("on cooldown")
	ErrLimit      = errors.New("daily limit reached")
	ErrNoBait     = errors.New("no bait available")
	ErrLavaLocked = errors.New("lava pool locked")
)

type Service struct {
	store   *store.Store
	cfg     *config.Config
	invSvc  *invsvc.Service
	loreSvc *loresvc.Service
	npcSvc  *npcsvc.Service
}

func New(s *store.Store, cfg *config.Config, loreSvc *loresvc.Service, npcSvc *npcsvc.Service) *Service {
	return &Service{store: s, cfg: cfg, invSvc: invsvc.New(s, cfg), loreSvc: loreSvc, npcSvc: npcSvc}
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

func (s *Service) CanFreeCast(userID int64) (bool, error) {
	ok, _, err := s.store.CheckGameLimit(userID, "fish_free", 1)
	return ok, err
}

func (s *Service) UseFreeCast(userID int64) error {
	return s.store.IncrementGameLimit(userID, "fish_free")
}

func (s *Service) HasBait(userID int64, tier BaitTier) (bool, error) {
	return s.invSvc.HasItem(userID, baitItemID(tier), 1), nil
}

func (s *Service) ConsumeBait(userID int64, tier BaitTier) error {
	itemID := baitItemID(tier)
	if itemID == "" {
		return ErrNoBait
	}
	return s.invSvc.RemoveItem(s.store.DB, userID, itemID, 1)
}

func (s *Service) GetFisherLevel(userID int64) (int, error) {
	var job model.Job
	err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "fisher").First(&job).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return job.Level, nil
}

func (s *Service) LavaUnlocked(userID int64) (bool, error) {
	lvl, err := s.GetFisherLevel(userID)
	if err != nil {
		return false, err
	}
	return lvl >= 10, nil
}

func isNight() bool {
	h := time.Now().UTC().Hour()
	return h < 6 || h >= 19
}

func (s *Service) GenerateFish(biome string, baitTier BaitTier) *FishFightState {
	night := isNight()

	var candidates []FishSpecies
	for _, f := range FishPool {
		if f.Secret == "ghost_carp" {
			if biome == "pond" && baitTier == BaitLegendary && night && rand.Float64() < 0.05 {
				candidates = append(candidates, f)
			}
			continue
		}
		if f.Secret == "cosmic_jellyfish" {
			if baitTier == BaitLegendary && rand.Float64() < 0.005 {
				candidates = append(candidates, f)
			}
			continue
		}
		if f.Biome != biome && f.Biome != "any" {
			continue
		}
		if tierWeight(f.BaitTier) > tierWeight(baitTier) {
			continue
		}
		candidates = append(candidates, f)
	}

	if len(candidates) == 0 {
		candidates = []FishSpecies{{Name: "Old Boot", ItemID: "old_boot", Biome: biome, BaitTier: BaitCommon, Strength: 0, Evasiveness: 0, Stamina: 15, MinWeight: 1, MaxWeight: 2, MinSize: 5, MaxSize: 10}}
	}

	var pick FishSpecies
	weights := make([]float64, len(candidates))
	total := 0.0
	for i, f := range candidates {
		var w float64
		switch baitTier {
		case BaitLegendary:
			w = float64(tierWeight(f.BaitTier))
		case BaitRare:
			w = 1.0
		default:
			w = 1.0 / float64(tierWeight(f.BaitTier))
		}
		weights[i] = w
		total += w
	}
	r := rand.Float64() * total
	cum := 0.0
	for i, w := range weights {
		cum += w
		if r <= cum {
			pick = candidates[i]
			break
		}
	}
	if pick.Name == "" {
		pick = candidates[len(candidates)-1]
	}

	weight := pick.MinWeight + rand.Intn(pick.MaxWeight-pick.MinWeight+1)
	size := pick.MinSize + rand.Intn(pick.MaxSize-pick.MinSize+1)

	golden := rand.Float64() < 0.01
	if golden {
		weight *= 2
		size *= 2
	}

	mutated := rand.Float64() < 0.013
	stamina := pick.Stamina
	if mutated {
		stamina = stamina + stamina/2
	}

	quiet := baitTier == BaitCommon && pick.Secret == "" && rand.Float64() < 0.35

	state := &FishFightState{
		Species:  pick,
		Tension:  100,
		Stamina:  stamina,
		Distance: 0,
		Weight:   weight,
		Size:     size,
		Golden:   golden,
		Mutated:  mutated,
		Mood:     "diving",
		Quiet:    quiet,
	}

	if mutated {
		state.Species.Strength += 3
		state.Species.Evasiveness += 3
	}

	if night && baitTier == BaitLegendary {
		state.Weight = state.Weight * 120 / 100
		state.Size = state.Size * 120 / 100
	}

	return state
}

func (s Service) ApplyAction(state *FishFightState, action string) *FightActionResult {
	result := &FightActionResult{}

	var tensionLoss, staminaLoss, tensionGain, staminaRecovered int
	switch action {
	case "reel":
		tensionLoss = 5 + rand.Intn(11)
		staminaLoss = 8 + rand.Intn(6)
		applyMoodModifiers(state, &tensionLoss, &staminaLoss, &tensionGain, &staminaRecovered, action)
		state.Tension -= tensionLoss
		state.Stamina -= staminaLoss

	case "pull":
		tensionLoss = 20 + rand.Intn(21) + state.Species.Strength*2
		staminaLoss = 15 + rand.Intn(21)
		applyMoodModifiers(state, &tensionLoss, &staminaLoss, &tensionGain, &staminaRecovered, action)
		state.Tension -= tensionLoss
		state.Stamina -= staminaLoss
		if state.Distance < 1 {
			state.Distance = 1
			result.DistanceStep = true
		}

	case "rest":
		tensionGain = 20 + rand.Intn(11)
		staminaRecovered = 3 + state.Species.Evasiveness + rand.Intn(3)
		applyMoodModifiers(state, &tensionLoss, &staminaLoss, &tensionGain, &staminaRecovered, action)
		state.Tension += tensionGain
		if state.Tension > 100 {
			state.Tension = 100
		}
		state.Stamina += staminaRecovered
		result.TensionDelta = tensionGain
		result.StaminaDelta = staminaRecovered
	}

	if action != "rest" {
		result.TensionDelta = -tensionLoss
		result.StaminaDelta = -staminaLoss
	}

	if state.Tension < 0 {
		state.Tension = 0
	}
	if state.Stamina < 0 {
		state.Stamina = 0
	}

	if state.Stamina <= 0 {
		state.Distance = 2
		result.Caught = true
		result.Mood = state.Mood
		return result
	}

	if state.Tension <= 0 {
		if !state.LuckyBreak {
			if rand.Intn(100) < 10 {
				state.Tension = 1
				state.LuckyBreak = true
				result.LuckyBreak = true
				result.Mood = state.Mood
				return result
			}
		}
		state.Escaped = true
		result.Escaped = true
		result.Mood = state.Mood
		return result
	}

	nextMood(state)
	result.Mood = state.Mood

	return result
}

func applyMoodModifiers(state *FishFightState, tensionLoss, staminaLoss, tensionGain, staminaRecovered *int, action string) {
	switch state.Mood {
	case "diving":
		if action == "pull" {
			*tensionLoss += 10
		}
		if action == "rest" {
			*staminaRecovered += 2
		}
	case "thrashing":
		if action == "reel" {
			*staminaLoss = *staminaLoss + *staminaLoss/2
		}
	case "tiring":
		if action == "pull" {
			*tensionLoss = *tensionLoss / 2
			if *tensionLoss < 1 {
				*tensionLoss = 1
			}
		}
	case "circling":
		if action == "reel" || action == "pull" {
			*staminaLoss += rand.Intn(*staminaLoss/2 + 1) - *staminaLoss/4
			if *staminaLoss < 1 {
				*staminaLoss = 1
			}
		}
	}
}

func nextMood(state *FishFightState) {
	staminaMax := state.Species.Stamina
	if staminaMax <= 0 {
		staminaMax = 1
	}
	staminaPct := state.Stamina * 100 / staminaMax

	var moods []string
	switch {
	case staminaPct > 70:
		moods = []string{"diving", "diving", "thrashing", "circling"}
	case staminaPct > 40:
		moods = []string{"diving", "thrashing", "circling", "circling"}
	default:
		moods = []string{"thrashing", "thrashing", "tiring", "circling"}
	}

	next := moods[rand.Intn(len(moods))]
	for next == state.Mood && len(moods) > 1 {
		next = moods[rand.Intn(len(moods))]
	}
	state.Mood = next
}

func (s *Service) ResolveCatch(userID int64, state *FishFightState) (*FightResolve, error) {
	xp := 15 + state.Species.Stamina + rand.Intn(11)
	if state.Golden {
		xp += 50
	}

	if charsvc.HasBuff(s.store, userID, "scavenger") {
		charsvc.ConsumeBuff(s.store, userID, "scavenger")
		_ = s.store.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
			DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + 2")}),
		}).Create(&model.Inventory{UserID: userID, ItemID: state.Species.ItemID, Quantity: 2}).Error
	} else {
		if err := s.store.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
			DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + 1")}),
		}).Create(&model.Inventory{UserID: userID, ItemID: state.Species.ItemID, Quantity: 1}).Error; err != nil {
			return nil, err
		}
	}

	if state.Mutated {
		_ = s.store.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
			DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + 1")}),
		}).Create(&model.Inventory{UserID: userID, ItemID: "mutagen", Quantity: 1}).Error
	}

	res := &FightResolve{
		ItemName: state.Species.Name,
		ItemID:   state.Species.ItemID,
		Weight:   state.Weight,
		Size:     state.Size,
		XP:       xp,
		Caught:   true,
		Golden:   state.Golden,
		Mutated:  state.Mutated,
		Secret:   state.Species.Secret,
	}

	if state.Species.Secret == "ghost_carp" || state.Species.Secret == "cosmic_jellyfish" {
		if frag := s.loreSvc.PickUndiscovered(s.store.DB, userID, universe.Category("tide_scroll")); frag != nil {
			if ok, _ := s.loreSvc.Discover(userID, frag.ID); ok {
				res.LoreID = frag.ID
				res.LoreName = frag.ID
			}
		}
	}

	rep := 1
	switch {
	case state.Golden:
		rep = 10
	case state.Species.Secret != "":
		rep = 15
	case tierWeight(state.Species.BaitTier) >= 3:
		rep = 8
	case tierWeight(state.Species.BaitTier) == 2:
		rep = 4
	}
	if state.Species.ItemID == "kraken_tentacle" {
		rep = 40
	}
	s.npcSvc.AddActivityReputation(userID, "fishing", rep)

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

	leveled, lvl := charsvc.AddXP(s.store, userID, xp)
	res.LeveledUp = leveled
	res.NewLevel = lvl

	return res, nil
}

func (s *Service) ResolveEscape(userID int64) (*FightResolve, error) {
	xp := 5

	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "fisher").First(&job).Error; err == nil {
		job.XP += xp
		next := jobXPForLevel(job.Level)
		if job.XP >= next {
			job.XP -= next
			job.Level++
		}
		_ = s.store.DB.Model(&model.Job{}).Where("user_id = ? AND job_name = ?", userID, "fisher").
			Updates(map[string]any{"xp": job.XP, "level": job.Level}).Error
	}

	if err := s.store.IncrementGameLimit(userID, "fish"); err != nil {
		return nil, err
	}
	if err := s.SetCooldown(userID); err != nil {
		return nil, err
	}

	leveled, lvl := charsvc.AddXP(s.store, userID, xp)

	return &FightResolve{XP: xp, Caught: false, LeveledUp: leveled, NewLevel: lvl}, nil
}

func (s *Service) RollMessageBottle() string {
	if rand.Float64() >= 0.02 {
		return ""
	}
	messages := []string{
		"fishing.bottle_lore",
		"fishing.bottle_hint_ghost",
		"fishing.bottle_hint_kraken",
		"fishing.bottle_letter_irian",
		"fishing.bottle_letter_elara",
	}
	return messages[rand.Intn(len(messages))]
}

func (s *Service) ResolveBottle(userID int64) (*FightResolve, error) {
	xp := 10
	var loreID string
	if frag := s.loreSvc.PickUndiscovered(s.store.DB, userID, universe.Category("tide_scroll")); frag != nil {
		if ok, _ := s.loreSvc.Discover(userID, frag.ID); ok {
			loreID = frag.ID
		}
	}

	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "fisher").First(&job).Error; err == nil {
		job.XP += xp
		next := jobXPForLevel(job.Level)
		if job.XP >= next {
			job.XP -= next
			job.Level++
		}
		_ = s.store.DB.Model(&model.Job{}).Where("user_id = ? AND job_name = ?", userID, "fisher").
			Updates(map[string]any{"xp": job.XP, "level": job.Level}).Error
	}

	if err := s.store.IncrementGameLimit(userID, "fish"); err != nil {
		return nil, err
	}
	if err := s.SetCooldown(userID); err != nil {
		return nil, err
	}

	leveled, lvl := charsvc.AddXP(s.store, userID, xp)

	return &FightResolve{XP: xp, Caught: true, LoreID: loreID, BottleMsg: "fishing.bottle_found", LeveledUp: leveled, NewLevel: lvl}, nil
}

func (s *Service) AddBait(userID int64, tier BaitTier) error {
	itemID := baitItemID(tier)
	if itemID == "" {
		return ErrNoBait
	}
	return s.invSvc.AddItem(s.store.DB, userID, itemID, 1)
}

func BiteWaitForBiome(biome string) int {
	switch biome {
	case "pond":
		return 3 + rand.Intn(8)
	case "river":
		return 6 + rand.Intn(10)
	case "ocean":
		return 10 + rand.Intn(12)
	case "lava":
		return 12 + rand.Intn(16)
	}
	return 8 + rand.Intn(8)
}

func baitItemID(tier BaitTier) string {
	switch tier {
	case BaitCommon:
		return "worm"
	case BaitRare:
		return "crayfish"
	case BaitLegendary:
		return "golden_lure"
	}
	return ""
}

func tierWeight(t BaitTier) int {
	switch t {
	case BaitCommon:
		return 1
	case BaitRare:
		return 2
	case BaitLegendary:
		return 3
	}
	return 0
}

func jobXPForLevel(level int) int {
	return 50 + level*25
}
