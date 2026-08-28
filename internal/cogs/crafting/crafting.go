package crafting

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	crtsvc "guacagamblebot/internal/service/crafting"
	researchsvc "guacagamblebot/internal/service/research"
	"guacagamblebot/internal/store"
)

// recipesPerPage caps how many recipes are rendered per page in the /recipes
// embed description so it always stays under Discord's 4096-character limit.
const recipesPerPage = 12

// maxMenuOptions caps the recipe options shown in the craft menu's select menu
// per page (Discord's hard limit is 25).
const maxMenuOptions = 25

// recipeCategories is the ordered list of category filter options for the
// craft menu. "all" must stay first.
var recipeCategories = []string{"all", "consumables", "tools", "structures", "pets", "equipment", "sets"}

var categoryEmojis = map[string]string{
	"all":         "📦",
	"consumables": "🍺",
	"tools":       "🔧",
	"structures":  "🌿",
	"pets":        "🐾",
	"equipment":   "⚔️",
	"sets":        "🐉",
}

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *crtsvc.Service
}

const maxCraftAmount = 10

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: crtsvc.New(s, cfg)}
	r.SlashWithOptions("craft", "cmd.craft.desc", []*discordgo.ApplicationCommandOption{
		{Type: discordgo.ApplicationCommandOptionString, Name: "item", Description: "Recipe to craft", Required: false, Autocomplete: true},
		{Type: discordgo.ApplicationCommandOptionInteger, Name: "quantity", Description: "How many (1-10)", Required: false, MinValue: float64Ptr(1), MaxValue: 10},
	}, c.onSlashCraft)
	r.Slash("recipes", "cmd.recipes.desc", c.onSlashRecipes)
	r.Prefix("craft", c.onCraftPrefix)
	r.Prefix("fabriquer", c.onCraftPrefix)
	r.Prefix("recipes", c.onRecipesPrefix)
	r.Prefix("recettes", c.onRecipesPrefix)
	r.Prefix("crafting", c.onRecipesPrefix)
	r.Prefix("rec", c.onRecipesPrefix)
	r.Component("crafting", "recipes_nav", c.onRecipesNav)
	r.Component("crafting", "craft_pick", c.onCraftPick)
	r.Component("crafting", "craft_filter", c.onCraftFilter)
	r.Component("crafting", "craft_nav", c.onCraftNav)
	r.Component("crafting", "craft_back", c.onCraftBack)
	r.Component("crafting", "craft_step", c.onCraftStep)
	r.Component("crafting", "craft_confirm", c.onCraftConfirm)
}

