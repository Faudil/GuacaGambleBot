package dailyquest

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	crafting "guacagamblebot/internal/service/crafting"
	invsvc "guacagamblebot/internal/service/inventory"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/service/use"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
	"guacagamblebot/internal/universe"
	"guacagamblebot/internal/universe/hoakhaven"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d := testutil.NewDB(t)
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	hoakhaven.Register()
	s := store.New(d, cfg)
	inv := invsvc.New(s, cfg)
	def := universe.Get("hoakhaven")
	require.NotNil(t, def)
	npc := npcsvc.New(s, cfg, def, inv)
	return New(s, npc), s
}

func TestGenerateProducesValidRecipe(t *testing.T) {
	svc, _ := testService(t)
	for i := 0; i < 200; i++ {
		recipe, err := svc.Generate(1)
		require.NoError(t, err)

		assert.NotEmpty(t, recipe.Requestor)
		assert.NotEmpty(t, recipe.TitleKey)
		assert.NotEmpty(t, recipe.IntroKey)
		assert.NotEmpty(t, recipe.MoodKey, "every quest carries a mood line")
		assert.NotEmpty(t, recipe.ThankKey, "every quest carries a thanks line")
		assert.GreaterOrEqual(t, len(recipe.Steps), 2)
		assert.LessOrEqual(t, len(recipe.Steps), 3)
		assert.Equal(t, store.DailyStepTurnIn, recipe.Steps[len(recipe.Steps)-1].Kind,
			"the last step is always a turn-in")
		for idx, st := range recipe.Steps {
			if idx == len(recipe.Steps)-1 {
				assert.Len(t, st.Items, 1)
			} else {
				assert.Equal(t, store.DailyStepActivity, st.Kind)
				assert.Greater(t, st.Count, 0)
				assert.NotEmpty(t, st.Stat)
			}
		}
		assert.Greater(t, recipe.Reward.Money, 0)
		assert.NotEmpty(t, recipe.Reward.ItemID)
		if recipe.Requestor == "town_board" {
			assert.Empty(t, recipe.Reward.RepNPC, "the Town Board grants no reputation")
			assert.Zero(t, recipe.Reward.RepPoints)
		} else {
			assert.Equal(t, recipe.Requestor, recipe.Reward.RepNPC)
			assert.Greater(t, recipe.Reward.RepPoints, 0)
		}
	}
}

func TestGenerateRespectsZoneAccess(t *testing.T) {
	svc, _ := testService(t)
	// A fresh player may only hunt in the starting zones (forest, cave,
	// desert); recipes must never contain a locked zone step.
	for i := 0; i < 300; i++ {
		recipe, err := svc.Generate(1)
		require.NoError(t, err)
		for _, st := range recipe.Steps {
			if st.Zone != "" {
				assert.Contains(t, []string{"forest", "cave", "desert"}, st.Zone,
					"locked zone %q must not be offered", st.Zone)
			}
		}
	}

	// Unlocking a progressive zone makes it eligible.
	require.NoError(t, svc.store.UnlockZone(1, "ocean"))
	var sawOcean bool
	for i := 0; i < 300; i++ {
		recipe, err := svc.Generate(1)
		require.NoError(t, err)
		for _, st := range recipe.Steps {
			if st.Zone == "ocean" {
				sawOcean = true
			}
		}
	}
	// Irian's pool offers the ocean hunt; with enough rolls it must appear.
	// (Probabilistic — 300 rolls with ~1/6 requestor and ~1/3 zone step odds
	// make a miss virtually impossible.)
	assert.True(t, sawOcean, "unlocked ocean zone should eventually be offered")
}

