package crafting

import (
	"fmt"
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

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *crtsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: crtsvc.New(s, cfg)}
	r.SlashWithOptions("craft", "Fabriquer un objet à partir de recettes.",
		[]*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "input", Description: "Le nom de l'objet et éventuellement la quantité (ex: iron_sword 3).", Required: true},
		}, c.onSlashCraft)
	r.Slash("recipes", "Voir les recettes de craft disponibles.", c.onSlashRecipes)
	r.Prefix("craft", c.onCraftPrefix)
	r.Prefix("fabriquer", c.onCraftPrefix)
	r.Prefix("recipes", c.onRecipesPrefix)
	r.Prefix("recettes", c.onRecipesPrefix)
	r.Prefix("crafting", c.onRecipesPrefix)
	r.Prefix("rec", c.onRecipesPrefix)
}

func (c *Cog) onSlashCraft(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	input := strings.TrimSpace(i.ApplicationCommandData().Options[0].StringValue())
	if input == "" {
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
				components.Embed("❌", i18n.T("crafting.no_args", lang), 0xe74c3c), nil))
		return
	}

	parts := strings.Fields(input)
	amount := 1
	itemQuery := ""

	if len(parts) > 1 && isNumeric(parts[0]) {
		amount, _ = strconv.Atoi(parts[0])
		itemQuery = strings.Join(parts[1:], " ")
	} else {
		itemQuery = input
	}

	if amount <= 0 {
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
				components.Embed("❌", i18n.T("crafting.invalid_qty", lang), 0xe74c3c), nil))
		return
	}

	itemQuery = strings.ToLower(strings.TrimSpace(itemQuery))

	recipeKey := c.resolveRecipeKey(itemQuery, lang)
	recipe, ok := crtsvc.Recipes[recipeKey]
	if !ok {
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
				components.Embed("❌", i18n.T("crafting.no_recipe", lang, map[string]any{"item": itemQuery}), 0xe74c3c), nil))
		return
	}

	charLeveled, charNewLevel, err := c.svc.Craft(userID, recipeKey, amount)
	if err != nil {
		var msg string
		switch err {
		case crtsvc.ErrNoLevel:
			msg = i18n.T("crafting.no_level", lang, map[string]any{"level": recipe.LevelRequired})
		case crtsvc.ErrNoIngredients:
			msg = i18n.T("crafting.no_ingredients", lang, map[string]any{"missing": "..."})
		case crtsvc.ErrNoRecipe:
			msg = i18n.T("crafting.no_recipe", lang, map[string]any{"item": itemQuery})
		case crtsvc.ErrResearchRequired:
			rn := c.researchName(recipe.RequiredResearch)
			if !c.isResearchCompleted(userID, recipe.RequiredResearch2) {
				rn = c.researchName(recipe.RequiredResearch2)
			}
			msg = fmt.Sprintf("🔬 You need to complete the **%s** research first!", rn)
		default:
			msg = err.Error()
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
				components.Embed("❌", msg, 0xe74c3c), nil))
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

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
			components.Embed("✅", msg, 0x2ecc71), nil))
}

func recipeDisplayInfo(recipe crtsvc.Recipe, lang string) string {
	ingStrs := make([]string, 0, len(recipe.Ingredients))
	for ing, qty := range recipe.Ingredients {
		ingStrs = append(ingStrs, fmt.Sprintf("%dx %s", qty, items.DisplayName(ing)))
	}
	ingStr := strings.Join(ingStrs, ", ")
	resName := items.DisplayName(recipe.Result)

	if recipe.IsEquipment {
		it := items.Get(recipe.Result)
		if it != nil {
			extra := ""
			if it.SetID != "" && it.SetName != "" {
				extra = fmt.Sprintf(" [%s Set]", it.SetName)
			}
			return fmt.Sprintf("%s (%s%s) | %s", resName, string(it.Rarity), extra, ingStr)
		}
		return fmt.Sprintf("%s [Equipment] | %s", resName, ingStr)
	}
	return fmt.Sprintf("%s | %s", resName, ingStr)
}

