package sanctuary

import (
	"errors"
	"math/rand"
	"time"

	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	ps "guacagamblebot/internal/service/pets"
	"guacagamblebot/internal/store"
)

var ErrNoSanctuary = errors.New("no sanctuary built")
var ErrSanctuaryFull = errors.New("sanctuary is full")
var ErrPetAlreadyInSanctuary = errors.New("pet is already in sanctuary")
var ErrPetNotInSanctuary = errors.New("pet is not in sanctuary")
var ErrShowcaseNoSlots = errors.New("showcase is full")
var ErrHouseSanctuaryCap = errors.New("sanctuary max tier for this house reached")

type SanctuaryTier struct {
	Tier       int
	Name       string
	Slots      int
	Price      int
	Materials  map[string]int
	BuildHours int
}

var SanctuaryTiers = map[int]SanctuaryTier{
	1: {Tier: 1, Name: "Small Paddock", Slots: 5, Price: 5000, BuildHours: 2,
		Materials: map[string]int{"wheat": 25, "coal": 10}},
	2: {Tier: 2, Name: "Animal Ranch", Slots: 15, Price: 25000, BuildHours: 8,
		Materials: map[string]int{"iron_ore": 10, "copper_ore": 5, "silver_ore": 3, "sardine": 10}},
	3: {Tier: 3, Name: "Wildlife Sanctuary", Slots: 30, Price: 75000, BuildHours: 24,
		Materials: map[string]int{"gold_nugget": 5, "emerald": 3, "platinum": 2, "rough_diamond": 1}},
	4: {Tier: 4, Name: "Legendary Menagerie", Slots: 50, Price: 250000, BuildHours: 72,
		Materials: map[string]int{"rough_diamond": 2, "gold_nugget": 5, "emerald": 3, "pure_dna": 1}},
}

type lootEntry struct {
	Item   string
	Weight float64
}

type biomeLootTable struct {
	Basic []lootEntry
	Mid   []lootEntry
	Rare  []lootEntry
}

var biomeLootTables = map[string]biomeLootTable{
	"forest": {
		Basic: []lootEntry{{"tomato", 0.30}, {"wheat", 0.25}, {"pebble", 0.20}, {"wheat_seed", 0.15}, {"worm", 0.10}},
		Mid:   []lootEntry{{"coal", 0.10}, {"iron_ore", 0.10}},
		Rare:  []lootEntry{{"golden_apple", 0.03}, {"emerald", 0.02}},
	},
	"cave": {
		Basic: []lootEntry{{"coal", 0.30}, {"pebble", 0.25}, {"iron_ore", 0.20}, {"potato_seed", 0.15}, {"worm", 0.10}},
		Mid:   []lootEntry{{"copper_ore", 0.10}, {"silver_ore", 0.10}},
		Rare:  []lootEntry{{"rough_diamond", 0.03}, {"emerald", 0.02}},
	},
	"desert": {
		Basic: []lootEntry{{"potato", 0.30}, {"pebble", 0.25}, {"coal", 0.20}, {"corn_seed", 0.15}},
		Mid:   []lootEntry{{"copper_ore", 0.10}, {"silver_ore", 0.10}},
		Rare:  []lootEntry{{"gold_nugget", 0.03}, {"emerald", 0.02}},
	},
	"mountain": {
		Basic: []lootEntry{{"iron_ore", 0.30}, {"coal", 0.25}, {"pebble", 0.20}, {"oat_seed", 0.15}},
		Mid:   []lootEntry{{"silver_ore", 0.10}, {"platinum", 0.10}},
		Rare:  []lootEntry{{"emerald", 0.03}, {"golden_apple", 0.01}, {"rough_diamond", 0.01}},
	},
	"ocean": {
		Basic: []lootEntry{{"sardine", 0.30}, {"old_boot", 0.25}, {"trout", 0.20}, {"carrot_seed", 0.10}, {"worm", 0.10}},
		Mid:   []lootEntry{{"salmon", 0.10}, {"swordfish", 0.10}, {"crayfish", 0.05}},
		Rare:  []lootEntry{{"shark", 0.02}, {"whale", 0.01}, {"kraken_tentacle", 0.02}, {"golden_lure", 0.01}},
	},
	"tundra": {
		Basic: []lootEntry{{"coal", 0.30}, {"iron_ore", 0.25}, {"pebble", 0.20}, {"pumpkin_seed", 0.10}},
		Mid:   []lootEntry{{"silver_ore", 0.10}, {"platinum", 0.10}, {"crayfish", 0.05}},
		Rare:  []lootEntry{{"emerald", 0.02}, {"rough_diamond", 0.02}, {"star_fruit", 0.01}},
	},
	"volcano": {
		Basic: []lootEntry{{"coal", 0.30}, {"gold_nugget", 0.25}, {"copper_ore", 0.20}, {"tomato_seed", 0.10}},
		Mid:   []lootEntry{{"platinum", 0.10}, {"rough_diamond", 0.05}, {"magma_carp", 0.05}, {"crayfish", 0.05}},
		Rare:  []lootEntry{{"emerald", 0.02}, {"lava_serpent", 0.02}, {"star_fruit", 0.01}},
	},
}

