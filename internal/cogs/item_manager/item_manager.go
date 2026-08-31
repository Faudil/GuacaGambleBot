package item_manager

import (
	"fmt"

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
	r.Slash("trade", "cmd.trade.desc", c.onSlashTrade)
	// /sell and !sell are owned by the trade cog; registering them here too
	// would duplicate the command name and fail Discord's bulk overwrite.
	// The accept/decline components stay registered so in-flight proposals
	// created before the change can still be resolved.
	r.Prefix("trade", c.onTradePrefix)
	r.Component("item_manager", "accept", c.onAccept)
	r.Component("item_manager", "decline", c.onDecline)
}

func (c *Cog) onSlashTrade(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
			components.Embed("📦", i18n.T("item_manager.trade_usage", lang), components.ColorWarning), nil))
}

func (c *Cog) onTradePrefix(b *interaction.Bot, sess *discordgo.Session, m *discordgo.Message) {
	_ = b
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("item_manager.trade_usage", lang))
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