func (c *Cog) onSlashRecipes(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	level := c.svc.GetCrafterLevel(userID)

	var unlocked, locked []string
	for _, recipe := range crtsvc.Recipes {
		resLine := recipeDisplayInfo(recipe, lang)
		levelOK := recipe.LevelRequired <= level
		researchOK := c.isResearchCompleted(userID, recipe.RequiredResearch)
		research2OK := c.isResearchCompleted(userID, recipe.RequiredResearch2)
		if levelOK && researchOK && research2OK {
			unlocked = append(unlocked, "✅ "+resLine)
		} else {
			reqs := ""
			if !levelOK {
				reqs = fmt.Sprintf("Lvl %d", recipe.LevelRequired)
			}
			if !researchOK {
				rName := c.researchName(recipe.RequiredResearch)
				reqs = addReq(reqs, "🔬"+rName)
			}
			if !research2OK {
				rName := c.researchName(recipe.RequiredResearch2)
				reqs = addReq(reqs, "🔬"+rName)
			}
			locked = append(locked, fmt.Sprintf("🔒 %s | %s", resLine, reqs))
		}
	}

	desc := i18n.T("crafting.desc_intro", lang)
	desc += i18n.T("crafting.unlocked_title", lang)
	if len(unlocked) > 0 {
		desc += strings.Join(unlocked, "\n") + "\n\n"
	} else {
		desc += i18n.T("crafting.no_unlocked", lang)
	}
	if len(locked) > 0 {
		desc += i18n.T("crafting.locked_title", lang) + strings.Join(locked, "\n")
	}

	embed := components.Embed(
		i18n.T("crafting.title", lang, map[string]any{"level": level}),
		desc, 0xe67e22,
	)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func addReq(reqs, add string) string {
	if reqs == "" {
		return add
	}
	return reqs + " + " + add
}

func (c *Cog) onRecipesPrefix(b *interaction.Bot, sess *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	level := c.svc.GetCrafterLevel(userID)

	var unlocked, locked []string
	for _, recipe := range crtsvc.Recipes {
		resLine := recipeDisplayInfo(recipe, lang)
		levelOK := recipe.LevelRequired <= level
		researchOK := c.isResearchCompleted(userID, recipe.RequiredResearch)
		research2OK := c.isResearchCompleted(userID, recipe.RequiredResearch2)
		if levelOK && researchOK && research2OK {
			unlocked = append(unlocked, "✅ "+resLine)
		} else {
			reqs := ""
			if !levelOK {
				reqs = fmt.Sprintf("Lvl %d", recipe.LevelRequired)
			}
			if !researchOK {
				rName := c.researchName(recipe.RequiredResearch)
				reqs = addReq(reqs, "🔬"+rName)
			}
			if !research2OK {
				rName := c.researchName(recipe.RequiredResearch2)
				reqs = addReq(reqs, "🔬"+rName)
			}
			locked = append(locked, fmt.Sprintf("🔒 %s | %s", resLine, reqs))
		}
	}

	desc := i18n.T("crafting.desc_intro", lang)
	desc += i18n.T("crafting.unlocked_title", lang)
	if len(unlocked) > 0 {
		desc += strings.Join(unlocked, "\n") + "\n\n"
	} else {
		desc += i18n.T("crafting.no_unlocked", lang)
	}
	if len(locked) > 0 {
		desc += i18n.T("crafting.locked_title", lang) + strings.Join(locked, "\n")
	}

	embed := components.Embed(
		i18n.T("crafting.title", lang, map[string]any{"level": level}),
		desc, 0xe67e22,
	)
	_, _ = sess.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func (c *Cog) onCraftPrefix(b *interaction.Bot, sess *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	content := strings.TrimSpace(strings.TrimPrefix(m.Content, "!craft "))
	content = strings.TrimSpace(strings.TrimPrefix(content, "!fabriquer "))

	if content == "" {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_args", lang))
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
	if _, ok := crtsvc.Recipes[recipeKey]; !ok {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_recipe", lang, map[string]any{"item": itemQuery}))
		return
	}

	charLeveled, charNewLevel, err := c.svc.Craft(userID, recipeKey, amount)
	if err != nil {
		switch err {
		case crtsvc.ErrNoLevel:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_level", lang, map[string]any{"level": crtsvc.Recipes[recipeKey].LevelRequired}))
		case crtsvc.ErrNoIngredients:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_ingredients", lang, map[string]any{"missing": "..."}))
		case crtsvc.ErrNoRecipe:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_recipe", lang, map[string]any{"item": itemQuery}))
		case crtsvc.ErrResearchRequired:
			recipe := crtsvc.Recipes[recipeKey]
			rName := c.researchName(recipe.RequiredResearch)
			if !c.isResearchCompleted(userID, recipe.RequiredResearch2) {
				rName = c.researchName(recipe.RequiredResearch2)
			}
			_, _ = sess.ChannelMessageSend(m.ChannelID, "🔬 You need to complete the **"+rName+"** research first!")
		default:
			_, _ = sess.ChannelMessageSend(m.ChannelID, "❌ "+err.Error())
		}
		return
	}

	recipe := crtsvc.Recipes[recipeKey]
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
	return items.DisplayName(name)
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
