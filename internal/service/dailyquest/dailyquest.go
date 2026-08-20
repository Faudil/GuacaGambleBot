package dailyquest

import (
	"errors"
	"math/rand"
	"strconv"
	"time"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/model"
	jsvc "guacagamblebot/internal/service/journal"
	npcsvc "guacagamblebot/internal/service/npcs"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
)

// ErrNoTemplate is returned when no activity template can be assembled for a
// user (e.g. no accessible hunt zone).
var ErrNoTemplate = errors.New("no daily quest template available")

// ── Tuning constants ────────────────────────────────────────────
// Kept in one place so balancing and future config plumbing touch nothing
// else. Later axes (new step kinds, requestors, flavor) do not need them.
const (
	levelDivisor        = 5  // activity counts grow by +1 every N character levels
	activityCountCap    = 12 // hard cap on activity step counts
	turnInCountCap      = 6  // hard cap on turn-in quantities
	antiRepeatWindow    = 5  // days a requestor/turn-in is rested
	jackpotPerStreakPct = 10 // jackpot chance gained per completed day
	jackpotMaxPct       = 100
	streakMoneyPerDay   = 25 // bonus credits per streak day
	streakMoneyCap      = 250
	// noRepBonus is paid instead of NPC reputation (e.g. the Town Board).
	noRepBonus = 50
)

// activityTemplate describes one possible activity step of a daily quest.
type activityTemplate struct {
	Stat    string
	Zone    string // set for zone-specific hunts (short zone key)
	Boss    bool   // set for "defeat your current boss" steps (stat resolved at generation)
	Min     int
	Max     int
	TextKey string
}

// turnInTemplate describes one possible final deliver step of a daily quest.
// TextKeys holds the requestor's reason lines, one chosen at generation so
// recurring requests read as the NPC's daily business rather than a loop.
type turnInTemplate struct {
	Item     string
	Min      int
	Max      int
	TextKeys []string
}

// requestorTemplate is the evergreen, repeatable request pool of one NPC.
// New requestors only need an entry here — weighting, gating and anti-repeat
// apply automatically.
type requestorTemplate struct {
	NPC         string
	TitleKey    string
	IntroKey    string
	ThankKeys   []string // completion lines, one chosen at generation
	Activities  []activityTemplate
	TurnIns     []turnInTemplate
	RewardItems []string
	RepPoints   int
	MinRank     int  // minimum journal rank required to be asked (0 = anyone)
	NoRep       bool // requestor grants no reputation (e.g. the Town Board)
}

// moodKeys are the atmospheric intro prefixes shared across requestors. The
// quest fiction is evergreen: the village's mood shifts, the requests recur.
var moodKeys = []string{
	"quests.daily.mood.fog",
	"quests.daily.mood.generator",
	"quests.daily.mood.rain",
	"quests.daily.mood.wind",
	"quests.daily.mood.dawn",
	"quests.daily.mood.dusk",
	"quests.daily.mood.cold",
	"quests.daily.mood.quiet",
	"quests.daily.mood.market",
	"quests.daily.mood.stars",
}

