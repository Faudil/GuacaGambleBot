package pets

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	petsvc "guacagamblebot/internal/service/pets"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *petsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: petsvc.New(s, cfg)}
	r.Slash("pets", "Gérer vos familiers", c.onSlashMenu)
	r.Slash("pet", "Gérer vos familiers", c.onSlashMenu)
	r.Slash("hatch", "Éclore un œuf de familier", c.onHatchCommand)
	r.Prefix("pets", c.onPrefixMenu)
	r.Prefix("pet", c.onPrefixMenu)
	r.Prefix("hatch", c.onHatchPrefix)
	r.Component("pets", "menu", c.onMenu)
	r.Component("pets", "pet", c.onPetDetail)
	r.Component("pets", "feed", c.onFeed)
	r.Component("pets", "rename_btn", c.onRenameOpen)
	r.Modal("pets", "rename_submit", c.onRenameSubmit)
	r.Component("pets", "delete", c.onDelete)
	r.Component("pets", "battle", c.onBattleSelect)
	r.Component("pets", "battle_accept", c.onBattleAccept)
	r.Component("pets", "battle_decline", c.onBattleDecline)
	r.Component("pets", "skills", c.onSkillSelect)
	r.Component("pets", "skill_choose", c.onSkillChoose)
	r.Component("pets", "interact", c.onInteractionChoice)
	r.Slash("artifact", "Gérer votre artefact de familier", c.onArtifactMenu)
	r.Slash("weekly", "Classement hebdomadaire des familiers", c.onWeeklyLeaderboard)
	r.Prefix("weekly", c.onWeeklyPrefix)
	r.Component("pets", "artifact_view", c.onArtifactView)
	r.Component("pets", "artifact_reset", c.onArtifactReset)
	r.Component("pets", "artifact_stat_choose", c.onArtifactStatChoose)
	r.Component("pets", "weekly_refresh", c.onWeeklyRefresh)
	r.Component("pets", "weekly_history", c.onWeeklyHistory)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(i, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	embed, comps := c.menuFromUser(interaction.ToInt64(m.Author.ID), lang)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) menu(i *discordgo.InteractionCreate, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	userID := interaction.ToInt64(interaction.UserID(i))
	return c.menuFromUser(userID, lang)
}

func (c *Cog) menuFromUser(userID int64, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	pets, err := c.svc.GetPets(userID)
	desc := ""
	if err != nil || len(pets) == 0 {
		desc = i18n.T("pets.list.no_pets", lang)
	} else {
		for _, p := range pets {
			status := i18n.T("pets.list.inactive", lang)
			if p.IsActive {
				status = i18n.T("pets.list.active", lang)
			}
			pt := petsvc.PetTypes[p.PetType]
			emoji := "🐾"
			if pt != nil {
				emoji = pt.Emoji
			}
			desc += fmt.Sprintf("%s **%s** - %s ID: `%d`\n", emoji, p.Nickname, status, p.ID)
		}
	}
	embed := components.Embed(i18n.T("pets.list.title", lang, map[string]any{"name": MentionUser(userID)}), desc, 0x2ecc71)
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("pets.list.footer", lang)}

	comps := []discordgo.MessageComponent{}
	if len(pets) > 0 {
		opts := make([]discordgo.SelectMenuOption, 0, len(pets))
		for _, p := range pets {
			pt := petsvc.PetTypes[p.PetType]
			emoji := "🐾"
			if pt != nil {
				emoji = pt.Emoji
			}
			label := p.Nickname
			if len(label) > 25 {
				label = label[:25]
			}
			opts = append(opts, discordgo.SelectMenuOption{
				Label: label,
				Value: strconv.FormatInt(p.ID, 10),
				Emoji: &discordgo.ComponentEmoji{Name: emoji},
			})
		}
		comps = append(comps, components.ActionRow(
			discordgo.SelectMenu{
				CustomID:    components.Encode("pets", "pet"),
				Placeholder: i18n.T("pets.list.footer", lang),
				Options:     opts,
			},
		))
	}
	return embed, comps
}

