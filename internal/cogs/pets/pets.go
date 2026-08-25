package pets

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	furnituresvc "guacagamblebot/internal/service/furniture"
	invsvc "guacagamblebot/internal/service/inventory"
	jsvc "guacagamblebot/internal/service/journal"
	npcsvc "guacagamblebot/internal/service/npcs"
	petsvc "guacagamblebot/internal/service/pets"
	sansvc "guacagamblebot/internal/service/sanctuary"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
)

type Cog struct {
	store  *store.Store
	cfg    *config.Config
	svc    *petsvc.Service
	sanSvc *sansvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	def := universe.Get(cfg.Universe)
	if def == nil {
		def = universe.Get("hoakhaven")
	}
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)
	c := &Cog{store: s, cfg: cfg, svc: petsvc.New(s, cfg, npcSvc), sanSvc: sansvc.New(s, cfg)}
	r.Slash("pets", "Gérer vos familiers", c.onSlashMenu)
	r.Slash("pet", "Gérer vos familiers", c.onSlashMenu)
	r.Slash("hatch", "Éclore un œuf de familier", c.onHatchCommand)
	r.Prefix("pets", c.onPrefixMenu)
	r.Prefix("pet", c.onPrefixMenu)
	r.Prefix("hatch", c.onHatchPrefix)
	r.Component("pets", "menu", c.onMenu)
	r.Component("pets", "hatch", c.onHatchButton)
	r.Component("pets", "pet", c.onPetDetail)
	r.Component("pets", "feed", c.onFeedMenu)
	r.Component("pets", "feed_select", c.onFeedSelect)
	r.Component("pets", "rename_btn", c.onRenameOpen)
	r.Modal("pets", "rename_submit", c.onRenameSubmit)
	r.Component("pets", "delete", c.onDelete)
	r.Component("pets", "delete_confirm", c.onDeleteConfirm)
	r.Component("pets", "battle", c.onBattleSelect)
	r.Component("pets", "battle_accept", c.onBattleAccept)
	r.Component("pets", "battle_decline", c.onBattleDecline)
	r.Component("pets", "skills", c.onSkillSelect)
	r.Component("pets", "skill_choose", c.onSkillChoose)
	r.Component("pets", "activate", c.onPetActivate)
	r.Component("pets", "interact", c.onInteractionChoice)
	r.Slash("heal", "Soigner ton familier actif", c.onHealCommand)
	r.Prefix("heal", c.onHealPrefix)
	r.Component("pets", "heal", c.onHealButton)
	r.Slash("play", "Jouer avec ton familier actif", c.onPlayCommand)
	r.Prefix("play", c.onPlayPrefix)
	r.Prefix("jouer", c.onPlayPrefix)
	r.Slash("artifact", "Gérer votre artefact de familier", c.onArtifactMenu)
	r.Slash("weekly", "Classement hebdomadaire des familiers", c.onWeeklyLeaderboard)
	r.Prefix("weekly", c.onWeeklyPrefix)
	r.Component("pets", "artifact_view", c.onArtifactView)
	r.Component("pets", "artifact_reset", c.onArtifactReset)
	r.Component("pets", "artifact_stat_choose", c.onArtifactStatChoose)
	r.Component("pets", "weekly_refresh", c.onWeeklyRefresh)
	r.Component("pets", "weekly_history", c.onWeeklyHistory)
	r.SlashWithOptions("pet-retire", "Send a pet to your sanctuary", []*discordgo.ApplicationCommandOption{
		{Type: discordgo.ApplicationCommandOptionString, Name: "pet", Description: "Pet to retire (autocomplete)", Required: true, Autocomplete: true},
	}, c.onSlashRetire)
	r.SlashWithOptions("pet-recall", "Recall a pet from your sanctuary", []*discordgo.ApplicationCommandOption{
		{Type: discordgo.ApplicationCommandOptionString, Name: "pet", Description: "Pet to recall (autocomplete)", Required: true, Autocomplete: true},
	}, c.onSlashRecall)
	r.Prefix("retire", c.onPrefixRetire)
	r.Prefix("recall", c.onPrefixRecall)
	r.Component("pets", "retire_btn", c.onRetireClick)
	r.Component("pets", "retire_confirm", c.onRetireConfirm)
	r.Component("pets", "recall_btn", c.onRecallClick)
	r.Component("pets", "recall_confirm", c.onRecallConfirm)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(b.Session, i, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	embed, comps := c.menuFromUser(userID, interaction.DisplayName(s, m.GuildID, &discordgo.Member{User: m.Author}, userID), lang)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) menu(s interaction.Session, i *discordgo.InteractionCreate, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	userID := interaction.ToInt64(interaction.UserID(i))
	return c.menuFromUser(userID, interaction.DisplayName(s, i.GuildID, i.Member, userID), lang)
}

func (c *Cog) menuFromUser(userID int64, pseudo, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	pets, err := c.svc.GetPets(userID)
	embed := components.Embed(i18n.T("pets.list.title", lang, map[string]any{"name": pseudo}), "", 0x2ecc71)

	comps := []discordgo.MessageComponent{}

	if err != nil || len(pets) == 0 {
		embed.Description = i18n.T("pets.list.no_pets", lang)
	} else {
		// Minimal truncate: Discord SelectMenu max 25 options and embed description 4096
		truncated := false
		displayPets := pets
		if len(pets) > 25 {
			displayPets = pets[:25]
			truncated = true
		}
		lines := make([]string, 0, len(displayPets))
		for _, p := range displayPets {
			lines = append(lines, c.petCardLine(p, lang))
		}
		desc := strings.Join(lines, "\n")
		if len([]rune(desc)) > 4000 {
			desc = string([]rune(desc)[:4000]) + "…"
		}
		if truncated {
			desc += fmt.Sprintf("\n*… +%d more pets (showing 25/%d)*", len(pets)-25, len(pets))
		}
		embed.Description = desc

		opts := make([]discordgo.SelectMenuOption, 0, len(displayPets))
		for _, p := range displayPets {
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
				CustomID:    components.EncodeOwner(userID, "pets", "pet"),
				Placeholder: i18n.T("pets.list.select_placeholder", lang),
				Options:     opts,
			},
		))
	}

	comps = append(comps, components.ActionRow(
		components.Button(i18n.T("pets.list.hatch_btn", lang), components.EncodeOwner(userID, "pets", "hatch"), discordgo.SuccessButton),
	))

	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("pets.list.footer", lang)}
	return embed, comps
}

// petCardLine renders a compact one-line card for a pet in the collection.
func (c *Cog) petCardLine(p model.UserPet, lang string) string {
	pt := petsvc.PetTypes[p.PetType]
	emoji := "🐾"
	if pt != nil {
		emoji = pt.Emoji
	}
	status := i18n.T("pets.list.inactive", lang)
	if p.IsActive {
		status = i18n.T("pets.list.active", lang)
	}
	rarityName := i18n.T("rarities."+petsvc.RarityBonus(petTypeRarityValue(p)), lang)
	hpBar := buildHPBar(p.HP, p.MaxHP)

	line := fmt.Sprintf("%s **%s** %s %s\n", emoji, p.Nickname, status, rarityName)
	line += fmt.Sprintf("  Lvl %d · %s %d/%d HP", p.Level, hpBar, p.HP, p.MaxHP)
	if p.SkillPoints > 0 {
		line += fmt.Sprintf(" · ⚡ %d SP", p.SkillPoints)
	}
	return line
}

