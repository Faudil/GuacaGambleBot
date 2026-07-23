package lotto

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	lottosvc "guacagamblebot/internal/service/lotto"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *lottosvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: lottosvc.New(s, cfg)}
	r.Slash("lotto", "Loterie du serveur : acheter un ticket.", c.onSlashMenu)
	r.Prefix("lotto", c.onPrefixMenu)
	r.Prefix("lt", c.onPrefixMenu)
	r.Component("lotto", "buy", c.onBuyOpen)
	r.Component("lotto", "jackpot", c.onJackpot)
	r.Component("lotto", "refresh", c.onRefresh)
	r.Modal("lotto", "buy_submit", c.onBuySubmit)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(lang, interaction.ToInt64(i.GuildID))
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	embed, comps := c.menu(lang, interaction.ToInt64(m.GuildID))
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) menu(lang string, serverID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	info, err := c.svc.Jackpot(serverID)
	desc := i18n.T("lotto.show_jackpot_desc", lang, map[string]any{
		"jackpot": 0,
		"price":   c.svc.TicketPrice,
	})
	if err == nil && info != nil {
		desc = i18n.T("lotto.show_jackpot_desc", lang, map[string]any{
			"jackpot": info.Jackpot,
			"price":   c.svc.TicketPrice,
		})
	}
	embed := components.Embed(
		i18n.T("lotto.show_jackpot_title", lang),
		desc,
		0x2ecc71,
	)
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("lotto.show_jackpot_footer", lang)}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("🎫 "+i18n.T("lotto.show_jackpot_footer", lang), components.Encode("lotto", "buy"), discordgo.PrimaryButton),
			components.Button(i18n.T("leaderboard.btn_refresh", lang), components.Encode("lotto", "refresh"), discordgo.SecondaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onBuyOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("lotto", "buy_submit"),
		i18n.T("lotto.ticket_valid_title", lang),
		components.TextInput("number", i18n.T("lotto.ticket_valid_footer", lang), true, "1-100", discordgo.TextInputShort, 1, 3),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onBuySubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	serverID := interaction.ToInt64(i.GuildID)
	values := interaction.ModalValues(i)

	number, err := strconv.Atoi(strings.TrimSpace(values["number"]))
	if err != nil || number < 1 || number > 100 {
		interaction.RespondError(b, i, lang, "lotto.invalid_number")
		return
	}

	res, lerr := c.svc.BuyTicket(userID, serverID, number)
	if lerr != nil {
		switch lerr {
		case lottosvc.ErrNoMoney:
			interaction.RespondError(b, i, lang, "lotto.no_money")
		case lottosvc.ErrInvalidNum:
			interaction.RespondError(b, i, lang, "lotto.invalid_number")
		default:
			interaction.RespondError(b, i, lang, "lotto.invalid_number")
		}
		return
	}

	var embed *discordgo.MessageEmbed
	if res.Win {
		embed = components.Embed(
			i18n.T("lotto.jackpot_title", lang),
			i18n.T("lotto.jackpot_win_desc", lang, map[string]any{
				"user":     interaction.Mention(userID),
				"number":   res.Number,
				"jackpot":  res.Jackpot,
				"new_pot":  res.NewJackpot,
			}),
			0xf1c40f,
		)
	} else {
		embed = components.Embed(
			i18n.T("lotto.ticket_valid_title", lang),
			i18n.T("lotto.ticket_valid_desc", lang, map[string]any{
				"number": res.Number,
				"added":  res.AddedValue,
				"total":  res.NewJackpot,
			}),
			0x3498db,
		)
		embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("lotto.ticket_valid_footer", lang)}
	}

	_, comps := c.menu(lang, serverID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))

	if len(res.Unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, res.Unlocks)
	}
}

func (c *Cog) onJackpot(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	serverID := interaction.ToInt64(i.GuildID)
	embed, comps := c.menu(lang, serverID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onRefresh(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.onJackpot(b, i)
}
