package criminality

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
)

func (c *Cog) onPrefixSheriff(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	serverID := interaction.ToInt64(m.GuildID)
	lang := c.store.GetLanguage(serverID)

	awake, err := c.store.IsAwakened(serverID)
	if err != nil || !awake {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.quest.sheriff_not_awake", lang))
		return
	}

	// Check for sub-commands: "!sheriff join"
	parts := strings.Fields(m.Content)
	if len(parts) >= 2 {
		switch parts[1] {
		case "join":
			c.onPrefixSheriffJoin(b, s, m)
			return
		}
	}

	embed := c.sheriffMenu(userID, lang)
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func (c *Cog) onPrefixWhisper(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	serverID := interaction.ToInt64(m.GuildID)
	lang := c.store.GetLanguage(serverID)

	awake, err := c.store.IsAwakened(serverID)
	if err != nil || !awake {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.quest.whisper_not_awake", lang))
		return
	}

	// Check for sub-commands: "!whisper join"
	parts := strings.Fields(m.Content)
	if len(parts) >= 2 {
		switch parts[1] {
		case "join":
			c.onPrefixWhisperJoin(b, s, m)
			return
		case "forgive":
			c.onPrefixForgivePath(b, s, m)
			return
		}
	}

	embed := c.whisperMenu(userID, lang)
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func (c *Cog) onSlashSheriff(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed := c.sheriffMenu(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}

func (c *Cog) onSlashWhisper(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed := c.whisperMenu(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}

func (c *Cog) sheriffMenu(userID int64, lang string) *discordgo.MessageEmbed {
	crim, err := c.store.GetCriminality(userID)
	if err != nil {
		return &discordgo.MessageEmbed{
			Title:       i18n.T("criminality.quest.sheriff_menu_title", lang),
			Description: i18n.T("criminality.quest.sheriff_error", lang),
			Color:       0x3498db,
		}
	}

	if crim.Alignment == "hunter" {
		return &discordgo.MessageEmbed{
			Title:       i18n.T("criminality.quest.sheriff_hunter_title", lang),
			Description: i18n.T("criminality.quest.sheriff_hunter_desc", lang),
			Color:       0x3498db,
		}
	}

	if crim.Alignment == "thief" {
		return &discordgo.MessageEmbed{
			Title:       i18n.T("criminality.quest.sheriff_menu_title", lang),
			Description: i18n.T("criminality.quest.sheriff_thief_desc", lang),
			Color:       0xe74c3c,
		}
	}

	hasQuest, _, _ := c.store.GetUserQuest(userID, "masked_shadow_falls_hunter")
	if hasQuest != nil {
		return &discordgo.MessageEmbed{
			Title:       i18n.T("criminality.quest.sheriff_questing_title", lang),
			Description: i18n.T("criminality.quest.sheriff_questing_desc", lang),
			Color:       0x3498db,
		}
	}

	return &discordgo.MessageEmbed{
		Title:       i18n.T("criminality.quest.sheriff_menu_title", lang),
		Description: i18n.T("criminality.quest.sheriff_default_desc", lang),
		Color:       0x3498db,
	}
}

func (c *Cog) whisperMenu(userID int64, lang string) *discordgo.MessageEmbed {
	crim, err := c.store.GetCriminality(userID)
	if err != nil {
		return &discordgo.MessageEmbed{
			Title:       i18n.T("criminality.quest.whisper_menu_title", lang),
			Description: i18n.T("criminality.quest.whisper_error", lang),
			Color:       0x8e44ad,
		}
	}

	if crim.Alignment == "thief" {
		return &discordgo.MessageEmbed{
			Title:       i18n.T("criminality.quest.whisper_thief_title", lang),
			Description: i18n.T("criminality.quest.whisper_thief_desc", lang),
			Color:       0x8e44ad,
		}
	}

	if crim.Alignment == "hunter" {
		return &discordgo.MessageEmbed{
			Title:       i18n.T("criminality.quest.whisper_menu_title", lang),
			Description: i18n.T("criminality.quest.whisper_hunter_desc", lang),
			Color:       0x8e44ad,
		}
	}

	hasQuest, _, _ := c.store.GetUserQuest(userID, "masked_shadow_falls_shadow")
	if hasQuest != nil {
		return &discordgo.MessageEmbed{
			Title:       i18n.T("criminality.quest.whisper_questing_title", lang),
			Description: i18n.T("criminality.quest.whisper_questing_desc", lang),
			Color:       0x8e44ad,
		}
	}

	return &discordgo.MessageEmbed{
		Title:       i18n.T("criminality.quest.whisper_menu_title", lang),
		Description: i18n.T("criminality.quest.whisper_default_desc", lang),
		Color:       0x8e44ad,
	}
}

func (c *Cog) onPrefixSheriffJoin(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))

	existing, _, _ := c.store.GetUserQuest(userID, "masked_shadow_falls_hunter")
	if existing != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.quest.sheriff_join_already", lang))
		return
	}

	crim, _ := c.store.GetCriminality(userID)
	if crim.Alignment == "hunter" {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.quest.sheriff_join_already_hunter", lang))
		return
	}
	if crim.Alignment == "thief" {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.quest.sheriff_join_thief", lang))
		return
	}

	if err := c.store.CreateQuest(userID, "masked_shadow_falls_hunter"); err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.error.failed_start_quest", lang, map[string]any{"error": err.Error()}))
		return
	}

	c.store.UpdateCriminality(userID, map[string]any{"alignment": "hunter"})
	c.store.AddDelveFlag(userID, "hunter_sworn", `{"source":"sheriff_vance"}`)
	c.store.AddCrimeRecord(userID, "hunter_sworn", `{"npc":"sheriff_vance"}`)

	embed := &discordgo.MessageEmbed{
		Title:       i18n.T("criminality.quest.sheriff_join_success_title", lang),
		Description: i18n.T("criminality.quest.sheriff_join_success_desc", lang),
		Color:       0x3498db,
		Footer:      &discordgo.MessageEmbedFooter{Text: i18n.T("criminality.quest.sheriff_join_success_footer", lang)},
	}
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func (c *Cog) onPrefixWhisperJoin(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))

	existing, _, _ := c.store.GetUserQuest(userID, "masked_shadow_falls_shadow")
	if existing != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.quest.whisper_join_already", lang))
		return
	}

	crim, _ := c.store.GetCriminality(userID)
	if crim.Alignment == "thief" {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.quest.whisper_join_already_thief", lang))
		return
	}
	if crim.Alignment == "hunter" {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.quest.whisper_join_hunter", lang))
		return
	}

	if err := c.store.CreateQuest(userID, "masked_shadow_falls_shadow"); err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.error.failed_start_quest", lang, map[string]any{"error": err.Error()}))
		return
	}

	c.store.UpdateCriminality(userID, map[string]any{"alignment": "thief"})
	c.store.AddDelveFlag(userID, "thief_sworn", `{"source":"the_whisper"}`)
	c.store.AddCrimeRecord(userID, "thief_sworn", `{"npc":"the_whisper"}`)

	embed := &discordgo.MessageEmbed{
		Title:       i18n.T("criminality.quest.whisper_join_success_title", lang),
		Description: i18n.T("criminality.quest.whisper_join_success_desc", lang),
		Color:       0x8e44ad,
		Footer:      &discordgo.MessageEmbedFooter{Text: i18n.T("criminality.quest.whisper_join_success_footer", lang)},
	}
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// Forgiveness path — the original victim can choose this via the victim-specific dialog.
func (c *Cog) onPrefixForgivePath(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))

	existing, _, _ := c.store.GetUserQuest(userID, "masked_shadow_falls_forgive")
	if existing != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.quest.forgive_already", lang))
		return
	}

	// Check if this player was the first victim
	ws, err := c.store.GetWorldState(interaction.ToInt64(m.GuildID))
	if err != nil || ws.FirstVictimID == nil || *ws.FirstVictimID != userID {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.quest.forgive_not_victim", lang))
		return
	}

	if err := c.store.CreateQuest(userID, "masked_shadow_falls_forgive"); err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.error.failed_start_quest", lang, map[string]any{"error": err.Error()}))
		return
	}

	c.store.AddDelveFlag(userID, "forgave_first_thief", fmt.Sprintf(`{"guild_id":%d}`, interaction.ToInt64(m.GuildID)))
	c.store.AddCrimeRecord(userID, "forgave_first_thief", `{}`)

	embed := &discordgo.MessageEmbed{
		Title:       i18n.T("criminality.quest.forgive_success_title", lang),
		Description: i18n.T("criminality.quest.forgive_success_desc", lang),
		Color:       0xf1c40f,
	}
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}