// requestors lists the NPCs who may ask for a daily quest. Each pool mixes
// generic activities, zone hunts, craft/use/boss objectives and a final item
// delivery with personal reason lines.
var requestors = []requestorTemplate{
	{
		NPC: "elara", TitleKey: "quests.daily.elara.title", IntroKey: "quests.daily.elara.intro", ThankKeys: []string{"quests.daily.elara.thanks1", "quests.daily.elara.thanks2"}, RepPoints: 30,
		Activities: []activityTemplate{
			{Stat: "items_farmed", Min: 3, Max: 5, TextKey: "quests.daily.step.farm"},
			{Stat: "pets_fed", Min: 2, Max: 3, TextKey: "quests.daily.step.pets"},
			{Stat: "items_digged", Min: 2, Max: 3, TextKey: "quests.daily.step.dig"},
			{Stat: "items_crafted", Min: 1, Max: 2, TextKey: "quests.daily.step.craft"},
		},
		TurnIns: []turnInTemplate{
			{Item: "wheat", Min: 2, Max: 3, TextKeys: []string{"quests.daily.elara.turnin_wheat", "quests.daily.elara.turnin_wheat_b"}},
			{Item: "wheat_seed", Min: 2, Max: 3, TextKeys: []string{"quests.daily.elara.turnin_seed", "quests.daily.elara.turnin_seed_b"}},
			{Item: "stonebread", Min: 1, Max: 1, TextKeys: []string{"quests.daily.elara.turnin_bread", "quests.daily.elara.turnin_bread_b"}},
			{Item: "carrot", Min: 2, Max: 3, TextKeys: []string{"quests.daily.elara.turnin_carrot"}},
			{Item: "zephyr_berries", Min: 1, Max: 1, TextKeys: []string{"quests.daily.elara.turnin_berries"}},
		},
		RewardItems: []string{"wheat_seed", "zephyr_berries", "growth_elixir"},
	},
	{
		NPC: "thorek", TitleKey: "quests.daily.thorek.title", IntroKey: "quests.daily.thorek.intro", ThankKeys: []string{"quests.daily.thorek.thanks1", "quests.daily.thorek.thanks2"}, RepPoints: 30,
		Activities: []activityTemplate{
			{Stat: "items_mined", Min: 3, Max: 5, TextKey: "quests.daily.step.mine"},
			{Stat: "items_sold_market", Min: 1, Max: 2, TextKey: "quests.daily.step.sell"},
			{Stat: "hunt", Zone: "cave", Min: 2, Max: 3, TextKey: "quests.daily.step.hunt_zone"},
			{Stat: "hunt", Zone: "mountain", Min: 2, Max: 3, TextKey: "quests.daily.step.hunt_zone"},
			{Stat: "items_crafted", Min: 1, Max: 2, TextKey: "quests.daily.step.craft"},
		},
		TurnIns: []turnInTemplate{
			{Item: "coal", Min: 2, Max: 3, TextKeys: []string{"quests.daily.thorek.turnin_coal", "quests.daily.thorek.turnin_coal_b"}},
			{Item: "iron_ore", Min: 2, Max: 3, TextKeys: []string{"quests.daily.thorek.turnin_iron", "quests.daily.thorek.turnin_iron_b"}},
			{Item: "iron_loaf", Min: 1, Max: 1, TextKeys: []string{"quests.daily.thorek.turnin_loaf"}},
			{Item: "rusty_magnet", Min: 1, Max: 1, TextKeys: []string{"quests.daily.thorek.turnin_magnet"}},
			{Item: "gold_nugget", Min: 1, Max: 1, TextKeys: []string{"quests.daily.thorek.turnin_nugget"}},
		},
		RewardItems: []string{"coal", "iron_ore", "gold_nugget"},
	},
	{
		NPC: "irian", TitleKey: "quests.daily.irian.title", IntroKey: "quests.daily.irian.intro", ThankKeys: []string{"quests.daily.irian.thanks1", "quests.daily.irian.thanks2"}, RepPoints: 30,
		Activities: []activityTemplate{
			{Stat: "items_fished", Min: 3, Max: 5, TextKey: "quests.daily.step.fish"},
			{Stat: "items_hunted", Min: 2, Max: 3, TextKey: "quests.daily.step.hunt"},
			{Stat: "hunt", Zone: "ocean", Min: 2, Max: 3, TextKey: "quests.daily.step.hunt_zone"},
			{Stat: "hunt", Zone: "forest", Min: 2, Max: 3, TextKey: "quests.daily.step.hunt_zone"},
			{Stat: "items_crafted", Min: 1, Max: 2, TextKey: "quests.daily.step.craft"},
		},
		TurnIns: []turnInTemplate{
			{Item: "sardine", Min: 2, Max: 3, TextKeys: []string{"quests.daily.irian.turnin_sardine", "quests.daily.irian.turnin_sardine_b"}},
			{Item: "trout", Min: 1, Max: 2, TextKeys: []string{"quests.daily.irian.turnin_trout", "quests.daily.irian.turnin_trout_b"}},
			{Item: "crayfish", Min: 1, Max: 2, TextKeys: []string{"quests.daily.irian.turnin_crayfish"}},
			{Item: "worm", Min: 2, Max: 3, TextKeys: []string{"quests.daily.irian.turnin_worm"}},
			{Item: "warrior_stew", Min: 1, Max: 1, TextKeys: []string{"quests.daily.irian.turnin_stew"}},
		},
		RewardItems: []string{"worm", "crayfish", "golden_lure"},
	},
	{
		NPC: "sheriff_vance", TitleKey: "quests.daily.vance.title", IntroKey: "quests.daily.vance.intro", ThankKeys: []string{"quests.daily.vance.thanks1", "quests.daily.vance.thanks2"}, RepPoints: 30,
		Activities: []activityTemplate{
			{Stat: "items_hunted", Min: 2, Max: 3, TextKey: "quests.daily.step.hunt"},
			{Stat: "items_sold_market", Min: 1, Max: 2, TextKey: "quests.daily.step.sell"},
			{Stat: "hunt", Zone: "desert", Min: 2, Max: 3, TextKey: "quests.daily.step.hunt_zone"},
			{Stat: "hunt", Zone: "cave", Min: 2, Max: 3, TextKey: "quests.daily.step.hunt_zone"},
			{Stat: "items_crafted", Min: 1, Max: 2, TextKey: "quests.daily.step.craft"},
		},
		TurnIns: []turnInTemplate{
			{Item: "iron_ore", Min: 2, Max: 3, TextKeys: []string{"quests.daily.vance.turnin_iron", "quests.daily.vance.turnin_iron_b"}},
			{Item: "wanted_poster", Min: 1, Max: 1, TextKeys: []string{"quests.daily.vance.turnin_poster"}},
			{Item: "worm", Min: 2, Max: 3, TextKeys: []string{"quests.daily.vance.turnin_worm"}},
			{Item: "coal", Min: 2, Max: 3, TextKeys: []string{"quests.daily.vance.turnin_coal"}},
		},
		RewardItems: []string{"iron_ore", "wanted_poster", "reinforced_badge"},
	},
	{
		NPC: "the_whisper", TitleKey: "quests.daily.whisper.title", IntroKey: "quests.daily.whisper.intro", ThankKeys: []string{"quests.daily.whisper.thanks1", "quests.daily.whisper.thanks2"}, RepPoints: 30,
		Activities: []activityTemplate{
			{Stat: "items_digged", Min: 2, Max: 3, TextKey: "quests.daily.step.dig"},
			{Stat: "items_sold_market", Min: 1, Max: 2, TextKey: "quests.daily.step.sell"},
			{Stat: "items_used", Min: 1, Max: 3, TextKey: "quests.daily.step.use"},
		},
		TurnIns: []turnInTemplate{
			{Item: "smoke_pellet", Min: 1, Max: 1, TextKeys: []string{"quests.daily.whisper.turnin_smoke", "quests.daily.whisper.turnin_smoke_b"}},
			{Item: "rusty_magnet", Min: 1, Max: 1, TextKeys: []string{"quests.daily.whisper.turnin_magnet"}},
			{Item: "coal", Min: 2, Max: 3, TextKeys: []string{"quests.daily.whisper.turnin_coal"}},
			{Item: "crayfish", Min: 1, Max: 2, TextKeys: []string{"quests.daily.whisper.turnin_crayfish"}},
			{Item: "wanted_poster", Min: 1, Max: 1, TextKeys: []string{"quests.daily.whisper.turnin_poster"}},
		},
		RewardItems: []string{"smoke_pellet", "rusty_magnet", "scratch_ticket"},
	},
	{
		NPC: "gamblebot", TitleKey: "quests.daily.gamblebot.title", IntroKey: "quests.daily.gamblebot.intro", ThankKeys: []string{"quests.daily.gamblebot.thanks1", "quests.daily.gamblebot.thanks2"}, RepPoints: 30,
		Activities: []activityTemplate{
			{Stat: "casino_games_played", Min: 3, Max: 5, TextKey: "quests.daily.step.casino"},
			{Stat: "items_sold_market", Min: 1, Max: 2, TextKey: "quests.daily.step.sell"},
			{Boss: true, Min: 1, Max: 1, TextKey: "quests.daily.step.boss"},
			{Stat: "items_crafted", Min: 1, Max: 2, TextKey: "quests.daily.step.craft"},
		},
		TurnIns: []turnInTemplate{
			{Item: "sardine", Min: 1, Max: 2, TextKeys: []string{"quests.daily.gamblebot.turnin_fish", "quests.daily.gamblebot.turnin_fish_b"}},
			{Item: "trout", Min: 1, Max: 1, TextKeys: []string{"quests.daily.gamblebot.turnin_fish"}},
			{Item: "beer", Min: 1, Max: 1, TextKeys: []string{"quests.daily.gamblebot.turnin_beer"}},
			{Item: "casino_token", Min: 1, Max: 1, TextKeys: []string{"quests.daily.gamblebot.turnin_token"}},
		},
		RewardItems: []string{"scratch_ticket", "casino_token", "rigged_coin"},
	},
	{
		NPC: "town_board", TitleKey: "quests.daily.board.title", IntroKey: "quests.daily.board.intro", ThankKeys: []string{"quests.daily.board.thanks1", "quests.daily.board.thanks2"}, NoRep: true,
		Activities: []activityTemplate{
			{Stat: "items_mined", Min: 3, Max: 5, TextKey: "quests.daily.step.mine"},
			{Stat: "items_farmed", Min: 3, Max: 5, TextKey: "quests.daily.step.farm"},
			{Stat: "items_fished", Min: 3, Max: 5, TextKey: "quests.daily.step.fish"},
			{Stat: "items_hunted", Min: 2, Max: 3, TextKey: "quests.daily.step.hunt"},
		},
		TurnIns: []turnInTemplate{
			{Item: "coal", Min: 2, Max: 3, TextKeys: []string{"quests.daily.board.turnin_coal"}},
			{Item: "wheat", Min: 2, Max: 3, TextKeys: []string{"quests.daily.board.turnin_wheat"}},
			{Item: "sardine", Min: 2, Max: 3, TextKeys: []string{"quests.daily.board.turnin_sardine"}},
		},
		RewardItems: []string{"iron_ore", "wheat_seed", "worm"},
	},
	{
		NPC: "the_chronicler", TitleKey: "quests.daily.chronicler.title", IntroKey: "quests.daily.chronicler.intro", ThankKeys: []string{"quests.daily.chronicler.thanks1", "quests.daily.chronicler.thanks2"}, RepPoints: 40, MinRank: 1,
		Activities: []activityTemplate{
			{Stat: "delve_completions", Min: 1, Max: 2, TextKey: "quests.daily.step.delve"},
			{Stat: "expedition_completions", Min: 1, Max: 2, TextKey: "quests.daily.step.expedition"},
			{Stat: "zone_bosses_defeated", Min: 1, Max: 1, TextKey: "quests.daily.step.zone_boss"},
		},
		TurnIns: []turnInTemplate{
			{Item: "gold_nugget", Min: 1, Max: 1, TextKeys: []string{"quests.daily.chronicler.turnin_nugget"}},
			{Item: "platinum", Min: 1, Max: 1, TextKeys: []string{"quests.daily.chronicler.turnin_platinum"}},
			{Item: "stonebread", Min: 1, Max: 1, TextKeys: []string{"quests.daily.chronicler.turnin_bread"}},
		},
		RewardItems: []string{"gold_nugget", "stonebread"},
	},
}

