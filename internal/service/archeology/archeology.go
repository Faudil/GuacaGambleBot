package archeology

import (
	"errors"
	"math/rand"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	furnituresvc "guacagamblebot/internal/service/furniture"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
)

const (
	JournalPageCount = 8

	sellBaseMult    = 1.2
	sellLuckyChance = 0.2
	sellLuckyMin    = 1.2
	sellLuckyMax    = 2.0
)

var (
	ErrDigLimit         = errors.New("dig daily limit reached")
	ErrNoMoney          = errors.New("not enough money")
	ErrFinished         = errors.New("game already finished")
	ErrLocked           = errors.New("site locked")
	ErrNoActions        = errors.New("no actions remaining")
	ErrNoFossils        = errors.New("not enough fossils")
	ErrNotGrindable     = errors.New("item cannot be ground into dust")
	ErrNotEnoughFossils = errors.New("not enough fossils to grind")
	ErrNoGeneticsLab    = errors.New("genetics lab furniture required")
	ErrResearchRequired = errors.New("reanimation research required")
)

// ReanimateResearch maps each reanimation pool tier to the research that must
// be completed in the genetics lab before it can be used.
var ReanimateResearch = map[string]string{
	"common":    "reanimate_common",
	"rare":      "reanimate_rare",
	"epic":      "reanimate_epic",
	"legendary": "reanimate_legendary",
	"pure_dna":  "reanimate_pure_dna",
}

// DustRates maps each grindable fossil to the bone dust it yields per unit.
// The rarer the fossil, the more dust it gives.
var DustRates = map[string]int{
	"damaged_fossil":     1,
	"common_fossil":      2,
	"rare_fossil":        5,
	"epic_fossil":        7,
	"legendary_fragment": 12,
	"pure_dna":           25,
	"cursed_artifact":    5,
	"purified_relic":     15,
	"shadow_fossil":      30,
}

// GrindableOrder lists the grindable fossils in display order (ascending rarity).
var GrindableOrder = []string{
	"damaged_fossil",
	"common_fossil",
	"rare_fossil",
	"epic_fossil",
	"legendary_fragment",
	"pure_dna",
	"cursed_artifact",
	"purified_relic",
	"shadow_fossil",
}

type LayerType int

const (
	LayerSoftSoil LayerType = iota
	LayerHardRock
	LayerGravel
	LayerClay
	LayerBedrock
)

type LayerProfile struct {
	Type   LayerType
	Depth  int
	Emoji  string
	NameID string
}

type ToolAction struct {
	ID       string
	NameID   string
	DepthRem int
	RiskPct  int
	IntLoss  int
	Emoji    string
}

type SiteDef struct {
	Key            string
	NameID         string
	DescID         string
	Depth          int
	Cost           int
	MinLevel       int
	LayerSeqs      [][]LayerType
	FossilRarities []string
	Color          int
}

var Tools = map[string]ToolAction{
	"dynamite": {ID: "dynamite", DepthRem: 20, RiskPct: 30, IntLoss: 20, Emoji: "🧨"},
	"hammer":   {ID: "hammer", DepthRem: 13, RiskPct: 15, IntLoss: 10, Emoji: "🔨"},
	"brush":    {ID: "brush", DepthRem: 3, RiskPct: 0, IntLoss: 0, Emoji: "🖌️"},
}

