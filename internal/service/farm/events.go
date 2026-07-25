package farm

import (
	"math/rand"
	"time"

	"gorm.io/gorm"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/model"
)

type EventType int

const (
	EventNone EventType = iota
	EventPest
	EventMerchant
	EventBlessing
	EventMysteriousSeed
	EventCropCircles
)

type EventChoice struct {
	Label    string
	CustomID string
	Style    int
}

type Event struct {
	Type        EventType
	Title       string
	Description string
	Choices     []EventChoice
	ZoneKey     string
	PlotIndex   int
	Data        map[string]any
}

type EventResult struct {
	Title       string
	Description string
	ItemGiven   string
	ItemQty     int
	CoinChange  int
	ClearEvent  bool
	BackToMenu  bool
}

func (s *Service) RollEvent(userID int64, zoneKey string, plots []PlotInfo) *Event {
	r := rand.Float64()

	if r < 0.05 {
		return s.rollMerchant(userID, zoneKey)
	}

	if r < 0.10 {
		return s.rollBlessing(userID, zoneKey)
	}

	if r < 0.20 {
		return s.rollCropCircles(userID, zoneKey)
	}

	hasGrowing := false
	var growingPlot PlotInfo
	for _, p := range plots {
		if p.ItemName != "" && !p.Ready {
			hasGrowing = true
			growingPlot = p
			break
		}
	}
	if hasGrowing && r < 0.30 {
		return s.rollPest(userID, zoneKey, growingPlot)
	}

	return nil
}

func (s *Service) rollMerchant(userID int64, zoneKey string) *Event {
	names := []string{"farm.merchant_name_1", "farm.merchant_name_2", "farm.merchant_name_3"}
	merchantName := names[rand.Intn(len(names))]

	exoticSeeds := []string{"star_fruit_seed", "golden_apple_seed", "strawberry_seed"}
	exoticSeed := exoticSeeds[rand.Intn(len(exoticSeeds))]
	price := 100 + rand.Intn(400)

	return &Event{
		Type:        EventMerchant,
		Title:       "farm.event_merchant_title",
		Description: "farm.event_merchant_desc",
		ZoneKey:     zoneKey,
		PlotIndex:   -1,
		Choices: []EventChoice{
			{Label: "farm.event_merchant_sell", CustomID: "farm::event::merchant::sell", Style: 3},
			{Label: "farm.event_merchant_trade", CustomID: "farm::event::merchant::buy::" + exoticSeed + "::" + itoa(price), Style: 1},
			{Label: "farm.event_merchant_leave", CustomID: "farm::event::merchant::leave", Style: 2},
		},
		Data: map[string]any{"merchant": merchantName, "seed": exoticSeed, "price": price},
	}
}

func (s *Service) rollBlessing(userID int64, zoneKey string) *Event {
	chance := 0.05
	level := s.getFarmerLevel(userID)
	chance += float64(level) * 0.005
	if chance > 0.20 {
		chance = 0.20
	}

	if rand.Float64() < chance {
		return &Event{
			Type:        EventBlessing,
			Title:       "farm.event_blessing_title",
			Description: "farm.event_blessing_desc",
			ZoneKey:     zoneKey,
			PlotIndex:   -1,
			Choices: []EventChoice{
				{Label: "farm.event_blessing_accept", CustomID: "farm::event::blessing::accept", Style: 3},
			},
		}
	}
	return nil
}

func (s *Service) rollCropCircles(userID int64, zoneKey string) *Event {
	if rand.Float64() < 0.005 {
		return &Event{
			Type:        EventCropCircles,
			Title:       "farm.event_crop_circles_title",
			Description: "farm.event_crop_circles_desc",
			ZoneKey:     zoneKey,
			PlotIndex:   -1,
			Choices: []EventChoice{
				{Label: "farm.event_crop_circles_accept", CustomID: "farm::event::crop_circles::accept", Style: 3},
			},
		}
	}
	return nil
}

func (s *Service) rollPest(userID int64, zoneKey string, plot PlotInfo) *Event {
	return &Event{
		Type:        EventPest,
		Title:       "farm.event_pest_title",
		Description: "farm.event_pest_desc",
		ZoneKey:     zoneKey,
		PlotIndex:   plot.PlotIndex,
		Choices: []EventChoice{
			{Label: "farm.event_pest_fight", CustomID: "farm::event::pest::fight", Style: 1},
			{Label: "farm.event_pest_pesticide", CustomID: "farm::event::pest::pesticide", Style: 3},
			{Label: "farm.event_pest_ignore", CustomID: "farm::event::pest::ignore", Style: 2},
		},
		Data: map[string]any{"plot": plot.PlotIndex},
	}
}

func (s *Service) ResolveEvent(userID int64, evt *Event, choice string) *EventResult {
	switch evt.Type {
	case EventPest:
		return s.resolvePest(userID, evt, choice)
	case EventMerchant:
		return s.resolveMerchant(userID, evt, choice)
	case EventBlessing:
		return s.resolveBlessing(userID, evt, choice)
	case EventCropCircles:
		return s.resolveCropCircles(userID, evt, choice)
	}
	return &EventResult{
		Title:       "farm.event_default_title",
		Description: "farm.event_default_desc",
		BackToMenu:  true,
	}
}

