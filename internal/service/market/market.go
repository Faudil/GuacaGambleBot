package market

import (
	"errors"
	"math"
	"math/rand"
	"time"

	"gorm.io/gorm"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
	"guacagamblebot/internal/store"
)

var (
	ErrNotSellable = errors.New("item not sellable on market")
	ErrNotFound    = errors.New("item not found")
	ErrNoItem      = errors.New("item not owned")
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

type MarketItem struct {
	Item         *items.Item
	CurrentPrice int
	BasePrice    int
	Multiplier   float64
}

type MarketCategory struct {
	Name  string
	Items []MarketItem
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) GetMarketPrices() []MarketCategory {
	now := time.Now().Format("2006-01-02")
	rng := rand.New(rand.NewSource(hashSeed(now)))

	catDefs := []struct {
		name     string
		category items.Category
	}{
		{"mining", items.Mining},
		{"fishing", items.Fishing},
		{"farming", items.Farming},
	}

	cats := make([]MarketCategory, 0, len(catDefs))
	for _, cd := range catDefs {
		all := items.ItemsByCategory(cd.category)
		mkt := make([]MarketItem, 0, len(all))
		for _, it := range all {
			mult := 0.5 + rng.Float64()*2.5
			mult = math.Round(mult*100) / 100
			mkt = append(mkt, MarketItem{
				Item:         &it,
				CurrentPrice: int(math.Max(1, float64(it.Price)*mult)),
				BasePrice:    it.Price,
				Multiplier:   mult,
			})
		}
		cats = append(cats, MarketCategory{Name: cd.name, Items: mkt})
	}
	return cats
}

func (s *Service) SellItem(userID int64, itemName string, amount int) (int, error) {
	market := s.GetMarketPrices()
	var found *MarketItem
	for _, cat := range market {
		for _, mi := range cat.Items {
			if mi.Item.ID == itemName {
				found = &mi
				break
			}
		}
		if found != nil {
			break
		}
	}
	if found == nil {
		return 0, ErrNotSellable
	}
	totalGain := found.CurrentPrice * amount

	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, itemName).First(&inv).Error; err != nil {
		return 0, ErrNoItem
	}
	if inv.Quantity < amount {
		return 0, ErrNoItem
	}

	if charsvc.HasBuff(s.store, userID, "golden_touch") {
		totalGain *= 2
		charsvc.ConsumeBuff(s.store, userID, "golden_touch")
	}

	if charsvc.HasBuff(s.store, userID, "insider_trading") {
		totalGain = totalGain * 15 / 10
		charsvc.ConsumeBuff(s.store, userID, "insider_trading")
	}

	err := s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Inventory{}).
			Where("user_id = ? AND item_id = ?", userID, itemName).
			UpdateColumn("quantity", gorm.Expr("quantity - ?", amount)).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).
			FirstOrCreate(&model.User{UserID: userID}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", userID).
			UpdateColumn("balance", gorm.Expr("balance + ?", totalGain)).Error; err != nil {
			return err
		}
		if err := achievement.IncrementStat(tx, userID, "items_sold_market", amount); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	charsvc.AddXP(s.store, userID, amount)
	return totalGain, nil
}

func hashSeed(s string) int64 {
	var h int64
	for _, c := range s {
		h = h*31 + int64(c)
	}
	return h
}
