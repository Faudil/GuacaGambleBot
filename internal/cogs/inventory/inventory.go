package inventory

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
	"guacagamblebot/internal/items"
	invsvc "guacagamblebot/internal/service/inventory"
	mktsvc "guacagamblebot/internal/service/market"
	"guacagamblebot/internal/store"
)

const maxSellOptions = 25

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *invsvc.Service
	mkt   *mktsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: invsvc.New(s, cfg), mkt: mktsvc.New(s, cfg)}
	r.SlashWithOptions("inventory", "cmd.inventory.desc",
		[]*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "Le joueur dont tu veux voir l'inventaire (optionnel)",
				Required:    false,
			},
		},
		c.onSlashMenu)
	r.Prefix("inventory", c.onPrefix)
	r.Prefix("inv", c.onPrefix)
	r.Prefix("bag", c.onPrefix)
	r.Prefix("sac", c.onPrefix)
	r.Component("inventory", "sell", c.onSellButton)
	r.Component("inventory", "pick", c.onPickItem)
	r.Component("inventory", "theme", c.onThemeSelect)
	r.Component("inventory", "nav", c.onNav)
	r.Modal("inventory", "sellqty", c.onSellModal)
}

// themeOrder lists the inventory categories in display order. Any category not
// listed here (and not mapped below) is collected under "other".
var themeOrder = []string{"mining", "fishing", "farming", "archeology", "food", "tools", "materials", "equipment", "special"}

// categoryOf returns the theme a inventory entry belongs to.
func categoryOf(e invsvc.InvEntry) string {
	if e.EquipInfo != nil {
		return "equipment"
	}
	cat := "other"
	if e.Item != nil {
		cat = string(e.Item.Category)
		if cat == "delve" {
			cat = "equipment"
		}
	}
	for _, t := range themeOrder {
		if cat == t {
			return cat
		}
	}
	return "other"
}

// presentThemes returns the non-empty themes of an inventory in display order,
// with uncategorized entries collected under "other".
func presentThemes(entries []invsvc.InvEntry) []string {
	present := make(map[string]bool)
	for _, e := range entries {
		present[categoryOf(e)] = true
	}
	var themes []string
	for _, cat := range themeOrder {
		if present[cat] {
			themes = append(themes, cat)
		}
	}
	if present["other"] {
		themes = append(themes, "other")
	}
	return themes
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	selfID := interaction.ToInt64(interaction.UserID(i))
	targetID := selfID

	opts := i.ApplicationCommandData().Options
	targetStr := ""
	if len(opts) > 0 && opts[0].Value != nil {
		if v, ok := opts[0].Value.(string); ok {
			targetStr = v
		}
	}
	if targetStr != "" {
		targetID = interaction.ToInt64(targetStr)
	}

	title := i.Member.User.Username
	if targetID != selfID {
		resolved := false
		if i.ApplicationCommandData().Resolved != nil && i.ApplicationCommandData().Resolved.Users != nil {
			if u, ok := i.ApplicationCommandData().Resolved.Users[targetStr]; ok {
				title = u.Username
				resolved = true
			}
		}
		if !resolved {
			title = interaction.DisplayName(b.Session, i.GuildID, i.Member, targetID)
		}
	}

	result, err := c.svc.GetInventory(targetID)
	if err != nil {
		interaction.RespondError(b, i, lang, "inventory.error")
		return
	}

	if len(result.Entries) == 0 {
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
				components.Embed("", i18n.T("inventory.empty", lang, map[string]any{"user": interaction.Mention(targetID)}), components.ColorDanger), nil))
		return
	}

	themes := presentThemes(result.Entries)
	embed := c.buildEmbed(lang, result, i18n.T("inventory.title", lang, map[string]any{"user": title}), themes[0], targetID == selfID)
	comps := c.buildComponents(lang, selfID, targetID, themes, themes[0], targetID == selfID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefix(b *interaction.Bot, sess *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	selfID := interaction.ToInt64(m.Author.ID)

	parts := strings.Fields(m.Content)
	targetID, valid := resolveTarget(parts, selfID)
	if !valid {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("inventory.invalid_user", lang))
		return
	}
	title := m.Author.Username
	if targetID != selfID {
		if len(m.Mentions) > 0 {
			title = m.Mentions[0].Username
		} else {
			title = interaction.DisplayName(sess, m.GuildID, &discordgo.Member{User: m.Author}, targetID)
		}
	}

	result, err := c.svc.GetInventory(targetID)
	if err != nil {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("inventory.error", lang))
		return
	}

	if len(result.Entries) == 0 {
		_, _ = sess.ChannelMessageSend(m.ChannelID, i18n.T("inventory.empty", lang, map[string]any{"user": interaction.Mention(targetID)}))
		return
	}

	themes := presentThemes(result.Entries)
	embed := c.buildEmbed(lang, result, i18n.T("inventory.title", lang, map[string]any{"user": title}), themes[0], targetID == selfID)
	comps := c.buildComponents(lang, selfID, targetID, themes, themes[0], targetID == selfID)
	_, _ = sess.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func (c *Cog) buildEmbed(lang string, result *invsvc.InvResult, title, cat string, showSell bool) *discordgo.MessageEmbed {
	embed := components.Embed(title, "", components.ColorInfo)
	embed.Fields = buildCategoryFields(result, cat, lang)
	footer := fmt.Sprintf(" — %d/%d", result.Current, result.Limit)
	if showSell {
		footer = i18n.T("inventory.footer", lang) + footer
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: footer}
	return embed
}

