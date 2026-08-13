package mining

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

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
			sess.riskTurns = eff.RiskTurns
		}
		msg := ""
		if eff.Message != "" {
			msg = i18n.T(eff.Message, "en")
		}
		membed, mcomps := c.mineEmbed("en", 42, msg)
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
