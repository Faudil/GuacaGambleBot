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

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: crtsvc.New(s, cfg)}
	r.Slash("craft", "Fabriquer un objet à partir de recettes.", c.onSlashCraft)
	r.Slash("recipes", "Voir les recettes de craft disponibles.", c.onSlashRecipes)
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
}

func (c *Cog) onSlashCraft(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.buildCraftMenu(userID, lang, "all", 1)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
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

	itemQuery = strings.ToLower(strings.TrimSpace(itemQuery))

	recipeKey := c.resolveRecipeKey(itemQuery, lang)
	recipe, ok := crtsvc.Recipes[recipeKey]
	if !ok {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_recipe", lang, map[string]any{"item": itemQuery}))
		return
	}

	charLeveled, charNewLevel, err := c.svc.Craft(userID, recipeKey, amount)
	if err != nil {
		switch err {
		case crtsvc.ErrNoLevel:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_level", lang, map[string]any{"level": recipe.LevelRequired}))
		case crtsvc.ErrNoIngredients:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_ingredients", lang, map[string]any{"missing": "..."}))
		case crtsvc.ErrNoRecipe:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_recipe", lang, map[string]any{"item": itemQuery}))
		case crtsvc.ErrResearchRequired:
			rName := c.researchName(recipe.RequiredResearch)
			if !c.isResearchCompleted(userID, recipe.RequiredResearch2) {
				rName = c.researchName(recipe.RequiredResearch2)
			}
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_research", lang, map[string]any{"research": rName}))
		default:
			_, _ = sess.ChannelMessageSend(m.ChannelID, "❌ "+err.Error())
		}
		return
	}

	resDisplay := c.displayName(recipe.Result, lang)
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

// unlockedRecipeKeys returns the recipe map keys the user can currently craft:
// level and both research gates satisfied.
func (c *Cog) unlockedRecipeKeys(userID int64, level int) []string {
	var out []string
	for key, recipe := range crtsvc.Recipes {
		if recipe.LevelRequired <= level &&
			c.isResearchCompleted(userID, recipe.RequiredResearch) &&
			c.isResearchCompleted(userID, recipe.RequiredResearch2) {
			out = append(out, key)
		}
	}
	return out
}

// buildRecipesView renders one page of the user's unlocked recipes with
// prev/page/next navigation. The page is clamped, so a stale button press
// still lands on a valid page.
func (c *Cog) buildRecipesView(userID int64, level int, lang string, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	keys := c.unlockedRecipeKeys(userID, level)
	totalPages := max(1, int(math.Ceil(float64(len(keys))/float64(recipesPerPage))))
	page = max(1, min(page, totalPages))

	desc := i18n.T("crafting.desc_intro", lang)
	desc += i18n.T("crafting.unlocked_title", lang)
	if len(keys) == 0 {
		desc += i18n.T("crafting.no_unlocked", lang)
	} else {
		start := (page - 1) * recipesPerPage
		end := min(start+recipesPerPage, len(keys))
		lines := make([]string, 0, end-start)
		for _, key := range keys[start:end] {
			lines = append(lines, "✅ "+recipeDisplayInfo(crtsvc.Recipes[key], lang))
		}
		desc += strings.Join(lines, "\n")
	}

	embed := components.Embed(
		i18n.T("crafting.title", lang, map[string]any{"level": level}),
		desc, 0xe67e22,
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
		"berserker_elixir", "adamant_tonic", "gale_draught", "oracles_insight":
		return "pets"
	case "garden_plot", "tropical_greenhouse", "enchanted_orchard":
		return "structures"
	case "bow", "rusty_magnet", "hook", "magnet", "electric_magnet":
		return "tools"
	default:
		return "consumables"
	}
}

// craftRecipes returns the recipe keys the user can craft, filtered by
// category and ordered by level then name so the menu stays stable across
// interactions (map iteration is random).
func (c *Cog) craftRecipes(userID int64, lang, category string) []string {
	level := c.svc.GetCrafterLevel(userID)
	keys := c.unlockedRecipeKeys(userID, level)
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
		if a.LevelRequired != b.LevelRequired {
			return a.LevelRequired < b.LevelRequired
		}
		return c.displayName(a.Result, lang) < c.displayName(b.Result, lang)
	})
	return keys
}

