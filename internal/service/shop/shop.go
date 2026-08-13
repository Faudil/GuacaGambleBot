package shop

import (
	"encoding/json"
	"errors"
	"math/rand"
	"time"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	if it.EquipSlot != "" {
		// Equipment item: create a UserEquipment instance with rolled affixes
		rar := it.Rarity
		affixes := items.RollAffixes(rar, it.EquipSlot)
		var applied []items.AppliedAffix
		for _, a := range affixes {
			applied = append(applied, items.AppliedAffix{
				ID:    a.ID,
				Name:  a.Name,
				Stat:  a.Stat,
				Value: items.RollAffixValue(a),
			})
		}
		totalSTR, totalDEX, totalINT, totalVIT, totalLUK := it.StatSTR, it.StatDEX, it.StatINT, it.StatVIT, it.StatLUK
		for _, a := range applied {
			switch a.Stat {
			case "str":
				totalSTR += a.Value
			case "dex":
				totalDEX += a.Value
			case "int":
				totalINT += a.Value
			case "vit":
				totalVIT += a.Value
			case "luk":
				totalLUK += a.Value
			}
		}
		affixData, _ := json.Marshal(applied)
		return s.store.DB.Transaction(func(tx *gorm.DB) error {
			if _, err := s.store.UpdateBalance(userID, -totalCost); err != nil {
				return err
			}
			_, err := s.store.CreateEquipment(userID, it.ID, it.Name, it.Emoji,
				string(rar), it.EquipSlot, it.MinLevel,
				totalSTR, totalDEX, totalINT, totalVIT, totalLUK,
				affixData, it.SetID)
			return err
		})
	}

	// Non-equipment: add to regular inventory
	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := s.store.UpdateBalance(userID, -totalCost); err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
			DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + ?", quantity)},
			)}).Create(&model.Inventory{UserID: userID, ItemID: itemName, Quantity: quantity}).Error
	})
}

func hashSeed(s string) int64 {
	var h int64
	for _, c := range s {
		h = h*31 + int64(c)
	}
	return h
}