// startZones mirrors the hunt zones open from the start (huntsvc.FirstZones);
// progressive zones are added to PlayerContext.AccessibleZones once unlocked.
var startZones = []string{"forest", "cave", "desert"}

var progressiveZones = []string{"mountain", "ocean", "tundra", "volcano"}

// PlayerContext is the per-user state the generator needs. It is fetched once
// per generation (loadContext) so the selection policies stay pure, unit
// testable, and free of database access.
type PlayerContext struct {
	Level            int
	Affinity         map[string]int // npc id -> reputation level
	Recent           []store.DailyHistoryEntry
	Streak           int
	JournalRank      int // gates the Chronicler requestor
	CurrentBossStage int // boss_league battle step index, -1 when none
	AccessibleZones  map[string]bool
}

// Service generates, advances and settles the procedural daily quest.
type Service struct {
	store *store.Store
	npcs  *npcsvc.Service
}

func New(st *store.Store, npcs *npcsvc.Service) *Service {
	return &Service{store: st, npcs: npcs}
}

// Generate assembles a fresh daily quest recipe for the user and logs it for
// anti-repeat and streak tracking.
func (s *Service) Generate(userID int64) (store.DailyRecipe, error) {
	recipe, err := s.generate(s.loadContext(userID))
	if err != nil {
		return recipe, err
	}
	if err := s.store.LogDailyQuest(userID, recipe.Requestor, recipe.TurnInItem()); err != nil {
		return recipe, err
	}
	return recipe, nil
}

