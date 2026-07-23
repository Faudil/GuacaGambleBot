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
		pTrait = personality.Emoji + " *" + personality.Name + "*"
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
	result := battle.Simulate(bp1, bp2)

	_ = c.svc.UpdatePet(pet1)
	_ = c.svc.UpdatePet(pet2)

	diff1, diff2 := c.svc.UpdateElo(pet1, pet2, battleResultToFloat(result, pet1.ID))

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
	if result.WinnerID == pet1.ID {
		embed.Color = 0x2ecc71
		embedDesc += fmt.Sprintf("\n\n🏆 **%s** wins! ELO: %s (%+d) | %s (%+d)",
			MentionUser(userID), strconv.Itoa(pet1.Elo), diff1, strconv.Itoa(pet2.Elo), diff2)
	} else if result.WinnerID == pet2.ID {
		embed.Color = 0xe74c3c
		embedDesc += fmt.Sprintf("\n\n🏆 **%s** wins! ELO: %s (%+d) | %s (%+d)",
			MentionUser(opponentID), strconv.Itoa(pet2.Elo), diff2, strconv.Itoa(pet1.Elo), diff1)
	} else {
		embed.Color = 0xf39c12
		embedDesc += fmt.Sprintf("\n\n🤝 Draw! ELO: %s (%+d) | %s (%+d)",
			strconv.Itoa(pet1.Elo), diff1, strconv.Itoa(pet2.Elo), diff2)
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
	eggType := c.findEgg(userID)
	if eggType == "" {
		interaction.RespondError(b, i, lang, "hatch.no_egg")
		return
	}
	if !c.svc.CanCreatePet(userID) {
		interaction.RespondError(b, i, lang, "hatch.no_slots")
		return
	}

	petType := petsvc.RollGacha("")
	pet, err := c.svc.CreatePet(userID, petType)
	if err != nil || pet == nil {
		interaction.RespondError(b, i, lang, "hatch.error")
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
	traitName := ""
	if pTrait != nil {
		traitName = pTrait.Emoji + " " + pTrait.Name
	}

	desc := fmt.Sprintf("%s The egg begins to crack...\n\n**%s %s** hatched!\n%s | Lvl 1 | %s\n\nUse `/pets` to see your new companion!",
		emoji, emoji, pet.Nickname, petType, traitName)

	embed := components.Embed(
		i18n.T("hatch.success_title", lang),
		desc,
		0xf1c40f,
	)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) hatchEggMessage(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message, userID int64, lang string) {
	eggType := c.findEgg(userID)
	if eggType == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("hatch.no_egg", lang))
		return
	}
	if !c.svc.CanCreatePet(userID) {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("hatch.no_slots", lang))
		return
	}
	petType := petsvc.RollGacha("")
	pet, err := c.svc.CreatePet(userID, petType)
	if err != nil || pet == nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("hatch.error", lang))
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
	_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s **%s** hatched! %s", emoji, pet.Nickname, petType))
}

func (c *Cog) findEgg(userID int64) string {
	var inv []model.Inventory
	c.store.DB.Where("user_id = ? AND (item_id = ? OR item_id = ?) AND quantity > 0",
		userID, "mystery_egg", "season_egg").Find(&inv)
	for _, iv := range inv {
		if iv.ItemID == "season_egg" {
			return "season_egg"
		}
	}
	for _, iv := range inv {
		if iv.ItemID == "mystery_egg" {
			return "mystery_egg"
		}
	}
	return ""
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
		interaction.RespondError(b, i, lang, "skills.no_points")
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
		interaction.RespondError(b, i, lang, "skills.maxed")
		return
	}

	rarity := petTypeRarity(pet)
	battleOpts := petsvc.RandomBattleSkills(rarity, 2)
	utilOpts := petsvc.RandomUtilitySkills(rarity, 1)
	options := append(battleOpts, utilOpts...)

	if len(options) == 0 {
		interaction.RespondError(b, i, lang, "skills.none_available")
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
		i18n.T("skills.choose_title", lang, map[string]any{"level": (slot+1)*10}),
		i18n.T("skills.choose_desc", lang),
		0x9b59b6,
	)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, []discordgo.MessageComponent{
			components.ActionRow(
				discordgo.SelectMenu{
					CustomID:    components.Encode("pets", "skill_choose", rest[0]),
					Placeholder: i18n.T("skills.select", lang),
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
		interaction.RespondError(b, i, lang, "skills.error")
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
			Content: i18n.T("skills.learned", lang, map[string]any{"name": skDef.Emoji + " " + skDef.Name}),
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

	interactionData := petsvc.ResolveInteraction(pet, choiceID)
	if interactionData == nil {
		interaction.RespondError(b, i, lang, "interact.invalid")
		return
	}

	c.svc.AddBond(pet, interactionData.BondReward)
	if interactionData.XPReward > 0 {
		c.svc.AddXP(pet, interactionData.XPReward)
	}
	c.svc.RecordHistory(pet, "interaction", interactionData.Detail)
	_ = c.svc.UpdatePet(pet)

	if interactionData.ItemReward != "" {
		_ = c.store.DB.Exec(
			`INSERT INTO inventory (user_id, item_id, quantity) VALUES (?, ?, 1)
			 ON CONFLICT(user_id, item_id) DO UPDATE SET quantity = quantity + 1`,
			pet.UserID, interactionData.ItemReward,
		)
	}

	// Set cooldown
	_ = c.store.SetCooldown(pet.UserID, "pet_interaction")

	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: interactionData.Detail,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// ─── Interaction Trigger ──────────────────────────────────────

func (c *Cog) tryInteraction(b *interaction.Bot, i *discordgo.InteractionCreate, pet *model.UserPet, context string) {
	ready, _ := c.store.CheckCooldown(pet.UserID, "pet_interaction", 180*time.Minute)
	if !ready {
		return
	}
	interaction := petsvc.MaybeTriggerInteraction(pet, context)
	if interaction == nil {
		return
	}

	opts := make([]discordgo.SelectMenuOption, 0, len(interaction.Choices))
	for _, ch := range interaction.Choices {
		opts = append(opts, discordgo.SelectMenuOption{
			Label: ch.Emoji + " " + ch.Label,
			Value: ch.ID,
		})
	}
	if len(opts) == 0 {
		return
	}

	embed := components.Embed("💬 "+pet.Nickname+" wants your attention!", interaction.Intro, 0x9b59b6)
	_, _ = b.Session.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			components.ActionRow(
				discordgo.SelectMenu{
					CustomID:    components.Encode("pets", "interact", strconv.FormatInt(pet.ID, 10)),
					Placeholder: "What do you do?",
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
