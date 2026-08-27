package mining

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	invsvc "guacagamblebot/internal/service/inventory"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
	"guacagamblebot/internal/universe"
	"guacagamblebot/internal/universe/hoakhaven"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d := testutil.NewDB(t)
	cfg := &config.Config{StartingBalance: 100}
	s := store.New(d, cfg)
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	require.NotNil(t, def)
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)
	svc := New(s, cfg, npcSvc)
	return svc, s
}

func TestDescend(t *testing.T) {
	svc, _ := testService(t)
	res, err := svc.Descend(1, 1, 2*(1-1), nil, "", 0, 0, true, true)
	require.NoError(t, err)
	if !res.Collapsed {
		assert.NotNil(t, res.Item)
		assert.NotEmpty(t, res.Bag)
	}
}

func TestDescendBlockedWhenInventoryFull(t *testing.T) {
	svc, s := testService(t)
	_, err := s.GetBalance(1)
	require.NoError(t, err)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "coal", store.BaseInventoryLimit))

	_, err = svc.Descend(1, 1, 2*(1-1), nil, "", 0, 0, true, true)
	assert.ErrorIs(t, err, store.ErrInventoryFull)
}

func TestDescendCollapse(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 1, XP: 0})
	bag := []BagEntry{}
	collapsed := false
	for i := 0; i < 50; i++ {
		res, err := svc.Descend(1, 40, 2*(40-1), bag, "", 0, 0, true, true)
		require.NoError(t, err)
		if res.Collapsed {
			collapsed = true
			break
		}
		bag = res.Bag
	}
	assert.True(t, collapsed, "should have collapsed at high depth")
}

func TestDescendCanGoAnyDepth(t *testing.T) {
	svc, _ := testService(t)
	for depth := 1; depth <= 30; depth++ {
		res, err := svc.Descend(1, depth, 2*(depth-1), nil, "", 0, 0, true, true)
		require.NoError(t, err)
		if res.Collapsed {
			break
		}
	}
}

func TestDescendDeeperGivesBetterLoot(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 20, XP: 0})
	shallowVal := 0
	deepVal := 0
	trials := 20
	for i := 0; i < trials; i++ {
		r1, err := svc.Descend(1, 3, 2*(3-1), nil, "", 0, 0, true, true)
		if err != nil || r1.Collapsed {
			continue
		}
		r2, err := svc.Descend(1, 15, 2*(15-1), nil, "", 0, 0, true, true)
		if err != nil || r2.Collapsed {
			continue
		}
		if r1.Item != nil {
			shallowVal += r1.Item.Value
		}
		if r2.Item != nil {
			deepVal += r2.Item.Value
		}
	}
	t.Logf("Shallow total value: %d, Deep total value: %d", shallowVal, deepVal)
	assert.Greater(t, deepVal, shallowVal, "deeper digging should yield more valuable loot")
}

func TestDescendRiskModAppliesToRoll(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 5, XP: 0})

	// Depth 20 with level 5 keeps a high base risk.
	assert.Greater(t, svc.RiskFor(1, 2*(20-1), "", 0, 0), 0)
	// A -90 risk modifier from event options zeroes the collapse chance.
	assert.Equal(t, 0, svc.RiskFor(1, 2*(20-1), "", 0, -90))

	for i := 0; i < 200; i++ {
		res, err := svc.Descend(1, 20, 2*(20-1), nil, "", 0, -90, true, true)
		require.NoError(t, err)
		require.False(t, res.Collapsed, "risk modifier must actually prevent collapse at 0%% risk")
	}
}

func TestRiskForStayIsHalfStepOfDescend(t *testing.T) {
	svc, _ := testService(t)

	base := svc.RiskFor(1, 10, "", 0, 0)
	descendStep := svc.RiskFor(1, 12, "", 0, 0) - base // +2 units: a full descend
	stayStep := svc.RiskFor(1, 11, "", 0, 0) - base    // +1 unit: staying to prospect again

	assert.Equal(t, 5, descendStep, "a full descend should raise risk by 5%%")
	assert.Less(t, stayStep, descendStep, "staying should raise risk by less than a full descend")
	assert.Greater(t, stayStep, 0, "staying should still raise risk somewhat")

	// Two stays land on the same risk level as one descend.
	assert.Equal(t, svc.RiskFor(1, 12, "", 0, 0), svc.RiskFor(1, 10+1+1, "", 0, 0))
}