var LayerDefs = map[LayerType]struct {
	BaseNameID string
	Effects    map[string]ToolEffect
	Emoji      string
}{
	LayerSoftSoil: {
		BaseNameID: "layer_soft_soil",
		Emoji:      "🟫",
		Effects: map[string]ToolEffect{
			"dynamite": {DepthMul: 1, RiskMul: 0.8, IntMul: 1},
			"hammer":   {DepthMul: 1, RiskMul: 0.7, IntMul: 1},
			"brush":    {DepthMul: 2, RiskMul: 0, IntMul: 0},
		},
	},
	LayerHardRock: {
		BaseNameID: "layer_hard_rock",
		Emoji:      "🪨",
		Effects: map[string]ToolEffect{
			"dynamite": {DepthMul: 1, RiskMul: 1, IntMul: 1},
			"hammer":   {DepthMul: 1, RiskMul: 1, IntMul: 1},
			"brush":    {DepthMul: 0.5, RiskMul: 0, IntMul: 0},
		},
	},
	LayerGravel: {
		BaseNameID: "layer_gravel",
		Emoji:      "🪨",
		Effects: map[string]ToolEffect{
			"dynamite": {DepthMul: 1, RiskMul: 0.9, IntMul: 1.1},
			"hammer":   {DepthMul: 0.8, RiskMul: 0.7, IntMul: 0.5},
			"brush":    {DepthMul: 1, RiskMul: 0, IntMul: 0},
		},
	},
	LayerClay: {
		BaseNameID: "layer_clay",
		Emoji:      "🟤",
		Effects: map[string]ToolEffect{
			"dynamite": {DepthMul: 0.75, RiskMul: 0.6, IntMul: 0.7},
			"hammer":   {DepthMul: 0.7, RiskMul: 0.3, IntMul: 0.5},
			"brush":    {DepthMul: 2, RiskMul: 0, IntMul: 0},
		},
	},
	LayerBedrock: {
		BaseNameID: "layer_bedrock",
		Emoji:      "⬛",
		Effects: map[string]ToolEffect{
			"dynamite": {DepthMul: 0.75, RiskMul: 1.1, IntMul: 1.0},
			"hammer":   {DepthMul: 0.6, RiskMul: 1.5, IntMul: 1.2},
			"brush":    {DepthMul: 0, RiskMul: 0, IntMul: 0},
		},
	},
}

type ToolEffect struct {
	DepthMul float64
	RiskMul  float64
	IntMul   float64
}

type GameState struct {
	UserID        int64
	PermitType    string
	Depth         int
	MaxDepth      int
	Integrity     int
	Actions       int
	Finished      bool
	CurrentLayer  LayerType
	LayerSeq      []LayerType
	LayerIdx      int
	LastTool      string
	Site          *SiteDef
	CursedDebuff  bool
	RevealedLayer bool
}

type DigResult struct {
	ItemName    string
	Value       int
	Quality     string
	Integrity   int
	IsEgg       bool
	IsShadow    bool
	IsCursed    bool
	JournalPage string
	XP          int
	Quantity    int
}

type ActionOutcome struct {
	State      GameState
	DepthRem   int
	IntLoss    int
	Damaged    bool
	Finished   bool
	LayerShift bool
	Event      *DigEvent
}

type DigEvent struct {
	Type    EventType
	TitleID string
	DescID  string
	Choices []EventChoice
	Data    map[string]any
}

type ActionType string

const (
	ActionDynamite ActionType = "dynamite"
	ActionHammer   ActionType = "hammer"
	ActionBrush    ActionType = "brush"
	ActionScan     ActionType = "scan"
)

type EventType int

const (
	EventNone EventType = iota
	EventFossilWhisper
	EventCaveIn
	EventGuardian
	EventBuriedTreasure
	EventFossilEgg
)

type EventChoice struct {
	LabelID string
	Value   string
	Style   int
}

type EventResult struct {
	TitleID       string
	DescID        string
	CoinChange    int
	ActionsLost   int
	IntLoss       int
	ItemGiven     string
	ItemQty       int
	DepthGain     int
	RevealedTool  string
	RevealedLayer LayerType
	BackToDig     bool
}

type Service struct {
	store  *store.Store
	cfg    *config.Config
	npcSvc *npcsvc.Service
}

func New(s *store.Store, cfg *config.Config, npcSvc *npcsvc.Service) *Service {
	return &Service{store: s, cfg: cfg, npcSvc: npcSvc}
}

