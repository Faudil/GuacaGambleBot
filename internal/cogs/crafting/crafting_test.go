package crafting

import (
	"os"
	"path/filepath"
	"strings"
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

func fieldValues(embed *discordgo.MessageEmbed) []string {
	vals := make([]string, 0, len(embed.Fields))
	for _, f := range embed.Fields {
		vals = append(vals, f.Value)
	}
	return vals
}

func anyFieldContains(vals []string, substr string) bool {
	for _, v := range vals {
		if strings.Contains(v, substr) {
			return true
		}
	}
	return false
}

func TestCraftMenuOnlyShowsUnlockedRecipes(t *testing.T) {
	c := testCog(t)
	// Level 1, no research completed: only the recipes without a research gate
	// and with a reachable level must be craftable.
	embed, comps := c.buildCraftMenu(1, "en", "all", 1)
	assert.Equal(t, "🛠️ Crafting Workshop (Level 1)", embed.Title)

	_, recipeSel := menuSelects(t, comps)
	opts := recipeOptions(recipeSel)
	require.Contains(t, opts, "beer")
	require.Contains(t, opts, "coffee")
	require.Contains(t, opts, "scratch_ticket")
	require.NotContains(t, opts, "craft_stick", "research-gated recipes must not be pickable")
	require.NotContains(t, opts, "rusty_magnet", "research-gated recipes must not be pickable")

	// ... but the embed still lists them, with what they need.
	vals := fieldValues(embed)
	assert.True(t, anyFieldContains(vals, "🔬 Tool Crafting"),
		"locked recipes must be visible with their research requirement")

	// Crafting an unlocked level-1 item succeeds.
	require.NoError(t, c.store.AddItemRaw(c.store.DB, 1, "wheat", 3))

	b := &interaction.Bot{Session: &stubSession{}}
	c.onCraftPick(b, craftPickInteraction("1", "beer"))
	embed = lastEmbed(t, b)
	assert.Contains(t, embed.Description, "1x Beer")

	var inv model.Inventory
	require.NoError(t, c.store.DB.Where("user_id = ? AND item_id = ?", 1, "beer").First(&inv).Error)
	assert.Equal(t, 1, inv.Quantity)
}

func TestCraftMenuLockedRecipesShowRequirements(t *testing.T) {
	c := testCog(t)
	// Page 1 mixes 3 unlocked recipes with the first locked ones; requirements
	// of late-alphabetical recipes land on later pages, so scan a few.
	vals := []string{}
	for page := 1; page <= 3; page++ {
		embed, _ := c.buildCraftMenu(1, "en", "all", page)
		vals = append(vals, fieldValues(embed)...)
	}

	assert.True(t, anyFieldContains(vals, "🔬 Tool Crafting"), "research-locked recipe must show the research name")
	assert.True(t, anyFieldContains(vals, "🔬 Common Equipment"), "equipment recipes must show their research name")

	// Level-locked recipes (no research gate) show the crafter level.
	embedCons, _ := c.buildCraftMenu(1, "en", "consumables", 1)
	assert.True(t, anyFieldContains(fieldValues(embedCons), "Lvl. 2"), "level-locked recipe must show the crafter level")
}

func TestCraftMenuResearchUnlocksRecipeAtLevel1(t *testing.T) {
	c := testCog(t)
	// Completing tool_crafting must unlock bow/rusty_magnet/hook even at
	// crafter level 1: the research is the only gate.
	require.NoError(t, c.store.DB.Create(&model.UserResearch{
		UserID: 1, ResearchID: "tool_crafting", Completed: true,
	}).Error)

	_, comps := c.buildCraftMenu(1, "en", "all", 1)
	_, recipeSel := menuSelects(t, comps)
	opts := recipeOptions(recipeSel)
	require.Contains(t, opts, "bow")
	require.Contains(t, opts, "rusty_magnet")
	require.Contains(t, opts, "hook")

	require.NoError(t, c.store.AddItemRaw(c.store.DB, 1, "oat", 2))
	require.NoError(t, c.store.AddItemRaw(c.store.DB, 1, "pebble", 2))
	b := &interaction.Bot{Session: &stubSession{}}
	c.onCraftPick(b, craftPickInteraction("1", "bow"))
	embed := lastEmbed(t, b)
	assert.Contains(t, embed.Description, "1x Bow")

	var inv model.Inventory
	require.NoError(t, c.store.DB.Where("user_id = ? AND item_id = ?", 1, "bow").First(&inv).Error)
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
	assert.Len(t, crtsvc.Recipes, 71)
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
	assert.Len(t, recipeSel3.Options, 21, "last page holds the remaining recipes")

	// A stale nav press landing beyond the last page is clamped.
	_, compsClamped := c.buildCraftMenu(1, "en", "all", 99)
	_, recipeSelClamped := menuSelects(t, compsClamped)
	assert.Len(t, recipeSelClamped.Options, 21)
}

func TestCraftMenuCategoryFilter(t *testing.T) {
	c := testCog(t)
	setCrafterLevel(t, c, 1, 15)
	completeAllResearch(t, c, 1)

	embed, comps := c.buildCraftMenu(1, "en", "pets", 1)
	assert.Len(t, embed.Fields, 25, "pets page 1 shows the first 25 recipes")
	embed2, _ := c.buildCraftMenu(1, "en", "pets", 2)
	assert.Len(t, embed2.Fields, 3, "pets page 2 shows the remaining recipes")
	_, recipeSel := menuSelects(t, comps)
	opts := recipeOptions(recipeSel)
	require.Contains(t, opts, "warrior_stew")
	require.Contains(t, opts, "berserker_elixir")
	require.Contains(t, opts, "lucky_roast")
	require.Contains(t, opts, "dragon_chili")
	require.Contains(t, opts, "colossus_draught")
	_, comps2 := c.buildCraftMenu(1, "en", "pets", 2)
	_, recipeSel2 := menuSelects(t, comps2)
	opts2 := recipeOptions(recipeSel2)
	require.Contains(t, opts2, "vitality_elixir", "later pet recipes appear on page 2")
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
	// Level 1 user with no unlocked equipment: the embed lists the locked
	// equipment recipes and the select menu falls back to a placeholder.
	embed, comps := c.buildCraftMenu(1, "en", "equipment", 1)
	require.NotEmpty(t, embed.Fields, "locked equipment recipes must still be listed")
	_, recipeSel := menuSelects(t, comps)
	assert.Len(t, recipeSel.Options, 1)
	assert.Equal(t, "_none", recipeSel.Options[0].Value)
}

func TestRecipesViewShowsLockedRecipes(t *testing.T) {
	c := testCog(t)
	embed, _ := c.buildRecipesView(1, 1, "en", 1)
	assert.Contains(t, embed.Description, "🔓 Unlocked Recipes")
	assert.Contains(t, embed.Description, "Beer", "unlocked recipes stay listed")

	// The unlocked section now fills page 1; the locked section and its
	// requirements land on later pages, so scan them all.
	desc := embed.Description
	for page := 2; page <= 6; page++ {
		e, _ := c.buildRecipesView(1, 1, "en", page)
		desc += e.Description
	}
	assert.Contains(t, desc, "🔒 Locked Recipes")
	assert.Contains(t, desc, "🔬 Tool Crafting", "research-gated recipes must show their research")
	assert.Contains(t, desc, "Lvl. 2", "level-gated recipes must show the crafter level")
}

func TestCraftSuccessShowsCraftAgainButton(t *testing.T) {
	c := testCog(t)
	require.NoError(t, c.store.AddItemRaw(c.store.DB, 1, "wheat", 3))

	b := &interaction.Bot{Session: &stubSession{}}
	c.onCraftPick(b, craftPickInteraction("1", "beer"))

	s, ok := b.Session.(*stubSession)
	require.True(t, ok)
	require.NotNil(t, s.last)
	require.NotNil(t, s.last.Data.Components)
	row, ok := s.last.Data.Components[0].(discordgo.ActionsRow)
	require.True(t, ok)
	btn, ok := row.Components[0].(discordgo.Button)
	require.True(t, ok)
	assert.Equal(t, "crafting::craft_back::1", btn.CustomID, "success must offer a craft-another button")
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

func TestCraftFilterSelectIsOwnerEncoded(t *testing.T) {
	c := testCog(t)
	_, comps := c.buildCraftMenu(1, "en", "all", 1)
	var filterID string
	for _, comp := range comps {
		row, ok := comp.(discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, sub := range row.Components {
			sel, ok := sub.(discordgo.SelectMenu)
			if !ok {
				continue
			}
			if strings.HasPrefix(sel.CustomID, "crafting::craft_filter") {
				filterID = sel.CustomID
			}
		}
	}
	assert.Equal(t, "crafting::craft_filter::1", filterID,
		"category select must carry the owner id so the router gates it")
}
