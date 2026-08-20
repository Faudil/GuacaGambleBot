package character

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/logger"
	"guacagamblebot/internal/model"
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
	r.Slash("character", "View your RPG player profile, stats, and equipment.", c.onSlashMenu)
	r.Slash("char", "View your RPG player profile, stats, and equipment.", c.onSlashMenu)
	r.Slash("profile", "View your RPG player profile, stats, and equipment.", c.onSlashMenu)
	r.Prefix("character", c.onPrefixMenu)
	r.Prefix("char", c.onPrefixMenu)
	r.Prefix("profile", c.onPrefixMenu)
	r.Component("character", "profile", c.onProfile)
	r.Component("character", "stats", c.onStats)
	r.Component("character", "equipment", c.onEquipment)
	r.Component("character", "stat_up_str", c.onStatUp("str"))
	r.Component("character", "stat_up_dex", c.onStatUp("dex"))
	r.Component("character", "stat_up_int", c.onStatUp("int"))
	r.Component("character", "stat_up_vit", c.onStatUp("vit"))
	r.Component("character", "stat_up_luk", c.onStatUp("luk"))
	r.Component("character", "perk", c.onPerkPick)
	r.Component("character", "equip_weapon", c.onEquipSelect(items.SlotWeapon))
	r.Component("character", "equip_armor", c.onEquipSelect(items.SlotArmor))
	r.Component("character", "equip_jewelry", c.onEquipSelect(items.SlotJewelry))
	r.Component("character", "equip_trinket", c.onEquipSelect(items.SlotTrinket))
	r.Component("character", "unequip_weapon", c.onUnequip(items.SlotWeapon))
	r.Component("character", "unequip_armor", c.onUnequip(items.SlotArmor))
	r.Component("character", "unequip_jewelry", c.onUnequip(items.SlotJewelry))
	r.Component("character", "unequip_trinket", c.onUnequip(items.SlotTrinket))
	r.Component("character", "equip_pick", c.onEquipPick)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	pseudo := interaction.DisplayName(b.Session, i.GuildID, i.Member, userID)
	embed := profileEmbed(c.svc, lang, userID, pseudo)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, profileButtons(lang, userID)))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	pseudo := interaction.DisplayName(s, m.GuildID, &discordgo.Member{User: m.Author}, userID)
	embed := profileEmbed(c.svc, lang, userID, pseudo)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: profileButtons(lang, userID),
	})
}

func (c *Cog) showView(view string, b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	var embed *discordgo.MessageEmbed
	var comps []discordgo.MessageComponent
	switch view {
	case "stats":
		embed = statsEmbed(c.svc, lang, userID)
		comps = statsButtons(c.svc, lang, userID)
	case "equipment":
		embed = equipmentEmbed(c.svc, lang, userID)
		comps = equipmentButtons(c.svc, lang, userID)
	default:
		embed = profileEmbed(c.svc, lang, userID, interaction.DisplayName(b.Session, i.GuildID, i.Member, userID))
		comps = profileButtons(lang, userID)
	}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// --- View handlers ---

func (c *Cog) onProfile(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.showView("profile", b, i)
}

func (c *Cog) onStats(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.showView("stats", b, i)
}

func (c *Cog) onEquipment(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.showView("equipment", b, i)
}

// --- Stat allocation ---

func (c *Cog) onStatUp(stat string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		userID := interaction.ToInt64(interaction.UserID(i))
		lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
		if err := c.store.AllocateStat(userID, stat); err != nil {
			key := "character.no_points"
			if errors.Is(err, store.ErrInvalidStat) {
				key = "character.invalid_stat"
			}
			interaction.RespondError(b, i, lang, key)
			return
		}
		c.showView("stats", b, i)
	}
}

