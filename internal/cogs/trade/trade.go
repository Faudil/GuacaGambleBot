// Package trade implements the /sell command: a player offers an item to a
// specific other player for a price, and that player accepts or declines.
package trade

import (
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
	invsvc "guacagamblebot/internal/service/inventory"
	tradesvc "guacagamblebot/internal/service/trade"
	"guacagamblebot/internal/store"
)

var minOne = float64(1)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *tradesvc.Service
	inv   *invsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: tradesvc.New(s, cfg), inv: invsvc.New(s, cfg)}
	r.SlashWithOptions("sell", "cmd.sell.desc",
		[]*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "item",
				Description:  "The item to sell",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "quantity",
				Description: "How many to sell",
				Required:    true,
				MinValue:    &minOne,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "price",
				Description: "Total price in coins",
				Required:    true,
				MinValue:    &minOne,
			},
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "buyer",
				Description: "Who you're selling to",
				Required:    true,
			},
		},
		c.onSlashSell)
	r.Prefix("sell", c.onPrefixSell)
	r.Component("trade", "accept", c.onAccept)
	r.Component("trade", "decline", c.onDecline)
}

func (c *Cog) onSlashSell(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		c.handleSellAutocomplete(b, i, userID)
		return
	}

	var itemQuery string
	var quantity, price int
	var buyerID int64
	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "item":
			itemQuery, _ = opt.Value.(string)
		case "quantity":
			if v, ok := opt.Value.(float64); ok {
				quantity = int(v)
			}
		case "price":
			if v, ok := opt.Value.(float64); ok {
				price = int(v)
			}
		case "buyer":
			if v, ok := opt.Value.(string); ok {
				buyerID = interaction.ToInt64(v)
			}
		}
	}

	offer, it, err := c.svc.CreateOffer(userID, buyerID, itemQuery, quantity, price)
	if err != nil {
		c.handleTradeError(b, i, lang, err)
		return
	}
	embed, comps := c.buildOfferMessage(offer, it, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) handleSellAutocomplete(b *interaction.Bot, i *discordgo.InteractionCreate, userID int64) {
	focused := ""
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "item" && opt.Focused {
			if v, ok := opt.Value.(string); ok {
				focused = strings.ToLower(v)
			}
		}
	}
	result, err := c.inv.GetInventory(userID)
	choices := []*discordgo.ApplicationCommandOptionChoice{}
	if err == nil {
		for _, e := range result.Entries {
			if e.EquipInfo != nil || e.Item == nil || e.Quantity <= 0 {
				continue
			}
			name := e.ItemName
			if focused != "" && !strings.Contains(strings.ToLower(name), focused) && !strings.Contains(strings.ToLower(e.Item.ID), focused) {
				continue
			}
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: name, Value: e.Item.ID})
			if len(choices) >= 25 {
				break
			}
		}
	}
	sort.Slice(choices, func(a, bb int) bool { return choices[a].Name < choices[bb].Name })
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}

func (c *Cog) onPrefixSell(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)

	content := strings.TrimSpace(strings.TrimPrefix(m.Content, b.Prefix+"sell"))
	parts := strings.Fields(content)
	if len(parts) < 4 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("trade.invalid", lang))
		return
	}
	buyerID, ok := interaction.ParseUserID(parts[len(parts)-1])
	if !ok {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("trade.invalid", lang))
		return
	}
	price, perr := strconv.Atoi(parts[len(parts)-2])
	quantity, qerr := strconv.Atoi(parts[len(parts)-3])
	if perr != nil || qerr != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("trade.invalid", lang))
		return
	}
	itemQuery := strings.Join(parts[:len(parts)-3], " ")

	offer, it, err := c.svc.CreateOffer(userID, buyerID, itemQuery, quantity, price)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, c.errorText(lang, err))
		return
	}
	embed, comps := c.buildOfferMessage(offer, it, lang)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) buildOfferMessage(offer model.TradeOffer, it *items.Item, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := components.Embed(
		i18n.T("trade.offer_title", lang),
		i18n.T("trade.offer_desc", lang, map[string]any{
			"seller":   interaction.Mention(offer.SellerID),
			"buyer":    interaction.Mention(offer.BuyerID),
			"quantity": offer.Quantity,
			"item":     it.LocalizedName(lang),
			"price":    offer.Price,
		}),
		components.ColorReward,
	)
	offerIDStr := strconv.FormatInt(offer.ID, 10)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("trade.btn_accept", lang), components.Encode("trade", "accept", offerIDStr), discordgo.SuccessButton),
			components.Button(i18n.T("trade.btn_decline", lang), components.Encode("trade", "decline", offerIDStr), discordgo.DangerButton),
		),
	}
	return embed, comps
}

func (c *Cog) onAccept(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	offerID, err := strconv.ParseInt(rest[0], 10, 64)
	if err != nil {
		return
	}

	offer, err := c.svc.Accept(offerID, userID)
	if err != nil {
		c.handleTradeError(b, i, lang, err)
		return
	}
	it := items.Get(offer.ItemID)
	itemName := offer.ItemID
	if it != nil {
		itemName = it.LocalizedName(lang)
	}
	embed := components.Embed(
		i18n.T("trade.accepted_title", lang),
		i18n.T("trade.accepted_desc", lang, map[string]any{
			"seller":   interaction.Mention(offer.SellerID),
			"buyer":    interaction.Mention(offer.BuyerID),
			"quantity": offer.Quantity,
			"item":     itemName,
			"price":    offer.Price,
		}),
		components.ColorSuccess,
	)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, []discordgo.MessageComponent{}))
}

func (c *Cog) onDecline(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	offerID, err := strconv.ParseInt(rest[0], 10, 64)
	if err != nil {
		return
	}

	offer, err := c.svc.Cancel(offerID, userID)
	if err != nil {
		c.handleTradeError(b, i, lang, err)
		return
	}
	embed := components.Embed(
		i18n.T("trade.declined_title", lang),
		i18n.T("trade.declined_desc", lang, map[string]any{
			"seller": interaction.Mention(offer.SellerID),
			"buyer":  interaction.Mention(offer.BuyerID),
		}),
		components.ColorMuted,
	)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, []discordgo.MessageComponent{}))
}

func (c *Cog) handleTradeError(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, err error) {
	interaction.RespondError(b, i, lang, c.errorKey(err))
}

func (c *Cog) errorText(lang string, err error) string {
	return i18n.T(c.errorKey(err), lang)
}

func (c *Cog) errorKey(err error) string {
	switch err {
	case tradesvc.ErrSelf:
		return "trade.err_self"
	case tradesvc.ErrAmount:
		return "trade.err_amount"
	case tradesvc.ErrNotTradeable:
		return "trade.err_not_tradeable"
	case store.ErrInsufficientItems:
		return "trade.err_insufficient_items"
	case store.ErrInsufficientFunds:
		return "trade.err_no_money"
	case store.ErrInventoryFull:
		return "trade.err_inventory_full"
	case store.ErrTradeNotFound:
		return "trade.err_not_found"
	case store.ErrTradeNotPending:
		return "trade.err_not_pending"
	case store.ErrTradeWrongUser:
		return "trade.err_wrong_user"
	default:
		return "trade.err_generic"
	}
}