func (c *Cog) onMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(b.Session, i, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onPetDetail(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
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
	if err != nil || pet == nil || pet.UserID != userID {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}
	embed, comps := c.petDetail(pet, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// petDetail renders the full pet detail view: stats, skills, history and the
// management buttons (including activating the pet when it is not active).
func (c *Cog) petDetail(pet *model.UserPet, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	userID := pet.UserID
	petIDStr := strconv.FormatInt(pet.ID, 10)
	pt := petsvc.PetTypes[pet.PetType]
	emoji := "🐾"
	if pt != nil {
		emoji = pt.Emoji
	}
	rarity := petTypeRarity(pet)
	rarityName := i18n.T("rarities."+petsvc.RarityBonus(rarity), lang)
	color := rarityHatchColor(rarity)

	personality := petsvc.PersonalityTraits[pet.Personality]
	pTrait := ""
	if personality != nil {
		pTrait = i18n.T("pets.personality."+pet.Personality, lang)
	}
	title := ""
	if pet.Title != "" {
		title = " 🏆 *" + pet.Title + "*"
	}

	embed := components.Embed(
		fmt.Sprintf("%s %s%s", emoji, pet.Nickname, title),
		fmt.Sprintf("%s · %s · Lvl %d", rarityName, pTrait, pet.Level),
		color,
	)

	// Combat
	hpBar := buildHPBar(pet.HP, pet.MaxHP)
	embed.Fields = append(embed.Fields, components.Field(
		i18n.T("pets.detail.combat", lang),
		fmt.Sprintf("HP %s %d/%d\n⚔️ ATK %d · 🛡️ DEF %d · 💨 SPD %d", hpBar, pet.HP, pet.MaxHP, pet.Atk, pet.Defense, pet.Speed),
		false,
	))

	// Precision
	embed.Fields = append(embed.Fields, components.Field(
		i18n.T("pets.detail.precision", lang),
		fmt.Sprintf("DGE %d%% · ACC %d%%\nCRIT %d%% / %.1fx · ✨ SPC %d%%", pet.DGE, pet.ACC, pet.CritC, pet.CritD, pet.SpcC),
		false,
	))

	// Progression
	prog := fmt.Sprintf("XP %d · 🏆 ELO %d\n💕 Bond %s %d/%d\n🍖 Fed %d", pet.XP, pet.Elo, buildBondBar(pet.BondLevel), pet.BondLevel, petsvc.MaxBond, pet.FoodEaten)
	if pet.SkillPoints > 0 {
		prog += fmt.Sprintf("\n⚡ **%d** skill point(s) available!", pet.SkillPoints)
	}
	embed.Fields = append(embed.Fields, components.Field(i18n.T("pets.detail.progression", lang), prog, false))

	// Skills
	skills, _ := c.svc.GetPetSkills(pet.ID)
	if len(skills) > 0 {
		skillLines := make([]string, 0, len(skills))
		for _, s := range skills {
			if def, ok := petsvc.AllPetSkills[s.SkillID]; ok {
				slot := (s.Slot + 1) * 10
				skillLines = append(skillLines, fmt.Sprintf("%s **%s** (lvl %d)", def.Emoji, def.Name, slot))
			}
		}
		if len(skillLines) > 0 {
			embed.Fields = append(embed.Fields, components.Field(
				i18n.T("pets.detail.skills", lang),
				strings.Join(skillLines, "\n"),
				false,
			))
		}
	}

	// History (last 3)
	hist := c.svc.GetHistory(pet)
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
		embed.Fields = append(embed.Fields, components.Field(
			i18n.T("pets.detail.history", lang),
			strings.Join(lines, "\n"),
			false,
		))
	}

	// Name button: "Name" when the pet still has its species name, "Rename" otherwise.
	nameKey := "pets.detail.btn_rename"
	if pet.Nickname == pet.PetType {
		nameKey = "pets.detail.btn_name"
	}

	row1 := []discordgo.MessageComponent{
		components.Button(i18n.T("pets.detail.back", lang), components.EncodeOwner(userID, "pets", "menu"), discordgo.SecondaryButton),
		components.Button(i18n.T(nameKey, lang), components.EncodeOwner(userID, "pets", "rename_btn", petIDStr), discordgo.PrimaryButton),
		components.Button(i18n.T("pets.detail.btn_feed", lang), components.EncodeOwner(userID, "pets", "feed", petIDStr), discordgo.SuccessButton),
		components.Button(i18n.T("pets.detail.btn_battle", lang), components.EncodeOwner(userID, "pets", "battle", petIDStr), discordgo.DangerButton),
	}
	activateBtn := components.Button(i18n.T("pets.detail.btn_activate", lang), components.EncodeOwner(userID, "pets", "activate", petIDStr), discordgo.PrimaryButton)
	if pet.IsActive {
		disabled := components.Button(i18n.T("pets.detail.btn_active", lang), components.EncodeOwner(userID, "pets", "pet", petIDStr), discordgo.SecondaryButton).(discordgo.Button)
		disabled.Disabled = true
		activateBtn = disabled
	}
	row1 = append(row1, activateBtn)
	row2 := []discordgo.MessageComponent{}
	if pet.SkillPoints > 0 {
		row2 = append(row2, components.Button(i18n.T("pets.detail.btn_skills", lang), components.EncodeOwner(userID, "pets", "skills", petIDStr), discordgo.PrimaryButton))
	}
	row2 = append(row2, components.Button(i18n.T("pets.detail.btn_heal", lang), components.EncodeOwner(userID, "pets", "heal", petIDStr), discordgo.SuccessButton))
	// Sanctuary retire / recall (Option A)
	if !pet.InSanctuary && !pet.OnExpedition {
		if tier, _, _, _ := c.sanSvc.GetSanctuaryInfo(userID); tier > 0 {
			if c.hasSanctuarySpace(userID) {
				row2 = append(row2, components.Button(i18n.T("pets.detail.btn_retire", lang), components.EncodeOwner(userID, "pets", "retire_btn", petIDStr), discordgo.SecondaryButton))
			} else {
				btn := components.Button(i18n.T("pets.detail.btn_retire", lang), components.EncodeOwner(userID, "pets", "retire_btn", petIDStr), discordgo.SecondaryButton).(discordgo.Button)
				btn.Disabled = true
				row2 = append(row2, btn)
			}
		}
	}
	if pet.InSanctuary {
		row2 = append(row2, components.Button(i18n.T("pets.detail.btn_recall", lang), components.EncodeOwner(userID, "pets", "recall_btn", petIDStr), discordgo.PrimaryButton))
	}
	row2 = append(row2, components.Button("🗑️", components.EncodeOwner(userID, "pets", "delete", petIDStr), discordgo.SecondaryButton))

	comps := []discordgo.MessageComponent{components.ActionRow(row1...)}
	if len(row2) > 0 {
		comps = append(comps, components.ActionRow(row2...))
	}
	return embed, comps
}

// ─── Sanctuary Retire / Recall (via /pet detail) ────────────────────────────

func (c *Cog) onRetireClick(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	petID, _ := strconv.ParseInt(rest[0], 10, 64)
	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil || pet.UserID != userID {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}
	if pet.InSanctuary {
		interaction.RespondError(b, i, lang, "pets.retire.already")
		return
	}
	if pet.OnExpedition {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: i18n.T("pets.retire.on_expedition", lang, map[string]any{"name": pet.Nickname}), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	tier, used, max, _ := c.sanSvc.GetSanctuaryInfo(userID)
	if tier == 0 {
		interaction.RespondError(b, i, lang, "sanctuary.no_sanctuary")
		return
	}
	if used >= max {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: i18n.T("pets.retire.full", lang), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	pt := petsvc.PetTypes[pet.PetType]
	emoji := "🐾"
	if pt != nil {
		emoji = pt.Emoji
	}
	embed := components.Embed(i18n.T("pets.retire.confirm_title", lang),
		i18n.T("pets.retire.confirm_desc", lang, map[string]any{"emoji": emoji, "name": pet.Nickname, "used": used, "max": max}),
		0xe67e22)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("pets.retire.confirm_btn", lang), components.EncodeOwner(userID, "pets", "retire_confirm", rest[0]), discordgo.DangerButton),
			components.Button(i18n.T("pets.retire.cancel_btn", lang), components.EncodeOwner(userID, "pets", "pet", rest[0]), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onRetireConfirm(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	petID, _ := strconv.ParseInt(rest[0], 10, 64)
	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil || pet.UserID != userID {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}
	if err := c.sanSvc.RetirePet(userID, petID); err != nil {
		msg := err.Error()
		if errors.Is(err, sansvc.ErrSanctuaryFull) {
			msg = i18n.T("pets.retire.full", lang)
		} else if errors.Is(err, sansvc.ErrPetAlreadyInSanctuary) {
			msg = i18n.T("pets.retire.already", lang)
		}
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: msg, Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	// Refresh detail to show InSanctuary state (now Recall button)
	updated, _ := c.svc.GetPetByID(petID)
	if updated != nil {
		pet = updated
	}
	embed, comps := c.petDetail(pet, lang)
	// Prepend success notice via embed description prefix
	embed.Description = i18n.T("pets.retire.success", lang, map[string]any{"name": pet.Nickname}) + "\n\n" + embed.Description
	_ = b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onRecallClick(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	petID, _ := strconv.ParseInt(rest[0], 10, 64)
	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil || pet.UserID != userID {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}
	if !pet.InSanctuary {
		interaction.RespondError(b, i, lang, "pets.recall.not_in_sanctuary")
		return
	}
	embed := components.Embed(i18n.T("pets.recall.confirm_title", lang),
		i18n.T("pets.recall.confirm_desc", lang, map[string]any{"name": pet.Nickname, "cost": 100}),
		0x3498db)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("pets.recall.confirm_btn", lang), components.EncodeOwner(userID, "pets", "recall_confirm", rest[0]), discordgo.SuccessButton),
			components.Button(i18n.T("pets.recall.cancel_btn", lang), components.EncodeOwner(userID, "pets", "pet", rest[0]), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onRecallConfirm(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	petID, _ := strconv.ParseInt(rest[0], 10, 64)
	if err := c.sanSvc.RecallPet(userID, petID); err != nil {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: err.Error(), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	pet, _ := c.svc.GetPetByID(petID)
	if pet == nil {
		embed, comps := c.menuFromUser(userID, interaction.DisplayName(b.Session, i.GuildID, i.Member, userID), lang)
		_ = b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
		return
	}
	embed, comps := c.petDetail(pet, lang)
	embed.Description = i18n.T("pets.recall.success", lang, map[string]any{"name": pet.Nickname}) + "\n\n" + embed.Description
	_ = b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// onPetActivate sets the selected pet as the player's active companion.
func (c *Cog) onPetActivate(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}
	petID, err := strconv.ParseInt(rest[0], 10, 64)
	if err != nil {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}
	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil || pet.UserID != userID {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}
	if err := c.svc.SetActivePet(userID, petID, interaction.ToInt64(i.GuildID)); err != nil {
		slog.Error("pets: failed to activate pet", "user", userID, "pet", petID, "error", err)
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}
	embed, comps := c.petDetail(pet, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// ─── Healing ───────────────────────────────────────────────────

var errPetOnExpedition = errors.New("pet is on expedition")

// healPet applies the heal flow: expedition/full-HP guards, hospital discount,
// balance deduction and HP restore. It returns the recovered HP and the price
// paid (0 when the community hospital makes the heal free).
func (c *Cog) healPet(pet *model.UserPet, serverID int64) (healed, cost int, err error) {
	if pet.OnExpedition {
		return 0, 0, errPetOnExpedition
	}
	if pet.HP >= pet.MaxHP {
		return 0, 0, petsvc.ErrPetAlreadyFullHP
	}
	missing := pet.MaxHP - pet.HP
	discount := c.getHospitalDiscount(serverID)
	discount += int(furnituresvc.EffectValue(c.store, pet.UserID, "pet_heal") * 100)
	if discount > 100 {
		discount = 100
	}
	cost = petsvc.HealCost(missing, discount)
	if err := c.svc.HealPet(pet, cost); err != nil {
		return 0, cost, err
	}
	return missing, cost, nil
}

// getHospitalDiscount returns the heal cost discount granted by the server's
// community hospital building (10% per level, capped at 100%).
func (c *Cog) getHospitalDiscount(serverID int64) int {
	if serverID == 0 {
		return 0
	}
	var sp model.ServerProject
	if err := c.store.DB.Where("server_id = ? AND project_id = ?", serverID, "hospital").First(&sp).Error; err != nil {
		return 0
	}
	discount := sp.Level * 10
	if discount > 100 {
		discount = 100
	}
	return discount
}

// healErrorMessage translates a heal flow failure into a localized message.
func (c *Cog) healErrorMessage(lang string, pet *model.UserPet, cost int, err error) string {
	switch err {
	case errPetOnExpedition:
		return i18n.T("pets.heal.on_expedition", lang, map[string]any{"name": pet.Nickname})
	case petsvc.ErrPetAlreadyFullHP:
		return i18n.T("pets.heal.full_hp", lang, map[string]any{"name": pet.Nickname})
	case petsvc.ErrInsufficientFunds:
		return i18n.T("pets.heal.no_money", lang, map[string]any{"price": cost})
	}
	return i18n.T("pets.heal.error", lang)
}

func (c *Cog) healSuccess(lang string, pet *model.UserPet, healed, cost int) *discordgo.MessageEmbed {
	return components.Embed("🏥",
		i18n.T("pets.heal.success", lang, map[string]any{"name": pet.Nickname, "hp": healed, "price": cost}),
		0x2ecc71)
}

// onHealCommand heals the caller's active pet via /heal.
func (c *Cog) onHealCommand(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	pet, err := c.svc.GetActivePet(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "hunt.no_pet")
		return
	}
	healed, cost, err := c.healPet(pet, interaction.ToInt64(i.GuildID))
	if err != nil {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: c.healErrorMessage(lang, pet, cost, err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, c.healSuccess(lang, pet, healed, cost), nil))
}

// onHealPrefix heals the caller's active pet via !heal.
func (c *Cog) onHealPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	pet, err := c.svc.GetActivePet(userID)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("hunt.no_pet", lang))
		return
	}
	healed, cost, err := c.healPet(pet, interaction.ToInt64(m.GuildID))
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, c.healErrorMessage(lang, pet, cost, err))
		return
	}
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, c.healSuccess(lang, pet, healed, cost))
}

// onHealButton heals the pet shown on the detail view and refreshes the embed.
func (c *Cog) onHealButton(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	petID, err := strconv.ParseInt(rest[0], 10, 64)
	if err != nil {
		return
	}
	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil || pet.UserID != userID {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}
	healed, cost, err := c.healPet(pet, interaction.ToInt64(i.GuildID))
	if err != nil {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: c.healErrorMessage(lang, pet, cost, err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	embed, comps := c.petDetail(pet, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
	_, _ = b.Session.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Content: i18n.T("pets.heal.success", lang, map[string]any{"name": pet.Nickname, "hp": healed, "price": cost}),
	})
}

// ─── Playing ───────────────────────────────────────────────────

func (c *Cog) onPlayCommand(b *interaction.Bot, i *discordgo.InteractionCreate) {
	start := time.Now()
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	if c.playCooldownActive(b, i, userID, lang) {
		return
	}
	cooldownDone := time.Since(start)
	pet, err := c.svc.GetActivePet(userID)
	if err != nil || pet == nil {
		interaction.RespondError(b, i, lang, "pets.play.no_pet")
		return
	}
	petDone := time.Since(start)
	if pet.OnExpedition {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: i18n.T("pets.play.on_expedition", lang, map[string]any{"name": pet.Nickname}),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	content := c.playWithPet(pet, lang)
	_ = c.store.SetCooldown(userID, "pet_play")
	cooldownSetDone := time.Since(start)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
	respondDone := time.Since(start)
	go c.tryInteraction(b, i, pet, "play")
	if respondDone > 50*time.Millisecond {
		slog.Info("pets: /play slow phases",
			"user", userID,
			"cooldown_check", cooldownDone.String(),
			"active_pet", petDone.String(),
			"cooldown_set", cooldownSetDone.String(),
			"total_to_respond", respondDone.String(),
		)
	}
}

func (c *Cog) onPlayPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	if remaining := c.playCooldownRemaining(userID); remaining > 0 {
		minutes := int(remaining.Minutes())
		if minutes < 1 {
			minutes = 1
		}
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.play.cooldown", lang, map[string]any{"minutes": minutes}))
		return
	}
	pet, err := c.svc.GetActivePet(userID)
	if err != nil || pet == nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.play.no_pet", lang))
		return
	}
	if pet.OnExpedition {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.play.on_expedition", lang, map[string]any{"name": pet.Nickname}))
		return
	}
	content := c.playWithPet(pet, lang)
	_ = c.store.SetCooldown(userID, "pet_play")
	_, _ = s.ChannelMessageSend(m.ChannelID, content)
}

// playCooldownRemaining returns the remaining cooldown for /play, or 0 when ready.
func (c *Cog) playCooldownRemaining(userID int64) time.Duration {
	if c.cfg.PlayCooldownMinutes <= 0 {
		return 0
	}
	var cd model.Cooldown
	err := c.store.DB.Where("user_id = ? AND activity_name = ?", userID, "pet_play").First(&cd).Error
	if err != nil {
		return 0
	}
	cooldown := time.Duration(c.cfg.PlayCooldownMinutes) * time.Minute
	elapsed := time.Since(cd.LastUsed)
	if elapsed >= cooldown {
		return 0
	}
	return cooldown - elapsed
}

// playCooldownActive replies with the cooldown message when the player must wait.
func (c *Cog) playCooldownActive(b *interaction.Bot, i *discordgo.InteractionCreate, userID int64, lang string) bool {
	remaining := c.playCooldownRemaining(userID)
	if remaining <= 0 {
		return false
	}
	minutes := int(remaining.Minutes())
	if minutes < 1 {
		minutes = 1
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("pets.play.cooldown", lang, map[string]any{"minutes": minutes}),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	return true
}

// petXPMultiplier returns the pet XP multiplier granted by a Genetics Lab
// placed in the user's active house.
func (c *Cog) petXPMultiplier(userID int64) float64 {
	return 1 + furnituresvc.EffectValue(c.store, userID, "pet_xp")
}

func (c *Cog) playWithPet(pet *model.UserPet, lang string) string {
	xpGain := int(float64(rand.Intn(16)+10) * c.petXPMultiplier(pet.UserID))
	lvlRes := c.svc.AddXP(pet, xpGain)
	c.svc.AddBond(pet, 2)
	c.svc.RecordHistory(pet, "played",
		"🎾 **"+pet.Nickname+"** had a great play session! (+"+itoa2(xpGain)+" XP)")
	if err := c.svc.UpdatePet(pet); err != nil {
		slog.Error("pets: failed to save pet after play", "user", pet.UserID, "pet", pet.ID, "error", err)
	}
	content := i18n.T("pets.play.success", lang, map[string]any{"name": pet.Nickname, "xp": xpGain})
	if lvlRes.Leveled {
		content += "\n" + i18n.T("pets.play.level_up", lang, map[string]any{"name": pet.Nickname, "level": pet.Level})
	}
	return content
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

func buildHPBar(hp, maxHP int) string {
	if maxHP <= 0 {
		return ""
	}
	percent := hp * 10 / maxHP
	if percent < 0 {
		percent = 0
	}
	if percent > 10 {
		percent = 10
	}
	return strings.Repeat("█", percent) + strings.Repeat("░", 10-percent)
}

// pvpRetroFrame renders one retro RPG battle frame for a PvP duel.
func (c *Cog) pvpRetroFrame(p1d, p2d components.DisplayPet, journal []string, lang string) *discordgo.MessageEmbed {
	return components.FightFrameEmbed(
		i18n.T("pets.battle.arena_title", lang),
		p1d, p2d,
		components.FightLabelsFor(lang, i18n.T("pets.battle.vs", lang)),
		journal,
	)
}

func (c *Cog) onRenameOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	petID := "0"
	if len(rest) > 0 {
		petID = rest[0]
	}
	modal := components.ModalResponse(
		components.EncodeOwner(userID, "pets", "rename_submit", petID),
		i18n.T("pets.rename.modal_title", lang),
		components.TextInput("name", i18n.T("pets.rename.input_label", lang), true, i18n.T("pets.rename.input_placeholder", lang), discordgo.TextInputShort, 1, 20),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onRenameSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.ModalSubmitData().CustomID)
	petIDStr := "0"
	if len(rest) > 0 {
		petIDStr = rest[0]
	}
	petID, _ := strconv.ParseInt(petIDStr, 10, 64)
	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil || pet.UserID != userID {
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

func (c *Cog) onFeedMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	petIDStr := "0"
	if len(rest) > 0 {
		petIDStr = rest[0]
	}
	petID, _ := strconv.ParseInt(petIDStr, 10, 64)
	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil || pet.UserID != userID {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}

	var inv []model.Inventory
	c.store.DB.Where("user_id = ? AND quantity > 0", userID).Find(&inv)
	opts := make([]discordgo.SelectMenuOption, 0, 25)
	for _, iv := range inv {
		def := petsvc.GetFeedItemDef(iv.ItemID)
		if def == nil {
			continue
		}
		it := items.Get(iv.ItemID)
		if it == nil {
			continue
		}
		desc := c.feedEffectDesc(def, lang)
		opts = append(opts, discordgo.SelectMenuOption{
			Label:       fmt.Sprintf("%s x%d", it.Name, iv.Quantity),
			Value:       iv.ItemID,
			Emoji:       &discordgo.ComponentEmoji{Name: it.Emoji},
			Description: desc,
		})
		if len(opts) >= 25 {
			break
		}
	}
	if len(opts) == 0 {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: i18n.T("pets.feed.no_food", lang, map[string]any{"name": pet.Nickname}),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	embed := components.Embed(
		i18n.T("pets.feed.menu_title", lang, map[string]any{"name": pet.Nickname}),
		i18n.T("pets.feed.menu_desc", lang, map[string]any{"current": pet.FoodEaten}),
		0x2ecc71,
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				components.ActionRow(
					discordgo.SelectMenu{
						CustomID:    components.EncodeOwner(userID, "pets", "feed_select", petIDStr),
						Placeholder: i18n.T("pets.feed.menu_placeholder", lang),
						Options:     opts,
					},
				),
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) feedEffectDesc(def *petsvc.FeedItemDef, lang string) string {
	parts := []string{}
	if def.Stat != "" {
		parts = append(parts, "+"+strconv.FormatFloat(def.Amount, 'f', -1, 64)+" "+i18n.T("pets.feed.stats."+def.Stat, lang))
	}
	if def.Bond > 0 {
		parts = append(parts, "💕 +"+strconv.Itoa(def.Bond))
	}
	return strings.Join(parts, " · ")
}

func (c *Cog) onFeedSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		return
	}
	itemID := data.Values[0]
	_, _, rest := components.Decode(data.CustomID)
	if len(rest) == 0 {
		return
	}
	petID, _ := strconv.ParseInt(rest[0], 10, 64)
	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil || pet.UserID != userID {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}

	def := petsvc.GetFeedItemDef(itemID)
	if def == nil {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: i18n.T("pets.feed.refuse", lang, map[string]any{
					"name": pet.Nickname, "item": items.LocalizedName(itemID, lang),
				}),
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var inv model.Inventory
	if err := c.store.DB.Where("user_id = ? AND item_id = ?", userID, itemID).First(&inv).Error; err != nil || inv.Quantity < 1 {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: i18n.T("pets.feed.no_inventory", lang, map[string]any{
					"name": items.LocalizedName(itemID, lang),
				}),
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if err := c.store.DB.Exec(
		`UPDATE inventory SET quantity = quantity - 1 WHERE user_id = ? AND item_id = ? AND quantity > 0`,
		userID, itemID,
	).Error; err != nil {
		slog.Error("pets: failed to consume food item", "user", userID, "item", itemID, "error", err)
		interaction.RespondError(b, i, lang, "pets.feed.error")
		return
	}

	fed, err := c.svc.FeedPet(pet, def)
	if err != nil || !fed {
		slog.Error("pets: failed to apply feed", "user", userID, "item", itemID, "error", err)
		interaction.RespondError(b, i, lang, "pets.feed.error")
		return
	}

	itemName := items.LocalizedName(itemID, lang)
	content := ""
	switch {
	case def.Stat != "" && def.Bond > 0:
		content = i18n.T("pets.feed.success", lang, map[string]any{
			"name": pet.Nickname, "item": itemName,
			"amount": def.Amount, "stat": i18n.T("pets.feed.stats."+def.Stat, lang),
			"bond": def.Bond, "current": pet.FoodEaten,
		})
	case def.Stat != "":
		content = i18n.T("pets.feed.success_raw", lang, map[string]any{
			"name": pet.Nickname, "item": itemName,
			"amount": def.Amount, "stat": i18n.T("pets.feed.stats."+def.Stat, lang),
			"current": pet.FoodEaten,
		})
	default:
		content = i18n.T("pets.feed.success_bond", lang, map[string]any{
			"name": pet.Nickname, "item": itemName, "bond": def.Bond,
		})
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	_ = c.store.RecordActivity(userID, "pets_fed", 1)
	_ = achievement.IncrementStat(b.DB, userID, "pets_fed", 1)

	if n, ok := c.store.PopQuestNotification(userID); ok {

		interaction.SendQuestNotification(b, i, n, lang)
	}

	if text, dm := jsvc.SceneLine(c.store, userID, "pets", lang); text != "" {
		interaction.SendJournalScene(b, i, text, dm)
	}
	c.tryInteraction(b, i, pet, "feed")
}

func (c *Cog) onDelete(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	petIDStr := "0"
	if len(rest) > 0 {
		petIDStr = rest[0]
	}
	petID, _ := strconv.ParseInt(petIDStr, 10, 64)
	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil || pet.UserID != userID {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}

	pt := petsvc.PetTypes[pet.PetType]
	emoji := "🐾"
	if pt != nil {
		emoji = pt.Emoji
	}
	rarityName := i18n.T("rarities."+petsvc.RarityBonus(petTypeRarity(pet)), lang)

	embed := components.Embed(
		i18n.T("pets.delete.confirm_title", lang),
		i18n.T("pets.delete.confirm_desc", lang, map[string]any{
			"emoji": emoji, "name": pet.Nickname, "rarity": rarityName, "level": pet.Level,
		}),
		0xe74c3c,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("pets.delete.confirm_btn", lang), components.EncodeOwner(userID, "pets", "delete_confirm", petIDStr), discordgo.DangerButton),
			components.Button(i18n.T("pets.delete.cancel_btn", lang), components.EncodeOwner(userID, "pets", "pet", petIDStr), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onDeleteConfirm(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	petIDStr := "0"
	if len(rest) > 0 {
		petIDStr = rest[0]
	}
	petID, _ := strconv.ParseInt(petIDStr, 10, 64)
	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil || pet.UserID != userID {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}

	pt := petsvc.PetTypes[pet.PetType]
	emoji := "🐾"
	if pt != nil {
		emoji = pt.Emoji
	}

	if err := c.svc.DeletePet(petID); err != nil {
		interaction.RespondError(b, i, lang, "pets.delete.error")
		return
	}
	_ = c.store.DB.Where("pet_id = ?", petID).Delete(&model.UserPetSkill{}).Error

	embed := components.Embed(
		i18n.T("pets.delete.success_title", lang, map[string]any{"name": pet.Nickname}),
		i18n.T("pets.delete.success_desc", lang, map[string]any{"emoji": emoji, "name": pet.Nickname}),
		0x2ecc71,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("pets.detail.back", lang), components.EncodeOwner(userID, "pets", "menu"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onBattleSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	petID, _ := strconv.ParseInt(rest[0], 10, 64)

	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil || pet.UserID != userID {
		interaction.RespondError(b, i, lang, "pets.equip.fail")
		return
	}

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
	emoji := "🐾"
	if pt := petsvc.PetTypes[pet.PetType]; pt != nil {
		emoji = pt.Emoji
	}
	embed := components.Embed(
		i18n.T("pets.battle.arena_title", lang),
		i18n.T("pets.battle.pick_opponent", lang, map[string]any{
			"challenger": MentionUser(userID),
			"pet":        emoji + " " + pet.Nickname,
		}),
		0xe74c3c,
	)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, []discordgo.MessageComponent{
			components.ActionRow(
				discordgo.SelectMenu{
					CustomID:    components.EncodeOwner(userID, "pets", "battle_accept", rest[0]),
					Placeholder: "Select opponent",
					Options:     opts,
				},
			),
		}))
}

// onBattleAccept handles both steps of a duel challenge:
//   - select variant: the challenger picks an opponent from the guild list,
//     which turns the picker into a challenge message with Accept/Decline buttons.
//   - button variant: the challenged opponent accepts the duel and the battle runs.
func (c *Cog) onBattleAccept(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	data := i.MessageComponentData()

	_, _, rest := components.Decode(data.CustomID)
	if len(rest) == 0 {
		return
	}
	petID, _ := strconv.ParseInt(rest[0], 10, 64)

	// Select variant: the challenger picks an opponent from the guild list.
	if len(data.Values) > 0 {
		opponentID := interaction.ToInt64(data.Values[0])
		if opponentID == userID {
			return
		}
		pet1, err := c.svc.GetPetByID(petID)
		if err != nil || pet1 == nil || pet1.UserID != userID {
			interaction.RespondError(b, i, lang, "pets.equip.fail")
			return
		}
		emoji := "🐾"
		if pt := petsvc.PetTypes[pet1.PetType]; pt != nil {
			emoji = pt.Emoji
		}
		embed := components.Embed(
			i18n.T("pets.battle.arena_title", lang),
			i18n.T("pets.battle.challenge_msg", lang, map[string]any{
				"opponent":   MentionUser(opponentID),
				"challenger": MentionUser(userID),
				"pet":        emoji + " " + pet1.Nickname,
			}),
			0xe74c3c,
		)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, []discordgo.MessageComponent{
				components.ActionRow(
					components.Button(i18n.T("pets.battle.accept_label", lang), components.EncodeOwner(userID, "pets", "battle_accept", rest[0], strconv.FormatInt(userID, 10), data.Values[0]), discordgo.SuccessButton),
					components.Button(i18n.T("pets.battle.decline_label", lang), components.EncodeOwner(userID, "pets", "battle_decline", rest[0], strconv.FormatInt(userID, 10), data.Values[0]), discordgo.DangerButton),
				),
			}))
		return
	}

	// Button variant: the challenged opponent accepts the duel.
	if len(rest) < 3 {
		return
	}
	challengerID, err := strconv.ParseInt(rest[1], 10, 64)
	if err != nil {
		return
	}
	opponentID, err := strconv.ParseInt(rest[2], 10, 64)
	if err != nil {
		return
	}
	if userID != opponentID {
		interaction.RespondError(b, i, lang, "pets.battle.wrong_opponent")
		return
	}
	c.runBattle(b, i, lang, petID, challengerID, opponentID)
}

// runBattle executes the duel between the challenger's selected pet and the
// opponent's active pet, then updates ELO, weekly scores, artifact XP, bonds
// and battle history.
func (c *Cog) runBattle(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, petID, challengerID, opponentID int64) {
	pet1, err := c.svc.GetPetByID(petID)
	if err != nil || pet1 == nil || pet1.UserID != challengerID {
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
	c.applyArtifacts(bp1, bp2, challengerID, opponentID, modID)

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
	c.svc.AddWeeklyScore(challengerID, serverID, weeklyScoreA, map[bool]int{true: 1, false: 0}[result.WinnerID == pet1.ID])
	c.svc.AddWeeklyScore(opponentID, serverID, weeklyScoreB, map[bool]int{true: 1, false: 0}[result.WinnerID == pet2.ID])

	_, art1Leveled, _ := c.svc.AddArtifactXP(challengerID, artXP1)
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
			MentionUser(challengerID), strconv.Itoa(pet1.Elo), diff1, strconv.Itoa(pet2.Elo), diff2, weeklyScoreA, weeklyScoreB, artLine1, artLine2)
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

	unlocks, _ := c.svc.CheckAndUnlock(challengerID)

	// Spawn frame: retro layout with full HP bars.
	p1d := components.DisplayPet{
		Name: pet1.Nickname, Emoji: bp1.Emoji, Level: pet1.Level,
		HP: bp1.MaxHP, MaxHP: bp1.MaxHP, Owner: interaction.DisplayName(b.Session, i.GuildID, i.Member, challengerID),
	}
	p2d := components.DisplayPet{
		Name: pet2.Nickname, Emoji: bp2.Emoji, Level: pet2.Level,
		HP: bp2.MaxHP, MaxHP: bp2.MaxHP, Owner: interaction.DisplayName(b.Session, i.GuildID, i.Member, opponentID),
	}
	spawn := c.pvpRetroFrame(p1d, p2d, []string{i18n.T("pets.battle.arena_intro", lang)}, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, spawn, nil))

	go interaction.AnimateFight(
		result.Turns,
		func(journal []string, t battle.BattleTurn) *discordgo.MessageEmbed {
			p1d.HP = t.Pet1HP
			p2d.HP = t.Pet2HP
			p1d.IsKO = t.Pet1HP <= 0
			p2d.IsKO = t.Pet2HP <= 0
			return c.pvpRetroFrame(p1d, p2d, journal, lang)
		},
		func(frame *discordgo.MessageEmbed, comps []discordgo.MessageComponent) {
			_, _ = b.Session.InteractionResponseEdit(i.Interaction, components.WebhookEditResponse(frame, comps))
		},
		func(_ []string) {
			_, _ = b.Session.InteractionResponseEdit(i.Interaction, components.WebhookEditResponse(embed, nil))

			if len(unlocks) > 0 {
				interaction.SendAchievements(b, i, lang, unlocks)
			}

			// Interaction trigger after battle
			if result.WinnerID == pet1.ID {
				c.tryInteraction(b, i, pet1, "battle")
			} else if result.WinnerID == pet2.ID {
				c.tryInteraction(b, i, pet2, "battle")
			}
		},
	)
}

func (c *Cog) onBattleDecline(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 3 {
		return
	}
	challengerID, _ := strconv.ParseInt(rest[1], 10, 64)
	if challengerID == 0 {
		return
	}
	opponentID, err := strconv.ParseInt(rest[2], 10, 64)
	if err != nil {
		return
	}
	if userID != opponentID {
		interaction.RespondError(b, i, lang, "pets.battle.wrong_opponent")
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("pets.battle.refused_msg", lang, map[string]any{"name": MentionUser(opponentID)}),
		},
	})
}

// ─── Hatch Command ───────────────────────────────────────────

func (c *Cog) onHatchButton(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	c.hatchEgg(b, i, userID, lang)
}

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

func (c *Cog) hasSanctuarySpace(userID int64) bool {
	if c.sanSvc == nil {
		return false
	}
	return c.sanSvc.HasSanctuarySpace(userID)
}

func (c *Cog) canHatchAnywhere(userID int64) bool {
	if c.svc.CanCreatePet(userID) {
		return true
	}
	return c.hasSanctuarySpace(userID)
}

func (c *Cog) hatchEgg(b *interaction.Bot, i *discordgo.InteractionCreate, userID int64, lang string) {
	eggType, hatchKey := c.findEgg(userID)
	if eggType == "" {
		interaction.RespondError(b, i, lang, "pets.hatch.no_egg")
		return
	}
	if !c.canHatchAnywhere(userID) {
		interaction.RespondError(b, i, lang, "pets.hatch.no_slots")
		return
	}
	activeFree := c.svc.CanCreatePet(userID)

	biome := hatchKey
	petType := ""
	if hatchKey == "prehistoric" {
		petType = petsvc.RollPrehistoric()
	} else {
		petType = petsvc.RollGacha("", hatchKey)
	}
	pet, err := c.svc.CreatePet(userID, petType, interaction.ToInt64(i.GuildID))
	if err != nil || pet == nil {
		interaction.RespondError(b, i, lang, "pets.hatch.error")
		return
	}
	// If active roster full but sanctuary has space, put pet directly into sanctuary
	sentToSanctuary := false
	if !activeFree && c.hasSanctuarySpace(userID) {
		pet.InSanctuary = true
		pet.IsActive = false
		_ = c.svc.UpdatePet(pet)
		sentToSanctuary = true
	}

	if err := c.store.DB.Exec(
		`UPDATE inventory SET quantity = quantity - 1 WHERE user_id = ? AND item_id = ? AND quantity > 0`,
		userID, eggType,
	).Error; err != nil {
		slog.Error("pets: failed to decrement egg inventory", "user", userID, "egg", eggType, "error", err)
	}

	rarity := petTypeRarity(pet)
	rarityKey := petsvc.RarityBonus(rarity)
	rarityName := i18n.T("rarities."+rarityKey, lang)
	color := rarityHatchColor(rarity)
	eggName := localizedEggName(eggType, lang)

	pTrait := petsvc.PersonalityTraits[pet.Personality]
	personality := i18n.T("pets.personality.brave", lang)
	if pTrait != nil {
		personality = i18n.T("pets.personality."+pet.Personality, lang)
	}

	// Step 1 — suspense: the egg trembles.
	step1 := components.Embed(
		i18n.T("pets.hatch.hatching_title", lang),
		i18n.T("pets.hatch.step1", lang, map[string]any{"egg": eggName}),
		hatchEggColor,
	)
	step1.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("pets.hatch.step_footer", lang, map[string]any{"egg": eggName})}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, step1, nil))

	go func() {
		time.Sleep(1500 * time.Millisecond)

		// Step 2 — the shell cracks; a personality stirs inside.
		step2 := components.Embed(
			i18n.T("pets.hatch.hatching_title", lang),
			i18n.T("pets.hatch.step2", lang)+"\n\n"+i18n.T("pets.hatch.step2_personality", lang, map[string]any{"personality": personality}),
			hatchEggColor,
		)
		step2.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("pets.hatch.step_footer", lang, map[string]any{"egg": eggName})}
		_, _ = b.Session.InteractionResponseEdit(i.Interaction, components.WebhookEditResponse(step2, nil))

		time.Sleep(1500 * time.Millisecond)

		// Step 3 — full reveal.
		emoji := "🐾"
		if pt := petsvc.PetTypes[pet.PetType]; pt != nil {
			emoji = pt.Emoji
		}
		biomeName := i18n.T("biomes."+biome, lang)
		desc := i18n.T("pets.hatch.success_desc", lang, map[string]any{
			"emoji": emoji, "pet": pet.Nickname, "type": petType,
			"personality": personality, "biome": biomeName, "egg": eggName, "rarity": rarityName,
		})
		if sentToSanctuary {
			desc += "\n\n" + i18n.T("pets.hatch.sent_to_sanctuary", lang)
		}
		final := components.Embed(
			i18n.T("pets.hatch.success_title", lang, map[string]any{"emoji": emoji}),
			desc,
			color,
		)
		_, _ = b.Session.InteractionResponseEdit(i.Interaction, components.WebhookEditResponse(final, nil))
	}()
}

func (c *Cog) hatchEggMessage(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message, userID int64, lang string) {
	eggType, hatchKey := c.findEgg(userID)
	if eggType == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.hatch.no_egg", lang))
		return
	}
	if !c.canHatchAnywhere(userID) {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.hatch.no_slots", lang))
		return
	}
	activeFreeMsg := c.svc.CanCreatePet(userID)
	biome := hatchKey
	petType := ""
	if hatchKey == "prehistoric" {
		petType = petsvc.RollPrehistoric()
	} else {
		petType = petsvc.RollGacha("", hatchKey)
	}
	pet, err := c.svc.CreatePet(userID, petType, interaction.ToInt64(m.GuildID))
	if err != nil || pet == nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.hatch.error", lang))
		return
	}
	sentToSanctuaryMsg := false
	if !activeFreeMsg && c.hasSanctuarySpace(userID) {
		pet.InSanctuary = true
		pet.IsActive = false
		_ = c.svc.UpdatePet(pet)
		sentToSanctuaryMsg = true
	}
	if err := c.store.DB.Exec(
		`UPDATE inventory SET quantity = quantity - 1 WHERE user_id = ? AND item_id = ? AND quantity > 0`,
		userID, eggType,
	).Error; err != nil {
		slog.Error("pets: failed to decrement egg inventory (msg)", "user", userID, "egg", eggType, "error", err)
	}

	rarity := petTypeRarity(pet)
	rarityKey := petsvc.RarityBonus(rarity)
	rarityName := i18n.T("rarities."+rarityKey, lang)
	color := rarityHatchColor(rarity)
	eggName := localizedEggName(eggType, lang)

	pTrait := petsvc.PersonalityTraits[pet.Personality]
	personality := i18n.T("pets.personality.brave", lang)
	if pTrait != nil {
		personality = i18n.T("pets.personality."+pet.Personality, lang)
	}

	// Step 1 — suspense.
	step1 := components.Embed(
		i18n.T("pets.hatch.hatching_title", lang),
		i18n.T("pets.hatch.step1", lang, map[string]any{"egg": eggName}),
		hatchEggColor,
	)
	step1.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("pets.hatch.step_footer", lang, map[string]any{"egg": eggName})}
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, step1)

	time.Sleep(1500 * time.Millisecond)

	// Step 2 — the shell cracks; a personality stirs inside.
	step2 := components.Embed(
		i18n.T("pets.hatch.hatching_title", lang),
		i18n.T("pets.hatch.step2", lang)+"\n\n"+i18n.T("pets.hatch.step2_personality", lang, map[string]any{"personality": personality}),
		hatchEggColor,
	)
	step2.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("pets.hatch.step_footer", lang, map[string]any{"egg": eggName})}
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, step2)

	time.Sleep(1500 * time.Millisecond)

	// Step 3 — full reveal.
	emoji := "🐾"
	if pt := petsvc.PetTypes[pet.PetType]; pt != nil {
		emoji = pt.Emoji
	}
	biomeName := i18n.T("biomes."+biome, lang)
	desc := i18n.T("pets.hatch.success_desc", lang, map[string]any{
		"emoji": emoji, "pet": pet.Nickname, "type": petType,
		"personality": personality, "biome": biomeName, "egg": eggName, "rarity": rarityName,
	})
	if sentToSanctuaryMsg {
		desc += "\n\n" + i18n.T("pets.hatch.sent_to_sanctuary", lang)
	}
	final := components.Embed(
		i18n.T("pets.hatch.success_title", lang, map[string]any{"emoji": emoji}),
		desc,
		color,
	)
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, final)
}

