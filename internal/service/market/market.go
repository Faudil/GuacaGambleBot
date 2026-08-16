package market

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
	"guacagamblebot/internal/store"
)

const (
	RotationSize   = 20
	MaxPerCategory = 5
	ItemsPerPage   = 5

	SellImpact = 0.01
	BuyImpact  = 0.007

	// BuySpread is the fraction above the quoted price that buyers pay for each
	// unit; SellSpread is the fraction below it that sellers receive. The
	// combined round-trip cost prevents pump-and-dump flipping.
	BuySpread  = 0.02
	SellSpread = 0.02

	DailyDecayRate = 0.10
	PriceFloorMult = 0.20
	PriceCeilMult  = 5.0

	// VendorSellMult is the fraction of the base price paid for items that
	// are not part of the weekly rotation.
	VendorSellMult = 0.50
)

var (
	ErrNotActive   = errors.New("item not in active rotation")
	ErrNotSellable = errors.New("item cannot be sold")
	ErrNotFound    = errors.New("item not found")
	ErrNoItem      = errors.New("item not owned")
	ErrNoMoney     = errors.New("insufficient funds")
	ErrInvalidQty  = errors.New("quantity must be positive")
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

type MarketItemView struct {
	Item         *items.Item
	CurrentPrice int
	BasePrice    int
	TrendPercent int
}

// PlayerSellItemView describes one of the player's sellable inventory items as
// shown in the market's sell view: the unit price (dynamic market price when
// the item is in the weekly rotation, the fixed vendor rate otherwise) and the
// quantity owned.
type PlayerSellItemView struct {
	Item       *items.Item
	UnitPrice  int
	Owned      int
	InRotation bool
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func currentWeekID() string {
	y, w := time.Now().ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

func (s *Service) ensureWeekRotation() error {
	weekID := currentWeekID()
	var count int64
	s.store.DB.Model(&model.MarketState{}).Where("week_id = ? AND is_active = ?", weekID, true).Count(&count)
	if count > 0 {
		return nil
	}
	return s.rotateMarket(weekID)
}

func (s *Service) rotateMarket(weekID string) error {
	all := items.MarketableItems()
	if len(all) == 0 {
		return nil
	}

	rng := rand.New(rand.NewSource(hashSeed(weekID)))
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	var selected []items.Item
	catCount := make(map[items.Category]int)
	for _, it := range all {
		if len(selected) >= RotationSize {
			break
		}
		if catCount[it.Category] >= MaxPerCategory {
			continue
		}
		selected = append(selected, it)
		catCount[it.Category]++
	}

	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.MarketState{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
			return err
		}
		today := time.Now().Format("2006-01-02")
		for _, it := range selected {
			tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "item_id"}},
				DoUpdates: clause.Assignments(map[string]any{
					"current_price": it.Price,
					"daily_sold":    0,
					"daily_bought":  0,
					"last_reset":    today,
					"week_id":       weekID,
					"is_active":     true,
				}),
			}).Create(&model.MarketState{
				ItemID:       it.ID,
				CurrentPrice: it.Price,
				LastReset:    today,
				WeekID:       weekID,
				IsActive:     true,
			})
		}
		return nil
	})
}

func (s *Service) ensureDayReset() error {
	today := time.Now().Format("2006-01-02")
	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		var states []model.MarketState
		if err := tx.Where("is_active = ? AND last_reset != ?", true, today).Find(&states).Error; err != nil {
			return err
		}
		for _, st := range states {
			it := items.Get(st.ItemID)
			if it == nil {
				continue
			}
			price := st.CurrentPrice
			gap := float64(it.Price - price)
			if math.Abs(gap) > 1 {
				price += int(math.Ceil(gap * DailyDecayRate))
			}
			price = clampInt(price, maxInt(1, int(float64(it.Price)*PriceFloorMult)),
				maxInt(2, int(float64(it.Price)*PriceCeilMult)))

			tx.Model(&model.MarketState{}).Where("item_id = ?", st.ItemID).Updates(map[string]any{
				"current_price": price,
				"daily_sold":    0,
				"daily_bought":  0,
				"last_reset":    today,
			})
		}
		return nil
	})
}

func (s *Service) GetMarket(category string, page, pageSize int) ([]MarketItemView, int, error) {
	if err := s.ensureWeekRotation(); err != nil {
		return nil, 0, err
	}
	if err := s.ensureDayReset(); err != nil {
		return nil, 0, err
	}

	query := s.store.DB.Model(&model.MarketState{}).Where("is_active = ?", true)
	countQuery := s.store.DB.Model(&model.MarketState{}).Where("is_active = ?", true)

	if category != "" && category != "all" {
		catItems := items.ItemsByCategory(items.Category(category))
		ids := make([]string, 0, len(catItems))
		for _, it := range catItems {
			if it.IsMarketable() {
				ids = append(ids, it.ID)
			}
		}
		query = query.Where("item_id IN ?", ids)
		countQuery = countQuery.Where("item_id IN ?", ids)
	}

	var total int64
	countQuery.Count(&total)

	var states []model.MarketState
	if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&states).Error; err != nil {
		return nil, 0, err
	}

	views := make([]MarketItemView, 0, len(states))
	for _, st := range states {
		it := items.Get(st.ItemID)
		if it == nil {
			continue
		}
		trend := 0
		if it.Price > 0 {
			trend = ((st.CurrentPrice - it.Price) * 100) / it.Price
		}
		views = append(views, MarketItemView{
			Item:         it,
			CurrentPrice: st.CurrentPrice,
			BasePrice:    it.Price,
			TrendPercent: trend,
		})
	}
	return views, int(total), nil
}

