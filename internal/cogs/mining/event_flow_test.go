package mining

import (
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/i18n"
	invsvc "guacagamblebot/internal/service/inventory"
	miningsvc "guacagamblebot/internal/service/mining"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
	"guacagamblebot/internal/universe/hoakhaven"
)

func TestEventFlowRepro(t *testing.T) {
	if err := i18n.Load("../../../locales"); err != nil {
		t.Fatal(err)
	}

	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "m.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	st := store.New(d, cfg)
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	require.NotNil(t, def)
	inv := invsvc.New(st, cfg)
	npc := npcsvc.New(st, cfg, def, inv)
	svc := miningsvc.New(st, cfg, npc)
	c := &Cog{store: st, cfg: cfg, svc: svc}

	lantern := &miningsvc.NarrativeEvent{
		ID:     "lantern_legendary",
		Stage:  miningsvc.StageShallow,
		Rarity: miningsvc.EventLegendary,
		Options: []miningsvc.NarrativeOption{
			{Label: "mining.ev_lantern_leg_o1", Desc: "mining.ev_lantern_leg_o1d"},
			{Label: "mining.ev_lantern_leg_o2", Desc: "mining.ev_lantern_leg_o2d"},
		},
	}
	sessions[42] = &userSession{depth: 6, bag: nil}
	embed, comps := c.eventEmbed("en", 42, lantern)
	if embed == nil {
		t.Fatal("eventEmbed returned nil embed")
	}
	if len(comps) == 0 {
		t.Fatal("eventEmbed returned no components")
	}
	eb, err := json.Marshal(embed)
	if err != nil {
		t.Fatalf("embed marshal: %v", err)
	}
	cb, err := json.Marshal(comps)
	if err != nil {
		t.Fatalf("components marshal: %v", err)
	}
	t.Logf("embed payload: %s", eb)
	t.Logf("comps payload: %s", cb)

	for _, idx := range []int{0, 1} {
		eff := svc.ApplyEventOption("lantern_legendary", idx, 6, nil)
		if eff == nil {
			t.Fatalf("option %d: nil effect", idx)
		}
		t.Logf("option %d: msg=%s items=%d riskMod=%d riskTurns=%d", idx, eff.Message, len(eff.Items), eff.RiskMod, eff.RiskTurns)
		if eff.Message != "" && i18n.T(eff.Message, "fr") == eff.Message {
			t.Fatalf("option %d: message key %q not found in fr", idx, eff.Message)
		}

		sess := sessions[42]
		for _, it := range eff.Items {
			found := false
			for i, e := range sess.bag {
				if e.Name == it.Name {
					sess.bag[i].Count += it.Count
					found = true
					break
				}
			}
			if !found {
				sess.bag = append(sess.bag, miningsvc.BagEntry{Name: it.Name, Count: it.Count})
			}
		}
		if eff.RiskTurns > 0 {
			sess.riskMod += eff.RiskMod
			sess.riskTurns += eff.RiskTurns
		}
		msg := ""
		if eff.Message != "" {
			msg = i18n.T(eff.Message, "en")
		}
		membed, mcomps := c.mineEmbed("en", 42, msg, nil)
		if membed == nil {
			t.Fatalf("option %d: mineEmbed nil embed", idx)
		}
		mb, err := json.Marshal(membed)
		if err != nil {
			t.Fatalf("option %d: mineEmbed marshal: %v", idx, err)
		}
		mcb, err := json.Marshal(mcomps)
		if err != nil {
			t.Fatalf("option %d: mineEmbed comps marshal: %v", idx, err)
		}
		t.Logf("option %d embed: %s", idx, mb)
		t.Logf("option %d comps: %s", idx, mcb)
	}
}

// TestConcurrentSessionAccess exercises the sessions map through the cog's
// lock-protected read path (mineEmbed) while a writer mutates the session
// exactly like onDescend does. Run with -race to catch any unlocked access to
// the shared session state (the "leave after restart/descend loses loot" bug).
func TestRiskTurnsDecay(t *testing.T) {
	sess := &userSession{riskMod: -15, riskTurns: 3}
	decayTurns(sess)
	require.Equal(t, 2, sess.riskTurns)
	require.Equal(t, -15, sess.riskMod, "modifier stays while turns remain")

	decayTurns(sess)
	decayTurns(sess)
	require.Equal(t, 0, sess.riskTurns)
	require.Equal(t, 0, sess.riskMod, "risk modifier must clear when turns run out")

	decayTurns(&userSession{riskMod: 0, riskTurns: 0})
}