func TestDescendLevelReducesRisk(t *testing.T) {
	svc, s := testService(t)
	s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 50, XP: 0})
	s.DB.Create(&model.Job{UserID: 2, JobName: "miner", Level: 1, XP: 0})

	var ml1, ml2 int
	svc.store.DB.Model(&model.Job{}).Where("user_id = ?", 1).Select("level").Scan(&ml1)
	svc.store.DB.Model(&model.Job{}).Where("user_id = ?", 2).Select("level").Scan(&ml2)

	collapse1 := 0
	collapse2 := 0
	trials := 30

	for i := 0; i < trials; i++ {
		_ = s.ResetGameLimit(1, "mine_descend")
		_ = s.ResetGameLimit(2, "mine_descend")
		r1, err := svc.Descend(1, 15, 2*(15-1), nil, "", 0, 0, true, true)
		require.NoError(t, err)
		r2, err := svc.Descend(2, 15, 2*(15-1), nil, "", 0, 0, true, true)
		require.NoError(t, err)
		if r1.Collapsed {
			collapse1++
		}
		if r2.Collapsed {
			collapse2++
		}
	}

	t.Logf("Level 50 collapses: %d/%d, Level 1 collapses: %d/%d", collapse1, trials, collapse2, trials)
	assert.Greater(t, collapse2, collapse1,
		"lower level miner should collapse more often at same depth")
}

func TestDescendHiddenChamber(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 1, XP: 0})
	found := false
	for i := 0; i < 500; i++ {
		// The daily descend limit would stop the loop long before the rare
		// event is found; reset it so the RNG-driven search can run to completion.
		_ = s.ResetGameLimit(1, "mine_descend")
		bag := []BagEntry{}
		res, err := svc.Descend(1, 40, 2*(40-1), bag, "", 0, 0, true, true)
		require.NoError(t, err)
		if res.Event != nil && res.Event.Type == "hidden_chamber" {
			found = true
			break
		}
	}
	assert.True(t, found, "should have found a hidden chamber eventually")
}

func TestDescendEventSpawn(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 20, XP: 0})
	found := false
	for i := 0; i < 100; i++ {
		_ = s.ResetGameLimit(1, "mine_descend")
		res, err := svc.Descend(1, 15, 2*(15-1), nil, "", 0, 0, true, true)
		require.NoError(t, err)
		if res.NarrativeEvent != nil {
			found = true
			eventID := res.NarrativeEvent.ID
			assert.NotEmpty(t, eventID)
			assert.GreaterOrEqual(t, len(res.NarrativeEvent.Options), 2)
			break
		}
	}
	assert.True(t, found, "should have spawned a narrative event")
}

func TestDescendOffersCharterWhenNoneAcceptedOrDeclined(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 1, XP: 0})
	found := false
	for i := 0; i < 200; i++ {
		_ = s.ResetGameLimit(1, "mine_descend")
		res, err := svc.Descend(1, charterOfferMinDepth, 2*(charterOfferMinDepth-1), nil, "", 0, 0, false, false)
		require.NoError(t, err)
		if res.NarrativeEvent != nil && res.NarrativeEvent.ID == NPCCharterEventID {
			found = true
			break
		}
	}
	assert.True(t, found, "should have offered a charter via an NPC event")
}

func TestDescendNeverOffersCharterOnceDeclinedOrAccepted(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 1, XP: 0})
	for i := 0; i < 200; i++ {
		_ = s.ResetGameLimit(1, "mine_descend")
		res, err := svc.Descend(1, charterOfferMinDepth, 2*(charterOfferMinDepth-1), nil, "", 0, 0, false, true)
		require.NoError(t, err)
		if res.NarrativeEvent != nil {
			assert.NotEqual(t, NPCCharterEventID, res.NarrativeEvent.ID)
		}
	}
}

