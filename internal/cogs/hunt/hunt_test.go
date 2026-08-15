package hunt

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/model"
	huntsvc "guacagamblebot/internal/service/hunt"
	invsvc "guacagamblebot/internal/service/inventory"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
	"guacagamblebot/internal/universe/hoakhaven"
)

func TestMain(m *testing.M) {
	_ = i18n.Load("../../../locales")
	os.Exit(m.Run())
}

func testCog(t *testing.T) *Cog {
	t.Helper()
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "hunt_cog.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, HuntMaxPerDay: 10, HuntCooldownSeconds: 10}
	s := store.New(d, cfg)
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	require.NotNil(t, def)
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)
	return &Cog{store: s, cfg: cfg, svc: huntsvc.New(s, cfg, npcSvc)}
}

func menuZoneOptions(t *testing.T) []discordgo.SelectMenuOption {
	t.Helper()
	_, comps := testCog(t).menu("fr", 1)
	require.Len(t, comps, 1)
	row, ok := comps[0].(discordgo.ActionsRow)
	require.True(t, ok)
	require.Len(t, row.Components, 1)
	sel, ok := row.Components[0].(discordgo.SelectMenu)
	require.True(t, ok)
	require.NotEmpty(t, sel.Options)
	return sel.Options
}

func TestMenuZoneOptionsValid(t *testing.T) {
	opts := menuZoneOptions(t)
	for _, o := range opts {
		_, ok := huntsvc.Zones[o.Value]
		assert.True(t, ok, "select menu option value %q must be a valid hunt zone key", o.Value)
	}
}

func TestMenuZonesSortedByLevel(t *testing.T) {
	opts := menuZoneOptions(t)
	for i := 1; i < len(opts); i++ {
		prev, cur := huntsvc.Zones[opts[i-1].Value], huntsvc.Zones[opts[i].Value]
		assert.LessOrEqual(t, prev.LevelMin, cur.LevelMin,
			"menu zones must be sorted by ascending level (got %s then %s)", opts[i-1].Value, opts[i].Value)
	}
}

func TestMenuShowsLockedZoneProgress(t *testing.T) {
	cog := testCog(t)
	_, comps := cog.menu("fr", 1)
	row := comps[0].(discordgo.ActionsRow)
	sel := row.Components[0].(discordgo.SelectMenu)

	byZone := make(map[string]discordgo.SelectMenuOption, len(sel.Options))
	for _, o := range sel.Options {
		byZone[o.Value] = o
	}

	// mountain requires 3 desert wins; user has none yet.
	opt, ok := byZone["mountain"]
	require.True(t, ok)
	assert.Contains(t, opt.Description, "0/3", "locked zone must show current/required progress")

	// Record 2 desert wins: progress must update to 2/3 and stay locked.
	_, err := cog.svc.RecordHuntWin(1, "desert")
	require.NoError(t, err)
	_, err = cog.svc.RecordHuntWin(1, "desert")
	require.NoError(t, err)
	_, comps = cog.menu("fr", 1)
	row = comps[0].(discordgo.ActionsRow)
	sel = row.Components[0].(discordgo.SelectMenu)
	byZone = make(map[string]discordgo.SelectMenuOption, len(sel.Options))
	for _, o := range sel.Options {
		byZone[o.Value] = o
	}
	opt = byZone["mountain"]
	assert.Contains(t, opt.Description, "2/3", "locked zone progress must update")

	// After the 3rd win the zone must be presented as unlocked (no lock marker).
	_, err = cog.svc.RecordHuntWin(1, "desert")
	require.NoError(t, err)
	_, comps = cog.menu("fr", 1)
	row = comps[0].(discordgo.ActionsRow)
	sel = row.Components[0].(discordgo.SelectMenu)
	byZone = make(map[string]discordgo.SelectMenuOption, len(sel.Options))
	for _, o := range sel.Options {
		byZone[o.Value] = o
	}
	opt = byZone["mountain"]
	assert.NotContains(t, opt.Description, "🔒", "unlocked zone must not show a lock marker")
}

func TestMenuDashboardDescFilled(t *testing.T) {
	cog := testCog(t)
	require.NoError(t, cog.store.DB.Create(&model.UserPet{
		UserID:   1,
		PetType:  "Chien",
		Nickname: "Rex",
		Level:    7,
		MaxHP:    100,
		HP:       100,
		IsActive: true,
	}).Error)

	embed, _ := cog.menu("fr", 1)
	assert.NotEmpty(t, embed.Description)
	assert.NotContains(t, embed.Description, "{name}", "dashboard description must fill the {name} placeholder")
	assert.NotContains(t, embed.Description, "{lvl}", "dashboard description must fill the {lvl} placeholder")
	assert.Contains(t, embed.Description, "Rex")
	assert.Contains(t, embed.Description, "7")
}
