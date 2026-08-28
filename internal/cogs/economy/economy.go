package economy

import (
	"errors"
	"fmt"
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
	dailyquestsvc "guacagamblebot/internal/service/dailyquest"
	economysvc "guacagamblebot/internal/service/economy"
	invsvc "guacagamblebot/internal/service/inventory"
	npcsvc "guacagamblebot/internal/service/npcs"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
)

var one = float64(1)

// Cog implements the Economy "embed interface": a single menu with buttons that
// replace the old `!balance`, `!daily` and `!give` commands.
type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *economysvc.Service
	dq    *dailyquestsvc.Service
}

// Register wires the cog into the router under both slash and prefix triggers.
func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	def := universe.Get(cfg.Universe)
	if def == nil {
		def = universe.Get("hoakhaven")
	}
	inv := invsvc.New(s, cfg)
	npc := npcsvc.New(s, cfg, def, inv)
	dq := dailyquestsvc.New(s, npc)
	c := &Cog{store: s, cfg: cfg, svc: economysvc.New(s, cfg, dq), dq: dq}
	r.Slash("economy", "cmd.economy.desc", c.onSlashMenu)
	r.Slash("eco", "cmd.economy.desc", c.onSlashMenu)
	r.Slash("bal", "cmd.balance.desc", c.onSlashBalance)
	r.Slash("balance", "cmd.balance.desc", c.onSlashBalance)
	r.Slash("daily", "cmd.daily.desc", c.onSlashDaily)
	r.Prefix("economy", c.onPrefixMenu)
	r.Prefix("eco", c.onPrefixMenu)
	r.Prefix("bal", c.onPrefixBalance)
	r.Prefix("balance", c.onPrefixBalance)
	r.Prefix("d", c.onPrefixDaily)
	r.Prefix("daily", c.onPrefixDaily)
	r.Prefix("give", c.onPrefixGive)
	r.Prefix("pay", c.onPrefixGive)
	r.SlashWithOptions("give", "cmd.give.desc",
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
	r.Component("economy", "daily_view", c.onDailyView)
	r.Component("economy", "daily_deliver", c.onDailyDeliver)
	r.Component("economy", "give", c.onGiveOpen)
	r.Modal("economy", "give_submit", c.onGiveSubmit)
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
	res, err := c.svc.Balance(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "economy.give_no_money")
		return
	}
	embed := components.Embed(i18n.T("economy.balance_title", lang), "", components.ColorInfo)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.wallet", lang), "$"+strconv.Itoa(res.Wallet), true),
		components.Field(i18n.T("economy.safe", lang), "$"+strconv.Itoa(res.Bank)+"/"+strconv.Itoa(res.MaxBank), true),
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
	embed := components.Embed(i18n.T("economy.balance_title", lang), "", components.ColorInfo)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.wallet", lang), "$"+strconv.Itoa(res.Wallet), true),
		components.Field(i18n.T("economy.safe", lang), "$"+strconv.Itoa(res.Bank)+"/"+strconv.Itoa(res.MaxBank), true),
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
	embed := components.Embed(i18n.T("economy.daily_title", lang), "", components.ColorSuccess)
	fields := []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.quantity", lang), "+$"+strconv.Itoa(res.Amount), true),
	}
	if res.Repaid > 0 {
		fields = append(fields, components.Field(i18n.T("economy.tax_repayment", lang), "-$"+strconv.Itoa(res.Repaid), false))
		for _, l := range res.Lenders {
			fields = append(fields, components.Field(
				i18n.T("economy.repaid_lender", lang, map[string]any{"lender": interaction.DisplayName(b.Session, i.GuildID, i.Member, l.LenderID)}),
				"$"+strconv.Itoa(l.Amount), false))
		}
	}
	fields = append(fields, components.Field(i18n.T("economy.your_balance", lang), "$"+strconv.Itoa(res.NewBalance), false))
	embed.Fields = fields
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("economy.daily_footer", lang)}
	if res.LeveledUp {
		embed.Description = i18n.T("character.level_up", lang, map[string]any{"level": res.NewLevel})
	}
	var comps []discordgo.MessageComponent
	if res.Recipe != nil {
		if embed.Description != "" {
			embed.Description += "\n"
		}
		embed.Description += i18n.T("quests.daily.new_quest", lang, map[string]any{"title": i18n.T(res.Recipe.TitleKey, lang)})
		comps = append(comps, components.ActionRow(
			components.Button("📜 "+i18n.T("quests.daily.view_btn", lang),
				components.EncodeOwner(userID, "economy", "daily_view"),
				discordgo.SuccessButton),
		))
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))

	if n, ok := c.store.PopQuestNotification(userID); ok {
		interaction.SendQuestNotification(b, i, n, lang)
	}

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
	embed := components.Embed(i18n.T("economy.daily_title", lang), "", components.ColorSuccess)
	fields := []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.quantity", lang), "+$"+strconv.Itoa(res.Amount), true),
	}
	if res.Repaid > 0 {
		fields = append(fields, components.Field(i18n.T("economy.tax_repayment", lang), "-$"+strconv.Itoa(res.Repaid), false))
		for _, l := range res.Lenders {
			fields = append(fields, components.Field(
				i18n.T("economy.repaid_lender", lang, map[string]any{"lender": interaction.DisplayName(s, m.GuildID, &discordgo.Member{User: m.Author}, l.LenderID)}),
				"$"+strconv.Itoa(l.Amount), false))
		}
	}
	fields = append(fields, components.Field(i18n.T("economy.your_balance", lang), "$"+strconv.Itoa(res.NewBalance), false))
	embed.Fields = fields
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("economy.daily_footer", lang)}
	if res.LeveledUp {
		embed.Description = i18n.T("character.level_up", lang, map[string]any{"level": res.NewLevel})
	}
	if res.Recipe != nil {
		embed.Description += "\n\n" + i18n.T("quests.daily.new_quest", lang, map[string]any{"title": i18n.T(res.Recipe.TitleKey, lang)})
	}
	if n, ok := c.store.PopQuestNotification(userID); ok {
		embed.Description += "\n\n" + questssvc.QuestNotificationMsg(n, lang)
	}
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func (c *Cog) onPrefixGive(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	sender := interaction.ToInt64(m.Author.ID)
	parts := strings.Fields(m.Content)
	if len(parts) < 3 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("economy.give_invalid", lang))
		return
	}
	recipient, ok := interaction.ParseUserID(parts[1])
	if !ok {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("economy.give_invalid", lang))
		return
	}
	amount, err := strconv.Atoi(parts[2])
	if err != nil || amount <= 0 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("economy.give_invalid", lang))
		return
	}
	_, _, gerr := c.svc.Give(sender, recipient, amount)
	if gerr != nil {
		switch gerr {
		case economysvc.ErrNoMoney:
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("economy.give_no_money", lang))
		default:
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("economy.give_invalid", lang))
		}
		return
	}
	embed := components.Embed(i18n.T("economy.give_title", lang), "", components.ColorSuccess)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.sender", lang), interaction.Mention(sender), true),
		components.Field(i18n.T("economy.receiver", lang), interaction.Mention(recipient), true),
		components.Field(i18n.T("economy.quantity", lang), "**$"+strconv.Itoa(amount)+"**", false),
	}
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func (c *Cog) menu(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := components.Embed(
		i18n.T("economy.menu_title", lang),
		i18n.T("economy.menu_desc", lang),
		components.ColorEconomy,
	)
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("economy.menu_footer", lang)}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("economy.btn_balance", lang), components.EncodeOwner(userID, "economy", "balance"), discordgo.PrimaryButton),
			components.Button(i18n.T("economy.btn_daily", lang), components.EncodeOwner(userID, "economy", "daily"), discordgo.SuccessButton),
			components.Button(i18n.T("economy.btn_give", lang), components.EncodeOwner(userID, "economy", "give"), discordgo.SecondaryButton),
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
	embed := components.Embed(i18n.T("economy.balance_title", lang), "", components.ColorInfo)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.wallet", lang), "$"+strconv.Itoa(res.Wallet), true),
		components.Field(i18n.T("economy.safe", lang), "$"+strconv.Itoa(res.Bank)+"/"+strconv.Itoa(res.MaxBank), true),
		components.Field(i18n.T("economy.daily_interest", lang), "+$"+strconv.Itoa(res.Interest)+" / jour", false),
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("economy.balance_footer", lang)}
	_, comps := c.menu(lang, userID)
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
	embed := components.Embed(i18n.T("economy.daily_title", lang), "", components.ColorSuccess)
	fields := []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.quantity", lang), "+$"+strconv.Itoa(res.Amount), true),
	}
	if res.Repaid > 0 {
		fields = append(fields, components.Field(i18n.T("economy.tax_repayment", lang), "-$"+strconv.Itoa(res.Repaid), false))
		for _, l := range res.Lenders {
			fields = append(fields, components.Field(
				i18n.T("economy.repaid_lender", lang, map[string]any{"lender": interaction.DisplayName(b.Session, i.GuildID, i.Member, l.LenderID)}),
				"$"+strconv.Itoa(l.Amount), false))
		}
	}
	fields = append(fields, components.Field(i18n.T("economy.your_balance", lang), "$"+strconv.Itoa(res.NewBalance), false))
	embed.Fields = fields
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("economy.daily_footer", lang)}
	_, comps := c.menu(lang, userID)
	if res.Recipe != nil {
		embed.Description = i18n.T("quests.daily.new_quest", lang, map[string]any{"title": i18n.T(res.Recipe.TitleKey, lang)})
		comps = append([]discordgo.MessageComponent{
			components.ActionRow(
				components.Button("📜 "+i18n.T("quests.daily.view_btn", lang),
					components.EncodeOwner(userID, "economy", "daily_view"),
					discordgo.SuccessButton),
			),
		}, comps...)
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))

	if n, ok := c.store.PopQuestNotification(userID); ok {
		interaction.SendQuestNotification(b, i, n, lang)
	}

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
	embed := components.Embed(i18n.T("economy.give_title", lang), "", components.ColorSuccess)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("economy.sender", lang), interaction.Mention(sender), true),
		components.Field(i18n.T("economy.receiver", lang), interaction.Mention(recipient), true),
		components.Field(i18n.T("economy.quantity", lang), "**$"+strconv.Itoa(amount)+"**", false),
	}
	_, comps := c.menu(lang, sender)
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
	embed := components.Embed(i18n.T("economy.give_title", lang), "", components.ColorSuccess)
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