func (c *Cog) onSlashCraft(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	// Autocomplete handling
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		c.handleCraftAutocomplete(b, i, lang, userID)
		return
	}

	// Parse slash options: item + quantity
	var itemOpt *string
	var qtyOpt *int
	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "item":
			if v, ok := opt.Value.(string); ok {
				itemOpt = &v
			}
		case "quantity":
			if v, ok := opt.Value.(float64); ok {
				qi := int(v)
				qtyOpt = &qi
			}
		}
	}
	if itemOpt != nil && strings.TrimSpace(*itemOpt) != "" {
		amount := 1
		if qtyOpt != nil {
			amount = *qtyOpt
		}
		if amount < 1 || amount > maxCraftAmount {
			_ = b.Session.InteractionRespond(i.Interaction,
				components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
					components.Embed("❌", i18n.T("crafting.invalid_qty_range", lang), components.ColorDanger), nil))
			return
		}
		query := strings.ToLower(strings.TrimSpace(*itemOpt))
		recipeKey := c.resolveRecipeKey(query, lang)
		recipe, ok := crtsvc.Recipes[recipeKey]
		if !ok {
			_ = b.Session.InteractionRespond(i.Interaction,
				components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
					components.Embed("❌", i18n.T("crafting.no_recipe", lang, map[string]any{"item": query}), components.ColorDanger), nil))
			return
		}
		// Enforce max craftable hint if requested exceeds possible
		if max := c.maxCraftable(userID, recipe); amount > max {
			_ = b.Session.InteractionRespond(i.Interaction,
				components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
					components.Embed("❌", i18n.T("crafting.no_ingredients", lang, map[string]any{"missing": "..."})+i18n.T("crafting.max_hint", lang, map[string]any{"max": max}), components.ColorDanger), nil))
			return
		}
		charLeveled, charNewLevel, err := c.svc.Craft(userID, recipeKey, amount)
		if err != nil {
			var msg string
			switch err {
			case crtsvc.ErrNoLevel:
				msg = i18n.T("crafting.no_level", lang, map[string]any{"level": recipe.LevelRequired})
			case crtsvc.ErrNoIngredients:
				max := c.maxCraftable(userID, recipe)
				msg = i18n.T("crafting.no_ingredients", lang, map[string]any{"missing": "..."}) + i18n.T("crafting.max_hint", lang, map[string]any{"max": max})
			case crtsvc.ErrNoRecipe:
				msg = i18n.T("crafting.no_recipe", lang, map[string]any{"item": query})
			case crtsvc.ErrResearchRequired:
				rn := c.researchName(recipe.RequiredResearch)
				if !c.isResearchCompleted(userID, recipe.RequiredResearch2) {
					rn = c.researchName(recipe.RequiredResearch2)
				}
				msg = i18n.T("crafting.no_research", lang, map[string]any{"research": rn})
			default:
				if err == store.ErrInventoryFull {
					msg = i18n.T("inventory.full", lang)
				} else {
					msg = err.Error()
				}
			}
			_ = b.Session.InteractionRespond(i.Interaction,
				components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
					components.Embed("❌", msg, components.ColorDanger), nil))
			return
		}
		resDisplay := items.LocalizedName(recipe.Result, lang)
		msg := i18n.T("crafting.success_msg", lang, map[string]any{"amount": amount, "item": resDisplay, "xp": recipe.XP * amount})
		if leveledUp, newLevel := c.svc.LevelUpCheck(userID); leveledUp {
			msg += i18n.T("crafting.level_up", lang, map[string]any{"level": newLevel})
		}
		if charLeveled {
			msg += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": charNewLevel})
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
				components.Embed("✅", msg, components.ColorSuccess), nil))
		return
	}

	embed, comps := c.buildCraftMenu(userID, lang, "all", 1)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) handleCraftAutocomplete(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, userID int64) {
	focused := ""
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "item" && opt.Focused {
			if v, ok := opt.Value.(string); ok {
				focused = strings.ToLower(v)
			}
		}
	}
	level := c.svc.GetCrafterLevel(userID)
	choices := []*discordgo.ApplicationCommandOptionChoice{}
	for key, rec := range crtsvc.Recipes {
		if !c.unlocked(userID, level, rec) {
			continue
		}
		name := items.LocalizedName(rec.Result, lang)
		if focused != "" && !strings.Contains(strings.ToLower(name), focused) && !strings.Contains(strings.ToLower(key), focused) && !strings.Contains(strings.ToLower(rec.Result), focused) {
			continue
		}
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: name, Value: key})
		if len(choices) >= 25 {
			break
		}
	}
	// sort by name for stable autocomplete
	sort.Slice(choices, func(a, b int) bool { return choices[a].Name < choices[b].Name })
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}

func (c *Cog) onCraftPrefix(b *interaction.Bot, sess *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	content := strings.TrimSpace(strings.TrimPrefix(m.Content, "!craft "))
	content = strings.TrimSpace(strings.TrimPrefix(content, "!fabriquer "))

	if content == "" {
		embed, comps := c.buildCraftMenu(userID, lang, "all", 1)
		_, _ = sess.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: comps,
		})
		return
	}

	parts := strings.Fields(content)
	amount := 1
	itemQuery := ""

	if len(parts) > 1 && parts[0] == "1" {
		itemQuery = strings.Join(parts[1:], " ")
	} else if len(parts) > 1 && isNumeric(parts[0]) {
		amount, _ = strconv.Atoi(parts[0])
		itemQuery = strings.Join(parts[1:], " ")
	} else {
		itemQuery = content
	}

	if amount <= 0 {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.invalid_qty", lang))
		return
	}
	if amount > maxCraftAmount {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.invalid_qty_range", lang))
		return
	}

	itemQuery = strings.ToLower(strings.TrimSpace(itemQuery))

	recipeKey := c.resolveRecipeKey(itemQuery, lang)
	recipe, ok := crtsvc.Recipes[recipeKey]
	if !ok {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_recipe", lang, map[string]any{"item": itemQuery}))
		return
	}

	// max hint for prefix
	if max := c.maxCraftable(userID, recipe); amount > max {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_ingredients", lang, map[string]any{"missing": "..."})+i18n.T("crafting.max_hint", lang, map[string]any{"max": max}))
		return
	}

	charLeveled, charNewLevel, err := c.svc.Craft(userID, recipeKey, amount)
	if err != nil {
		switch err {
		case crtsvc.ErrNoLevel:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_level", lang, map[string]any{"level": recipe.LevelRequired}))
		case crtsvc.ErrNoIngredients:
			max := c.maxCraftable(userID, recipe)
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_ingredients", lang, map[string]any{"missing": "..."})+i18n.T("crafting.max_hint", lang, map[string]any{"max": max}))
		case crtsvc.ErrNoRecipe:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_recipe", lang, map[string]any{"item": itemQuery}))
		case crtsvc.ErrResearchRequired:
			rName := c.researchName(recipe.RequiredResearch)
			if !c.isResearchCompleted(userID, recipe.RequiredResearch2) {
				rName = c.researchName(recipe.RequiredResearch2)
			}
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_research", lang, map[string]any{"research": rName}))
		default:
			if err == store.ErrInventoryFull {
				_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("inventory.full", lang))
			} else {
				_, _ = sess.ChannelMessageSend(m.ChannelID, "❌ "+err.Error())
			}
		}
		return
	}

	resDisplay := items.LocalizedName(recipe.Result, lang)
	msg := i18n.T("crafting.success_msg", lang, map[string]any{"amount": amount, "item": resDisplay, "xp": recipe.XP * amount})

	if leveledUp, newLevel := c.svc.LevelUpCheck(userID); leveledUp {
		msg += i18n.T("crafting.level_up", lang, map[string]any{"level": newLevel})
	}
	if charLeveled {
		msg += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": charNewLevel})
	}

	_, _ = sess.ChannelMessageSend(m.ChannelID, msg)
}