func TestClaimCompletesAndGrantsReputation(t *testing.T) {
	svc, s := testService(t)
	recipe := store.DailyRecipe{
		Requestor: "thorek", TitleKey: "quests.daily.thorek.title", IntroKey: "quests.daily.thorek.intro",
		Steps: []store.DailyStep{
			{Kind: store.DailyStepActivity, Stat: "items_mined", Count: 1, TextKey: "quests.daily.step.mine"},
			{Kind: store.DailyStepTurnIn, Items: map[string]int{"coal": 1}, TextKey: "quests.daily.thorek.turnin_coal"},
		},
		Reward: store.DailyReward{Money: 100, ItemID: "iron_ore", RepNPC: "thorek", RepPoints: 30},
	}
	data, err := json.Marshal(recipe)
	require.NoError(t, err)
	require.NoError(t, s.StartDailyQuest(1, string(data)))

	// Claiming while still on the activity step is refused.
	_, _, err = svc.Claim(1)
	require.ErrorIs(t, err, store.ErrDailyNotTurnIn)

	// Complete the activity step, deliver the items and claim.
	require.NoError(t, s.RecordActivity(1, "items_mined", 1))
	require.NoError(t, s.AddItemRaw(s.DB, 1, "coal", 1))

	claimed, completed, err := svc.Claim(1)
	require.NoError(t, err)
	require.True(t, completed)
	assert.Equal(t, "thorek", claimed.Reward.RepNPC)

	rep, err := svc.npcs.GetReputation(1, "thorek")
	require.NoError(t, err)
	assert.Equal(t, 30, rep.Reputation, "completion grants requestor reputation")

	// Claiming again after completion is rejected.
	_, _, err = svc.Claim(1)
	require.ErrorIs(t, err, store.ErrDailyNotActive)
}

func TestClaimMissingItems(t *testing.T) {
	svc, s := testService(t)
	recipe := store.DailyRecipe{
		Requestor: "elara", TitleKey: "quests.daily.elara.title", IntroKey: "quests.daily.elara.intro",
		Steps: []store.DailyStep{
			{Kind: store.DailyStepActivity, Stat: "items_farmed", Count: 1, TextKey: "quests.daily.step.farm"},
			{Kind: store.DailyStepTurnIn, Items: map[string]int{"wheat": 2}, TextKey: "quests.daily.elara.turnin_wheat"},
		},
		Reward: store.DailyReward{Money: 100},
	}
	data, err := json.Marshal(recipe)
	require.NoError(t, err)
	require.NoError(t, s.StartDailyQuest(1, string(data)))
	require.NoError(t, s.RecordActivity(1, "items_farmed", 1))

	_, _, err = svc.Claim(1)
	require.Error(t, err)
	var missing *store.DailyMissingItemsError
	require.ErrorAs(t, err, &missing)
	require.Len(t, missing.Items, 1)
	assert.Equal(t, "wheat", missing.Items[0].ItemID)
	assert.Equal(t, 2, missing.Items[0].Needed)
}

// ─── Axis C: selection policies (pure, no DB) ───────────────────

func elaraTemplate(t *testing.T) requestorTemplate {
	for _, r := range requestors {
		if r.NPC == "elara" {
			return r
		}
	}
	t.Fatal("elara template missing")
	return requestorTemplate{}
}

func TestScaleCountBounds(t *testing.T) {
	for i := 0; i < 200; i++ {
		low := scaleCount(3, 5, 1, 12)
		assert.GreaterOrEqual(t, low, 3)
		assert.LessOrEqual(t, low, 5)

		high := scaleCount(3, 5, 50, 12)
		assert.GreaterOrEqual(t, high, 3)
		assert.LessOrEqual(t, high, 12)
	}
	// High level + small range: level bonus pushes toward the cap.
	seen := map[int]bool{}
	for i := 0; i < 200; i++ {
		seen[scaleCount(1, 1, 100, 12)] = true
	}
	assert.Equal(t, map[int]bool{12: true}, seen, "level bonus must push counts up to the cap")
}

func TestJackpotChanceScalesWithStreak(t *testing.T) {
	assert.Equal(t, 0, jackpotChance(PlayerContext{}), "no completed day, no jackpot chance")
	assert.Equal(t, 10, jackpotChance(PlayerContext{Streak: 1}), "+10% per completed day")
	assert.Equal(t, 50, jackpotChance(PlayerContext{Streak: 5}))
	assert.Equal(t, 100, jackpotChance(PlayerContext{Streak: 20}), "capped at 100%")
}

func TestSundayRepMult(t *testing.T) {
	monday := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	sunday := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, 1, sundayRepMult(monday))
	assert.Equal(t, 2, sundayRepMult(sunday))
}

