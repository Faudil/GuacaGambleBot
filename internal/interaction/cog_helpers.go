package interaction

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/logger"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
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

// NotYourMenu replies ephemerally that only the embed's owner may interact with
// the message. Returns true when the clicker is the owner, false otherwise.
func NotYourMenu(b *Bot, i *discordgo.InteractionCreate, lang string, ownerID int64) bool {
	uid := ToInt64(UserID(i))
	if uid == ownerID {
		return true
	}
	content := i18n.T("common.not_your_menu", lang)
	if ownerID > 0 {
		content = i18n.T("common.not_your_menu", lang, map[string]any{"user": Mention(ownerID)})
	}
	if b.Session != nil {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
	return false
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

// SendQuestNotification posts a follow-up ephemeral embed (visible only to the
// player) about a quest event surfaced by RecordActivity.
func SendQuestNotification(b *Bot, i *discordgo.InteractionCreate, n store.QuestNotification, lang string) {
	if n.QuestID == "" {
		return
	}
	embed := components.Embed(i18n.T("quests.notification_title", lang), questssvc.QuestNotificationMsg(n, lang), 0x9b59b6)
	_, _ = b.Session.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
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
