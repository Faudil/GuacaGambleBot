package economy

import (
	"errors"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	economysvc "guacagamblebot/internal/service/economy"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
)

var one = float64(1)

// Cog implements the Economy "embed interface": a single menu with buttons that
// replace the old `!balance`, `!daily` and `!give` commands.
type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *economysvc.Service
}

// Register wires the cog into the router under both slash and prefix triggers.
func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: economysvc.New(s, cfg)}
	r.Slash("economy", "Économie : solde, daily, don.", c.onSlashMenu)
	r.Slash("eco", "Économie : solde, daily, don.", c.onSlashMenu)
	r.Slash("bal", "Voir ton solde économique.", c.onSlashBalance)
	r.Slash("balance", "Voir ton solde économique.", c.onSlashBalance)
	r.Slash("daily", "Réclamer ta récompense quotidienne.", c.onSlashDaily)
	r.Prefix("economy", c.onPrefixMenu)
	r.Prefix("eco", c.onPrefixMenu)
	r.Prefix("bal", c.onPrefixBalance)
	r.Prefix("balance", c.onPrefixBalance)
	r.Prefix("d", c.onPrefixDaily)
	r.SlashWithOptions("give", "Donner de l'argent à un autre utilisateur.",
		[]*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "recipient",
				Description: "L'utilisateur qui recevra l'argent",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "amount",
				Description: "Le montant à donner",
				Required:    true,
				MinValue:    &one,
			},
		},
		c.onSlashGive)
	r.Component("economy", "balance", c.onBalance)
	r.Component("economy", "daily", c.onDaily)
	r.Component("economy", "give", c.onGiveOpen)
	r.Modal("economy", "give_submit", c.onGiveSubmit)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
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

func (c *Cog) onSlashBalance(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	res, err := c.svc.Balance(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "economy.give_no_money")
		return
	}
	embed := components.Embed(i18n.T("economy.balance_title", lang), "", 0x3498db)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.wallet", lang), "$"+strconv.Itoa(res.Wallet), true),
		components.Field(i18n.T("economy.safe", lang), "$"+strconv.Itoa(res.Bank)+" / 500", true),
		components.Field(i18n.T("economy.daily_interest", lang), "+$"+strconv.Itoa(res.Interest)+" / jour", false),
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("economy.balance_footer", lang)}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onPrefixBalance(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	res, err := c.svc.Balance(userID)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("economy.give_no_money", lang))
		return
	}
	embed := components.Embed(i18n.T("economy.balance_title", lang), "", 0x3498db)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.wallet", lang), "$"+strconv.Itoa(res.Wallet), true),
		components.Field(i18n.T("economy.safe", lang), "$"+strconv.Itoa(res.Bank)+" / 500", true),
		components.Field(i18n.T("economy.daily_interest", lang), "+$"+strconv.Itoa(res.Interest)+" / jour", false),
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("economy.balance_footer", lang)}
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func (c *Cog) onSlashDaily(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	res, err := c.svc.Daily(userID)
	if err != nil {
		if errors.Is(err, economysvc.ErrAlreadyClaimed) {
			interaction.RespondError(b, i, lang, "economy.daily_cooldown")
			return
		}
		interaction.RespondError(b, i, lang, "economy.give_no_money")
		return
	}
	embed := components.Embed(i18n.T("economy.daily_title", lang), "", 0x2ecc71)
	fields := []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.quantity", lang), "+$"+strconv.Itoa(res.Amount), true),
	}
	if res.Repaid > 0 {
		fields = append(fields, components.Field(i18n.T("economy.tax_repayment", lang), "-$"+strconv.Itoa(res.Repaid), false))
		for _, l := range res.Lenders {
			fields = append(fields, components.Field(
				i18n.T("economy.repaid_lender", lang, map[string]any{"lender": interaction.Mention(l.LenderID)}),
				"$"+strconv.Itoa(l.Amount), false))
		}
	}
	fields = append(fields, components.Field(i18n.T("economy.your_balance", lang), "$"+strconv.Itoa(res.NewBalance), false))
	embed.Fields = fields
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("economy.daily_footer", lang)}
	if qid := c.store.PopQuestCompleted(userID); qid != "" {
		embed.Description = questssvc.QuestCompletedMsg(qid, lang)
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))

	if len(res.Unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, res.Unlocks)
	}
}

