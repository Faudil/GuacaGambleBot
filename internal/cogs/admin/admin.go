package admin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	adminsvc "guacagamblebot/internal/service/admin"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *adminsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: adminsvc.New(s, cfg)}
	r.Slash("admin", "Panneau d'administration.", c.onSlashMenu)
	r.Prefix("airdrop", c.onAirdrop)
	r.Prefix("rain", c.onAirdrop)
	r.Prefix("airdrop_item", c.onAirdropItem)
	r.Prefix("rain_item", c.onAirdropItem)
	r.Prefix("givecrowns", c.onGiveCrowns)
	r.Prefix("addcrowns", c.onGiveCrowns)
	r.Prefix("setlang", c.onSetLang)
	r.Prefix("reseteconomy", c.onResetEconomy)
	r.Component("admin", "airdrop", c.onAirdropBtn)
	r.Component("admin", "givecrowns", c.onGiveCrownsBtn)
	r.Component("admin", "reseteconomy", c.onResetEconomyBtn)
	r.Component("admin", "setlang", c.onSetLangBtn)
	r.Component("admin", "airdrop_item", c.onAirdropItemBtn)
	r.Modal("admin", "airdrop_submit", c.onAirdropSubmit)
	r.Modal("admin", "givecrowns_submit", c.onGiveCrownsSubmit)
	r.Modal("admin", "airdrop_item_submit", c.onAirdropItemSubmit)
}

func isAdmin(i *discordgo.InteractionCreate) bool {
	if i.Member != nil && i.Member.Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}
	return false
}

// isAdminMsg reports whether the author of a prefix message is an administrator,
// using the gateway state when the message member payload lacks permissions.
func isAdminMsg(s *discordgo.Session, m *discordgo.Message) bool {
	if m.Member != nil && m.Member.Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}
	if perms, err := s.State.MessagePermissions(m); err == nil {
		return perms&discordgo.PermissionAdministrator != 0
	}
	return false
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if !isAdmin(i) {
		interaction.RespondError(b, i, lang, "admin.no_permission_money")
		return
	}
	embed, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	embed, comps := c.menu(lang)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) menu(lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := components.Embed(
		i18n.T("admin.menu_title", lang),
		i18n.T("admin.menu_desc", lang),
		0xe74c3c,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("admin.btn_airdrop", lang), components.Encode("admin", "airdrop"), discordgo.DangerButton),
			components.Button(i18n.T("admin.btn_givecrowns", lang), components.Encode("admin", "givecrowns"), discordgo.PrimaryButton),
		),
		components.ActionRow(
			components.Button(i18n.T("admin.btn_reseteconomy", lang), components.Encode("admin", "reseteconomy"), discordgo.DangerButton),
			components.Button(i18n.T("admin.btn_setlang", lang), components.Encode("admin", "setlang"), discordgo.SecondaryButton),
			components.Button(i18n.T("admin.btn_airdrop_item", lang), components.Encode("admin", "airdrop_item"), discordgo.SecondaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onAirdropBtn(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if !isAdmin(i) {
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: components.ModalResponse(
			components.Encode("admin", "airdrop_submit"),
			i18n.T("admin.airdrop_modal_title", lang),
			components.TextInput("user_id", i18n.T("admin.user_id_label", lang), true, "123456789", discordgo.TextInputShort, 1, 50),
			components.TextInput("amount", i18n.T("admin.amount_label", lang), true, "100", discordgo.TextInputShort, 1, 12),
		),
	})
}

func (c *Cog) onAirdropSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if !isAdmin(i) {
		return
	}
	values := interaction.ModalValues(i)
	userID, ok := interaction.ParseUserID(values["user_id"])
	if !ok {
		interaction.RespondError(b, i, lang, "admin.amount_positive")
		return
	}
	amount, err := strconv.Atoi(strings.TrimSpace(values["amount"]))
	if err != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "admin.amount_positive")
		return
	}
	if _, err := c.svc.GiveMoney(userID, amount); err != nil {
		interaction.RespondError(b, i, lang, "admin.amount_positive")
		return
	}
	embed := components.Embed(
		i18n.T("admin.airdrop_title", lang),
		i18n.T("admin.airdrop_desc", lang, map[string]any{"amount": amount, "user": interaction.Mention(userID)}),
		0xf1c40f,
	)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("admin.gifted_by", lang, map[string]any{"author": interaction.Mention(interaction.ToInt64(interaction.UserID(i)))}),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onAirdropItemBtn(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if !isAdmin(i) {
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: components.ModalResponse(
			components.Encode("admin", "airdrop_item_submit"),
			i18n.T("admin.airdrop_item_modal_title", lang),
			components.TextInput("user_id", i18n.T("admin.user_id_label", lang), true, "123456789", discordgo.TextInputShort, 1, 50),
			components.TextInput("item", i18n.T("admin.item_label", lang), true, "coal", discordgo.TextInputShort, 1, 100),
			components.TextInput("quantity", i18n.T("admin.quantity_label", lang), true, "1", discordgo.TextInputShort, 1, 12),
		),
	})
}