// buildComponents builds the inventory's interactive rows: a theme select menu
// plus prev/page/next navigation and, for the owner's own view, the sell button.
func (c *Cog) buildComponents(lang string, viewerID, targetID int64, themes []string, current string, showSell bool) []discordgo.MessageComponent {
	catOptions := make([]discordgo.SelectMenuOption, 0, len(themes))
	for _, cat := range themes {
		catOptions = append(catOptions, discordgo.SelectMenuOption{
			Label:   i18n.T("inventory.category_"+cat, lang),
			Value:   cat,
			Default: cat == current,
		})
	}
	themeSelect := discordgo.SelectMenu{
		CustomID:    components.EncodeOwner(viewerID, "inventory", "theme", strconv.FormatInt(targetID, 10)),
		Placeholder: i18n.T("inventory.nav_theme_placeholder", lang),
		Options:     catOptions,
	}

	page := indexOf(themes, current) + 1
	total := len(themes)
	targetStr := strconv.FormatInt(targetID, 10)

	prevBtn := discordgo.Button{
		Label:    i18n.T("inventory.nav_prev", lang),
		CustomID: components.EncodeOwner(viewerID, "inventory", "nav", targetStr, "prev", fmt.Sprintf("%d", page)),
		Style:    discordgo.SecondaryButton,
		Disabled: page <= 1,
	}
	pageBtn := discordgo.Button{
		Label:    i18n.T("inventory.nav_page", lang, map[string]any{"page": page, "total": total}),
		CustomID: "_disabled",
		Style:    discordgo.SecondaryButton,
		Disabled: true,
	}
	nextBtn := discordgo.Button{
		Label:    i18n.T("inventory.nav_next", lang),
		CustomID: components.EncodeOwner(viewerID, "inventory", "nav", targetStr, "next", fmt.Sprintf("%d", page)),
		Style:    discordgo.SecondaryButton,
		Disabled: page >= total,
	}

	rows := []discordgo.MessageComponent{components.ActionRow(themeSelect)}
	navRow := []discordgo.MessageComponent{prevBtn, pageBtn, nextBtn}
	if showSell {
		navRow = append(navRow, components.Button(i18n.T("inventory.sell_button", lang),
			components.EncodeOwner(viewerID, "inventory", "sell"), discordgo.SecondaryButton))
	}
	return append(rows, components.ActionRow(navRow...))
}

func indexOf(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}

// updateInventoryMessage edits the inventory message to show the given theme.
func (c *Cog) updateInventoryMessage(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, viewerID, targetID int64, result *invsvc.InvResult, themes []string, cat string) {
	if !contains(themes, cat) {
		cat = themes[0]
	}
	userLabel := interaction.DisplayName(b.Session, i.GuildID, i.Member, targetID)
	embed := c.buildEmbed(lang, result, i18n.T("inventory.title", lang, map[string]any{"user": userLabel}), cat, targetID == viewerID)
	comps := c.buildComponents(lang, viewerID, targetID, themes, cat, targetID == viewerID)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: comps,
		},
	})
}

// onThemeSelect jumps directly to the theme picked in the select menu.
func (c *Cog) onThemeSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	viewerID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 1 {
		return
	}
	targetID, err := strconv.ParseInt(rest[0], 10, 64)
	if err != nil {
		return
	}
	cat := i.MessageComponentData().Values[0]

	result, err := c.svc.GetInventory(targetID)
	if err != nil {
		interaction.RespondError(b, i, lang, "inventory.error")
		return
	}
	themes := presentThemes(result.Entries)
	if !contains(themes, cat) {
		return
	}
	c.updateInventoryMessage(b, i, lang, viewerID, targetID, result, themes, cat)
}

// onNav handles prev/next pagination between the inventory themes.
func (c *Cog) onNav(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	viewerID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 3 {
		return
	}
	targetID, err := strconv.ParseInt(rest[0], 10, 64)
	if err != nil {
		return
	}
	action := rest[1]
	curPage, _ := strconv.Atoi(rest[2])

	result, err := c.svc.GetInventory(targetID)
	if err != nil {
		interaction.RespondError(b, i, lang, "inventory.error")
		return
	}
	themes := presentThemes(result.Entries)
	if len(themes) == 0 {
		return
	}

	page := curPage
	switch action {
	case "prev":
		page--
	case "next":
		page++
	}
	if page < 1 {
		page = 1
	}
	if page > len(themes) {
		page = len(themes)
	}
	c.updateInventoryMessage(b, i, lang, viewerID, targetID, result, themes, themes[page-1])
}

