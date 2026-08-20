package hunt

import (
	"errors"
	"math/rand"
	"time"

	"gorm.io/gorm"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
	jobssvc "guacagamblebot/internal/service/jobs"
	npcsvc "guacagamblebot/internal/service/npcs"
	petsvc "guacagamblebot/internal/service/pets"
	"guacagamblebot/internal/store"
)

var (
	ErrHuntLimit    = errors.New("hunt daily limit reached")
	ErrHuntCooldown = errors.New("hunt cooldown active")
	ErrZoneLocked   = errors.New("hunt zone locked")
)

// FirstZones lists the zones available from the start without unlocking.
var FirstZones = map[string]bool{
	"forest": true,
	"cave":   true,
	"desert": true,
}

// ZoneUnlockRequirements describes how each progressive zone is unlocked:
// the zone key maps to the previous zone and the number of wins required
// there before the zone becomes accessible.
var ZoneUnlockRequirements = map[string]struct {
	Previous     string
	RequiredWins int
}{
	"mountain": {Previous: "desert", RequiredWins: 3},
	"ocean":    {Previous: "mountain", RequiredWins: 3},
	"tundra":   {Previous: "ocean", RequiredWins: 3},
	"volcano":  {Previous: "tundra", RequiredWins: 3},
}

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
	Key       string
	Emoji     string
	LevelMin  int
	LevelMax  int
	XPMult    float64
	Enemies   []EnemyTemplate
	Boss      EnemyTemplate
	LootTable []LootEntry
}

// BossSpawnChance is the probability (per hunt) that the zone boss appears
// instead of a regular enemy.
const BossSpawnChance = 0.12

