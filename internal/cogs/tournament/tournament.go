package tournament

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	tournamentsvc "guacagamblebot/internal/service/tournament"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store       *store.Store
	cfg         *config.Config
	svc         *tournamentsvc.Service
	tournaments map[int64]*tournamentsvc.TournamentState
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{
		store:       s,
		cfg:         cfg,
		svc:         tournamentsvc.New(s, cfg),
		tournaments: map[int64]*tournamentsvc.TournamentState{},
	}
	r.Slash("tournament", "cmd.tournament.desc", c.onSlash)
	r.Prefix("tournament", c.onPrefix)
	r.Prefix("tournoi", c.onPrefix)
}

func (c *Cog) onSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	c.handleSlash(i, b, lang)
}

func (c *Cog) handleSlash(i *discordgo.InteractionCreate, b *interaction.Bot, lang string) {
	userID := interaction.ToInt64(interaction.UserID(i))
	opts := i.ApplicationCommandData().Options
	sub := ""
	feeStr := ""
	for _, opt := range opts {
		switch opt.Name {
		case "action":
			sub = opt.StringValue()
		case "fee":
			feeStr = opt.StringValue()
		}
	}
	embed := c.handleAction(userID, interaction.ToInt64(i.GuildID), sub, feeStr, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	parts := strings.Fields(m.Content)
	sub := ""
	feeStr := ""
	if len(parts) > 1 {
		sub = parts[1]
	}
	if len(parts) > 2 {
		feeStr = parts[2]
	}
	embed := c.handleAction(interaction.ToInt64(m.Author.ID), interaction.ToInt64(m.GuildID), sub, feeStr, lang)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
}

func (c *Cog) handleAction(userID, serverID int64, sub, feeStr, lang string) *discordgo.MessageEmbed {
	switch sub {
	case "create":
		return c.create(userID, serverID, feeStr, lang)
	case "join":
		return c.join(userID, serverID, lang)
	case "start":
		return c.start(userID, serverID, lang)
	case "cancel":
		return c.cancel(userID, serverID, lang)
	default:
		return components.Embed(
			i18n.T("tournament.title", lang),
			i18n.T("tournament.desc", lang),
			components.ColorReward,
		)
	}
}

func (c *Cog) create(userID, serverID int64, feeStr, lang string) *discordgo.MessageEmbed {
	fee, err := strconv.Atoi(feeStr)
	if err != nil || fee < 0 {
		return components.Embed("❌", i18n.T("tournament.invalid_fee", lang), components.ColorDanger)
	}
	if _, ok := c.tournaments[serverID]; ok {
		return components.Embed("❌", i18n.T("tournament.already_prep", lang), components.ColorDanger)
	}
	bal, err := c.svc.GetBalance(userID)
	if err != nil || bal < fee {
		return components.Embed("❌", i18n.T("tournament.no_money", lang, map[string]any{"fee": fee}), components.ColorDanger)
	}
	if fee > 0 {
		_, _ = c.svc.UpdateBalance(userID, -fee)
	}
	c.tournaments[serverID] = &tournamentsvc.TournamentState{
		ServerID: serverID, CreatorID: userID, Fee: fee,
		Players: []tournamentsvc.TournamentPlayer{{UserID: userID}},
		Started: false,
	}
	return components.Embed(
		i18n.T("tournament.new_title", lang),
		i18n.T("tournament.new_desc", lang, map[string]any{"user": MentionUser(userID), "fee": fee}),
		components.ColorReward,
	)
}

func (c *Cog) join(userID, serverID int64, lang string) *discordgo.MessageEmbed {
	t, ok := c.tournaments[serverID]
	if !ok {
		return components.Embed("❌", i18n.T("tournament.not_found", lang), components.ColorDanger)
	}
	if t.Started {
		return components.Embed("❌", i18n.T("tournament.started_error", lang), components.ColorDanger)
	}
	for _, p := range t.Players {
		if p.UserID == userID {
			return components.Embed("❌", i18n.T("tournament.already_joined", lang), components.ColorDanger)
		}
	}
	bal, err := c.svc.GetBalance(userID)
	if err != nil || bal < t.Fee {
		return components.Embed("❌", i18n.T("tournament.join_no_money", lang, map[string]any{"fee": t.Fee}), components.ColorDanger)
	}
	pet, err := c.svc.GetActivePet(userID)
	if err != nil || pet == nil {
		return components.Embed("❌", i18n.T("tournament.join_no_pet", lang), components.ColorDanger)
	}
	if t.Fee > 0 {
		_, _ = c.svc.UpdateBalance(userID, -t.Fee)
	}
	t.Players = append(t.Players, tournamentsvc.TournamentPlayer{UserID: userID, Pet: pet})
	cashPrize := len(t.Players) * t.Fee
	return components.Embed(
		"✅",
		i18n.T("tournament.joined_msg", lang, map[string]any{"user": MentionUser(userID), "pet": pet.Nickname, "prize": cashPrize}),
		components.ColorSuccess,
	)
}

func (c *Cog) start(userID, serverID int64, lang string) *discordgo.MessageEmbed {
	t, ok := c.tournaments[serverID]
	if !ok {
		return components.Embed("❌", i18n.T("tournament.no_tournament", lang), components.ColorDanger)
	}
	if userID != t.CreatorID {
		return components.Embed("❌", i18n.T("tournament.only_creator_start", lang), components.ColorDanger)
	}
	if len(t.Players) < 2 {
		return components.Embed("❌", i18n.T("tournament.min_players", lang), components.ColorDanger)
	}
	t.Started = true
	tournamentsvc.ShufflePlayers(t.Players)
	cashPrize := len(t.Players) * t.Fee

	embed := components.Embed(
		i18n.T("tournament.started_title", lang),
		i18n.T("tournament.started_desc", lang, map[string]any{"count": len(t.Players), "prize": cashPrize}),
		components.ColorDanger,
	)
	delete(c.tournaments, serverID)
	return embed
}

func (c *Cog) cancel(userID, serverID int64, lang string) *discordgo.MessageEmbed {
	t, ok := c.tournaments[serverID]
	if !ok {
		return components.Embed("❌", i18n.T("tournament.no_tournament", lang), components.ColorDanger)
	}
	if userID != t.CreatorID {
		return components.Embed("❌", i18n.T("tournament.cancel_error", lang), components.ColorDanger)
	}
	if t.Started {
		return components.Embed("❌", i18n.T("tournament.cancel_too_late", lang), components.ColorDanger)
	}
	if t.Fee > 0 {
		for _, p := range t.Players {
			_, _ = c.svc.UpdateBalance(p.UserID, t.Fee)
		}
	}
	delete(c.tournaments, serverID)
	return components.Embed("🛑", i18n.T("tournament.cancelled_msg", lang), components.ColorWarning)
}

func MentionUser(id int64) string {
	return "<@" + strconv.FormatInt(id, 10) + ">"
}