func (c *Cog) onAirdropItemSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if !isAdmin(i) {
		return
	}
	values := interaction.ModalValues(i)
	item := c.resolveItem(values["item"])
	if item == nil {
		interaction.RespondError(b, i, lang, "admin.item_not_found")
		return
	}
	quantity, err := strconv.Atoi(strings.TrimSpace(values["quantity"]))
	if err != nil || quantity <= 0 {
		interaction.RespondError(b, i, lang, "admin.quantity_positive")
		return
	}
	author := interaction.Mention(interaction.ToInt64(interaction.UserID(i)))
	if all, _, ok := c.parseTarget(values["user_id"]); ok && all {
		rows, err := c.svc.AirdropItemAll(item.ID, quantity)
		if err != nil {
			interaction.RespondError(b, i, lang, "admin.item_not_found")
			return
		}
		embed := c.itemAirdropEmbed(lang, item, quantity, true, 0, rows, author)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
		return
	}
	userID, ok := interaction.ParseUserID(values["user_id"])
	if !ok {
		interaction.RespondError(b, i, lang, "admin.item_not_found")
		return
	}
	if err := c.svc.GiveItem(userID, item.ID, quantity); err != nil {
		interaction.RespondError(b, i, lang, "admin.item_not_found")
		return
	}
	embed := c.itemAirdropEmbed(lang, item, quantity, false, userID, 0, author)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onAirdropItem(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	if !isAdminMsg(s, m) {
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	parts := strings.Fields(m.Content)
	if len(parts) < 3 {
		return
	}
	all, userID, ok := c.parseTarget(parts[1])
	if !ok {
		return
	}
	quantity := 1
	itemName := strings.Join(parts[2:], " ")
	if last, err := strconv.Atoi(parts[len(parts)-1]); err == nil && last > 0 {
		quantity = last
		itemName = strings.Join(parts[2:len(parts)-1], " ")
	}
	item := c.resolveItem(itemName)
	if item == nil {
		return
	}
	author := m.Author.Username
	if all {
		rows, err := c.svc.AirdropItemAll(item.ID, quantity)
		if err != nil {
			return
		}
		_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{c.itemAirdropEmbed(lang, item, quantity, true, 0, rows, author)},
		})
		return
	}
	if err := c.svc.GiveItem(userID, item.ID, quantity); err != nil {
		return
	}
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{c.itemAirdropEmbed(lang, item, quantity, false, userID, 0, author)},
	})
}

// parseTarget interprets the target argument of an item airdrop: "all",
// "@everyone" or "@here" select every player, anything else is parsed as a
// user mention or raw user id.
func (c *Cog) parseTarget(s string) (all bool, userID int64, ok bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "all", "@everyone", "@here":
		return true, 0, true
	}
	uid, ok := interaction.ParseUserID(s)
	return false, uid, ok
}

// resolveItem finds an item by exact id or name, falling back to a
// case-insensitive name match.
func (c *Cog) resolveItem(name string) *items.Item {
	if it := items.Get(name); it != nil {
		return it
	}
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, it := range items.AllItems() {
		if strings.ToLower(it.Name) == lower {
			return items.Get(it.ID)
		}
	}
	return nil
}

// itemAirdropEmbed builds the confirmation embed for an item airdrop, for a
// single recipient or for every player.
func (c *Cog) itemAirdropEmbed(lang string, it *items.Item, quantity int, all bool, userID int64, rows int, author string) *discordgo.MessageEmbed {
	var title, desc string
	if all {
		title = i18n.T("admin.airdrop_item_all_title", lang)
		desc = i18n.T("admin.airdrop_item_all_desc", lang, map[string]any{
			"quantity":  quantity,
			"item_name": it.Name,
			"rows":      rows,
		})
	} else {
		name := it.Name
		if quantity > 1 {
			name = fmt.Sprintf("%dx %s", quantity, it.Name)
		}
		title = i18n.T("admin.airdrop_item_title", lang)
		desc = i18n.T("admin.airdrop_item_desc", lang, map[string]any{
			"item_name": name,
			"user":      interaction.Mention(userID),
		})
	}
	embed := components.Embed(title, desc, 0xf1c40f)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("admin.gifted_by", lang, map[string]any{"author": author}),
	}
	return embed
}

