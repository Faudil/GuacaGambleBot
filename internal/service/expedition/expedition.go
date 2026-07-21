package expedition

import (
	"encoding/json"
	"math/rand"
	"time"

	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
	"guacagamblebot/internal/store"
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) DB() *gorm.DB { return s.store.DB }

type ExpeditionEvent struct {
	Time int    `json:"time"`
	Type string `json:"type"`
	Text string `json:"text"`
	XP   int    `json:"xp,omitempty"`
	Loot string `json:"loot,omitempty"`
}

type ExpeditionResult struct {
	Log   []ExpeditionEvent
	XP    int
	Items []string
}

func (s *Service) Generate(petType string, petLevel int, durationHours int, lang string) *ExpeditionResult {
	commonLoot := []string{"pebble", "coal", "sardine", "wheat", "tomato"}
	rareLoot := []string{"iron_ore", "salmon", "corn", "strawberry"}
	epicLoot := []string{"gold_nugget", "shark", "star_fruit", "emerald"}
	locations := []string{"forest", "desert", "cave", "plains", "mountain", "swamp", "valley", "coral", "volcano"}

	numEvents := durationHours * 2
	if durationHours == 1 {
		numEvents = 3
	}
	totalXP := durationHours * 25
	events := make([]ExpeditionEvent, 0, numEvents)
	items := make([]string, 0)

	for i := 0; i < numEvents; i++ {
		eventTime := int(float64(i+1) * float64(durationHours*60) / float64(numEvents+1))

		roll := rand.Float64() * 100
		var eType string
		switch {
		case roll < 40:
			eType = "exploration"
		case roll < 70:
			eType = "combat"
		case roll < 90:
			eType = "loot"
		default:
			eType = "rest"
		}

		ev := ExpeditionEvent{Time: eventTime, Type: eType}

		switch eType {
		case "exploration":
			loc := locations[rand.Intn(len(locations))]
			xp := (10 + rand.Intn(21)) * petLevel
			totalXP += xp
			ev.Text = "🐾 **" + petType + "** explores " + loc + " and gains **" + itoa(xp) + " XP**."
			ev.XP = xp
		case "combat":
			enemySpecies := randomPetSpecies()
			enemyLvl := max(1, petLevel-2+rand.Intn(5))
			winChance := 0.6 + float64(petLevel-enemyLvl)*0.05
			if winChance < 0.2 {
				winChance = 0.2
			} else if winChance > 0.95 {
				winChance = 0.95
			}
			if rand.Float64() < winChance {
				xp := (40 + rand.Intn(41)) * enemyLvl
				totalXP += xp
				ev.Text = "⚔️ **" + petType + "** defeated a wild **" + enemySpecies + "**! (+" + itoa(xp) + " XP)"
				ev.XP = xp
				if rand.Float64() < 0.3 {
					var item string
					if rand.Float64() < 0.8 {
						item = commonLoot[rand.Intn(len(commonLoot))]
					} else {
						item = rareLoot[rand.Intn(len(rareLoot))]
					}
					items = append(items, item)
					ev.Loot = item
				}
			} else {
				ev.Text = "🤕 **" + petType + "** encountered a wild **" + enemySpecies + "** and had to flee..."
				totalXP += 10
			}
		case "loot":
			roll2 := rand.Float64()
			var item string
			switch {
			case roll2 < 0.05:
				item = epicLoot[rand.Intn(len(epicLoot))]
			case roll2 < 0.25:
				item = rareLoot[rand.Intn(len(rareLoot))]
			default:
				item = commonLoot[rand.Intn(len(commonLoot))]
			}
			items = append(items, item)
			ev.Text = "🎁 **" + petType + "** found a **" + item + "**!"
			ev.Loot = item
		case "rest":
			ev.Text = "💤 **" + petType + "** takes a short nap by a stream."
		}

		events = append(events, ev)
	}

	return &ExpeditionResult{Log: events, XP: totalXP, Items: items}
}

func (s *Service) Start(userID, petID int64, durationHours int, result *ExpeditionResult) (*model.PetExpedition, error) {
	now := time.Now()
	duration := time.Duration(durationHours) * time.Hour

	logJSON, _ := json.Marshal(result.Log)
	itemsJSON, _ := json.Marshal(result.Items)

	exp := model.PetExpedition{
		UserID:      userID,
		PetID:       petID,
		StartTime:   now,
		EndTime:     now.Add(duration),
		RewardXP:    result.XP,
		RewardItems: string(itemsJSON),
		Log:         string(logJSON),
		IsClaimed:   false,
	}
	err := s.store.DB.Create(&exp).Error
	if err != nil {
		return nil, err
	}
	s.store.DB.Model(&model.UserPet{}).Where("id = ?", petID).Update("on_expedition", true)
	return &exp, nil
}

func (s *Service) GetActive(userID int64) (*model.PetExpedition, error) {
	var exp model.PetExpedition
	err := s.store.DB.Where("user_id = ? AND is_claimed = ?", userID, false).First(&exp).Error
	if err != nil {
		return nil, err
	}
	return &exp, nil
}

func (s *Service) Claim(exp *model.PetExpedition) error {
	charsvc.AddXP(s.store, exp.UserID, exp.RewardXP/2)
	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.PetExpedition{}).
			Where("id = ?", exp.ID).
			Update("is_claimed", true).Error; err != nil {
			return err
		}
		return tx.Model(&model.UserPet{}).
			Where("id = ?", exp.PetID).
			Update("on_expedition", false).Error
	})
}

var petSpecies = []string{
	"Escargot", "Souris", "Cochon", "Grenouille", "Taupe", "Pélican", "Mouton", "Abeille",
	"Chien", "Chat", "Cheval", "Renard", "Singe", "Ours",
	"Chameau", "Panda", "Tigre", "Pieuvre", "Dragon",
	"Hamster", "Fourmi", "Hérisson", "Canard", "Chouette", "Paresseux",
	"Kangourou", "Iguane", "Gorille", "Scorpion", "Bison",
	"Aigle", "Rhino", "Crocodile", "Putois", "Dauphin", "Léopard", "Lion", "Ours polaire",
	"Tyrannosaure", "Diplodocus", "Mamouth", "Mégalodon", "Kraken", "Licorne", "Phoenix",
	"Cerbère", "Fenrir", "Ratatosk", "Nidhögg", "Bedawang",
}

func randomPetSpecies() string {
	return petSpecies[rand.Intn(len(petSpecies))]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	if neg {
		out = "-" + out
	}
	return out
}