var Zones = map[string]Zone{
	"forest": {
		Key: "forest_zone", Emoji: "🌲", LevelMin: 1, LevelMax: 8, XPMult: 1.0,
		Enemies: []EnemyTemplate{
			{Name: "Slime Gluant", Emoji: "💧", HP: 25, Atk: 5, Def: 2, Spd: 5},
			{Name: "Sanglier Sauvage", Emoji: "🐗", HP: 35, Atk: 8, Def: 5, Spd: 10},
		},
		Boss: EnemyTemplate{Name: "Seigneur de la Forêt", Emoji: "🌳", HP: 55, Atk: 12, Def: 5, Spd: 8},
		LootTable: []LootEntry{
			{Item: "pebble", Chance: 0.50, MaxQty: 3},
			{Item: "tomato", Chance: 0.30, MaxQty: 2},
			{Item: "coal", Chance: 0.15, MaxQty: 1},
			{Item: "wheat_seed", Chance: 0.25, MaxQty: 2},
			{Item: "wheat", Chance: 0.25, MaxQty: 2},
			{Item: "tomato_seed", Chance: 0.10, MaxQty: 1},
			{Item: "worm", Chance: 0.15, MaxQty: 2},
			{Item: "forest_egg", Chance: 0.03, MaxQty: 1},
		},
	},
	"cave": {
		Key: "cave_zone", Emoji: "🦇", LevelMin: 6, LevelMax: 14, XPMult: 1.5,
		Enemies: []EnemyTemplate{
			{Name: "Gobelin Voleur", Emoji: "👺", HP: 40, Atk: 18, Def: 5, Spd: 25},
			{Name: "Araignée Géante", Emoji: "🕷️", HP: 50, Atk: 15, Def: 8, Spd: 30},
		},
		Boss: EnemyTemplate{Name: "Roi des Gobelins", Emoji: "👑", HP: 90, Atk: 28, Def: 10, Spd: 20},
		LootTable: []LootEntry{
			{Item: "coal", Chance: 0.60, MaxQty: 3},
			{Item: "iron_ore", Chance: 0.40, MaxQty: 2},
			{Item: "sardine", Chance: 0.20, MaxQty: 1},
			{Item: "potato_seed", Chance: 0.15, MaxQty: 2},
			{Item: "carrot_seed", Chance: 0.12, MaxQty: 1},
			{Item: "worm", Chance: 0.20, MaxQty: 2},
			{Item: "crayfish", Chance: 0.05, MaxQty: 1},
			{Item: "cave_egg", Chance: 0.025, MaxQty: 1},
		},
	},
	"desert": {
		Key: "desert_zone", Emoji: "🏜️", LevelMin: 10, LevelMax: 20, XPMult: 2.5,
		Enemies: []EnemyTemplate{
			{Name: "Scarabée de Sable", Emoji: "🪲", HP: 60, Atk: 22, Def: 12, Spd: 15},
			{Name: "Coyote Affamé", Emoji: "🐺", HP: 70, Atk: 28, Def: 8, Spd: 30},
		},
		Boss: EnemyTemplate{Name: "Roi Scorpion", Emoji: "🦂", HP: 130, Atk: 40, Def: 18, Spd: 25},
		LootTable: []LootEntry{
			{Item: "copper_ore", Chance: 0.50, MaxQty: 3},
			{Item: "silver_ore", Chance: 0.30, MaxQty: 2},
			{Item: "gold_nugget", Chance: 0.15, MaxQty: 1},
			{Item: "corn_seed", Chance: 0.15, MaxQty: 2},
			{Item: "corn", Chance: 0.15, MaxQty: 2},
			{Item: "pumpkin_seed", Chance: 0.10, MaxQty: 1},
			{Item: "crayfish", Chance: 0.05, MaxQty: 1},
			{Item: "desert_egg", Chance: 0.022, MaxQty: 1},
		},
	},
	"mountain": {
		Key: "mountain_zone", Emoji: "🏔️", LevelMin: 14, LevelMax: 25, XPMult: 4.0,
		Enemies: []EnemyTemplate{
			{Name: "Chèvre des Rochers", Emoji: "🐐", HP: 80, Atk: 25, Def: 15, Spd: 20},
			{Name: "Géant de Pierre", Emoji: "🗿", HP: 110, Atk: 30, Def: 25, Spd: 5},
		},
		Boss: EnemyTemplate{Name: "Titan de Pierre", Emoji: "🗿", HP: 200, Atk: 48, Def: 35, Spd: 10},
		LootTable: []LootEntry{
			{Item: "iron_ore", Chance: 0.50, MaxQty: 4},
			{Item: "emerald", Chance: 0.20, MaxQty: 1},
			{Item: "platinum", Chance: 0.15, MaxQty: 1},
			{Item: "oat_seed", Chance: 0.15, MaxQty: 2},
			{Item: "oat", Chance: 0.15, MaxQty: 2},
			{Item: "coffee_seed", Chance: 0.08, MaxQty: 1},
			{Item: "crayfish", Chance: 0.04, MaxQty: 1},
			{Item: "mountain_egg", Chance: 0.02, MaxQty: 1},
		},
	},
	"ocean": {
		Key: "ocean_zone", Emoji: "🌊", LevelMin: 18, LevelMax: 30, XPMult: 6.0,
		Enemies: []EnemyTemplate{
			{Name: "Requin des Abysses", Emoji: "🦈", HP: 100, Atk: 35, Def: 12, Spd: 35},
			{Name: "Méduse Toxique", Emoji: "🪼", HP: 80, Atk: 28, Def: 8, Spd: 20},
		},
		Boss: EnemyTemplate{Name: "Kraken des Abysses", Emoji: "🐙", HP: 260, Atk: 55, Def: 25, Spd: 40},
		LootTable: []LootEntry{
			{Item: "sardine", Chance: 0.50, MaxQty: 3},
			{Item: "swordfish", Chance: 0.25, MaxQty: 1},
			{Item: "old_boot", Chance: 0.15, MaxQty: 1},
			{Item: "carrot_seed", Chance: 0.10, MaxQty: 1},
			{Item: "cocoa_seed", Chance: 0.08, MaxQty: 1},
			{Item: "worm", Chance: 0.20, MaxQty: 2},
			{Item: "crayfish", Chance: 0.08, MaxQty: 1},
			{Item: "golden_lure", Chance: 0.02, MaxQty: 1},
			{Item: "ocean_egg", Chance: 0.018, MaxQty: 1},
		},
	},
	"tundra": {
		Key: "tundra_zone", Emoji: "❄️", LevelMin: 24, LevelMax: 35, XPMult: 8.0,
		Enemies: []EnemyTemplate{
			{Name: "Loup des Glaces", Emoji: "🐺", HP: 120, Atk: 38, Def: 15, Spd: 30},
			{Name: "Yeti Furieux", Emoji: "🦍", HP: 150, Atk: 35, Def: 30, Spd: 10},
		},
		Boss: EnemyTemplate{Name: "Grand Loup Blanc", Emoji: "🐺", HP: 320, Atk: 60, Def: 40, Spd: 35},
		LootTable: []LootEntry{
			{Item: "platinum", Chance: 0.40, MaxQty: 3},
			{Item: "emerald", Chance: 0.25, MaxQty: 2},
			{Item: "rough_diamond", Chance: 0.15, MaxQty: 1},
			{Item: "pumpkin_seed", Chance: 0.10, MaxQty: 1},
			{Item: "golden_apple_seed", Chance: 0.04, MaxQty: 1},
			{Item: "crayfish", Chance: 0.05, MaxQty: 1},
			{Item: "tundra_egg", Chance: 0.016, MaxQty: 1},
		},
	},
	"volcano": {
		Key: "volcano_zone", Emoji: "🌋", LevelMin: 30, LevelMax: 40, XPMult: 10.0,
		Enemies: []EnemyTemplate{
			{Name: "Golem de Magma", Emoji: "🗿", HP: 140, Atk: 40, Def: 30, Spd: 5},
			{Name: "Drake de Feu", Emoji: "🐉", HP: 120, Atk: 45, Def: 18, Spd: 30},
		},
		Boss: EnemyTemplate{Name: "Dragon Primordial", Emoji: "🐉", HP: 400, Atk: 75, Def: 45, Spd: 40},
		LootTable: []LootEntry{
			{Item: "gold_nugget", Chance: 0.50, MaxQty: 4},
			{Item: "rough_diamond", Chance: 0.30, MaxQty: 2},
			{Item: "magma_carp", Chance: 0.20, MaxQty: 2},
			{Item: "coffee_seed", Chance: 0.10, MaxQty: 1},
			{Item: "star_fruit_seed", Chance: 0.04, MaxQty: 1},
			{Item: "crayfish", Chance: 0.05, MaxQty: 1},
			{Item: "golden_lure", Chance: 0.02, MaxQty: 1},
			{Item: "volcano_egg", Chance: 0.015, MaxQty: 1},
		},
	},
}