// GetPlayerSellItems lists the player's owned, sellable items with their unit
// sell price — the dynamic market price when the item is in the weekly
// rotation, the fixed vendor rate otherwise. Rotation items come first, then
// items sorted by unit price descending. Items that cannot be sold are
// excluded.
func (s *Service) GetPlayerSellItems(userID int64, page, pageSize int) ([]PlayerSellItemView, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = ItemsPerPage
	}
	if err := s.ensureWeekRotation(); err != nil {
		return nil, 0, err
	}
	if err := s.ensureDayReset(); err != nil {
		return nil, 0, err
	}

	var invs []model.Inventory
	if err := s.store.DB.Where("user_id = ? AND quantity > 0", userID).Find(&invs).Error; err != nil {
		return nil, 0, err
	}

	var states []model.MarketState
	if err := s.store.DB.Where("is_active = ?", true).Find(&states).Error; err != nil {
		return nil, 0, err
	}
	rotPrice := make(map[string]int, len(states))
	for _, st := range states {
		rotPrice[st.ItemID] = st.CurrentPrice
	}

	var views []PlayerSellItemView
	for _, inv := range invs {
		it := items.Get(inv.ItemID)
		if it == nil || !it.IsSellable() {
			continue
		}
		price, inRotation := rotPrice[inv.ItemID]
		if !inRotation {
			price = vendorPrice(it.Price)
		}
		views = append(views, PlayerSellItemView{
			Item:       it,
			UnitPrice:  price,
			Owned:      inv.Quantity,
			InRotation: inRotation,
		})
	}

	sort.SliceStable(views, func(i, j int) bool {
		if views[i].InRotation != views[j].InRotation {
			return views[i].InRotation
		}
		return views[i].UnitPrice > views[j].UnitPrice
	})

	total := len(views)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return views[start:end], total, nil
}

func (s *Service) BuyItem(userID int64, itemID string, amount int) (int, bool, int, error) {
	if amount <= 0 {
		return 0, false, 0, ErrInvalidQty
	}
	if err := s.ensureWeekRotation(); err != nil {
		return 0, false, 0, err
	}
	if err := s.ensureDayReset(); err != nil {
		return 0, false, 0, err
	}

	it := items.Get(itemID)
	if it == nil {
		return 0, false, 0, ErrNotFound
	}

	var st model.MarketState
	if err := s.store.DB.Where("item_id = ? AND is_active = ?", itemID, true).First(&st).Error; err != nil {
		return 0, false, 0, ErrNotActive
	}

	buyStep := maxInt(1, int(float64(it.Price)*BuyImpact))
	minP := maxInt(1, int(float64(it.Price)*PriceFloorMult))
	maxP := maxInt(minP+1, int(float64(it.Price)*PriceCeilMult))
	totalCost, newPrice := ladderedBuy(st.CurrentPrice, amount, buyStep, minP, maxP)

	bal, err := s.store.GetBalance(userID)
	if err != nil {
		return 0, false, 0, err
	}
	if bal < totalCost {
		return 0, false, 0, ErrNoMoney
	}

	err = s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("user_id = ?", userID).
			UpdateColumn("balance", gorm.Expr("balance - ?", totalCost)).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
			DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + ?", amount)}),
		}).Create(&model.Inventory{UserID: userID, ItemID: itemID, Quantity: amount}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.MarketState{}).
			Where("item_id = ?", itemID).
			Updates(map[string]any{
				"current_price": newPrice,
				"daily_bought":  gorm.Expr("daily_bought + ?", amount),
			}).Error; err != nil {
			return err
		}
		if err := achievement.IncrementStat(tx, userID, "items_bought_market", amount); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, false, 0, err
	}
	leveled, lvl := charsvc.AddXP(s.store, userID, amount)
	return totalCost, leveled, lvl, nil
}