type Service struct {
	store  *store.Store
	cfg    *config.Config
	petSvc *ps.Service
}

func New(s *store.Store, cfg *config.Config, petSvc ...*ps.Service) *Service {
	svc := &Service{store: s, cfg: cfg}
	if len(petSvc) > 0 {
		svc.petSvc = petSvc[0]
	}
	return svc
}

func (s *Service) GetSanctuary(userID int64) (*model.UserSanctuary, error) {
	var san model.UserSanctuary
	err := s.store.DB.Where("user_id = ?", userID).First(&san).Error
	if err != nil {
		return nil, err
	}
	return &san, nil
}

func (s *Service) GetOrCreateSanctuary(userID int64) (*model.UserSanctuary, error) {
	var san model.UserSanctuary
	err := s.store.DB.Where("user_id = ?", userID).First(&san).Error
	if err == gorm.ErrRecordNotFound {
		san = model.UserSanctuary{UserID: userID, Tier: 0, LastCollect: nil}
		s.store.DB.Create(&san)
		return &san, nil
	}
	if err != nil {
		return nil, err
	}
	return &san, nil
}

func (s *Service) GetMaxSlots(userID int64) int {
	san, err := s.GetSanctuary(userID)
	if err != nil {
		return 0
	}
	if t, ok := SanctuaryTiers[san.Tier]; ok {
		return t.Slots
	}
	return 0
}

func (s *Service) GetHouseMaxTier(userID int64) int {
	var h model.UserHousing
	if err := s.store.DB.Where("user_id = ? AND is_active = ?", userID, true).First(&h).Error; err != nil {
		return 0
	}
	ht := getHouseDef(h.HouseType)
	if ht == nil {
		return 0
	}
	return ht.MaxSanctuaryTier
}

func (s *Service) GetHouseMaxSlots(userID int64) int {
	maxTier := s.GetHouseMaxTier(userID)
	if t, ok := SanctuaryTiers[maxTier]; ok {
		return t.Slots
	}
	return 0
}

// HasSanctuarySpace reports whether the sanctuary has a free slot.
func (s *Service) HasSanctuarySpace(userID int64) bool {
	return s.GetUsedSlots(userID) < s.GetMaxSlots(userID)
}

// getHouseDef avoids importing housing service (circular). Duplicate minimal house meta.
func getHouseDef(houseType string) *houseMeta {
	if m, ok := houseMaxTier[houseType]; ok {
		return m
	}
	return nil
}

type houseMeta struct {
	MaxSanctuaryTier int
}

var houseMaxTier = map[string]*houseMeta{
	"cardboard_box": {MaxSanctuaryTier: 1},
	"wooden_shack":  {MaxSanctuaryTier: 2},
	"brick_house":   {MaxSanctuaryTier: 3},
	"mansion":       {MaxSanctuaryTier: 3},
	"gilded_palace": {MaxSanctuaryTier: 4},
}