func TestRollRewardSundayRep(t *testing.T) {
	sunday := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	req := requestorTemplate{NPC: "elara", RepPoints: 30, RewardItems: []string{"wheat_seed"}}

	reward := rollReward(PlayerContext{}, req, 2, sunday)
	assert.Equal(t, 60, reward.RepPoints, "Sunday doubles reputation with the requestor")

	// The Town Board never grants rep, even on Sunday.
	board := requestorTemplate{NPC: "town_board", NoRep: true, RepPoints: 0, RewardItems: []string{"iron_ore"}}
	reward = rollReward(PlayerContext{}, board, 2, sunday)
	assert.Empty(t, reward.RepNPC)
	assert.Zero(t, reward.RepPoints)

	// A full-streak player always rolls the jackpot (100%).
	var allJackpot = true
	for i := 0; i < 20; i++ {
		r := rollReward(PlayerContext{Streak: 10}, req, 2, sunday)
		if r.ItemID != "forest_egg" {
			allJackpot = false
		}
	}
	assert.True(t, allJackpot, "100% jackpot odds must always roll")
}

func TestPickRequestorAntiRepeat(t *testing.T) {
	var recent []store.DailyHistoryEntry
	for _, r := range requestors[1:] { // rest every requestor but the first
		recent = append(recent, store.DailyHistoryEntry{DateStr: "2026-01-01", Requestor: r.NPC, TurnIn: "x"})
	}
	ctx := PlayerContext{Recent: recent, Affinity: map[string]int{}}
	for i := 0; i < 200; i++ {
		assert.Equal(t, requestors[0].NPC, pickRequestor(ctx).NPC,
			"rested requestors must not be picked while one remains")
	}

	// An all-rested pool falls back to the full set (never panics, still valid).
	full := PlayerContext{Recent: recent, Affinity: map[string]int{}}
	for _, r := range requestors {
		full.Recent = append(full.Recent, store.DailyHistoryEntry{DateStr: "2026-01-02", Requestor: r.NPC, TurnIn: "x"})
	}
	for i := 0; i < 200; i++ {
		picked := pickRequestor(full).NPC
		assert.NotEmpty(t, picked)
	}
}

func TestPickRequestorAffinityWeighting(t *testing.T) {
	// Give the whisper overwhelming affinity; they must dominate picks.
	ctx := PlayerContext{Affinity: map[string]int{"the_whisper": 100}}
	counts := map[string]int{}
	for i := 0; i < 2000; i++ {
		counts[pickRequestor(ctx).NPC]++
	}
	assert.Greater(t, counts["the_whisper"], 1500,
		"affinity 100 must dominate a pool whose other weights are 1")
}

func TestBuildStepsTurnInAntiRepeat(t *testing.T) {
	req := elaraTemplate(t)
	ctx := PlayerContext{
		AccessibleZones: map[string]bool{"forest": true, "cave": true, "desert": true},
		Recent:          []store.DailyHistoryEntry{{DateStr: "2026-01-01", Requestor: "elara", TurnIn: "wheat"}},
	}
	for i := 0; i < 200; i++ {
		steps := buildSteps(req, ctx)
		last := steps[len(steps)-1]
		for item := range last.Items {
			assert.NotEqual(t, "wheat", item, "recent turn-in must be rested")
		}
	}
}

func TestGenerateLogsDailyQuest(t *testing.T) {
	svc, s := testService(t)
	recipe, err := svc.Generate(1)
	require.NoError(t, err)

	entries, err := s.RecentDailyQuests(1, 5)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, recipe.Requestor, entries[0].Requestor)
	assert.Equal(t, recipe.TurnInItem(), entries[0].TurnIn)
	assert.False(t, entries[0].Completed)
}

func TestClaimCompletesLogAndStat(t *testing.T) {
	svc, s := testService(t)
	recipe, err := svc.Generate(1)
	require.NoError(t, err)
	data, err := json.Marshal(recipe)
	require.NoError(t, err)
	require.NoError(t, s.StartDailyQuest(1, string(data)))

	// Drive the generated steps: complete each activity, stock the turn-in.
	for _, st := range recipe.Steps {
		if st.Kind == store.DailyStepActivity {
			require.NoError(t, s.RecordActivity(1, st.Stat, st.Count))
		} else {
			for item, qty := range st.Items {
				require.NoError(t, s.AddItemRaw(s.DB, 1, item, qty))
			}
		}
	}

	_, completed, err := svc.Claim(1)
	require.NoError(t, err)
	require.True(t, completed)

	entries, err := s.RecentDailyQuests(1, 5)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.True(t, entries[0].Completed, "day must be logged as completed")

	assert.Equal(t, 1, svc.Streak(1), "one completed day starts the streak")

	var stat model.UserStat
	require.NoError(t, s.DB.Where("user_id = 1").First(&stat).Error)
	assert.Equal(t, 1, stat.DailyQuestsCompleted)
}

