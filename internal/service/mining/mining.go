package mining

import (
	"errors"
	"math/rand"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
	"guacagamblebot/internal/store"
)

var ErrMineLimit = errors.New("mining daily limit reached")

type MineItem struct {
	Name  string
	Value int
}

func lootAtDepth(depth int) []MineItem {
	switch {
	case depth <= 1:
		return []MineItem{{"pebble", 1}, {"coal", 5}}
	case depth <= 3:
		return []MineItem{{"coal", 5}, {"iron_ore", 10}, {"copper_ore", 15}}
	case depth == 4:
		return []MineItem{{"copper_ore", 15}, {"silver_ore", 25}, {"gold_nugget", 50}}
	case depth == 5:
		return []MineItem{{"silver_ore", 25}, {"gold_nugget", 50}, {"emerald", 100}}
	case depth <= 7:
		return []MineItem{{"gold_nugget", 50}, {"platinum", 75}, {"emerald", 100}}
	case depth <= 9:
		return []MineItem{{"platinum", 75}, {"emerald", 100}, {"rough_diamond", 300}}
	case depth <= 14:
		return []MineItem{{"emerald", 100}, {"rough_diamond", 300}, {"ancient_alloy", 500}}
	case depth <= 19:
		return []MineItem{{"rough_diamond", 300}, {"ancient_alloy", 500}, {"kethari_crystal", 1000}}
	case depth <= 24:
		return []MineItem{{"ancient_alloy", 500}, {"kethari_crystal", 1000}, {"primordial_geode", 2000}}
	case depth <= 29:
		return []MineItem{{"kethari_crystal", 1000}, {"primordial_geode", 2000}, {"resonance_core", 5000}}
	default:
		return []MineItem{{"primordial_geode", 2000}, {"resonance_core", 5000}}
	}
}

const (
	dailyDescendLimit = 50

	steelPickaxeItem     = "steel_pickaxe"
	diamondDrillItem     = "diamond_drill"
	ghostVeilBuff        = "ghostly_veil"
	ancientCoreShardItem = "ancient_core_shard"
	ancientAlloyItem     = "ancient_alloy"

	loreKethari  = "mine_lore_kethari"
	loreEngine   = "mine_lore_engine"
	loreFracture = "mine_lore_fracture"
	loreKing     = "mine_lore_king"
)

type BranchChoice string

const (
	BranchCareful     BranchChoice = "careful"
	BranchAggressive  BranchChoice = "aggressive"
	BranchSearchVeins BranchChoice = "search_veins"
	BranchRest        BranchChoice = "rest"
)

func (c BranchChoice) RiskMod() int {
	switch c {
	case BranchCareful:
		return -10
	case BranchAggressive:
		return 15
	case BranchSearchVeins:
		return 0
	case BranchRest:
		return -20
	}
	return 0
}

func (c BranchChoice) LootTierOffset() int {
	switch c {
	case BranchAggressive:
		return 1
	case BranchSearchVeins:
		return 0
	default:
		return 0
	}
}

func (c BranchChoice) AllowLoot() bool {
	return c != BranchRest
}

type ToolInfo struct {
	ItemID        string
	MinLevel      int
	LootTierBonus int
	RiskReduction int
}

var miningTools = []ToolInfo{
	{ItemID: "", MinLevel: 1, LootTierBonus: 0, RiskReduction: 0},
	{ItemID: steelPickaxeItem, MinLevel: 5, LootTierBonus: 1, RiskReduction: 5},
	{ItemID: diamondDrillItem, MinLevel: 10, LootTierBonus: 2, RiskReduction: 10},
}

func GetToolInfo(itemID string) ToolInfo {
	for _, t := range miningTools {
		if t.ItemID == itemID {
			return t
		}
	}
	return miningTools[0]
}

func (s *Service) OwnedTools(userID int64, level int) []ToolInfo {
	all := AvailableTools(level)
	var owned []ToolInfo
	for _, t := range all {
		if t.ItemID == "" {
			owned = append(owned, t)
			continue
		}
		has, _ := s.HasItem(userID, t.ItemID)
		if has {
			owned = append(owned, t)
		}
	}
	return owned
}

func LockedTools(level int) []ToolInfo {
	var out []ToolInfo
	for _, t := range miningTools {
		if t.MinLevel > level && t.ItemID != "" {
			out = append(out, t)
		}
	}
	return out
}

