package inventory

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	invsvc "guacagamblebot/internal/service/inventory"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *invsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: invsvc.New(s, cfg)}
	r.Slash("inventory", "Voir ton inventaire.", c.onSlashMenu)
	r.Prefix("inventory", c.onPrefix)
	r.Prefix("inv", c.onPrefix)
	r.Prefix("bag", c.onPrefix)
	r.Prefix("sac", c.onPrefix)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	result, err := c.svc.GetInventory(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "inventory.error")
		return
	}

	if len(result.Entries) == 0 {
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
				components.Embed("", i18n.T("inventory.empty", lang, map[string]any{"user": interaction.Mention(userID)}), 0xe74c3c), nil))
		return
	}

	embed := components.Embed(
		i18n.T("inventory.title", lang, map[string]any{"user": i.Member.User.Username}),
		"", 0x3498db,
	)
	embed.Fields = buildFields(result, lang)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("inventory.footer", lang) + fmt.Sprintf(" — %d/%d", result.Current, result.Limit),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onPrefix(b *interaction.Bot, sess *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)

	result, err := c.svc.GetInventory(userID)
	if err != nil {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("inventory.error", lang))
		return
	}

	if len(result.Entries) == 0 {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("inventory.empty", lang, map[string]any{"user": m.Author.Mention()}))
		return
	}

	embed := components.Embed(
		i18n.T("inventory.title", lang, map[string]any{"user": m.Author.Username}),
		"", 0x3498db,
	)
	embed.Fields = buildFields(result, lang)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("inventory.footer", lang) + fmt.Sprintf(" — %d/%d", result.Current, result.Limit),
	}
	_, _ = sess.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func buildFields(res *invsvc.InvResult, lang string) []*discordgo.MessageEmbedField {
	catOrder := []string{"mining", "fishing", "farming", "archeology", "food", "tools", "materials", "special"}
	grouped := make(map[string][]invsvc.InvEntry)
	for _, e := range res.Entries {
		cat := "other"
		if e.Item != nil {
			cat = string(e.Item.Category)
		}
		grouped[cat] = append(grouped[cat], e)
	}
	var fields []*discordgo.MessageEmbedField
	for _, cat := range catOrder {
		entries, ok := grouped[cat]
		if !ok {
			continue
		}
		val := ""
		for _, e := range entries {
			emoji := "⚪"
			if e.Item != nil {
				emoji = e.Item.Emoji
			}
			val += fmt.Sprintf("%s **%s** : `x%d`\n", emoji, displayName(e.ItemName, lang), e.Quantity)
		}
		catName := i18n.T("inventory.category_"+cat, lang)
		fields = append(fields, components.Field(catName, val, false))
	}
	return fields
}

func displayName(name, lang string) string {
	return items.DisplayName(name)
}