// ─── Axis B: new step kinds + Axis A gates ─────────────────────

func gamblebotTemplate(t *testing.T) requestorTemplate {
	for _, r := range requestors {
		if r.NPC == "gamblebot" {
			return r
		}
	}
	t.Fatal("gamblebot template missing")
	return requestorTemplate{}
}

func TestBuildStepsBossStepResolvedToCurrentStage(t *testing.T) {
	req := gamblebotTemplate(t)

	// No current boss: the boss template is never picked.
	ctx := PlayerContext{CurrentBossStage: -1, AccessibleZones: map[string]bool{"forest": true}}
	for i := 0; i < 200; i++ {
		for _, st := range buildSteps(req, ctx) {
			assert.NotEqual(t, "boss_stage_", st.Stat[:min(len(st.Stat), len("boss_stage_"))], "no boss step without a current boss")
		}
	}

	// Current boss stage 2: any boss step targets exactly that stage, count 1.
	ctx = PlayerContext{CurrentBossStage: 2, AccessibleZones: map[string]bool{"forest": true}}
	var sawBoss bool
	for i := 0; i < 300; i++ {
		for _, st := range buildSteps(req, ctx) {
			if st.Stat == "boss_stage_2" {
				sawBoss = true
				assert.Equal(t, 1, st.Count, "boss steps are single kills")
			} else {
				assert.NotEqual(t, "boss_stage_2", st.Stat)
			}
		}
	}
	assert.True(t, sawBoss, "boss step must appear eventually for a requestor with one")
}

func TestBuildStepsCraftAndUseSteps(t *testing.T) {
	req := elaraTemplate(t)
	ctx := PlayerContext{AccessibleZones: map[string]bool{"forest": true}}
	var sawCraft bool
	for i := 0; i < 300; i++ {
		for _, st := range buildSteps(req, ctx) {
			if st.Stat == "items_crafted" {
				sawCraft = true
			}
		}
	}
	assert.True(t, sawCraft, "elara's pool must include a craft step")

	whisper := requestorTemplate{}
	for _, r := range requestors {
		if r.NPC == "the_whisper" {
			whisper = r
		}
	}
	var sawUse bool
	for i := 0; i < 300; i++ {
		for _, st := range buildSteps(whisper, ctx) {
			if st.Stat == "items_used" {
				sawUse = true
			}
		}
	}
	assert.True(t, sawUse, "the whisper's pool must include a use step")
}

func TestPickRequestorJournalGate(t *testing.T) {
	ctx := PlayerContext{JournalRank: 0, Affinity: map[string]int{}}
	for i := 0; i < 500; i++ {
		assert.NotEqual(t, "the_chronicler", pickRequestor(ctx).NPC,
			"the chronicler must not ask before journal rank 1")
	}

	ctx.JournalRank = 1
	var sawChronicler bool
	for i := 0; i < 1000; i++ {
		if pickRequestor(ctx).NPC == "the_chronicler" {
			sawChronicler = true
		}
	}
	assert.True(t, sawChronicler, "rank 1 unlocks the chronicler requestor")
}

func TestPickRequestorIncludesTownBoard(t *testing.T) {
	ctx := PlayerContext{JournalRank: 0, Affinity: map[string]int{}}
	var sawBoard bool
	for i := 0; i < 1000; i++ {
		if pickRequestor(ctx).NPC == "town_board" {
			sawBoard = true
		}
	}
	assert.True(t, sawBoard, "the town board is always available")
}