func AvailableTools(level int) []ToolInfo {
	var out []ToolInfo
	for _, t := range miningTools {
		if t.MinLevel <= level {
			out = append(out, t)
		}
	}
	return out
}

type LoreFragment struct {
	ID    string
	Title string
	Depth int
}

func (t ToolInfo) LocaleNameKey() string {
	switch t.ItemID {
	case steelPickaxeItem:
		return "mining.tool_name_steel"
	case diamondDrillItem:
		return "mining.tool_name_diamond"
	default:
		return "mining.tool_name_iron"
	}
}

func (t ToolInfo) LocaleDescKey() string {
	switch t.ItemID {
	case steelPickaxeItem:
		return "mining.tool_desc_steel"
	case diamondDrillItem:
		return "mining.tool_desc_diamond"
	default:
		return "mining.tool_desc_iron"
	}
}

func (t ToolInfo) Emoji() string {
	switch t.ItemID {
	case steelPickaxeItem:
		return "⛏️"
	case diamondDrillItem:
		return "🔧"
	default:
		return "🪨"
	}
}

var MiningLore = []LoreFragment{
	{ID: loreKethari, Title: "The Kethari — Children of Stone", Depth: 13},
	{ID: loreEngine, Title: "The Resonance Engine", Depth: 21},
	{ID: loreFracture, Title: "The Great Fracture", Depth: 33},
	{ID: loreKing, Title: "The Sleeping King", Depth: 42},
}

func DepthFlavorKey(depth int) string {
	switch {
	case depth <= 2:
		return "mining.flavor_tier1"
	case depth <= 4:
		return "mining.flavor_tier2"
	case depth <= 6:
		return "mining.flavor_tier3"
	case depth <= 9:
		return "mining.flavor_tier4"
	case depth <= 12:
		return "mining.flavor_tier5"
	case depth <= 15:
		return "mining.flavor_tier6"
	default:
		return "mining.flavor_tier7"
	}
}

func DepthColor(depth int) int {
	switch {
	case depth <= 3:
		return 0x4A90D9
	case depth <= 6:
		return 0x7B5EA7
	case depth <= 9:
		return 0x8B4513
	case depth <= 12:
		return 0x4A0E4E
	case depth <= 15:
		return 0x2C1810
	case depth <= 20:
		return 0x1A0A2E
	case depth <= 30:
		return 0x0A0000
	default:
		return 0x000000
	}
}

func LoreAtDepth(depth int) *LoreFragment {
	for _, l := range MiningLore {
		if l.Depth == depth {
			return &l
		}
	}
	return nil
}

type BagEntry struct {
	Name  string
	Count int
}

type MiningEvent struct {
	Type  string
	Items []BagEntry
	Buff  string
}

type DescendResult struct {
	Item      *MineItem
	Collapsed bool
	Bag       []BagEntry
	Event     *MiningEvent
	LoreID    string
}

type LeaveResult struct {
	XP      int
	Bag     []BagEntry
	Unlocks []*achievement.Achievement
	ToolID  string
}

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) GetMinerLevel(userID int64) (int, error) {
	var job model.Job
	err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "miner").First(&job).Error
	if err == gorm.ErrRecordNotFound {
		return 1, nil
	}
	if err != nil {
		return 1, err
	}
	return job.Level, nil
}

func (s *Service) HasLore(userID int64, loreID string) (bool, error) {
	var count int64
	err := s.store.DB.Model(&model.UserLoreEntry{}).
		Where("user_id = ? AND lore_id = ?", userID, loreID).
		Count(&count).Error
	return count > 0, err
}

func (s *Service) GrantLore(userID int64, loreID string) error {
	return s.store.DB.Create(&model.UserLoreEntry{
		UserID: userID,
		LoreID: loreID,
	}).Error
}

func (s *Service) GrantItem(userID int64, itemID string, quantity int) error {
	return s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
		DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + ?", quantity)}),
	}).Create(&model.Inventory{UserID: userID, ItemID: itemID, Quantity: quantity}).Error
}