func (s *Service) SellItem(userID int64, itemID string, amount int) (int, bool, int, error) {
	if amount <= 0 {
		return 0, false, 0, ErrInvalidQty
	}
	if err := s.ensureWeekRotation(); err != nil {
		return 0, false, 0, err
	}
	if err := s.ensureDayReset(); err != nil {
		return 0, false, 0, err
	}

	it := items.Get(itemID)
	if it == nil {
		return 0, false, 0, ErrNotFound
	}
	if it.Price <= 0 {
		return 0, false, 0, ErrNotSellable
	}

	// Items in the active rotation sell at the dynamic market price;
	// everything else sells at the fixed vendor rate.
	var st model.MarketState
	inRotation := s.store.DB.Where("item_id = ? AND is_active = ?", itemID, true).First(&st).Error == nil
	unitPrice := vendorPrice(it.Price)
	if inRotation {
		unitPrice = st.CurrentPrice
	}

	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, itemID).First(&inv).Error; err != nil {
		return 0, false, 0, ErrNoItem
	}
	if inv.Quantity < amount {
		return 0, false, 0, ErrNoItem
	}

	newPrice := 0
	totalGain := unitPrice * amount
	if inRotation {
		sellStep := maxInt(1, int(float64(it.Price)*SellImpact))
		minP := maxInt(1, int(float64(it.Price)*PriceFloorMult))
		maxP := maxInt(minP+1, int(float64(it.Price)*PriceCeilMult))
		totalGain, newPrice = ladderedSell(st.CurrentPrice, amount, sellStep, minP, maxP)
	}
	if charsvc.HasPassive(s.store, userID, "perk_trader") {
		totalGain = totalGain * 105 / 100
	}
	if charsvc.HasBuff(s.store, userID, "golden_touch") {
		totalGain *= 2
		charsvc.ConsumeBuff(s.store, userID, "golden_touch")
	}
	if charsvc.HasBuff(s.store, userID, "insider_trading") {
		totalGain = totalGain * 15 / 10
		charsvc.ConsumeBuff(s.store, userID, "insider_trading")
	}

	// Ensure user row exists before the transaction
	if _, err := s.store.GetBalance(userID); err != nil {
		return 0, false, 0, err
	}

	err := s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Inventory{}).
			Where("user_id = ? AND item_id = ?", userID, itemID).
			UpdateColumn("quantity", gorm.Expr("quantity - ?", amount)).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("user_id = ?", userID).
			UpdateColumn("balance", gorm.Expr("balance + ?", totalGain)).Error; err != nil {
			return err
		}
		if inRotation {
			if err := tx.Model(&model.MarketState{}).
				Where("item_id = ?", itemID).
				Updates(map[string]any{
					"current_price": newPrice,
					"daily_sold":    gorm.Expr("daily_sold + ?", amount),
				}).Error; err != nil {
				return err
			}
		}
		if err := achievement.IncrementStat(tx, userID, "items_sold_market", amount); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, false, 0, err
	}
	_ = s.store.RecordActivity(userID, "items_sold_market", 1)
	leveled, lvl := charsvc.AddXP(s.store, userID, amount)
	return totalGain, leveled, lvl, nil
}

// SellPricesFor returns the unit sell price of each item: the dynamic market
// price for items in the active rotation, or the fixed vendor rate otherwise.
func (s *Service) SellPricesFor(itemIDs []string) map[string]int {
	prices := make(map[string]int, len(itemIDs))
	if len(itemIDs) == 0 {
		return prices
	}
	if err := s.ensureWeekRotation(); err != nil {
		return prices
	}
	if err := s.ensureDayReset(); err != nil {
		return prices
	}

	var states []model.MarketState
	if err := s.store.DB.Where("is_active = ? AND item_id IN ?", true, itemIDs).Find(&states).Error; err != nil {
		return prices
	}
	inRotation := make(map[string]bool, len(states))
	for _, st := range states {
		inRotation[st.ItemID] = true
		prices[st.ItemID] = st.CurrentPrice
	}
	for _, id := range itemIDs {
		if inRotation[id] {
			continue
		}
		it := items.Get(id)
		if it != nil && it.Price > 0 {
			prices[id] = vendorPrice(it.Price)
		}
	}
	return prices
}

// vendorPrice is the fixed amount the vendor pays for an item outside the
// weekly rotation.
func vendorPrice(base int) int {
	return maxInt(1, int(float64(base)*VendorSellMult))
}

// buyBid is the amount a buyer pays for one unit at the quoted price.
func buyBid(price int) int {
	return price + maxInt(1, int(math.Ceil(float64(price)*BuySpread)))
}

// sellAsk is the amount a seller receives for one unit at the quoted price.
func sellAsk(price int) int {
	return maxInt(1, price-maxInt(1, int(math.Ceil(float64(price)*SellSpread))))
}

// ladderedBuy simulates buying amount units one at a time so the price moves
// during the trade instead of a single jump at the end. It returns the total
// cost (the sum of the bid paid for each unit) and the final market price.
func ladderedBuy(price, amount, step, minP, maxP int) (cost, final int) {
	final = price
	for i := 0; i < amount; i++ {
		cost += buyBid(final)
		final = clampInt(final+step, minP, maxP)
	}
	return cost, final
}

// ladderedSell simulates selling amount units one at a time so the price moves
// during the trade. It returns the total gain (the sum of the ask received for
// each unit) and the final market price.
func ladderedSell(price, amount, step, minP, maxP int) (gain, final int) {
	final = price
	for i := 0; i < amount; i++ {
		gain += sellAsk(final)
		final = clampInt(final-step, minP, maxP)
	}
	return gain, final
}

func hashSeed(s string) int64 {
	var h int64
	for _, c := range s {
		h = h*31 + int64(c)
	}
	return h
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
