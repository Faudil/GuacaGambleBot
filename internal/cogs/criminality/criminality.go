package criminality

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	crimsvc "guacagamblebot/internal/service/criminality"
	"guacagamblebot/internal/store"
)

// parseUserMention extracts a user ID from a Discord mention string like <@12345> or <@!12345>.
func parseUserMention(s string) int64 {
	clean := strings.Trim(s, "<@!> ")
	n, err := strconv.ParseInt(clean, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *crimsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{
		store: s,
		cfg:   cfg,
		svc:   crimsvc.New(s, cfg),
	}

	r.Prefix("steal", c.onPrefixSteal)
	r.Prefix("burgle", c.onPrefixBurgle)
	r.Prefix("bounty", c.onPrefixBounty)
	r.Prefix("hunt", c.onPrefixHunt)
	r.Prefix("track", c.onPrefixTrack)
	r.Prefix("report", c.onPrefixReport)
	r.Prefix("forgive", c.onPrefixForgive)
	r.Prefix("notoriety", c.onPrefixNotoriety)
	r.Prefix("sheriff", c.onPrefixSheriff)
	r.Prefix("whisper", c.onPrefixWhisper)
	r.Prefix("cleanse", c.onPrefixCleanse)
	r.Prefix("blessing", c.onPrefixBlessing)
	r.Prefix("chronicle", c.onPrefixChronicle)

	r.Slash("steal", "cmd.steal.desc", c.onSlashSteal)
	r.Slash("burgle", "cmd.burgle.desc", c.onSlashBurgle)
	r.Slash("bounty", "cmd.bounty.desc", c.onSlashBounty)
	r.Slash("crimhunt", "cmd.crimhunt.desc", c.onSlashHunt)
	r.Slash("track", "cmd.track.desc", c.onSlashTrack)
	r.Slash("report", "cmd.report.desc", c.onSlashReport)
	r.Slash("forgive", "cmd.forgive.desc", c.onSlashForgive)
	r.Slash("notoriety", "cmd.notoriety.desc", c.onSlashNotoriety)
	r.Slash("sheriff", "cmd.sheriff.desc", c.onSlashSheriff)
	r.Slash("whisper", "cmd.whisper.desc", c.onSlashWhisper)
}

func (c *Cog) requireAwake(b *interaction.Bot, i *discordgo.InteractionCreate, command, lang string) bool {
	userID := interaction.ToInt64(i.Member.User.ID)
	serverID := interaction.ToInt64(i.GuildID)
	allowed, msg, err := c.svc.IsCommandAllowed(userID, serverID, command, lang)
	if err != nil || !allowed {
		if err != nil {
			msg = err.Error()
		}
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return false
	}
	return true
}

func (c *Cog) requireAwakePrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message, command, lang string) bool {
	userID := interaction.ToInt64(m.Author.ID)
	serverID := interaction.ToInt64(m.GuildID)
	allowed, msg, err := c.svc.IsCommandAllowed(userID, serverID, command, lang)
	if err != nil || !allowed {
		if err != nil {
			msg = err.Error()
		}
		_, _ = s.ChannelMessageSend(m.ChannelID, msg)
		return false
	}
	return true
}

func (c *Cog) embedResponse(title, desc string, color int, footer string) *discordgo.MessageEmbed {
	e := &discordgo.MessageEmbed{
		Title:       title,
		Description: desc,
		Color:       color,
	}
	if footer != "" {
		e.Footer = &discordgo.MessageEmbedFooter{Text: footer}
	}
	return e
}

// -- Stub handlers (full implementations in later phases) --

func (c *Cog) onPrefixSteal(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	serverID := interaction.ToInt64(m.GuildID)
	lang := c.store.GetLanguage(serverID)
	allowed, msg, err := c.svc.IsCommandAllowed(userID, serverID, "steal", lang)
	if err != nil {
		msg = err.Error()
	}
	if !allowed {
		_, _ = s.ChannelMessageSend(m.ChannelID, msg)
		return
	}

	parts := strings.Fields(m.Content)
	if len(parts) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.steal.usage", lang))
		return
	}
	targetID := parseUserMention(parts[1])
	if targetID == 0 || targetID == userID {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.error.invalid_target", lang))
		return
	}

	result := c.svc.AttemptPickpocket(userID, targetID, serverID, lang)
	c.sendStealResult(s, m.ChannelID, result, lang)
}