func (c *Cog) onMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(i, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onPetDetail(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	data := i.MessageComponentData()
	petIDStr := ""
	if len(data.Values) > 0 {
		petIDStr = data.Values[0]
	} else {
		_, _, rest := components.Decode(data.CustomID)
		if len(rest) > 0 {
			petIDStr = rest[0]
		}
	}
	petID, _ := strconv.ParseInt(petIDStr, 10, 64)
	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}
	pt := petsvc.PetTypes[pet.PetType]
	emoji := "🐾"
	if pt != nil {
		emoji = pt.Emoji
	}
	rarity := i18n.T("rarities."+petsvc.RarityBonus(petTypeRarity(pet)), lang)
	personality := petsvc.PersonalityTraits[pet.Personality]

	// Build bond bar
	bondBar := buildBondBar(pet.BondLevel)
	// Build skills list
	skills, _ := c.svc.GetPetSkills(pet.ID)
	skillStr := ""
	if len(skills) > 0 {
		skillLines := make([]string, 0, len(skills))
		for _, s := range skills {
			if def, ok := petsvc.AllPetSkills[s.SkillID]; ok {
				slot := (s.Slot + 1) * 10
				skillLines = append(skillLines, fmt.Sprintf("  %s **%s** (lvl %d)", def.Emoji, def.Name, slot))
			}
		}
		if len(skillLines) > 0 {
			skillStr = "\n**Skills:**\n" + strings.Join(skillLines, "\n")
		}
	}
	// Build history (last 3)
	hist := c.svc.GetHistory(pet)
	histStr := ""
	if len(hist) > 0 {
		lines := make([]string, 0, 3)
		start := len(hist) - 3
		if start < 0 {
			start = 0
		}
		for _, h := range hist[start:] {
			d := h.Detail
			if len([]rune(d)) > 80 {
				d = string([]rune(d)[:80]) + "..."
			}
			lines = append(lines, "📜 "+d)
		}
		histStr = "\n**Recent History:**\n" + strings.Join(lines, "\n")
	}

	pTrait := ""
	if personality != nil {
		pTrait = i18n.T("pets.personality."+pet.Personality, lang)
	}
	title := ""
	if pet.Title != "" {
		title = " 🏆 *" + pet.Title + "*"
	}

	desc := fmt.Sprintf("**%s** | %s | Lvl %d%s | %s\n\n",
		rarity, emoji, pet.Level, title, pTrait)
	desc += fmt.Sprintf("HP: %d/%d | ATK: %d | DEF: %d | SPD: %d\n", pet.HP, pet.MaxHP, pet.Atk, pet.Defense, pet.Speed)
	desc += fmt.Sprintf("DGE: %d | ACC: %d | CRIT: %d/%0.1f\n", pet.DGE, pet.ACC, pet.CritC, pet.CritD)
	desc += fmt.Sprintf("ELO: %d | XP: %d\n", pet.Elo, pet.XP)
	desc += fmt.Sprintf("\n**Bond:** %s %d/%d", bondBar, pet.BondLevel, petsvc.MaxBond)
	if pet.SkillPoints > 0 {
		desc += fmt.Sprintf("\n\n⚡ **%d** skill point(s) available! Use `/pets skills` to choose.", pet.SkillPoints)
	}
	desc += skillStr
	desc += histStr

	embed := components.Embed(
		fmt.Sprintf("%s %s", emoji, pet.Nickname),
		desc,
		0x3498db,
	)

	buttons := []discordgo.MessageComponent{
		components.Button(i18n.T("pets.rename.success", lang), components.Encode("pets", "rename_btn", petIDStr), discordgo.PrimaryButton),
		components.Button(i18n.T("pets.feed.miam_title", lang), components.Encode("pets", "feed", petIDStr), discordgo.SuccessButton),
		components.Button(i18n.T("pets.battle.arena_title", lang), components.Encode("pets", "battle", petIDStr), discordgo.DangerButton),
	}
	if pet.SkillPoints > 0 {
		buttons = append(buttons,
			components.Button("⚡ Skills", components.Encode("pets", "skills", petIDStr), discordgo.PrimaryButton))
	}
	buttons = append(buttons, components.Button("🗑️", components.Encode("pets", "delete", petIDStr), discordgo.SecondaryButton))

	comps := []discordgo.MessageComponent{
		components.ActionRow(buttons...),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func buildBondBar(level int) string {
	filled := level / 10
	total := 10
	bar := ""
	for i := 0; i < filled; i++ {
		bar += "▓"
	}
	for i := filled; i < total; i++ {
		bar += "░"
	}
	return bar
}

func (c *Cog) onRenameOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	petID := "0"
	if len(rest) > 0 {
		petID = rest[0]
	}
	modal := components.ModalResponse(
		components.Encode("pets", "rename_submit", petID),
		i18n.T("pets.rename.success", lang),
		components.TextInput("name", i18n.T("pets.rename.success", lang), true, i18n.T("pets.rename.success", lang), discordgo.TextInputShort, 1, 20),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onRenameSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_, _, rest := components.Decode(i.ModalSubmitData().CustomID)
	petIDStr := "0"
	if len(rest) > 0 {
		petIDStr = rest[0]
	}
	petID, _ := strconv.ParseInt(petIDStr, 10, 64)
	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}
	vals := interaction.ModalValues(i)
	newName := strings.TrimSpace(vals["name"])
	if len(newName) > 20 || newName == "" {
		interaction.RespondError(b, i, lang, "pets.rename.too_long")
		return
	}
	pet.Nickname = newName
	_ = c.svc.UpdatePet(pet)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("pets.rename.success", lang, map[string]any{"name": newName}),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) onFeed(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	petIDStr := "0"
	if len(rest) > 0 {
		petIDStr = rest[0]
	}
	petID, _ := strconv.ParseInt(petIDStr, 10, 64)
	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}

	ready, _ := c.store.CheckCooldown(pet.UserID, "pet_feed", 5*time.Minute)
	if !ready {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⏳ Your pet was fed recently. Wait a few minutes before feeding again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🍖 **%s** has been fed! (+5 HP)", pet.Nickname),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	pet.HP = min(pet.MaxHP, pet.HP+5)
	c.svc.AddBond(pet, 1)
	_ = c.svc.UpdatePet(pet)
	_ = c.store.SetCooldown(pet.UserID, "pet_feed")
	_ = c.store.RecordActivity(userID, "pets_fed", 1)

	// Check for interaction
	c.tryInteraction(b, i, pet, "feed")
}

func (c *Cog) onDelete(b *interaction.Bot, i *discordgo.InteractionCreate) {
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	petIDStr := "0"
	if len(rest) > 0 {
		petIDStr = rest[0]
	}
	petID, _ := strconv.ParseInt(petIDStr, 10, 64)
	_ = c.svc.DeletePet(petID)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🗑️ Pet deleted.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) onBattleSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	petID, _ := strconv.ParseInt(rest[0], 10, 64)

	members, err := b.Session.GuildMembers(i.GuildID, "", 100)
	if err != nil {
		interaction.RespondError(b, i, lang, "pets.battle.wrong_opponent")
		return
	}
	opts := make([]discordgo.SelectMenuOption, 0)
	for _, m := range members {
		if m.User.ID == interaction.UserID(i) || m.User.Bot {
			continue
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label: m.User.Username,
			Value: m.User.ID,
			Emoji: &discordgo.ComponentEmoji{Name: "⚔️"},
		})
		if len(opts) >= 25 {
			break
		}
	}
	if len(opts) == 0 {
		interaction.RespondError(b, i, lang, "pets.battle.wrong_opponent")
		return
	}
	embed := components.Embed(i18n.T("pets.battle.arena_title", lang), i18n.T("pets.battle.challenge_msg", lang, map[string]any{"challenger": MentionUser(userID), "pet": petID}), 0xe74c3c)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, []discordgo.MessageComponent{
			components.ActionRow(
				discordgo.SelectMenu{
					CustomID:    components.Encode("pets", "battle_accept", rest[0]),
					Placeholder: "Select opponent",
					Options:     opts,
				},
			),
		}))
}