func TestApplyEventOption(t *testing.T) {
	svc, _ := testService(t)
	eff := svc.ApplyEventOption("collapse", 0, 5, nil)
	assert.NotNil(t, eff)
	assert.NotEmpty(t, eff.Message)
}

func TestApplyEventOptionShrineMystery(t *testing.T) {
	svc, _ := testService(t)
	types := map[string]int{"a": 0, "b": 0, "c": 0, "d": 0}
	for i := 0; i < 100; i++ {
		eff := svc.ApplyEventOption("shrine", 1, 10, nil)
		msg := eff.Message
		switch msg {
		case "mining.ev_shrine_r2a":
			types["a"]++
		case "mining.ev_shrine_r2b":
			types["b"]++
		case "mining.ev_shrine_r2c":
			types["c"]++
		default:
			types["d"]++
		}
	}
	t.Logf("Shrine mystery distribution: %+v", types)
	assert.Greater(t, types["a"], 0)
	assert.Greater(t, types["b"], 0)
}

func TestEnterMine(t *testing.T) {
	svc, _ := testService(t)
	for i := 0; i < dailyDescendLimit; i++ {
		require.NoError(t, svc.EnterMine(1))
	}
	err := svc.EnterMine(1)
	require.ErrorIs(t, err, ErrMineLimit)
}

func TestRemainingEntries(t *testing.T) {
	svc, _ := testService(t)
	r, err := svc.RemainingEntries(1)
	require.NoError(t, err)
	assert.Equal(t, dailyDescendLimit, r)

	require.NoError(t, svc.EnterMine(1))
	r, err = svc.RemainingEntries(1)
	require.NoError(t, err)
	assert.Equal(t, dailyDescendLimit-1, r)
}

func TestDescendDoesNotConsumeEntry(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 1, XP: 0})
	res, err := svc.Descend(1, 1, 2*(1-1), nil, "", 0, 0, true, true)
	require.NoError(t, err)
	if !res.Collapsed {
		r, rerr := svc.RemainingEntries(1)
		require.NoError(t, rerr)
		assert.Equal(t, dailyDescendLimit, r, "digging must not consume the expedition quota")
	}
}

func TestPersistSessionRoundTrip(t *testing.T) {
	svc, _ := testService(t)
	ps := &PersistedSession{
		Depth:          12,
		ToolID:         "steel_pickaxe",
		GhostVeilTurns: 3,
		RiskMod:        -10,
		RiskTurns:      5,
		Bag:            []BagEntry{{Name: "coal", Count: 3}, {Name: "emerald", Count: 1}},
	}
	require.NoError(t, svc.SaveSession(1, ps))

	got, err := svc.LoadSession(1)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, ps.Depth, got.Depth)
	assert.Equal(t, ps.ToolID, got.ToolID)
	assert.Equal(t, ps.GhostVeilTurns, got.GhostVeilTurns)
	assert.Equal(t, ps.RiskMod, got.RiskMod)
	assert.Equal(t, ps.RiskTurns, got.RiskTurns)
	assert.Equal(t, ps.Bag, got.Bag)
}

func TestLoadSessionStaleAutoGrants(t *testing.T) {
	svc, s := testService(t)
	require.NoError(t, svc.SaveSession(1, &PersistedSession{
		Depth:  5,
		ToolID: "",
		Bag:    []BagEntry{{Name: "coal", Count: 3}, {Name: "iron_ore", Count: 2}},
	}))
	require.NoError(t, s.DB.Model(&model.MiningSession{}).
		Where("user_id = ?", 1).
		Update("updated_at", time.Now().Add(-3*time.Hour)).Error)

	got, err := svc.LoadSession(1)
	require.NoError(t, err)
	assert.Nil(t, got, "stale session must be removed and not resumed")

	var inv model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "coal").First(&inv).Error)
	assert.Equal(t, 3, inv.Quantity)
	var iron model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "iron_ore").First(&iron).Error)
	assert.Equal(t, 2, iron.Quantity)

	stored, err := svc.LoadSession(1)
	require.NoError(t, err)
	assert.Nil(t, stored, "stale row must be gone after auto-grant")
}