func (s *Service) NewGame(userID int64, siteKey string) (*GameState, error) {
	ok, _, err := s.store.CheckGameLimit(userID, "dig", 10)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrDigLimit
	}

	free, err := s.store.FreeSlots(s.store.DB, userID)
	if err != nil {
		return nil, err
	}
	if free <= 0 {
		return nil, store.ErrInventoryFull
	}

	site, ok := Sites[siteKey]
	if !ok {
		return nil, errors.New("unknown site")
	}

	if siteKey != "riverbed" && siteKey != "cliffside" {
		level := s.GetArcheologistLevel(userID)
		if level < site.MinLevel {
			return nil, ErrLocked
		}
	}
	if site.Cost > 0 {
		bal, err := s.store.GetBalance(userID)
		if err != nil {
			return nil, err
		}
		if bal < site.Cost {
			return nil, ErrNoMoney
		}
		if _, err := s.store.UpdateBalance(userID, -site.Cost); err != nil {
			return nil, err
		}
	}

	_ = s.store.IncrementGameLimit(userID, "dig")

	seq := site.LayerSeqs[rand.Intn(len(site.LayerSeqs))]
	currentLayer := seq[0]
	layerDepth := site.Depth / len(seq)

	var cursed bool
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, "cursed_artifact").First(&model.Inventory{}).Error; err == nil {
		cursed = true
	}

	return &GameState{
		UserID:       userID,
		PermitType:   siteKey,
		Depth:        layerDepth,
		MaxDepth:     site.Depth,
		Integrity:    100,
		Actions:      6,
		CurrentLayer: currentLayer,
		LayerSeq:     seq,
		LayerIdx:     0,
		Site:         site,
		CursedDebuff: cursed,
	}, nil
}

func (s *Service) ApplyAction(state *GameState, action ActionType) *ActionOutcome {
	if state.Finished {
		return &ActionOutcome{Finished: true}
	}
	if action == ActionScan {
		state.Actions--
		state.RevealedLayer = true
		if state.Actions <= 0 {
			state.Finished = true
		}
		return &ActionOutcome{
			State:      *state,
			LayerShift: false,
			Finished:   state.Finished,
		}
	}

	state.LastTool = string(action)
	tool := Tools[string(action)]
	layer := LayerDefs[state.CurrentLayer]
	effect := layer.Effects[string(action)]

	depthRem := int(float64(tool.DepthRem) * effect.DepthMul)
	if depthRem < 1 {
		depthRem = 1
	}

	masteryBonus := 1.0
	uses := s.getToolUses(state.UserID, string(action))
	if uses >= 200 {
		masteryBonus = 1.30
	} else if uses >= 100 {
		masteryBonus = 1.20
	} else if uses >= 50 {
		masteryBonus = 1.10
	}
	depthRem = int(float64(depthRem) * masteryBonus)

	if depthRem > state.Depth {
		depthRem = state.Depth
	}
	state.Depth -= depthRem

	riskPct := int(float64(tool.RiskPct) * effect.RiskMul)
	if state.CursedDebuff {
		riskPct += 10
	}
	if state.PermitType == "safe" || state.PermitType == "riverbed" {
		riskPct /= 2
	}
	damaged := false
	intLoss := 0
	if riskPct > 0 && rand.Intn(100) < riskPct {
		intLoss = int(float64(tool.IntLoss) * effect.IntMul)
		if intLoss < 1 {
			intLoss = 1
		}
		state.Integrity -= intLoss
		if state.Integrity < 0 {
			state.Integrity = 0
		}
		damaged = true
	}

	state.Actions--
	s.incrementToolUses(state.UserID, string(action))

	layerShift := false
	if state.Depth <= 0 && state.LayerIdx < len(state.LayerSeq)-1 {
		state.LayerIdx++
		state.CurrentLayer = state.LayerSeq[state.LayerIdx]
		state.Depth += state.Site.Depth / len(state.LayerSeq)
		layerShift = true
	}

	finished := state.Depth <= 0 || state.Integrity <= 0 || state.Actions <= 0
	state.Finished = finished

	return &ActionOutcome{
		State:      *state,
		DepthRem:   depthRem,
		IntLoss:    intLoss,
		Damaged:    damaged,
		Finished:   finished,
		LayerShift: layerShift,
	}
}

