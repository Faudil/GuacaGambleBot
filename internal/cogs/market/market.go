package market

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	mktsvc "guacagamblebot/internal/service/market"
	questssvc "guacagamblebot/internal/service/quests"
	jsvc "guacagamblebot/internal/service/journal"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *mktsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: mktsvc.New(s, cfg)}
	r.Slash("market", "Marché : voir les prix et échanger des objets.", c.onSlashMenu)
	r.Prefix("market", c.onPrefix)
	r.Prefix("market_sell", c.onSellPrefix)
	r.Prefix("ms", c.onSellPrefix)
	r.Prefix("m_s", c.onSellPrefix)
	r.Component("market", "select", c.onSelectItem)
	r.Component("market", "filter", c.onFilter)
	r.Component("market", "nav", c.onNav)
	r.Component("market", "action", c.onActionChoice)
	r.Modal("market", "order", c.onOrderModal)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	c.sendMarketMessage(b, i, lang, userID, "all", 1, false)
}

func (c *Cog) onPrefix(b *interaction.Bot, sess *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)

	parts := strings.Fields(m.Content)
	category := "all"
	if len(parts) > 1 {
		cat := strings.ToLower(parts[1])
		switch cat {
		case "mine", "mining":
			category = "mining"
		case "fish", "fishing":
			category = "fishing"
		case "farm", "farming":
			category = "farming"
		case "arch", "archeology":
			category = "archeology"
		case "tools":
			category = "tools"
		case "food":
			category = "food"
		}
	}

	c.sendMarketPrefix(b, sess, m, lang, userID, category, 1)
}

func (c *Cog) sendMarketMessage(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, userID int64, category string, page int, edit bool) {
	pageSize := mktsvc.ItemsPerPage
	if page < 1 {
		page = 1
	}

	views, total, err := c.svc.GetMarket(category, page, pageSize)
	if err != nil {
		interaction.RespondError(b, i, lang, "market.error")
		return
	}

	totalPages := max(1, int(math.Ceil(float64(total)/float64(pageSize))))
	bal, _ := c.store.GetBalance(userID)
	weekID := currentWeekID()

	embed := c.buildMarketEmbed(views, lang, weekID, category, page, totalPages, bal)
	comps := c.buildMarketComponents(views, total, category, page, totalPages, lang)

	if edit {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: comps,
			},
		})
	} else {
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
	}
}

func (c *Cog) sendMarketPrefix(b *interaction.Bot, sess *discordgo.Session, m *discordgo.Message, lang string, userID int64, category string, page int) {
	pageSize := mktsvc.ItemsPerPage
	if page < 1 {
		page = 1
	}

	views, total, err := c.svc.GetMarket(category, page, pageSize)
	if err != nil {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("market.error", lang))
		return
	}

	totalPages := max(1, int(math.Ceil(float64(total)/float64(pageSize))))
	bal, _ := c.store.GetBalance(userID)
	weekID := currentWeekID()

	embed := c.buildMarketEmbed(views, lang, weekID, category, page, totalPages, bal)
	comps := c.buildMarketComponents(views, total, category, page, totalPages, lang)

	_, _ = sess.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) buildMarketEmbed(views []mktsvc.MarketItemView, lang, weekID, category string, page, totalPages, balance int) *discordgo.MessageEmbed {
	catLabel := i18n.T("market.cat_all", lang)
	if category != "" && category != "all" {
		catLabel = i18n.T("market.cat_"+category, lang)
	}
	desc := i18n.T("market.greeting", lang, map[string]any{"week": weekID, "filter": catLabel})
	embed := components.Embed(i18n.T("market.title", lang), desc, 0xf1c40f)

	for _, v := range views {
		var trendStr string
		if v.TrendPercent > 0 {
			trendStr = i18n.T("market.trend_up", lang, map[string]any{"pct": v.TrendPercent})
		} else if v.TrendPercent < 0 {
			trendStr = i18n.T("market.trend_down", lang, map[string]any{"pct": -v.TrendPercent})
		} else {
			trendStr = i18n.T("market.trend_stable", lang)
		}

		name := fmt.Sprintf("%s %s", v.Item.Emoji, c.displayName(v.Item.Name, lang))
		embed.Fields = append(embed.Fields, components.Field(
			name,
			i18n.T("market.item_line", lang, map[string]any{"price": v.CurrentPrice, "trend": trendStr}),
			true,
		))
	}

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("market.footer", lang, map[string]any{"page": page, "total": totalPages, "balance": balance}),
	}
	return embed
}