// NewEnemy builds a regular zone enemy as a battle pet scaled to a random
// level within the zone range.
func NewEnemy(zoneKey string) *battle.BattlePet {
	zone, ok := Zones[zoneKey]
	if !ok || len(zone.Enemies) == 0 {
		return &battle.BattlePet{ID: -1, Nickname: "Shadow Beast", Emoji: "👾", PetType: "Shadow Beast", Level: 1, HP: 50, MaxHP: 50, Atk: 10, Defense: 5, Speed: 5, DGE: 5, ACC: 5, CritC: 5, CritD: 1.5}
	}
	t := zone.Enemies[rand.Intn(len(zone.Enemies))]
	lvl := zone.LevelMin + rand.Intn(zone.LevelMax-zone.LevelMin+1)
	return buildBattleEnemy(t, lvl)
}

// NewZoneEncounter rolls a hunt encounter: with BossSpawnChance probability the
// zone boss appears instead of a regular enemy. It returns the enemy and
// whether it is the zone boss.
func NewZoneEncounter(zoneKey string) (*battle.BattlePet, bool) {
	zone, ok := Zones[zoneKey]
	if !ok {
		return NewEnemy(zoneKey), false
	}
	if zone.Boss.Name != "" && rand.Float64() < BossSpawnChance {
		lvl := zone.LevelMin + rand.Intn(zone.LevelMax-zone.LevelMin+1)
		return buildBattleEnemy(zone.Boss, lvl), true
	}
	return NewEnemy(zoneKey), false
}