type battleChallenge struct {
	ChallengerID int64
	OpponentID   int64
	PetID        int64
	Lang         string
}

func (c *Cog) onBattleAccept(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	data := i.MessageComponentData()
	opponentIDStr := ""
	if len(data.Values) > 0 {
		opponentIDStr = data.Values[0]
	} else {
		return
	}
	opponentID := interaction.ToInt64(opponentIDStr)
	if opponentID == userID {
		return
	}
	_, _, rest := components.Decode(data.CustomID)
	if len(rest) == 0 {
		return
	}
	petID, _ := strconv.ParseInt(rest[0], 10, 64)

	pet1, err := c.svc.GetPetByID(petID)
	if err != nil || pet1 == nil || pet1.UserID != userID {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}

	pet2, err := c.svc.GetActivePet(opponentID)
	if err != nil || pet2 == nil {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: i18n.T("pets.battle.opponent_no_pet", lang, map[string]any{"name": MentionUser(opponentID)}),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	bp1 := c.petToBattlePet(pet1)
	bp2 := c.petToBattlePet(pet2)

	modID, _ := c.getActiveModID(interaction.ToInt64(i.GuildID))
	c.applyArtifacts(bp1, bp2, userID, opponentID, modID)

	result := battle.Simulate(bp1, bp2, modID)

	_ = c.svc.UpdatePet(pet1)
	_ = c.svc.UpdatePet(pet2)

	diff1, diff2 := c.svc.UpdateElo(pet1, pet2, battleResultToFloat(result, pet1.ID))

	serverID := interaction.ToInt64(i.GuildID)

	// Weekly score tracking
	var weeklyScoreA, weeklyScoreB int
	artXP1, artXP2 := 0, 0
	if result.WinnerID == pet1.ID {
		weeklyScoreA = c.svc.CalculateScoreDelta(pet1.Elo-diff1, pet2.Elo-diff2)
		weeklyScoreB = 5
		artXP1, artXP2 = petsvc.ArtifactPVPWinXP, petsvc.ArtifactPVPLossXP
	} else if result.WinnerID == pet2.ID {
		weeklyScoreA = 5
		weeklyScoreB = c.svc.CalculateScoreDelta(pet2.Elo-diff2, pet1.Elo-diff1)
		artXP1, artXP2 = petsvc.ArtifactPVPLossXP, petsvc.ArtifactPVPWinXP
	} else {
		weeklyScoreA = 5
		weeklyScoreB = 5
		artXP1, artXP2 = petsvc.ArtifactPVPLossXP, petsvc.ArtifactPVPLossXP
	}
	c.svc.AddWeeklyScore(userID, serverID, weeklyScoreA, map[bool]int{true: 1, false: 0}[result.WinnerID == pet1.ID])
	c.svc.AddWeeklyScore(opponentID, serverID, weeklyScoreB, map[bool]int{true: 1, false: 0}[result.WinnerID == pet2.ID])

	_, art1Leveled, _ := c.svc.AddArtifactXP(userID, artXP1)
	_, art2Leveled, _ := c.svc.AddArtifactXP(opponentID, artXP2)

	// Bond + history for battle
	if result.WinnerID == pet1.ID {
		c.svc.AddBond(pet1, 2)
		c.svc.AddBond(pet2, 1)
		c.svc.RecordHistory(pet1, "pvp_win", "⚔️ **"+pet1.Nickname+"** defeated **"+pet2.Nickname+"** in a PVP battle!")
		c.svc.RecordHistory(pet2, "pvp_loss", "😰 **"+pet2.Nickname+"** lost a PVP battle against **"+pet1.Nickname+"**.")
	} else if result.WinnerID == pet2.ID {
		c.svc.AddBond(pet1, 1)
		c.svc.AddBond(pet2, 2)
		c.svc.RecordHistory(pet1, "pvp_loss", "😰 **"+pet1.Nickname+"** lost a PVP battle against **"+pet2.Nickname+"**.")
		c.svc.RecordHistory(pet2, "pvp_win", "⚔️ **"+pet2.Nickname+"** defeated **"+pet1.Nickname+"** in a PVP battle!")
	} else {
		c.svc.AddBond(pet1, 1)
		c.svc.AddBond(pet2, 1)
		c.svc.RecordHistory(pet1, "pvp_draw", "🤝 **"+pet1.Nickname+"** fought **"+pet2.Nickname+"** to a draw!")
	}

	_ = c.svc.UpdatePet(pet1)
	_ = c.svc.UpdatePet(pet2)
	embedDesc := strings.Join(result.Log, "\n")
	embed := components.Embed(i18n.T("pets.battle.arena_title", lang), embedDesc, 0x2ecc71)
	artLine1 := fmt.Sprintf("🔮 Artifact +%d XP", artXP1)
	if art1Leveled {
		artLine1 += " ⬆️"
	}
	artLine2 := fmt.Sprintf("🔮 Artifact +%d XP", artXP2)
	if art2Leveled {
		artLine2 += " ⬆️"
	}

	if result.WinnerID == pet1.ID {
		embed.Color = 0x2ecc71
		embedDesc += fmt.Sprintf("\n\n🏆 **%s** wins! ELO: %s (%+d) | %s (%+d)\n📊 Weekly: +%d | +%d\n%s | %s",
			MentionUser(userID), strconv.Itoa(pet1.Elo), diff1, strconv.Itoa(pet2.Elo), diff2, weeklyScoreA, weeklyScoreB, artLine1, artLine2)
	} else if result.WinnerID == pet2.ID {
		embed.Color = 0xe74c3c
		embedDesc += fmt.Sprintf("\n\n🏆 **%s** wins! ELO: %s (%+d) | %s (%+d)\n📊 Weekly: +%d | +%d\n%s | %s",
			MentionUser(opponentID), strconv.Itoa(pet2.Elo), diff2, strconv.Itoa(pet1.Elo), diff1, weeklyScoreA, weeklyScoreB, artLine1, artLine2)
	} else {
		embed.Color = 0xf39c12
		embedDesc += fmt.Sprintf("\n\n🤝 Draw! ELO: %s (%+d) | %s (%+d)\n📊 Weekly: +%d | +%d\n%s | %s",
			strconv.Itoa(pet1.Elo), diff1, strconv.Itoa(pet2.Elo), diff2, weeklyScoreA, weeklyScoreB, artLine1, artLine2)
	}

	if art1Leveled || art2Leveled {
		embedDesc += "\n\n💠 Use `/artifact` to assign your new stat point!"
	}
	embed.Description = embedDesc

	sid := interaction.ToInt64(i.GuildID)
	if sid != 0 {
		_, _ = c.svc.GetServerElo(pet1.ID, sid)
		_, _ = c.svc.GetServerElo(pet2.ID, sid)
		_ = c.svc.UpdateServerElo(pet1.ID, sid, pet1.Elo)
		_ = c.svc.UpdateServerElo(pet2.ID, sid, pet2.Elo)
	}

	unlocks, _ := c.svc.CheckAndUnlock(userID)
	if len(unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, unlocks)
	}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))

	// Interaction trigger after battle
	if result.WinnerID == pet1.ID {
		c.tryInteraction(b, i, pet1, "battle")
	} else if result.WinnerID == pet2.ID {
		c.tryInteraction(b, i, pet2, "battle")
	}
}