func (s *Service) GetToolEffectiveness(state *GameState, action ActionType) map[string]int {
	tool := Tools[string(action)]
	layer := LayerDefs[state.CurrentLayer]
	effect := layer.Effects[string(action)]
	depth := int(float64(tool.DepthRem) * effect.DepthMul)
	risk := int(float64(tool.RiskPct) * effect.RiskMul)
	return map[string]int{"depth": depth, "risk": risk}
}

func (s *Service) Resolve(state *GameState) *DigResult {
	if state.Integrity <= 0 {
		xp := 10
		return &DigResult{ItemName: "bone_dust", Value: 1, Quality: "disaster", Integrity: 0, XP: xp, Quantity: 1}
	}
	if state.Depth > 0 && state.Actions <= 0 {
		xp := 10
		return &DigResult{ItemName: "bone_dust", Value: 1, Quality: "disaster", Integrity: state.Integrity, XP: xp, Quantity: 1}
	}

	if state.Integrity < 50 {
		xp := 25
		return &DigResult{ItemName: "damaged_fossil", Value: 50, Quality: "damaged", Integrity: state.Integrity, XP: xp, Quantity: 1}
	}

	xpBase := state.Integrity / 2

	// A Magnetic Coil placed in the active house boosts rare find chances.
	digLuck := furnituresvc.EffectValue(s.store, state.UserID, "dig_luck")

	rarityPool := state.Site.FossilRarities

	if state.Integrity >= 95 && state.LastTool == "brush" {
		if rand.Float64() < 0.02+digLuck {
			xp := 200
			return &DigResult{ItemName: "coelacanth_egg", Value: 2500, Quality: "living", Integrity: state.Integrity, XP: xp, IsEgg: true, Quantity: 1}
		}
	}

	if state.Integrity < 15 && state.Integrity > 5 && state.LastTool != "brush" {
		if rand.Float64() < 0.15+digLuck {
			xp := 300
			return &DigResult{ItemName: "shadow_fossil", Value: 5000, Quality: "shadow", Integrity: state.Integrity, XP: xp, IsShadow: true, Quantity: 1}
		}
	}

	if state.Integrity < 30 && state.LastTool == "dynamite" {
		if rand.Float64() < 0.05+digLuck {
			xp := 100
			return &DigResult{ItemName: "cursed_artifact", Value: 800, Quality: "cursed", Integrity: state.Integrity, XP: xp, IsCursed: true, Quantity: 1}
		}
	}

	journalRoll := rand.Float64()
	if journalRoll < 0.01+digLuck {
		for i := 1; i <= JournalPageCount; i++ {
			pageID := itoa(i)
			var inv model.Inventory
			if err := s.store.DB.Where("user_id = ? AND item_id = ?", state.UserID, "journal_page_"+pageID).First(&inv).Error; err != nil {
				xp := 50
				return &DigResult{ItemName: "journal_page_" + pageID, Value: 1, Quality: "journal", Integrity: state.Integrity, XP: xp, JournalPage: "journal_page_" + pageID, Quantity: 1}
			}
		}
	}

	xp := xpBase
	var chosen string
	if len(rarityPool) == 1 {
		chosen = rarityPool[0]
	} else {
		totalWeight := 0
		weights := make([]int, len(rarityPool))
		for i, r := range rarityPool {
			w := fossilWeight(r)
			weights[i] = w
			totalWeight += w
		}
		roll := rand.Intn(totalWeight)
		cumulative := 0
		for i, w := range weights {
			cumulative += w
			if roll < cumulative {
				chosen = rarityPool[i]
				break
			}
		}
	}

	itemMap := map[string]string{
		"common":    "common_fossil",
		"rare":      "rare_fossil",
		"epic":      "epic_fossil",
		"legendary": "legendary_fragment",
		"pure_dna":  "pure_dna",
	}
	valueMap := map[string]int{
		"common": 150, "rare": 300, "epic": 500, "legendary": 1000, "pure_dna": 3000,
	}

	itemID := itemMap[chosen]
	val := valueMap[chosen]
	xp += val / 2

	quantity := 1
	if state.Integrity >= 100 {
		quantity = 2
	}
	level := s.GetArcheologistLevel(state.UserID)
	quantity += level / 5
	if quantity < 1 {
		quantity = 1
	}
	if quantity > 10 {
		quantity = 10
	}

	return &DigResult{ItemName: itemID, Value: val, Quality: chosen, Integrity: state.Integrity, XP: xp, Quantity: quantity}
}