func (s *Service) GetUsedSlots(userID int64) int {
	var count int64
	s.store.DB.Model(&model.UserPet{}).Where("user_id = ? AND in_sanctuary = ?", userID, true).Count(&count)
	return int(count)
}

func (s *Service) CanStartConstruction(userID int64, tier int) bool {
	_, ok := SanctuaryTiers[tier]
	if !ok {
		return false
	}
	if tier > s.GetHouseMaxTier(userID) {
		return false
	}
	san, err := s.GetSanctuary(userID)
	if err != nil || san == nil {
		return tier == 1
	}
	if san.UnderConstruction != nil {
		return false
	}
	if tier != san.Tier+1 {
		return false
	}
	return true
}

func (s *Service) StartConstruction(userID int64, tier int) error {
	if tier > s.GetHouseMaxTier(userID) {
		return ErrHouseSanctuaryCap
	}
	if !s.CanStartConstruction(userID, tier) {
		return errors.New("cannot start construction")
	}
	t := SanctuaryTiers[tier]
	var user model.User
	if err := s.store.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		return err
	}
	if user.Balance < t.Price {
		return errors.New("not enough money")
	}
	for item, qty := range t.Materials {
		var inv model.Inventory
		if err := s.store.DB.Where("user_id = ? AND item_id = ? AND quantity >= ?", userID, item, qty).First(&inv).Error; err != nil {
			return errors.New("missing materials: " + item)
		}
	}
	user.Balance -= t.Price
	s.store.DB.Save(&user)
	for item, qty := range t.Materials {
		s.store.DB.Exec(
			`UPDATE inventory SET quantity = quantity - ? WHERE user_id = ? AND item_id = ? AND quantity > 0`,
			qty, userID, item,
		)
	}
	now := time.Now()
	finish := now.Add(time.Duration(t.BuildHours) * time.Hour)
	san, err := s.GetOrCreateSanctuary(userID)
	if err != nil {
		return err
	}
	tierStr := itoa(tier)
	san.UnderConstruction = &tierStr
	san.FinishTime = &finish
	return s.store.DB.Save(san).Error
}

func (s *Service) CompleteConstruction(userID int64) error {
	san, err := s.GetSanctuary(userID)
	if err != nil {
		return err
	}
	if san.UnderConstruction == nil || san.FinishTime == nil {
		return errors.New("nothing to complete")
	}
	if time.Now().Before(*san.FinishTime) {
		return errors.New("construction not yet finished")
	}
	tier, _ := atoi(*san.UnderConstruction)
	san.Tier = tier
	san.UnderConstruction = nil
	san.FinishTime = nil
	return s.store.DB.Save(san).Error
}

func (s *Service) RetirePet(userID int64, petID int64) error {
	used := s.GetUsedSlots(userID)
	max := s.GetMaxSlots(userID)
	if used >= max {
		return ErrSanctuaryFull
	}
	var pet model.UserPet
	if err := s.store.DB.Where("id = ? AND user_id = ?", petID, userID).First(&pet).Error; err != nil {
		return err
	}
	if pet.InSanctuary {
		return ErrPetAlreadyInSanctuary
	}
	pet.InSanctuary = true
	pet.IsActive = false
	return s.store.DB.Save(&pet).Error
}

func (s *Service) RecallPet(userID int64, petID int64) error {
	var pet model.UserPet
	if err := s.store.DB.Where("id = ? AND user_id = ?", petID, userID).First(&pet).Error; err != nil {
		return err
	}
	if !pet.InSanctuary {
		return ErrPetNotInSanctuary
	}
	var activeCount int64
	s.store.DB.Model(&model.UserPet{}).Where("user_id = ? AND in_sanctuary = ?", userID, false).Count(&activeCount)
	var user model.User
	s.store.DB.Where("user_id = ?", userID).First(&user)
	maxSlots := 3 + user.ExtraPetSlots
	if int(activeCount) >= maxSlots {
		return errors.New("active roster is full")
	}
	var u model.User
	if err := s.store.DB.Where("user_id = ?", userID).First(&u).Error; err != nil {
		return err
	}
	if u.Balance < 100 {
		return errors.New("not enough money ($100 fee to recall pet)")
	}
	u.Balance -= 100
	s.store.DB.Save(&u)
	pet.InSanctuary = false
	return s.store.DB.Save(&pet).Error
}

