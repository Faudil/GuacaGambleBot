package delve

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
)

func (c *Cog) currentWeekStart() string {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1))
	return monday.Format("2006-01-02")
}

func (c *Cog) gauntletLeaderboard(guildID int64, lang string) *discordgo.MessageEmbed {
	weekStart := c.currentWeekStart()
	scores, err := c.store.GetDelveGauntletLeaderboard(guildID, weekStart, 10)
	if err != nil || len(scores) == 0 {
		return &discordgo.MessageEmbed{
			Title:       i18n.T("delve.gauntlet.title", lang),
			Description: i18n.T("delve.gauntlet.empty", lang),
			Color:       0xf1c40f,
		}
	}

	desc := ""
	for i, s := range scores {
		medal := ""
		switch i {
		case 0:
			medal = "🥇"
		case 1:
			medal = "🥈"
		case 2:
			medal = "🥉"
		default:
			medal = fmt.Sprintf("#%d", i+1)
		}
		desc += i18n.T("delve.gauntlet.line", lang, map[string]any{
			"medal": medal, "user": fmt.Sprintf("%d", s.UserID),
			"floor": fmt.Sprintf("%d", s.Floor), "score": fmt.Sprintf("%d", s.Score),
		}) + "\n"
	}

	return &discordgo.MessageEmbed{
		Title:       i18n.T("delve.gauntlet.title_date", lang, map[string]any{"date": weekStart}),
		Description: desc,
		Color:       0xf1c40f,
		Footer:      &discordgo.MessageEmbedFooter{Text: i18n.T("delve.gauntlet.footer", lang)},
	}
}

func (c *Cog) gauntletStart(userID, guildID int64) error {
	weekStart := c.currentWeekStart()
	seed := hashString(weekStart)

	session, err := c.svc.StartSession(userID, guildID, 0)
	if err != nil {
		return err
	}
	c.saveSession(session)
	return c.store.SaveDelveGauntletScore(&model.DelveGauntletScore{
		UserID:    userID,
		GuildID:   guildID,
		WeekStart: weekStart,
		Seed:      seed,
	})
}

func (c *Cog) onPrefixGauntlet(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	guildID := interaction.ToInt64(m.GuildID)
	lang := c.store.GetLanguage(guildID)
	parts := strings.Fields(m.Content)
	if len(parts) > 1 && parts[1] == "start" {
		userID := interaction.ToInt64(m.Author.ID)
		if c.loadSession(userID) != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("delve.gauntlet.active", lang))
			return
		}
		if err := c.gauntletStart(userID, guildID); err != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("delve.gauntlet.failed", lang, map[string]any{"err": err.Error()}))
			return
		}
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("delve.gauntlet.started", lang))
		return
	}
	embed := c.gauntletLeaderboard(guildID, lang)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
}

func hashString(s string) string {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	return fmt.Sprintf("%d", h)
}