func (s *Service) HasItem(userID int64, itemID string) (bool, error) {
	var inv model.Inventory
	err := s.store.DB.Where("user_id = ? AND item_id = ? AND quantity > 0", userID, itemID).First(&inv).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) ConsumeItem(userID int64, itemID string) error {
	res := s.store.DB.Model(&model.Inventory{}).
		Where("user_id = ? AND item_id = ? AND quantity > 0", userID, itemID).
		UpdateColumn("quantity", gorm.Expr("quantity - 1"))
	if res.Error != nil {
		return res.Error
	}
	return s.store.DB.Where("user_id = ? AND item_id = ? AND quantity <= 0", userID, itemID).
		Delete(&model.Inventory{}).Error
}

func (s *Service) Descend(userID int64, depth int, bag []BagEntry, choice BranchChoice, toolID string, ghostVeilTurns int) (*DescendResult, error) {
	ok, _, err := s.store.CheckGameLimit(userID, "mine_descend", dailyDescendLimit)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrMineLimit
	}

	ml, err := s.GetMinerLevel(userID)
	if err != nil {
		return nil, err
	}

	ti := GetToolInfo(toolID)
	if ti.MinLevel > ml {
		toolID = ""
		ti = GetToolInfo("")
	}

	risk := (depth - 1) * 5
	risk += choice.RiskMod()
	risk -= ti.RiskReduction

	levelReduc := int(float64(ml) * 1.5)
	risk -= levelReduc

	vitReduc := int(charsvc.GetVITReduction(s.store, userID) * 100)
	risk -= vitReduc

	if ghostVeilTurns > 0 {
		risk -= 10
	}

	if charsvc.HasBuff(s.store, userID, "reinforce") {
		risk = 0
		charsvc.ConsumeBuff(s.store, userID, "reinforce")
	}

	if risk < 0 {
		risk = 0
	}

	_ = s.store.IncrementGameLimit(userID, "mine_descend")

	roll := rand.Intn(100) + 1

	hiddenChamber := false
	if roll <= risk && rand.Intn(100) < 10 {
		hiddenChamber = true
		risk = 0
		roll = risk + 1
	}

	if roll <= risk {
		return &DescendResult{Collapsed: true, Bag: bag}, nil
	}

	var event *MiningEvent
	easterRoll := rand.Intn(100) + 1

	if depth >= 10 && easterRoll <= 1 {
		_ = s.GrantItem(userID, ancientAlloyItem, 1)
		event = &MiningEvent{Type: "ancient_forge", Items: []BagEntry{{Name: ancientAlloyItem, Count: 1}}}
	} else if depth >= 6 && easterRoll <= 3 {
		event = &MiningEvent{Type: "ghost_miner", Buff: ghostVeilBuff}
	} else if depth >= 7 && easterRoll <= 6 {
		if !hiddenChamber {
			extra := lootAtDepth(depth)
			if len(extra) > 0 {
				it := extra[rand.Intn(len(extra))]
				event = &MiningEvent{Type: "whispering_runes", Items: []BagEntry{{Name: it.Name, Count: 1}}}
				found := false
				for i, e := range bag {
					if e.Name == it.Name {
						bag[i].Count++
						found = true
						break
					}
				}
				if !found {
					bag = append(bag, BagEntry{Name: it.Name, Count: 1})
				}
			}
		}
	}

	var loreID string
	if lf := LoreAtDepth(depth); lf != nil {
		has, err := s.HasLore(userID, lf.ID)
		if err == nil && !has {
			_ = s.GrantLore(userID, lf.ID)
			if lf.ID == loreKethari {
				_ = s.GrantItem(userID, ancientCoreShardItem, 1)
			}
			loreID = lf.ID
		}
	}

	if hiddenChamber {
		bonusCount := 2 + rand.Intn(4)
		var chamberItems []BagEntry
		pool := lootAtDepth(depth)
		for i := 0; i < bonusCount; i++ {
			if len(pool) == 0 {
				break
			}
			it := pool[rand.Intn(len(pool))]
			found := false
			for j, e := range chamberItems {
				if e.Name == it.Name {
					chamberItems[j].Count++
					found = true
					break
				}
			}
			if !found {
				chamberItems = append(chamberItems, BagEntry{Name: it.Name, Count: 1})
			}
			found2 := false
			for j, e := range bag {
				if e.Name == it.Name {
					bag[j].Count++
					found2 = true
					break
				}
			}
			if !found2 {
				bag = append(bag, BagEntry{Name: it.Name, Count: 1})
			}
		}
		event = &MiningEvent{Type: "hidden_chamber", Items: chamberItems}
		return &DescendResult{Item: nil, Bag: bag, Event: event, LoreID: loreID}, nil
	}

	if !choice.AllowLoot() {
		return &DescendResult{Bag: bag, Event: event, LoreID: loreID}, nil
	}

	lvl := depth + choice.LootTierOffset() + ti.LootTierBonus
	if lvl < 1 {
		lvl = 1
	}
	pool := lootAtDepth(lvl)

	if charsvc.HasBuff(s.store, userID, "midas_touch") {
		var filtered []MineItem
		for _, it := range pool {
			if it.Value >= 50 {
				filtered = append(filtered, it)
			}
		}
		if len(filtered) > 0 {
			pool = filtered
			charsvc.ConsumeBuff(s.store, userID, "midas_touch")
		}
	} else if charsvc.HasBuff(s.store, userID, "nose_for_treasure") {
		var filtered []MineItem
		for _, it := range pool {
			if it.Value >= 100 {
				filtered = append(filtered, it)
			}
		}
		if len(filtered) > 0 {
			pool = filtered
		}
		charsvc.ConsumeBuff(s.store, userID, "nose_for_treasure")
	}

	if choice == BranchSearchVeins {
		var highValue []MineItem
		for _, it := range pool {
			if it.Value >= 25 {
				highValue = append(highValue, it)
			}
		}
		if len(highValue) > 0 {
			pool = highValue
		}
	}

	item := pool[rand.Intn(len(pool))]

	found := false
	for i, e := range bag {
		if e.Name == item.Name {
			bag[i].Count++
			found = true
			break
		}
	}
	if !found {
		bag = append(bag, BagEntry{Name: item.Name, Count: 1})
	}

	return &DescendResult{Item: &item, Bag: bag, Event: event, LoreID: loreID}, nil
}

