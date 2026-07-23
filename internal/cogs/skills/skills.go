package skills

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	charsvc "guacagamblebot/internal/service/character"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *charsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: charsvc.New(s, cfg)}
	r.Slash("skills", "View and activate your RPG skills.", c.onSlashMenu)
	r.Prefix("skills", c.onPrefixMenu)
	r.Prefix("sk", c.onPrefixMenu)
	r.Component("skills", "refresh", c.onRefresh)
	r.Component("skills", "activate", c.onActivate)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.skillsView(lang, b, i)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	embed, comps := c.buildDisplay(lang, interaction.ToInt64(m.Author.ID))
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) onRefresh(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.buildDisplay(lang, interaction.ToInt64(interaction.UserID(i)))
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onActivate(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	// The skill ID is encoded in the custom_id as "skills::activate::{skillID}"
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	skillID := ""
	if len(rest) > 0 {
		skillID = rest[0]
	}

	sk, ok := charsvc.GetSkill(skillID)
	if !ok {
		interaction.RespondError(b, i, lang, "skills.not_found")
		return
	}

	// Check availability
	available, reason, err := c.store.CheckSkillAvailability(userID, skillID, sk.DailyLimit, sk.CooldownMins)
	if err != nil || !available {
		msg := i18n.T("skills.unavailable", lang, map[string]any{"reason": reason})
		if err != nil {
			msg = i18n.T("skills.error", lang)
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(
				discordgo.InteractionResponseChannelMessageWithSource,
				components.Embed(i18n.T("skills.activate_title", lang), msg, 0xe74c3c),
				nil,
			))
		return
	}

	// Activate the skill: record usage + set buff
	if err := c.svc.ActivateSkill(userID, skillID); err != nil {
		interaction.RespondError(b, i, lang, "skills.error")
		return
	}
	if err := c.svc.SetBuff(userID, skillID); err != nil {
		interaction.RespondError(b, i, lang, "skills.error")
		return
	}

	// Confirm and refresh the display
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(
			discordgo.InteractionResponseChannelMessageWithSource,
			components.Embed(
				i18n.T("skills.activated_title", lang),
				i18n.T("skills.activated_desc", lang, map[string]any{"skill": sk.Emoji + " " + sk.Name}),
				0x2ecc71,
			),
			nil,
		))
}

// skillsView is the first-time slash response that builds the initial display.
func (c *Cog) skillsView(lang string, b *interaction.Bot, i *discordgo.InteractionCreate) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	return c.buildDisplay(lang, interaction.ToInt64(interaction.UserID(i)))
}

func (c *Cog) buildDisplay(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	statuses, err := c.svc.GetSkills(userID)
	if err != nil {
		return components.Embed(
			i18n.T("skills.title", lang),
			i18n.T("skills.error", lang),
			0xe74c3c,
		), nil
	}

	sb := &strings.Builder{}
	var availButtons []discordgo.MessageComponent

	for _, st := range statuses {
		switch st.Reason {
		case "available":
			fmt.Fprintf(sb, "%s **%s** — *%s*\n> %s | CD: %dm\n\n",
				st.Emoji, st.Name, st.Description, i18n.T("skills.uses_left", lang, map[string]any{"left": st.UsesLeft, "max": st.DailyLimit}), st.CooldownMins)
			availButtons = append(availButtons,
				components.Button(st.Emoji+" "+st.Name, components.Encode("skills", "activate", st.ID), discordgo.SuccessButton))
		case "locked":
			fmt.Fprintf(sb, "🔒 **%s** — %s\n> %s **Lv.%d**\n\n",
				st.Name, st.Description, i18n.T("skills.unlocks_at", lang), st.UnlockLevel)
		case "on cooldown":
			fmt.Fprintf(sb, "⏳ **%s** — *%s*\n> %s %dm\n\n",
				st.Name, st.Description, i18n.T("skills.on_cooldown", lang), st.CooldownMins)
		case "daily limit reached":
			fmt.Fprintf(sb, "📊 **%s** — *%s*\n> %s (%d/%d)\n\n",
				st.Name, st.Description, i18n.T("skills.limit_reached", lang), 0, st.DailyLimit)
		}
	}

	// Build components: refresh button + up to 10 activate buttons
	var comps []discordgo.MessageComponent
	comps = append(comps, components.ActionRow(
		components.Button(i18n.T("skills.btn_refresh", lang), components.Encode("skills", "refresh"), discordgo.SecondaryButton),
	))
	for _, b := range availButtons {
		if len(comps) < 4 { // max 4 rows (1 refresh + 3 skill rows)
			comps = append(comps, components.ActionRow(b))
		}
	}

	return components.Embed(
		i18n.T("skills.title", lang), sb.String(), 0x3498db,
	), comps
}
