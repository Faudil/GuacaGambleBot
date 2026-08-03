package hunt

import (
	"errors"
	"math/rand"
	"time"

	"gorm.io/gorm"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
	"guacagamblebot/internal/store"
)

var ErrHuntLimit = errors.New("hunt daily limit reached")

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
	"forest": {
		Key: "forest_zone", Emoji: "🌲", LevelMin: 1, LevelMax: 8, XPMult: 1.0,
		Enemies: []EnemyTemplate{
			{Name: "Slime Gluant", Emoji: "💧", HP: 25, Atk: 5, Def: 2, Spd: 5},
			{Name: "Sanglier Sauvage", Emoji: "🐗", HP: 35, Atk: 8, Def: 5, Spd: 10},
		},
		LootTable: []LootEntry{
			{Item: "pebble", Chance: 0.50, MaxQty: 3},
			{Item: "tomato", Chance: 0.30, MaxQty: 2},
			{Item: "coal", Chance: 0.15, MaxQty: 1},
			{Item: "forest_egg", Chance: 0.02, MaxQty: 1},
		},
	},
	"cave": {
		Key: "cave_zone", Emoji: "🦇", LevelMin: 6, LevelMax: 14, XPMult: 1.5,
		Enemies: []EnemyTemplate{
			{Name: "Gobelin Voleur", Emoji: "👺", HP: 40, Atk: 18, Def: 5, Spd: 25},
			{Name: "Araignée Géante", Emoji: "🕷️", HP: 50, Atk: 15, Def: 8, Spd: 30},
		},
		LootTable: []LootEntry{
			{Item: "coal", Chance: 0.60, MaxQty: 3},
			{Item: "iron_ore", Chance: 0.40, MaxQty: 2},
			{Item: "sardine", Chance: 0.20, MaxQty: 1},
			{Item: "cave_egg", Chance: 0.02, MaxQty: 1},
		},
	},
	"desert": {
		Key: "desert_zone", Emoji: "🏜️", LevelMin: 10, LevelMax: 20, XPMult: 2.5,
		Enemies: []EnemyTemplate{
			{Name: "Scarabée de Sable", Emoji: "🪲", HP: 60, Atk: 22, Def: 12, Spd: 15},
			{Name: "Coyote Affamé", Emoji: "🐺", HP: 70, Atk: 28, Def: 8, Spd: 30},
		},
		LootTable: []LootEntry{
			{Item: "copper_ore", Chance: 0.50, MaxQty: 3},
			{Item: "silver_ore", Chance: 0.30, MaxQty: 2},
			{Item: "gold_nugget", Chance: 0.15, MaxQty: 1},
			{Item: "desert_egg", Chance: 0.025, MaxQty: 1},
		},
	},
	"mountain": {
		Key: "mountain_zone", Emoji: "🏔️", LevelMin: 14, LevelMax: 25, XPMult: 4.0,
		Enemies: []EnemyTemplate{
			{Name: "Chèvre des Rochers", Emoji: "🐐", HP: 80, Atk: 25, Def: 15, Spd: 20},
			{Name: "Géant de Pierre", Emoji: "🗿", HP: 110, Atk: 30, Def: 25, Spd: 5},
		},
		LootTable: []LootEntry{
			{Item: "iron_ore", Chance: 0.50, MaxQty: 4},
			{Item: "emerald", Chance: 0.20, MaxQty: 1},
			{Item: "platinum", Chance: 0.15, MaxQty: 1},
			{Item: "mountain_egg", Chance: 0.025, MaxQty: 1},
		},
	},
	"ocean": {
		Key: "ocean_zone", Emoji: "🌊", LevelMin: 18, LevelMax: 30, XPMult: 6.0,
		Enemies: []EnemyTemplate{
			{Name: "Requin des Abysses", Emoji: "🦈", HP: 100, Atk: 35, Def: 12, Spd: 35},
			{Name: "Méduse Toxique", Emoji: "🪼", HP: 80, Atk: 28, Def: 8, Spd: 20},
		},
		LootTable: []LootEntry{
			{Item: "sardine", Chance: 0.50, MaxQty: 3},
			{Item: "swordfish", Chance: 0.25, MaxQty: 1},
			{Item: "old_boot", Chance: 0.15, MaxQty: 1},
			{Item: "ocean_egg", Chance: 0.025, MaxQty: 1},
		},
	},
	"tundra": {
		Key: "tundra_zone", Emoji: "❄️", LevelMin: 24, LevelMax: 35, XPMult: 8.0,
		Enemies: []EnemyTemplate{
			{Name: "Loup des Glaces", Emoji: "🐺", HP: 120, Atk: 38, Def: 15, Spd: 30},
			{Name: "Yeti Furieux", Emoji: "🦍", HP: 150, Atk: 35, Def: 30, Spd: 10},
		},
		LootTable: []LootEntry{
			{Item: "platinum", Chance: 0.40, MaxQty: 3},
			{Item: "emerald", Chance: 0.25, MaxQty: 2},
			{Item: "rough_diamond", Chance: 0.15, MaxQty: 1},
			{Item: "tundra_egg", Chance: 0.03, MaxQty: 1},
		},
	},
	"volcano": {
		Key: "volcano_zone", Emoji: "🌋", LevelMin: 30, LevelMax: 40, XPMult: 10.0,
		Enemies: []EnemyTemplate{
			{Name: "Golem de Magma", Emoji: "🗿", HP: 140, Atk: 40, Def: 30, Spd: 5},
			{Name: "Drake de Feu", Emoji: "🐉", HP: 120, Atk: 45, Def: 18, Spd: 30},
		},
		LootTable: []LootEntry{
			{Item: "gold_nugget", Chance: 0.50, MaxQty: 4},
			{Item: "rough_diamond", Chance: 0.30, MaxQty: 2},
			{Item: "magma_carp", Chance: 0.20, MaxQty: 2},
			{Item: "volcano_egg", Chance: 0.03, MaxQty: 1},
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
	zone, ok := Zones[zoneKey]
	if !ok || len(zone.Enemies) == 0 {
		return &Combatant{Name: "Shadow Beast", Emoji: "👾", HP: 50, MaxHP: 50, Atk: 10, Def: 5, Level: 1, IsAlive: true}
	}
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
	CharLeveledUp bool
	CharNewLevel  int
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
	ok, _, err := s.store.CheckGameLimit(userID, "hunt", 30)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrHuntLimit
	}

	ready, err := s.store.CheckCooldown(userID, "hunt", 30*time.Second)
	if err != nil {
		return nil, err
	}
	if !ready {
		return nil, ErrHuntLimit
	}

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

	strBonus := charsvc.GetSTRBonus(s.store, userID)
	vitBonus := charsvc.GetVITReduction(s.store, userID)
	petAtk = int(float64(petAtk) * strBonus)
	petHP = int(float64(petHP) * (1.0 + vitBonus))
	if petHP > petMaxHP {
		petHP = petMaxHP
	}

	if charsvc.HasBuff(s.store, userID, "pet_bond") {
		petAtk = int(float64(petAtk) * 1.25)
		petHP = int(float64(petHP) * 1.25)
		petDef = int(float64(petDef) * 1.25)
		if petHP > petMaxHP*125/100 {
			petHP = petMaxHP * 125 / 100
		}
		charsvc.ConsumeBuff(s.store, userID, "pet_bond")
	}

	if charsvc.HasBuff(s.store, userID, "bulwark") {
		petHP = petMaxHP
		// first round immunity: skip the enemy's first attack
		charsvc.ConsumeBuff(s.store, userID, "bulwark")
	}

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

	// Persist HP changes; XP and leveling are handled by the caller via petsvc.AddXP
	s.store.DB.Model(&pet).Update("hp", petHP)

	charLeveled, charLvl := charsvc.AddXP(s.store, userID, xp)

	if playerWon && charsvc.HasBuff(s.store, userID, "scavenger") {
		for _, loot := range zone.LootTable {
			if rand.Float64() < loot.Chance {
				qty := rand.Intn(loot.MaxQty) + 1
				for i := 0; i < qty; i++ {
					s.store.DB.Exec(
						`INSERT INTO inventory (user_id, item_id, quantity) VALUES (?, ?, 1)
						 ON CONFLICT(user_id, item_id) DO UPDATE SET quantity = quantity + 1`,
						userID, loot.Item,
					)
					lootItems = append(lootItems, loot.Item)
				}
			}
		}
		charsvc.ConsumeBuff(s.store, userID, "scavenger")
	}

	_ = s.store.IncrementGameLimit(userID, "hunt")
	_ = s.store.SetCooldown(userID, "hunt")
	_ = s.store.RecordActivity(userID, "items_hunted", 1)

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
		CharLeveledUp: charLeveled,
		CharNewLevel:  charLvl,
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