func buildBattleEnemy(t EnemyTemplate, lvl int) *battle.BattlePet {
	return &battle.BattlePet{
		ID:       -1,
		Nickname: t.Name,
		Emoji:    t.Emoji,
		PetType:  t.Name,
		Level:    lvl,
		HP:       t.HP + lvl*5,
		MaxHP:    t.HP + lvl*5,
		Atk:      t.Atk + lvl*2,
		Defense:  t.Def + lvl*1,
		Speed:    t.Spd,
		DGE:      5,
		ACC:      5,
		CritC:    5,
		CritD:    1.5,
	}
}

type BattleResult struct {
	PetStartHP    int
	PetHP         int
	PetMaxHP      int
	EnemyHP       int
	EnemyMaxHP    int
	EnemyName     string
	EnemyEmoji    string
	EnemyLevel    int
	IsBoss        bool
	PlayerWon     bool
	EnemyWon      bool
	Turns         []battle.BattleTurn
	XP            int
	Loot          []string
	LeveledUp     bool
	NewLevel      int
	CharLeveledUp bool
	CharNewLevel  int
}

var ErrNoPet = errors.New("no active pet")
var ErrPetKO = errors.New("pet is KO")

type Service struct {
	store  *store.Store
	cfg    *config.Config
	npcSvc *npcsvc.Service
}

func New(s *store.Store, cfg *config.Config, npcSvc *npcsvc.Service) *Service {
	return &Service{store: s, cfg: cfg, npcSvc: npcSvc}
}

// HasZoneAccess reports whether a user may hunt in the given zone. The first
// zones are always open; the later zones require a prior unlock.
func (s *Service) HasZoneAccess(userID int64, zoneKey string) (bool, error) {
	if FirstZones[zoneKey] {
		return true, nil
	}
	return s.store.HasUnlockedZone(userID, zoneKey)
}

// RecordHuntWin registers a victory in a zone. When the required number of
// wins in the previous zone is reached, the next zone is unlocked and its
// key is returned so callers can announce it.
func (s *Service) RecordHuntWin(userID int64, zoneKey string) (string, error) {
	wins, err := s.store.IncrementZoneWins(userID, zoneKey)
	if err != nil {
		return "", err
	}
	for next, req := range ZoneUnlockRequirements {
		if req.Previous != zoneKey || wins < req.RequiredWins {
			continue
		}
		already, err := s.store.HasUnlockedZone(userID, next)
		if err != nil {
			return "", err
		}
		if !already {
			if err := s.store.UnlockZone(userID, next); err != nil {
				return "", err
			}
			return next, nil
		}
	}
	return "", nil
}

// RecordZoneBossKill registers a zone boss kill and returns the new total
// boss kill count for that zone.
func (s *Service) RecordZoneBossKill(userID int64, zoneKey string) (int, error) {
	return s.store.IncrementZoneBossKills(userID, zoneKey)
}