func (c *Cog) onGiveCrownsBtn(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if !isAdmin(i) {
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: components.ModalResponse(
			components.Encode("admin", "givecrowns_submit"),
			i18n.T("admin.givecrowns_modal_title", lang),
			components.TextInput("user_id", i18n.T("admin.user_id_label", lang), true, "123456789", discordgo.TextInputShort, 1, 50),
			components.TextInput("amount", i18n.T("admin.amount_label", lang), true, "10", discordgo.TextInputShort, 1, 12),
		),
	})
}

func (c *Cog) onGiveCrownsSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if !isAdmin(i) {
		return
	}
	values := interaction.ModalValues(i)
	userID, ok := interaction.ParseUserID(values["user_id"])
	if !ok {
		interaction.RespondError(b, i, lang, "admin.amount_positive")
		return
	}
	amount, err := strconv.Atoi(strings.TrimSpace(values["amount"]))
	if err != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "admin.amount_positive")
		return
	}
	if err := c.svc.GiveCrowns(userID, amount); err != nil {
		interaction.RespondError(b, i, lang, "admin.amount_positive")
		return
	}
	embed := components.Embed("✅",
		fmt.Sprintf("Added `%d` 👑 Crowns to %s.", amount, interaction.Mention(userID)),
		0xf1c40f)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onResetEconomyBtn(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if !isAdmin(i) {
		return
	}
	if err := c.svc.ResetEconomy(); err != nil {
		interaction.RespondError(b, i, lang, "admin.amount_positive")
		return
	}
	embed := components.Embed("🔄", "Economy reset complete!", 0xe74c3c)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onSetLangBtn(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if !isAdmin(i) {
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: components.ModalResponse(
			components.Encode("admin", "setlang_submit"),
			i18n.T("admin.setlang_modal_title", lang),
			components.TextInput("lang", i18n.T("admin.lang_label", lang), true, "en", discordgo.TextInputShort, 2, 2),
		),
	})
}

func (c *Cog) onSetLang(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	if !isAdminMsg(s, m) {
		return
	}
	parts := strings.Fields(m.Content)
	if len(parts) < 2 {
		return
	}
	lang := parts[1]
	if lang != "fr" && lang != "en" {
		return
	}
	serverID := interaction.ToInt64(m.GuildID)
	if err := c.svc.SetLanguage(serverID, lang); err != nil {
		return
	}
	_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("admin.lang_set", lang))
}

func (c *Cog) onAirdrop(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	if !isAdminMsg(s, m) {
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	parts := strings.Fields(m.Content)
	if len(parts) < 3 {
		return
	}
	userID, ok := interaction.ParseUserID(parts[1])
	if !ok {
		return
	}
	amount, err := strconv.Atoi(parts[2])
	if err != nil || amount <= 0 {
		return
	}
	if _, err := c.svc.GiveMoney(userID, amount); err != nil {
		return
	}
	embed := components.Embed(
		i18n.T("admin.airdrop_title", lang),
		i18n.T("admin.airdrop_desc", lang, map[string]any{"amount": amount, "user": interaction.Mention(userID)}),
		0xf1c40f,
	)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("admin.gifted_by", lang, map[string]any{"author": m.Author.Username}),
	}
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}})
}

func (c *Cog) onGiveCrowns(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	if !isAdminMsg(s, m) {
		return
	}
	parts := strings.Fields(m.Content)
	if len(parts) < 3 {
		return
	}
	userID, ok := interaction.ParseUserID(parts[1])
	if !ok {
		return
	}
	amount, err := strconv.Atoi(parts[2])
	if err != nil || amount <= 0 {
		return
	}
	if err := c.svc.GiveCrowns(userID, amount); err != nil {
		return
	}
	_, _ = s.ChannelMessageSend(m.ChannelID,
		fmt.Sprintf("✅ Added `%d` 👑 Crowns to %s.", amount, interaction.Mention(userID)))
}

func (c *Cog) onResetEconomy(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	if !isAdminMsg(s, m) {
		return
	}
	if err := c.svc.ResetEconomy(); err != nil {
		return
	}
	_, _ = s.ChannelMessageSend(m.ChannelID, "🔄 Economy has been reset.")
}