// loadContext gathers the user state used by the selection policies.
func (s *Service) loadContext(userID int64) PlayerContext {
	ctx := PlayerContext{
		Affinity:         map[string]int{},
		AccessibleZones:  map[string]bool{},
		CurrentBossStage: -1,
	}
	for _, z := range startZones {
		ctx.AccessibleZones[z] = true
	}
	if level, err := s.store.CharacterLevel(userID); err == nil {
		ctx.Level = level
	}
	ctx.JournalRank = jsvc.HighestRank(s.store, userID)
	ctx.CurrentBossStage = s.currentBossStage(userID)
	for _, req := range requestors {
		var rep model.UserNPCReputation
		if err := s.store.DB.Where("user_id = ? AND npc_id = ?", userID, req.NPC).First(&rep).Error; err == nil {
			ctx.Affinity[req.NPC] = rep.Level
		}
	}
	if recent, err := s.store.RecentDailyQuests(userID, antiRepeatWindow); err == nil {
		ctx.Recent = recent
	}
	ctx.Streak = s.store.DailyStreak(userID)
	for _, z := range progressiveZones {
		if ok, _ := s.store.HasUnlockedZone(userID, z); ok {
			ctx.AccessibleZones[z] = true
		}
	}
	return ctx
}

// currentBossStage returns the boss_stage of the player's current boss_league
// battle step, or -1 when no boss is currently fightable.
func (s *Service) currentBossStage(userID int64) int {
	def := questssvc.QuestRegistry["boss_league"]
	if def == nil {
		return -1
	}
	var uq model.UserQuest
	if err := s.store.DB.Where("user_id = ? AND quest_id = 'boss_league'", userID).First(&uq).Error; err != nil || uq.Status != "ACTIVE" {
		return -1
	}
	var uqd model.UserQuestData
	if err := s.store.DB.Where("user_id = ? AND quest_id = 'boss_league'", userID).First(&uqd).Error; err != nil {
		return -1
	}
	if uqd.StepIndex >= len(def.Steps) {
		return -1
	}
	step := def.Steps[uqd.StepIndex]
	if step.Type != questssvc.StepBossBattle {
		return -1
	}
	return toInt(step.Extra["boss_stage"])
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}

