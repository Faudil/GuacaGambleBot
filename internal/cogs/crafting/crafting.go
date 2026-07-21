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
	crtsvc "guacagamblebot/internal/service/crafting"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *crtsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: crtsvc.New(s, cfg)}
	r.Prefix("craft", c.onCraftPrefix)
	r.Prefix("fabriquer", c.onCraftPrefix)
	r.Prefix("recipes", c.onRecipesPrefix)
	r.Prefix("recettes", c.onRecipesPrefix)
	r.Prefix("crafting", c.onRecipesPrefix)
}

func (c *Cog) onRecipesPrefix(b *interaction.Bot, sess *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	level := c.svc.GetCrafterLevel(userID)

	var unlocked, locked []string
	for _, recipe := range crtsvc.Recipes {
		ingStrs := make([]string, 0, len(recipe.Ingredients))
		for ing, qty := range recipe.Ingredients {
			ingStrs = append(ingStrs, fmt.Sprintf("%dx %s", qty, c.displayName(ing, lang)))
		}
		ingStr := strings.Join(ingStrs, ", ")
		resName := c.displayName(recipe.Result, lang)
		if recipe.LevelRequired <= level {
			unlocked = append(unlocked, i18n.T("crafting.unlock_line", lang, map[string]any{"item": resName, "ingredients": ingStr}))
		} else {
			locked = append(locked, i18n.T("crafting.lock_line", lang, map[string]any{"item": resName, "level": recipe.LevelRequired, "ingredients": ingStr}))
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

	if err := c.svc.Craft(userID, recipeKey, amount); err != nil {
		switch err {
		case crtsvc.ErrNoLevel:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_level", lang, map[string]any{"level": crtsvc.Recipes[recipeKey].LevelRequired}))
		case crtsvc.ErrNoIngredients:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_ingredients", lang, map[string]any{"missing": "..."}))
		case crtsvc.ErrNoRecipe:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("crafting.no_recipe", lang, map[string]any{"item": itemQuery}))
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

	_, _ = sess.ChannelMessageSend(m.ChannelID, msg)
}

func (c *Cog) resolveRecipeKey(query, lang string) string {
	if _, ok := crtsvc.Recipes[query]; ok {
		return query
	}
	for key := range crtsvc.Recipes {
		if strings.ToLower(key) == query {
			return key
		}
		if strings.ToLower(c.displayName(key, lang)) == query {
			return key
		}
	}
	return query
}

func (c *Cog) displayName(name, lang string) string {
	k := "items." + name + ".name"
	translated := i18n.T(k, lang)
	if translated == k {
		return name
	}
	return translated
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