func (s *Service) GetArcheologistLevel(userID int64) int {
	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "archeologist").First(&job).Error; err != nil {
		return 0
	}
	return job.Level
}

func (s *Service) GetArcheologistXP(userID int64) (int, int) {
	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "archeologist").First(&job).Error; err != nil {
		return 0, 50
	}
	next := 50 + job.Level*25
	return job.XP, next
}

func (s *Service) addArcheologistXP(db *gorm.DB, userID int64, xp int) {
	var job model.Job
	if err := db.Where("user_id = ? AND job_name = ?", userID, "archeologist").First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			job = model.Job{UserID: userID, JobName: "archeologist", Level: 1, XP: xp}
			s.levelUpJob(&job)
			if err := db.Create(&job).Error; err != nil {
				return
			}
		}
		return
	}
	job.XP += xp
	s.levelUpJob(&job)
	db.Model(&model.Job{}).Where("user_id = ? AND job_name = ?", userID, "archeologist").
		Updates(map[string]any{"xp": job.XP, "level": job.Level})
}

// levelUpJob applies as many level-ups as the job's XP warrants.
func (s *Service) levelUpJob(job *model.Job) {
	next := 50 + job.Level*25
	for job.XP >= next {
		job.XP -= next
		job.Level++
		next = 50 + job.Level*25
	}
}

type SiteInfo struct {
	Key      string
	NameID   string
	DescID   string
	Cost     int
	MinLevel int
	Depth    int
	Color    int
	Unlocked bool
}

func (s *Service) GetSiteInfo(userID int64) []SiteInfo {
	level := s.GetArcheologistLevel(userID)
	var infos []SiteInfo
	for _, site := range Sites {
		unlocked := true
		if site.MinLevel > level {
			unlocked = false
		}
		infos = append(infos, SiteInfo{
			Key:      site.Key,
			NameID:   site.NameID,
			DescID:   site.DescID,
			Cost:     site.Cost,
			MinLevel: site.MinLevel,
			Depth:    site.Depth,
			Color:    site.Color,
			Unlocked: unlocked,
		})
	}
	return infos
}

func (s *Service) AwardResult(userID int64, res *DigResult) error {
	// Reputation uses its own connection, so it stays outside the transaction
	// below (a second connection writing while the transaction holds the
	// write lock would only stall on busy_timeout).
	s.addDigReputation(userID, res.Quality)

	err := s.store.DB.Transaction(func(tx *gorm.DB) error {
		if res.ItemName != "" {
			qty := res.Quantity
			if qty < 1 {
				qty = 1
			}
			if err := s.store.AddItemRaw(tx, userID, res.ItemName, qty); err != nil {
				return err
			}
			s.trackFossilHarvest(tx, userID, res.ItemName, qty)
		}
		if res.XP > 0 {
			s.addArcheologistXP(tx, userID, res.XP)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return s.store.RecordActivity(userID, "items_digged", 1)
}

// addDigReputation awards a small reputation bonus scaled by the rarity of
// the dig result with the linked NPC (ZARA in scifi).
func (s *Service) addDigReputation(userID int64, quality string) {
	points := map[string]int{
		"damaged":   1,
		"common":    1,
		"rare":      2,
		"epic":      3,
		"journal":   3,
		"legendary": 4,
		"living":    5,
		"cursed":    5,
		"pure_dna":  5,
		"shadow":    10,
	}[quality]
	if points > 0 {
		s.npcSvc.AddActivityReputation(userID, "archeology", points)
	}
}

func (s *Service) trackFossilHarvest(db *gorm.DB, userID int64, fossilID string, quantity int) {
	// Single atomic upsert instead of a read-then-write, so concurrent awards
	// can never lose an increment.
	_ = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "fossil_id"}},
		DoUpdates: clause.Assignments(map[string]any{"count": gorm.Expr("count + ?", quantity)}),
	}).Create(&model.UserFossilHarvest{UserID: userID, FossilID: fossilID, Count: quantity}).Error
}

