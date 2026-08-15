package bank

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	banksvc "guacagamblebot/internal/service/bank"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
)

// Cog implements the Bank "embed interface": a menu with deposit, withdraw and
// balance actions.
type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *banksvc.Service
}

// Register wires the cog into the router under both slash and prefix triggers.
func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: banksvc.New(s, cfg)}
	r.Slash("bank", "Banque : dépôt, retrait, solde.", c.onSlashMenu)
	r.Slash("bankbal", "Voir ton solde bancaire.", c.onSlashBalance)
	r.SlashWithOptions("deposit", "Déposer de l'argent dans ta banque.",
		[]*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Le montant à déposer.", Required: true},
		}, c.onSlashDeposit)
	r.SlashWithOptions("withdraw", "Retirer de l'argent de ta banque.",
		[]*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Le montant à retirer.", Required: true},
		}, c.onSlashWithdraw)
	r.Prefix("bank", c.onPrefixMenu)
	r.Prefix("b", c.onPrefixMenu)
	r.Prefix("bankbal", c.onPrefixBalance)
	r.Prefix("deposit", c.onPrefixDeposit)
	r.Prefix("withdraw", c.onPrefixWithdraw)
	r.Component("bank", "deposit", c.onDepositOpen)
	r.Component("bank", "withdraw", c.onWithdrawOpen)
	r.Component("bank", "balance", c.onBalance)
	r.Modal("bank", "deposit_submit", c.onDepositSubmit)
	r.Modal("bank", "withdraw_submit", c.onWithdrawSubmit)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.menu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	embed, comps := c.menu(lang, userID)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) onSlashBalance(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	wallet, bank, interest, err := c.svc.Info(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "bank.invalid")
		return
	}
	embed := components.Embed(i18n.T("bank.balance_title", lang), "", 0x3498db)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("bank.wallet", lang), "$"+strconv.Itoa(wallet), true),
		components.Field(i18n.T("bank.safe", lang), "$"+strconv.Itoa(bank), true),
		components.Field(i18n.T("bank.daily_interest", lang), "+$"+strconv.Itoa(interest)+" / jour", false),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onPrefixBalance(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	wallet, bank, interest, err := c.svc.Info(userID)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("bank.invalid", lang))
		return
	}
	embed := components.Embed(i18n.T("bank.balance_title", lang), "", 0x3498db)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("bank.wallet", lang), "$"+strconv.Itoa(wallet), true),
		components.Field(i18n.T("bank.safe", lang), "$"+strconv.Itoa(bank), true),
		components.Field(i18n.T("bank.daily_interest", lang), "+$"+strconv.Itoa(interest)+" / jour", false),
	}
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func (c *Cog) onSlashDeposit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	amount := int(i.ApplicationCommandData().Options[0].IntValue())
	if amount <= 0 {
		interaction.RespondError(b, i, lang, "bank.invalid")
		return
	}
	wallet, bank, derr := c.svc.Deposit(int(userID), amount)
	if derr != nil {
		switch derr {
		case banksvc.ErrNoMoney:
			interaction.RespondError(b, i, lang, "bank.insufficient")
		default:
			interaction.RespondError(b, i, lang, "bank.invalid")
		}
		return
	}
	embed := components.Embed(i18n.T("bank.deposit_title", lang), "", 0x2ecc71)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("bank.amount_label", lang), "**$"+strconv.Itoa(amount)+"**", false),
		components.Field(i18n.T("bank.wallet", lang), "$"+strconv.Itoa(wallet), true),
		components.Field(i18n.T("bank.safe", lang), "$"+strconv.Itoa(bank), true),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
	if n, ok := c.store.PopQuestNotification(userID); ok {
		interaction.SendQuestNotification(b, i, n, lang)
	}
}

