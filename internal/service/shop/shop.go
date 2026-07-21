package shop

import (
	"errors"
	"math/rand"
	"time"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"gorm.io/gorm"
)

var (
	ErrNotFound = errors.New("item not found")
	ErrNoMoney  = errors.New("insufficient funds")
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

type ShopOffer struct {
	Item       *items.Item
	Price      int
	Discounted bool
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) DailyOffers(count int) []ShopOffer {
	all := items.AllItems()
	seed := time.Now().Format("2006-01-02")
	rng := rand.New(rand.NewSource(hashSeed(seed)))
	n := count
	if n > len(all) {
		n = len(all)
	}
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
	offers := make([]ShopOffer, 0, n)
	for i := 0; i < n; i++ {
		it := all[i]
		price := it.Price
		discounted := rng.Float64() < 0.35
		if discounted {
			disc := rng.Intn(26) + 5
			price = price * (100 - disc) / 100
			if price < 1 {
				price = 1
			}
		}
		offers = append(offers, ShopOffer{Item: &it, Price: price, Discounted: discounted})
	}
	return offers
}

func (s *Service) BuyItem(userID int64, itemName string, quantity int) error {
	it := items.Get(itemName)
	if it == nil {
		return ErrNotFound
	}
	totalCost := it.Price * quantity
	bal, err := s.store.GetBalance(userID)
	if err != nil {
		return err
	}
	if bal < totalCost {
		return ErrNoMoney
	}
	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := s.store.UpdateBalance(userID, -totalCost); err != nil {
			return err
		}
		var dbItem model.Item
		if err := tx.Where("name = ?", itemName).FirstOrCreate(&dbItem, model.Item{
			Name: itemName, Price: it.Price, Description: it.Description, EffectType: it.EffectType,
		}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ? AND item_id = ?", userID, dbItem.ID).
			FirstOrCreate(&model.Inventory{UserID: userID, ItemID: dbItem.ID, Quantity: 0}).
			UpdateColumn("quantity", gorm.Expr("quantity + ?", quantity)).Error
	})
}

func hashSeed(s string) int64 {
	var h int64
	for _, c := range s {
		h = h*31 + int64(c)
	}
	return h
}
