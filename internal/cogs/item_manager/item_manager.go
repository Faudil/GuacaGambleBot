package item_manager

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	tradesvc "guacagamblebot/internal/service/item_manager"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *tradesvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: tradesvc.New(s, cfg)}
	r.Slash("trade", "Échanger des objets avec un autre joueur.", c.onSlashTrade)
	r.SlashWithOptions("sell", "Vendre un objet à un autre joueur.",
		[]*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionUser, Name: "recipient", Description: "Le joueur à qui vendre.", Required: true},
			{Type: discordgo.ApplicationCommandOptionString, Name: "item", Description: "Le nom de l'objet à vendre.", Required: true},
			{Type: discordgo.ApplicationCommandOptionInteger, Name: "price", Description: "Le prix de vente.", Required: true},
		}, c.onSlashSell)
	r.Prefix("trade", c.onTradePrefix)
	r.Prefix("sell", c.onSellPrefix)
	r.Component("item_manager", "accept", c.onAccept)
	r.Component("item_manager", "decline", c.onDecline)
}

func (c *Cog) onSlashTrade(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
			components.Embed("📦", i18n.T("item_manager.trade_usage", lang), 0xe67e22), nil))
}

func (c *Cog) onSlashSell(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	opts := i.ApplicationCommandData().Options
	recipientID := interaction.ToInt64(opts[0].StringValue())
	itemName := strings.ToLower(opts[1].StringValue())
	priceVal := int(opts[2].IntValue())

	if priceVal <= 0 {
		interaction.RespondError(b, i, lang, "loan.invalid_amount")
		return
	}

	sellerID := interaction.ToInt64(interaction.UserID(i))
	if sellerID == recipientID {
		interaction.RespondError(b, i, lang, "economy.give_invalid")
		return
	}

	embed := components.Embed(
		i18n.T("item_manager.trade_proposal_title", lang),
		i18n.T("item_manager.trade_proposal_desc", lang, map[string]any{
			"seller": interaction.Mention(sellerID),
			"item":   items.LocalizedName(itemName, lang),
			"price":  priceVal,
			"buyer":  interaction.Mention(recipientID),
		}),
		0xe67e22,
	)

	btns := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("item_manager.accept_label", lang), components.Encode("item_manager", "accept", fmt.Sprint(sellerID), fmt.Sprint(recipientID), itemName, fmt.Sprint(priceVal)), discordgo.SuccessButton),
			components.Button(i18n.T("item_manager.refuse_label", lang), components.Encode("item_manager", "decline", fmt.Sprint(sellerID), fmt.Sprint(recipientID)), discordgo.DangerButton),
		),
	}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, btns))
}

func (c *Cog) onTradePrefix(b *interaction.Bot, sess *discordgo.Session, m *discordgo.Message) {
	_ = b
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("item_manager.trade_usage", lang))
}

func (c *Cog) onSellPrefix(b *interaction.Bot, sess *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	content := strings.TrimSpace(strings.TrimPrefix(m.Content, "!sell "))
	if content == "" {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("item_manager.trade_usage", lang))
		return
	}

	parts := strings.SplitN(content, " ", 3)
	if len(parts) < 3 {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("item_manager.trade_usage", lang))
		return
	}

	recipientMention := parts[0]
	itemName := strings.ToLower(parts[1])
	priceVal := int(interaction.ToInt64(parts[2]))
	if priceVal <= 0 {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("loan.invalid_amount", lang))
		return
	}

	recipientID, ok := interaction.ParseUserID(recipientMention)
	if !ok {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("economy.give_invalid", lang))
		return
	}

	sellerID := interaction.ToInt64(m.Author.ID)
	if sellerID == recipientID {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("economy.give_invalid", lang))
		return
	}

	embed := components.Embed(
		i18n.T("item_manager.trade_proposal_title", lang),
		i18n.T("item_manager.trade_proposal_desc", lang, map[string]any{
			"seller": m.Author.Mention(),
			"item":   items.LocalizedName(itemName, lang),
			"price":  priceVal,
			"buyer":  "<@" + fmt.Sprint(recipientID) + ">",
		}),
		0xe67e22,
	)

	btns := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("item_manager.accept_label", lang), components.Encode("item_manager", "accept", fmt.Sprint(sellerID), fmt.Sprint(recipientID), itemName, fmt.Sprint(priceVal)), discordgo.SuccessButton),
			components.Button(i18n.T("item_manager.refuse_label", lang), components.Encode("item_manager", "decline", fmt.Sprint(sellerID), fmt.Sprint(recipientID)), discordgo.DangerButton),
		),
	}

	_, _ = sess.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Content:    "<@" + fmt.Sprint(recipientID) + ">",
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: btns,
	})
}

func (c *Cog) onAccept(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 5 {
		interaction.RespondError(b, i, lang, "item_manager.unknown_error")
		return
	}
	sellerID := interaction.ToInt64(rest[0])
	buyerID := interaction.ToInt64(rest[1])
	itemName := rest[2]
	price := int(interaction.ToInt64(rest[3]))
	_ = rest[4]

	userID := interaction.ToInt64(interaction.UserID(i))
	if userID != buyerID {
		interaction.RespondError(b, i, lang, "item_manager.not_for_you")
		return
	}

	result := c.svc.TransferItem(sellerID, buyerID, itemName, price)
	displayName := items.LocalizedName(itemName, lang)

	if result == tradesvc.TradeSuccess {
		content := i18n.T("item_manager.trade_success", lang, map[string]any{
			"seller": "<@" + fmt.Sprint(sellerID) + ">",
			"item":   displayName,
			"buyer":  "<@" + fmt.Sprint(buyerID) + ">",
			"price":  price,
		})
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{Content: content, Components: []discordgo.MessageComponent{}},
		})
		for _, uid := range []int64{sellerID, buyerID} {
			if unlocks, err := achievement.CheckAndUnlock(b.DB, uid); err == nil && len(unlocks) > 0 {
				interaction.SendAchievements(b, i, lang, unlocks)
			}
		}
	} else {
		var key string
		switch result {
		case tradesvc.TradeNoMoney:
			key = "item_manager.no_money"
		case tradesvc.TradeNoItem:
			key = "item_manager.no_item"
		case tradesvc.TradeNoSpace:
			key = "inventory.full"
		default:
			key = "item_manager.unknown_error"
		}
		interaction.RespondError(b, i, lang, key)
	}
}

func (c *Cog) onDecline(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 2 {
		interaction.RespondError(b, i, lang, "item_manager.unknown_error")
		return
	}
	sellerID := interaction.ToInt64(rest[0])
	buyerID := interaction.ToInt64(rest[1])
	userID := interaction.ToInt64(interaction.UserID(i))
	if userID != sellerID && userID != buyerID {
		interaction.RespondError(b, i, lang, "item_manager.not_for_you")
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Content: i18n.T("item_manager.trade_cancelled", lang), Components: []discordgo.MessageComponent{}},
	})
}
