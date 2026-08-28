package blackjack

import (
	"strconv"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	bjsvc "guacagamblebot/internal/service/blackjack"
	charsvc "guacagamblebot/internal/service/character"
	"guacagamblebot/internal/store"
)

type pendingChallenge struct {
	ChallengerID int64
	Amount       int
}

type activeGame struct {
	State   *bjsvc.GameState
	MsgID   string
	Player1 int64
	Player2 int64
}

type Cog struct {
	store             *store.Store
	cfg               *config.Config
	svc               *bjsvc.Service
	mu                sync.RWMutex
	pendingChallenges map[int64]pendingChallenge
	activeGames       map[int64]*activeGame
}

var one = float64(1)

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{
		store:             s,
		cfg:               cfg,
		svc:               bjsvc.New(),
		pendingChallenges: map[int64]pendingChallenge{},
		activeGames:       map[int64]*activeGame{},
	}
	opts := []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        "opponent",
			Description: "Player to challenge to blackjack",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "amount",
			Description: "Bet amount per player",
			Required:    true,
			MinValue:    &one,
		},
	}
	r.SlashWithOptions("blackjack", "cmd.blackjack.desc", opts, c.onSlashChallenge)
	r.SlashWithOptions("bj", "cmd.blackjack.desc", opts, c.onSlashChallenge)
	r.Prefix("blackjack", c.onPrefixChallenge)
	r.Prefix("bj", c.onPrefixChallenge)
	r.Component("blackjack", "challenge", c.onChallengeOpen)
	r.Component("blackjack", "accept", c.onAccept)
	r.Component("blackjack", "deny", c.onDeny)
	r.Component("blackjack", "hit", c.onHit)
	r.Component("blackjack", "stand", c.onStand)
	r.Modal("blackjack", "challenge_submit", c.onChallengeSubmit)
}

func (c *Cog) onSlashChallenge(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	challengerID := interaction.ToInt64(interaction.UserID(i))

	opponentID, amount, ok := parseSlashOptions(i)
	if !ok {
		interaction.RespondError(b, i, lang, "blackjack.invalid_opponent")
		return
	}
	if opponentID == challengerID || opponentID == 0 {
		interaction.RespondError(b, i, lang, "blackjack.invalid_opponent")
		return
	}
	if amount <= 0 {
		interaction.RespondError(b, i, lang, "blackjack.invalid_bet")
		return
	}
	if err := c.createChallenge(challengerID, opponentID, amount, b, i, lang); err != nil {
		// createChallenge already responded with error
		return
	}
}

func parseSlashOptions(i *discordgo.InteractionCreate) (int64, int, bool) {
	var opponentID int64
	var amount int
	foundOpponent := false
	foundAmount := false
	for _, opt := range i.ApplicationCommandData().Options {
		switch opt.Name {
		case "opponent":
			foundOpponent = true
			// User option: Value is string snowflake, also Resolved
			if opt.Value != nil {
				if s, ok := opt.Value.(string); ok {
					opponentID = interaction.ToInt64(s)
				}
			}
			// Fallback via UserValue or StringValue
			if opponentID == 0 {
				if u := opt.UserValue(nil); u != nil {
					opponentID = interaction.ToInt64(u.ID)
				} else if s := opt.StringValue(); s != "" {
					opponentID = interaction.ToInt64(s)
				}
			}
			// Last fallback: check resolved map with opt.Value
			if opponentID == 0 && i.ApplicationCommandData().Resolved != nil && i.ApplicationCommandData().Resolved.Users != nil {
				for id := range i.ApplicationCommandData().Resolved.Users {
					opponentID = interaction.ToInt64(id)
					break
				}
			}
		case "amount":
			foundAmount = true
			amount = int(opt.IntValue())
			if amount == 0 && opt.Value != nil {
				switch v := opt.Value.(type) {
				case float64:
					amount = int(v)
				case int:
					amount = v
				case int64:
					amount = int(v)
				}
			}
		}
	}
	// Fallback: if options were positional without names (some discordgo versions), use index
	if !foundOpponent && len(i.ApplicationCommandData().Options) >= 2 {
		opts := i.ApplicationCommandData().Options
		// try index 0 as opponent
		if opponentID == 0 {
			if s, ok := opts[0].Value.(string); ok {
				opponentID = interaction.ToInt64(s)
			} else if u := opts[0].UserValue(nil); u != nil {
				opponentID = interaction.ToInt64(u.ID)
			} else {
				opponentID = interaction.ToInt64(opts[0].StringValue())
			}
		}
		if amount == 0 {
			amount = int(opts[1].IntValue())
		}
		foundOpponent = opponentID != 0
		foundAmount = amount != 0
	}
	return opponentID, amount, foundOpponent && foundAmount
}