func (c *Cog) buildMarketComponents(views []mktsvc.MarketItemView, total int, category string, page, totalPages int, lang string) []discordgo.MessageComponent {
	catOptions := []discordgo.SelectMenuOption{
		{Label: i18n.T("market.cat_all", lang), Value: "all", Emoji: &discordgo.ComponentEmoji{Name: "📦"}},
		{Label: i18n.T("market.cat_mining", lang), Value: "mining", Emoji: &discordgo.ComponentEmoji{Name: "⛏️"}},
		{Label: i18n.T("market.cat_fishing", lang), Value: "fishing", Emoji: &discordgo.ComponentEmoji{Name: "🎣"}},
		{Label: i18n.T("market.cat_farming", lang), Value: "farming", Emoji: &discordgo.ComponentEmoji{Name: "🌾"}},
		{Label: i18n.T("market.cat_archeology", lang), Value: "archeology", Emoji: &discordgo.ComponentEmoji{Name: "🦴"}},
		{Label: i18n.T("market.cat_tools", lang), Value: "tools", Emoji: &discordgo.ComponentEmoji{Name: "🔧"}},
		{Label: i18n.T("market.cat_food", lang), Value: "food", Emoji: &discordgo.ComponentEmoji{Name: "🍪"}},
	}

	filter := discordgo.SelectMenu{
		CustomID:    components.Encode("market", "filter", fmt.Sprintf("%d", page)),
		Placeholder: i18n.T("market.filter_placeholder", lang),
		Options:     catOptions,
	}

	var itemOptions []discordgo.SelectMenuOption
	for _, v := range views {
		priceStr := i18n.T("market.item_price", lang, map[string]any{"price": v.CurrentPrice})
		label := fmt.Sprintf("%s — %s", v.Item.Name, priceStr)
		if len(label) > 100 {
			label = label[:97] + "..."
		}
		var desc string
		if v.TrendPercent > 0 {
			desc = i18n.T("market.trend_desc_up", lang, map[string]any{"pct": v.TrendPercent})
		} else if v.TrendPercent < 0 {
			desc = i18n.T("market.trend_desc_down", lang, map[string]any{"pct": -v.TrendPercent})
		} else {
			desc = i18n.T("market.trend_desc_stable", lang)
		}
		if len(desc) > 100 {
			desc = desc[:97] + "..."
		}
		itemOptions = append(itemOptions, discordgo.SelectMenuOption{
			Label:       label,
			Value:       v.Item.ID,
			Emoji:       &discordgo.ComponentEmoji{Name: v.Item.Emoji},
			Description: desc,
		})
	}

	if len(itemOptions) == 0 {
		itemOptions = append(itemOptions, discordgo.SelectMenuOption{
			Label:       i18n.T("market.empty_label", lang),
			Value:       "_none",
			Description: i18n.T("market.empty_desc", lang),
			Default:     true,
		})
	}

	itemSelect := discordgo.SelectMenu{
		CustomID:    components.Encode("market", "select", fmt.Sprintf("%d", page), category),
		Placeholder: i18n.T("market.select_placeholder", lang),
		Options:     itemOptions,
	}

	prevBtn := discordgo.Button{
		Label:    i18n.T("market.nav_prev", lang),
		CustomID: components.Encode("market", "nav", "prev", fmt.Sprintf("%d", page), category),
		Style:    discordgo.SecondaryButton,
		Disabled: page <= 1,
	}

	pageBtn := discordgo.Button{
		Label:    i18n.T("market.nav_page", lang, map[string]any{"page": page, "total": totalPages}),
		CustomID: "_disabled",
		Style:    discordgo.SecondaryButton,
		Disabled: true,
	}

	refreshBtn := discordgo.Button{
		Label:    i18n.T("market.nav_refresh", lang),
		CustomID: components.Encode("market", "nav", "refresh", fmt.Sprintf("%d", page), category),
		Style:    discordgo.PrimaryButton,
	}

	nextBtn := discordgo.Button{
		Label:    i18n.T("market.nav_next", lang),
		CustomID: components.Encode("market", "nav", "next", fmt.Sprintf("%d", page), category),
		Style:    discordgo.SecondaryButton,
		Disabled: page >= totalPages,
	}

	return []discordgo.MessageComponent{
		components.ActionRow(filter),
		components.ActionRow(itemSelect),
		components.ActionRow(prevBtn, pageBtn, refreshBtn, nextBtn),
	}
}

