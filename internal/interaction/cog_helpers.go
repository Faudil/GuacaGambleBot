package interaction

import (
	"runtime/debug"
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

// goSafe runs a post-response side effect (notification follow-ups, journal
// scenes) in its own goroutine so the Discord round-trip never sits on the
// interaction handler's critical path: the user's main reply has already been
// sent, and these confirmations arrive a beat later either way. A panic in the
// detached goroutine is recovered and logged so it can never crash the process
// (the handler's own recover no longer covers it once it runs asynchronously).
func goSafe(name string, f func()) {
	go func() {
		defer func() {
			if v := recover(); v != nil {
				logger.Log().Error("panic recovered in async side effect",
					"task", name,
					"panic", v,
					"stack", string(debug.Stack()),
				)
			}
		}()
		f()
	}()
}

// RespondError replies to an interaction with a single ephemeral error/info
// message translated via the given locale key. Optional params are forwarded
// to the translation.
func RespondError(b *Bot, i *discordgo.InteractionCreate, lang, key string, params ...map[string]any) {
	err := b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T(key, lang, params...),
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

// achievementsPerEmbed bounds how many unlocked achievements share one
// follow-up embed: each entry is ~100 characters, and the description must stay
// under Discord's 4096-character limit or the notification is rejected.
const achievementsPerEmbed = 15

// SendAchievements posts follow-up embeds listing the newly unlocked
// achievements, chunked so large unlock batches can never overflow the embed
// description limit.
func SendAchievements(b *Bot, i *discordgo.InteractionCreate, lang string, unlocks []*achievement.Achievement) {
	if len(unlocks) == 0 {
		return
	}
	uid, gid := UserID(i), i.GuildID
	goSafe("achievement_notification", func() {
		for start := 0; start < len(unlocks); start += achievementsPerEmbed {
			end := min(start+achievementsPerEmbed, len(unlocks))
			desc := ""
			for _, a := range unlocks[start:end] {
				name := i18n.T("achievements."+a.ID+".name", lang)
				adesc := i18n.T("achievements."+a.ID+".desc", lang)
				glory := i18n.T("achievements.ui.new_achievement_glory", lang, map[string]any{"glory": a.Glory})
				desc += "🎖️ **" + name + "** " + a.Emoji + "\n" + glory + "\n" + adesc + "\n\n"
			}
			embed := components.Embed(i18n.T("achievements.ui.new_achievement_title", lang), desc, 0xf1c40f)
			if _, err := b.Session.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{Embeds: []*discordgo.MessageEmbed{embed}}); err != nil {
				logger.Log().Error("failed to send achievement unlock notification",
					"error", err,
					"user", uid,
					"guild", gid,
				)
			}
		}
	})
}

// SendQuestNotification posts a follow-up ephemeral embed (visible only to the
// player) about a quest event surfaced by RecordActivity.
func SendQuestNotification(b *Bot, i *discordgo.InteractionCreate, n store.QuestNotification, lang string) {
	if n.QuestID == "" {
		return
	}
	embed := components.Embed(i18n.T("quests.notification_title", lang), questssvc.QuestNotificationMsg(n, lang), 0x9b59b6)
	goSafe("quest_notification", func() {
		_, _ = b.Session.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		})
	})
}

// SendJournalScene delivers an atmospheric journal scene: as a private message
// when dm is set (falling back to an ephemeral follow-up when DMs fail or are
// blocked), otherwise as an ephemeral follow-up visible only to the player.
func SendJournalScene(b *Bot, i *discordgo.InteractionCreate, text string, dm bool) {
	if text == "" || b.Session == nil {
		return
	}
	uid := UserID(i)
	goSafe("journal_scene", func() {
		if dm && uid != "" {
			if ch, err := b.Session.UserChannelCreate(uid); err == nil {
				if _, err := b.Session.ChannelMessageSend(ch.ID, text); err == nil {
					return
				}
			}
		}
		embed := components.Embed("🕯️", text, 0x2c3e50)
		_, _ = b.Session.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		})
	})
}

// SendJournalSceneMsg delivers a journal scene in a prefix flow: a private
// message when dm is set (falling back to the channel), otherwise a direct
// channel message.
func SendJournalSceneMsg(s *discordgo.Session, channelID, userID, text string, dm bool) {
	if text == "" || s == nil {
		return
	}
	goSafe("journal_scene_msg", func() {
		if dm && userID != "" {
			if ch, err := s.UserChannelCreate(userID); err == nil {
				if _, err := s.ChannelMessageSend(ch.ID, text); err == nil {
					return
				}
			}
		}
		_, _ = s.ChannelMessageSend(channelID, text)
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
