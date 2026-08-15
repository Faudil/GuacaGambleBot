package duel

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	duelsvc "guacagamblebot/internal/service/duel"
	"guacagamblebot/internal/store"
)

type pendingDuel struct {
	ChallengerID int64
	Amount       int
}

type Cog struct {
	store        *store.Store
	cfg          *config.Config
	svc          *duelsvc.Service
	pendingDuels map[int64]pendingDuel
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: duelsvc.New(s, cfg), pendingDuels: map[int64]pendingDuel{}}
	r.Slash("duel", "Provoque quelqu'un en duel (50/50).", c.onSlashMenu)
	r.Prefix("duel", c.onPrefixMenu)
	r.Prefix("du", c.onPrefixMenu)
	r.Component("duel", "challenge", c.onChallengeOpen)
	r.Component("duel", "accept", c.onAccept)
	r.Component("duel", "deny", c.onDeny)
	r.Modal("duel", "challenge_submit", c.onChallengeSubmit)
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
		i18n.T("duel.challenge_title", lang),
		i18n.T("duel.challenge_desc", lang, map[string]any{
			"challenger": "?",
			"opponent":   "?",
			"amount":     0,
			"user":       "?",
		}),
		0xe74c3c,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("⚔️ "+i18n.T("duel.result_title", lang), components.Encode("duel", "challenge"), discordgo.PrimaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onChallengeOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("duel", "challenge_submit"),
		i18n.T("duel.challenge_title", lang),
		components.TextInput("opponent", "Opponent mention", true, "@user", discordgo.TextInputShort, 1, 50),
		components.TextInput("amount", i18n.T("economy.quantity", lang), true, "100", discordgo.TextInputShort, 1, 12),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onChallengeSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	challengerID := interaction.ToInt64(interaction.UserID(i))
	values := interaction.ModalValues(i)

	opponentID, ok := interaction.ParseUserID(values["opponent"])
	if !ok {
		interaction.RespondError(b, i, lang, "duel.invalid_opponent")
		return
	}

	amount, err := strconv.Atoi(strings.TrimSpace(values["amount"]))
	if err != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "duel.invalid_bet")
		return
	}

	cb, err := c.store.GetBalance(challengerID)
	if err != nil || cb < amount {
		interaction.RespondError(b, i, lang, "duel.no_money_self")
		return
	}
	ob, err := c.store.GetBalance(opponentID)
	if err != nil || ob < amount {
		interaction.RespondError(b, i, lang, "duel.no_money_opponent")
		return
	}

	c.pendingDuels[opponentID] = pendingDuel{ChallengerID: challengerID, Amount: amount}

	embed := components.Embed(
		i18n.T("duel.challenge_title", lang),
		i18n.T("duel.challenge_desc", lang, map[string]any{
			"challenger": interaction.Mention(challengerID),
			"opponent":   interaction.Mention(opponentID),
			"amount":     amount,
			"user":       interaction.Mention(opponentID),
		}),
		0xe74c3c,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("✅ "+i18n.T("item_manager.accept_label", lang), components.Encode("duel", "accept"), discordgo.SuccessButton),
			components.Button("❌ "+i18n.T("item_manager.refuse_label", lang), components.Encode("duel", "deny"), discordgo.DangerButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onAccept(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	opponentID := interaction.ToInt64(interaction.UserID(i))
	pd, ok := c.pendingDuels[opponentID]
	if !ok {
		interaction.RespondError(b, i, lang, "duel.no_challenge")
		return
	}
	delete(c.pendingDuels, opponentID)

	cb, err := c.store.GetBalance(pd.ChallengerID)
	if err != nil || cb < pd.Amount {
		interaction.RespondError(b, i, lang, "duel.money_spent_cancel")
		return
	}
	ob, err := c.store.GetBalance(opponentID)
	if err != nil || ob < pd.Amount {
		interaction.RespondError(b, i, lang, "duel.money_spent_cancel")
		return
	}

	res, derr := c.svc.Duel(pd.ChallengerID, opponentID, pd.Amount)
	if derr != nil {
		interaction.RespondError(b, i, lang, "duel.money_spent_cancel")
		return
	}

	desc := i18n.T("duel.roll_msg", lang, map[string]any{
		"user":  interaction.Mention(res.ChallengerID),
		"die1":  res.Die1C,
		"die2":  res.Die2C,
		"total": res.TotalC,
	})
	desc += i18n.T("duel.roll_msg", lang, map[string]any{
		"user":  interaction.Mention(res.OpponentID),
		"die1":  res.Die1O,
		"die2":  res.Die2O,
		"total": res.TotalO,
	})

	color := 0xf1c40f
	if res.IsDraw {
		desc += i18n.T("duel.draw_msg", lang)
		color = 0x95a5a6
	} else {
		desc += i18n.T("duel.win_msg", lang, map[string]any{
			"user":   interaction.Mention(res.WinnerID),
			"amount": res.Amount,
		})
		color = 0xf1c40f
	}

	embed := components.Embed(i18n.T("duel.result_title", lang), desc, color)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))

	for _, uid := range []int64{pd.ChallengerID, opponentID} {
		unlocks, _ := achievement.CheckAndUnlock(b.DB, uid)
		if len(unlocks) > 0 {
			interaction.SendAchievements(b, i, lang, unlocks)
		}
	}
}

func (c *Cog) onDeny(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	opponentID := interaction.ToInt64(interaction.UserID(i))
	if _, ok := c.pendingDuels[opponentID]; ok {
		delete(c.pendingDuels, opponentID)
		embed := components.Embed("", i18n.T("duel.deny_msg", lang, map[string]any{"user": interaction.Mention(opponentID)}), 0x95a5a6)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))
	} else {
		interaction.RespondError(b, i, lang, "duel.no_deny")
	}
}