func (c *Cog) onPrefixBurgle(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	serverID := interaction.ToInt64(m.GuildID)
	lang := c.store.GetLanguage(serverID)
	allowed, msg, err := c.svc.IsCommandAllowed(userID, serverID, "burgle", lang)
	if err != nil {
		msg = err.Error()
	}
	if !allowed {
		_, _ = s.ChannelMessageSend(m.ChannelID, msg)
		return
	}

	parts := strings.Fields(m.Content)
	if len(parts) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.burgle.usage", lang))
		return
	}
	targetID := parseUserMention(parts[1])
	if targetID == 0 || targetID == userID {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.error.invalid_target", lang))
		return
	}

	result := c.svc.AttemptBurgle(userID, targetID, serverID, lang)
	c.sendBurgleResult(s, m.ChannelID, result, lang)
}

func (c *Cog) onPrefixBounty(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	serverID := interaction.ToInt64(m.GuildID)
	lang := c.store.GetLanguage(serverID)
	allowed, msg, err := c.svc.IsCommandAllowed(userID, serverID, "bounty", lang)
	if err != nil {
		msg = err.Error()
	}
	if !allowed {
		_, _ = s.ChannelMessageSend(m.ChannelID, msg)
		return
	}

	parts := strings.Fields(m.Content)
	if len(parts) >= 3 {
		// !bounty @player amount
		targetID := parseUserMention(parts[1])
		if targetID == 0 || targetID == userID {
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.bounty.invalid_target", lang))
			return
		}
		amount, err := strconv.Atoi(parts[2])
		if err != nil || amount < 100 {
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.bounty.min_amount", lang))
			return
		}

		anonymous := len(parts) >= 4 && parts[3] == "anonymous"
		msg, err := c.svc.PlaceBounty(userID, targetID, amount, anonymous, lang)
		if err != nil {
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.bounty.failed", lang, map[string]any{"error": err.Error()}))
			return
		}
		_, _ = s.ChannelMessageSend(m.ChannelID, msg)
		return
	}

	// !bounty — list bounties
	listing := c.svc.ListBounties(lang)
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, c.embedResponse(
		i18n.T("criminality.bounty.board_title", lang), listing, components.ColorInfo,
		i18n.T("criminality.bounty.usage_list", lang)))
}

func (c *Cog) onPrefixHunt(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	serverID := interaction.ToInt64(m.GuildID)
	lang := c.store.GetLanguage(serverID)
	allowed, msg, err := c.svc.IsCommandAllowed(userID, serverID, "hunt", lang)
	if err != nil {
		msg = err.Error()
	}
	if !allowed {
		_, _ = s.ChannelMessageSend(m.ChannelID, msg)
		return
	}

	parts := strings.Fields(m.Content)
	if len(parts) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.hunt.usage", lang))
		return
	}

	// Check for "engage" subcommand
	if len(parts) >= 2 && strings.ToLower(parts[1]) == "engage" {
		if len(parts) < 3 {
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.hunt.usage_engage", lang))
			return
		}
		targetID := parseUserMention(parts[2])
		if targetID == 0 || targetID == userID {
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.hunt.invalid_target", lang))
			return
		}
		capture := len(parts) >= 4 && strings.ToLower(parts[3]) == "capture"
		result := c.svc.EngageHunt(userID, targetID, capture, lang)
		c.sendHuntResult(s, m.ChannelID, result, lang)
		return
	}

	targetID := parseUserMention(parts[1])
	if targetID == 0 || targetID == userID {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.hunt.invalid_target", lang))
		return
	}

	// Start hunt (tracking phase)
	msg, ok := c.svc.StartHunt(userID, targetID, serverID, lang)
	_, _ = s.ChannelMessageSend(m.ChannelID, msg)
	if ok {
		_ = ok
	}
}