func TestCurrentBossStage(t *testing.T) {
	svc, s := testService(t)
	assert.Equal(t, -1, svc.currentBossStage(1), "no boss league quest")

	require.NoError(t, s.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "boss_league", Status: "ACTIVE",
	}).Error)
	require.NoError(t, s.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "boss_league", StepIndex: 1,
	}).Error)
	assert.Equal(t, 0, svc.currentBossStage(1), "battle step 1 is stage 0")

	require.NoError(t, s.DB.Model(&model.UserQuestData{}).
		Where("user_id = 1 AND quest_id = 'boss_league'").
		Update("step_index", 5).Error)
	assert.Equal(t, 2, svc.currentBossStage(1), "battle step 5 is stage 2")

	// A dialogue step is not a fightable boss.
	require.NoError(t, s.DB.Model(&model.UserQuestData{}).
		Where("user_id = 1 AND quest_id = 'boss_league'").
		Update("step_index", 6).Error)
	assert.Equal(t, -1, svc.currentBossStage(1))

	// A completed league quest is not fightable.
	require.NoError(t, s.DB.Model(&model.UserQuest{}).
		Where("user_id = 1 AND quest_id = 'boss_league'").
		Update("status", "COMPLETED").Error)
	assert.Equal(t, -1, svc.currentBossStage(1))
}

func TestCraftActivityAdvancesDailyQuest(t *testing.T) {
	_, s := testService(t)
	recipe := store.DailyRecipe{
		Steps: []store.DailyStep{
			{Kind: store.DailyStepActivity, Stat: "items_crafted", Count: 1, TextKey: "quests.daily.step.craft"},
			{Kind: store.DailyStepTurnIn, Items: map[string]int{"coal": 1}, TextKey: "x"},
		},
		Reward: store.DailyReward{Money: 100},
	}
	data, err := json.Marshal(recipe)
	require.NoError(t, err)
	require.NoError(t, s.StartDailyQuest(1, string(data)))

	require.NoError(t, s.AddItemRaw(s.DB, 1, "wheat", 3))
	craftsvc := crafting.New(s, &config.Config{})
	_, _, err = craftsvc.Craft(1, "beer", 1) // beer = 3 wheat
	require.NoError(t, err)

	d, err := s.GetDailyQuestData(1)
	require.NoError(t, err)
	assert.Equal(t, 1, d.StepIndex, "a craft must advance an items_crafted step")
}

func TestUseActivityAdvancesDailyQuest(t *testing.T) {
	_, s := testService(t)
	recipe := store.DailyRecipe{
		Steps: []store.DailyStep{
			{Kind: store.DailyStepActivity, Stat: "items_used", Count: 1, TextKey: "quests.daily.step.use"},
			{Kind: store.DailyStepTurnIn, Items: map[string]int{"coal": 1}, TextKey: "x"},
		},
		Reward: store.DailyReward{Money: 100},
	}
	data, err := json.Marshal(recipe)
	require.NoError(t, err)
	require.NoError(t, s.StartDailyQuest(1, string(data)))

	require.NoError(t, s.AddItemRaw(s.DB, 1, "beer", 1))
	_, err = use.New(s, &config.Config{}).Apply(1, "beer")
	require.NoError(t, err)

	d, err := s.GetDailyQuestData(1)
	require.NoError(t, err)
	assert.Equal(t, 1, d.StepIndex, "using an item must advance an items_used step")
}

// ─── Axis D: narrative texture ─────────────────────────────────

func TestAllRequestorsHaveThanks(t *testing.T) {
	for _, r := range requestors {
		assert.NotEmpty(t, r.ThankKeys, "requestor %q needs thank lines", r.NPC)
		for _, k := range r.ThankKeys {
			assert.Contains(t, k, "quests.daily.", "thank keys must be i18n keys")
		}
	}
	assert.Len(t, moodKeys, 10, "mood pool should stay at 10 for now")
}

func TestJackpotChanceServiceMethod(t *testing.T) {
	svc, s := testService(t)
	assert.Equal(t, 0, svc.JackpotChance(1), "no completed days")

	// One completed day -> 10%.
	require.NoError(t, s.DB.Create(&model.UserDailyLog{
		UserID: 1, DateStr: time.Now().Format("2006-01-02"),
		Requestor: "elara", TurnInItem: "wheat", Completed: true,
	}).Error)
	assert.Equal(t, 10, svc.JackpotChance(1))
}