func (c *Cog) onBattleDecline(b *interaction.Bot, i *discordgo.InteractionCreate) {
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "🏃 Battle declined.", Flags: discordgo.MessageFlagsEphemeral},
	})
}

// ─── Hatch Command ───────────────────────────────────────────

func (c *Cog) onHatchCommand(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	c.hatchEgg(b, i, userID, lang)
}

func (c *Cog) onHatchPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	c.hatchEggMessage(b, s, m, userID, lang)
}

func (c *Cog) hatchEgg(b *interaction.Bot, i *discordgo.InteractionCreate, userID int64, lang string) {
	eggType, biome := c.findEgg(userID)
	if eggType == "" {
		interaction.RespondError(b, i, lang, "pets.hatch.no_egg")
		return
	}
	if !c.svc.CanCreatePet(userID) {
		interaction.RespondError(b, i, lang, "pets.hatch.no_slots")
		return
	}

	petType := petsvc.RollGacha("", biome)
	pet, err := c.svc.CreatePet(userID, petType)
	if err != nil || pet == nil {
		interaction.RespondError(b, i, lang, "pets.hatch.error")
		return
	}

	_ = c.store.DB.Exec(
		`UPDATE inventory SET quantity = quantity - 1 WHERE user_id = ? AND item_id = ? AND quantity > 0`,
		userID, eggType,
	)

	pt := petsvc.PetTypes[pet.PetType]
	emoji := "🐾"
	if pt != nil {
		emoji = pt.Emoji
	}
	pTrait := petsvc.PersonalityTraits[pet.Personality]
	traitKey := "pets.personality.brave.name"
	if pTrait != nil {
		traitKey = "pets.personality." + pet.Personality + ".name"
	}
	traitName := i18n.T(traitKey, lang)

	biomeName := i18n.T("biomes."+biome, lang)
	eggName := items.DisplayName(eggType)

	desc := i18n.T("pets.hatch.success_desc", lang, map[string]any{
		"emoji": emoji, "pet": pet.Nickname, "type": petType, "personality": traitName, "biome": biomeName, "egg": eggName,
	})

	embed := components.Embed(
		i18n.T("pets.hatch.success_title", lang),
		desc,
		0xf1c40f,
	)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) hatchEggMessage(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message, userID int64, lang string) {
	eggType, biome := c.findEgg(userID)
	if eggType == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.hatch.no_egg", lang))
		return
	}
	if !c.svc.CanCreatePet(userID) {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.hatch.no_slots", lang))
		return
	}
	petType := petsvc.RollGacha("", biome)
	pet, err := c.svc.CreatePet(userID, petType)
	if err != nil || pet == nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.hatch.error", lang))
		return
	}
	_ = c.store.DB.Exec(
		`UPDATE inventory SET quantity = quantity - 1 WHERE user_id = ? AND item_id = ? AND quantity > 0`,
		userID, eggType,
	)
	pt := petsvc.PetTypes[pet.PetType]
	emoji := "🐾"
	if pt != nil {
		emoji = pt.Emoji
	}
	biomeName := i18n.T("biomes."+biome, lang)
	_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.hatch.prefix_success", lang, map[string]any{"emoji": emoji, "pet": pet.Nickname, "type": petType, "biome": biomeName}))
}

