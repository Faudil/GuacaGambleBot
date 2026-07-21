package roulette

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	rlt "guacagamblebot/internal/service/roulette"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	games map[int64]*rlt.Game // serverID -> game
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, games: map[int64]*rlt.Game{}}
	r.Slash("roulette", "Roulette russe : rejoignez ou créez une partie.", c.onSlashMenu)
	r.Prefix("roulette", c.onPrefixMenu)
	r.Component("roulette", "new", c.onNewOpen)
	r.Component("roulette", "join", c.onJoin)
	r.Component("roulette", "start", c.onStart)
	r.Component("roulette", "trigger", c.onTrigger)
	r.Modal("roulette", "new_submit", c.onNewSubmit)
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
	game := c.games[serverID]
	desc := i18n.T("roulette.open_msg", lang, map[string]any{"amount": 0, "user": "?"})
	if game != nil {
		playerMentions := ""
		for _, p := range game.Players {
			playerMentions += interaction.Mention(p.UserID) + " "
		}
		desc = i18n.T("roulette.open_msg", lang, map[string]any{
			"amount": game.EntryFee,
			"user":   interaction.Mention(game.LeaderID),
		})
		if len(game.Players) > 0 {
			desc += "\n\n**Players:** " + playerMentions
		}
	}

	embed := components.Embed(
		i18n.T("roulette.finish_title", lang),
		desc,
		0xe74c3c,
	)

	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("roulette.join_label", lang), components.Encode("roulette", "new"), discordgo.PrimaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onNewOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("roulette", "new_submit"),
		i18n.T("roulette.open_msg", lang, map[string]any{"amount": "", "user": ""}),
		components.TextInput("entry_fee", i18n.T("economy.quantity", lang), true, "100", discordgo.TextInputShort, 1, 12),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onNewSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	serverID := interaction.ToInt64(i.GuildID)
	leaderID := interaction.ToInt64(interaction.UserID(i))
	values := interaction.ModalValues(i)

	entryFee, err := strconv.Atoi(strings.TrimSpace(values["entry_fee"]))
	if err != nil || entryFee <= 0 {
		interaction.RespondError(b, i, lang, "blackjack.invalid_bet")
		return
	}

	bal, err := c.store.GetBalance(leaderID)
	if err != nil || bal < entryFee {
		interaction.RespondError(b, i, lang, "roulette.no_money")
		return
	}

	if _, err := c.store.UpdateBalance(leaderID, -entryFee); err != nil {
		interaction.RespondError(b, i, lang, "roulette.no_money")
		return
	}

	game := rlt.NewGame(leaderID, entryFee)
	c.games[serverID] = game

	embed := components.Embed(
		i18n.T("roulette.finish_title", lang),
		i18n.T("roulette.open_msg", lang, map[string]any{
			"amount": entryFee,
			"user":   interaction.Mention(leaderID),
		})+"\n\n👤 "+interaction.Mention(leaderID),
		0xe74c3c,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("roulette.join_label", lang), components.Encode("roulette", "join"), discordgo.PrimaryButton),
			components.Button(i18n.T("roulette.start_label", lang), components.Encode("roulette", "start"), discordgo.SuccessButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onJoin(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	serverID := interaction.ToInt64(i.GuildID)
	userID := interaction.ToInt64(interaction.UserID(i))

	game, ok := c.games[serverID]
	if !ok || game == nil {
		interaction.RespondError(b, i, lang, "roulette.no_money")
		return
	}
	if game.IsActive {
		interaction.RespondError(b, i, lang, "roulette.already_joined")
		return
	}

	bal, err := c.store.GetBalance(userID)
	if err != nil || bal < game.EntryFee {
		interaction.RespondError(b, i, lang, "roulette.no_money")
		return
	}

	if err := game.AddPlayer(userID); err != nil {
		interaction.RespondError(b, i, lang, "roulette.already_joined")
		return
	}

	if _, err := c.store.UpdateBalance(userID, -game.EntryFee); err != nil {
		interaction.RespondError(b, i, lang, "roulette.no_money")
		return
	}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
			components.Embed(
				"",
				i18n.T("roulette.joined_msg", lang, map[string]any{"user": interaction.Mention(userID)}),
				0x2ecc71,
			),
			nil,
		))
}