func localizedEggName(eggType, lang string) string {
	key := "items." + eggType + ".name"
	loc := i18n.T(key, lang)
	if loc != key {
		return loc
	}
	return items.DisplayName(eggType)
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

// prehistoricEggs are the fossilized eggs that hatch a prehistoric pet from
// any rarity (see pets.RollPrehistoric).
var prehistoricEggs = map[string]bool{
	"fossilized_egg": true,
}

// eggPriority is the hatch order when the player owns several eggs, most
// valuable first.
var eggPriority = []string{
	"volcano_egg", "tundra_egg", "ocean_egg", "mountain_egg",
	"desert_egg", "cave_egg", "forest_egg", "fossilized_egg",
}

func (c *Cog) findEgg(userID int64) (string, string) {
	var inv []model.Inventory
	eggIDs := make([]string, 0, len(eggBiomes)+len(prehistoricEggs))
	for id := range eggBiomes {
		eggIDs = append(eggIDs, id)
	}
	for id := range prehistoricEggs {
		eggIDs = append(eggIDs, id)
	}
	c.store.DB.Where("user_id = ? AND item_id IN ? AND quantity > 0", userID, eggIDs).Find(&inv)
	for _, id := range eggPriority {
		for _, iv := range inv {
			if iv.ItemID != id {
				continue
			}
			if prehistoricEggs[id] {
				return id, "prehistoric"
			}
			return id, eggBiomes[id]
		}
	}
	return "", ""
}

// ─── Skill Selection ──────────────────────────────────────────

func (c *Cog) onSkillSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	petID, _ := strconv.ParseInt(rest[0], 10, 64)
	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil || pet.SkillPoints <= 0 || pet.UserID != userID {
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
			Label:       sk.Emoji + " " + sk.Name,
			Value:       sk.ID + ":" + itoa2(slot),
			Description: truncate(sk.Description, 50),
		})
	}

	embed := components.Embed(
		i18n.T("pets.skills.choose_title", lang, map[string]any{"level": (slot + 1) * 10}),
		i18n.T("pets.skills.choose_desc", lang),
		0x9b59b6,
	)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, []discordgo.MessageComponent{
			components.ActionRow(
				discordgo.SelectMenu{
					CustomID:    components.EncodeOwner(userID, "pets", "skill_choose", rest[0]),
					Placeholder: i18n.T("pets.skills.select", lang),
					Options:     opts,
				},
			),
		}))
}