func (s *Service) ExecuteHunt(userID int64, zoneKey string) (*BattleResult, error) {
	ok, _, err := s.store.CheckGameLimit(userID, "hunt", s.cfg.HuntMaxPerDay)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrHuntLimit
	}

	ready, err := s.store.CheckCooldown(userID, "hunt", time.Duration(s.cfg.HuntCooldownSeconds)*time.Second)
	if err != nil {
		return nil, err
	}
	if !ready {
		return nil, ErrHuntCooldown
	}

	free, err := s.store.FreeSlots(s.store.DB, userID)
	if err != nil {
		return nil, err
	}
	if free <= 0 {
		return nil, store.ErrInventoryFull
	}

	zone, ok := Zones[zoneKey]
	if !ok {
		return nil, errors.New("invalid zone")
	}

	access, err := s.HasZoneAccess(userID, zoneKey)
	if err != nil {
		return nil, err
	}
	if !access {
		return nil, ErrZoneLocked
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

	enemy, isBoss := NewZoneEncounter(zoneKey)

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

	petEmoji := "🐾"
	if pt := petsvc.PetTypes[pet.PetType]; pt != nil {
		petEmoji = pt.Emoji
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
		charsvc.ConsumeBuff(s.store, userID, "bulwark")
	}

	var skills []model.UserPetSkill
	s.store.DB.Where("pet_id = ?", pet.ID).Find(&skills)
	skillIDs := make([]string, 0, len(skills))
	for _, sk := range skills {
		skillIDs = append(skillIDs, sk.SkillID)
	}

	petBP := &battle.BattlePet{
		ID: pet.ID, Nickname: pet.Nickname, Emoji: petEmoji, PetType: pet.PetType,
		Level: pet.Level, HP: petHP, MaxHP: max(petMaxHP, petHP),
		Atk: petAtk, Defense: petDef, Speed: pet.Speed,
		DGE: pet.DGE, ACC: pet.ACC, CritC: pet.CritC, CritD: pet.CritD, SpcC: pet.SpcC,
		Skills: skillIDs,
	}

	petStartHP := petHP

	battleResult := battle.SimulatePreserveHP(petBP, enemy)

	playerWon := petBP.IsAlive() && !enemy.IsAlive()
	enemyWon := !petBP.IsAlive() && enemy.IsAlive()

	var xp int
	var lootItems []string
	leveledUp := false
	newLevel := pet.Level

	if playerWon {
		if isBoss {
			s.npcSvc.AddActivityReputation(userID, "hunting", 5)
		} else {
			s.npcSvc.AddActivityReputation(userID, "hunting", 1)
		}
		baseXP := enemy.Level * (15 + rand.Intn(11))
		xp = int(float64(baseXP) * zone.XPMult)

		if err := achievement.IncrementStat(s.store.DB, userID, "pve_wins", 1); err != nil {
			return nil, err
		}

		for _, loot := range zone.LootTable {
			if rand.Float64() < loot.Chance {
				qty := rand.Intn(loot.MaxQty) + 1
				for i := 0; i < qty; i++ {
					if err := s.store.AddItemRaw(s.store.DB, userID, loot.Item, 1); err != nil {
						return nil, err
					}
					lootItems = append(lootItems, loot.Item)
				}
			}
		}

		// Hunts can drop level-appropriate gear (better odds on boss hunts).
		dropChance := 0.08
		if isBoss {
			dropChance = 0.25
		}
		char, _ := s.store.EnsureCharacter(userID)
		charLvl := 1
		if char != nil {
			charLvl = char.Level
		}
		if gear, ok := s.rollHuntGear(userID, charLvl, dropChance); ok {
			if err := s.grantGearInstance(userID, gear); err != nil {
				return nil, err
			}
			lootItems = append(lootItems, gear.ID)
		}
	} else if enemyWon {
		baseXP := enemy.Level * (15 + rand.Intn(11)) / 10
		xp = baseXP
	} else {
		baseXP := enemy.Level * (15 + rand.Intn(11)) / 2
		xp = int(float64(baseXP) * zone.XPMult)
	}

	// Persist HP changes; XP and leveling are handled by the caller via petsvc.AddXP
	s.store.DB.Model(&pet).Update("hp", petBP.HP)

	charLeveled, charLvl := charsvc.AddXP(s.store, userID, xp)

	if err := s.grantHunterJobXP(userID, xp); err != nil {
		return nil, err
	}

	if playerWon && charsvc.HasBuff(s.store, userID, "scavenger") {
		for _, loot := range zone.LootTable {
			if rand.Float64() < loot.Chance {
				qty := rand.Intn(loot.MaxQty) + 1
				for i := 0; i < qty; i++ {
					if err := s.store.AddItemRaw(s.store.DB, userID, loot.Item, 1); err != nil {
						return nil, err
					}
					lootItems = append(lootItems, loot.Item)
				}
			}
		}
		charsvc.ConsumeBuff(s.store, userID, "scavenger")
	}

	_ = s.store.IncrementGameLimit(userID, "hunt")
	_ = s.store.SetCooldown(userID, "hunt")
	_ = s.store.RecordActivity(userID, "items_hunted", 1)
	// Zone-specific stat so daily/quest objectives can target one hunt zone.
	_ = s.store.RecordActivity(userID, "hunt_"+zoneKey, 1)

	unlocks, err := achievement.CheckAndUnlock(s.store.DB, userID)
	if err != nil {
		return nil, err
	}
	_ = unlocks

	return &BattleResult{
		PetStartHP:    petStartHP,
		PetHP:         petBP.HP,
		PetMaxHP:      petMaxHP,
		EnemyHP:       enemy.HP,
		EnemyMaxHP:    enemy.MaxHP,
		EnemyName:     enemy.Nickname,
		EnemyEmoji:    enemy.Emoji,
		EnemyLevel:    enemy.Level,
		IsBoss:        isBoss,
		PlayerWon:     playerWon,
		EnemyWon:      enemyWon,
		Turns:         battleResult.Turns,
		XP:            xp,
		Loot:          lootItems,
		LeveledUp:     leveledUp,
		NewLevel:      newLevel,
		CharLeveledUp: charLeveled,
		CharNewLevel:  charLvl,
	}, nil
}