func (s *Service) LeaveMine(userID int64, bag []BagEntry, toolID string) (*LeaveResult, error) {
	strMult := charsvc.GetSTRBonus(s.store, userID)
	if strMult > 1.0 {
		for i := range bag {
			bag[i].Count = int(float64(bag[i].Count) * strMult)
		}
	}

	if charsvc.HasBuff(s.store, userID, "scavenger") {
		for i := range bag {
			bag[i].Count += bag[i].Count / 2
		}
		charsvc.ConsumeBuff(s.store, userID, "scavenger")
	}

	totalXP := len(bag) * 10
	for _, e := range bag {
		totalXP += e.Count * 5
	}

	for _, e := range bag {
		if err := s.store.DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
			DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + ?", e.Count)}),
		}).Create(&model.Inventory{UserID: userID, ItemID: e.Name, Quantity: e.Count}).Error; err != nil {
			return nil, err
		}
	}

	if len(bag) > 0 {
		if err := achievement.IncrementStat(s.store.DB, userID, "items_mined", len(bag)); err != nil {
			return nil, err
		}
		if err := s.store.RecordActivity(userID, "items_mined", len(bag)); err != nil {
			return nil, err
		}
	}

	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "miner").First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			job = model.Job{UserID: userID, JobName: "miner", Level: 1, XP: totalXP}
			if err := s.store.DB.Create(&job).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		job.XP += totalXP
		next := jobXPForLevel(job.Level)
		for job.XP >= next {
			job.XP -= next
			job.Level++
			next = jobXPForLevel(job.Level)
		}
		if err := s.store.DB.Model(&model.Job{}).Where("user_id = ? AND job_name = ?", userID, "miner").
			Updates(map[string]any{"xp": job.XP, "level": job.Level}).Error; err != nil {
			return nil, err
		}
	}

	if toolID != "" {
		_ = s.ConsumeItem(userID, toolID)
	}

	charsvc.AddXP(s.store, userID, totalXP)

	unlocks, err := achievement.CheckAndUnlock(s.store.DB, userID)
	if err != nil {
		return nil, err
	}

	return &LeaveResult{XP: totalXP, Bag: bag, Unlocks: unlocks, ToolID: toolID}, nil
}

func jobXPForLevel(level int) int {
	return 50 + level*25
}

func LoreDisplayName(loreID string) string {
	for _, l := range MiningLore {
		if l.ID == loreID {
			return l.Title
		}
	}
	return loreID
}

func GhostVeilBuffID() string    { return ghostVeilBuff }
func LoreKethariID() string       { return loreKethari }
func AncientCoreShardItemID() string { return ancientCoreShardItem }