// ─── Procedural daily quest view ────────────────────────────────

// onDailyView renders the active daily quest with its steps, progress and the
// delivery button when the turn-in step is current.
func (c *Cog) onDailyView(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.buildDailyEmbed(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// onDailyDeliver claims the current turn-in step of the daily quest.
func (c *Cog) onDailyDeliver(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	recipe, completed, err := c.dq.Claim(userID)
	if err != nil {
		var missing *store.DailyMissingItemsError
		if errors.As(err, &missing) {
			var lines []string
			for _, m := range missing.Items {
				lines = append(lines, i18n.T("quests.daily.missing_item", lang, map[string]any{
					"item": items.LocalizedName(m.ItemID, lang), "needed": m.Needed, "have": m.Have,
				}))
			}
			embed := components.Embed("❌ "+i18n.T("quests.daily.title", lang),
				i18n.T("quests.daily.missing_items", lang)+"\n"+strings.Join(lines, "\n"), components.ColorDanger)
			comps := []discordgo.MessageComponent{
				components.ActionRow(
					components.Button("↩️", components.EncodeOwner(userID, "economy", "daily_view"), discordgo.SecondaryButton),
				),
			}
			_ = b.Session.InteractionRespond(i.Interaction,
				components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
			return
		}
		interaction.RespondError(b, i, lang, "quests.daily.title")
		return
	}

	if completed {
		desc := ""
		if recipe.ThankKey != "" {
			desc += "*" + i18n.T(recipe.ThankKey, lang) + "*\n\n"
		}
		desc += i18n.T("quests.daily.completed", lang, map[string]any{
			"title": i18n.T(recipe.TitleKey, lang), "rewards": c.dailyRewardText(lang, recipe.Reward),
		})
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button("↩️", components.EncodeOwner(userID, "economy", "daily"), discordgo.SecondaryButton),
			),
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
				components.Embed("✅ "+i18n.T("quests.daily.title", lang), desc, components.ColorSuccess), comps))
		return
	}

	embed, comps := c.buildDailyEmbed(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// buildDailyEmbed renders the active daily quest: requestor intro, each step
// with state/progress, the reward preview and a Deliver button on the turn-in.
func (c *Cog) buildDailyEmbed(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	recipe, data, err := c.dq.Current(userID)
	if err != nil {
		return components.Embed(i18n.T("quests.daily.title", lang), i18n.T("quests.daily.no_active", lang), components.ColorSuccess), nil
	}
	uq, _, uerr := c.store.GetUserQuest(userID, "daily_quest")
	if uerr != nil || uq == nil || uq.Status == "COMPLETED" {
		return components.Embed(i18n.T("quests.daily.title", lang),
			i18n.T("quests.daily.completed_short", lang, map[string]any{"title": i18n.T(recipe.TitleKey, lang)}), components.ColorSuccess), nil
	}

	npcName := c.npcName(recipe.Requestor)
	desc := ""
	if recipe.MoodKey != "" {
		desc += i18n.T(recipe.MoodKey, lang) + "\n\n"
	}
	desc += i18n.T(recipe.IntroKey, lang, map[string]any{"npc": npcName}) + "\n\n"
	for idx, st := range recipe.Steps {
		state := "⬜"
		if idx < data.StepIndex {
			state = "✅"
		} else if idx == data.StepIndex {
			state = "🔵"
		}
		line := c.dailyStepText(lang, st)
		if st.Kind == store.DailyStepActivity && idx == data.StepIndex {
			line += fmt.Sprintf(" (%d/%d)", data.ProgressValue, st.Count)
		}
		desc += fmt.Sprintf("%s **%d.** %s\n", state, idx+1, line)
	}
	desc += "\n" + i18n.T("quests.daily.reward_label", lang) + " " + c.dailyRewardText(lang, recipe.Reward)
	if streak := c.dq.Streak(userID); streak > 0 {
		desc += "\n" + i18n.T("quests.daily.streak", lang, map[string]any{"n": streak})
	}
	desc += "\n" + i18n.T("quests.daily.jackpot_odds", lang, map[string]any{
		"pct": c.dq.JackpotChance(userID),
	})
	if isSunday() && recipe.Reward.RepNPC != "" {
		desc += "\n" + i18n.T("quests.daily.special_rep", lang, map[string]any{"npc": npcName})
	}

	npcData := c.npcData(recipe.Requestor)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("↩️", components.EncodeOwner(userID, "economy", "daily"), discordgo.SecondaryButton),
		),
	}
	if npcData != nil {
		comps = append(comps, components.ActionRow(
			components.Button("💬 "+i18n.T("npcs.chat_btn", lang),
				components.EncodeOwner(userID, "npc", recipe.Requestor), discordgo.PrimaryButton),
		))
	}
	if data.StepIndex < len(recipe.Steps) && recipe.Steps[data.StepIndex].Kind == store.DailyStepTurnIn {
		comps = append(comps, components.ActionRow(
			components.Button("📦 "+i18n.T("quests.daily.deliver_btn", lang),
				components.EncodeOwner(userID, "economy", "daily_deliver"), discordgo.SuccessButton),
		))
	}
	return components.Embed(i18n.T("quests.daily.title", lang)+" — "+i18n.T(recipe.TitleKey, lang), desc, components.ColorSuccess), comps
}

