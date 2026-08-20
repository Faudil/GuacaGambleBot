package veil

import (
	"strconv"
	"sync"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/assets"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	veilsvc "guacagamblebot/internal/service/veil"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store       *store.Store
	cfg         *config.Config
	svc         *veilsvc.Service
	activeRaids map[int64]*model.VeilRaid
	mgStates    map[int64]any
	mgType      map[int64]string
	breachVotes map[int64]map[int64]string
	turnActions map[int64]map[int64]string
	mu          sync.RWMutex
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{
		store:       s,
		cfg:         cfg,
		svc:         veilsvc.New(s, cfg),
		activeRaids: make(map[int64]*model.VeilRaid),
		mgStates:    make(map[int64]any),
		mgType:      make(map[int64]string),
		breachVotes: make(map[int64]map[int64]string),
		turnActions: make(map[int64]map[int64]string),
	}

	r.Slash("raid", "Veil Rift raid commands", c.onSlashRaid)
	r.Prefix("raid", c.onPrefixRaid)

	actions := []string{
		"whisper_answer", "flame_protect", "flame_extinguish", "flame_scout",
		"guard_atk", "guard_prot", "guard_skip",
		"breach_awe", "breach_defy", "breach_fear",
		"boss_atk1", "boss_atk2", "boss_atk3", "boss_add",
		"boss_heal2", "boss_prot2", "boss_heal3", "boss_prot3", "stabilize", "anchor",
		"mg_more", "mg_lock", "mg_roll", "mg_reroll", "mg_confirm_heal", "mg_plus", "mg_minus", "mg_confirm_shield",
		"join", "start_btn",
	}
	for _, a := range actions {
		r.Component("veil", a, c.onAction)
	}

	r.Modal("veil", "whisper_modal", c.onWhisperModal)
}

func (c *Cog) getRaid(userID int64) *model.VeilRaid {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, raid := range c.activeRaids {
		for _, pid := range veilsvc.GetParticipantsWith(raid) {
			if pid == userID {
				return raid
			}
		}
	}
	return nil
}

func (c *Cog) getRaidByID(raidID int64) *model.VeilRaid {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.activeRaids[raidID]
}

func (c *Cog) respond(b *interaction.Bot, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent) {
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// respondBoss is respond but uploads the raid boss's picture (Vault Guardian)
// with the embed. A raid without a picture degrades to a plain respond.
func (c *Cog) respondBoss(b *interaction.Bot, i *discordgo.InteractionCreate, raid *model.VeilRaid, embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent) {
	image := ""
	if raid != nil {
		image = raid.BossImage
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		assets.Response(discordgo.InteractionResponseUpdateMessage, embed, comps, image))
}

func (c *Cog) respondEphemeral(b *interaction.Bot, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent) {
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: comps,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) errorEphemeral(b *interaction.Bot, i *discordgo.InteractionCreate, msg string) {
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) onSlashRaid(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	ok, msg := c.svc.ValidateGate(userID, lang)
	if !ok {
		c.errorEphemeral(b, i, msg)
		return
	}

	raid, err := c.svc.CreateRaid(userID, interaction.ToInt64(i.GuildID), lang)
	if err != nil {
		c.errorEphemeral(b, i, err.Error())
		return
	}

	c.mu.Lock()
	c.activeRaids[raid.ID] = raid
	c.mu.Unlock()

	embed := &discordgo.MessageEmbed{
		Title:       i18n.T("veil.group.title", lang),
		Description: i18n.T("veil.group.desc_created", lang, map[string]any{"leader": userID}),
		Color:       0x9b59b6,
		Footer:      &discordgo.MessageEmbedFooter{Text: i18n.T("veil.group.footer", lang, map[string]any{"id": raid.ID})},
	}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("veil.group.btn_join", lang), components.Encode("veil", "join"), discordgo.SuccessButton),
			components.Button(i18n.T("veil.group.btn_start", lang), components.Encode("veil", "start_btn"), discordgo.DangerButton),
		),
	}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixRaid(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))

	ok, msg := c.svc.ValidateGate(userID, lang)
	if !ok {
		_, _ = s.ChannelMessageSend(m.ChannelID, msg)
		return
	}

	raid, err := c.svc.CreateRaid(userID, interaction.ToInt64(m.GuildID), lang)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, err.Error())
		return
	}

	c.mu.Lock()
	c.activeRaids[raid.ID] = raid
	c.mu.Unlock()

	embed := &discordgo.MessageEmbed{
		Title:       i18n.T("veil.group.title", lang),
		Description: i18n.T("veil.group.desc_created", lang, map[string]any{"leader": userID}),
		Color:       0x9b59b6,
	}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("veil.group.btn_join", lang), components.Encode("veil", "join"), discordgo.SuccessButton),
			components.Button(i18n.T("veil.group.btn_start", lang), components.Encode("veil", "start_btn"), discordgo.DangerButton),
		),
	}
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) handleJoin(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	raid := c.findFormingRaid(i.GuildID)
	if raid == nil {
		c.errorEphemeral(b, i, i18n.T("veil.group.err_not_found", lang))
		return
	}

	if err := c.svc.JoinRaid(raid, userID, lang); err != nil {
		c.errorEphemeral(b, i, err.Error())
		return
	}

	ids := veilsvc.GetParticipantsWith(raid)
	embed := &discordgo.MessageEmbed{
		Title:       i18n.T("veil.group.title", lang),
		Description: i18n.T("veil.group.desc_updated", lang, map[string]any{"leader": raid.LeaderID, "count": len(ids)}),
		Color:       0x9b59b6,
	}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("veil.group.btn_join", lang), components.Encode("veil", "join"), discordgo.SuccessButton),
			components.Button(i18n.T("veil.group.btn_start", lang), components.Encode("veil", "start_btn"), discordgo.DangerButton),
		),
	}
	c.respond(b, i, embed, comps)
}

func (c *Cog) handleStartRaid(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	raid := c.findFormingRaid(i.GuildID)
	if raid == nil {
		c.errorEphemeral(b, i, i18n.T("veil.group.err_not_found", lang))
		return
	}
	if raid.LeaderID != userID {
		c.errorEphemeral(b, i, i18n.T("veil.gate.err_not_leader", lang))
		return
	}

	if err := c.svc.StartRaid(raid, userID, lang); err != nil {
		c.errorEphemeral(b, i, err.Error())
		return
	}

	res := veilsvc.StartEncounter(raid, lang)
	c.respond(b, i, res.PublicEmbed, res.Comps)
}

func (c *Cog) findFormingRaid(guildID string) *model.VeilRaid {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, raid := range c.activeRaids {
		if strconv.FormatInt(raid.GuildID, 10) == guildID && raid.Status == "forming" {
			return raid
		}
	}
	return nil
}

var _ = interaction.ToInt64
