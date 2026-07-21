package shop

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	shop "guacagamblebot/internal/service/shop"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *shop.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: shop.New(s, cfg)}
	r.Prefix("shop", c.onPrefix)
	r.Prefix("boutique", c.onPrefix)
	r.Component("shop", "buy", c.onBuy)
}

func (c *Cog) onPrefix(b *interaction.Bot, sess *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)

	offers := c.svc.DailyOffers(4)
	if len(offers) == 0 {
		_, _ = sess.ChannelMessageSend(m.ChannelID, "❌ No items available today.")
		return
	}

	bal, _ := c.store.GetBalance(userID)
	desc := fmt.Sprintf(i18n.T("shop.personal_desc", lang)+"\n\n", m.Author.Mention())
	embed := components.Embed(
		i18n.T("shop.personal_title", lang, map[string]any{"user": m.Author.Username}),
		desc, 0x3498db,
	)
	embed.Fields = make([]*discordgo.MessageEmbedField, 0, len(offers))
	var btns []discordgo.MessageComponent
	for i, offer := range offers {
		priceStr := fmt.Sprintf("$%d", offer.Price)
		if offer.Discounted {
			priceStr += " 🔥"
		}
		name := c.displayName(offer.Item.Name, lang)
		embed.Fields = append(embed.Fields, components.Field(
			name,
			fmt.Sprintf("%s\n%s: **%s**", offer.Item.Description, i18n.T("shop.price_label", lang), priceStr),
			true,
		))
		btns = append(btns, components.Button(
			fmt.Sprintf("%s (%s)", name, priceStr),
			components.Encode("shop", "buy", fmt.Sprintf("%d", i), fmt.Sprintf("%d", userID)),
			discordgo.PrimaryButton,
		))
	}
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("shop.balance_footer", lang, map[string]any{"balance": bal}),
	}
	_, _ = sess.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{components.ActionRow(btns...)},
	})
}

func (c *Cog) onBuy(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 2 {
		interaction.RespondError(b, i, lang, "shop.error")
		return
	}
	offerIdx := interaction.ToInt64(rest[0])
	ownerID := interaction.ToInt64(rest[1])
	userID := interaction.ToInt64(interaction.UserID(i))

	if userID != ownerID {
		interaction.RespondError(b, i, lang, "shop.not_your_shop")
		return
	}
	offers := c.svc.DailyOffers(4)
	if int(offerIdx) >= len(offers) {
		interaction.RespondError(b, i, lang, "shop.error")
		return
	}
	offer := offers[offerIdx]

	if err := c.svc.BuyItem(userID, offer.Item.Name, 1); err != nil {
		switch err {
		case shop.ErrNoMoney:
			interaction.RespondError(b, i, lang, "shop.too_broke_item")
		default:
			interaction.RespondError(b, i, lang, "shop.error")
		}
		return
	}

	embed := i18n.T("shop.buy_success", lang, map[string]any{"item": c.displayName(offer.Item.Name, lang), "price": offer.Price})
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: embed, Flags: discordgo.MessageFlagsEphemeral},
	})

	if unlocks, err := achievement.CheckAndUnlock(b.DB, userID); err == nil && len(unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, unlocks)
	}
}

func (c *Cog) displayName(name, lang string) string {
	k := "items." + name + ".name"
	translated := i18n.T(k, lang)
	if translated == k {
		return name
	}
	return translated
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