func TestLeaveMine(t *testing.T) {
	svc, _ := testService(t)
	bag := []BagEntry{{Name: "pebble", Count: 3}, {Name: "coal", Count: 1}}
	res, err := svc.LeaveMine(1, bag, "")
	require.NoError(t, err)
	assert.Equal(t, bag, res.Bag)
	assert.Greater(t, res.XP, 0)
}

func TestLeaveMineEmpty(t *testing.T) {
	svc, _ := testService(t)
	res, err := svc.LeaveMine(1, nil, "")
	require.NoError(t, err)
	assert.Empty(t, res.Bag)
	assert.Equal(t, 0, res.XP)
}

func TestLeaveMineAddsCharacterXP(t *testing.T) {
	svc, s := testService(t)
	bag := []BagEntry{{Name: "pebble", Count: 3}}
	_, err := svc.LeaveMine(1, bag, "")
	require.NoError(t, err)

	c, err := s.GetCharacter(1)
	require.NoError(t, err)
	assert.Greater(t, c.XP, 0)
}

func TestMineReinforceBuffPreventsCollapse(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 1, XP: 0})
	_ = s.SetActiveBuff(1, "reinforce")

	res, err := svc.Descend(1, 40, 2*(40-1), nil, "", 0, 0, true, true)
	require.NoError(t, err)
	assert.False(t, res.Collapsed, "reinforce should prevent collapse on first descend")

	has, _ := s.HasActiveBuff(1, "reinforce")
	assert.False(t, has, "reinforce should be consumed after one descend")
}

func TestMineScavengerBuffDoublesItems(t *testing.T) {
	svc, s := testService(t)
	_ = s.SetActiveBuff(1, "scavenger")

	bag := []BagEntry{{Name: "pebble", Count: 2}, {Name: "coal", Count: 1}}
	res, err := svc.LeaveMine(1, bag, "")
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(res.Bag), 2)
	for _, e := range res.Bag {
		if e.Name == "pebble" {
			assert.Greater(t, e.Count, 2)
		}
	}

	has, _ := s.HasActiveBuff(1, "scavenger")
	assert.False(t, has, "scavenger should be consumed after leave")
}

func TestGetMinerLevelDefault(t *testing.T) {
	svc, _ := testService(t)
	lvl, err := svc.GetMinerLevel(999)
	require.NoError(t, err)
	assert.Equal(t, 1, lvl)
}

func TestAvailableToolsByLevel(t *testing.T) {
	t1 := AvailableTools(1)
	assert.Len(t, t1, 1)
	assert.Equal(t, "", t1[0].ItemID)

	t5 := AvailableTools(5)
	assert.Len(t, t5, 2)

	t10 := AvailableTools(10)
	assert.Len(t, t10, 3)
}

func TestLockedToolsByLevel(t *testing.T) {
	l1 := LockedTools(1)
	assert.Len(t, l1, 2)
	l5 := LockedTools(5)
	assert.Len(t, l5, 1)
	l10 := LockedTools(10)
	assert.Len(t, l10, 0)
}

func TestLoreAtDepth(t *testing.T) {
	assert.NotNil(t, LoreAtDepth(13))
	assert.NotNil(t, LoreAtDepth(21))
	assert.NotNil(t, LoreAtDepth(33))
	assert.NotNil(t, LoreAtDepth(42))
	assert.Nil(t, LoreAtDepth(1))
}