func (c *Cog) onSkillChoose(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
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
	if err != nil || pet == nil || pet.UserID != userID {
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
	userID := interaction.ToInt64(interaction.UserID(i))
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		return
	}
	choiceID := data.Values[0]
	_, _, rest := components.Decode(data.CustomID)
	if len(rest) == 0 {
		return
	}
	petID, _ := strconv.ParseInt(rest[0], 10, 64)

	pet, err := c.svc.GetPetByID(petID)
	if err != nil || pet == nil || pet.UserID != userID {
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
		c.svc.AddXP(pet, int(float64(reward.XPReward)*c.petXPMultiplier(pet.UserID)))
	}
	// Resolve choice detail via i18n
	detailKey := fmt.Sprintf("pets.interact.%s.choices.%s.detail", findInteractionID(choiceID), choiceID)
	detail := i18n.T(detailKey, lang, map[string]any{"name": pet.Nickname})
	if detail == detailKey {
		detail = i18n.T("pets.interact.default_detail", lang, map[string]any{"name": pet.Nickname})
	}
	c.svc.RecordHistory(pet, "interaction", detail)
	if err := c.svc.UpdatePet(pet); err != nil {
		slog.Error("pets: failed to save pet after interaction", "user", pet.UserID, "pet", pet.ID, "error", err)
	}

	if reward.ItemReward != "" {
		if err := c.store.AddItemRaw(c.store.DB, pet.UserID, reward.ItemReward, 1); err != nil {
			slog.Error("pets: failed to award interaction item", "user", pet.UserID, "item", reward.ItemReward, "error", err)
		}
	}

	if err := c.store.SetCooldown(pet.UserID, "pet_interaction"); err != nil {
		slog.Error("pets: failed to set interaction cooldown", "user", pet.UserID, "error", err)
	}

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
	userID := pet.UserID
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
					CustomID:    components.EncodeOwner(userID, "pets", "interact", strconv.FormatInt(pet.ID, 10)),
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
		ID: pet.ID, Nickname: pet.Nickname, Emoji: emoji, PetType: pet.PetType,
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

func petTypeRarityValue(p model.UserPet) string {
	if pt, ok := petsvc.PetTypes[p.PetType]; ok {
		return pt.Rarity
	}
	return "common"
}

const hatchEggColor = 0xF1E3C8

func rarityHatchColor(rarity string) int {
	switch rarity {
	case petsvc.RarityLegendary:
		return 0xFFD700
	case petsvc.RarityEpic:
		return 0x9B59B6
	case petsvc.RarityRare:
		return 0x2ECC71
	default:
		return 0x95A5A6
	}
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
				components.Button(emoji, components.EncodeOwner(userID, "pets", "artifact_stat_choose", strconv.FormatInt(int64(i), 10)), discordgo.PrimaryButton))
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
		components.Button("🔄 Reset", components.EncodeOwner(userID, "pets", "artifact_reset"), discordgo.DangerButton),
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

	if _, err := c.store.UpdateBalance(userID, -items.Get("artifact_shard").Price); err != nil {
		slog.Error("pets: failed to deduct artifact reset cost", "user", userID, "error", err)
	}
	if err := c.store.DB.Exec(
		`UPDATE inventory SET quantity = quantity - 1 WHERE user_id = ? AND item_id = ? AND quantity > 0`,
		userID, "artifact_shard",
	).Error; err != nil {
		slog.Error("pets: failed to consume artifact shard", "user", userID, "error", err)
	}
	if _, err := c.svc.ResetArtifact(userID); err != nil {
		slog.Error("pets: failed to reset artifact", "user", userID, "error", err)
	}

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
	userID := interaction.ToInt64(interaction.UserID(i))
	serverID := interaction.ToInt64(i.GuildID)
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
			components.Button(i18n.T("leaderboard.btn_refresh", lang), components.EncodeOwner(userID, "pets", "weekly_refresh"), discordgo.PrimaryButton),
			components.Button("📜 History", components.EncodeOwner(userID, "pets", "weekly_history"), discordgo.SecondaryButton),
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

// ─── Sanctuary Slash Autocomplete + Prefix ────────────────────────────────

func (c *Cog) onSlashRetire(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		c.handleRetireAutocomplete(b, i, lang, userID)
		return
	}
	val := ""
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "pet" {
			if v, ok := opt.Value.(string); ok {
				val = strings.TrimSpace(v)
			}
		}
	}
	if val == "" {
		interaction.RespondError(b, i, lang, "pets.retire.not_found")
		return
	}
	pet := c.findPetByAutocompleteValue(userID, val, false)
	if pet == nil {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: i18n.T("pets.retire.not_found", lang), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	if err := c.sanSvc.RetirePet(userID, pet.ID); err != nil {
		msg := err.Error()
		if errors.Is(err, sansvc.ErrSanctuaryFull) {
			msg = i18n.T("pets.retire.full", lang)
		} else if errors.Is(err, sansvc.ErrPetAlreadyInSanctuary) {
			msg = i18n.T("pets.retire.already", lang)
		}
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: msg, Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: i18n.T("pets.retire.success", lang, map[string]any{"name": pet.Nickname}), Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (c *Cog) onSlashRecall(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		c.handleRecallAutocomplete(b, i, lang, userID)
		return
	}
	val := ""
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Name == "pet" {
			if v, ok := opt.Value.(string); ok {
				val = strings.TrimSpace(v)
			}
		}
	}
	if val == "" {
		interaction.RespondError(b, i, lang, "pets.recall.not_found")
		return
	}
	pet := c.findPetByAutocompleteValue(userID, val, true)
	if pet == nil {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: i18n.T("pets.recall.not_found", lang), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	if err := c.sanSvc.RecallPet(userID, pet.ID); err != nil {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: err.Error(), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: i18n.T("pets.recall.success", lang, map[string]any{"name": pet.Nickname}), Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (c *Cog) handleRetireAutocomplete(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, userID int64) {
	focused := ""
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Focused {
			if v, ok := opt.Value.(string); ok {
				focused = strings.ToLower(strings.TrimSpace(v))
			}
		}
	}
	pets, _ := c.svc.GetActiveRosterPets(userID)
	choices := []*discordgo.ApplicationCommandOptionChoice{}
	for _, p := range pets {
		if p.InSanctuary || p.OnExpedition {
			continue
		}
		name := p.Nickname
		low := strings.ToLower(name + " " + p.PetType)
		if focused != "" && !strings.Contains(low, focused) {
			continue
		}
		display := name + " (" + p.PetType + " Lvl " + strconv.Itoa(p.Level) + ")"
		if len([]rune(display)) > 100 {
			display = string([]rune(display)[:100])
		}
		// Use nickname as value; duplicates resolved by picking first match
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: display, Value: strconv.FormatInt(p.ID, 10)})
		if len(choices) >= 25 {
			break
		}
	}
	sort.Slice(choices, func(a, b int) bool { return choices[a].Name < choices[b].Name })
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}