// generate assembles a recipe from the player context: one affinity-weighted,
// rested requestor, 1-2 scaled activity steps and a rested turn-in, plus a
// reward scaled by level and streak. It is pure — no database access.
func (s *Service) generate(ctx PlayerContext) (store.DailyRecipe, error) {
	req := pickRequestor(ctx)
	steps := buildSteps(req, ctx)
	if len(steps) < 2 {
		return store.DailyRecipe{}, ErrNoTemplate
	}
	return store.DailyRecipe{
		Requestor: req.NPC,
		TitleKey:  req.TitleKey,
		IntroKey:  req.IntroKey,
		MoodKey:   moodKeys[rand.Intn(len(moodKeys))],
		ThankKey:  req.ThankKeys[rand.Intn(len(req.ThankKeys))],
		Steps:     steps,
		Reward:    rollReward(ctx, req, len(steps), time.Now()),
	}, nil
}

// pickRequestor selects a requestor weighted by the player's affinity with
// them (weight 1 + reputation level), resting NPCs seen recently and skipping
// requestors whose journal-rank gate is not met.
func pickRequestor(ctx PlayerContext) requestorTemplate {
	pool := requestors
	var rested []requestorTemplate
	for _, r := range requestors {
		if r.MinRank > ctx.JournalRank || ctx.recentlySeenRequestor(r.NPC) {
			continue
		}
		rested = append(rested, r)
	}
	if len(rested) > 0 {
		pool = rested
	}
	total := 0
	for _, r := range pool {
		total += 1 + ctx.Affinity[r.NPC]
	}
	roll := rand.Intn(total)
	for _, r := range pool {
		roll -= 1 + ctx.Affinity[r.NPC]
		if roll < 0 {
			return r
		}
	}
	return pool[len(pool)-1]
}