func TestLootAtDepth(t *testing.T) {
	// Procedural curve: lootAtDepth returns 2-3 nearby candidates, rollOre gives Count
	a := lootAtDepth(1)
	assert.GreaterOrEqual(t, len(a), 2)
	assert.LessOrEqual(t, len(a), 3)
	// rollOre shallow should often be pebble/coal
	foundLow := false
	for i := 0; i < 20; i++ {
		it := rollOre(1)
		if it.Name == "pebble" || it.Name == "coal" {
			foundLow = true
			break
		}
	}
	assert.True(t, foundLow, "shallow roll should sometimes be pebble/coal")
	assert.GreaterOrEqual(t, len(lootAtDepth(15)), 2)
	assert.GreaterOrEqual(t, len(lootAtDepth(20)), 2)
	b := lootAtDepth(30)
	assert.GreaterOrEqual(t, len(b), 2)
	assert.LessOrEqual(t, len(b), 3)
	// rollOre deep should tend to high-value ores
	it := rollOre(30)
	assert.GreaterOrEqual(t, it.Value, 1000, "deep roll should be high value")
	assert.GreaterOrEqual(t, it.Count, 1)
}

func TestRollOreCountVariance(t *testing.T) {
	seenMulti := false
	for i := 0; i < 50; i++ {
		it := rollOre(1)
		if it.Count > 1 {
			seenMulti = true
			break
		}
	}
	assert.True(t, seenMulti, "shallow ore should sometimes drop 2-3")
	// high-value ore should always be 1
	for i := 0; i < 20; i++ {
		it := rollOre(30)
		if it.Value >= 300 {
			assert.Equal(t, 1, it.Count, "gem+ ore should be single")
			break
		}
	}
}

func TestOreValueCurveMonotonic(t *testing.T) {
	prev := oreValueCurve(1)
	for d := 2; d <= 30; d++ {
		cur := oreValueCurve(d)
		assert.GreaterOrEqual(t, cur, prev, "curve must be non-decreasing at depth %d", d)
		prev = cur
	}
	assert.Greater(t, oreValueCurve(15), oreValueCurve(5), "deep should exceed shallow")
}

func TestRollOreGating(t *testing.T) {
	for i := 0; i < 200; i++ {
		it := rollOre(19)
		assert.NotEqual(t, "rough_diamond", it.Name, "diamond must not appear before depth 20")
		assert.NotContains(t, []string{"kethari_crystal", "primordial_geode", "resonance_core"}, it.Name, "ultra-rare must not appear before depth 30")
	}
	for i := 0; i < 200; i++ {
		it := rollOre(29)
		assert.NotContains(t, []string{"kethari_crystal", "primordial_geode", "resonance_core"}, it.Name, "ultra-rare must not appear before depth 30")
	}
	foundDiamond := false
	foundUltra := false
	for i := 0; i < 200; i++ {
		if rollOre(20).Name == "rough_diamond" {
			foundDiamond = true
		}
		if rollOre(30).Value >= 1000 {
			foundUltra = true
		}
		if foundDiamond && foundUltra {
			break
		}
	}
	assert.True(t, foundDiamond, "diamond should appear at depth 20+")
	assert.True(t, foundUltra, "ultra-rare should appear at depth 30+")
}

func TestPickNarrativeEvent(t *testing.T) {
	ev1 := pickNarrativeEvent(1)
	assert.Nil(t, ev1)

	ev3 := pickNarrativeEvent(3)
	assert.NotNil(t, ev3)
	assert.NotEmpty(t, ev3.Options)

	ev17 := pickNarrativeEvent(17)
	assert.NotNil(t, ev17)
}

func TestAddItemRawInitializesToolDurability(t *testing.T) {
	svc, s := testService(t)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "steel_pickaxe", 1))
	var inv model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "steel_pickaxe").First(&inv).Error)
	assert.Equal(t, 25, inv.Durability)
	assert.Equal(t, 1, inv.Quantity)

	require.NoError(t, s.AddItemRaw(s.DB, 1, "diamond_drill", 2))
	var drill model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "diamond_drill").First(&drill).Error)
	assert.Equal(t, 50, drill.Durability)
	assert.Equal(t, 2, drill.Quantity)

	require.NoError(t, s.AddItemRaw(s.DB, 1, "coal", 3))
	var coal model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "coal").First(&coal).Error)
	assert.Equal(t, 0, coal.Durability, "non-tool items must not carry durability")

	assert.Equal(t, 25, svc.ToolDurability(1, "steel_pickaxe"))
	assert.Equal(t, 50, svc.ToolDurability(1, "diamond_drill"))
	assert.Equal(t, 0, svc.ToolDurability(1, ""))
	assert.Equal(t, 0, svc.ToolDurability(1, "coal"))
}