func (c *Cog) onPerkPick(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(interaction.UserID(i))
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	desc, err := charsvc.ApplyPerk(c.store, userID, rest[0])
	if err != nil {
		interaction.RespondError(b, i, lang, "character.no_perk_points")
		return
	}
	embed := statsEmbed(c.svc, lang, userID)
	embed.Description += "\n\n✅ " + desc
	comps := statsButtons(c.svc, lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// --- Equipment ---

// maxEquipMenuOptions caps the equip select menu at Discord's hard limit of 25
// options. Players routinely hold dozens of unequipped pieces (weapons above
// all), and a menu beyond the cap is rejected by Discord, making the equip
// silently fail.
const maxEquipMenuOptions = 25

func (c *Cog) onEquipSelect(slot string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
		userID := interaction.ToInt64(interaction.UserID(i))

		items, err := c.store.GetUnequippedBySlot(userID, slot)
		if err != nil || len(items) == 0 {
			interaction.RespondError(b, i, lang, "character.equip_none_available")
			return
		}

		slotName := slotDisplayName(slot, lang)
		char, _ := c.store.EnsureCharacter(userID)
		charLevel := 1
		if char != nil {
			charLevel = char.Level
		}
		opts := equipSelectOptions(items, charLevel, lang)

		if err := b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: i18n.T("character.equip_select_title", lang, map[string]any{"slot": slotName}),
				Flags:   discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					components.ActionRow(
						discordgo.SelectMenu{
							MenuType:    discordgo.StringSelectMenu,
							CustomID:    components.EncodeOwner(userID, "character", "equip_pick", slot),
							Placeholder: i18n.T("character.equip_select_placeholder", lang),
							Options:     opts,
						},
					),
				},
			},
		}); err != nil {
			logger.Log().Warn("failed to send equip select menu",
				"error", err,
				"user", interaction.UserID(i),
				"guild", i.GuildID,
			)
		}
	}
}

// equipSelectOptions turns unequipped equipment rows into select menu options,
// capped at Discord's 25-option limit. Usable pieces (at or below the player's
// level) are listed first, then locked ones; within each group items are sorted
// by rarity, required level and name so the best gear is visible.
func equipSelectOptions(rows []model.UserEquipment, charLevel int, lang string) []discordgo.SelectMenuOption {
	sorted := make([]model.UserEquipment, len(rows))
	copy(sorted, rows)
	sort.SliceStable(sorted, func(a, b int) bool {
		eqA, eqB := sorted[a], sorted[b]
		usableA, usableB := eqA.MinLevel <= charLevel, eqB.MinLevel <= charLevel
		if usableA != usableB {
			return usableA
		}
		rarA, rarB := items.Rarity(eqA.Rarity).Rank(), items.Rarity(eqB.Rarity).Rank()
		if rarA != rarB {
			return rarA > rarB
		}
		if eqA.MinLevel != eqB.MinLevel {
			return eqA.MinLevel > eqB.MinLevel
		}
		return eqA.Name < eqB.Name
	})

	opts := make([]discordgo.SelectMenuOption, 0, min(len(sorted), maxEquipMenuOptions))
	for _, eq := range sorted {
		if len(opts) >= maxEquipMenuOptions {
			break
		}
		label := fmt.Sprintf("[%s] %s", eq.Rarity, eq.Name)
		if len(label) > 100 {
			label = label[:100]
		}
		desc := statSummaryLine(eq.StatSTR, eq.StatDEX, eq.StatINT, eq.StatVIT, eq.StatLUK)
		if eq.MinLevel > charLevel {
			desc = i18n.T("character.requires_level_lbl", lang, map[string]any{"level": eq.MinLevel}) + " · " + desc
		}
		if len(desc) > 100 {
			desc = desc[:100]
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label:       label,
			Value:       fmt.Sprintf("%d", eq.ID),
			Description: desc,
			Emoji:       &discordgo.ComponentEmoji{Name: rarityEmoji(eq.Rarity)},
		})
	}
	return opts
}

func (c *Cog) onEquipPick(b *interaction.Bot, i *discordgo.InteractionCreate) {
	data := i.Interaction.MessageComponentData()
	if len(data.Values) == 0 {
		return
	}
	userID := interaction.ToInt64(interaction.UserID(i))

	equipID, err := strconv.ParseUint(data.Values[0], 10, 64)
	if err != nil {
		return
	}

	if err := c.store.EquipInstance(userID, uint(equipID)); err != nil {
		lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
		if errors.Is(err, store.ErrLevelTooLow) {
			interaction.RespondError(b, i, lang, "character.requires_level")
			return
		}
		interaction.RespondError(b, i, lang, "character.equip_error")
		return
	}

	c.grantFullSetFlag(userID)

	// Defer then update the character view
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed := equipmentEmbed(c.svc, lang, userID)
	comps := equipmentButtons(c.svc, lang, userID)
	_, _ = b.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    i.ChannelID,
		ID:         i.Message.ID,
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &comps,
	})
}

// grantFullSetFlag records the delve chronicle flag the first time the player
// equips a full (4-piece) delve set.
func (c *Cog) grantFullSetFlag(userID int64) {
	equipped, err := c.store.GetEquipped(userID)
	if err != nil {
		return
	}
	var setIDs []string
	for _, eq := range equipped {
		if eq.SetID != "" {
			setIDs = append(setIDs, eq.SetID)
		}
	}
	_, _, _, _, _, infos := items.CalculateSetBonuses(setIDs)
	for _, info := range infos {
		if info.Pieces >= 4 {
			_ = c.store.AddDelveFlag(userID, "full_set_equipped", `{"source":"equip"}`)
			return
		}
	}
}