// recipeDisplayInfo renders one line of the /recipes embed description.
func recipeDisplayInfo(recipe crtsvc.Recipe, lang string) string {
	resName := items.LocalizedName(recipe.Result, lang)

	if recipe.IsEquipment {
		it := items.Get(recipe.Result)
		if it != nil {
			extra := ""
			if it.SetID != "" && it.SetName != "" {
				extra = fmt.Sprintf(" [%s Set]", it.SetName)
			}
			return fmt.Sprintf("%s (%s%s) | %s", resName, string(it.Rarity), extra, recipeIngredients(recipe, lang))
		}
		return fmt.Sprintf("%s [Equipment] | %s", resName, recipeIngredients(recipe, lang))
	}
	return fmt.Sprintf("%s | %s", resName, recipeIngredients(recipe, lang))
}

func (c *Cog) onSlashRecipes(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	level := c.svc.GetCrafterLevel(userID)
	embed, comps := c.buildRecipesView(userID, level, lang, 1)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onRecipesNav(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 2 {
		return
	}
	action := rest[0]
	page, _ := strconv.Atoi(rest[1])
	switch action {
	case "prev":
		page--
	case "next":
		page++
	}
	level := c.svc.GetCrafterLevel(userID)
	embed, comps := c.buildRecipesView(userID, level, lang, page)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onRecipesPrefix(b *interaction.Bot, sess *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	level := c.svc.GetCrafterLevel(userID)
	embed, comps := c.buildRecipesView(userID, level, lang, 1)
	_, _ = sess.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

// unlocked reports whether the user can craft the recipe right now: crafter
// level and both research gates satisfied.
func (c *Cog) unlocked(userID int64, level int, recipe crtsvc.Recipe) bool {
	return recipe.LevelRequired <= level &&
		c.isResearchCompleted(userID, recipe.RequiredResearch) &&
		c.isResearchCompleted(userID, recipe.RequiredResearch2)
}

// unlockedRecipeKeys returns the recipe map keys the user can currently craft.
func (c *Cog) unlockedRecipeKeys(userID int64, level int) []string {
	var out []string
	for key, recipe := range crtsvc.Recipes {
		if c.unlocked(userID, level, recipe) {
			out = append(out, key)
		}
	}
	return out
}

// recipeLock returns what is blocking a locked recipe: the research id that
// still needs to be completed, or the crafter level required (both empty only
// when the recipe is actually unlocked).
func (c *Cog) recipeLock(userID int64, recipe crtsvc.Recipe) (researchID string, level int) {
	if !c.isResearchCompleted(userID, recipe.RequiredResearch) {
		return recipe.RequiredResearch, 0
	}
	if !c.isResearchCompleted(userID, recipe.RequiredResearch2) {
		return recipe.RequiredResearch2, 0
	}
	return "", recipe.LevelRequired
}

// buildRecipesView renders one page of all recipes — unlocked ones under the
// 🔓 section, locked ones under 🔒 with their missing requirement — with
// prev/page/next navigation. The page is clamped, so a stale button press
// still lands on a valid page.
func (c *Cog) buildRecipesView(userID int64, level int, lang string, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	keys := c.craftRecipes(userID, level, lang, "all")

	// Build the full pageable line list: unlocked section first, then locked.
	lines := []string{i18n.T("crafting.unlocked_title", lang)}
	unlockedCount := 0
	for _, key := range keys {
		recipe := crtsvc.Recipes[key]
		if !c.unlocked(userID, level, recipe) {
			continue
		}
		unlockedCount++
		lines = append(lines, "✅ "+recipeDisplayInfo(recipe, lang))
	}
	if unlockedCount == 0 {
		lines = append(lines, i18n.T("crafting.no_unlocked", lang))
	}
	lines = append(lines, i18n.T("crafting.locked_title", lang))
	lockedCount := 0
	for _, key := range keys {
		recipe := crtsvc.Recipes[key]
		if c.unlocked(userID, level, recipe) {
			continue
		}
		lockedCount++
		researchID, needLevel := c.recipeLock(userID, recipe)
		line := ""
		if researchID != "" {
			line = i18n.T("crafting.lock_research_line", lang, map[string]any{
				"item":        items.LocalizedName(recipe.Result, lang),
				"research":    c.researchName(researchID),
				"ingredients": recipeIngredients(recipe, lang),
			})
		} else {
			line = i18n.T("crafting.lock_line", lang, map[string]any{
				"item":        items.LocalizedName(recipe.Result, lang),
				"level":       needLevel,
				"ingredients": recipeIngredients(recipe, lang),
			})
		}
		lines = append(lines, "🔒 "+line)
	}
	if lockedCount == 0 {
		lines = append(lines, i18n.T("crafting.no_locked", lang))
	}

	totalPages := max(1, int(math.Ceil(float64(len(lines))/float64(recipesPerPage))))
	page = max(1, min(page, totalPages))

	desc := i18n.T("crafting.desc_intro", lang)
	start := (page - 1) * recipesPerPage
	end := min(start+recipesPerPage, len(lines))
	desc += strings.Join(lines[start:end], "\n")

	embed := components.Embed(
		i18n.T("crafting.title", lang, map[string]any{"level": level}),
		desc, components.ColorWarning,
	)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("crafting.nav_page", lang, map[string]any{"page": page, "total": totalPages}),
	}

	navPage := i18n.T("crafting.nav_page", lang, map[string]any{"page": page, "total": totalPages})
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			discordgo.Button{
				Label:    i18n.T("crafting.nav_prev", lang),
				CustomID: components.EncodeOwner(userID, "crafting", "recipes_nav", "prev", strconv.Itoa(page)),
				Style:    discordgo.SecondaryButton,
				Disabled: page <= 1,
			},
			discordgo.Button{
				Label:    navPage,
				CustomID: "_disabled",
				Style:    discordgo.SecondaryButton,
				Disabled: true,
			},
			discordgo.Button{
				Label:    i18n.T("crafting.nav_next", lang),
				CustomID: components.EncodeOwner(userID, "crafting", "recipes_nav", "next", strconv.Itoa(page)),
				Style:    discordgo.SecondaryButton,
				Disabled: page >= totalPages,
			},
		),
	}
	return embed, comps
}