func (s *Service) SellResult(userID int64, res *DigResult) (price, newBal int, lucky bool, mult float64, err error) {
	price, lucky, mult = sellPrice(res.Value, res.Quantity, rand.Float64())

	err = s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.store.UpdateBalanceTx(tx, userID, price); err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("user_id = ?", userID).Pluck("balance", &newBal).Error; err != nil {
			return err
		}
		if res.ItemName != "" {
			qty := res.Quantity
			if qty < 1 {
				qty = 1
			}
			s.trackFossilHarvest(tx, userID, res.ItemName, qty)
		}
		semiXP := res.XP / 2
		if semiXP > 0 {
			s.addArcheologistXP(tx, userID, semiXP)
		}
		return nil
	})
	if err != nil {
		return 0, 0, false, 0, err
	}
	return price, newBal, lucky, mult, nil
}

// sellPrice computes the sell price for a dig result: the base value scaled by
// the number of fossils found and a fixed premium, with a lucky chance of
// rolling a higher multiplier. The roll is passed in so the logic stays
// deterministic and testable.
func sellPrice(value, qty int, roll float64) (price int, lucky bool, mult float64) {
	if qty < 1 {
		qty = 1
	}
	price = int(float64(value) * sellBaseMult * float64(qty))
	if roll >= sellLuckyChance {
		return price, false, 1.0
	}
	mult = sellLuckyMin + (roll/sellLuckyChance)*(sellLuckyMax-sellLuckyMin)
	if mult > sellLuckyMax {
		mult = sellLuckyMax
	}
	if mult < sellLuckyMin {
		mult = sellLuckyMin
	}
	return int(float64(price) * mult), true, mult
}

// GrindFossils converts fossils of the given item into bone dust. The dust
// yield scales with the fossil's rarity (see DustRates).
func (s *Service) GrindFossils(userID int64, itemID string, quantity int) (int, error) {
	rate, ok := DustRates[itemID]
	if !ok {
		return 0, ErrNotGrindable
	}
	if quantity < 1 {
		return 0, ErrNotEnoughFossils
	}

	dust := 0
	err := s.store.DB.Transaction(func(tx *gorm.DB) error {
		var inv model.Inventory
		if err := tx.Where("user_id = ? AND item_id = ? AND quantity >= ?", userID, itemID, quantity).First(&inv).Error; err != nil {
			return ErrNotEnoughFossils
		}
		if err := tx.Model(&model.Inventory{}).
			Where("user_id = ? AND item_id = ?", userID, itemID).
			UpdateColumn("quantity", gorm.Expr("quantity - ?", quantity)).Error; err != nil {
			return err
		}
		dust = rate * quantity
		return s.store.AddItemRaw(tx, userID, "bone_dust", dust)
	})
	if err != nil {
		return 0, err
	}
	return dust, nil
}

