package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
	"guacagamblebot/internal/store"
)

func testStore(t *testing.T) *store.Store {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "sk.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	return store.New(d, &config.Config{StartingBalance: 100})
}

func TestOverclockGrantsHuntAndFishCredits(t *testing.T) {
	st := testStore(t)
	require.NoError(t, st.IncrementGameLimit(1, "fish"))
	require.NoError(t, st.IncrementGameLimit(1, "fish"))
	require.NoError(t, st.IncrementGameLimit(1, "fish"))
	require.NoError(t, st.IncrementGameLimit(1, "hunt"))
	require.NoError(t, st.IncrementGameLimit(1, "hunt"))

	require.NoError(t, st.GrantGameLimitCredit(1, "fish", 3))
	require.NoError(t, st.GrantGameLimitCredit(1, "hunt", 3))
	require.NoError(t, st.GrantGameLimitCredit(1, "mine_descend", 3))

	_, remaining, err := st.CheckGameLimit(1, "fish", 10)
	require.NoError(t, err)
	assert.Equal(t, 10, remaining, "fish must get 3 credits back")

	_, remaining, err = st.CheckGameLimit(1, "hunt", 10)
	require.NoError(t, err)
	assert.Equal(t, 10, remaining, "hunt must get 2 credits back (capped at full limit)")
}

func TestBuildDisplayUsesFrenchDescription(t *testing.T) {
	require.NoError(t, i18n.Load("../../../locales"))
	cfg := &config.Config{StartingBalance: 100}
	st := testStore(t)
	c := &Cog{store: st, cfg: cfg, svc: charsvc.New(st, cfg)}
	_, err := c.store.EnsureCharacter(1)
	require.NoError(t, err)
	require.NoError(t, c.store.DB.Model(&model.UserCharacter{}).
		Where("user_id = ?", 1).Update("level", 40).Error)

	embed, _ := c.buildDisplay("fr", 1)
	assert.Contains(t, embed.Description, "Accorde 3 actions de pêche, de chasse et de mine")
	assert.NotContains(t, embed.Description, "skills.desc_")
}

func TestBuildDisplayRendersAllAvailableSkillButtons(t *testing.T) {
	require.NoError(t, i18n.Load("../../../locales"))
	cfg := &config.Config{StartingBalance: 100}
	st := testStore(t)
	c := &Cog{store: st, cfg: cfg, svc: charsvc.New(st, cfg)}
	_, err := c.store.EnsureCharacter(1)
	require.NoError(t, err)
	require.NoError(t, c.store.DB.Model(&model.UserCharacter{}).
		Where("user_id = ?", 1).Update("level", 100).Error)

	_, comps := c.buildDisplay("en", 1)
	assert.LessOrEqual(t, len(comps), 5, "must respect Discord's 5-row limit")
	var got []string
	for _, comp := range comps {
		row, ok := comp.(discordgo.ActionsRow)
		require.True(t, ok)
		for _, comp := range row.Components {
			b, ok := comp.(discordgo.Button)
			require.True(t, ok)
			_, action, rest := components.Decode(b.CustomID)
			if action == "refresh" {
				continue
			}
			require.NotEmpty(t, rest)
			got = append(got, rest[0])
		}
	}
	assert.Equal(t, 15, len(got), "all 15 skills must have an activate button at level 100")
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
