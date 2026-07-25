package delve

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

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

func (c *Cog) gauntletLeaderboard(guildID int64) *discordgo.MessageEmbed {
	weekStart := c.currentWeekStart()
	scores, err := c.store.GetDelveGauntletLeaderboard(guildID, weekStart, 10)
	if err != nil || len(scores) == 0 {
		return &discordgo.MessageEmbed{
			Title:       "🏆 Weekly Gauntlet",
			Description: "No scores yet this week. Use `!gauntlet start` to begin your run!",
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
		desc += fmt.Sprintf("%s <@%d> — Floor %d (Score: %d)\n", medal, s.UserID, s.Floor, s.Score)
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🏆 Weekly Gauntlet · %s", weekStart),
		Description: desc,
		Color:       0xf1c40f,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Everyone gets the same seed — pure skill!"},
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
	parts := strings.Fields(m.Content)
	if len(parts) > 1 && parts[1] == "start" {
		userID := interaction.ToInt64(m.Author.ID)
		if c.loadSession(userID) != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, "You already have an active run!")
			return
		}
		if err := c.gauntletStart(userID, interaction.ToInt64(m.GuildID)); err != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Failed to start gauntlet: %v", err))
			return
		}
		_, _ = s.ChannelMessageSend(m.ChannelID, "⚔️ **Weekly Gauntlet started!** Use `!delve` commands to navigate. Your seed is fixed for the week.")
		return
	}
	embed := c.gauntletLeaderboard(interaction.ToInt64(m.GuildID))
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