var eggBiomes = map[string]string{
	"forest_egg":   "forest",
	"cave_egg":     "cave",
	"desert_egg":   "desert",
	"mountain_egg": "mountain",
	"ocean_egg":    "ocean",
	"tundra_egg":   "tundra",
	"volcano_egg":  "volcano",
}

func (c *Cog) findEgg(userID int64) (string, string) {
	var inv []model.Inventory
	eggIDs := make([]string, 0, len(eggBiomes))
	for id := range eggBiomes {
		eggIDs = append(eggIDs, id)
	}
	c.store.DB.Where("user_id = ? AND item_id IN ? AND quantity > 0", userID, eggIDs).Find(&inv)
	priority := []string{"volcano_egg", "tundra_egg", "ocean_egg", "mountain_egg", "desert_egg", "cave_egg", "forest_egg"}
	for _, id := range priority {
		for _, iv := range inv {
			if iv.ItemID == id {
				return id, eggBiomes[id]
			}
		}
	}
	return "", ""
}

// ─── Skill Selection ──────────────────────────────────────────

func (c *Cog) onSkillSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	petID, _ := strconv.ParseInt(rest[0], 10, 64)
	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil || pet.SkillPoints <= 0 {
		interaction.RespondError(b, i, lang, "pets.skills.no_points")
		return
	}

	// Determine which slot to fill
	existing, _ := c.svc.GetPetSkills(petID)
	usedSlots := make(map[int]bool)
	for _, s := range existing {
		usedSlots[s.Slot] = true
	}
	slot := -1
	for s := 0; s < 5; s++ {
		if !usedSlots[s] {
			slot = s
			break
		}
	}
	if slot < 0 {
		interaction.RespondError(b, i, lang, "pets.skills.maxed")
		return
	}

	rarity := petTypeRarity(pet)
	battleOpts := petsvc.RandomBattleSkills(rarity, 2)
	utilOpts := petsvc.RandomUtilitySkills(rarity, 1)
	options := append(battleOpts, utilOpts...)

	if len(options) == 0 {
		interaction.RespondError(b, i, lang, "pets.skills.none_available")
		return
	}

	opts := make([]discordgo.SelectMenuOption, 0, len(options))
	for _, sk := range options {
		opts = append(opts, discordgo.SelectMenuOption{
			Label: sk.Emoji + " " + sk.Name,
			Value: sk.ID + ":" + itoa2(slot),
			Description: truncate(sk.Description, 50),
		})
	}

	embed := components.Embed(
		i18n.T("pets.skills.choose_title", lang, map[string]any{"level": (slot+1)*10}),
		i18n.T("pets.skills.choose_desc", lang),
		0x9b59b6,
	)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, []discordgo.MessageComponent{
			components.ActionRow(
				discordgo.SelectMenu{
					CustomID:    components.Encode("pets", "skill_choose", rest[0]),
					Placeholder: i18n.T("pets.skills.select", lang),
					Options:     opts,
				},
			),
		}))
}

func (c *Cog) onSkillChoose(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		return
	}
	_, _, rest := components.Decode(data.CustomID)
	if len(rest) == 0 {
		return
	}
	petID, _ := strconv.ParseInt(rest[0], 10, 64)

	// Value format: "skillID:slot"
	parts := strings.SplitN(data.Values[0], ":", 2)
	if len(parts) != 2 {
		return
	}
	skillID := parts[0]
	slot, _ := strconv.Atoi(parts[1])

	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}

	if err := c.svc.SelectSkill(petID, slot, skillID); err != nil {
		interaction.RespondError(b, i, lang, "pets.skills.error")
		return
	}
	_ = c.svc.SpendSkillPoint(pet)

	skDef, ok := petsvc.AllPetSkills[skillID]
	if !ok {
		skDef = &petsvc.PetSkill{Name: skillID, Emoji: "⭐"}
	}
	c.svc.RecordHistory(pet, "skill_learned",
		"⭐ **"+pet.Nickname+"** learned **"+skDef.Emoji+" "+skDef.Name+"** at level "+itoa2((slot+1)*10)+"!")

	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("pets.skills.learned", lang, map[string]any{"name": skDef.Emoji + " " + skDef.Name}),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// ─── Interaction Choice ───────────────────────────────────────