func (c *Cog) onPrefixChallenge(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	parts := strings.Fields(m.Content)
	// parts[0] is "blackjack" or "bj", need at least 3: cmd, opponent, amount
	if len(parts) < 3 {
		// No args -> show help/menu
		embed, comps := c.menu(lang)
		_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: comps,
		})
		return
	}
	// Try to parse opponent from mention or raw id; amount is last token
	opponentID, ok := interaction.ParseUserID(parts[1])
	if !ok {
		// Also try to get from Discord mentions array
		if len(m.Mentions) > 0 {
			opponentID = interaction.ToInt64(m.Mentions[0].ID)
			ok = true
		}
	}
	if !ok {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("blackjack.invalid_opponent", lang))
		return
	}
	amountStr := parts[2]
	// allow amount to be second or third token if opponent mention contains space? Already split, amount is last
	if len(parts) > 3 {
		amountStr = parts[len(parts)-1]
	}
	amount, err := strconv.Atoi(strings.TrimSpace(amountStr))
	if err != nil || amount <= 0 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("blackjack.invalid_bet", lang))
		return
	}
	challengerID := interaction.ToInt64(m.Author.ID)
	if opponentID == challengerID {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("blackjack.invalid_opponent", lang))
		return
	}
	cb, err := c.store.GetBalance(challengerID)
	if err != nil || cb < amount {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("blackjack.no_money_self", lang))
		return
	}
	ob, err := c.store.GetBalance(opponentID)
	if err != nil || ob < amount {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("blackjack.no_money_opponent", lang))
		return
	}
	c.mu.Lock()
	c.pendingChallenges[opponentID] = pendingChallenge{ChallengerID: challengerID, Amount: amount}
	c.mu.Unlock()

	embed := components.Embed(
		i18n.T("blackjack.title", lang),
		i18n.T("blackjack.challenge_msg", lang, map[string]any{
			"challenger": interaction.Mention(challengerID),
			"opponent":   interaction.Mention(opponentID),
			"amount":     amount,
		}),
		components.ColorInfo,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("blackjack.accept_label", lang), components.Encode("blackjack", "accept"), discordgo.SuccessButton),
			components.Button(i18n.T("blackjack.deny_label", lang), components.Encode("blackjack", "deny"), discordgo.DangerButton),
		),
	}
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) createChallenge(challengerID, opponentID int64, amount int, b *interaction.Bot, i *discordgo.InteractionCreate, lang string) error {
	cb, err := c.store.GetBalance(challengerID)
	if err != nil || cb < amount {
		interaction.RespondError(b, i, lang, "blackjack.no_money_self")
		return errNoMoney
	}
	ob, err := c.store.GetBalance(opponentID)
	if err != nil || ob < amount {
		interaction.RespondError(b, i, lang, "blackjack.no_money_opponent")
		return errNoMoney
	}

	c.mu.Lock()
	c.pendingChallenges[opponentID] = pendingChallenge{ChallengerID: challengerID, Amount: amount}
	c.mu.Unlock()

	embed := components.Embed(
		i18n.T("blackjack.title", lang),
		i18n.T("blackjack.challenge_msg", lang, map[string]any{
			"challenger": interaction.Mention(challengerID),
			"opponent":   interaction.Mention(opponentID),
			"amount":     amount,
		}),
		components.ColorInfo,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("blackjack.accept_label", lang), components.Encode("blackjack", "accept"), discordgo.SuccessButton),
			components.Button(i18n.T("blackjack.deny_label", lang), components.Encode("blackjack", "deny"), discordgo.DangerButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
	return nil
}

var errNoMoney = &challengeError{}

type challengeError struct{}

func (e *challengeError) Error() string { return "no money" }

func (c *Cog) menu(lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := components.Embed(
		i18n.T("blackjack.title", lang),
		i18n.T("blackjack.challenge_msg", lang, map[string]any{"challenger": "?", "opponent": "?", "amount": 0}),
		components.ColorInfo,
	)
	embed.Description += "\n\n" + i18n.T("blackjack.howto", lang)
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

	if err := c.createChallenge(challengerID, opponentID, amount, b, i, lang); err != nil {
		return
	}
}

func (c *Cog) onAccept(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	opponentID := interaction.ToInt64(interaction.UserID(i))
	c.mu.Lock()
	pc, ok := c.pendingChallenges[opponentID]
	if !ok {
		c.mu.Unlock()
		interaction.RespondError(b, i, lang, "blackjack.no_money_problem")
		return
	}
	delete(c.pendingChallenges, opponentID)
	c.mu.Unlock()

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
	// Fix bug: store active game for both players so Hit/Stand can find it
	c.mu.Lock()
	ag := &activeGame{State: gs, Player1: pc.ChallengerID, Player2: opponentID}
	c.activeGames[pc.ChallengerID] = ag
	c.activeGames[opponentID] = ag
	c.mu.Unlock()

	embed, comps := c.gameEmbed(b, i, gs, lang)

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onDeny(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	opponentID := interaction.ToInt64(interaction.UserID(i))
	c.mu.Lock()
	if _, ok := c.pendingChallenges[opponentID]; ok {
		delete(c.pendingChallenges, opponentID)
		c.mu.Unlock()
		embed := components.Embed("", i18n.T("blackjack.deny_msg", lang, map[string]any{"user": interaction.Mention(opponentID)}), components.ColorMuted)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))
		return
	}
	c.mu.Unlock()
	interaction.RespondError(b, i, lang, "blackjack.no_money_problem")
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
		// Hit already set Finished, but keep for safety
		gs.Finished[userID] = true
		winnerID, reason, isDraw, over := gs.CheckGameOver()
		if over {
			c.endGame(b, i, gs, winnerID, reason, isDraw, lang)
			return
		}
	}

	embed, comps := c.gameEmbed(b, i, gs, lang)
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

	embed, comps := c.gameEmbed(b, i, gs, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) findGame(userID int64) *bjsvc.GameState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, g := range c.activeGames {
		if g.Player1 == userID || g.Player2 == userID {
			return g.State
		}
	}
	return nil
}

