package pets

import (
	"fmt"
	"strconv"
	"strings"

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
	r.Prefix("pets", c.onPrefixMenu)
	r.Prefix("pet", c.onPrefixMenu)
	r.Component("pets", "menu", c.onMenu)
	r.Component("pets", "pet", c.onPetDetail)
	r.Component("pets", "feed", c.onFeed)
	r.Component("pets", "rename_btn", c.onRenameOpen)
	r.Modal("pets", "rename_submit", c.onRenameSubmit)
	r.Component("pets", "delete", c.onDelete)
	r.Component("pets", "battle", c.onBattleSelect)
	r.Component("pets", "battle_accept", c.onBattleAccept)
	r.Component("pets", "battle_decline", c.onBattleDecline)
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
	embed := components.Embed(
		fmt.Sprintf("%s %s", emoji, pet.Nickname),
		fmt.Sprintf("**%s** | %s | Lvl %d\n\nHP: %d/%d | ATK: %d | DEF: %d | SPD: %d\nDGE: %d | ACC: %d | CRIT: %d/%0.1f\nELO: %d | XP: %d",
			rarity, emoji, pet.Level,
			pet.HP, pet.MaxHP, pet.Atk, pet.Defense, pet.Speed,
			pet.DGE, pet.ACC, pet.CritC, pet.CritD,
			pet.Elo, pet.XP),
		0x3498db,
	)
	act := components.Encode("pets", "")
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("pets.rename.success", lang), components.Encode("pets", "rename_btn", petIDStr), discordgo.PrimaryButton),
			components.Button(i18n.T("pets.feed.miam_title", lang), components.Encode("pets", "feed", petIDStr), discordgo.SuccessButton),
			components.Button(i18n.T("pets.battle.arena_title", lang), components.Encode("pets", "battle", petIDStr), discordgo.DangerButton),
			components.Button("🗑️", components.Encode("pets", "delete", petIDStr), discordgo.SecondaryButton),
		),
	}
	_ = components.Encode("pets", "back")
	_ = act
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
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
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🍖 **%s** has been fed! (+5 HP)", pet.Nickname),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	pet.HP = min(pet.MaxHP, pet.HP+5)
	_ = c.svc.UpdatePet(pet)
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

	bp1 := petToBattlePet(pet1)
	bp2 := petToBattlePet(pet2)
	result := battle.Simulate(bp1, bp2)

	_ = c.svc.UpdatePet(pet1)
	_ = c.svc.UpdatePet(pet2)

	diff1, diff2 := c.svc.UpdateElo(pet1, pet2, battleResultToFloat(result, pet1.ID))
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
}

func (c *Cog) onBattleDecline(b *interaction.Bot, i *discordgo.InteractionCreate) {
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: "🏃 Battle declined.", Flags: discordgo.MessageFlagsEphemeral},
	})
}

func petToBattlePet(pet *model.UserPet) *battle.BattlePet {
	pt := petsvc.PetTypes[pet.PetType]
	emoji := "🐾"
	if pt != nil {
		emoji = pt.Emoji
	}
	return &battle.BattlePet{
		ID: pet.ID, Nickname: pet.Nickname, Emoji: emoji,
		Level: pet.Level, HP: pet.HP, MaxHP: pet.MaxHP,
		Atk: pet.Atk, Defense: pet.Defense, Speed: pet.Speed,
		DGE: pet.DGE, ACC: pet.ACC, CritC: pet.CritC, CritD: pet.CritD, SpcC: pet.SpcC,
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