func (s *Service) Reanimate(userID int64, rarity string) (petName string, success bool, err error) {
	pool, ok := ReanimatePools[rarity]
	if !ok {
		return "", false, errors.New("invalid rarity")
	}

	// Reanimation is gated behind the Genetics Lab furniture and the
	// one-time research for the fossil tier.
	if !furnituresvc.HasFurniture(s.store, userID, "genetics_lab") {
		return "", false, ErrNoGeneticsLab
	}
	researchID := ReanimateResearch[rarity]
	var r model.UserResearch
	if err := s.store.DB.Where("user_id = ? AND research_id = ? AND completed = ?", userID, researchID, true).First(&r).Error; err != nil {
		return "", false, ErrResearchRequired
	}

	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ? AND quantity >= ?", userID, pool.ItemName, 5).First(&inv).Error; err != nil {
		return "", false, ErrNoFossils
	}

	level := s.GetArcheologistLevel(userID)
	successRate := 0.50 + float64(level)*0.02
	if successRate > 0.90 {
		successRate = 0.90
	}

	if rand.Float64() < successRate {
		if inv.Quantity <= 5 {
			s.store.DB.Delete(&inv)
		} else {
			s.store.DB.Model(&inv).UpdateColumn("quantity", gorm.Expr("quantity - 5"))
		}

		petName := pool.Pets[rand.Intn(len(pool.Pets))]
		pet := model.UserPet{
			UserID:   userID,
			PetType:  petName,
			Nickname: petName,
			Level:    1,
			XP:       0,
			MaxHP:    50,
			HP:       50,
			Atk:      10,
			Defense:  5,
			Speed:    10,
			DGE:      5,
			ACC:      0,
			CritC:    5,
			CritD:    1.5,
			IsActive: false,
		}
		if err := s.store.DB.Create(&pet).Error; err != nil {
			return "", false, err
		}
		s.addArcheologistXP(s.store.DB, userID, 100)
		return petName, true, nil
	}

	if inv.Quantity <= 3 {
		s.store.DB.Delete(&inv)
	} else {
		s.store.DB.Model(&inv).UpdateColumn("quantity", gorm.Expr("quantity - 3"))
	}
	s.addArcheologistXP(s.store.DB, userID, 25)
	return "", false, nil
}

func (s *Service) GetFossilCount(userID int64, itemID string) int {
	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, itemID).First(&inv).Error; err != nil {
		return 0
	}
	return inv.Quantity
}

func fossilWeight(rarity string) int {
	switch rarity {
	case "common":
		return 50
	case "rare":
		return 30
	case "epic":
		return 15
	case "legendary":
		return 4
	case "pure_dna":
		return 1
	}
	return 10
}

func (s *Service) getToolUses(userID int64, toolID string) int {
	col := "tool_" + toolID + "_uses"
	var val int
	s.store.DB.Model(&model.UserStat{}).Where("user_id = ?", userID).Pluck(col, &val)
	return val
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

func GetLayerNameID(lt LayerType) string {
	switch lt {
	case LayerSoftSoil:
		return "arch.layer_soft_soil"
	case LayerHardRock:
		return "arch.layer_hard_rock"
	case LayerGravel:
		return "arch.layer_gravel"
	case LayerClay:
		return "arch.layer_clay"
	case LayerBedrock:
		return "arch.layer_bedrock"
	}
	return ""
}

func GetLayerEmoji(lt LayerType) string {
	switch lt {
	case LayerSoftSoil:
		return "🟫"
	case LayerHardRock:
		return "🪨"
	case LayerGravel:
		return "🔘"
	case LayerClay:
		return "🟤"
	case LayerBedrock:
		return "⬛"
	}
	return "❓"
}

var ReanimatePools = map[string]struct {
	ItemName string
	Pets     []string
}{
	"common":    {ItemName: "common_fossil", Pets: []string{"Trilobite", "Ammonite", "Anomalocaris", "Orthoceras", "Méganeura"}},
	"rare":      {ItemName: "rare_fossil", Pets: []string{"Archéoptéryx", "Ptéranodon", "Dimétrodon", "Smilodon", "Mégalocéros", "Doedicurus"}},
	"epic":      {ItemName: "epic_fossil", Pets: []string{"Mosasaurus", "Titanoboa", "Phorusrhacos", "Rhinocéros laineux", "Entelodon"}},
	"legendary": {ItemName: "legendary_fragment", Pets: []string{"Dragon", "Tyrannosaure", "Diplodocus", "Mamouth"}},
	"pure_dna":  {ItemName: "pure_dna", Pets: []string{"Mégalodon", "Kraken", "Licorne", "Phoenix", "Cerbère"}},
}