// recipeCategory classifies a recipe into one of the craft menu's filter
// categories.
func recipeCategory(recipe crtsvc.Recipe) string {
	if strings.HasPrefix(recipe.RequiredResearch, "set_") {
		return "sets"
	}
	if recipe.IsEquipment {
		return "equipment"
	}
	switch recipe.Result {
	case "warrior_stew", "stonebread", "zephyr_berries", "hunters_soup",
		"lucky_roast", "thunder_steak", "heart_stew",
		"dragon_chili", "iron_loaf", "storm_porridge", "falcon_pie",
		"clover_salad", "volcano_ribs", "giant_noodles",
		"berserker_elixir", "adamant_tonic", "gale_draught", "oracles_insight",
		"fatalist_elixir", "ruin_tonic", "vitality_elixir",
		"skull_elixir", "bastion_tonic", "tempest_draught", "seer_elixir",
		"gamblers_tonic", "annihilator_elixir", "colossus_draught":
		return "pets"
	case "garden_plot", "tropical_greenhouse", "enchanted_orchard":
		return "structures"
	case "bow", "rusty_magnet", "hook", "magnet", "electric_magnet":
		return "tools"
	default:
		return "consumables"
	}
}

// craftRecipes returns every recipe key — unlocked and locked — filtered by
// category and ordered so the menu stays stable across interactions (map
// iteration is random): craftable recipes first, then locked ones, each group
// by level then name.
func (c *Cog) craftRecipes(userID int64, level int, lang, category string) []string {
	keys := make([]string, 0, len(crtsvc.Recipes))
	for key := range crtsvc.Recipes {
		keys = append(keys, key)
	}
	if category != "" && category != "all" {
		filtered := make([]string, 0, len(keys))
		for _, key := range keys {
			if recipeCategory(crtsvc.Recipes[key]) == category {
				filtered = append(filtered, key)
			}
		}
		keys = filtered
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := crtsvc.Recipes[keys[i]], crtsvc.Recipes[keys[j]]
		au, bu := c.unlocked(userID, level, a), c.unlocked(userID, level, b)
		if au != bu {
			return au
		}
		if a.LevelRequired != b.LevelRequired {
			return a.LevelRequired < b.LevelRequired
		}
		return items.LocalizedName(a.Result, lang) < items.LocalizedName(b.Result, lang)
	})
	return keys
}