func (c *Cog) onPrefixTrack(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	serverID := interaction.ToInt64(m.GuildID)
	lang := c.store.GetLanguage(serverID)
	parts := strings.Fields(m.Content)
	if len(parts) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.track.usage", lang))
		return
	}

	targetID := parseUserMention(parts[1])
	if targetID == 0 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.track.invalid_target", lang))
		return
	}

	msg, found := c.svc.TrackProgress(userID, targetID, lang)
	_, _ = s.ChannelMessageSend(m.ChannelID, msg)
	if found {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.hunt.track_engage_hint", lang, map[string]any{"mention": parts[1]}))
	}
}

func (c *Cog) onPrefixReport(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	serverID := interaction.ToInt64(m.GuildID)
	lang := c.store.GetLanguage(serverID)
	parts := strings.Fields(m.Content)
	if len(parts) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.report.usage", lang))
		return
	}
	thiefID := parseUserMention(parts[1])
	if thiefID == 0 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.report.invalid_target", lang))
		return
	}

	recs, err := c.store.GetTheftRecordsForVictim(userID)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.error.no_records", lang))
		return
	}
	for _, r := range recs {
		if r.ThiefID == thiefID {
			if r.Forgiven {
				_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.error.already_forgiven", lang))
				return
			}
			c.store.AddNotoriety(thiefID, 5)
			c.store.AddCrimeRecord(userID, "reported_thief", fmt.Sprintf(`{"thief_id":%d}`, thiefID))
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.report.success", lang, map[string]any{"thief": thiefID}))
			return
		}
	}
	_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.report.no_record", lang))
}

func (c *Cog) onPrefixForgive(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	serverID := interaction.ToInt64(m.GuildID)
	lang := c.store.GetLanguage(serverID)
	parts := strings.Fields(m.Content)
	if len(parts) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.forgive.usage", lang))
		return
	}
	thiefID := parseUserMention(parts[1])
	if thiefID == 0 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.forgive.invalid_target", lang))
		return
	}

	recs, err := c.store.GetTheftRecordsForVictim(userID)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.error.no_records", lang))
		return
	}
	for _, r := range recs {
		if r.ThiefID == thiefID {
			if r.Forgiven {
				_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.error.already_forgiven_short", lang))
				return
			}
			c.store.ForgiveTheft(r.ID)
			c.store.AddCrimeRecord(userID, "forgave_thief", fmt.Sprintf(`{"thief_id":%d}`, thiefID))
			_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.forgive.success", lang, map[string]any{"thief": thiefID}))
			return
		}
	}
	_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.forgive.no_record", lang))
}

