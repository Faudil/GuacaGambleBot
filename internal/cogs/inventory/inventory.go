package inventory

import (
	"fmt"
	"strings"

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
	catOrder := []string{"mining", "fishing", "farming", "archeology", "food", "tools", "materials", "equipment", "special"}
	grouped := make(map[string][]invsvc.InvEntry)
	for _, e := range res.Entries {
		cat := "other"
		if e.Item != nil {
			cat = string(e.Item.Category)
			if cat == "delve" {
				cat = "equipment"
			}
		}
		if e.EquipInfo != nil {
			cat = "equipment"
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
			if e.EquipInfo != nil {
				info := e.EquipInfo
				rarEmoji := "⬜"
				switch info.Rarity {
				case "uncommon":
					rarEmoji = "🟩"
				case "rare":
					rarEmoji = "🔵"
				case "epic":
					rarEmoji = "🟣"
				case "legendary":
					rarEmoji = "🟠"
				}
				statParts := []string{}
				if info.StatSTR > 0 {
					statParts = append(statParts, fmt.Sprintf("STR+%d", info.StatSTR))
				}
				if info.StatDEX > 0 {
					statParts = append(statParts, fmt.Sprintf("DEX+%d", info.StatDEX))
				}
				if info.StatINT > 0 {
					statParts = append(statParts, fmt.Sprintf("INT+%d", info.StatINT))
				}
				if info.StatVIT > 0 {
					statParts = append(statParts, fmt.Sprintf("VIT+%d", info.StatVIT))
				}
				if info.StatLUK > 0 {
					statParts = append(statParts, fmt.Sprintf("LUK+%d", info.StatLUK))
				}
				statStr := ""
				if len(statParts) > 0 {
					statStr = " (`" + strings.Join(statParts, " ") + "`)"
				}
				tag := ""
				if info.IsEquipped {
					tag = " ✅"
				}
				val += fmt.Sprintf("%s %s **%s**%s%s\n", rarEmoji, info.Emoji, e.ItemName, statStr, tag)
			} else {
				emoji := "⚪"
				if e.Item != nil {
					emoji = e.Item.Emoji
				}
				val += fmt.Sprintf("%s **%s** : `x%d`\n", emoji, displayName(e.ItemName, lang), e.Quantity)
			}
		}
		if val != "" {
			catName := i18n.T("inventory.category_"+cat, lang)
			fields = append(fields, components.Field(catName, val, false))
		}
	}

	// Show equipment that didn't fit in the standard categories
	if eqEntries, ok := grouped["equipment"]; ok {
		_ = eqEntries // already handled above in the loop
	}
	return fields
}

func displayName(name, lang string) string {
	return items.DisplayName(name)
}