func (c *Cog) gameEmbed(b *interaction.Bot, i *discordgo.InteractionCreate, gs *bjsvc.GameState, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	s1 := gs.Hands[gs.Player1ID].Score()
	s2 := gs.Hands[gs.Player2ID].Score()

	statusP1 := ""
	statusP2 := ""
	if gs.Finished[gs.Player1ID] && gs.Finished[gs.Player2ID] {
		// both finished, no turn indicator
	} else if gs.Turn == gs.Player1ID && !gs.Finished[gs.Player1ID] {
		statusP1 = i18n.T("blackjack.player_turn", lang, map[string]any{"user": ""})
		statusP2 = i18n.T("blackjack.player_waiting", lang, map[string]any{"user": ""})
	} else if gs.Turn == gs.Player2ID && !gs.Finished[gs.Player2ID] {
		statusP2 = i18n.T("blackjack.player_turn", lang, map[string]any{"user": ""})
		statusP1 = i18n.T("blackjack.player_waiting", lang, map[string]any{"user": ""})
	} else if !gs.Finished[gs.Player1ID] {
		// Fallback: show waiting if Turn points to finished player
		statusP1 = i18n.T("blackjack.player_waiting", lang, map[string]any{"user": ""})
		statusP2 = i18n.T("blackjack.player_turn", lang, map[string]any{"user": ""})
	}

	desc := i18n.T("blackjack.pot", lang, map[string]any{"amount": gs.Amount * 2}) + "\n\n"
	desc += "👤 **" + interaction.Mention(gs.Player1ID) + "** " + statusP1 + "\n"
	desc += i18n.T("blackjack.hand", lang, map[string]any{"hand": gs.Hands[gs.Player1ID].Format(), "score": s1}) + "\n\n"
	desc += "👤 **" + interaction.Mention(gs.Player2ID) + "** " + statusP2 + "\n"
	desc += i18n.T("blackjack.hand", lang, map[string]any{"hand": gs.Hands[gs.Player2ID].Format(), "score": s2})

	color := components.ColorInfo
	if gs.Turn == gs.Player2ID {
		color = components.ColorArcane
	}

	embed := components.Embed(i18n.T("blackjack.title", lang), desc, color)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("blackjack.footer", lang, map[string]any{"user": interaction.DisplayName(b.Session, i.GuildID, i.Member, gs.Turn)}),
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
	c.mu.Lock()
	delete(c.activeGames, gs.Player1ID)
	delete(c.activeGames, gs.Player2ID)
	c.mu.Unlock()

	embed, _ := c.gameEmbed(b, i, gs, lang)
	color := components.ColorMuted

	if isDraw {
		_, _ = c.store.UpdateBalance(gs.Player1ID, gs.Amount)
		_, _ = c.store.UpdateBalance(gs.Player2ID, gs.Amount)
		resultText := i18n.T("blackjack.draw_msg", lang, map[string]any{"reason": i18n.T("blackjack.draw", lang)})
		embed.Description += "\n\n" + i18n.T("blackjack.game_over", lang) + "\n" + resultText
	} else {
		color = components.ColorReward
		loserID := gs.Player2ID
		if winnerID == gs.Player2ID {
			loserID = gs.Player1ID
		}

		payout := gs.Amount * 2
		fever := ""
		if charsvc.ConsumeBuff(c.store, winnerID, "jackpot_fever") {
			payout = gs.Amount * 3
			fever = "\n🔥 **Jackpot Fever!** The payout is tripled!"
		}
		_, _ = c.store.UpdateBalance(winnerID, payout)

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
			"amount": payout,
		}) + fever
		embed.Description += "\n\n" + i18n.T("blackjack.game_over", lang) + "\n" + resultText
	}

	embed.Color = color

	// Participation XP: both players earn XP and see their level-ups.
	for _, uid := range []int64{gs.Player1ID, gs.Player2ID} {
		leveled, lvl := charsvc.AddXP(c.store, uid, 10)
		if leveled {
			embed.Description += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": lvl})
		}
	}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))

	for _, uid := range []int64{gs.Player1ID, gs.Player2ID} {
		_ = c.store.RecordActivity(uid, "casino_games_played", 1)
		if n, ok := c.store.PopQuestNotification(uid); ok {
			interaction.SendQuestNotification(b, i, n, lang)
		}
		unlocks, _ := achievement.CheckAndUnlock(b.DB, uid)
		if len(unlocks) > 0 {
			interaction.SendAchievements(b, i, lang, unlocks)
		}
	}
}