// buildCraftMenu renders the /craft embed and its components: a category
// filter, a recipe select menu and prev/page/next navigation.
func (c *Cog) buildCraftMenu(userID int64, lang, category string, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	level := c.svc.GetCrafterLevel(userID)
	recipes := c.craftRecipes(userID, lang, category)
	totalPages := max(1, int(math.Ceil(float64(len(recipes))/float64(maxMenuOptions))))
	page = max(1, min(page, totalPages))

	catLabel := i18n.T("crafting.cat_all", lang)
	if category != "" && category != "all" {
		catLabel = i18n.T("crafting.cat_"+category, lang)
	}

	desc := i18n.T("crafting.menu_desc", lang, map[string]any{"filter": catLabel})
	embed := components.Embed(i18n.T("crafting.menu_title", lang, map[string]any{"level": level}), desc, 0xe67e22)

	if len(recipes) == 0 {
		embed.Description += i18n.T("crafting.no_unlocked", lang)
	} else {
		start := (page - 1) * maxMenuOptions
		end := min(start+maxMenuOptions, len(recipes))
		for _, key := range recipes[start:end] {
			embed.Fields = append(embed.Fields, recipeField(crtsvc.Recipes[key], lang))
		}
	}
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("crafting.nav_page", lang, map[string]any{"page": page, "total": totalPages}),
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(c.categoryFilter(category, lang)),
		components.ActionRow(c.recipeSelect(userID, recipes, page, lang)),
		components.ActionRow(c.craftNavButtons(userID, category, page, totalPages, lang)...),
	}
	return embed, comps
}

func recipeField(recipe crtsvc.Recipe, lang string) *discordgo.MessageEmbedField {
	return components.Field(
		fmt.Sprintf("%s %s", recipeEmoji(recipe), items.LocalizedName(recipe.Result, lang)),
		i18n.T("crafting.menu_recipe_line", lang, map[string]any{
			"ingredients": recipeIngredients(recipe, lang),
			"xp":          recipe.XP,
		}),
		true,
	)
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

func (c *Cog) categoryFilter(category, lang string) discordgo.SelectMenu {
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
		CustomID:    components.Encode("crafting", "craft_filter"),
		Placeholder: i18n.T("crafting.filter_placeholder", lang),
		Options:     opts,
	}
}

func (c *Cog) recipeSelect(userID int64, recipes []string, page int, lang string) discordgo.SelectMenu {
	options := make([]discordgo.SelectMenuOption, 0, maxMenuOptions)
	start := (page - 1) * maxMenuOptions
	end := min(start+maxMenuOptions, len(recipes))
	for _, key := range recipes[start:end] {
		recipe := crtsvc.Recipes[key]
		desc := recipeIngredients(recipe, lang)
		if len(desc) > 100 {
			desc = desc[:97] + "..."
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       c.displayName(recipe.Result, lang),
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

	totalPages := max(1, int(math.Ceil(float64(len(c.craftRecipes(userID, lang, category)))/float64(maxMenuOptions))))
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

// onCraftPick crafts one copy of the recipe chosen in the menu and replaces
// the menu with the result.
func (c *Cog) onCraftPick(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	values := i.MessageComponentData().Values
	if len(values) == 0 || values[0] == "_none" {
		interaction.RespondError(b, i, lang, "crafting.empty")
		return
	}
	recipeKey := values[0]
	recipe, ok := crtsvc.Recipes[recipeKey]
	if !ok {
		interaction.RespondError(b, i, lang, "crafting.no_recipe", map[string]any{"item": recipeKey})
		return
	}

	charLeveled, charNewLevel, err := c.svc.Craft(userID, recipeKey, 1)
	if err != nil {
		var msg string
		switch err {
		case crtsvc.ErrNoLevel:
			msg = i18n.T("crafting.no_level", lang, map[string]any{"level": recipe.LevelRequired})
		case crtsvc.ErrNoIngredients:
			msg = i18n.T("crafting.no_ingredients", lang, map[string]any{"missing": "..."})
		case crtsvc.ErrNoRecipe:
			msg = i18n.T("crafting.no_recipe", lang, map[string]any{"item": recipeKey})
		case crtsvc.ErrResearchRequired:
			rn := c.researchName(recipe.RequiredResearch)
			if !c.isResearchCompleted(userID, recipe.RequiredResearch2) {
				rn = c.researchName(recipe.RequiredResearch2)
			}
			msg = i18n.T("crafting.no_research", lang, map[string]any{"research": rn})
		default:
			msg = err.Error()
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
				components.Embed("❌", msg, 0xe74c3c), nil))
		return
	}

	resDisplay := c.displayName(recipe.Result, lang)
	msg := i18n.T("crafting.success_msg", lang, map[string]any{"amount": 1, "item": resDisplay, "xp": recipe.XP})

	if leveledUp, newLevel := c.svc.LevelUpCheck(userID); leveledUp {
		msg += i18n.T("crafting.level_up", lang, map[string]any{"level": newLevel})
	}
	if charLeveled {
		msg += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": charNewLevel})
	}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
			components.Embed("✅", msg, 0x2ecc71), nil))
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
		if strings.ToLower(c.displayName(key, lang)) == q {
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

func (c *Cog) displayName(name, lang string) string {
	return items.LocalizedName(name, lang)
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