func (c *Cog) onPrefixDeposit(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	parts := strings.Fields(m.Content)
	if len(parts) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("bank.invalid", lang))
		return
	}
	amount, err := strconv.Atoi(parts[1])
	if err != nil || amount <= 0 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("bank.invalid", lang))
		return
	}
	wallet, bank, derr := c.svc.Deposit(int(userID), amount)
	if derr != nil {
		switch derr {
		case banksvc.ErrNoMoney:
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("bank.insufficient", lang))
		default:
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("bank.invalid", lang))
		}
		return
	}
	embed := components.Embed(i18n.T("bank.deposit_title", lang), "", 0x2ecc71)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("bank.amount_label", lang), "**$"+strconv.Itoa(amount)+"**", false),
		components.Field(i18n.T("bank.wallet", lang), "$"+strconv.Itoa(wallet), true),
		components.Field(i18n.T("bank.safe", lang), "$"+strconv.Itoa(bank), true),
	}
	if n, ok := c.store.PopQuestNotification(userID); ok {
		embed.Description = questssvc.QuestNotificationMsg(n, lang)
	}
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func (c *Cog) onSlashWithdraw(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	amount := int(i.ApplicationCommandData().Options[0].IntValue())
	if amount <= 0 {
		interaction.RespondError(b, i, lang, "bank.invalid")
		return
	}
	wallet, bank, werr := c.svc.Withdraw(int(userID), amount)
	if werr != nil {
		switch werr {
		case banksvc.ErrNoMoney:
			interaction.RespondError(b, i, lang, "bank.insufficient")
		default:
			interaction.RespondError(b, i, lang, "bank.invalid")
		}
		return
	}
	embed := components.Embed(i18n.T("bank.withdraw_title", lang), "", 0xe67e22)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("bank.amount_label", lang), "**$"+strconv.Itoa(amount)+"**", false),
		components.Field(i18n.T("bank.wallet", lang), "$"+strconv.Itoa(wallet), true),
		components.Field(i18n.T("bank.safe", lang), "$"+strconv.Itoa(bank), true),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onPrefixWithdraw(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	parts := strings.Fields(m.Content)
	if len(parts) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("bank.invalid", lang))
		return
	}
	amount, err := strconv.Atoi(parts[1])
	if err != nil || amount <= 0 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("bank.invalid", lang))
		return
	}
	wallet, bank, werr := c.svc.Withdraw(int(userID), amount)
	if werr != nil {
		switch werr {
		case banksvc.ErrNoMoney:
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("bank.insufficient", lang))
		default:
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("bank.invalid", lang))
		}
		return
	}
	embed := components.Embed(i18n.T("bank.withdraw_title", lang), "", 0xe67e22)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("bank.amount_label", lang), "**$"+strconv.Itoa(amount)+"**", false),
		components.Field(i18n.T("bank.wallet", lang), "$"+strconv.Itoa(wallet), true),
		components.Field(i18n.T("bank.safe", lang), "$"+strconv.Itoa(bank), true),
	}
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func (c *Cog) menu(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := components.Embed(
		i18n.T("bank.menu_title", lang),
		i18n.T("bank.menu_desc", lang),
		0xf1c40f,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("bank.btn_deposit", lang), components.EncodeOwner(userID, "bank", "deposit"), discordgo.PrimaryButton),
			components.Button(i18n.T("bank.btn_withdraw", lang), components.EncodeOwner(userID, "bank", "withdraw"), discordgo.SecondaryButton),
			components.Button(i18n.T("bank.btn_balance", lang), components.EncodeOwner(userID, "bank", "balance"), discordgo.SuccessButton),
		),
	}
	return embed, comps
}

func (c *Cog) onDepositOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("bank", "deposit_submit"),
		i18n.T("bank.deposit_modal_title", lang),
		components.TextInput("amount", i18n.T("bank.amount_label", lang), true, "100", discordgo.TextInputShort, 1, 12),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onWithdrawOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("bank", "withdraw_submit"),
		i18n.T("bank.withdraw_title", lang),
		components.TextInput("amount", i18n.T("bank.amount_label", lang), true, "100", discordgo.TextInputShort, 1, 12),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onDepositSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	values := interaction.ModalValues(i)
	amount, err := strconv.Atoi(strings.TrimSpace(values["amount"]))
	if err != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "bank.invalid")
		return
	}
	wallet, bank, derr := c.svc.Deposit(int(userID), amount)
	if derr != nil {
		switch derr {
		case banksvc.ErrNoMoney:
			interaction.RespondError(b, i, lang, "bank.insufficient")
		default:
			interaction.RespondError(b, i, lang, "bank.invalid")
		}
		return
	}
	embed := components.Embed(i18n.T("bank.deposit_title", lang), "", 0x2ecc71)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("bank.amount_label", lang), "**$"+strconv.Itoa(amount)+"**", false),
		components.Field(i18n.T("bank.wallet", lang), "$"+strconv.Itoa(wallet), true),
		components.Field(i18n.T("bank.safe", lang), "$"+strconv.Itoa(bank), true),
	}
	_, comps := c.menu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
	if n, ok := c.store.PopQuestNotification(userID); ok {
		interaction.SendQuestNotification(b, i, n, lang)
	}
}

func (c *Cog) onWithdrawSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	values := interaction.ModalValues(i)
	amount, err := strconv.Atoi(strings.TrimSpace(values["amount"]))
	if err != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "bank.invalid")
		return
	}
	wallet, bank, werr := c.svc.Withdraw(int(userID), amount)
	if werr != nil {
		switch werr {
		case banksvc.ErrNoMoney:
			interaction.RespondError(b, i, lang, "bank.insufficient")
		default:
			interaction.RespondError(b, i, lang, "bank.invalid")
		}
		return
	}
	embed := components.Embed(i18n.T("bank.withdraw_title", lang), "", 0xe67e22)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("bank.amount_label", lang), "**$"+strconv.Itoa(amount)+"**", false),
		components.Field(i18n.T("bank.wallet", lang), "$"+strconv.Itoa(wallet), true),
		components.Field(i18n.T("bank.safe", lang), "$"+strconv.Itoa(bank), true),
	}
	_, comps := c.menu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onBalance(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	wallet, bank, interest, err := c.svc.Info(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "bank.invalid")
		return
	}
	embed := components.Embed(i18n.T("bank.balance_title", lang), "", 0x3498db)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("bank.wallet", lang), "$"+strconv.Itoa(wallet), true),
		components.Field(i18n.T("bank.safe", lang), "$"+strconv.Itoa(bank), true),
		components.Field(i18n.T("bank.daily_interest", lang), "+$"+strconv.Itoa(interest)+" / jour", false),
	}
	_, comps := c.menu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}