func (c *Cog) onStart(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	serverID := interaction.ToInt64(i.GuildID)
	userID := interaction.ToInt64(interaction.UserID(i))

	game, ok := c.games[serverID]
	if !ok || game == nil {
		interaction.RespondError(b, i, lang, "roulette.no_money")
		return
	}

	if err := game.Start(userID); err != nil {
		switch err {
		case rlt.ErrNotLeader:
			interaction.RespondError(b, i, lang, "roulette.only_leader")
		case rlt.ErrMinPlayers:
			interaction.RespondError(b, i, lang, "roulette.min_players")
		default:
			interaction.RespondError(b, i, lang, "roulette.no_money")
		}
		return
	}

	cp := game.CurrentPlayer()
	embed := components.Embed(
		i18n.T("roulette.finish_title", lang),
		i18n.T("roulette.start_announcement", lang, map[string]any{"user": interaction.Mention(cp.UserID)}),
		0xe74c3c,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("roulette.trigger_label", lang), components.Encode("roulette", "trigger"), discordgo.DangerButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onTrigger(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	serverID := interaction.ToInt64(i.GuildID)
	userID := interaction.ToInt64(interaction.UserID(i))

	game, ok := c.games[serverID]
	if !ok || game == nil || !game.IsActive {
		interaction.RespondError(b, i, lang, "roulette.not_your_turn")
		return
	}

	ok, result, survivors, share := game.Trigger(userID)
	if !ok {
		interaction.RespondError(b, i, lang, "roulette.not_your_turn")
		return
	}

	if result == "dead" {
		_ = achievement.IncrementStat(b.DB, userID, "roulette_lost", 1)
		_ = achievement.IncrementStat(b.DB, userID, "roulette_spent", game.EntryFee)
		_ = achievement.IncrementStat(b.DB, userID, "roulette_money_lost", game.EntryFee)

		desc := i18n.T("roulette.bang_msg", lang, map[string]any{"user": interaction.Mention(userID)})
		desc += "\n" + i18n.T("roulette.survivors_win", lang, map[string]any{
			"user":  interaction.Mention(userID),
			"fee":   game.EntryFee,
			"count": len(survivors),
			"share": share,
		})

		for _, s := range survivors {
			if _, err := c.store.UpdateBalance(s.UserID, share); err != nil {
				continue
			}
			_ = achievement.IncrementStat(b.DB, s.UserID, "roulette_won", 1)
			_ = achievement.IncrementStat(b.DB, s.UserID, "roulette_spent", game.EntryFee)
			net := share - game.EntryFee
			if net > 0 {
				_ = achievement.IncrementStat(b.DB, s.UserID, "roulette_money_won", net)
			}
		}

		embed := components.Embed(i18n.T("roulette.finish_title", lang), desc, 0xe74c3c)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))

		for _, s := range survivors {
			unlocks, _ := achievement.CheckAndUnlock(b.DB, s.UserID)
			if len(unlocks) > 0 {
				interaction.SendAchievements(b, i, lang, unlocks)
			}
		}

		delete(c.games, serverID)
		return
	}

	cp := game.CurrentPlayer()
	desc := i18n.T("roulette.clic_msg", lang, map[string]any{"user": interaction.Mention(userID)})
	desc += "\n" + i18n.T("roulette.next_turn", lang, map[string]any{"user": interaction.Mention(cp.UserID)})

	embed := components.Embed(i18n.T("roulette.finish_title", lang), desc, 0x95a5a6)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("roulette.trigger_label", lang), components.Encode("roulette", "trigger"), discordgo.DangerButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}