func (c *Cog) onUnequip(slot string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		userID := interaction.ToInt64(interaction.UserID(i))
		_ = c.store.UnequipSlot(userID, slot)
		c.showView("equipment", b, i)
	}
}

// --- Embed builders ---

func profileEmbed(svc *charsvc.Service, lang string, userID int64, pseudo string) *discordgo.MessageEmbed {
	res, err := svc.Profile(userID)
	if err != nil {
		return components.Embed(
			i18n.T("character.profile_title", lang),
			i18n.T("character.profile_error", lang),
			0x9b59b6,
		)
	}

	pct := 0
	if res.XPNext > 0 {
		pct = res.XP * 100 / res.XPNext
	}
	bar := xpBar(res.XP, res.XPNext, 15)

	desc := fmt.Sprintf(
		"**%s** %d\n%s\n%s **%d/%d** (%d%%)\n\n",
		i18n.T("character.level_label", lang), res.Level,
		bar,
		i18n.T("character.xp_label", lang), res.XP, res.XPNext, pct,
	)

	if res.SkillPoints > 0 {
		desc += fmt.Sprintf("✨ **%s:** %d\n", i18n.T("character.skill_points_label", lang), res.SkillPoints)
	}
	desc += fmt.Sprintf(
		"👑 **%s:** %d\n📊 **%s:** %d\n\n",
		i18n.T("character.crowns_label", lang), res.Crowns,
		i18n.T("character.total_jobs_label", lang), res.TotalJobLevel,
	)
	desc += fmt.Sprintf(
		"🏦 %s: $%d | 🔒 %s: $%d\n🏆 %s: %d | 🌟 %s: %d",
		i18n.T("character.wallet_label", lang), res.Wallet,
		i18n.T("character.bank_label", lang), res.Bank,
		i18n.T("character.achievements_label", lang), res.AchCount,
		i18n.T("character.glory_label", lang), res.GloryTotal,
	)

	if res.Mastery {
		desc += "\n\n🏅 **" + i18n.T("character.mastery_label", lang) + "**"
	}

	embed := components.Embed(
		i18n.T("character.profile_title", lang, map[string]any{"user": pseudo}),
		desc,
		0x9b59b6,
	)
	return embed
}

func statsEmbed(svc *charsvc.Service, lang string, userID int64) *discordgo.MessageEmbed {
	res, err := svc.Profile(userID)
	if err != nil {
		return components.Embed(
			i18n.T("character.stats_title", lang),
			i18n.T("character.profile_error", lang),
			0x9b59b6,
		)
	}

	sb := &strings.Builder{}
	if res.SkillPoints > 0 {
		fmt.Fprintf(sb, "✨ **%s:** %d — %s\n\n", i18n.T("character.skill_points_label", lang), res.SkillPoints, i18n.T("character.stat_alloc_hint", lang))
	}
	if res.PerkPoints > 0 {
		fmt.Fprintf(sb, "⭐ **%s:** %d — %s\n\n", i18n.T("character.perk_points_label", lang), res.PerkPoints, i18n.T("character.perk_pick_hint", lang))
	}

	statLine := func(emoji, name, desc string, base, bonus int) string {
		line := fmt.Sprintf("%s **%s:** %d", emoji, name, base)
		if bonus > 0 {
			line += fmt.Sprintf(" (+%d)", bonus)
		}
		return fmt.Sprintf("%s — *%s*\n", line, desc)
	}

	sb.WriteString(statLine("💪", i18n.T("character.stat_str", lang), i18n.T("character.stat_str_desc", lang), res.STR, res.EquipSTR))
	sb.WriteString(statLine("🤸", i18n.T("character.stat_dex", lang), i18n.T("character.stat_dex_desc", lang), res.DEX, res.EquipDEX))
	sb.WriteString(statLine("🧠", i18n.T("character.stat_int", lang), i18n.T("character.stat_int_desc", lang), res.INT, res.EquipINT))
	sb.WriteString(statLine("❤️", i18n.T("character.stat_vit", lang), i18n.T("character.stat_vit_desc", lang), res.VIT, res.EquipVIT))
	sb.WriteString(statLine("🍀", i18n.T("character.stat_luk", lang), i18n.T("character.stat_luk_desc", lang), res.LUK, res.EquipLUK))

	// Show set bonuses
	for _, s := range res.SetBonuses {
		if s.ActiveTier != nil {
			fmt.Fprintf(sb, "\n%s %s (%d/%d) %s", s.SetEmoji, s.SetName, s.Pieces, s.ActiveTier.Pieces, s.ActiveTier.Desc)
		}
	}

	embed := components.Embed(
		i18n.T("character.stats_title", lang),
		sb.String(),
		0x9b59b6,
	)
	return embed
}

