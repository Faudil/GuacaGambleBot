package blackjack

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	bjsvc "guacagamblebot/internal/service/blackjack"
	"guacagamblebot/internal/store"
)

type pendingChallenge struct {
	ChallengerID int64
	Amount       int
}

type activeGame struct {
	State    *bjsvc.GameState
	MsgID    string
	Player1  int64
	Player2  int64
}

type Cog struct {
	store             *store.Store
	cfg               *config.Config
	svc               *bjsvc.Service
	pendingChallenges map[int64]pendingChallenge
	activeGames       map[int64]*activeGame
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{
		store:             s,
		cfg:               cfg,
		svc:               bjsvc.New(),
		pendingChallenges: map[int64]pendingChallenge{},
		activeGames:       map[int64]*activeGame{},
	}
	r.Slash("blackjack", "Blackjack PvP : défiez un joueur.", c.onSlashMenu)
	r.Slash("bj", "Blackjack PvP : défiez un joueur.", c.onSlashMenu)
	r.Prefix("blackjack", c.onPrefixMenu)
	r.Prefix("bj", c.onPrefixMenu)
	r.Component("blackjack", "challenge", c.onChallengeOpen)
	r.Component("blackjack", "accept", c.onAccept)
	r.Component("blackjack", "hit", c.onHit)
	r.Component("blackjack", "stand", c.onStand)
	r.Modal("blackjack", "challenge_submit", c.onChallengeSubmit)
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
		i18n.T("blackjack.title", lang),
		i18n.T("blackjack.challenge_msg", lang, map[string]any{"challenger": "?", "opponent": "?", "amount": 0}),
		0x3498db,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("blackjack.accept_label", lang), components.Encode("blackjack", "challenge"), discordgo.PrimaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onChallengeOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("blackjack", "challenge_submit"),
		i18n.T("blackjack.challenge_msg", lang, map[string]any{"challenger": "", "opponent": "", "amount": 0}),
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
	if !ok || opponentID == challengerID {
		interaction.RespondError(b, i, lang, "blackjack.invalid_opponent")
		return
	}

	amount, err := strconv.Atoi(strings.TrimSpace(values["amount"]))
	if err != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "blackjack.invalid_bet")
		return
	}

	cb, err := c.store.GetBalance(challengerID)
	if err != nil || cb < amount {
		interaction.RespondError(b, i, lang, "blackjack.no_money_self")
		return
	}
	ob, err := c.store.GetBalance(opponentID)
	if err != nil || ob < amount {
		interaction.RespondError(b, i, lang, "blackjack.no_money_opponent")
		return
	}

	c.pendingChallenges[opponentID] = pendingChallenge{ChallengerID: challengerID, Amount: amount}

	embed := components.Embed(
		i18n.T("blackjack.title", lang),
		i18n.T("blackjack.challenge_msg", lang, map[string]any{
			"challenger": interaction.Mention(challengerID),
			"opponent":   interaction.Mention(opponentID),
			"amount":     amount,
		}),
		0x3498db,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("blackjack.accept_label", lang), components.Encode("blackjack", "accept"), discordgo.SuccessButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onAccept(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	opponentID := interaction.ToInt64(interaction.UserID(i))
	pc, ok := c.pendingChallenges[opponentID]
	if !ok {
		interaction.RespondError(b, i, lang, "blackjack.no_money_problem")
		return
	}
	delete(c.pendingChallenges, opponentID)

	cb, err := c.store.GetBalance(pc.ChallengerID)
	if err != nil || cb < pc.Amount {
		interaction.RespondError(b, i, lang, "blackjack.no_money_problem")
		return
	}
	ob, err := c.store.GetBalance(opponentID)
	if err != nil || ob < pc.Amount {
		interaction.RespondError(b, i, lang, "blackjack.no_money_problem")
		return
	}

	ok, _, lerr := c.store.CheckGameLimit(pc.ChallengerID, "blackjack", 10)
	if lerr != nil {
		interaction.RespondError(b, i, lang, "blackjack.no_money_problem")
		return
	}
	if !ok {
		interaction.RespondError(b, i, lang, "blackjack.no_money_problem")
		return
	}

	if _, err := c.store.UpdateBalance(pc.ChallengerID, -pc.Amount); err != nil {
		interaction.RespondError(b, i, lang, "blackjack.no_money_problem")
		return
	}
	if _, err := c.store.UpdateBalance(opponentID, -pc.Amount); err != nil {
		_, _ = c.store.UpdateBalance(pc.ChallengerID, pc.Amount)
		interaction.RespondError(b, i, lang, "blackjack.no_money_problem")
		return
	}

	_ = c.store.IncrementGameLimit(pc.ChallengerID, "blackjack")

	gs := c.svc.NewGame(pc.ChallengerID, opponentID, pc.Amount)
	embed, comps := c.gameEmbed(gs, lang)

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onHit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	gs := c.findGame(userID)
	if gs == nil {
		interaction.RespondError(b, i, lang, "blackjack.not_your_turn")
		return
	}

	ok, bust := gs.Hit(userID)
	if !ok {
		interaction.RespondError(b, i, lang, "blackjack.not_your_turn")
		return
	}

	if bust {
		gs.Finished[userID] = true
		winnerID, reason, isDraw, over := gs.CheckGameOver()
		if over {
			c.endGame(b, i, gs, winnerID, reason, isDraw, lang)
			return
		}
	}

	embed, comps := c.gameEmbed(gs, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onStand(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	gs := c.findGame(userID)
	if gs == nil {
		interaction.RespondError(b, i, lang, "blackjack.not_your_turn")
		return
	}

	if !gs.Stand(userID) {
		interaction.RespondError(b, i, lang, "blackjack.not_your_turn")
		return
	}

	winnerID, reason, isDraw, over := gs.CheckGameOver()
	if over {
		c.endGame(b, i, gs, winnerID, reason, isDraw, lang)
		return
	}

	embed, comps := c.gameEmbed(gs, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) findGame(userID int64) *bjsvc.GameState {
	for _, g := range c.activeGames {
		if g.Player1 == userID || g.Player2 == userID {
			return g.State
		}
	}
	return nil
}

func (c *Cog) gameEmbed(gs *bjsvc.GameState, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	s1 := gs.Hands[gs.Player1ID].Score()
	s2 := gs.Hands[gs.Player2ID].Score()

	statusP1 := ""
	statusP2 := ""
	if gs.Turn == gs.Player1ID && !gs.Finished[gs.Player1ID] {
		statusP1 = i18n.T("blackjack.player_turn", lang, map[string]any{"user": ""})
		statusP2 = i18n.T("blackjack.player_waiting", lang, map[string]any{"user": ""})
	} else if gs.Turn == gs.Player2ID && !gs.Finished[gs.Player2ID] {
		statusP2 = i18n.T("blackjack.player_turn", lang, map[string]any{"user": ""})
		statusP1 = i18n.T("blackjack.player_waiting", lang, map[string]any{"user": ""})
	}

	desc := i18n.T("blackjack.pot", lang, map[string]any{"amount": gs.Amount * 2}) + "\n\n"
	desc += "👤 **" + interaction.Mention(gs.Player1ID) + "** " + statusP1 + "\n"
	desc += i18n.T("blackjack.hand", lang, map[string]any{"hand": gs.Hands[gs.Player1ID].Format(), "score": s1}) + "\n\n"
	desc += "👤 **" + interaction.Mention(gs.Player2ID) + "** " + statusP2 + "\n"
	desc += i18n.T("blackjack.hand", lang, map[string]any{"hand": gs.Hands[gs.Player2ID].Format(), "score": s2})

	color := 0x3498db
	if gs.Turn == gs.Player2ID {
		color = 0x9b59b6
	}

	embed := components.Embed(i18n.T("blackjack.title", lang), desc, color)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("blackjack.footer", lang, map[string]any{"user": interaction.Mention(gs.Turn)}),
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("blackjack.hit_label", lang), components.Encode("blackjack", "hit"), discordgo.SuccessButton),
			components.Button(i18n.T("blackjack.stand_label", lang), components.Encode("blackjack", "stand"), discordgo.DangerButton),
		),
	}

	return embed, comps
}

func (c *Cog) endGame(b *interaction.Bot, i *discordgo.InteractionCreate, gs *bjsvc.GameState, winnerID int64, reason string, isDraw bool, lang string) {
	delete(c.activeGames, gs.Player1ID)
	delete(c.activeGames, gs.Player2ID)

	embed, _ := c.gameEmbed(gs, lang)
	color := 0x95a5a6

	if isDraw {
		_, _ = c.store.UpdateBalance(gs.Player1ID, gs.Amount)
		_, _ = c.store.UpdateBalance(gs.Player2ID, gs.Amount)
		resultText := i18n.T("blackjack.draw_msg", lang, map[string]any{"reason": i18n.T("blackjack.draw", lang)})
		embed.Description += "\n\n" + i18n.T("blackjack.game_over", lang) + "\n" + resultText
	} else {
		color = 0xf1c40f
		loserID := gs.Player2ID
		if winnerID == gs.Player2ID {
			loserID = gs.Player1ID
		}
		_, _ = c.store.UpdateBalance(winnerID, gs.Amount*2)

		_ = achievement.IncrementStat(b.DB, winnerID, "blackjack_won", 1)
		_ = achievement.IncrementStat(b.DB, loserID, "blackjack_lost", 1)
		_ = achievement.IncrementStat(b.DB, winnerID, "blackjack_spent", gs.Amount)
		_ = achievement.IncrementStat(b.DB, winnerID, "blackjack_money_won", gs.Amount)
		_ = achievement.IncrementStat(b.DB, loserID, "blackjack_spent", gs.Amount)
		_ = achievement.IncrementStat(b.DB, loserID, "blackjack_money_lost", gs.Amount)

		var reasonText string
		if reason == "bust_p1" {
			reasonText = i18n.T("blackjack.bust", lang, map[string]any{"user": interaction.Mention(gs.Player1ID)})
		} else if reason == "bust_p2" {
			reasonText = i18n.T("blackjack.bust", lang, map[string]any{"user": interaction.Mention(gs.Player2ID)})
		} else {
			s1 := gs.Hands[gs.Player1ID].Score()
			s2 := gs.Hands[gs.Player2ID].Score()
			reasonText = i18n.T("blackjack.beat", lang, map[string]any{"s1": s1, "s2": s2})
		}

		resultText := i18n.T("blackjack.win_msg", lang, map[string]any{
			"user":   interaction.Mention(winnerID),
			"reason": reasonText,
			"amount": gs.Amount * 2,
		})
		embed.Description += "\n\n" + i18n.T("blackjack.game_over", lang) + "\n" + resultText
	}

	embed.Color = color
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))

	for _, uid := range []int64{gs.Player1ID, gs.Player2ID} {
		unlocks, _ := achievement.CheckAndUnlock(b.DB, uid)
		if len(unlocks) > 0 {
			interaction.SendAchievements(b, i, lang, unlocks)
		}
	}
}