// buildCraftMenu renders the /craft embed and its components: a category
// filter, a recipe select menu and prev/page/next navigation. The embed lists
// every recipe of the active category — locked ones show what they need; the
// select menu only contains the recipes the user can craft right now.
func (c *Cog) buildCraftMenu(userID int64, lang, category string, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	level := c.svc.GetCrafterLevel(userID)
	recipes := c.craftRecipes(userID, level, lang, category)
	totalPages := max(1, int(math.Ceil(float64(len(recipes))/float64(maxMenuOptions))))
	page = max(1, min(page, totalPages))

	catLabel := i18n.T("crafting.cat_all", lang)
	if category != "" && category != "all" {
		catLabel = i18n.T("crafting.cat_"+category, lang)
	}

	desc := i18n.T("crafting.menu_desc", lang, map[string]any{"filter": catLabel})
	embed := components.Embed(i18n.T("crafting.menu_title", lang, map[string]any{"level": level}), desc, components.ColorWarning)

	start := (page - 1) * maxMenuOptions
	end := min(start+maxMenuOptions, len(recipes))
	for _, key := range recipes[start:end] {
		embed.Fields = append(embed.Fields, c.recipeMenuField(userID, level, crtsvc.Recipes[key], lang))
	}
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("crafting.nav_page", lang, map[string]any{"page": page, "total": totalPages}),
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(c.categoryFilter(userID, category, lang)),
		components.ActionRow(c.recipeSelect(userID, level, recipes, page, lang)),
		components.ActionRow(c.craftNavButtons(userID, category, page, totalPages, lang)...),
	}
	return embed, comps
}

func (c *Cog) recipeMenuField(userID int64, level int, recipe crtsvc.Recipe, lang string) *discordgo.MessageEmbedField {
	name := fmt.Sprintf("%s %s", recipeEmoji(recipe), items.LocalizedName(recipe.Result, lang))
	if c.unlocked(userID, level, recipe) {
		return components.Field(name,
			i18n.T("crafting.menu_recipe_line", lang, map[string]any{
				"ingredients": recipeIngredients(recipe, lang),
				"xp":          recipe.XP,
			}), true)
	}
	researchID, needLevel := c.recipeLock(userID, recipe)
	if researchID != "" {
		return components.Field(name,
			i18n.T("crafting.menu_locked_research", lang, map[string]any{
				"ingredients": recipeIngredients(recipe, lang),
				"research":    c.researchName(researchID),
			}), true)
	}
	return components.Field(name,
		i18n.T("crafting.menu_locked_level", lang, map[string]any{
			"ingredients": recipeIngredients(recipe, lang),
			"level":       needLevel,
		}), true)
}

func recipeIngredients(recipe crtsvc.Recipe, lang string) string {
	ingStrs := make([]string, 0, len(recipe.Ingredients))
	for ing, qty := range recipe.Ingredients {
		ingStrs = append(ingStrs, fmt.Sprintf("%dx %s", qty, items.LocalizedName(ing, lang)))
	}
	return strings.Join(ingStrs, ", ")
}

func recipeEmoji(recipe crtsvc.Recipe) string {
	if it := items.Get(recipe.Result); it != nil {
		return it.Emoji
	}
	return "🔨"
}

func (c *Cog) categoryFilter(userID int64, category, lang string) discordgo.SelectMenu {
	opts := make([]discordgo.SelectMenuOption, 0, len(recipeCategories))
	for _, cat := range recipeCategories {
		opts = append(opts, discordgo.SelectMenuOption{
			Label:   i18n.T("crafting.cat_"+cat, lang),
			Value:   cat,
			Emoji:   &discordgo.ComponentEmoji{Name: categoryEmojis[cat]},
			Default: cat == category,
		})
	}
	return discordgo.SelectMenu{
		CustomID:    components.EncodeOwner(userID, "crafting", "craft_filter"),
		Placeholder: i18n.T("crafting.filter_placeholder", lang),
		Options:     opts,
	}
}

func (c *Cog) recipeSelect(userID int64, level int, recipes []string, page int, lang string) discordgo.SelectMenu {
	options := make([]discordgo.SelectMenuOption, 0, maxMenuOptions)
	start := (page - 1) * maxMenuOptions
	end := min(start+maxMenuOptions, len(recipes))
	for _, key := range recipes[start:end] {
		recipe := crtsvc.Recipes[key]
		if !c.unlocked(userID, level, recipe) || c.maxCraftable(userID, recipe) <= 0 {
			continue
		}
		desc := recipeIngredients(recipe, lang)
		if len(desc) > 100 {
			desc = desc[:97] + "..."
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       items.LocalizedName(recipe.Result, lang),
			Value:       key,
			Emoji:       &discordgo.ComponentEmoji{Name: recipeEmoji(recipe)},
			Description: desc,
		})
	}
	if len(options) == 0 {
		options = append(options, discordgo.SelectMenuOption{
			Label:       i18n.T("crafting.empty_label", lang),
			Value:       "_none",
			Description: i18n.T("crafting.empty_desc", lang),
			Default:     true,
		})
	}
	return discordgo.SelectMenu{
		CustomID:    components.EncodeOwner(userID, "crafting", "craft_pick"),
		Placeholder: i18n.T("crafting.select_placeholder", lang),
		Options:     options,
	}
}