func equipmentEmbed(svc *charsvc.Service, lang string, userID int64) *discordgo.MessageEmbed {
	equipped, _ := svc.Store().GetEquipped(userID)

	equippedBySlot := map[string]model.UserEquipment{}
	for _, eq := range equipped {
		equippedBySlot[eq.EquipSlot] = eq
	}

	slotLine := func(slot string, eq model.UserEquipment) string {
		if eq.ID == 0 {
			return fmt.Sprintf("❌ **%s:** %s\n", slotDisplayName(slot, lang), i18n.T("character.empty_slot", lang))
		}
		rarEmoji := rarityEmoji(eq.Rarity)
		line := fmt.Sprintf("✅ **%s:** %s %s %s (`Lv %d`)\n", slotDisplayName(slot, lang), rarEmoji, eq.Emoji, eq.Name, eq.MinLevel)
		line += statSummaryLine(eq.StatSTR, eq.StatDEX, eq.StatINT, eq.StatVIT, eq.StatLUK)

		// Show affixes
		var affixes []items.AppliedAffix
		if err := json.Unmarshal([]byte(eq.Affixes), &affixes); err == nil && len(affixes) > 0 {
			var affixParts []string
			for _, a := range affixes {
				affixParts = append(affixParts, fmt.Sprintf("%s %s+%d", a.Name, statEmoji(a.Stat), a.Value))
			}
			line += "└ " + strings.Join(affixParts, ", ") + "\n"
		}

		// Show set info
		if eq.SetID != "" {
			if set, ok := items.SetsByName[eq.SetID]; ok {
				line += fmt.Sprintf("└ %s %s\n", set.Emoji, set.Name)
			}
		}

		return line
	}

	desc := slotLine(items.SlotWeapon, equippedBySlot[items.SlotWeapon])
	desc += slotLine(items.SlotArmor, equippedBySlot[items.SlotArmor])
	desc += slotLine(items.SlotJewelry, equippedBySlot[items.SlotJewelry])
	desc += slotLine(items.SlotTrinket, equippedBySlot[items.SlotTrinket])

	embed := components.Embed(
		i18n.T("character.equipment_title", lang),
		desc,
		0x9b59b6,
	)
	return embed
}

// --- Button builders ---

func profileButtons(lang string, userID int64) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("character.btn_stats", lang), components.EncodeOwner(userID, "character", "stats"), discordgo.SecondaryButton),
			components.Button(i18n.T("character.btn_equipment", lang), components.EncodeOwner(userID, "character", "equipment"), discordgo.SecondaryButton),
		),
	}
}

func statsButtons(svc *charsvc.Service, lang string, userID int64) []discordgo.MessageComponent {
	res, _ := svc.Profile(userID)
	rows := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("character.btn_profile", lang), components.EncodeOwner(userID, "character", "profile"), discordgo.SecondaryButton),
			components.Button(i18n.T("character.btn_equipment", lang), components.EncodeOwner(userID, "character", "equipment"), discordgo.SecondaryButton),
		),
	}
	if res.SkillPoints > 0 {
		rows = append(rows, components.ActionRow(
			components.Button("STR +", components.EncodeOwner(userID, "character", "stat_up_str"), discordgo.SuccessButton),
			components.Button("DEX +", components.EncodeOwner(userID, "character", "stat_up_dex"), discordgo.SuccessButton),
			components.Button("INT +", components.EncodeOwner(userID, "character", "stat_up_int"), discordgo.SuccessButton),
			components.Button("VIT +", components.EncodeOwner(userID, "character", "stat_up_vit"), discordgo.SuccessButton),
			components.Button("LUK +", components.EncodeOwner(userID, "character", "stat_up_luk"), discordgo.SuccessButton),
		))
	}
	if res.PerkPoints > 0 {
		if char, err := svc.Store().EnsureCharacter(userID); err == nil {
			choices := charsvc.RollPerkChoices(char)
			if len(choices) > 0 {
				var perkRow []discordgo.MessageComponent
				for _, p := range choices {
					perkRow = append(perkRow, components.Button(
						p.Emoji+" "+p.Name,
						components.EncodeOwner(userID, "character", "perk", p.ID),
						discordgo.PrimaryButton,
					))
				}
				rows = append(rows, components.ActionRow(perkRow...))
			}
		}
	}
	return rows
}