func (c *Cog) onInteractionChoice(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	data := i.MessageComponentData()
	_, _, rest := components.Decode(data.CustomID)
	if len(rest) < 2 {
		return
	}
	petID, _ := strconv.ParseInt(rest[0], 10, 64)
	choiceID := rest[1]

	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}

	reward := petsvc.ResolveInteraction(choiceID)
	if reward == nil {
		interaction.RespondError(b, i, lang, "pets.interact.invalid")
		return
	}

	c.svc.AddBond(pet, reward.BondReward)
	if reward.XPReward > 0 {
		c.svc.AddXP(pet, reward.XPReward)
	}
	// Resolve choice detail via i18n
	detailKey := fmt.Sprintf("pets.interact.%s.choices.%s.detail", findInteractionID(choiceID), choiceID)
	detail := i18n.T(detailKey, lang, map[string]any{"name": pet.Nickname})
	if detail == detailKey {
		detail = i18n.T("pets.interact.default_detail", lang, map[string]any{"name": pet.Nickname})
	}
	c.svc.RecordHistory(pet, "interaction", detail)
	_ = c.svc.UpdatePet(pet)

	if reward.ItemReward != "" {
		_ = c.store.DB.Exec(
			`INSERT INTO inventory (user_id, item_id, quantity) VALUES (?, ?, 1)
			 ON CONFLICT(user_id, item_id) DO UPDATE SET quantity = quantity + 1`,
			pet.UserID, reward.ItemReward,
		)
	}

	_ = c.store.SetCooldown(pet.UserID, "pet_interaction")

	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: detail,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// ─── Interaction Trigger ──────────────────────────────────────

func (c *Cog) tryInteraction(b *interaction.Bot, i *discordgo.InteractionCreate, pet *model.UserPet, context string) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	ready, _ := c.store.CheckCooldown(pet.UserID, "pet_interaction", 180*time.Minute)
	if !ready {
		return
	}
	ir := petsvc.MaybeTriggerInteraction(pet, context)
	if ir == nil {
		return
	}

	// Resolve intro text via i18n
	intro := i18n.T(ir.IntroKey(pet.Personality), lang)
	if intro == ir.IntroKey(pet.Personality) {
		intro = i18n.T(ir.GenericIntroKey(), lang)
	}

	opts := make([]discordgo.SelectMenuOption, 0, len(ir.Choices))
	for _, ch := range ir.Choices {
		label := i18n.T(ch.ChoiceLabelKey(), lang)
		if label == ch.ChoiceLabelKey() {
			label = ch.Emoji + " " + ch.ID
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label: ch.Emoji + " " + label,
			Value: ch.ID,
		})
	}
	if len(opts) == 0 {
		return
	}

	embed := components.Embed(
		i18n.T("pets.interact.title", lang, map[string]any{"name": pet.Nickname}),
		intro,
		0x9b59b6,
	)
	_, _ = b.Session.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			components.ActionRow(
				discordgo.SelectMenu{
					CustomID:    components.Encode("pets", "interact", strconv.FormatInt(pet.ID, 10)),
					Placeholder: i18n.T("pets.interact.placeholder", lang),
					Options:     opts,
				},
			),
		},
		Flags: discordgo.MessageFlagsEphemeral,
	})
}

// ─── Helpers ──────────────────────────────────────────────────

func (c *Cog) petToBattlePet(pet *model.UserPet) *battle.BattlePet {
	pt := petsvc.PetTypes[pet.PetType]
	emoji := "🐾"
	if pt != nil {
		emoji = pt.Emoji
	}
	skills, _ := c.svc.GetPetSkills(pet.ID)
	skillIDs := make([]string, len(skills))
	for i, s := range skills {
		skillIDs[i] = s.SkillID
	}
	return &battle.BattlePet{
		ID: pet.ID, Nickname: pet.Nickname, Emoji: emoji,
		Level: pet.Level, HP: pet.HP, MaxHP: pet.MaxHP,
		Atk: pet.Atk, Defense: pet.Defense, Speed: pet.Speed,
		DGE: pet.DGE, ACC: pet.ACC, CritC: pet.CritC, CritD: pet.CritD, SpcC: pet.SpcC,
		Skills: skillIDs,
	}
}

func (c *Cog) getActiveModID(serverID int64) (string, error) {
	mod, err := c.svc.GetActiveModifier(serverID)
	if err != nil || mod == nil {
		return "", err
	}
	return mod.Modifier, nil
}

func (c *Cog) applyArtifacts(bp1, bp2 *battle.BattlePet, userID1, userID2 int64, modID string) {
	c.svc.ApplyArtifactToBattle(userID1, bp1, modID)
	c.svc.ApplyArtifactToBattle(userID2, bp2, modID)
}