// buildSteps composes the recipe's steps: 1-2 activity steps (zone hunts
// filtered by access, boss steps resolved to the current stage, never
// repeating a stat) and a final rested turn-in.
func buildSteps(req requestorTemplate, ctx PlayerContext) []store.DailyStep {
	var steps []store.DailyStep

	var avail []activityTemplate
	for _, a := range req.Activities {
		if a.Zone != "" && !ctx.AccessibleZones[a.Zone] {
			continue
		}
		if a.Boss && ctx.CurrentBossStage < 0 {
			continue
		}
		avail = append(avail, a)
	}
	picked := map[string]bool{}
	for n := 1 + rand.Intn(2); n > 0; n-- {
		var cands []activityTemplate
		for _, a := range avail {
			key := a.Stat + a.Zone
			if a.Boss {
				key = "boss"
			}
			if !picked[key] {
				cands = append(cands, a)
			}
		}
		if len(cands) == 0 {
			break
		}
		a := cands[rand.Intn(len(cands))]
		key := a.Stat + a.Zone
		if a.Boss {
			key = "boss"
		}
		picked[key] = true
		stat, count := a.Stat, scaleCount(a.Min, a.Max, ctx.Level, activityCountCap)
		if a.Boss {
			stat = "boss_stage_" + strconv.Itoa(ctx.CurrentBossStage)
			count = 1
		}
		steps = append(steps, store.DailyStep{
			Kind: store.DailyStepActivity, Stat: stat, Zone: a.Zone,
			Count: count, TextKey: a.TextKey,
		})
	}

	var tins []turnInTemplate
	for _, ti := range req.TurnIns {
		if ctx.recentlySeenTurnIn(req.NPC, ti.Item) {
			continue
		}
		tins = append(tins, ti)
	}
	if len(tins) == 0 {
		tins = req.TurnIns
	}
	ti := tins[rand.Intn(len(tins))]
	textKey := ti.TextKeys[rand.Intn(len(ti.TextKeys))]
	steps = append(steps, store.DailyStep{
		Kind:    store.DailyStepTurnIn,
		Items:   map[string]int{ti.Item: scaleCount(ti.Min, ti.Max, ctx.Level, turnInCountCap)},
		TextKey: textKey,
	})
	return steps
}

// scaleCount grows a rolled count with the player's level, capped so dailies
// stay bite-size.
func scaleCount(min, max, level, cap int) int {
	rolled := min + rand.Intn(max-min+1)
	c := rolled + level/levelDivisor
	if c > cap {
		c = cap
	}
	if c < min {
		c = min
	}
	return c
}

