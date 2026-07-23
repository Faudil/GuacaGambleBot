package interaction

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/logger"
)

// RespondError replies to an interaction with a single ephemeral error/info
// message translated via the given locale key.
func RespondError(b *Bot, i *discordgo.InteractionCreate, lang, key string) {
	err := b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T(key, lang),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		logger.Log().Error("failed to send error response",
			"key", key,
			"user", UserID(i),
			"guild", i.GuildID,
			"error", err,
		)
	} else {
		logger.Log().Info("responded with error",
			"key", key,
			"user", UserID(i),
			"guild", i.GuildID,
		)
	}
}

// SendAchievements posts a follow-up embed listing the newly unlocked
// achievements.
func SendAchievements(b *Bot, i *discordgo.InteractionCreate, lang string, unlocks []*achievement.Achievement) {
	desc := ""
	for _, a := range unlocks {
		name := i18n.T("achievements."+a.ID+".name", lang)
		adesc := i18n.T("achievements."+a.ID+".desc", lang)
		glory := i18n.T("achievements.ui.new_achievement_glory", lang, map[string]any{"glory": a.Glory})
		desc += "🎖️ **" + name + "** " + a.Emoji + "\n" + glory + "\n" + adesc + "\n\n"
	}
	embed := components.Embed(i18n.T("achievements.ui.new_achievement_title", lang), desc, 0xf1c40f)
	_, _ = b.Session.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{Embeds: []*discordgo.MessageEmbed{embed}})
}

// Mention formats a Discord user mention from a numeric user id.
func Mention(id int64) string {
	return "<@" + strconv.FormatInt(id, 10) + ">"
}

// ToInt64 parses a snowflake string into an int64, returning 0 on failure.
func ToInt64(s string) int64 {
	id, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return id
}

// ParseUserID extracts a numeric user id from a mention, an @id or a raw id.
func ParseUserID(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "<@")
	s = strings.TrimPrefix(s, "@")
	s = strings.TrimPrefix(s, "!")
	s = strings.TrimSuffix(s, ">")
	s = strings.TrimSpace(s)
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}