func (c *Cog) onPrefixNotoriety(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	serverID := interaction.ToInt64(m.GuildID)
	lang := c.store.GetLanguage(serverID)

	// Apply passive decay
	_, _ = c.svc.DecayNotoriety(userID)

	crim, err := c.store.GetCriminality(userID)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("criminality.notoriety.failed", lang))
		return
	}
	embed := &discordgo.MessageEmbed{
		Title: i18n.T("criminality.notoriety.title", lang),
		Color: notorietyColor(crim.Notoriety),
		Fields: []*discordgo.MessageEmbedField{
			{Name: i18n.T("criminality.notoriety.notoriety_field", lang), Value: fmt.Sprintf("%d/100", crim.Notoriety), Inline: true},
			{Name: i18n.T("criminality.notoriety.alignment_field", lang), Value: crim.Alignment, Inline: true},
			{Name: i18n.T("criminality.notoriety.thief_rank_field", lang), Value: fmt.Sprintf("%d", crim.ThiefRank), Inline: true},
			{Name: i18n.T("criminality.notoriety.hunter_rank_field", lang), Value: fmt.Sprintf("%d", crim.HunterRank), Inline: true},
		},
	}
	if crim.PrisonUntil != nil && crim.PrisonUntil.After(time.Now()) {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  i18n.T("criminality.notoriety.prison_field", lang),
			Value: i18n.T("criminality.notoriety.prison_value", lang, map[string]any{"until": crim.PrisonUntil.Format("Jan 2 15:04")}),
		})
	}
	if crim.PacifistUntil != nil && crim.PacifistUntil.After(time.Now()) {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  i18n.T("criminality.notoriety.pacifist_field", lang),
			Value: i18n.T("criminality.notoriety.pacifist_value", lang, map[string]any{"until": crim.PacifistUntil.Format("Jan 2 15:04")}),
		})
	}
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// Chronicle — show criminality history
func (c *Cog) onPrefixChronicle(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	serverID := interaction.ToInt64(m.GuildID)
	lang := c.store.GetLanguage(serverID)
	records, err := c.store.GetCrimeRecords(userID)
	if err != nil || len(records) == 0 {
		_, _ = s.ChannelMessageSendEmbed(m.ChannelID, c.embedResponse(
			i18n.T("criminality.chronicle.title", lang),
			i18n.T("criminality.chronicle.empty", lang), components.ColorUnderworld, ""))
		return
	}

	eventKeys := []string{
		"stole", "was_stolen_from", "burgled", "mask_claimed",
		"hunter_sworn", "thief_sworn", "hunt_started", "hunt_won", "hunt_lost",
		"bounty_placed", "bounty_received", "clean_slate", "pacifist_blessing",
		"reported_thief", "forgave_thief", "forgave_first_thief",
		"awakening_first_theft", "awakening_first_victim",
	}
	eventLabels := make(map[string]string, len(eventKeys))
	for _, k := range eventKeys {
		eventLabels[k] = i18n.T("criminality.chronicle.event."+k, lang)
	}

	var lines []string
	for _, r := range records {
		label := eventLabels[r.Event]
		if label == "" {
			label = r.Event
		}
		lines = append(lines, fmt.Sprintf("%s — %s", r.CreatedAt.Format("Jan 2 15:04"), label))
	}

	if len(lines) > 20 {
		lines = lines[:20]
	}

	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
		Title:       i18n.T("criminality.chronicle.title", lang),
		Description: strings.Join(lines, "\n"),
		Color:       components.ColorUnderworld,
		Footer:      &discordgo.MessageEmbedFooter{Text: i18n.T("criminality.chronicle.footer", lang)},
	})
}

// Clean Slate — reset notoriety at a cost
func (c *Cog) onPrefixCleanse(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	msg, err := c.svc.ApplyCleanSlate(userID, lang)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, err.Error())
		return
	}
	_, _ = s.ChannelMessageSend(m.ChannelID, msg)
}

// Pacifist's Blessing — protection for 7 days
func (c *Cog) onPrefixBlessing(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	msg, err := c.svc.ApplyPacifistBlessing(userID, lang)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, err.Error())
		return
	}
	_, _ = s.ChannelMessageSend(m.ChannelID, msg)
}

// -- Slash stubs --

func (c *Cog) onSlashSteal(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if !c.requireAwake(b, i, "steal", lang) {
		return
	}
	// Slash steal would need options parsing; for now show interactive
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("criminality.steal.usage", lang),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) onSlashBurgle(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if !c.requireAwake(b, i, "burgle", lang) {
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("criminality.burgle.usage", lang),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) onSlashBounty(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if !c.requireAwake(b, i, "bounty", lang) {
		return
	}
	listing := c.svc.ListBounties(lang)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{c.embedResponse(
				i18n.T("criminality.bounty.board_title", lang), listing, components.ColorInfo,
				i18n.T("criminality.bounty.usage_list", lang))},
		},
	})
}