func (c *Cog) craftNavButtons(userID int64, category string, page, totalPages int, lang string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.Button{
			Label:    i18n.T("crafting.nav_prev", lang),
			CustomID: components.EncodeOwner(userID, "crafting", "craft_nav", "prev", strconv.Itoa(page), category),
			Style:    discordgo.SecondaryButton,
			Disabled: page <= 1,
		},
		discordgo.Button{
			Label:    i18n.T("crafting.nav_page", lang, map[string]any{"page": page, "total": totalPages}),
			CustomID: "_disabled",
			Style:    discordgo.SecondaryButton,
			Disabled: true,
		},
		discordgo.Button{
			Label:    i18n.T("crafting.nav_next", lang),
			CustomID: components.EncodeOwner(userID, "crafting", "craft_nav", "next", strconv.Itoa(page), category),
			Style:    discordgo.SecondaryButton,
			Disabled: page >= totalPages,
		},
	}
}

// onCraftFilter switches the craft menu's category, resetting to page 1.
func (c *Cog) onCraftFilter(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		interaction.RespondError(b, i, lang, "crafting.empty")
		return
	}
	category := values[0]
	embed, comps := c.buildCraftMenu(userID, lang, category, 1)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// onCraftNav pages through the craft menu with clamped bounds.
func (c *Cog) onCraftNav(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 3 {
		return
	}
	action := rest[0]
	page, _ := strconv.Atoi(rest[1])
	category := rest[2]

	level := c.svc.GetCrafterLevel(userID)
	totalPages := max(1, int(math.Ceil(float64(len(c.craftRecipes(userID, level, lang, category)))/float64(maxMenuOptions))))
	switch action {
	case "prev":
		page--
	case "next":
		page++
	}
	page = max(1, min(page, totalPages))

	embed, comps := c.buildCraftMenu(userID, lang, category, page)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// onCraftPick now opens a quantity stepper instead of crafting 1 immediately.
func (c *Cog) onCraftPick(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	values := i.MessageComponentData().Values
	if len(values) == 0 || values[0] == "_none" {
		interaction.RespondError(b, i, lang, "crafting.empty")
		return
	}
	recipeKey := values[0]
	if _, ok := crtsvc.Recipes[recipeKey]; !ok {
		interaction.RespondError(b, i, lang, "crafting.no_recipe", map[string]any{"item": recipeKey})
		return
	}
	embed, comps := c.buildCraftQuantityView(userID, recipeKey, lang, 1)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onCraftStep(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 2 {
		return
	}
	recipeKey := rest[0]
	qty, _ := strconv.Atoi(rest[1])
	if _, ok := crtsvc.Recipes[recipeKey]; !ok {
		interaction.RespondError(b, i, lang, "crafting.no_recipe", map[string]any{"item": recipeKey})
		return
	}
	qty = max(1, min(qty, maxCraftAmount))
	// also clamp to max craftable so stepper never exceeds feasible
	if max := c.maxCraftable(userID, crtsvc.Recipes[recipeKey]); max > 0 && qty > max {
		qty = max
	}
	embed, comps := c.buildCraftQuantityView(userID, recipeKey, lang, qty)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onCraftConfirm(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 2 {
		return
	}
	recipeKey := rest[0]
	qty, _ := strconv.Atoi(rest[1])
	if qty < 1 || qty > maxCraftAmount {
		interaction.RespondError(b, i, lang, "crafting.invalid_qty_range")
		return
	}
	recipe, ok := crtsvc.Recipes[recipeKey]
	if !ok {
		interaction.RespondError(b, i, lang, "crafting.no_recipe", map[string]any{"item": recipeKey})
		return
	}
	charLeveled, charNewLevel, err := c.svc.Craft(userID, recipeKey, qty)
	if err != nil {
		var msg string
		switch err {
		case crtsvc.ErrNoLevel:
			msg = i18n.T("crafting.no_level", lang, map[string]any{"level": recipe.LevelRequired})
		case crtsvc.ErrNoIngredients:
			max := c.maxCraftable(userID, recipe)
			msg = i18n.T("crafting.no_ingredients", lang, map[string]any{"missing": "..."}) + i18n.T("crafting.max_hint", lang, map[string]any{"max": max})
		case crtsvc.ErrNoRecipe:
			msg = i18n.T("crafting.no_recipe", lang, map[string]any{"item": recipeKey})
		case crtsvc.ErrResearchRequired:
			rn := c.researchName(recipe.RequiredResearch)
			if !c.isResearchCompleted(userID, recipe.RequiredResearch2) {
				rn = c.researchName(recipe.RequiredResearch2)
			}
			msg = i18n.T("crafting.no_research", lang, map[string]any{"research": rn})
		default:
			if err == store.ErrInventoryFull {
				msg = i18n.T("inventory.full", lang)
			} else {
				msg = err.Error()
			}
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
				components.Embed("❌", msg, components.ColorDanger), nil))
		return
	}
	resDisplay := items.LocalizedName(recipe.Result, lang)
	msg := i18n.T("crafting.success_msg", lang, map[string]any{"amount": qty, "item": resDisplay, "xp": recipe.XP * qty})
	if leveledUp, newLevel := c.svc.LevelUpCheck(userID); leveledUp {
		msg += i18n.T("crafting.level_up", lang, map[string]any{"level": newLevel})
	}
	if charLeveled {
		msg += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": charNewLevel})
	}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("crafting.craft_again_btn", lang),
				components.EncodeOwner(userID, "crafting", "craft_back"), discordgo.PrimaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
			components.Embed("✅", msg, components.ColorSuccess), comps))
}

