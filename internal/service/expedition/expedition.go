package expedition

import (
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
	"time"

	"gorm.io/gorm"

	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
	petsvc "guacagamblebot/internal/service/pets"
	"guacagamblebot/internal/store"
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

var ErrPetKO = errors.New("pet is knocked out")

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) DB() *gorm.DB { return s.store.DB }

// ExpeditionEvent is one entry of the adventure log. Structured fields
// (Location, Enemy, Item, CombatResult) let cogs render a localized log;
// Text is kept as a fallback for rows created before the structured format.
type ExpeditionEvent struct {
	Time         int    `json:"time"`
	Type         string `json:"type"`
	Text         string `json:"text"`
	XP           int    `json:"xp,omitempty"`
	Loot         string `json:"loot,omitempty"`
	Location     string `json:"location,omitempty"`
	Enemy        string `json:"enemy,omitempty"`
	EnemyLevel   int    `json:"enemy_level,omitempty"`
	Item         string `json:"item,omitempty"`
	CombatResult string `json:"combat_result,omitempty"`
}

type ExpeditionResult struct {
	Log   []ExpeditionEvent
	XP    int
	Items []string
	PetHP int
}

func (s *Service) Generate(pet *model.UserPet, durationHours int) *ExpeditionResult {
	commonLoot := []string{"pebble", "coal", "sardine", "wheat", "tomato", "wheat_seed", "carrot_seed", "worm"}
	rareLoot := []string{"iron_ore", "salmon", "corn", "strawberry", "potato_seed", "tomato_seed", "pumpkin_seed", "crayfish"}
	epicLoot := []string{"gold_nugget", "shark", "star_fruit", "emerald", "coffee_seed", "cocoa_seed", "strawberry_seed", "golden_lure",
		"miner_helmet", "hunters_bow", "golden_ring", "ancient_amulet"}
	locations := []string{"forest", "desert", "cave", "plains", "mountain", "swamp", "valley", "coral", "volcano"}

	numEvents := durationHours * 2
	if durationHours == 1 {
		numEvents = 3
	}
	totalXP := durationHours * 25
	events := make([]ExpeditionEvent, 0, numEvents)
	items := make([]string, 0)
	petHP := pet.HP

	petBonded := charsvc.ConsumeBuff(s.store, pet.UserID, "pet_bond")
	bulwarked := charsvc.ConsumeBuff(s.store, pet.UserID, "bulwark")
	if bulwarked {
		petHP = pet.MaxHP
	}

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
			xp := (10 + rand.Intn(21)) * pet.Level
			totalXP += xp
			ev.Location = loc
			ev.XP = xp
			ev.Text = "🐾 **" + pet.Nickname + "** explores " + loc + " and gains **" + itoa(xp) + " XP**."
		case "combat":
			if petHP <= 0 {
				ev.Text = "😵 **" + pet.Nickname + "** is K.O. and cannot fight for now."
				break
			}
			enemySpecies := randomPetSpecies()
			enemyLvl := max(1, pet.Level-2+rand.Intn(5))
			ev.Enemy = enemySpecies
			ev.EnemyLevel = enemyLvl

			petBP := s.petBattlePet(pet)
			petBP.HP = petHP
			if petBonded {
				petBP.Atk = petBP.Atk * 5 / 4
				petBP.Defense = petBP.Defense * 5 / 4
				petBP.Speed = petBP.Speed * 5 / 4
				petBP.DGE = petBP.DGE * 5 / 4
				petBP.ACC = petBP.ACC * 5 / 4
				petBP.CritC = petBP.CritC * 5 / 4
			}
			enemyBP := wildBattlePet(enemySpecies, enemyLvl)

			battle.SimulatePreserveHP(petBP, enemyBP)
			if bulwarked && petBP.HP < petHP {
				petBP.HP = petHP - (petHP-petBP.HP)/2
			}
			petHP = petBP.HP

			switch {
			case petBP.IsAlive() && !enemyBP.IsAlive():
				xp := (40 + rand.Intn(41)) * enemyLvl
				totalXP += xp
				ev.CombatResult = "win"
				ev.XP = xp
				ev.Text = "⚔️ **" + pet.Nickname + "** defeated a wild **" + enemySpecies + "** (Lvl " + itoa(enemyLvl) + ")! (+" + itoa(xp) + " XP)"
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
			case !petBP.IsAlive():
				ev.CombatResult = "loss"
				ev.Text = "💀 **" + pet.Nickname + "** was knocked out by a wild **" + enemySpecies + "** (Lvl " + itoa(enemyLvl) + ")!"
				totalXP += 10
			default:
				ev.CombatResult = "stalemate"
				ev.Text = "🤕 **" + pet.Nickname + "** fought a wild **" + enemySpecies + "** (Lvl " + itoa(enemyLvl) + ") to a stalemate and withdrew."
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
			ev.Item = item
			ev.Loot = item
			ev.Text = "🎁 **" + pet.Nickname + "** found a **" + item + "**!"
			ev.Loot = item
		case "rest":
			ev.Text = "💤 **" + pet.Nickname + "** takes a short nap by a stream."
		}

		events = append(events, ev)
	}

	return &ExpeditionResult{Log: events, XP: totalXP, Items: items, PetHP: petHP}
}

