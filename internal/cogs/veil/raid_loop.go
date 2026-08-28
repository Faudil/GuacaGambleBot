package veil

import (
	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	veilsvc "guacagamblebot/internal/service/veil"
)

func (c *Cog) onAction(b *interaction.Bot, i *discordgo.InteractionCreate) {
	_, action, rest := components.Decode(i.MessageComponentData().CustomID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	switch action {
	case "join":
		c.handleJoin(b, i)
	case "start_btn":
		c.handleStartRaid(b, i)
	case "whisper_answer":
		c.openWhisperModal(b, i)
	case "flame_protect", "flame_extinguish", "flame_scout":
		c.handleFlameAction(b, i, action, lang)
	case "guard_atk", "guard_prot", "guard_skip":
		c.handleGuardianAction(b, i, action, lang)
	case "breach_awe", "breach_defy", "breach_fear":
		c.handleBreachAction(b, i, action)

	case "boss_atk1", "boss_atk2", "boss_atk3", "boss_add", "boss_heal2", "boss_prot2", "boss_heal3", "boss_prot3", "stabilize":
		c.handleBossAction(b, i, action, lang)
	case "anchor":
		c.handleAnchor(b, i)

	case "mg_more", "mg_lock", "mg_roll", "mg_reroll", "mg_confirm_heal", "mg_plus", "mg_minus", "mg_confirm_shield":
		c.handleMinigameAction(b, i, action, rest)
	}
}

func (c *Cog) openWhisperModal(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: components.ModalResponse(
			components.Encode("veil", "whisper_modal"),
			i18n.T("veil.encounter.whisper_modal_title", lang),
			components.TextInput(
				"answer",
				i18n.T("veil.encounter.whisper_modal_label", lang),
				true,
				i18n.T("veil.encounter.whisper_modal_placeholder", lang),
				discordgo.TextInputShort, 1, 100,
			),
		),
	})
}

func (c *Cog) onWhisperModal(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	raid := c.getRaid(userID)
	if raid == nil {
		c.errorEphemeral(b, i, i18n.T("veil.encounter.not_in_raid", lang))
		return
	}

	answer := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	correct, desc := veilsvc.CheckWhisperAnswer(raid, answer, lang)

	if correct {
		raid.Phase = "shifting_flames"
		c.svc.Store().SaveVeilRaid(raid)
		res := veilsvc.StartEncounter(raid, lang)
		c.respond(b, i, res.PublicEmbed, res.Comps)
	} else {
		c.errorEphemeral(b, i, desc)
	}
}

func (c *Cog) handleFlameAction(b *interaction.Bot, i *discordgo.InteractionCreate, action string, lang string) {
	userID := interaction.ToInt64(i.Member.User.ID)
	raid := c.getRaid(userID)
	if raid == nil {
		c.errorEphemeral(b, i, i18n.T("veil.encounter.not_in_raid", lang))
		return
	}

	actionMap := map[int64]string{userID: action}
	complete, desc, newMech := veilsvc.ProcessFlameTurn(raid.Mechanics, actionMap, lang)

	raid.Mechanics = newMech

	if complete {
		raid.Phase = "guardian"
		c.svc.Store().SaveVeilRaid(raid)
		res := veilsvc.StartEncounter(raid, lang)
		c.respond(b, i, res.PublicEmbed, res.Comps)
	} else {
		c.svc.Store().SaveVeilRaid(raid)
		embed := &discordgo.MessageEmbed{
			Title:       i18n.T("veil.encounter.flames_title", lang),
			Description: desc,
			Color:       components.ColorWarning,
		}
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("veil.encounter.flames_btn_protect", lang), components.Encode("veil", "flame_protect"), discordgo.PrimaryButton),
				components.Button(i18n.T("veil.encounter.flames_btn_extinguish", lang), components.Encode("veil", "flame_extinguish"), discordgo.SuccessButton),
				components.Button(i18n.T("veil.encounter.flames_btn_scout", lang), components.Encode("veil", "flame_scout"), discordgo.SecondaryButton),
			),
		}
		c.respond(b, i, embed, comps)
	}
}

func (c *Cog) handleGuardianAction(b *interaction.Bot, i *discordgo.InteractionCreate, action string, lang string) {
	userID := interaction.ToInt64(i.Member.User.ID)
	raid := c.getRaid(userID)
	if raid == nil {
		c.errorEphemeral(b, i, i18n.T("veil.encounter.not_in_raid", lang))
		return
	}

	actionMap := map[int64]string{userID: action}
	complete, desc, newMech := veilsvc.ProcessGuardianTurn(raid.Mechanics, actionMap, lang)

	raid.Mechanics = newMech

	if complete {
		raid.Phase = "breach"
		c.svc.Store().SaveVeilRaid(raid)
		res := veilsvc.StartEncounter(raid, lang)
		c.respond(b, i, res.PublicEmbed, res.Comps)
	} else {
		c.svc.Store().SaveVeilRaid(raid)
		embed := &discordgo.MessageEmbed{
			Title:       i18n.T("veil.encounter.guardian_title", lang),
			Description: desc,
			Color:       components.ColorInfo,
		}
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("veil.encounter.guardian_btn_atk", lang), components.Encode("veil", "guard_atk"), discordgo.DangerButton),
				components.Button(i18n.T("veil.encounter.guardian_btn_prot", lang), components.Encode("veil", "guard_prot"), discordgo.PrimaryButton),
				components.Button(i18n.T("veil.encounter.guardian_btn_skip", lang), components.Encode("veil", "guard_skip"), discordgo.SecondaryButton),
			),
		}
		c.respond(b, i, embed, comps)
	}
}

func (c *Cog) handleBreachAction(b *interaction.Bot, i *discordgo.InteractionCreate, action string) {
	userID := interaction.ToInt64(i.Member.User.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	raid := c.getRaid(userID)
	if raid == nil {
		c.errorEphemeral(b, i, i18n.T("veil.encounter.not_in_raid", lang))
		return
	}

	vote := ""
	switch action {
	case "breach_awe":
		vote = "awe"
	case "breach_defy":
		vote = "defiance"
	case "breach_fear":
		vote = "fear"
	}

	c.mu.Lock()
	if c.breachVotes[raid.ID] == nil {
		c.breachVotes[raid.ID] = map[int64]string{}
	}
	c.breachVotes[raid.ID][userID] = vote
	c.mu.Unlock()

	total := len(veilsvc.GetParticipantsWith(raid))
	c.mu.RLock()
	cast := len(c.breachVotes[raid.ID])
	c.mu.RUnlock()

	if cast >= total {
		c.mu.RLock()
		counts := map[string]int{}
		for _, v := range c.breachVotes[raid.ID] {
			counts[v]++
		}
		c.mu.RUnlock()

		boon := veilsvc.GetBreachBoon(counts, lang)
		raid.Phase = "boss_p1"
		c.mu.Lock()
		delete(c.breachVotes, raid.ID)
		c.mu.Unlock()

		res := veilsvc.StartBossPhase(raid, 1, lang)
		res.Embed.Description = i18n.T("veil.encounter.breach_boon_prepend", lang, map[string]any{"boon": boon, "desc": res.Embed.Description})
		c.svc.Store().SaveVeilRaid(raid)
		c.respondBoss(b, i, raid, res.Embed, res.Comps)
	} else {
		c.errorEphemeral(b, i, i18n.T("veil.encounter.vote_waiting", lang, map[string]any{"choice": vote, "cast": cast, "total": total}))
	}
}