func TestConsumeToolDurabilityBreaksAfterMax(t *testing.T) {
	svc, s := testService(t)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "steel_pickaxe", 1))

	for i := 0; i < 24; i++ {
		broke, err := svc.ConsumeToolDurability(1, "steel_pickaxe")
		require.NoError(t, err)
		assert.False(t, broke, "tool must not break before 25 digs (dig %d)", i+1)
	}
	assert.Equal(t, 1, svc.ToolDurability(1, "steel_pickaxe"))

	broke, err := svc.ConsumeToolDurability(1, "steel_pickaxe")
	require.NoError(t, err)
	assert.True(t, broke, "tool must break on the 25th dig")

	var count int64
	s.DB.Model(&model.Inventory{}).Where("user_id = ? AND item_id = ?", 1, "steel_pickaxe").Count(&count)
	assert.Equal(t, int64(0), count, "last tool in stack must be removed on break")
	assert.Equal(t, 0, svc.ToolDurability(1, "steel_pickaxe"))
}

func TestConsumeToolDurabilitySwapsStack(t *testing.T) {
	svc, s := testService(t)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "steel_pickaxe", 2))

	var inv model.Inventory
	for i := 0; i < 25; i++ {
		broke, err := svc.ConsumeToolDurability(1, "steel_pickaxe")
		require.NoError(t, err)
		if i == 24 {
			assert.True(t, broke, "active tool breaks after 25 digs")
		} else {
			assert.False(t, broke)
		}
	}
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "steel_pickaxe").First(&inv).Error)
	assert.Equal(t, 1, inv.Quantity, "one unit consumed, one remains")
	assert.Equal(t, 25, inv.Durability, "fresh tool in the stack starts at full durability")
}

func TestConsumeToolDurabilityBaseAndMissing(t *testing.T) {
	svc, s := testService(t)
	broke, err := svc.ConsumeToolDurability(1, "")
	require.NoError(t, err)
	assert.False(t, broke, "base tool never breaks")

	broke, err = svc.ConsumeToolDurability(1, "steel_pickaxe")
	require.NoError(t, err)
	assert.True(t, broke, "missing tool should fall back to base tool")

	require.NoError(t, s.AddItemRaw(s.DB, 1, "diamond_drill", 1))
	broke, err = svc.ConsumeToolDurability(1, "diamond_drill")
	require.NoError(t, err)
	assert.False(t, broke)
}

func TestDescendConsumesToolDurability(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "miner", Level: 5, XP: 0})
	require.NoError(t, s.AddItemRaw(s.DB, 1, "steel_pickaxe", 1))
	_ = s.ResetGameLimit(1, "mine_descend")

	bag := []BagEntry{}
	for i := 0; i < 5; i++ {
		res, err := svc.Descend(1, 1, 2*(1-1), bag, "steel_pickaxe", 0, 0, true, true)
		require.NoError(t, err)
		require.False(t, res.Collapsed)
		require.False(t, res.ToolBroke)
		bag = res.Bag
	}
	assert.Equal(t, 20, svc.ToolDurability(1, "steel_pickaxe"), "5 digs must consume 5 durability")
}

func TestLeaveMineDoesNotConsumeTool(t *testing.T) {
	svc, s := testService(t)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "steel_pickaxe", 1))

	bag := []BagEntry{{Name: "coal", Count: 1}}
	_, err := svc.LeaveMine(1, bag, "steel_pickaxe")
	require.NoError(t, err)

	var inv model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "steel_pickaxe").First(&inv).Error)
	assert.Equal(t, 1, inv.Quantity, "leaving the mine must not consume the tool")
}