// dailyStepText renders one recipe step's line with its placeholders.
func (c *Cog) dailyStepText(lang string, st store.DailyStep) string {
	switch st.Kind {
	case store.DailyStepActivity:
		vars := map[string]any{"n": st.Count}
		if st.Zone != "" {
			vars["zone"] = i18n.T("hunt."+st.Zone+"_zone", lang)
		}
		return i18n.T(st.TextKey, lang, vars)
	case store.DailyStepTurnIn:
		for itemID, qty := range st.Items {
			return i18n.T(st.TextKey, lang, map[string]any{"n": qty, "item": items.LocalizedName(itemID, lang)})
		}
	}
	return ""
}

// dailyRewardText renders the recipe reward as a single display string.
func (c *Cog) dailyRewardText(lang string, r store.DailyReward) string {
	var parts []string
	if r.Money > 0 {
		parts = append(parts, i18n.T("quests.reward_money", lang, map[string]any{"amount": r.Money}))
	}
	if r.Crowns > 0 {
		parts = append(parts, i18n.T("quests.reward_crowns", lang, map[string]any{"amount": r.Crowns}))
	}
	if r.ItemID != "" {
		if it := items.Get(r.ItemID); it != nil {
			parts = append(parts, it.Emoji+" "+items.LocalizedName(it.ID, lang))
		}
	}
	if r.RepPoints > 0 {
		parts = append(parts, i18n.T("quests.daily.reward_rep", lang, map[string]any{"points": r.RepPoints}))
	}
	return strings.Join(parts, " · ")
}

// npcData resolves an NPC's display data for the configured universe, or nil
// when the universe has no such NPC (e.g. the Town Board).
func (c *Cog) npcData(npcID string) *universe.NPCData {
	def := universe.Get(c.cfg.Universe)
	if def == nil {
		def = universe.Get("hoakhaven")
	}
	if def != nil {
		if n, ok := def.NPCs[npcID]; ok {
			return n
		}
	}
	return nil
}

// npcName resolves an NPC's display name for the configured universe, falling
// back to the NPC id when the universe is unknown.
func (c *Cog) npcName(npcID string) string {
	if n := c.npcData(npcID); n != nil {
		return n.Name
	}
	return npcID
}

// isSunday reports whether today is Sunday (the day-of-week special).
func isSunday() bool {
	return time.Now().Weekday() == time.Sunday
}