func (c *Cog) handleRecallAutocomplete(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, userID int64) {
	focused := ""
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Focused {
			if v, ok := opt.Value.(string); ok {
				focused = strings.ToLower(strings.TrimSpace(v))
			}
		}
	}
	pets, _ := c.svc.GetSanctuaryPets(userID)
	choices := []*discordgo.ApplicationCommandOptionChoice{}
	for _, p := range pets {
		low := strings.ToLower(p.Nickname + " " + p.PetType)
		if focused != "" && !strings.Contains(low, focused) {
			continue
		}
		display := p.Nickname + " (" + p.PetType + " Lvl " + strconv.Itoa(p.Level) + ")"
		if len([]rune(display)) > 100 {
			display = string([]rune(display)[:100])
		}
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: display, Value: strconv.FormatInt(p.ID, 10)})
		if len(choices) >= 25 {
			break
		}
	}
	sort.Slice(choices, func(a, b int) bool { return choices[a].Name < choices[b].Name })
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
	_ = lang
}

func (c *Cog) findPetByAutocompleteValue(userID int64, val string, inSanctuary bool) *model.UserPet {
	if id, err := strconv.ParseInt(val, 10, 64); err == nil {
		if p, err := c.svc.GetPetByID(id); err == nil && p != nil && p.UserID == userID && p.InSanctuary == inSanctuary {
			return p
		}
	}
	// Fallback nickname match (case-insensitive)
	var pets []model.UserPet
	if inSanctuary {
		pets, _ = c.svc.GetSanctuaryPets(userID)
	} else {
		pets, _ = c.svc.GetActiveRosterPets(userID)
	}
	low := strings.ToLower(val)
	for i := range pets {
		if strings.ToLower(pets[i].Nickname) == low || strings.ToLower(pets[i].PetType) == low {
			return &pets[i]
		}
	}
	for i := range pets {
		if strings.Contains(strings.ToLower(pets[i].Nickname+" "+pets[i].PetType), low) {
			return &pets[i]
		}
	}
	return nil
}

func (c *Cog) onPrefixRetire(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	args := strings.Fields(m.Content)
	if len(args) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.retire.not_found", lang))
		return
	}
	q := strings.Join(args[1:], " ")
	pet := c.findPetByAutocompleteValue(userID, q, false)
	if pet == nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.retire.not_found", lang))
		return
	}
	if err := c.sanSvc.RetirePet(userID, pet.ID); err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, err.Error())
		return
	}
	_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.retire.success", lang, map[string]any{"name": pet.Nickname}))
}

func (c *Cog) onPrefixRecall(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	args := strings.Fields(m.Content)
	if len(args) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.recall.not_found", lang))
		return
	}
	q := strings.Join(args[1:], " ")
	pet := c.findPetByAutocompleteValue(userID, q, true)
	if pet == nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.recall.not_found", lang))
		return
	}
	if err := c.sanSvc.RecallPet(userID, pet.ID); err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, err.Error())
		return
	}
	_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("pets.recall.success", lang, map[string]any{"name": pet.Nickname}))
}
