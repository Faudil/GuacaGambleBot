package hunt

import (
	"errors"
	"math/rand"

	"gorm.io/gorm"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type EnemyTemplate struct {
	Name  string
	Emoji string
	HP    int
	Atk   int
	Def   int
	Spd   int
}

type LootEntry struct {
	Item   string
	Chance float64
	MaxQty int
}

type Zone struct {
	Key         string
	Emoji       string
	LevelMin    int
	LevelMax    int
	XPMult      float64
	Enemies     []EnemyTemplate
	LootTable   []LootEntry
}

var Zones = map[string]Zone{
	"easy": {
		Key: "easy_zone", Emoji: "🌲", LevelMin: 1, LevelMax: 5, XPMult: 1.0,
		Enemies: []EnemyTemplate{
			{Name: "Slime Gluant", Emoji: "💧", HP: 25, Atk: 5, Def: 2, Spd: 5},
			{Name: "Sanglier Sauvage", Emoji: "🐗", HP: 35, Atk: 8, Def: 5, Spd: 10},
		},
		LootTable: []LootEntry{
			{Item: "pebble", Chance: 0.50, MaxQty: 3},
			{Item: "tomato", Chance: 0.30, MaxQty: 2},
			{Item: "coal", Chance: 0.15, MaxQty: 1},
		},
	},
	"medium": {
		Key: "medium_zone", Emoji: "🦇", LevelMin: 8, LevelMax: 12, XPMult: 2.5,
		Enemies: []EnemyTemplate{
			{Name: "Gobelin Voleur", Emoji: "👺", HP: 40, Atk: 18, Def: 5, Spd: 25},
			{Name: "Araignée Géante", Emoji: "🕷️", HP: 50, Atk: 15, Def: 8, Spd: 30},
		},
		LootTable: []LootEntry{
			{Item: "coal", Chance: 0.60, MaxQty: 3},
			{Item: "iron_ore", Chance: 0.40, MaxQty: 2},
			{Item: "sardine", Chance: 0.20, MaxQty: 1},
		},
	},
	"hard": {
		Key: "hard_zone", Emoji: "🌋", LevelMin: 15, LevelMax: 20, XPMult: 5.0,
		Enemies: []EnemyTemplate{
			{Name: "Golem de Magma", Emoji: "🗿", HP: 100, Atk: 25, Def: 20, Spd: 2},
			{Name: "Drake de Feu", Emoji: "🐉", HP: 80, Atk: 35, Def: 12, Spd: 25},
		},
		LootTable: []LootEntry{
			{Item: "copper_ore", Chance: 0.50, MaxQty: 5},
			{Item: "gold_nugget", Chance: 0.30, MaxQty: 3},
			{Item: "rough_diamond", Chance: 0.20, MaxQty: 2},
		},
	},
}

type Combatant struct {
	Name    string
	Emoji   string
	HP      int
	MaxHP   int
	Atk     int
	Def     int
	Level   int
	IsAlive bool
}

func NewEnemy(zoneKey string) *Combatant {
	zone := Zones[zoneKey]
	t := zone.Enemies[rand.Intn(len(zone.Enemies))]
	lvl := zone.LevelMin + rand.Intn(zone.LevelMax-zone.LevelMin+1)
	return &Combatant{
		Name:    t.Name,
		Emoji:   t.Emoji,
		HP:      t.HP + lvl*5,
		MaxHP:   t.HP + lvl*5,
		Atk:     t.Atk + lvl*2,
		Def:     t.Def + lvl*1,
		Level:   lvl,
		IsAlive: true,
	}
}

type BattleLogEntry struct {
	Text string
}

type BattleResult struct {
	PetHP      int
	PetMaxHP   int
	EnemyHP    int
	EnemyMaxHP int
	PlayerWon  bool
	EnemyWon   bool
	Log        []BattleLogEntry
	XP         int
	Loot       []string
	LeveledUp  bool
	NewLevel   int
}

var ErrNoPet = errors.New("no active pet")
var ErrPetKO = errors.New("pet is KO")

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) ExecuteHunt(userID int64, zoneKey string) (*BattleResult, error) {
	zone, ok := Zones[zoneKey]
	if !ok {
		return nil, errors.New("invalid zone")
	}

	var pet model.UserPet
	if err := s.store.DB.Where("user_id = ? AND is_active = ?", userID, true).First(&pet).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNoPet
		}
		return nil, err
	}

	if pet.HP <= 0 {
		return nil, ErrPetKO
	}

	enemy := NewEnemy(zoneKey)

	petHP := pet.HP
	petMaxHP := pet.MaxHP
	petAtk := pet.Atk
	petDef := pet.Defense

	var log []BattleLogEntry

	for petHP > 0 && enemy.HP > 0 {
		petDmg := petAtk - enemy.Def
		if petDmg < 1 {
			petDmg = 1
		}
		if rand.Intn(100) < pet.CritC {
			petDmg = int(float64(petDmg) * pet.CritD)
		}
		enemy.HP -= petDmg
		if enemy.HP < 0 {
			enemy.HP = 0
		}
		log = append(log, BattleLogEntry{Text: petDmgMsg(pet.Nickname, petDmg)})

		if enemy.HP <= 0 {
			enemy.IsAlive = false
			break
		}

		enemyDmg := enemy.Atk - petDef
		if enemyDmg < 1 {
			enemyDmg = 1
		}
		petHP -= enemyDmg
		if petHP < 0 {
			petHP = 0
		}
		log = append(log, BattleLogEntry{Text: enemyDmgMsg(enemy.Name, enemyDmg)})
	}

	playerWon := enemy.HP <= 0
	enemyWon := petHP <= 0

	var xp int
	var lootItems []string
	leveledUp := false
	newLevel := pet.Level

	if playerWon {
		baseXP := enemy.Level * (15 + rand.Intn(11))
		xp = int(float64(baseXP) * zone.XPMult)

		if err := achievement.IncrementStat(s.store.DB, userID, "pve_wins", 1); err != nil {
			return nil, err
		}

		for _, loot := range zone.LootTable {
			if rand.Float64() < loot.Chance {
				qty := rand.Intn(loot.MaxQty) + 1
				for i := 0; i < qty; i++ {
					if err := s.store.DB.Exec(
						`INSERT INTO inventory (user_id, item_id, quantity) VALUES (?, ?, 1)
						 ON CONFLICT(user_id, item_id) DO UPDATE SET quantity = quantity + 1`,
						userID, loot.Item,
					).Error; err == nil {
						lootItems = append(lootItems, loot.Item)
					}
				}
			}
		}
	} else if enemyWon {
		baseXP := enemy.Level * (15 + rand.Intn(11)) / 10
		xp = baseXP
	} else {
		baseXP := enemy.Level * (15 + rand.Intn(11)) / 2
		xp = int(float64(baseXP) * zone.XPMult)
	}

	pet.XP += xp
	next := 100 + pet.Level*50
	if pet.XP >= next {
		pet.XP -= next
		pet.Level++
		pet.MaxHP += 10
		pet.HP = petHP + 10
		if pet.HP > pet.MaxHP {
			pet.HP = pet.MaxHP
		}
		pet.Atk += 2
		pet.Defense += 1
		leveledUp = true
		newLevel = pet.Level

		s.store.DB.Model(&pet).Updates(map[string]any{
			"xp": pet.XP, "level": pet.Level, "max_hp": pet.MaxHP,
			"hp": pet.HP, "atk": pet.Atk, "defense": pet.Defense,
		})
	} else {
		s.store.DB.Model(&pet).Updates(map[string]any{
			"xp": pet.XP, "hp": petHP,
		})
	}

	unlocks, err := achievement.CheckAndUnlock(s.store.DB, userID)
	if err != nil {
		return nil, err
	}
	_ = unlocks

	return &BattleResult{
		PetHP:      petHP,
		PetMaxHP:   petMaxHP,
		EnemyHP:    enemy.HP,
		EnemyMaxHP: enemy.MaxHP,
		PlayerWon:  playerWon,
		EnemyWon:   enemyWon,
		Log:        log,
		XP:         xp,
		Loot:       lootItems,
		LeveledUp:  leveledUp,
		NewLevel:   newLevel,
	}, nil
}

func petDmgMsg(name string, dmg int) string {
	return name + " deals " + itoa(dmg) + " damage!"
}

func enemyDmgMsg(name string, dmg int) string {
	return name + " deals " + itoa(dmg) + " damage!"
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