// sundayRepMult returns the reputation multiplier on Sunday (double rep),
// 1 any other day. The jackpot is not part of the day special: its odds grow
// purely from the completion streak.
func sundayRepMult(now time.Time) int {
	if now.Weekday() == time.Sunday {
		return 2
	}
	return 1
}

// rollReward builds the reward: credits scaled by steps, level and streak, a
// requestor-flavored item, and a streak-boosted jackpot chance. Requestors
// without NPC representation pay extra credits instead of reputation. Sunday
// doubles the reputation reward.
func rollReward(ctx PlayerContext, req requestorTemplate, steps int, now time.Time) store.DailyReward {
	money := 100 + steps*100 + rand.Intn(100) + ctx.Level*2
	bonus := ctx.Streak * streakMoneyPerDay
	if bonus > streakMoneyCap {
		bonus = streakMoneyCap
	}
	if req.NoRep {
		money += noRepBonus
	}
	reward := store.DailyReward{
		Money:     money + bonus,
		ItemID:    req.RewardItems[rand.Intn(len(req.RewardItems))],
		RepPoints: req.RepPoints,
	}
	if !req.NoRep {
		reward.RepNPC = req.NPC
		reward.RepPoints *= sundayRepMult(now)
	}
	if rand.Intn(100) < jackpotChance(ctx) {
		reward.ItemID = "forest_egg"
		reward.Crowns = 10
	}
	return reward
}

// jackpotChance returns the jackpot probability in percent: each completed
// daily quest day adds jackpotPerStreakPct, capped at 100%.
func jackpotChance(ctx PlayerContext) int {
	c := ctx.Streak * jackpotPerStreakPct
	if c > jackpotMaxPct {
		c = jackpotMaxPct
	}
	return c
}

func (ctx PlayerContext) recentlySeenRequestor(npc string) bool {
	for _, e := range ctx.Recent {
		if e.Requestor == npc {
			return true
		}
	}
	return false
}

func (ctx PlayerContext) recentlySeenTurnIn(npc, item string) bool {
	for _, e := range ctx.Recent {
		if e.Requestor == npc && e.TurnIn == item {
			return true
		}
	}
	return false
}

// Streak reports the player's consecutive completed daily quests.
func (s *Service) Streak(userID int64) int {
	return s.store.DailyStreak(userID)
}

// JackpotChance reports the player's current jackpot probability in percent,
// driven by their completion streak.
func (s *Service) JackpotChance(userID int64) int {
	return jackpotChance(PlayerContext{Streak: s.store.DailyStreak(userID)})
}

// Current returns the user's active daily quest recipe and progress.
func (s *Service) Current(userID int64) (*store.DailyRecipe, *model.UserQuestData, error) {
	recipe, err := s.store.GetDailyRecipe(userID)
	if err != nil {
		return nil, nil, err
	}
	data, err := s.store.GetDailyQuestData(userID)
	if err != nil {
		return nil, nil, err
	}
	return recipe, data, nil
}

// Claim delivers the current turn-in step. When the quest completes, the day
// is logged as completed, the completion stat is incremented and the
// reputation reward is granted.
func (s *Service) Claim(userID int64) (*store.DailyRecipe, bool, error) {
	recipe, err := s.store.GetDailyRecipe(userID)
	if err != nil {
		return nil, false, err
	}
	completed, err := s.store.ClaimDailyTurnIn(userID)
	if err != nil {
		return nil, false, err
	}
	if completed {
		if err := s.store.CompleteDailyQuest(userID, recipe.Requestor, recipe.TurnInItem()); err != nil {
			return nil, false, err
		}
		if err := achievement.IncrementStat(s.store.DB, userID, "daily_quests_completed", 1); err != nil {
			return nil, false, err
		}
		if recipe.Reward.RepNPC != "" && recipe.Reward.RepPoints > 0 {
			if _, err := s.npcs.AddReputation(userID, recipe.Reward.RepNPC, recipe.Reward.RepPoints); err != nil {
				return nil, false, err
			}
		}
	}
	return recipe, completed, nil
}