func (c *Cog) onFilter(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	category := i.MessageComponentData().Values[0]
	c.sendMarketMessage(b, i, lang, userID, category, 1, true)
}

func (c *Cog) onSelectItem(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	itemID := i.MessageComponentData().Values[0]

	if itemID == "_none" {
		interaction.RespondError(b, i, lang, "market.no_items")
		return
	}

	it := items.Get(itemID)
	if it == nil {
		interaction.RespondError(b, i, lang, "market.item_not_found")
		return
	}

	price := it.Price
	var st model.MarketState
	if err := c.store.DB.Where("item_id = ?", itemID).First(&st).Error; err == nil && st.CurrentPrice > 0 {
		price = st.CurrentPrice
	}

	embed := components.Embed(
		i18n.T("market.modal_title", lang, map[string]any{"item": c.displayName(it.Name, lang)}),
		i18n.T("market.action_confirm", lang, map[string]any{"price": price}),
		0xf1c40f,
	)

	buyBtn := components.Button(i18n.T("market.action_buy", lang),
		components.Encode("market", "action", "buy", itemID), discordgo.PrimaryButton)
	sellBtn := components.Button(i18n.T("market.action_sell", lang),
		components.Encode("market", "action", "sell", itemID), discordgo.DangerButton)

	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{components.ActionRow(buyBtn, sellBtn)},
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) onActionChoice(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)

	if len(rest) < 2 {
		interaction.RespondError(b, i, lang, "market.error")
		return
	}
	action, itemID := rest[0], rest[1]
	if action != "buy" && action != "sell" {
		interaction.RespondError(b, i, lang, "market.error")
		return
	}

	it := items.Get(itemID)
	if it == nil {
		interaction.RespondError(b, i, lang, "market.item_not_found")
		return
	}

	modal := components.ModalResponse(
		components.Encode("market", "order", action, itemID),
		i18n.T("market.modal_title", lang, map[string]any{"item": c.displayName(it.Name, lang)}),
		components.TextInput("amount",
			i18n.T("market.modal_amount_label", lang), true, "1",
			discordgo.TextInputShort, 1, 5),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onNav(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)

	if len(rest) < 3 {
		return
	}
	action := rest[0]
	curPage, _ := strconv.Atoi(rest[1])
	category := rest[2]

	// Re-fetch to get accurate total
	views, total, _ := c.svc.GetMarket(category, curPage, mktsvc.ItemsPerPage)
	_ = views
	totalPages := max(1, int(math.Ceil(float64(total)/float64(mktsvc.ItemsPerPage))))

	newPage := curPage
	switch action {
	case "prev":
		newPage = curPage - 1
	case "next":
		newPage = curPage + 1
	case "refresh":
		newPage = curPage
	}
	newPage = max(1, min(newPage, totalPages))

	c.sendMarketMessage(b, i, lang, userID, category, newPage, true)
}

