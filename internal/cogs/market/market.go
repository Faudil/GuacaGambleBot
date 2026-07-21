package market

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	mktsvc "guacagamblebot/internal/service/market"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *mktsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: mktsvc.New(s, cfg)}
	r.Prefix("market", c.onPrefix)
	r.Prefix("market_sell", c.onSellPrefix)
	r.Prefix("ms", c.onSellPrefix)
	r.Prefix("m_s", c.onSellPrefix)
	r.Component("market", "sell", c.onSell)
}

func (c *Cog) onPrefix(b *interaction.Bot, sess *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	categories := c.svc.GetMarketPrices()

	parts := strings.Fields(m.Content)
	var selected []int
	if len(parts) > 1 {
		switch parts[1] {
		case "mine", "minage", "mining":
			selected = []int{0}
		case "fish", "pêche", "fishing":
			selected = []int{1}
		case "farm", "ferme", "farming":
			selected = []int{2}
		}
	}
	if selected == nil {
		selected = []int{0, 1, 2}
	}

	for _, idx := range selected {
		if idx >= len(categories) {
			continue
		}
		cat := categories[idx]
		titleKey := "market.title_" + cat.Name
		embed := components.Embed(i18n.T(titleKey, lang), "", 0xf1c40f)
		var btns []discordgo.MessageComponent
		for _, mi := range cat.Items {
			name := c.displayName(mi.Item.Name, lang)
			embed.Fields = append(embed.Fields, components.Field(
				name,
				i18n.T("market.sale_price", lang, map[string]any{"price": mi.CurrentPrice, "base": mi.BasePrice}),
				true,
			))
			btns = append(btns, components.Button(
				fmt.Sprintf("%s — $%d", name, mi.CurrentPrice),
				components.Encode("market", "sell", mi.Item.Name),
				discordgo.PrimaryButton,
			))
		}
		_, _ = sess.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{components.ActionRow(btns...)},
		})
	}
}

func (c *Cog) onSellPrefix(b *interaction.Bot, sess *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	parts := strings.Fields(m.Content)
	if len(parts) < 2 {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("market.invalid_id", lang))
		return
	}
	itemName := strings.ToLower(parts[1])
	amount := 1
	if len(parts) > 2 {
		if a, err := strconv.Atoi(parts[2]); err == nil && a > 0 {
			amount = a
		}
	}

	gain, err := c.svc.SellItem(userID, itemName, amount)
	if err != nil {
		switch err {
		case mktsvc.ErrNotFound:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("market.item_not_found", lang))
		case mktsvc.ErrNotSellable:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("market.item_not_sellable", lang))
		case mktsvc.ErrNoItem:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("market.no_item", lang, map[string]any{"item": c.displayName(itemName, lang), "amount": amount}))
		default:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("market.invalid_id", lang))
		}
		return
	}
	_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("market.sold_msg", lang, map[string]any{
		"amount": amount, "item": c.displayName(itemName, lang), "gain": gain,
	}))

	if unlocks, err := achievement.CheckAndUnlock(b.DB, userID); err == nil && len(unlocks) > 0 {
		interaction.SendAchievements(b, nil, lang, unlocks)
	}
}

func (c *Cog) onSell(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 1 {
		interaction.RespondError(b, i, lang, "market.invalid_id")
		return
	}
	itemName := rest[0]

	modal := components.ModalResponse(
		components.Encode("market", "sell_confirm", itemName),
		i18n.T("market.sell_modal_title", lang),
		components.TextInput("amount", i18n.T("market.amount_label", lang), true, "1", discordgo.TextInputShort, 1, 5),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) displayName(name, lang string) string {
	k := "items." + name + ".name"
	translated := i18n.T(k, lang)
	if translated == k {
		return name
	}
	return translated
}