func (s *Service) resolvePest(userID int64, evt *Event, choice string) *EventResult {
	switch choice {
	case "fight":
		return &EventResult{
			Title:      "farm.event_pest_win_title",
			Description: "farm.event_pest_win_desc",
			BackToMenu: true,
		}

	case "pesticide":
		var inv model.Inventory
		if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, "fertilizer").First(&inv).Error; err == nil && inv.Quantity >= 1 {
			if inv.Quantity <= 1 {
				s.store.DB.Delete(&inv)
			} else {
				s.store.DB.Model(&inv).UpdateColumn("quantity", gorm.Expr("quantity - 1"))
			}
			return &EventResult{
				Title:       "farm.event_pest_pesticide_title",
				Description: "farm.event_pest_pesticide_desc",
				BackToMenu:  true,
			}
		}
		bal, _ := s.store.GetBalance(userID)
		cost := 50
		if bal >= cost {
			s.store.UpdateBalance(userID, -cost)
			return &EventResult{
				Title:       "farm.event_pest_pesticide_title",
				Description: "farm.event_pest_pesticide_desc",
				CoinChange:  -cost,
				BackToMenu:  true,
			}
		}
		return &EventResult{
			Title:       "farm.event_pest_ignore_title",
			Description: "farm.event_pest_ignore_desc",
			BackToMenu:  true,
		}

	case "ignore":
		if rand.Float64() < 0.50 {
			return &EventResult{
				Title:       "farm.event_pest_ignore_survive_title",
				Description: "farm.event_pest_ignore_survive_desc",
				BackToMenu:  true,
			}
		}
		_ = achievement.IncrementStat(s.store.DB, userID, "items_farmed", -1)
		s.store.DB.Where("user_id = ? AND zone_key = ? AND plot_index = ?", userID, evt.ZoneKey, evt.PlotIndex).Delete(&model.UserFarming{})
		return &EventResult{
			Title:       "farm.event_pest_ignore_destroyed_title",
			Description: "farm.event_pest_ignore_destroyed_desc",
			BackToMenu:  true,
		}
	}
	return nil
}

func (s *Service) resolveMerchant(userID int64, evt *Event, choice string) *EventResult {
	switch choice {
	case "sell":
		var readyPlots []model.UserFarming
		s.store.DB.Where("user_id = ?", userID).Find(&readyPlots)
		totalValue := 0
		for _, p := range readyPlots {
			if time.Since(p.PlantTime).Seconds() >= float64(p.GrowTime) {
				crop := cropBySeedName(p.ItemName)
				if crop.Name != "" {
					totalValue += crop.Value * 150 / 100
				}
				s.store.DB.Delete(&p)
			}
		}
		if totalValue > 0 {
			s.store.UpdateBalance(userID, totalValue)
		}
		return &EventResult{
			Title:      "farm.event_merchant_sell_title",
			Description: "farm.event_merchant_sell_desc",
			CoinChange: totalValue,
			BackToMenu: true,
		}

	case "buy":
		seed, _ := evt.Data["seed"].(string)
		priceVal, _ := evt.Data["price"].(int)
		bal, _ := s.store.GetBalance(userID)
		if bal < priceVal {
			return &EventResult{
				Title:       "farm.event_merchant_no_money_title",
				Description: "farm.event_merchant_no_money_desc",
				BackToMenu:  true,
			}
		}
		s.store.UpdateBalance(userID, -priceVal)
		s.store.AddItemRaw(s.store.DB, userID, seed, 1)
		return &EventResult{
			Title:       "farm.event_merchant_bought_title",
			Description: "farm.event_merchant_bought_desc",
			CoinChange:  -priceVal,
			ItemGiven:   seed,
			ItemQty:     1,
			BackToMenu:  true,
		}

	default:
		return &EventResult{
			Title:       "farm.event_merchant_leave_title",
			Description: "farm.event_merchant_leave_desc",
			BackToMenu:  true,
		}
	}
}

func (s *Service) resolveBlessing(userID int64, evt *Event, choice string) *EventResult {
	if choice == "accept" {
		SetBlessing(userID, evt.ZoneKey)
	}
	return &EventResult{
		Title:       "farm.event_blessing_result_title",
		Description: "farm.event_blessing_result_desc",
		BackToMenu:  true,
	}
}

func (s *Service) resolveCropCircles(userID int64, evt *Event, choice string) *EventResult {
	var plots []model.UserFarming
	s.store.DB.Where("user_id = ? AND zone_key = ?", userID, evt.ZoneKey).Find(&plots)
	for _, p := range plots {
		if p.GrowTime > 60 {
			newTime := p.GrowTime / 2
			if newTime < 60 {
				newTime = 60
			}
			s.store.DB.Model(&p).UpdateColumn("grow_time", newTime)
		}
	}
	return &EventResult{
		Title:       "farm.event_crop_circles_result_title",
		Description: "farm.event_crop_circles_result_desc",
		BackToMenu:  true,
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
