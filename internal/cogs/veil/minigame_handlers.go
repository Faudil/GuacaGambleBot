package veil

import (
	"strconv"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	veilsvc "guacagamblebot/internal/service/veil"
)

func (c *Cog) startMinigame(b *interaction.Bot, i *discordgo.InteractionCreate, userID int64, mgType string, lang string) {
	c.mu.Lock()
	c.mgType[userID] = mgType
	switch mgType {
	case "attack":
		c.mgStates[userID] = veilsvc.NewAttackGamble()
		c.mu.Unlock()
		embed := &discordgo.MessageEmbed{
			Title:       i18n.T("veil.minigame.atk_title", lang),
			Description: i18n.T("veil.minigame.atk_desc", lang, map[string]any{"mult": 1, "dmg": 15}),
			Color:       0xe74c3c,
		}
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("veil.minigame.atk_btn_more", lang), components.Encode("veil", "mg_more"), discordgo.DangerButton),
				components.Button(i18n.T("veil.minigame.atk_btn_lock", lang), components.Encode("veil", "mg_lock"), discordgo.SuccessButton),
			),
		}
		c.respondEphemeral(b, i, embed, comps)

	case "heal":
		c.mgStates[userID] = veilsvc.NewDiceState()
		c.mu.Unlock()
		embed := &discordgo.MessageEmbed{
			Title:       i18n.T("veil.minigame.heal_title", lang),
			Description: i18n.T("veil.minigame.heal_desc", lang),
			Color:       0x2ecc71,
		}
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("veil.minigame.heal_btn_roll", lang), components.Encode("veil", "mg_roll"), discordgo.PrimaryButton),
				components.Button(i18n.T("veil.minigame.heal_btn_confirm", lang), components.Encode("veil", "mg_confirm_heal"), discordgo.SuccessButton),
			),
		}
		c.respondEphemeral(b, i, embed, comps)

	case "protect":
		st := veilsvc.NewShieldState()
		c.mgStates[userID] = st
		c.mu.Unlock()
		embed := &discordgo.MessageEmbed{
			Title:       i18n.T("veil.minigame.shield_title", lang),
			Description: veilsvc.ShieldDescription(st, lang),
			Color:       0x3498db,
		}
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("veil.minigame.shield_btn_down", lang), components.Encode("veil", "mg_minus"), discordgo.SecondaryButton),
				components.Button(i18n.T("veil.minigame.shield_btn_up", lang), components.Encode("veil", "mg_plus"), discordgo.PrimaryButton),
				components.Button(i18n.T("veil.minigame.shield_btn_confirm", lang), components.Encode("veil", "mg_confirm_shield"), discordgo.SuccessButton),
			),
		}
		c.respondEphemeral(b, i, embed, comps)
	}
}