func equipmentButtons(svc *charsvc.Service, lang string, userID int64) []discordgo.MessageComponent {
	equipped, _ := svc.Store().GetEquipped(userID)
	eqBySlot := map[string]bool{}
	for _, e := range equipped {
		eqBySlot[e.EquipSlot] = true
	}

	row1 := []discordgo.MessageComponent{
		components.Button(i18n.T("character.btn_profile", lang), components.EncodeOwner(userID, "character", "profile"), discordgo.SecondaryButton),
		components.Button(i18n.T("character.btn_stats", lang), components.EncodeOwner(userID, "character", "stats"), discordgo.SecondaryButton),
	}

	row2 := []discordgo.MessageComponent{}
	for _, slot := range items.EquipSlots {
		if eqBySlot[slot] {
			label := i18n.T("character.btn_unequip", lang, map[string]any{"slot": slotDisplayName(slot, lang)})
			row2 = append(row2, components.Button(label, components.EncodeOwner(userID, "character", "unequip_"+slot), discordgo.DangerButton))
		} else {
			label := i18n.T("character.btn_equip", lang, map[string]any{"slot": slotDisplayName(slot, lang)})
			row2 = append(row2, components.Button(label, components.EncodeOwner(userID, "character", "equip_"+slot), discordgo.SuccessButton))
		}
	}

	return []discordgo.MessageComponent{
		components.ActionRow(row1...),
		components.ActionRow(row2...),
	}
}

// --- Helpers ---

func xpBar(current, needed, width int) string {
	if needed <= 0 {
		return "[" + strings.Repeat("█", width) + "]"
	}
	filled := current * width / needed
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}

func slotDisplayName(slot, lang string) string {
	switch slot {
	case items.SlotWeapon:
		return i18n.T("character.slot_weapon", lang)
	case items.SlotArmor:
		return i18n.T("character.slot_armor", lang)
	case items.SlotJewelry:
		return i18n.T("character.slot_jewelry", lang)
	case items.SlotTrinket:
		return i18n.T("character.slot_trinket", lang)
	}
	return slot
}

func statStr(it *items.Item) string {
	parts := []string{}
	if it.StatSTR > 0 {
		parts = append(parts, "+"+strconv.Itoa(it.StatSTR)+" STR")
	}
	if it.StatDEX > 0 {
		parts = append(parts, "+"+strconv.Itoa(it.StatDEX)+" DEX")
	}
	if it.StatINT > 0 {
		parts = append(parts, "+"+strconv.Itoa(it.StatINT)+" INT")
	}
	if it.StatVIT > 0 {
		parts = append(parts, "+"+strconv.Itoa(it.StatVIT)+" VIT")
	}
	if it.StatLUK > 0 {
		parts = append(parts, "+"+strconv.Itoa(it.StatLUK)+" LUK")
	}
	if len(parts) == 0 {
		return ""
	}
	return "(" + strings.Join(parts, ", ") + ")\n"
}

func statSummaryLine(str, dex, intt, vit, luk int) string {
	parts := []string{}
	if str > 0 {
		parts = append(parts, fmt.Sprintf("STR+%d", str))
	}
	if dex > 0 {
		parts = append(parts, fmt.Sprintf("DEX+%d", dex))
	}
	if intt > 0 {
		parts = append(parts, fmt.Sprintf("INT+%d", intt))
	}
	if vit > 0 {
		parts = append(parts, fmt.Sprintf("VIT+%d", vit))
	}
	if luk > 0 {
		parts = append(parts, fmt.Sprintf("LUK+%d", luk))
	}
	if len(parts) == 0 {
		return ""
	}
	return "`" + strings.Join(parts, " · ") + "`\n"
}

func rarityEmoji(rarity string) string {
	switch strings.ToLower(rarity) {
	case "common":
		return "⬜"
	case "uncommon":
		return "🟩"
	case "rare":
		return "🔵"
	case "epic":
		return "🟣"
	case "legendary":
		return "🟠"
	}
	return "⬜"
}

func statEmoji(stat string) string {
	switch stat {
	case "str":
		return "💪"
	case "dex":
		return "🤸"
	case "int":
		return "🧠"
	case "vit":
		return "❤️"
	case "luk":
		return "🍀"
	}
	return ""
}