func (c *Cog) onPrefixDaily(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	res, err := c.svc.Daily(userID)
	if err != nil {
		if errors.Is(err, economysvc.ErrAlreadyClaimed) {
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("economy.daily_cooldown", lang))
			return
		}
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("economy.give_no_money", lang))
		return
	}
	embed := components.Embed(i18n.T("economy.daily_title", lang), "", 0x2ecc71)
	fields := []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.quantity", lang), "+$"+strconv.Itoa(res.Amount), true),
	}
	if res.Repaid > 0 {
		fields = append(fields, components.Field(i18n.T("economy.tax_repayment", lang), "-$"+strconv.Itoa(res.Repaid), false))
		for _, l := range res.Lenders {
			fields = append(fields, components.Field(
				i18n.T("economy.repaid_lender", lang, map[string]any{"lender": interaction.Mention(l.LenderID)}),
				"$"+strconv.Itoa(l.Amount), false))
		}
	}
	fields = append(fields, components.Field(i18n.T("economy.your_balance", lang), "$"+strconv.Itoa(res.NewBalance), false))
	embed.Fields = fields
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("economy.daily_footer", lang)}
	if qid := c.store.PopQuestCompleted(userID); qid != "" {
		embed.Description = questssvc.QuestCompletedMsg(qid, lang)
	}
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func (c *Cog) menu(lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := components.Embed(
		i18n.T("economy.menu_title", lang),
		i18n.T("economy.menu_desc", lang),
		0x1abc9c,
	)
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("economy.menu_footer", lang)}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("economy.btn_balance", lang), components.Encode("economy", "balance"), discordgo.PrimaryButton),
			components.Button(i18n.T("economy.btn_daily", lang), components.Encode("economy", "daily"), discordgo.SuccessButton),
			components.Button(i18n.T("economy.btn_give", lang), components.Encode("economy", "give"), discordgo.SecondaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onBalance(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	res, err := c.svc.Balance(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "economy.give_no_money")
		return
	}
	embed := components.Embed(i18n.T("economy.balance_title", lang), "", 0x3498db)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.wallet", lang), "$"+strconv.Itoa(res.Wallet), true),
		components.Field(i18n.T("economy.safe", lang), "$"+strconv.Itoa(res.Bank)+" / 500", true),
		components.Field(i18n.T("economy.daily_interest", lang), "+$"+strconv.Itoa(res.Interest)+" / jour", false),
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("economy.balance_footer", lang)}
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onDaily(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	res, err := c.svc.Daily(userID)
	if err != nil {
		if errors.Is(err, economysvc.ErrAlreadyClaimed) {
			interaction.RespondError(b, i, lang, "economy.daily_cooldown")
			return
		}
		interaction.RespondError(b, i, lang, "economy.give_no_money")
		return
	}
	embed := components.Embed(i18n.T("economy.daily_title", lang), "", 0x2ecc71)
	fields := []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.quantity", lang), "+$"+strconv.Itoa(res.Amount), true),
	}
	if res.Repaid > 0 {
		fields = append(fields, components.Field(i18n.T("economy.tax_repayment", lang), "-$"+strconv.Itoa(res.Repaid), false))
		for _, l := range res.Lenders {
			fields = append(fields, components.Field(
				i18n.T("economy.repaid_lender", lang, map[string]any{"lender": interaction.Mention(l.LenderID)}),
				"$"+strconv.Itoa(l.Amount), false))
		}
	}
	fields = append(fields, components.Field(i18n.T("economy.your_balance", lang), "$"+strconv.Itoa(res.NewBalance), false))
	embed.Fields = fields
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("economy.daily_footer", lang)}
	if qid := c.store.PopQuestCompleted(userID); qid != "" {
		embed.Description = questssvc.QuestCompletedMsg(qid, lang)
	}
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))

	if len(res.Unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, res.Unlocks)
	}
}

func (c *Cog) onGiveOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("economy", "give_submit"),
		i18n.T("economy.give_modal_title", lang),
		components.TextInput("recipient", i18n.T("economy.give_recipient_label", lang), true, "@user", discordgo.TextInputShort, 1, 50),
		components.TextInput("amount", i18n.T("economy.give_amount_label", lang), true, "100", discordgo.TextInputShort, 1, 12),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onGiveSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	sender := interaction.ToInt64(interaction.UserID(i))
	values := interaction.ModalValues(i)
	recipient, ok := interaction.ParseUserID(values["recipient"])
	if !ok {
		interaction.RespondError(b, i, lang, "economy.give_invalid")
		return
	}
	amount, err := strconv.Atoi(strings.TrimSpace(values["amount"]))
	if err != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "economy.give_invalid")
		return
	}
	sb, rb, gerr := c.svc.Give(sender, recipient, amount)
	if gerr != nil {
		switch gerr {
		case economysvc.ErrSelf:
			interaction.RespondError(b, i, lang, "economy.give_invalid")
		case economysvc.ErrNoMoney:
			interaction.RespondError(b, i, lang, "economy.give_no_money")
		default:
			interaction.RespondError(b, i, lang, "economy.give_invalid")
		}
		return
	}
	embed := components.Embed(i18n.T("economy.give_title", lang), "", 0x2ecc71)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.sender", lang), interaction.Mention(sender), true),
		components.Field(i18n.T("economy.receiver", lang), interaction.Mention(recipient), true),
		components.Field(i18n.T("economy.quantity", lang), "**$"+strconv.Itoa(amount)+"**", false),
	}
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))

	// Achievements for both parties (mirrors the Python behaviour).
	for _, uid := range []int64{sender, recipient} {
		if unlocks, uerr := achievement.CheckAndUnlock(b.DB, uid); uerr == nil && len(unlocks) > 0 {
			interaction.SendAchievements(b, i, lang, unlocks)
		}
	}
	_ = rb
	_ = sb
}

func (c *Cog) onSlashGive(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	sender := interaction.ToInt64(interaction.UserID(i))
	opts := i.ApplicationCommandData().Options

	recipient := interaction.ToInt64(opts[0].StringValue())
	amount := int(opts[1].IntValue())

	sb, rb, gerr := c.svc.Give(sender, recipient, amount)
	if gerr != nil {
		switch gerr {
		case economysvc.ErrSelf:
			interaction.RespondError(b, i, lang, "economy.give_invalid")
		case economysvc.ErrNoMoney:
			interaction.RespondError(b, i, lang, "economy.give_no_money")
		default:
			interaction.RespondError(b, i, lang, "economy.give_invalid")
		}
		return
	}
	embed := components.Embed(i18n.T("economy.give_title", lang), "", 0x2ecc71)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.sender", lang), interaction.Mention(sender), true),
		components.Field(i18n.T("economy.receiver", lang), interaction.Mention(recipient), true),
		components.Field(i18n.T("economy.quantity", lang), "**$"+strconv.Itoa(amount)+"**", false),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))

	for _, uid := range []int64{sender, recipient} {
		if unlocks, uerr := achievement.CheckAndUnlock(b.DB, uid); uerr == nil && len(unlocks) > 0 {
			interaction.SendAchievements(b, i, lang, unlocks)
		}
	}
	_ = sb
	_ = rb
}