func (c *Cog) handleMinigameAction(b *interaction.Bot, i *discordgo.InteractionCreate, action string, rest []string) {
	userID := interaction.ToInt64(i.Member.User.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	c.mu.RLock()
	state := c.mgStates[userID]
	actType := c.mgType[userID]
	c.mu.RUnlock()

	if state == nil {
		return
	}

	switch action {
	case "mg_more":
		if st, ok := state.(*veilsvc.AttackGambleState); ok {
			mult, failed := veilsvc.PressMorePower(st)
			if failed {
				st.Locked = true
				embed := &discordgo.MessageEmbed{
					Title:       i18n.T("veil.minigame.atk_fail_title", lang, map[string]any{"mult": mult}),
					Description: i18n.T("veil.minigame.atk_fail_desc", lang),
					Color:       0xe74c3c,
				}
				c.respond(b, i, embed, nil)
				return
			}
			odds := veilsvc.GambleOdds(mult + 1)
			embed := &discordgo.MessageEmbed{
				Title:       i18n.T("veil.minigame.atk_gamble_title", lang),
				Description: i18n.T("veil.minigame.atk_gamble_desc", lang, map[string]any{"mult": mult, "odds": odds, "dmg": mult * 15}),
				Color:       0xe74c3c,
			}
			comps := []discordgo.MessageComponent{
				components.ActionRow(
					components.Button(i18n.T("veil.minigame.atk_btn_more", lang), components.Encode("veil", "mg_more"), discordgo.DangerButton),
					components.Button(i18n.T("veil.minigame.atk_btn_lock", lang), components.Encode("veil", "mg_lock"), discordgo.SuccessButton),
				),
			}
			c.respond(b, i, embed, comps)
		}

	case "mg_lock":
		if st, ok := state.(*veilsvc.AttackGambleState); ok {
			dmg := veilsvc.LockDamage(st)
			value := dmg * 15
			c.respond(b, i, &discordgo.MessageEmbed{
				Title:       i18n.T("veil.minigame.atk_lock_title", lang),
				Description: i18n.T("veil.minigame.atk_lock_desc", lang, map[string]any{"dmg": value}),
				Color:       0x2ecc71,
			}, nil)
			c.resolveWithMiniGameResult(b, i, userID, actType, value)
		}

	case "mg_roll":
		if st, ok := state.(*veilsvc.DiceState); ok {
			veilsvc.RollDice(st)
			c.showDiceEmbed(b, i, st, lang)
		}

	case "mg_reroll":
		if st, ok := state.(*veilsvc.DiceState); ok && len(rest) > 0 {
			idx, _ := strconv.Atoi(rest[0])
			veilsvc.RerollDie(st, idx)
			c.showDiceEmbed(b, i, st, lang)
		}

	case "mg_confirm_heal":
		if st, ok := state.(*veilsvc.DiceState); ok {
			_, healPercent := veilsvc.EvaluateDiceHand(st.Dice, lang)
			c.respond(b, i, &discordgo.MessageEmbed{
				Title:       i18n.T("veil.minigame.heal_result_title", lang),
				Description: i18n.T("veil.minigame.heal_result_desc", lang, map[string]any{"pct": healPercent}),
				Color:       0x2ecc71,
			}, nil)
			c.resolveWithMiniGameResult(b, i, userID, actType, healPercent)
		}

	case "mg_plus":
		if st, ok := state.(*veilsvc.ShieldState); ok {
			veilsvc.AdjustShieldIntensity(st, 1)
			c.showShieldEmbed(b, i, st, lang)
		}

	case "mg_minus":
		if st, ok := state.(*veilsvc.ShieldState); ok {
			veilsvc.AdjustShieldIntensity(st, -1)
			c.showShieldEmbed(b, i, st, lang)
		}

	case "mg_confirm_shield":
		if st, ok := state.(*veilsvc.ShieldState); ok {
			val, backlash := veilsvc.ConfirmShield(st)
			desc := i18n.T("veil.minigame.shield_result_desc", lang, map[string]any{"val": val})
			if backlash > 0 {
				desc += i18n.T("veil.minigame.shield_backlash", lang, map[string]any{"dmg": backlash})
			}
			c.respond(b, i, &discordgo.MessageEmbed{
				Title:       i18n.T("veil.minigame.shield_result_title", lang),
				Description: desc,
				Color:       0x2ecc71,
			}, nil)
			c.resolveWithMiniGameResult(b, i, userID, actType, val)
		}
	}

	c.mu.Lock()
	delete(c.mgStates, userID)
	delete(c.mgType, userID)
	c.mu.Unlock()
}

func (c *Cog) resolveWithMiniGameResult(b *interaction.Bot, i *discordgo.InteractionCreate, userID int64, actType string, value int) {
	c.mu.Lock()
	raid := c.getRaid(userID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	c.mu.Unlock()
	if raid == nil {
		return
	}
	c.recordAndCheckResolution(b, i, raid, userID, actType, value, lang)
}

func (c *Cog) showDiceEmbed(b *interaction.Bot, i *discordgo.InteractionCreate, st *veilsvc.DiceState, lang string) {
	comps := []discordgo.MessageComponent{}
	if st.Rerolls > 0 {
		comps = append(comps, components.ActionRow(
			components.Button("1️⃣", components.Encode("veil", "mg_reroll", "0"), discordgo.SecondaryButton),
			components.Button("2️⃣", components.Encode("veil", "mg_reroll", "1"), discordgo.SecondaryButton),
			components.Button("3️⃣", components.Encode("veil", "mg_reroll", "2"), discordgo.SecondaryButton),
			components.Button("4️⃣", components.Encode("veil", "mg_reroll", "3"), discordgo.SecondaryButton),
			components.Button("5️⃣", components.Encode("veil", "mg_reroll", "4"), discordgo.SecondaryButton),
		))
	}
	comps = append(comps, components.ActionRow(
		components.Button(i18n.T("veil.minigame.heal_btn_roll", lang), components.Encode("veil", "mg_roll"), discordgo.PrimaryButton),
		components.Button(i18n.T("veil.minigame.heal_btn_confirm", lang), components.Encode("veil", "mg_confirm_heal"), discordgo.SuccessButton),
	))
	c.respond(b, i, &discordgo.MessageEmbed{
		Title:       i18n.T("veil.minigame.dice_title", lang),
		Description: veilsvc.DiceHandDescription(st, lang),
		Color:       0x2ecc71,
	}, comps)
}

func (c *Cog) showShieldEmbed(b *interaction.Bot, i *discordgo.InteractionCreate, st *veilsvc.ShieldState, lang string) {
	c.respond(b, i, &discordgo.MessageEmbed{
		Title:       i18n.T("veil.minigame.shield_title", lang),
		Description: veilsvc.ShieldDescription(st, lang),
		Color:       0x3498db,
	}, []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("veil.minigame.shield_btn_down", lang), components.Encode("veil", "mg_minus"), discordgo.SecondaryButton),
			components.Button(i18n.T("veil.minigame.shield_btn_up", lang), components.Encode("veil", "mg_plus"), discordgo.PrimaryButton),
			components.Button(i18n.T("veil.minigame.shield_btn_confirm", lang), components.Encode("veil", "mg_confirm_shield"), discordgo.SuccessButton),
		),
	})
}