func TestLeaveResultDisplay(t *testing.T) {
	if err := i18n.Load("../../../locales"); err != nil {
		t.Fatal(err)
	}

	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "m.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	st := store.New(d, cfg)
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	require.NotNil(t, def)
	inv := invsvc.New(st, cfg)
	npc := npcsvc.New(st, cfg, def, inv)
	c := &Cog{store: st, cfg: cfg, svc: miningsvc.New(st, cfg, npc)}

	title, color := c.leaveResultDisplay(&miningsvc.LeaveResult{XP: 0, Bag: nil}, "en")
	require.NotContains(t, title, "{xp}", "empty result must not leak the {xp} placeholder")
	require.Contains(t, title, "+0")
	require.Equal(t, components.ColorMuted, color)

	title, color = c.leaveResultDisplay(&miningsvc.LeaveResult{
		XP:  42,
		Bag: []miningsvc.BagEntry{{Name: "coal", Count: 3}},
	}, "en")
	require.Contains(t, title, "Coal")
	require.Contains(t, title, "+42")
	require.Equal(t, components.ColorSuccess, color)
}

func TestConcurrentSessionAccess(t *testing.T) {
	if err := i18n.Load("../../../locales"); err != nil {
		t.Fatal(err)
	}

	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "m.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	st := store.New(d, cfg)
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	require.NotNil(t, def)
	inv := invsvc.New(st, cfg)
	npc := npcsvc.New(st, cfg, def, inv)
	svc := miningsvc.New(st, cfg, npc)
	c := &Cog{store: st, cfg: cfg, svc: svc}

	sessionsMu.Lock()
	sessions[42] = &userSession{depth: 5, bag: []miningsvc.BagEntry{{Name: "coal", Count: 3}}}
	sessionsMu.Unlock()

	const readers = 8
	const iterations = 200
	var wg sync.WaitGroup

	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				embed, comps := c.mineEmbed("en", 42, "", nil)
				if embed == nil || len(comps) == 0 {
					t.Error("mineEmbed returned nil embed/components")
				}
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			sessionsMu.Lock()
			sess := sessions[42]
			sess.bag = append(sess.bag, miningsvc.BagEntry{Name: "pebble", Count: 1})
			sess.depth++
			sessionsMu.Unlock()
		}
	}()

	wg.Wait()

	sessionsMu.Lock()
	sess := sessions[42]
	require.NotNil(t, sess)
	require.Len(t, sess.bag, iterations+1)
	sessionsMu.Unlock()
	delete(sessions, 42)
}

// TestSessionSurvivesRestart simulates a bot restart: the session is persisted,
// the in-memory map is wiped, and loadSession must restore the full expedition
// (depth, tool, effects and loot) from the DB.
func TestSessionSurvivesRestart(t *testing.T) {
	if err := i18n.Load("../../../locales"); err != nil {
		t.Fatal(err)
	}

	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "m.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	st := store.New(d, cfg)
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	require.NotNil(t, def)
	inv := invsvc.New(st, cfg)
	npc := npcsvc.New(st, cfg, def, inv)
	svc := miningsvc.New(st, cfg, npc)
	c := &Cog{store: st, cfg: cfg, svc: svc}

	sessionsMu.Lock()
	sessions[42] = &userSession{
		depth:          9,
		toolID:         "steel_pickaxe",
		ghostVeilTurns: 2,
		riskMod:        -10,
		riskTurns:      3,
		bag:            []miningsvc.BagEntry{{Name: "coal", Count: 3}, {Name: "emerald", Count: 1}},
	}
	sessionsMu.Unlock()
	c.persistSession(42)

	sessionsMu.Lock()
	delete(sessions, 42)
	sessionsMu.Unlock()

	restored := c.loadSession(42)
	require.NotNil(t, restored, "session must be restored from the DB after a restart")
	require.Equal(t, 9, restored.depth)
	require.Equal(t, "steel_pickaxe", restored.toolID)
	require.Equal(t, 2, restored.ghostVeilTurns)
	require.Equal(t, -10, restored.riskMod)
	require.Equal(t, 3, restored.riskTurns)
	require.Equal(t, []miningsvc.BagEntry{{Name: "coal", Count: 3}, {Name: "emerald", Count: 1}}, restored.bag)

	_ = c.svc.DeleteSession(42)
	sessionsMu.Lock()
	delete(sessions, 42)
	sessionsMu.Unlock()
}