// onCraftBack reopens the craft menu so the user can craft another item.
func (c *Cog) onCraftBack(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.buildCraftMenu(userID, lang, "all", 1)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) resolveRecipeKey(query, lang string) string {
	if _, ok := crtsvc.Recipes[query]; ok {
		return query
	}
	q := strings.ToLower(query)
	for key, recipe := range crtsvc.Recipes {
		if strings.ToLower(key) == q {
			return key
		}
		if it := items.Get(recipe.Result); it != nil {
			if strings.ToLower(it.Name) == q || strings.ToLower(it.ID) == q {
				return key
			}
		}
		if strings.ToLower(items.LocalizedName(key, lang)) == q {
			return key
		}
	}
	return query
}

func (c *Cog) isResearchCompleted(userID int64, researchID string) bool {
	if researchID == "" {
		return true
	}
	var r model.UserResearch
	err := c.store.DB.Where("user_id = ? AND research_id = ? AND completed = ?", userID, researchID, true).First(&r).Error
	return err == nil
}

func (c *Cog) researchName(researchID string) string {
	if rd, ok := researchsvc.ResearchDefs[researchID]; ok {
		return rd.Name
	}
	return researchID
}

func float64Ptr(v float64) *float64 { return &v }

// maxCraftable returns how many copies the user can craft limited by ingredients and inventory space, capped at maxCraftAmount.
func (c *Cog) maxCraftable(userID int64, recipe crtsvc.Recipe) int {
	// ingredient multiplier: workbench discount + efficiency buff (without consuming)
	mult := 1.0
	var buff model.ActiveBuff
	if err := c.store.DB.Where("user_id = ? AND skill_id = ?", userID, "efficiency").First(&buff).Error; err == nil {
		mult *= 0.5
	}
	var house model.UserHousing
	if err := c.store.DB.Where("user_id = ? AND is_active = ?", userID, true).First(&house).Error; err == nil {
		var cnt int64
		c.store.DB.Model(&model.UserFurniture{}).Where("user_id = ? AND house_type = ? AND furniture_id = ?", userID, house.HouseType, "workbench").Count(&cnt)
		if cnt > 0 {
			mult *= 0.9
		}
	}

	// max by ingredients
	maxByIng := maxCraftAmount
	first := true
	for ing, qty := range recipe.Ingredients {
		var inv model.Inventory
		if err := c.store.DB.Where("user_id = ? AND item_id = ?", userID, ing).First(&inv).Error; err != nil {
			return 0
		}
		reqPer := max(1, int(float64(qty)*mult))
		if reqPer == 0 {
			reqPer = 1
		}
		possible := inv.Quantity / reqPer
		if first || possible < maxByIng {
			maxByIng = possible
			first = false
		}
	}
	if maxByIng > maxCraftAmount {
		maxByIng = maxCraftAmount
	}
	if maxByIng < 0 {
		maxByIng = 0
	}
	// limit by free slots (effectiveAmount includes perfect_forge doubling)
	effMult := 1
	var pf model.ActiveBuff
	if err := c.store.DB.Where("user_id = ? AND skill_id = ?", userID, "perfect_forge").First(&pf).Error; err == nil {
		effMult = 2
	}
	free, err := c.store.FreeSlots(c.store.DB, userID)
	if err != nil {
		return maxByIng
	}
	maxBySlots := free / effMult
	if maxBySlots < maxByIng {
		maxByIng = maxBySlots
	}
	if maxByIng > maxCraftAmount {
		maxByIng = maxCraftAmount
	}
	if maxByIng < 0 {
		maxByIng = 0
	}
	return maxByIng
}

