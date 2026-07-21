package crafting

import (
	"errors"
	"math"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
	"guacagamblebot/internal/store"
)

var (
	ErrNoRecipe      = errors.New("recipe not found")
	ErrNoLevel       = errors.New("level too low")
	ErrNoIngredients = errors.New("missing ingredients")
)

type Recipe struct {
	Result        string
	Ingredients   map[string]int
	LevelRequired int
	XP            int
}

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

var Recipes = map[string]Recipe{
	"beer":                {Result: "beer", Ingredients: map[string]int{"wheat": 3}, LevelRequired: 1, XP: 10},
	"coffee":              {Result: "coffee", Ingredients: map[string]int{"coffee_bean": 3}, LevelRequired: 1, XP: 10},
	"scratch_ticket":      {Result: "scratch_ticket", Ingredients: map[string]int{"coal": 1, "pebble": 1}, LevelRequired: 1, XP: 10},
	"fertilizer":          {Result: "fertilizer", Ingredients: map[string]int{"rotten_plant": 3, "coal": 1}, LevelRequired: 2, XP: 15},
	"forget_potion":       {Result: "forget_potion", Ingredients: map[string]int{"rotten_plant": 2, "pufferfish": 1}, LevelRequired: 2, XP: 20},
	"fortune_cookie":      {Result: "fortune_cookie", Ingredients: map[string]int{"wheat": 2, "strawberry": 1}, LevelRequired: 2, XP: 20},
	"bow":                 {Result: "bow", Ingredients: map[string]int{"oat": 2, "pebble": 2}, LevelRequired: 3, XP: 25},
	"rusty_magnet":        {Result: "rusty_magnet", Ingredients: map[string]int{"iron_ore": 3, "pebble": 5}, LevelRequired: 3, XP: 20},
	"hook":                {Result: "hook", Ingredients: map[string]int{"iron_ore": 1, "silver_ore": 1}, LevelRequired: 3, XP: 25},
	"identity_scroll":     {Result: "identity_scroll", Ingredients: map[string]int{"rotten_plant": 2, "silver_ore": 1}, LevelRequired: 4, XP: 35},
	"magnet":              {Result: "magnet", Ingredients: map[string]int{"iron_ore": 5, "copper_ore": 1}, LevelRequired: 5, XP: 40},
	"rigged_coin":         {Result: "rigged_coin", Ingredients: map[string]int{"gold_nugget": 1, "pebble": 2, "coal": 1}, LevelRequired: 5, XP: 45},
	"casino_token":        {Result: "casino_token", Ingredients: map[string]int{"gold_nugget": 1, "silver_ore": 1}, LevelRequired: 6, XP: 50},
	"garden_plot":         {Result: "garden_plot", Ingredients: map[string]int{"gold_nugget": 2, "pebble": 20}, LevelRequired: 7, XP: 80},
	"electric_magnet":     {Result: "electric_magnet", Ingredients: map[string]int{"platinum": 2, "copper_ore": 5}, LevelRequired: 7, XP: 60},
	"tropical_greenhouse": {Result: "tropical_greenhouse", Ingredients: map[string]int{"gold_nugget": 5, "platinum": 2}, LevelRequired: 9, XP: 120},
	"vip_ticket":          {Result: "vip_ticket", Ingredients: map[string]int{"rough_diamond": 3, "platinum": 2}, LevelRequired: 9, XP: 150},
	"enchanted_orchard":   {Result: "enchanted_orchard", Ingredients: map[string]int{"rough_diamond": 2, "emerald": 2}, LevelRequired: 10, XP: 250},
	"mystery_egg":         {Result: "mystery_egg", Ingredients: map[string]int{"rough_diamond": 1, "golden_apple": 1, "pure_dna": 1, "bone_dust": 10}, LevelRequired: 10, XP: 200},
}

func (s *Service) GetCrafterLevel(userID int64) int {
	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "crafter").First(&job).Error; err != nil {
		return 1
	}
	return job.Level
}

func (s *Service) Craft(userID int64, recipeKey string, amount int) error {
	recipe, ok := Recipes[recipeKey]
	if !ok {
		return ErrNoRecipe
	}
	level := s.GetCrafterLevel(userID)
	if level < recipe.LevelRequired {
		return ErrNoLevel
	}

	intMult := charsvc.GetINTBonus(s.store, userID)
	charXP := int(float64(recipe.XP*amount) * intMult)
	totalXP := recipe.XP * amount

	effectiveAmount := amount
	ingMultiplier := 1.0
	if charsvc.HasBuff(s.store, userID, "efficiency") {
		ingMultiplier = 0.5
		charsvc.ConsumeBuff(s.store, userID, "efficiency")
	}

	if charsvc.HasBuff(s.store, userID, "perfect_forge") {
		charsvc.ConsumeBuff(s.store, userID, "perfect_forge")
		// perfect forge makes the output quality higher (represented by double output)
		effectiveAmount = amount * 2
	}

	if err := s.store.DB.Transaction(func(tx *gorm.DB) error {
		for ing, qty := range recipe.Ingredients {
			req := max(1, int(float64(qty*amount)*ingMultiplier))
			var inv model.Inventory
			if err := tx.Where("user_id = ? AND item_id = ? AND quantity >= ?", userID, ing, req).First(&inv).Error; err != nil {
				return ErrNoIngredients
			}
		}
		for ing, qty := range recipe.Ingredients {
			req := max(1, int(float64(qty*amount)*ingMultiplier))
			if err := tx.Model(&model.Inventory{}).
				Where("user_id = ? AND item_id = ?", userID, ing).
				UpdateColumn("quantity", gorm.Expr("quantity - ?", req)).Error; err != nil {
				return err
			}
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
			DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + ?", effectiveAmount)}),
		}).Create(&model.Inventory{UserID: userID, ItemID: recipe.Result, Quantity: effectiveAmount}).Error; err != nil {
			return err
		}
		tx.Where("user_id = ? AND job_name = ?", userID, "crafter").
			FirstOrCreate(&model.Job{UserID: userID, JobName: "crafter", Level: 1, XP: 0})
		if err := tx.Model(&model.Job{}).
			Where("user_id = ? AND job_name = ?", userID, "crafter").
			UpdateColumn("xp", gorm.Expr("xp + ?", totalXP)).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	charsvc.AddXP(s.store, userID, charXP)
	return nil
}

func (s *Service) LevelUpCheck(userID int64) (bool, int) {
	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "crafter").First(&job).Error; err != nil {
		return false, 1
	}
	xpNeeded := job.Level * 100
	if job.XP >= xpNeeded {
		newLevel := job.Level + 1
		newXP := job.XP - xpNeeded
		s.store.DB.Model(&job).Where("user_id = ? AND job_name = ?", userID, "crafter").
			Updates(map[string]any{"level": newLevel, "xp": newXP})
		return true, newLevel
	}
	return false, job.Level
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func floor(f float64) int {
	return int(math.Floor(f))
}
