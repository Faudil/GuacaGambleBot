package shop

import (
	"encoding/json"
	"errors"
	"math/rand"
	"time"

	"gorm.io/gorm"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/store"
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

// shopOfferable reports whether an item may be offered by the daily shop.
// Free, collectible and award-only items, legendary set pieces, legendary
// activity drops and items that are exclusive to their dedicated activity
// (criminality, boss leagues, ...) are never sold here.
func shopOfferable(it items.Item) bool {
	if it.Price <= 0 || it.EffectType == "collectible" {
		return false
	}
	if it.ShopExcluded || it.SetID != "" || it.IsLegendaryDrop() {
		return false
	}
	return true
}

// OfferableItems returns every item that may appear in the daily shop.
func OfferableItems() []items.Item {
	all := items.AllItems()
	out := make([]items.Item, 0, len(all))
	for _, it := range all {
		if shopOfferable(it) {
			out = append(out, it)
		}
	}
	return out
}

// NextRefresh returns when today's offers expire and the shop rolls over to a
// new set, i.e. the next local midnight after now. Must stay in sync with the
// "2006-01-02" seed used by DailyOffers.
func (s *Service) NextRefresh() time.Time {
	now := time.Now()
	next := now.AddDate(0, 0, 1)
	return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
}

func (s *Service) DailyOffers(count int) []ShopOffer {
	all := OfferableItems()
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

// OfferForItem returns today's offer for the given item ID, if it is present in
// the daily rotation.
func (s *Service) OfferForItem(itemID string) (ShopOffer, bool) {
	for _, offer := range s.DailyOffers(4) {
		if offer.Item.ID == itemID {
			return offer, true
		}
	}
	return ShopOffer{}, false
}

// BuyItem purchases quantity of the item at the given unit price. The balance is
// deducted and the item is granted atomically. unitPrice is the price charged
// per unit (normally the offer's discounted price); it is not derived from the
// item's base price so the displayed price matches the charged price.
func (s *Service) BuyItem(userID int64, itemName string, quantity, unitPrice int) error {
	it := items.Get(itemName)
	if it == nil {
		return ErrNotFound
	}
	totalCost := unitPrice * quantity
	bal, err := s.store.GetBalance(userID)
	if err != nil {
		return err
	}
	if bal < totalCost {
		return ErrNoMoney
	}

	need := quantity
	if it.EquipSlot != "" {
		need = 1
	}
	free, err := s.store.FreeSlots(s.store.DB, userID)
	if err != nil {
		return err
	}
	if free < need {
		return store.ErrInventoryFull
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
			if err := s.store.UpdateBalanceTx(tx, userID, -totalCost); err != nil {
				return err
			}
			_, err := s.store.CreateEquipmentTx(tx, userID, it.ID, it.Name, it.Emoji,
				string(rar), it.EquipSlot, it.MinLevel,
				totalSTR, totalDEX, totalINT, totalVIT, totalLUK,
				affixData, it.SetID)
			return err
		})
	}

	// Non-equipment: add to regular inventory under the canonical item ID.
	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.store.UpdateBalanceTx(tx, userID, -totalCost); err != nil {
			return err
		}
		return s.store.AddItemRaw(tx, userID, it.ID, quantity)
	})
}

func hashSeed(s string) int64 {
	var h int64
	for _, c := range s {
		h = h*31 + int64(c)
	}
	return h
}
