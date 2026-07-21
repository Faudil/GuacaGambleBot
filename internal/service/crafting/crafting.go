package crafting

import (
	"errors"
	"math"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"gorm.io/gorm"
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
	"bière":                 {Result: "bière", Ingredients: map[string]int{"blé": 3}, LevelRequired: 1, XP: 10},
	"café":                  {Result: "café", Ingredients: map[string]int{"grain de café": 3}, LevelRequired: 1, XP: 10},
	"ticket à gratter":      {Result: "ticket à gratter", Ingredients: map[string]int{"charbon": 1, "caillou": 1}, LevelRequired: 1, XP: 10},
	"engrais":               {Result: "engrais", Ingredients: map[string]int{"plante pourrie": 3, "charbon": 1}, LevelRequired: 2, XP: 15},
	"potion d'oubli":        {Result: "potion d'oubli", Ingredients: map[string]int{"plante pourrie": 2, "poisson-globe": 1}, LevelRequired: 2, XP: 20},
	"fortune cookie":        {Result: "fortune cookie", Ingredients: map[string]int{"blé": 2, "fraise": 1}, LevelRequired: 2, XP: 20},
	"arc":                   {Result: "arc", Ingredients: map[string]int{"avoine": 2, "caillou": 2}, LevelRequired: 3, XP: 25},
	"aimant rouillé":        {Result: "aimant rouillé", Ingredients: map[string]int{"minerai de fer": 3, "caillou": 5}, LevelRequired: 3, XP: 20},
	"hameçon":               {Result: "hameçon", Ingredients: map[string]int{"minerai de fer": 1, "minerai d'argent": 1}, LevelRequired: 3, XP: 25},
	"parchemin d'identité":  {Result: "parchemin d'identité", Ingredients: map[string]int{"plante pourrie": 2, "minerai d'argent": 1}, LevelRequired: 4, XP: 35},
	"aimant":                {Result: "aimant", Ingredients: map[string]int{"minerai de fer": 5, "minerai de cuivre": 1}, LevelRequired: 5, XP: 40},
	"pièce truquée":         {Result: "pièce truquée", Ingredients: map[string]int{"pépite d'or": 1, "caillou": 2, "charbon": 1}, LevelRequired: 5, XP: 45},
	"jeton de casino":       {Result: "jeton de casino", Ingredients: map[string]int{"pépite d'or": 1, "minerai d'argent": 1}, LevelRequired: 6, XP: 50},
	"terrain : potager":     {Result: "terrain : potager", Ingredients: map[string]int{"pépite d'or": 2, "caillou": 20}, LevelRequired: 7, XP: 80},
	"aimant électrique":     {Result: "aimant électrique", Ingredients: map[string]int{"platine": 2, "minerai de cuivre": 5}, LevelRequired: 7, XP: 60},
	"terrain : serre tropicale": {Result: "terrain : serre tropicale", Ingredients: map[string]int{"pépite d'or": 5, "platine": 2}, LevelRequired: 9, XP: 120},
	"ticket vip":            {Result: "ticket vip", Ingredients: map[string]int{"diamant brut": 3, "platine": 2}, LevelRequired: 9, XP: 150},
	"terrain : verger enchanté": {Result: "terrain : verger enchanté", Ingredients: map[string]int{"diamant brut": 2, "émeraude": 2}, LevelRequired: 10, XP: 250},
	"œuf mystère":           {Result: "œuf mystère", Ingredients: map[string]int{"diamant brut": 1, "pomme dorée": 1, "adn pur": 1, "poussière d'os": 10}, LevelRequired: 10, XP: 200},
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

	var dbItem model.Item
	if err := s.store.DB.Where("name = ?", recipe.Result).First(&dbItem).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			dbItem = model.Item{Name: recipe.Result, Price: 0, Description: "", EffectType: ""}
			if err := s.store.DB.Create(&dbItem).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		for ing, qty := range recipe.Ingredients {
			req := qty * amount
			var ingItem model.Item
			if err := tx.Where("name = ?", ing).First(&ingItem).Error; err != nil {
				return ErrNoIngredients
			}
			var inv model.Inventory
			if err := tx.Where("user_id = ? AND item_id = ?", userID, ingItem.ID).First(&inv).Error; err != nil {
				return ErrNoIngredients
			}
			if inv.Quantity < req {
				return ErrNoIngredients
			}
		}
		for ing, qty := range recipe.Ingredients {
			req := qty * amount
			var ingItem model.Item
			if err := tx.Where("name = ?", ing).First(&ingItem).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Inventory{}).
				Where("user_id = ? AND item_id = ?", userID, ingItem.ID).
				UpdateColumn("quantity", gorm.Expr("quantity - ?", req)).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("user_id = ? AND item_id = ?", userID, dbItem.ID).
			FirstOrCreate(&model.Inventory{UserID: userID, ItemID: dbItem.ID, Quantity: 0}).
			UpdateColumn("quantity", gorm.Expr("quantity + ?", amount)).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ? AND job_name = ?", userID, "crafter").
			FirstOrCreate(&model.Job{UserID: userID, JobName: "crafter", Level: 1, XP: 0}).
			UpdateColumn("xp", gorm.Expr("xp + ?", recipe.XP*amount)).Error; err != nil {
			return err
		}
		return nil
	})
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
