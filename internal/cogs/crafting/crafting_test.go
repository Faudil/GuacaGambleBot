package crafting

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
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	crtsvc "guacagamblebot/internal/service/crafting"
	"guacagamblebot/internal/store"
)

func TestMain(m *testing.M) {
	_ = i18n.Load("../../../locales")
	os.Exit(m.Run())
}

func testCog(t *testing.T) *Cog {
	t.Helper()
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "crafting_cog.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	s := store.New(d, cfg)
	require.NoError(t, s.SaveServerSetting(&model.ServerSetting{ServerID: 100, Language: "en"}))
	return &Cog{store: s, cfg: cfg, svc: crtsvc.New(s, cfg)}
}

// setCrafterLevel upserts the user's crafter job level.
func setCrafterLevel(t *testing.T, c *Cog, userID int64, level int) {
	t.Helper()
	require.NoError(t, c.store.DB.Transaction(func(tx *gorm.DB) error {
		return tx.Model(&model.Job{}).
			Where("user_id = ? AND job_name = ?", userID, "crafter").
			FirstOrCreate(&model.Job{UserID: userID, JobName: "crafter", Level: 1}).
			UpdateColumn("level", level).Error
	}))
}

// completeAllResearch marks every research used by the recipe list as done.
func completeAllResearch(t *testing.T, c *Cog, userID int64) {
	t.Helper()
	seen := map[string]bool{}
	for _, r := range crtsvc.Recipes {
		for _, rid := range []string{r.RequiredResearch, r.RequiredResearch2} {
			if rid == "" || seen[rid] {
				continue
			}
			seen[rid] = true
			require.NoError(t, c.store.DB.Create(&model.UserResearch{
				UserID: userID, ResearchID: rid, Completed: true,
			}).Error)
		}
	}
}

func menuSelects(t *testing.T, comps []discordgo.MessageComponent) (filter, recipe discordgo.SelectMenu) {
	t.Helper()
	require.Len(t, comps, 3)
	row0, ok := comps[0].(discordgo.ActionsRow)
	require.True(t, ok)
	filter, ok = row0.Components[0].(discordgo.SelectMenu)
	require.True(t, ok)

	row1, ok := comps[1].(discordgo.ActionsRow)
	require.True(t, ok)
	recipe, ok = row1.Components[0].(discordgo.SelectMenu)
	require.True(t, ok)
	return filter, recipe
}

func recipeOptions(sel discordgo.SelectMenu) map[string]discordgo.SelectMenuOption {
	byValue := make(map[string]discordgo.SelectMenuOption, len(sel.Options))
	for _, opt := range sel.Options {
		byValue[opt.Value] = opt
	}
	return byValue
}

func TestCraftMenuOnlyShowsUnlockedRecipes(t *testing.T) {
	c := testCog(t)
	// Level 1, no research completed: only the level-1 recipes without a
	// research gate must be listed.
	embed, comps := c.buildCraftMenu(1, "en", "all", 1)
	assert.Equal(t, "🛠️ Crafting Workshop (Level 1)", embed.Title)

	_, recipeSel := menuSelects(t, comps)
	opts := recipeOptions(recipeSel)
	require.Contains(t, opts, "beer")
	require.Contains(t, opts, "coffee")
	require.Contains(t, opts, "scratch_ticket")
	require.NotContains(t, opts, "craft_stick", "research-gated recipes must be hidden")
	require.NotContains(t, opts, "rusty_magnet", "level-gated recipes must be hidden")

	// Crafting a level-1 item without ingredients must fail.
	require.NoError(t, c.store.AddItemRaw(c.store.DB, 1, "wheat", 3))
	require.NoError(t, c.store.AddItemRaw(c.store.DB, 1, "coffee_bean", 3))

	b := &interaction.Bot{Session: &stubSession{}}
	inter := craftPickInteraction("1", "beer")
	c.onCraftPick(b, inter)
	embed = lastEmbed(t, b)
	assert.Contains(t, embed.Description, "1x Beer")

	var inv model.Inventory
	require.NoError(t, c.store.DB.Where("user_id = ? AND item_id = ?", 1, "beer").First(&inv).Error)
	assert.Equal(t, 1, inv.Quantity)
}

func TestCraftMenuMissingIngredients(t *testing.T) {
	c := testCog(t)
	b := &interaction.Bot{Session: &stubSession{}}
	c.onCraftPick(b, craftPickInteraction("1", "beer"))
	embed := lastEmbed(t, b)
	assert.Contains(t, embed.Description, "missing ingredients")
}

