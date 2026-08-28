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
	r.Slash("skills", "cmd.skills.desc", c.onSlashMenu)
	r.Prefix("skills", c.onPrefixMenu)
	r.Prefix("sk", c.onPrefixMenu)
	r.Component("skills", "refresh", c.onRefresh)
	r.Component("skills", "activate", c.onActivate)
	r.Component("skills", "sw", c.onSecondWindPick)
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
				components.Embed(i18n.T("skills.activate_title", lang), msg, components.ColorDanger),
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

	switch skillID {
	case "overclock":
		// Immediate effect: grant extra fishing/hunting uses and mining descends.
		_ = c.store.GrantGameLimitCredit(userID, "fish", 3)
		_ = c.store.GrantGameLimitCredit(userID, "hunt", 3)
		_ = c.store.GrantGameLimitCredit(userID, "mine_descend", 3)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(
				discordgo.InteractionResponseChannelMessageWithSource,
				components.Embed(
					i18n.T("skills.activated_title", lang),
					i18n.T("skills.overclock_done", lang),
					components.ColorSuccess,
				),
				nil,
			))
		return
	case "second_wind":
		// Immediate effect: pick one daily game limit to reset.
		embed := components.Embed(
			i18n.T("skills.sw_title", lang),
			i18n.T("skills.sw_desc", lang),
			components.ColorInfo,
		)
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button("🎰 "+i18n.T("skills.game_slots", lang), components.EncodeOwner(userID, "skills", "sw", "slots"), discordgo.PrimaryButton),
				components.Button("🪙 "+i18n.T("skills.game_coinflip", lang), components.EncodeOwner(userID, "skills", "sw", "coinflip"), discordgo.PrimaryButton),
			),
			components.ActionRow(
				components.Button("🃏 "+i18n.T("skills.game_blackjack", lang), components.EncodeOwner(userID, "skills", "sw", "blackjack"), discordgo.PrimaryButton),
				components.Button("🔫 "+i18n.T("skills.game_roulette", lang), components.EncodeOwner(userID, "skills", "sw", "roulette"), discordgo.PrimaryButton),
			),
			components.ActionRow(
				components.Button("🎫 "+i18n.T("skills.game_lotto", lang), components.EncodeOwner(userID, "skills", "sw", "lotto"), discordgo.PrimaryButton),
				components.Button("🎲 "+i18n.T("skills.game_bet", lang), components.EncodeOwner(userID, "skills", "sw", "bet"), discordgo.PrimaryButton),
			),
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
		return
	}

	// Confirm and refresh the display
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(
			discordgo.InteractionResponseChannelMessageWithSource,
			components.Embed(
				i18n.T("skills.activated_title", lang),
				i18n.T("skills.activated_desc", lang, map[string]any{"skill": sk.Emoji + " " + sk.Name}),
				components.ColorSuccess,
			),
			nil,
		))
}

func (c *Cog) onSecondWindPick(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		interaction.RespondError(b, i, lang, "skills.not_found")
		return
	}
	game := rest[0]

	if err := c.store.ResetGameLimit(userID, game); err != nil {
		interaction.RespondError(b, i, lang, "skills.error")
		return
	}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(
			discordgo.InteractionResponseUpdateMessage,
			components.Embed(
				i18n.T("skills.activated_title", lang),
				i18n.T("skills.sw_picked", lang, map[string]any{"game": i18n.T("skills.game_"+game, lang)}),
				components.ColorSuccess,
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
			components.ColorDanger,
		), nil
	}

	sb := &strings.Builder{}
	var availButtons []discordgo.MessageComponent

	for _, st := range statuses {
		if st.Reason == "locked" {
			continue
		}
		desc := st.Description
		if t := i18n.T("skills.desc_"+st.ID, lang); t != "skills.desc_"+st.ID {
			desc = t
		}
		switch st.Reason {
		case "available":
			fmt.Fprintf(sb, "%s **%s** — *%s*\n> %s | CD: %dm\n\n",
				st.Emoji, st.Name, desc, i18n.T("skills.uses_left", lang, map[string]any{"left": st.UsesLeft, "max": st.DailyLimit}), st.CooldownMins)
			availButtons = append(availButtons,
				components.Button(st.Emoji+" "+st.Name, components.EncodeOwner(userID, "skills", "activate", st.ID), discordgo.SuccessButton))
		case "on cooldown":
			fmt.Fprintf(sb, "⏳ **%s** — *%s*\n> %s %dm\n\n",
				st.Name, desc, i18n.T("skills.on_cooldown", lang), st.CooldownMins)
		case "daily limit reached":
			fmt.Fprintf(sb, "📊 **%s** — *%s*\n> %s (%d/%d)\n\n",
				st.Name, desc, i18n.T("skills.limit_reached", lang), 0, st.DailyLimit)
		}
	}

	// Build components: refresh button + activate buttons packed 4 per row
	// (max 4 skill rows, within Discord's 5-row limit).
	var comps []discordgo.MessageComponent
	comps = append(comps, components.ActionRow(
		components.Button(i18n.T("skills.btn_refresh", lang), components.EncodeOwner(userID, "skills", "refresh"), discordgo.SecondaryButton),
	))
	for i := 0; i < len(availButtons) && len(comps) < 5; i += 4 {
		end := i + 4
		if end > len(availButtons) {
			end = len(availButtons)
		}
		row := make([]discordgo.MessageComponent, 0, end-i)
		for _, b := range availButtons[i:end] {
			row = append(row, b)
		}
		comps = append(comps, components.ActionRow(row...))
	}

	return components.Embed(
		i18n.T("skills.title", lang), sb.String(), components.ColorInfo,
	), comps
}