// resolveTarget returns the requested user id from a command's arguments.
// With no argument it falls back to the caller. The second return value is
// false when an argument was given but could not be parsed as a user.
func resolveTarget(args []string, selfID int64) (int64, bool) {
	if len(args) < 2 {
		return selfID, true
	}
	id, ok := interaction.ParseUserID(args[1])
	if !ok {
		return 0, false
	}
	return id, true
}

func (c *Cog) onSellButton(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	result, err := c.svc.GetInventory(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "inventory.error")
		return
	}

	var sellables []invsvc.InvEntry
	for _, e := range result.Entries {
		if e.EquipInfo == nil && e.Item != nil && e.Item.IsSellable() {
			sellables = append(sellables, e)
		}
	}

	if len(sellables) == 0 {
		interaction.RespondError(b, i, lang, "inventory.sell_none")
		return
	}

	ids := make([]string, 0, len(sellables))
	for _, e := range sellables {
		ids = append(ids, e.Item.ID)
	}
	prices := c.mkt.SellPricesFor(ids)

	options := make([]discordgo.SelectMenuOption, 0, len(sellables))
	for _, e := range sellables {
		if len(options) >= maxSellOptions {
			break
		}
		label := fmt.Sprintf("%s x%d — $%d", items.LocalizedName(e.Item.Name, lang), e.Quantity, prices[e.Item.ID])
		if len(label) > 100 {
			label = label[:97] + "..."
		}
		options = append(options, discordgo.SelectMenuOption{
			Label: label,
			Value: e.Item.ID,
			Emoji: &discordgo.ComponentEmoji{Name: e.Item.Emoji},
		})
	}

	selectMenu := discordgo.SelectMenu{
		CustomID:    components.EncodeOwner(result.UserID, "inventory", "pick"),
		Placeholder: i18n.T("inventory.sell_placeholder", lang),
		Options:     options,
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
			components.Embed("", i18n.T("inventory.sell_choose", lang), components.ColorInfo),
			[]discordgo.MessageComponent{components.ActionRow(selectMenu)}))
}

func (c *Cog) onPickItem(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	itemID := i.MessageComponentData().Values[0]

	it := items.Get(itemID)
	if it == nil {
		interaction.RespondError(b, i, lang, "inventory.error")
		return
	}

	modal := components.ModalResponse(
		components.EncodeOwner(userID, "inventory", "sellqty", itemID),
		i18n.T("inventory.sell_modal_title", lang, map[string]any{"item": items.LocalizedName(it.Name, lang)}),
		components.TextInput("amount",
			i18n.T("inventory.sell_amount_label", lang), true, "1",
			discordgo.TextInputShort, 1, 5),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onSellModal(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	ownerID, ok := components.OwnerID(i.ModalSubmitData().CustomID)
	if !ok || !interaction.NotYourMenu(b, i, lang, ownerID) {
		return
	}
	_, _, rest := components.Decode(i.ModalSubmitData().CustomID)

	if len(rest) < 1 {
		interaction.RespondError(b, i, lang, "inventory.error")
		return
	}
	itemID := rest[0]
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

	gain, leveled, newLevel, err := c.mkt.SellItem(userID, itemID, amount)
	if err != nil {
		switch err {
		case mktsvc.ErrNotSellable:
			interaction.RespondError(b, i, lang, "market.item_not_sellable")
		case mktsvc.ErrNotFound:
			interaction.RespondError(b, i, lang, "market.item_not_found")
		case mktsvc.ErrNoItem:
			interaction.RespondError(b, i, lang, "market.no_item")
		default:
			interaction.RespondError(b, i, lang, "inventory.error")
		}
		return
	}

	content := i18n.T("market.sold_msg", lang, map[string]any{
		"amount": amount, "item": items.LocalizedName(it.Name, lang), "gain": gain,
	})
	if leveled {
		content += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": newLevel})
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})

	if n, ok := c.store.PopQuestNotification(userID); ok {
		interaction.SendQuestNotification(b, i, n, lang)
	}
	if unlocks, err := achievement.CheckAndUnlock(b.DB, userID); err == nil && len(unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, unlocks)
	}
}

// buildCategoryFields renders a single theme (category) of the inventory as one
// embed field, using the same entry formatting as before.
func buildCategoryFields(res *invsvc.InvResult, cat, lang string) []*discordgo.MessageEmbedField {
	val := ""
	for _, e := range res.Entries {
		if categoryOf(e) != cat {
			continue
		}
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
			val += fmt.Sprintf("%s **%s** : `x%d`\n", emoji, items.LocalizedName(e.ItemName, lang), e.Quantity)
		}
	}
	if val == "" {
		return nil
	}
	return []*discordgo.MessageEmbedField{components.Field(i18n.T("inventory.category_"+cat, lang), val, false)}
}