func TestCraftMenuPaginates(t *testing.T) {
	c := testCog(t)
	setCrafterLevel(t, c, 1, 15)
	completeAllResearch(t, c, 1)

	embed, comps := c.buildCraftMenu(1, "en", "all", 1)
	assert.Len(t, crtsvc.Recipes, 51)
	assert.Len(t, embed.Fields, 25, "page 1 shows the first 25 recipes")

	_, recipeSel := menuSelects(t, comps)
	assert.Len(t, recipeSel.Options, 25)

	navRow, ok := comps[2].(discordgo.ActionsRow)
	require.True(t, ok)
	require.Len(t, navRow.Components, 3)
	nextBtn, ok := navRow.Components[2].(discordgo.Button)
	require.True(t, ok)
	assert.False(t, nextBtn.Disabled, "next must be enabled when more pages exist")

	_, comps3 := c.buildCraftMenu(1, "en", "all", 3)
	_, recipeSel3 := menuSelects(t, comps3)
	assert.Len(t, recipeSel3.Options, 1, "last page holds the remaining recipe")

	// A stale nav press landing beyond the last page is clamped.
	_, compsClamped := c.buildCraftMenu(1, "en", "all", 99)
	_, recipeSelClamped := menuSelects(t, compsClamped)
	assert.Len(t, recipeSelClamped.Options, 1)
}

func TestCraftMenuCategoryFilter(t *testing.T) {
	c := testCog(t)
	setCrafterLevel(t, c, 1, 15)
	completeAllResearch(t, c, 1)

	embed, comps := c.buildCraftMenu(1, "en", "pets", 1)
	assert.Len(t, embed.Fields, 8, "all pet food and potion recipes")
	_, recipeSel := menuSelects(t, comps)
	opts := recipeOptions(recipeSel)
	require.Contains(t, opts, "warrior_stew")
	require.Contains(t, opts, "berserker_elixir")
	require.NotContains(t, opts, "beer", "other categories must be excluded")
	require.NotContains(t, opts, "craft_stick", "equipment must be excluded")

	_, compsSets := c.buildCraftMenu(1, "en", "sets", 1)
	_, setSel := menuSelects(t, compsSets)
	setOpts := recipeOptions(setSel)
	require.Contains(t, setOpts, "craft_dragon_slayer_sword")
	require.Contains(t, setOpts, "craft_arcane_weaver_staff")
	require.NotContains(t, setOpts, "craft_stick", "plain equipment is not a set")
}

func TestCraftMenuEmptyCategory(t *testing.T) {
	c := testCog(t)
	// Level 1 user with no unlocked recipes: the select menu must still render
	// with a disabled-style placeholder option.
	embed, comps := c.buildCraftMenu(1, "en", "equipment", 1)
	assert.Contains(t, embed.Description, "No recipes unlocked")
	_, recipeSel := menuSelects(t, comps)
	assert.Len(t, recipeSel.Options, 1)
	assert.Equal(t, "_none", recipeSel.Options[0].Value)
}

func TestCraftMenuSelectsCarryOwnerID(t *testing.T) {
	c := testCog(t)
	_, comps := c.buildCraftMenu(1, "en", "all", 1)
	_, recipeSel := menuSelects(t, comps)
	owner, ok := components.OwnerID(recipeSel.CustomID)
	require.True(t, ok)
	assert.Equal(t, int64(1), owner)
}

// stubSession records interaction responses without a live Discord connection.
type stubSession struct {
	interaction.Session
	last *discordgo.InteractionResponse
}

func (s *stubSession) InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
	s.last = r
	return nil
}

func craftPickInteraction(userID string, values ...string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		Type:    discordgo.InteractionMessageComponent,
		GuildID: "100",
		Token:   "tok",
		Member:  &discordgo.Member{User: &discordgo.User{ID: userID}},
		Data:    discordgo.MessageComponentInteractionData{CustomID: "crafting::craft_pick::" + userID, Values: values},
	}}
}

func lastEmbed(t *testing.T, b *interaction.Bot) *discordgo.MessageEmbed {
	t.Helper()
	s, ok := b.Session.(*stubSession)
	require.True(t, ok)
	require.NotNil(t, s.last)
	require.Len(t, s.last.Data.Embeds, 1)
	return s.last.Data.Embeds[0]
}