func battleResultToFloat(result *battle.BattleResult, pet1ID int64) float64 {
	if result.WinnerID == pet1ID {
		return 1.0
	} else if result.WinnerID == 0 {
		return 0.5
	}
	return 0.0
}

func MentionUser(id int64) string {
	return "<@" + strconv.FormatInt(id, 10) + ">"
}

func petTypeRarity(pet *model.UserPet) string {
	if pt, ok := petsvc.PetTypes[pet.PetType]; ok {
		return pt.Rarity
	}
	return "common"
}

func findInteractionID(choiceID string) string {
	// Simple lookup: the choice IDs are prefixed by interaction type
	// We match known prefix patterns
	typeMap := map[string]string{
		"fetch": "play_time", "tug": "play_time", "ignore": "play_time",
		"feed_treat": "snack_time", "share_meal": "snack_time", "cook": "snack_time",
		"explore": "explore_together", "follow": "explore_together",
		"brush": "grooming", "bath": "grooming", "massage": "grooming",
		"spar": "training", "teach": "training", "praise": "training",
		"stand_together": "rescue", "investigate": "rescue", "retreat": "rescue",
	}
	if id, ok := typeMap[choiceID]; ok {
		return id
	}
	return "play_time"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func itoa2(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	return out
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// ─── Artifact UI ────────────────────────────────────────────────

func (c *Cog) onArtifactMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.artifactView(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) artifactView(userID int64, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	a, err := c.svc.GetArtifact(userID)
	if err != nil {
		a, err = c.svc.CreateArtifact(userID)
		if err != nil {
			return components.Embed("Artifact", "Could not create artifact.", 0xe74c3c), nil
		}
	}

	statIDs := []string{a.Stat1, a.Stat2, a.Stat3}
	statLvls := []int{a.Stat1Lvl, a.Stat2Lvl, a.Stat3Lvl}

	statLines := ""
	for i, sid := range statIDs {
		def := petsvc.GetArtifactStat(sid)
		emoji := "❓"
		name := sid
		if def != nil {
			emoji = def.Emoji
			name = def.Name
		}
		statLines += fmt.Sprintf("%s **%s** — Lvl %d\n", emoji, name, statLvls[i])
	}

	xpBar := buildArtifactXPBar(a.XP, petsvc.ArtifactXPForLevel(a.Level), a.Level)

	unspentLine := ""
	allocBtns := []discordgo.MessageComponent{}
	if a.UnspentPoints > 0 {
		unspentLine = fmt.Sprintf("\n\n⚡ **%d unspent stat point(s)!** Choose a stat to improve:", a.UnspentPoints)
		for i, sid := range statIDs {
			def := petsvc.GetArtifactStat(sid)
			emoji := "❓"
			if def != nil {
				emoji = def.Emoji
			}
			allocBtns = append(allocBtns,
				components.Button(emoji, components.Encode("pets", "artifact_stat_choose", strconv.FormatInt(int64(i), 10)), discordgo.PrimaryButton))
		}
	}

	desc := fmt.Sprintf("**Level %d** %s\n\n**Stats:**\n%s%s",
		a.Level, xpBar, statLines, unspentLine)
	embed := components.Embed("💠 Pet Artifact", desc, 0x9b59b6)

	comps := []discordgo.MessageComponent{}
	if len(allocBtns) > 0 {
		comps = append(comps, components.ActionRow(allocBtns...))
	}
	comps = append(comps, components.ActionRow(
		components.Button("🔄 Reset", components.Encode("pets", "artifact_reset"), discordgo.DangerButton),
	))
	return embed, comps
}

func buildArtifactXPBar(xp, needed, level int) string {
	if level >= petsvc.ArtifactMaxLevel {
		return "`[██████████]` **MAX**"
	}
	if needed <= 0 {
		needed = 1
	}
	filled := xp * 10 / needed
	if filled > 10 {
		filled = 10
	}
	bar := "["
	for i := 0; i < 10; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += fmt.Sprintf("] %d/%d", xp, needed)
	return "`" + bar + "`"
}

func (c *Cog) onArtifactView(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.artifactView(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onArtifactReset(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	var inv model.Inventory
	err := c.store.DB.Where("user_id = ? AND item_id = ?", userID, "artifact_shard").First(&inv).Error
	if err != nil || inv.Quantity < 1 {
		interaction.RespondError(b, i, lang, "artifact.no_shard")
		return
	}

	cb, err := c.store.GetBalance(userID)
	if err != nil || cb < items.Get("artifact_shard").Price {
		interaction.RespondError(b, i, lang, "artifact.no_money")
		return
	}

	_, _ = c.store.UpdateBalance(userID, -items.Get("artifact_shard").Price)
	_ = c.store.DB.Exec(
		`UPDATE inventory SET quantity = quantity - 1 WHERE user_id = ? AND item_id = ? AND quantity > 0`,
		userID, "artifact_shard",
	)
	_, _ = c.svc.ResetArtifact(userID)

	embed, comps := c.artifactView(userID, lang)
	embed.Description = "✅ Artifact reset! New random stats assigned.\n\n" + embed.Description
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}



func (c *Cog) onArtifactStatChoose(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	_, _, rest := components.Decode(i.MessageComponentData().CustomID)

	statPos := 0
	if len(rest) > 0 {
		statPos, _ = strconv.Atoi(rest[0])
	}
	if statPos < 0 || statPos > 2 {
		return
	}

	_, err := c.svc.LevelArtifactStat(userID, statPos)
	if err != nil {
		interaction.RespondError(b, i, lang, "artifact.error")
		return
	}

	a, _ := c.svc.GetArtifact(userID)
	statIDs := []string{a.Stat1, a.Stat2, a.Stat3}
	def := petsvc.GetArtifactStat(statIDs[statPos])
	name := statIDs[statPos]
	if def != nil {
		name = def.Emoji + " " + def.Name
	}
	lvl := 0
	switch statPos {
	case 0:
		lvl = a.Stat1Lvl
	case 1:
		lvl = a.Stat2Lvl
	case 2:
		lvl = a.Stat3Lvl
	}

	if a.UnspentPoints > 0 {
		embed, comps := c.artifactView(userID, lang)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
	} else {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "✅ " + name + " improved to Lvl " + itoa2(lvl) + "!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

// ─── Weekly Leaderboard UI ─────────────────────────────────────

func (c *Cog) onWeeklyPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	args := strings.Fields(m.Content)
	if len(args) < 2 {
		return
	}
	sub := strings.ToLower(args[1])
	if sub == "history" || sub == "prev" || sub == "last" {
		userID := interaction.ToInt64(m.Author.ID)
		embed := c.weeklyHistoryEmbed(userID, interaction.ToInt64(m.GuildID), lang)
		_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
	}
}

func (c *Cog) onWeeklyLeaderboard(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.weeklyLeaderboardEmbed(i, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) weeklyLeaderboardEmbed(i *discordgo.InteractionCreate, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	serverID := interaction.ToInt64(i.GuildID)
	userID := interaction.ToInt64(interaction.UserID(i))
	ranks, _ := c.svc.GetWeeklyLeaderboard(i.GuildID, 10)

	lines := ""
	if len(ranks) == 0 {
		lines = i18n.T("weekly.empty", lang)
	} else {
		for pos, r := range ranks {
			emoji := ""
			switch pos {
			case 0:
				emoji = "👑"
			case 1:
				emoji = "🥈"
			case 2:
				emoji = "🥉"
			default:
				emoji = strconv.Itoa(pos+1) + "."
			}
			lines += fmt.Sprintf("%s <@%d> — **%d** pts (W:%d L:%d)\n", emoji, r.UserID, r.Score, r.Wins, r.Losses)
		}
	}

	rankPos, _ := c.svc.GetRankPosition(userID, serverID)
	var personalLine string
	if rankPos > 0 {
		wr, err := c.svc.GetWeeklyRank(userID, serverID)
		if err == nil {
			personalLine = fmt.Sprintf("\n**Your rank:** #%d — **%d** pts (W:%d L:%d)", rankPos, wr.Score, wr.Wins, wr.Losses)
		}
	}

	mod, _ := c.svc.EnsureWeeklyModifier(serverID)
	modLine := ""
	if mod != nil {
		modDef := petsvc.GetModifierDef(mod.Modifier)
		if modDef != nil {
			boosted := ""
			for _, s := range petsvc.SplitModStats(mod.Boosted) {
				if def := petsvc.GetArtifactStat(s); def != nil {
					boosted += def.Emoji + " "
				}
			}
			nerfed := ""
			for _, s := range petsvc.SplitModStats(mod.Nerfed) {
				if def := petsvc.GetArtifactStat(s); def != nil {
					nerfed += def.Emoji + " "
				}
			}
			modLine = fmt.Sprintf("\n\n**Weekly Modifier:** %s %s\nBoosted: %s | Nerfed: %s",
				modDef.Emoji, modDef.Name, boosted, nerfed)
		}
	}

	desc := lines + modLine + personalLine
	embed := components.Embed(i18n.T("weekly.title", lang), desc, 0xf1c40f)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("leaderboard.btn_refresh", lang), components.Encode("pets", "weekly_refresh"), discordgo.PrimaryButton),
			components.Button("📜 History", components.Encode("pets", "weekly_history"), discordgo.SecondaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onWeeklyHistory(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	serverID := interaction.ToInt64(i.GuildID)
	embed := c.weeklyHistoryEmbed(userID, serverID, lang)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) onWeeklyRefresh(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.weeklyLeaderboardEmbed(i, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) weeklyHistoryEmbed(userID, serverID int64, lang string) *discordgo.MessageEmbed {
	history, _ := c.svc.GetWeeklyRankHistory(userID, serverID, 5)

	desc := ""
	if len(history) == 0 {
		desc = i18n.T("weekly.history_empty", lang)
	} else {
		for _, r := range history {
			rank := 0
			var count int64
			_ = c.store.DB.Model(&model.WeeklyRank{}).
				Where("server_id = ? AND week_id = ? AND score > ?", serverID, r.WeekID, r.Score).
				Count(&count)
			rank = int(count) + 1

			desc += fmt.Sprintf("**%s** — #%d — %d pts (W:%d L:%d)\n", r.WeekID, rank, r.Score, r.Wins, r.Losses)
		}
	}

	embed := components.Embed(i18n.T("weekly.history_title", lang, map[string]any{"user": "<@" + strconv.FormatInt(userID, 10) + ">"}), desc, 0x9b59b6)
	return embed
}
