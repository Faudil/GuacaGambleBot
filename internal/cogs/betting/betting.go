package betting

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	bettingsvc "guacagamblebot/internal/service/betting"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *bettingsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: bettingsvc.New(s, cfg)}
	r.Slash("betting", "Paris personnalisés : créer, parier, clôturer.", c.onSlashMenu)
	r.Prefix("betting", c.onPrefixMenu)
	r.Component("betting", "create", c.onCreateOpen)
	r.Component("betting", "place", c.onPlaceOpen)
	r.Component("betting", "close", c.onCloseOpen)
	r.Component("betting", "odds", c.onOddsOpen)
	r.Component("betting", "freeze", c.onFreezeOpen)
	r.Modal("betting", "create_submit", c.onCreateSubmit)
	r.Modal("betting", "place_submit", c.onPlaceSubmit)
	r.Modal("betting", "close_submit", c.onCloseSubmit)
	r.Modal("betting", "odds_submit", c.onOddsSubmit)
	r.Modal("betting", "freeze_submit", c.onFreezeSubmit)
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

func (c *Cog) menu(lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := components.Embed(
		i18n.T("betting.status_title", lang, map[string]any{"id": "?"}),
		i18n.T("betting.odds_footer_open", lang),
		0x9b59b6,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("🎲 "+i18n.T("item_manager.accept_label", lang), components.Encode("betting", "create"), discordgo.SuccessButton),
			components.Button("🎯 "+i18n.T("betting.won_msg", lang, map[string]any{"user": "", "amount": ""}), components.Encode("betting", "place"), discordgo.PrimaryButton),
		),
		components.ActionRow(
			components.Button("📊 "+i18n.T("betting.total_bet", lang), components.Encode("betting", "odds"), discordgo.SecondaryButton),
			components.Button("🔒 "+i18n.T("betting.result_title", lang, map[string]any{"id": ""}), components.Encode("betting", "close"), discordgo.DangerButton),
			components.Button("❄️ "+i18n.T("betting.finished", lang), components.Encode("betting", "freeze"), discordgo.SecondaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onCreateOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("betting", "create_submit"),
		i18n.T("betting.odds_footer_open", lang),
		components.TextInput("description", "Description", true, "Who will win?", discordgo.TextInputShort, 1, 100),
		components.TextInput("option1", "Option A", true, "Team A", discordgo.TextInputShort, 1, 50),
		components.TextInput("option2", "Option B", true, "Team B", discordgo.TextInputShort, 1, 50),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onPlaceOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("betting", "place_submit"),
		i18n.T("betting.odds_footer_open", lang),
		components.TextInput("bet_id", "Bet ID", true, "1", discordgo.TextInputShort, 1, 12),
		components.TextInput("option", "a or b", true, "a", discordgo.TextInputShort, 1, 1),
		components.TextInput("amount", i18n.T("economy.quantity", lang), true, "100", discordgo.TextInputShort, 1, 12),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onCloseOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("betting", "close_submit"),
		i18n.T("betting.result_title", lang, map[string]any{"id": ""}),
		components.TextInput("bet_id", "Bet ID", true, "1", discordgo.TextInputShort, 1, 12),
		components.TextInput("winner", "Winning option (a or b)", true, "a", discordgo.TextInputShort, 1, 1),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onOddsOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("betting", "odds_submit"),
		i18n.T("betting.total_bet", lang),
		components.TextInput("bet_id", "Bet ID", true, "1", discordgo.TextInputShort, 1, 12),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onFreezeOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("betting", "freeze_submit"),
		i18n.T("betting.finished", lang),
		components.TextInput("bet_id", "Bet ID", true, "1", discordgo.TextInputShort, 1, 12),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onCreateSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	values := interaction.ModalValues(i)

	desc := strings.TrimSpace(values["description"])
	opt1 := strings.TrimSpace(values["option1"])
	opt2 := strings.TrimSpace(values["option2"])
	if desc == "" || opt1 == "" || opt2 == "" {
		interaction.RespondError(b, i, lang, "betting.id_not_found")
		return
	}

	betID, err := c.svc.CreateBet(userID, desc, opt1, opt2)
	if err != nil {
		interaction.RespondError(b, i, lang, "betting.id_not_found")
		return
	}

	embed := components.Embed(
		i18n.T("betting.created", lang, map[string]any{"id": betID, "desc": desc}),
		"",
		0x2ecc71,
	)
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onPlaceSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	values := interaction.ModalValues(i)

	betID, err := strconv.ParseInt(strings.TrimSpace(values["bet_id"]), 10, 64)
	if err != nil {
		interaction.RespondError(b, i, lang, "betting.not_found")
		return
	}

	choice := strings.ToLower(strings.TrimSpace(values["option"]))
	amount, aerr := strconv.Atoi(strings.TrimSpace(values["amount"]))
	if aerr != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "betting.no_money")
		return
	}

	perr := c.svc.PlaceBet(userID, betID, choice, amount)
	if perr != nil {
		switch perr {
		case bettingsvc.ErrNotFound:
			interaction.RespondError(b, i, lang, "betting.not_found")
		case bettingsvc.ErrClosed:
			interaction.RespondError(b, i, lang, "betting.finished")
		case bettingsvc.ErrFrozen:
			interaction.RespondError(b, i, lang, "betting.frozen")
		case bettingsvc.ErrNoMoney:
			interaction.RespondError(b, i, lang, "betting.no_money")
		default:
			interaction.RespondError(b, i, lang, "betting.not_found")
		}
		return
	}

	embed := components.Embed(
		i18n.T("betting.placed", lang, map[string]any{"choice": choice}),
		"",
		0x2ecc71,
	)
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onCloseSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	values := interaction.ModalValues(i)

	betID, err := strconv.ParseInt(strings.TrimSpace(values["bet_id"]), 10, 64)
	if err != nil {
		interaction.RespondError(b, i, lang, "betting.id_not_found")
		return
	}

	winner := strings.ToLower(strings.TrimSpace(values["winner"]))

	res, cerr := c.svc.CloseBet(userID, betID, winner)
	if cerr != nil {
		switch cerr {
		case bettingsvc.ErrNotFound:
			interaction.RespondError(b, i, lang, "betting.id_not_found")
		case bettingsvc.ErrNotCreator:
			interaction.RespondError(b, i, lang, "betting.only_creator")
		case bettingsvc.ErrClosed:
			interaction.RespondError(b, i, lang, "betting.already_closed")
		case bettingsvc.ErrInvalidOpt:
			interaction.RespondError(b, i, lang, "betting.invalid_choice")
		default:
			interaction.RespondError(b, i, lang, "betting.id_not_found")
		}
		return
	}

	winningOptName := "A"
	if winner == "b" {
		winningOptName = "B"
	}

	desc := i18n.T("item_manager.winner", lang) + ": **" + winningOptName + "**"
	if res.WinningPool == 0 {
		desc = i18n.T("betting.closed_house_keeps", lang, map[string]any{
			"option": winningOptName,
			"pool":   res.TotalPool,
		})
	}

	embed := components.Embed(
		i18n.T("betting.result_title", lang, map[string]any{"id": betID}),
		desc,
		0x9b59b6,
	)

	if len(res.WagerResults) > 0 {
		var winners []string
		for _, wr := range res.WagerResults {
			if wr.Won {
				winners = append(winners, i18n.T("betting.won_msg", lang, map[string]any{
					"user":   interaction.Mention(wr.UserID),
					"amount": wr.Amount,
				}))
			}
		}
		if len(winners) > 0 {
			embed.Fields = []*discordgo.MessageEmbedField{
				components.Field(i18n.T("betting.total_pool", lang), "$"+strconv.Itoa(res.TotalPool), false),
				components.Field(i18n.T("betting.winners", lang), strings.Join(winners, "\n"), false),
			}
		}
	}

	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onOddsSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	values := interaction.ModalValues(i)

	betID, err := strconv.ParseInt(strings.TrimSpace(values["bet_id"]), 10, 64)
	if err != nil {
		interaction.RespondError(b, i, lang, "betting.not_found")
		return
	}

	odds, oerr := c.svc.ShowOdds(betID)
	if oerr != nil {
		interaction.RespondError(b, i, lang, "betting.not_found")
		return
	}

	embed := components.Embed(
		i18n.T("betting.status_title", lang, map[string]any{"id": odds.BetID}),
		odds.Description,
		0x3498db,
	)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("betting.total_bet", lang), "$"+strconv.Itoa(odds.Total), false),
		components.Field(i18n.T("betting.option_a", lang, map[string]any{"name": odds.Option1}), "$"+strconv.Itoa(odds.Pool1)+" | "+odds.Odds1, true),
		components.Field(i18n.T("betting.option_b", lang, map[string]any{"name": odds.Option2}), "$"+strconv.Itoa(odds.Pool2)+" | "+odds.Odds2, true),
	}

	if odds.Status == "CLOSE" || odds.Status == "FROZEN" {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("betting.odds_footer_closed", lang, map[string]any{"winner": odds.Winner})}
	} else {
		embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("betting.odds_footer_open", lang)}
	}

	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onFreezeSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	values := interaction.ModalValues(i)

	betID, err := strconv.ParseInt(strings.TrimSpace(values["bet_id"]), 10, 64)
	if err != nil {
		interaction.RespondError(b, i, lang, "betting.id_not_found")
		return
	}

	ferr := c.svc.FreezeBet(userID, betID)
	if ferr != nil {
		switch ferr {
		case bettingsvc.ErrNotFound:
			interaction.RespondError(b, i, lang, "betting.id_not_found")
		case bettingsvc.ErrNotCreator:
			interaction.RespondError(b, i, lang, "betting.only_creator")
		case bettingsvc.ErrClosed:
			interaction.RespondError(b, i, lang, "betting.already_closed")
		case bettingsvc.ErrFrozen:
			interaction.RespondError(b, i, lang, "betting.frozen")
		default:
			interaction.RespondError(b, i, lang, "betting.id_not_found")
		}
		return
	}

	embed := components.Embed(
		i18n.T("betting.freeze_success", lang, map[string]any{"desc": "Bet #" + strconv.FormatInt(betID, 10)}),
		"",
		0x2ecc71,
	)
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}