// petBattlePet converts a pet into its battle form with its learned skills.
func (s *Service) petBattlePet(pet *model.UserPet) *battle.BattlePet {
	emoji := "🐾"
	if pt := petsvc.PetTypes[pet.PetType]; pt != nil {
		emoji = pt.Emoji
	}
	var skills []model.UserPetSkill
	s.store.DB.Where("pet_id = ?", pet.ID).Find(&skills)
	skillIDs := make([]string, 0, len(skills))
	for _, sk := range skills {
		skillIDs = append(skillIDs, sk.SkillID)
	}
	return &battle.BattlePet{
		ID: pet.ID, Nickname: pet.Nickname, Emoji: emoji, PetType: pet.PetType,
		Level: pet.Level, HP: pet.HP, MaxHP: pet.MaxHP,
		Atk: pet.Atk, Defense: pet.Defense, Speed: pet.Speed,
		DGE: pet.DGE, ACC: pet.ACC, CritC: pet.CritC, CritD: pet.CritD, SpcC: pet.SpcC,
		Skills: skillIDs,
	}
}

// wildBattlePet builds a wild opponent of the given species with stats scaled
// to its level using the same growth curve as player pets.
func wildBattlePet(species string, level int) *battle.BattlePet {
	pt := petsvc.PetTypes[species]
	if pt == nil {
		pt = petsvc.PetTypes["Escargot"]
	}
	lvl := max(1, level)
	spc := 0
	if lvl >= 5 {
		spc = (lvl / 5) * 5
		if spc > 50 {
			spc = 50
		}
	}
	maxHP := pt.MaxHP + 2*(lvl-1)
	return &battle.BattlePet{
		ID: -1, Nickname: species, Emoji: pt.Emoji, PetType: species,
		Level: lvl, HP: maxHP, MaxHP: maxHP,
		Atk: pt.Atk + (lvl - 1), Defense: pt.Defense + (lvl-1)/2,
		Speed: pt.Speed + (lvl-1)/5, DGE: pt.DGE + (lvl-1)/5,
		ACC: pt.ACC + (lvl-1)/5, CritC: pt.CritC, CritD: pt.CritD, SpcC: spc,
	}
}

func (s *Service) Start(userID, petID int64, durationHours int, result *ExpeditionResult) (*model.PetExpedition, error) {
	var pet model.UserPet
	if err := s.store.DB.First(&pet, petID).Error; err != nil {
		return nil, err
	}
	if pet.HP <= 0 {
		return nil, ErrPetKO
	}
	free, err := s.store.FreeSlots(s.store.DB, userID)
	if err != nil {
		return nil, err
	}
	if free <= 0 {
		return nil, store.ErrInventoryFull
	}
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
	err = s.store.DB.Create(&exp).Error
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"on_expedition": true}
	if result.PetHP >= 0 {
		updates["hp"] = result.PetHP
	}
	s.store.DB.Model(&model.UserPet{}).Where("id = ?", petID).Updates(updates)
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

func (s *Service) Claim(exp *model.PetExpedition) (leveled bool, lvl int, err error) {
	leveled, lvl = charsvc.AddXP(s.store, exp.UserID, exp.RewardXP/2)

	// Hand out the loot the pet brought back: gear becomes equipment
	// instances, everything else lands in the regular inventory.
	var rawItems []string
	_ = json.Unmarshal([]byte(exp.RewardItems), &rawItems)
	for _, id := range rawItems {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		it := items.Get(id)
		if it != nil && it.EquipSlot != "" {
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
			if _, gerr := s.store.CreateEquipmentFromAffixes(exp.UserID, it.ID, it.Name, it.Emoji,
				string(rar), it.EquipSlot, it.MinLevel,
				it.StatSTR, it.StatDEX, it.StatINT, it.StatVIT, it.StatLUK,
				applied, it.SetID); gerr != nil {
				continue
			}
		} else if gerr := s.store.AddItemRaw(s.store.DB, exp.UserID, id, 1); gerr != nil {
			continue
		}
	}

	err = s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.PetExpedition{}).
			Where("id = ?", exp.ID).
			Update("is_claimed", true).Error; err != nil {
			return err
		}
		return tx.Model(&model.UserPet{}).
			Where("id = ?", exp.PetID).
			Update("on_expedition", false).Error
	})
	return
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