func (s *Service) SetShowcase(userID int64, petID int64, slot int) error {
	var pet model.UserPet
	if err := s.store.DB.Where("id = ? AND user_id = ?", petID, userID).First(&pet).Error; err != nil {
		return err
	}
	if !pet.InSanctuary {
		return errors.New("pet must be in sanctuary to showcase")
	}
	if slot > 5 || slot < 0 {
		return errors.New("invalid showcase slot")
	}
	if slot == 0 {
		pet.ShowcaseSlot = 0
		return s.store.DB.Save(&pet).Error
	}
	var count int64
	s.store.DB.Model(&model.UserPet{}).
		Where("user_id = ? AND in_sanctuary = ? AND showcase_slot > ?", userID, true, 0).
		Count(&count)
	if count >= 5 && pet.ShowcaseSlot == 0 {
		return ErrShowcaseNoSlots
	}
	s.store.DB.Model(&model.UserPet{}).
		Where("user_id = ? AND showcase_slot = ?", userID, slot).
		Update("showcase_slot", 0)
	pet.ShowcaseSlot = slot
	return s.store.DB.Save(&pet).Error
}

func (s *Service) CollectResources(userID int64) ([]string, int, error) {
	san, err := s.GetSanctuary(userID)
	if err != nil {
		return nil, 0, ErrNoSanctuary
	}
	if san.Tier == 0 {
		return nil, 0, ErrNoSanctuary
	}
	now := time.Now()
	if san.LastCollect != nil && now.Sub(*san.LastCollect) < 24*time.Hour {
		return nil, 0, errors.New("can collect once per 24 hours")
	}
	var pets []model.UserPet
	s.store.DB.Where("user_id = ? AND in_sanctuary = ?", userID, true).Find(&pets)
	if len(pets) == 0 {
		return nil, 0, errors.New("no pets in sanctuary")
	}
	var items []string
	for _, p := range pets {
		pt := ps.PetTypes[p.PetType]
		if pt == nil {
			continue
		}
		chance := 0.10
		switch pt.Rarity {
		case ps.RarityRare:
			chance = 0.20
		case ps.RarityEpic:
			chance = 0.35
		case ps.RarityLegendary:
			chance = 0.50
		}
		if rand.Float64() >= chance {
			continue
		}
		item := randomTieredBiomeResource(pt.Biome)
		if item != "" {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		now2 := time.Now()
		san.LastCollect = &now2
		s.store.DB.Save(san)
		return nil, 0, nil
	}
	for _, item := range items {
		s.store.AddItemRaw(s.store.DB, userID, item, 1)
	}
	now2 := time.Now()
	san.LastCollect = &now2
	s.store.DB.Save(san)
	return items, len(items), nil
}

func (s *Service) GetSanctuaryInfo(userID int64) (int, int, int, error) {
	san, err := s.GetOrCreateSanctuary(userID)
	if err != nil {
		return 0, 0, 0, err
	}
	tier := san.Tier
	used := s.GetUsedSlots(userID)
	max := 0
	if t, ok := SanctuaryTiers[tier]; ok {
		max = t.Slots
	}
	return tier, used, max, nil
}

// ── Side upgrades: Fusion (Trade-Up) & Ascendancy (Transcend) ───────────────

var (
	ErrFusionNoResearch   = errors.New("fusion research not completed")
	ErrFusionWrongCount   = errors.New("wrong number of pets for fusion")
	ErrFusionMixedRarity  = errors.New("all pets must be same rarity")
	ErrFusionLegendaryMax = errors.New("legendary pets cannot be fused")
	ErrFusionSameID       = errors.New("duplicate pet ids")
	ErrFusionNotOwned     = errors.New("pet not owned")
	ErrFusionActivePet    = errors.New("active pet cannot be fused")
	ErrAscendNotReady     = errors.New("pet not ready to transcend")
	ErrAscendLocked       = errors.New("pet is locked after transcend")
)

// TradeUpCosts adds money+items on top of pet consumption. Keeps python exact counts
// plus a resource sink. Random target species per tier.
var TradeUpCosts = map[string]struct {
	Money int
	Items map[string]int
}{
	ps.RarityCommon: {Money: 5000, Items: map[string]int{"bone_dust": 20, "coal": 10}},
	ps.RarityRare:   {Money: 20000, Items: map[string]int{"iron_ore": 10, "silver_ore": 5, "bone_dust": 20}},
	ps.RarityEpic:   {Money: 75000, Items: map[string]int{"gold_nugget": 5, "emerald": 2, "platinum": 1}},
}

// TradeUpResearch reuses forge fusion researches per spec.
var TradeUpResearch = map[string]string{
	ps.RarityCommon: "fusion_common",
	ps.RarityRare:   "fusion_rare",
	ps.RarityEpic:   "fusion_epic",
}

// TradeUp fuses pets of same rarity into one random pet of next rarity (instant).
func (s *Service) TradeUp(userID int64, petIDs []int64) (*model.UserPet, error) {
	if len(petIDs) == 0 {
		return nil, ErrFusionWrongCount
	}
	seen := map[int64]bool{}
	for _, id := range petIDs {
		if seen[id] {
			return nil, ErrFusionSameID
		}
		seen[id] = true
	}
	var pets []model.UserPet
	if err := s.store.DB.Where("id IN ?", petIDs).Find(&pets).Error; err != nil {
		return nil, err
	}
	if len(pets) != len(petIDs) {
		return nil, ErrFusionNotOwned
	}
	rarity := ""
	for i, p := range pets {
		if p.UserID != userID {
			return nil, ErrFusionNotOwned
		}
		if p.IsActive {
			return nil, ErrFusionActivePet
		}
		if p.OnExpedition {
			return nil, errors.New("pet on expedition cannot be fused")
		}
		pt := ps.PetTypes[p.PetType]
		if pt == nil {
			return nil, errors.New("unknown pet type")
		}
		if i == 0 {
			rarity = pt.Rarity
		} else if pt.Rarity != rarity {
			return nil, ErrFusionMixedRarity
		}
	}
	if rarity == ps.RarityLegendary {
		return nil, ErrFusionLegendaryMax
	}
	reqCount, targetRarity := ps.TradeUpRarity(rarity)
	if reqCount == 0 || targetRarity == "" {
		return nil, ErrFusionWrongCount
	}
	if len(petIDs) != reqCount {
		return nil, ErrFusionWrongCount
	}
	// Research gate (reuse forge researches if present)
	if rid, ok := TradeUpResearch[rarity]; ok && rid != "" {
		var r model.UserResearch
		if err := s.store.DB.Where("user_id = ? AND research_id = ? AND completed = ?", userID, rid, true).First(&r).Error; err != nil {
			return nil, ErrFusionNoResearch
		}
	}
	// Resource gate
	cost := TradeUpCosts[rarity]
	if cost.Money > 0 {
		bal, err := s.store.GetBalance(userID)
		if err != nil {
			return nil, err
		}
		if bal < cost.Money {
			return nil, errors.New("not enough money")
		}
	}
	for itemID, qty := range cost.Items {
		var inv model.Inventory
		if err := s.store.DB.Where("user_id = ? AND item_id = ? AND quantity >= ?", userID, itemID, qty).First(&inv).Error; err != nil {
			return nil, errors.New("missing materials: " + itemID)
		}
	}
	// Pick random target species
	newPetName := ps.RollGacha(targetRarity, ps.Biomes[rand.Intn(len(ps.Biomes))])
	if newPetName == "" {
		return nil, errors.New("failed to roll new pet")
	}
	var newPet *model.UserPet
	err := s.store.DB.Transaction(func(tx *gorm.DB) error {
		if cost.Money > 0 {
			if err := tx.Model(&model.User{}).Where("user_id = ?", userID).UpdateColumn("balance", gorm.Expr("balance - ?", cost.Money)).Error; err != nil {
				return err
			}
		}
		for itemID, qty := range cost.Items {
			if err := tx.Model(&model.Inventory{}).Where("user_id = ? AND item_id = ?", userID, itemID).UpdateColumn("quantity", gorm.Expr("quantity - ?", qty)).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("id IN ?", petIDs).Delete(&model.UserPet{}).Error; err != nil {
			return err
		}
		for _, id := range petIDs {
			_ = tx.Where("pet_id = ?", id).Delete(&model.UserPetSkill{}).Error
		}
		pt := ps.PetTypes[newPetName]
		if pt == nil {
			return errors.New("unknown target pet")
		}
		p := &model.UserPet{
			UserID:      userID,
			PetType:     newPetName,
			Nickname:    newPetName,
			Level:       1,
			MaxHP:       pt.MaxHP,
			HP:          pt.MaxHP,
			Atk:         pt.Atk,
			Defense:     pt.Defense,
			Speed:       pt.Speed,
			DGE:         pt.DGE,
			ACC:         pt.ACC,
			CritC:       pt.CritC,
			CritD:       pt.CritD,
			Bonus:       pt.Bonus,
			Elo:         1000,
			Personality: ps.RandomPersonality(),
			History:     `[]`,
		}
		if err := tx.Create(p).Error; err != nil {
			return err
		}
		newPet = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return newPet, nil
}

// Transcend increments TrsLvl of active pet by consuming same species sacrifice. Instant, then locks 24h.
func (s *Service) Transcend(userID int64, sacrificeID int64) (*model.UserPet, error) {
	active, err := func() (*model.UserPet, error) {
		var p model.UserPet
		if err := s.store.DB.Where("user_id = ? AND is_active = ?", userID, true).First(&p).Error; err != nil {
			return nil, err
		}
		return &p, nil
	}()
	if err != nil {
		return nil, errors.New("no active pet")
	}
	if active.Level < 20 {
		return nil, errors.New("active pet must be level 20")
	}
	if active.TranscendLockedUntil != nil && time.Now().Before(*active.TranscendLockedUntil) {
		return nil, ErrAscendLocked
	}
	var sac model.UserPet
	if err := s.store.DB.Where("id = ? AND user_id = ?", sacrificeID, userID).First(&sac).Error; err != nil {
		return nil, errors.New("sacrifice not found")
	}
	if sac.ID == active.ID {
		return nil, errors.New("cannot sacrifice active pet")
	}
	if sac.PetType != active.PetType {
		return nil, errors.New("sacrifice must be same species")
	}
	if sac.OnExpedition {
		return nil, errors.New("pet on expedition cannot be sacrificed")
	}
	err = s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.UserPet{}, sac.ID).Error; err != nil {
			return err
		}
		_ = tx.Where("pet_id = ?", sac.ID).Delete(&model.UserPetSkill{}).Error
		lock := time.Now().Add(24 * time.Hour)
		active.TrsLvl++
		active.TranscendLockedUntil = &lock
		return tx.Save(active).Error
	})
	if err != nil {
		return nil, err
	}
	return active, nil
}

func randomRoll(pool []lootEntry) string {
	r := rand.Float64()
	cum := 0.0
	for _, entry := range pool {
		cum += entry.Weight
		if r < cum {
			return entry.Item
		}
	}
	if len(pool) > 0 {
		return pool[len(pool)-1].Item
	}
	return ""
}

func randomTieredBiomeResource(biome string) string {
	table, ok := biomeLootTables[biome]
	if !ok {
		return ""
	}
	r := rand.Float64()
	switch {
	case r < 0.75:
		return randomRoll(table.Basic)
	case r < 0.95:
		return randomRoll(table.Mid)
	default:
		return randomRoll(table.Rare)
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

func atoi(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("not a number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