func (c *Cog) buildCraftQuantityView(userID int64, recipeKey, lang string, qty int) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	recipe := crtsvc.Recipes[recipeKey]
	craftMax := c.maxCraftable(userID, recipe)
	if craftMax == 0 {
		craftMax = 1
	}
	if qty < 1 {
		qty = 1
	}
	if qty > craftMax {
		qty = craftMax
	}
	if qty > maxCraftAmount {
		qty = maxCraftAmount
	}
	itemName := items.LocalizedName(recipe.Result, lang)
	// scaled ingredients for qty
	scaledParts := make([]string, 0, len(recipe.Ingredients))
	for ing, base := range recipe.Ingredients {
		scaledParts = append(scaledParts, fmt.Sprintf("%dx %s", base*qty, items.LocalizedName(ing, lang)))
	}
	totalIng := strings.Join(scaledParts, ", ")
	perUnit := recipeIngredients(recipe, lang)
	desc := i18n.T("crafting.qty_desc", lang, map[string]any{
		"ingredients":      perUnit,
		"totalIngredients": totalIng,
		"qty":              qty,
		"max":              craftMax,
		"xp":               recipe.XP * qty,
	})
	embed := components.Embed(
		i18n.T("crafting.qty_title", lang, map[string]any{"item": itemName}),
		desc, components.ColorWarning,
	)
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("crafting.craft_qty_label", lang, map[string]any{"qty": qty})}

	// stepper buttons
	qm1 := qty - 1
	if qm1 < 1 {
		qm1 = 1
	}
	qm5 := qty - 5
	if qm5 < 1 {
		qm5 = 1
	}
	qp1 := qty + 1
	if qp1 > maxCraftAmount {
		qp1 = maxCraftAmount
	}
	if qp1 > craftMax {
		qp1 = craftMax
	}
	qp5 := qty + 5
	if qp5 > maxCraftAmount {
		qp5 = maxCraftAmount
	}
	if qp5 > craftMax {
		qp5 = craftMax
	}

	disableMinus1 := qty <= 1
	disablePlus1 := qty >= craftMax || qty >= maxCraftAmount
	// The ±5 buttons are also disabled when clamping makes them redundant
	// with their ±1 counterpart (same target quantity): two enabled buttons
	// with the same custom_id are just as invalid to Discord as two disabled
	// ones, so this doubles as a duplicate-custom_id guard.
	disableMinus5 := qty <= 1 || qm5 == qm1
	disablePlus5 := qty >= craftMax || qty >= maxCraftAmount || qp5 == qp1

	// Disabled stepper buttons still need a custom_id distinct from every other
	// component on the message — Discord rejects the whole update if two
	// buttons share one, which happens here whenever the clamped target
	// quantity collapses to the same value (e.g. qm1 == qm5 == 1 at qty=1).
	stepID := func(action string, disabled bool, target int) string {
		if disabled {
			return "_disabled_" + action
		}
		return components.EncodeOwner(userID, "crafting", "craft_step", recipeKey, strconv.Itoa(target))
	}
	row1 := []discordgo.MessageComponent{
		discordgo.Button{Label: i18n.T("crafting.step_minus5", lang), CustomID: stepID("m5", disableMinus5, qm5), Style: discordgo.SecondaryButton, Disabled: disableMinus5},
		discordgo.Button{Label: i18n.T("crafting.step_minus1", lang), CustomID: stepID("m1", disableMinus1, qm1), Style: discordgo.SecondaryButton, Disabled: disableMinus1},
		discordgo.Button{Label: fmt.Sprintf("%d/%d", qty, craftMax), CustomID: "_disabled", Style: discordgo.SecondaryButton, Disabled: true},
		discordgo.Button{Label: i18n.T("crafting.step_plus1", lang), CustomID: stepID("p1", disablePlus1, qp1), Style: discordgo.SecondaryButton, Disabled: disablePlus1},
		discordgo.Button{Label: i18n.T("crafting.step_plus5", lang), CustomID: stepID("p5", disablePlus5, qp5), Style: discordgo.SecondaryButton, Disabled: disablePlus5},
	}
	row2 := []discordgo.MessageComponent{
		discordgo.Button{Label: i18n.T("crafting.qty_confirm", lang, map[string]any{"qty": qty}), CustomID: components.EncodeOwner(userID, "crafting", "craft_confirm", recipeKey, strconv.Itoa(qty)), Style: discordgo.SuccessButton, Disabled: qty > craftMax},
		discordgo.Button{Label: i18n.T("crafting.qty_back", lang), CustomID: components.EncodeOwner(userID, "crafting", "craft_back"), Style: discordgo.SecondaryButton},
	}
	comps := []discordgo.MessageComponent{components.ActionRow(row1...), components.ActionRow(row2...)}
	return embed, comps
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