func (c *Cog) onSlashHunt(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if !c.requireAwake(b, i, "hunt", lang) {
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("criminality.hunt.usage", lang),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) onSlashTrack(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: i18n.T("criminality.track.not_implemented", lang), Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (c *Cog) onSlashReport(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: i18n.T("criminality.report.not_implemented", lang), Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (c *Cog) onSlashForgive(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: i18n.T("criminality.forgive.not_implemented", lang), Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (c *Cog) onSlashNotoriety(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	crim, err := c.store.GetCriminality(userID)
	if err != nil {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: i18n.T("criminality.notoriety.failed", lang),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	embed := &discordgo.MessageEmbed{
		Title: i18n.T("criminality.notoriety.title", lang),
		Color: notorietyColor(crim.Notoriety),
		Fields: []*discordgo.MessageEmbedField{
			{Name: i18n.T("criminality.notoriety.notoriety_field", lang), Value: fmt.Sprintf("%d/100", crim.Notoriety), Inline: true},
			{Name: i18n.T("criminality.notoriety.alignment_field", lang), Value: crim.Alignment, Inline: true},
			{Name: i18n.T("criminality.notoriety.thief_rank_field", lang), Value: fmt.Sprintf("%d", crim.ThiefRank), Inline: true},
			{Name: i18n.T("criminality.notoriety.hunter_rank_field", lang), Value: fmt.Sprintf("%d", crim.HunterRank), Inline: true},
		},
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
	})
}

func (c *Cog) sendStealResult(s *discordgo.Session, channelID string, result *crimsvc.StealResult, lang string) {
	color := components.ColorDanger
	if result.Success {
		color = components.ColorSuccess
	}
	embed := &discordgo.MessageEmbed{
		Title: result.Message,
		Color: color,
	}
	if result.NotorietyGain > 0 {
		embed.Description = i18n.T("criminality.steal.result_notoriety", lang, map[string]any{"amount": result.NotorietyGain})
	}
	if result.Success && result.GoldStolen > 0 {
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: "💰 Stolen", Value: i18n.T("criminality.steal.result_stolen", lang, map[string]any{"gold": result.GoldStolen}), Inline: true},
			{Name: "🌑 Infamy", Value: i18n.T("criminality.steal.result_infamy", lang, map[string]any{"amount": result.NotorietyGain}), Inline: true},
		}
	}
	_, _ = s.ChannelMessageSendEmbed(channelID, embed)
}

func (c *Cog) sendBurgleResult(s *discordgo.Session, channelID string, result *crimsvc.BurgleResult, lang string) {
	color := components.ColorDanger
	if result.Success {
		color = components.ColorSuccess
	}
	title := result.Message
	if !result.Success {
		title = i18n.T("criminality.burgle.fail_title", lang) + result.Message
	}
	embed := &discordgo.MessageEmbed{
		Title: title,
		Color: color,
	}
	if result.NotorietyGain > 0 {
		embed.Description = i18n.T("criminality.burgle.result_notoriety", lang, map[string]any{"amount": result.NotorietyGain})
	}
	if result.Success && result.ItemName != "" {
		embed.Fields = []*discordgo.MessageEmbedField{
			{Name: i18n.T("criminality.burgle.result_item", lang), Value: result.ItemName, Inline: true},
		}
	}
	_, _ = s.ChannelMessageSendEmbed(channelID, embed)
}

func (c *Cog) sendHuntResult(s *discordgo.Session, channelID string, result *crimsvc.HuntResult, lang string) {
	color := components.ColorDanger
	if result.Success {
		color = components.ColorSuccess
	}
	embed := &discordgo.MessageEmbed{
		Title: result.Message,
		Color: color,
	}
	if result.Success {
		desc := i18n.T("criminality.hunt.result_desc", lang, map[string]any{"merit": result.MeritGained, "gold": result.GoldReward})
		if result.Captured {
			desc += i18n.T("criminality.hunt.result_captured", lang)
		}
		embed.Description = desc
	}
	_, _ = s.ChannelMessageSendEmbed(channelID, embed)
}

func notorietyColor(n int) int {
	switch {
	case n >= 80:
		return components.ColorDanger // red
	case n >= 50:
		return components.ColorWarning // orange
	case n >= 20:
		return components.ColorReward // yellow
	default:
		return components.ColorSuccess // green
	}
}
