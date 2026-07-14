package onboarding

import (
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

// Cog drives the per-server onboarding experience: when the bot joins a guild
// it posts a configuration menu (channel, language, prefix, enabled) and the
// same menu can be reopened at any time with the `/setup` command.
type Cog struct {
	store *store.Store
	cfg   *config.Config
}

// Register wires the onboarding cog into the router and listens for the bot
// being added to a guild.
func Register(r *interaction.Router, st *store.Store, cfg *config.Config) {
	c := &Cog{store: st, cfg: cfg}
	r.Slash("setup", "Configure the bot for this server.", c.onSetupSlash)
	r.Prefix("setup", c.onSetupPrefix)
	r.Component("onboarding", "channel", c.onChannelSelect)
	r.Component("onboarding", "language", c.onLanguageSelect)
	r.Component("onboarding", "advanced", c.onAdvanced)
	r.Modal("onboarding", "advanced_submit", c.onAdvancedSubmit)
	r.Component("onboarding", "toggle", c.onToggle)
	r.Component("onboarding", "finish", c.onFinish)
	r.Component("onboarding", "reconfigure", c.onReconfigure)

	r.Session().AddHandler(c.onGuildCreate)
}

// onGuildCreate fires for every guild the bot is in (including on (re)connect).
// We only post the onboarding menu once per guild, tracked via OnboardedAt.
func (c *Cog) onGuildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	if g.Unavailable {
		return
	}
	gid := interaction.ToInt64(g.ID)
	if gid == 0 {
		return
	}
	ss, _ := c.store.GetServerSetting(gid)
	if ss != nil && ss.OnboardedAt != nil {
		return
	}
	lang := c.store.GetLanguage(gid)
	embed, comps := c.menu(lang, ss)

	sent := false
	if target := c.pickChannel(g); target != "" {
		if _, err := s.ChannelMessageSendComplex(target, &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: comps,
		}); err == nil {
			sent = true
		}
	}
	if !sent && g.OwnerID != "" {
		if ch, err := s.UserChannelCreate(g.OwnerID); err == nil {
			_, _ = s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: comps,
			})
		}
	}

	// Persist the onboarded marker so we don't re-post on the next connect.
	if ss == nil {
		ss = &model.ServerSetting{ServerID: gid, Enabled: true}
	}
	if ss.Language == "" {
		ss.Language = "fr"
	}
	now := time.Now()
	ss.OnboardedAt = &now
	_ = c.store.SaveServerSetting(ss)
}

// pickChannel returns a text channel the bot can post the onboarding message
// to: the system channel first, then the first guild text channel.
func (c *Cog) pickChannel(g *discordgo.GuildCreate) string {
	if g.SystemChannelID != "" {
		return g.SystemChannelID
	}
	for _, ch := range g.Channels {
		if ch.Type == discordgo.ChannelTypeGuildText {
			return ch.ID
		}
	}
	return ""
}