// grantHunterJobXP awards hunter job XP for a hunt, creating the job row on
// first use and leveling it up like the other activity jobs.
func (s *Service) grantHunterJobXP(userID int64, xp int) error {
	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "hunter").First(&job).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		job = model.Job{UserID: userID, JobName: "hunter", Level: 1, XP: xp}
		levelUpJob(&job)
		return s.store.DB.Create(&job).Error
	}
	job.XP += xp
	levelUpJob(&job)
	return s.store.DB.Model(&model.Job{}).Where("user_id = ? AND job_name = ?", userID, "hunter").
		Updates(map[string]any{"xp": job.XP, "level": job.Level}).Error
}

// levelUpJob applies as many level-ups as the job's XP warrants.
func levelUpJob(job *model.Job) {
	next := jobssvc.XPForLevel(job.Level)
	for job.XP >= next {
		job.XP -= next
		job.Level++
		next = jobssvc.XPForLevel(job.Level)
	}
}

// rollHuntGear decides whether a hunt drops gear and picks a level-appropriate
// piece (within 10 levels below the player, never above).
func (s *Service) rollHuntGear(userID int64, charLevel int, chance float64) (items.Item, bool) {
	if rand.Float64() >= chance {
		return items.Item{}, false
	}
	var pool []items.Item
	for _, it := range items.AllItems() {
		if it.EquipSlot == "" || it.MinLevel <= 0 || it.MinLevel > charLevel {
			continue
		}
		if it.MinLevel < charLevel-9 {
			continue
		}
		pool = append(pool, it)
	}
	if len(pool) == 0 {
		return items.Item{}, false
	}
	return pool[rand.Intn(len(pool))], true
}

// grantGearInstance turns a catalog gear item into an equipment instance with
// rolled affixes for the player.
func (s *Service) grantGearInstance(userID int64, it items.Item) error {
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
	_, err := s.store.CreateEquipmentFromAffixes(userID, it.ID, it.Name, it.Emoji,
		string(rar), it.EquipSlot, it.MinLevel,
		it.StatSTR, it.StatDEX, it.StatINT, it.StatVIT, it.StatLUK,
		applied, it.SetID)
	return err
}