func (c *Cog) onOrderModal(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.ModalSubmitData().CustomID)

	if len(rest) < 2 {
		interaction.RespondError(b, i, lang, "market.error")
		return
	}
	action, itemID := rest[0], rest[1]
	values := interaction.ModalValues(i)
	amountStr := strings.TrimSpace(values["amount"])
	amount, err := strconv.Atoi(amountStr)
	if err != nil || amount <= 0 || amount > 999 {
		interaction.RespondError(b, i, lang, "market.invalid_amount")
		return
	}

	it := items.Get(itemID)
	if it == nil {
		interaction.RespondError(b, i, lang, "market.item_not_found")
		return
	}

	switch action {
	case "buy":
		cost, leveled, newLevel, err := c.svc.BuyItem(userID, itemID, amount)
		if err != nil {
			c.handleOrderError(b, i, lang, err)
			return
		}
		content := i18n.T("market.bought_msg", lang, map[string]any{
			"amount": amount, "item": c.displayName(it.Name, lang), "cost": cost,
		})
		if leveled {
			content += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": newLevel})
		}
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})

	case "sell":
		gain, leveled, newLevel, err := c.svc.SellItem(userID, itemID, amount)
		if err != nil {
			c.handleOrderError(b, i, lang, err)
			return
		}
		content := i18n.T("market.sold_msg", lang, map[string]any{
			"amount": amount, "item": c.displayName(it.Name, lang), "gain": gain,
		})
		if leveled {
			content += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": newLevel})
		}
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if n, ok := c.store.PopQuestNotification(userID); ok {
			interaction.SendQuestNotification(b, i, n, lang)
		}

		if text, dm := jsvc.SceneLine(c.store, userID, "market", lang); text != "" {
			interaction.SendJournalScene(b, i, text, dm)
		}
	default:
		interaction.RespondError(b, i, lang, "market.error")
		return
	}

	if unlocks, err := achievement.CheckAndUnlock(b.DB, userID); err == nil && len(unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, unlocks)
	}
}

func (c *Cog) handleOrderError(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, err error) {
	switch err {
	case mktsvc.ErrNotActive:
		interaction.RespondError(b, i, lang, "market.item_not_sellable")
	case mktsvc.ErrNotFound:
		interaction.RespondError(b, i, lang, "market.item_not_found")
	case mktsvc.ErrNoItem:
		interaction.RespondError(b, i, lang, "market.no_item")
	case mktsvc.ErrNoMoney:
		interaction.RespondError(b, i, lang, "market.no_money")
	case mktsvc.ErrInvalidQty:
		interaction.RespondError(b, i, lang, "market.invalid_amount")
	default:
		interaction.RespondError(b, i, lang, "market.error")
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
	it := items.Get(itemName)
	if it == nil {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("market.item_not_found", lang))
		return
	}

	amount := 1
	if len(parts) > 2 {
		if a, err := strconv.Atoi(parts[2]); err == nil && a > 0 {
			amount = a
		}
	}

	gain, leveled, newLevel, err := c.svc.SellItem(userID, it.ID, amount)
	if err != nil {
		switch err {
		case mktsvc.ErrNotFound:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("market.item_not_found", lang))
		case mktsvc.ErrNotActive:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("market.item_not_sellable", lang))
		case mktsvc.ErrNoItem:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("market.no_item", lang, map[string]any{"item": c.displayName(it.Name, lang), "amount": amount}))
		case mktsvc.ErrInvalidQty:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("market.invalid_amount", lang))
		default:
			_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("market.error", lang))
		}
		return
	}
	sellMsg := i18n.T("market.sold_msg", lang, map[string]any{
		"amount": amount, "item": c.displayName(it.Name, lang), "gain": gain,
	})
	if leveled {
		sellMsg += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": newLevel})
	}
	if n, ok := c.store.PopQuestNotification(userID); ok {
		sellMsg += "\n\n" + questssvc.QuestNotificationMsg(n, lang)
	}

	if text, dm := jsvc.SceneLine(c.store, userID, "market", lang); text != "" {
		interaction.SendJournalSceneMsg(sess, m.ChannelID, m.Author.ID, text, dm)
	}
	_, _ = sess.ChannelMessageSend(m.ChannelID, sellMsg)

	if unlocks, err := achievement.CheckAndUnlock(b.DB, userID); err == nil && len(unlocks) > 0 {
		interaction.SendAchievements(b, nil, lang, unlocks)
	}
}

func (c *Cog) displayName(name, lang string) string {
	return items.DisplayName(name)
}

func currentWeekID() string {
	y, w := time.Now().ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}