func (c *Cog) onSetupSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	gid := interaction.ToInt64(i.GuildID)
	lang := c.store.GetLanguage(gid)
	embed, comps := c.menu(lang, c.current(gid))
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onSetupPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	gid := interaction.ToInt64(m.GuildID)
	lang := c.store.GetLanguage(gid)
	embed, comps := c.menu(lang, c.current(gid))
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) onChannelSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	gid := interaction.ToInt64(i.GuildID)
	lang := c.store.GetLanguage(gid)
	for _, v := range i.MessageComponentData().Values {
		if cid := interaction.ToInt64(v); cid != 0 {
			c.save(gid, func(ss *model.ServerSetting) { ss.ChannelID = cid })
		}
	}
	embed, comps := c.menu(lang, c.current(gid))
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onLanguageSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	gid := interaction.ToInt64(i.GuildID)
	lang := c.store.GetLanguage(gid)
	for _, v := range i.MessageComponentData().Values {
		if v == "en" || v == "fr" {
			c.save(gid, func(ss *model.ServerSetting) { ss.Language = v })
		}
	}
	embed, comps := c.menu(lang, c.current(gid))
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onAdvanced(b *interaction.Bot, i *discordgo.InteractionCreate) {
	gid := interaction.ToInt64(i.GuildID)
	lang := c.store.GetLanguage(gid)
	ss := c.current(gid)
	modal := components.ModalResponse(
		components.Encode("onboarding", "advanced_submit"),
		i18n.T("onboarding.advanced_title", lang),
		components.TextInput("prefix", i18n.T("onboarding.prefix_label", lang), true, c.prefixLabel(ss), discordgo.TextInputShort, 1, 5),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onAdvancedSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	gid := interaction.ToInt64(i.GuildID)
	lang := c.store.GetLanguage(gid)
	values := interaction.ModalValues(i)
	if p := strings.TrimSpace(values["prefix"]); p != "" {
		c.save(gid, func(ss *model.ServerSetting) { ss.Prefix = p })
	}
	embed, comps := c.menu(lang, c.current(gid))
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onToggle(b *interaction.Bot, i *discordgo.InteractionCreate) {
	gid := interaction.ToInt64(i.GuildID)
	lang := c.store.GetLanguage(gid)
	c.save(gid, func(ss *model.ServerSetting) { ss.Enabled = !ss.Enabled })
	embed, comps := c.menu(lang, c.current(gid))
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onFinish(b *interaction.Bot, i *discordgo.InteractionCreate) {
	gid := interaction.ToInt64(i.GuildID)
	lang := c.store.GetLanguage(gid)
	ss := c.current(gid)
	if ss.ChannelID == 0 {
		interaction.RespondError(b, i, lang, "onboarding.need_channel")
		return
	}
	embed := components.Embed(
		i18n.T("onboarding.configured_title", lang),
		i18n.T("onboarding.configured_desc", lang, map[string]any{"channel": "<#"+strconv.FormatInt(ss.ChannelID, 10)+">"}),
		0x57f287,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("onboarding.btn_reconfigure", lang), components.Encode("onboarding", "reconfigure"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onReconfigure(b *interaction.Bot, i *discordgo.InteractionCreate) {
	gid := interaction.ToInt64(i.GuildID)
	lang := c.store.GetLanguage(gid)
	embed, comps := c.menu(lang, c.current(gid))
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// menu builds the configuration embed and its interactive components.
func (c *Cog) menu(lang string, ss *model.ServerSetting) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	if ss == nil {
		ss = &model.ServerSetting{}
	}
	embed := components.Embed(
		i18n.T("onboarding.menu_title", lang),
		i18n.T("onboarding.menu_desc", lang),
		0x5865f2,
	)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("onboarding.field_channel", lang), c.channelLabel(ss.ChannelID), true),
		components.Field(i18n.T("onboarding.field_language", lang), ss.Language, true),
		components.Field(i18n.T("onboarding.field_prefix", lang), c.prefixLabel(ss), true),
		components.Field(i18n.T("onboarding.field_status", lang), c.statusLabel(ss, lang), true),
	}

	channelSelect := discordgo.SelectMenu{
		MenuType:     discordgo.ChannelSelectMenu,
		CustomID:     components.Encode("onboarding", "channel"),
		Placeholder:  i18n.T("onboarding.channel_placeholder", lang),
		MinValues:    intPtr(0),
		MaxValues:    1,
		ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
	}
	if ss.ChannelID != 0 {
		channelSelect.DefaultValues = []discordgo.SelectMenuDefaultValue{{
			ID:   strconv.FormatInt(ss.ChannelID, 10),
			Type: discordgo.SelectMenuDefaultValueChannel,
		}}
	}

	langSelect := discordgo.SelectMenu{
		MenuType:    discordgo.StringSelectMenu,
		CustomID:    components.Encode("onboarding", "language"),
		Placeholder: i18n.T("onboarding.language_placeholder", lang),
		MinValues:   intPtr(1),
		MaxValues:   1,
		Options: []discordgo.SelectMenuOption{
			{Label: "English", Value: "en", Default: ss.Language == "en"},
			{Label: "Français", Value: "fr", Default: ss.Language == "fr"},
		},
	}

	toggleLabel := i18n.T("onboarding.btn_disable", lang)
	toggleStyle := discordgo.DangerButton
	if !ss.Enabled {
		toggleLabel = i18n.T("onboarding.btn_enable", lang)
		toggleStyle = discordgo.SuccessButton
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(channelSelect),
		components.ActionRow(langSelect),
		components.ActionRow(
			components.Button(i18n.T("onboarding.btn_advanced", lang), components.Encode("onboarding", "advanced"), discordgo.SecondaryButton),
			components.Button(toggleLabel, components.Encode("onboarding", "toggle"), toggleStyle),
			components.Button(i18n.T("onboarding.btn_finish", lang), components.Encode("onboarding", "finish"), discordgo.PrimaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) current(gid int64) *model.ServerSetting {
	ss, _ := c.store.GetServerSetting(gid)
	if ss == nil {
		ss = &model.ServerSetting{ServerID: gid}
	}
	return ss
}

func (c *Cog) save(gid int64, mut func(*model.ServerSetting)) {
	ss := c.current(gid)
	mut(ss)
	_ = c.store.SaveServerSetting(ss)
}

func (c *Cog) channelLabel(channelID int64) string {
	if channelID == 0 {
		return "—"
	}
	return "<#" + strconv.FormatInt(channelID, 10) + ">"
}

func (c *Cog) prefixLabel(ss *model.ServerSetting) string {
	if ss.Prefix != "" {
		return ss.Prefix
	}
	return c.cfg.Prefix
}

func (c *Cog) statusLabel(ss *model.ServerSetting, lang string) string {
	if ss.Enabled {
		return i18n.T("onboarding.status_enabled", lang)
	}
	return i18n.T("onboarding.status_disabled", lang)
}

func intPtr(n int) *int { return &n }
