package casino

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	casinosvc "guacagamblebot/internal/service/casino"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *casinosvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: casinosvc.New(s, cfg)}
	r.Slash("casino", "Casino : machines à sous et pile ou face.", c.onSlashMenu)
	r.Prefix("casino", c.onPrefixMenu)
	r.Component("casino", "slots", c.onSlotsOpen)
	r.Component("casino", "coinflip", c.onCoinflipOpen)
	r.Modal("casino", "slots_submit", c.onSlotsSubmit)
	r.Modal("casino", "coinflip_submit", c.onCoinflipSubmit)
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
		i18n.T("slots.title", lang),
		i18n.T("slots.state_start", lang),
		0xf1c40f,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("🎰 "+i18n.T("slots.title", lang), components.Encode("casino", "slots"), discordgo.PrimaryButton),
			components.Button("🪙 "+i18n.T("coinflip.legit_label", lang), components.Encode("casino", "coinflip"), discordgo.SuccessButton),
		),
	}
	return embed, comps
}

func (c *Cog) onSlotsOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("casino", "slots_submit"),
		i18n.T("slots.title", lang),
		components.TextInput("amount", i18n.T("economy.quantity", lang), true, "50", discordgo.TextInputShort, 1, 12),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onCoinflipOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("casino", "coinflip_submit"),
		i18n.T("coinflip.legit_label", lang),
		components.TextInput("choice", i18n.T("coinflip.legit_label", lang), true, "heads/tails", discordgo.TextInputShort, 1, 10),
		components.TextInput("amount", i18n.T("economy.quantity", lang), true, "100", discordgo.TextInputShort, 1, 12),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onSlotsSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	values := interaction.ModalValues(i)

	amount, err := strconv.Atoi(strings.TrimSpace(values["amount"]))
	if err != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "coinflip.invalid_bet")
		return
	}

	res, serr := c.svc.SpinSlots(userID, amount)
	if serr != nil {
		switch serr {
		case casinosvc.ErrNoMoney:
			interaction.RespondError(b, i, lang, "slots.no_money")
		default:
			interaction.RespondError(b, i, lang, "coinflip.invalid_bet")
		}
		return
	}

	desc := res.Symbol1 + " | " + res.Symbol2 + " | " + res.Symbol3 + "\n\n"
	flavor := c.getSlotsFlavor(res.WinType, res.WinSym, lang)
	if res.IsWin {
		desc += flavor
		desc += "\n" + i18n.T("slots.gain", lang, map[string]any{"amount": res.Payout})
	} else {
		desc += flavor
		desc += "\n" + i18n.T("slots.loss", lang, map[string]any{"amount": amount})
	}

	color := 0xe74c3c
	if res.WinType == "JACKPOT" {
		color = 0xf1c40f
	} else if res.IsWin {
		color = 0x2ecc71
	}

	embed := components.Embed(i18n.T("slots.title", lang), desc, color)
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))

	unlocks, _ := achievement.CheckAndUnlock(b.DB, userID)
	if len(unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, unlocks)
	}
}

func (c *Cog) onCoinflipSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	values := interaction.ModalValues(i)

	choice := strings.TrimSpace(strings.ToLower(values["choice"]))
	amount, err := strconv.Atoi(strings.TrimSpace(values["amount"]))
	if err != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "coinflip.invalid_bet")
		return
	}

	res, cerr := c.svc.Coinflip(userID, choice, amount, false)
	if cerr != nil {
		switch cerr {
		case casinosvc.ErrNoMoney:
			interaction.RespondError(b, i, lang, "coinflip.no_money")
		case casinosvc.ErrChoice:
			interaction.RespondError(b, i, lang, "coinflip.choice_error")
		case casinosvc.ErrMaxBet:
			interaction.RespondError(b, i, lang, "coinflip.max_bet")
		default:
			interaction.RespondError(b, i, lang, "coinflip.invalid_bet")
		}
		return
	}

	var text string
	color := 0x2ecc71
	if res.Win {
		text = i18n.T("coinflip.win_msg", lang, map[string]any{"result": strings.ToUpper(res.Result)})
	} else {
		text = i18n.T("coinflip.lose_msg", lang, map[string]any{"result": strings.ToUpper(res.Result)})
		color = 0xe74c3c
	}

	embed := components.Embed(i18n.T("slots.title", lang), text, color)
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))

	unlocks, _ := achievement.CheckAndUnlock(b.DB, userID)
	if len(unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, unlocks)
	}
}

func (c *Cog) getSlotsFlavor(winType, symbol, lang string) string {
	if winType == "JACKPOT" {
		switch symbol {
		case "💎":
			return i18n.T("slots.jackpot_diamond", lang)
		case "🔔":
			return i18n.T("slots.jackpot_bell", lang)
		default:
			return i18n.T("slots.jackpot_generic", lang, map[string]any{"symbol": symbol})
		}
	}
	if winType == "PAIRE" {
		switch symbol {
		case "💎":
			return i18n.T("slots.pair_diamond", lang)
		case "🔔":
			return i18n.T("slots.pair_bell", lang)
		case "🍋":
			return i18n.T("slots.pair_lemon", lang)
		case "🍇":
			return i18n.T("slots.pair_grape", lang)
		case "🍒":
			return i18n.T("slots.pair_cherry", lang)
		}
	}
	return i18n.T("slots.lose_generic", lang)
}
