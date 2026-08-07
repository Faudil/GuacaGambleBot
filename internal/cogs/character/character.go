package character

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
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
	r.Component("character", "equip_weapon", c.onEquipSelect("weapon"))
	r.Component("character", "equip_armor", c.onEquipSelect("armor"))
	r.Component("character", "equip_accessory", c.onEquipSelect("accessory"))
	r.Component("character", "equip_trinket", c.onEquipSelect("trinket"))
	r.Component("character", "unequip_weapon", c.onUnequip("weapon"))
	r.Component("character", "unequip_armor", c.onUnequip("armor"))
	r.Component("character", "unequip_accessory", c.onUnequip("accessory"))
	r.Component("character", "unequip_trinket", c.onUnequip("trinket"))
	r.Component("character", "equip_pick", c.onEquipPick)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed := profileEmbed(c.svc, lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, profileButtons(lang)))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	embed := profileEmbed(c.svc, lang, interaction.ToInt64(m.Author.ID))
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: profileButtons(lang),
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
		embed = profileEmbed(c.svc, lang, userID)
		comps = profileButtons(lang)
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
		_ = c.store.AllocateStat(userID, stat)
		c.showView("stats", b, i)
	}
}

// --- Equipment ---

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
		opts := make([]discordgo.SelectMenuOption, 0, len(items))
		for _, eq := range items {
			label := fmt.Sprintf("[%s] %s", eq.Rarity, eq.Name)
			desc := statSummaryLine(eq.StatSTR, eq.StatDEX, eq.StatINT, eq.StatVIT, eq.StatLUK)
			if len(desc) > 100 {
				desc = desc[:100]
			}
			rarEmoji := rarityEmoji(eq.Rarity)
			opts = append(opts, discordgo.SelectMenuOption{
				Label:       label,
				Value:       fmt.Sprintf("%d", eq.ID),
				Description: desc,
				Emoji:       &discordgo.ComponentEmoji{Name: rarEmoji},
			})
		}

		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: i18n.T("character.equip_select_title", lang, map[string]any{"slot": slotName}),
				Flags:   discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					components.ActionRow(
						discordgo.SelectMenu{
							MenuType:    discordgo.StringSelectMenu,
							CustomID:    components.Encode("character", "equip_pick", slot),
							Placeholder: i18n.T("character.equip_select_placeholder", lang),
							Options:     opts,
						},
					),
				},
			},
		})
	}
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
		interaction.RespondError(b, i, lang, "character.equip_error")
		return
	}

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

func (c *Cog) onUnequip(slot string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		userID := interaction.ToInt64(interaction.UserID(i))
		_ = c.store.UnequipSlot(userID, slot)
		c.showView("equipment", b, i)
	}
}

// --- Embed builders ---

func profileEmbed(svc *charsvc.Service, lang string, userID int64) *discordgo.MessageEmbed {
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
		i18n.T("character.profile_title", lang, map[string]any{"user": interaction.Mention(userID)}),
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
		fmt.Fprintf(sb, "✨ **%s:** %d\n\n", i18n.T("character.skill_points_label", lang), res.SkillPoints)
	}

	statLine := func(emoji, name string, base, bonus int) string {
		if bonus > 0 {
			return fmt.Sprintf("%s **%s:** %d (+%d)\n", emoji, name, base, bonus)
		}
		return fmt.Sprintf("%s **%s:** %d\n", emoji, name, base)
	}

	sb.WriteString(statLine("💪", i18n.T("character.stat_str", lang), res.STR, res.EquipSTR))
	sb.WriteString(statLine("🤸", i18n.T("character.stat_dex", lang), res.DEX, res.EquipDEX))
	sb.WriteString(statLine("🧠", i18n.T("character.stat_int", lang), res.INT, res.EquipINT))
	sb.WriteString(statLine("❤️", i18n.T("character.stat_vit", lang), res.VIT, res.EquipVIT))
	sb.WriteString(statLine("🍀", i18n.T("character.stat_luk", lang), res.LUK, res.EquipLUK))

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
		line := fmt.Sprintf("✅ **%s:** %s %s %s\n", slotDisplayName(slot, lang), rarEmoji, eq.Emoji, eq.Name)
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

	desc := slotLine("weapon", equippedBySlot["weapon"])
	desc += slotLine("armor", equippedBySlot["armor"])
	desc += slotLine("accessory", equippedBySlot["accessory"])
	desc += slotLine("trinket", equippedBySlot["trinket"])

	embed := components.Embed(
		i18n.T("character.equipment_title", lang),
		desc,
		0x9b59b6,
	)
	return embed
}

// --- Button builders ---

func profileButtons(lang string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("character.btn_stats", lang), components.Encode("character", "stats"), discordgo.SecondaryButton),
			components.Button(i18n.T("character.btn_equipment", lang), components.Encode("character", "equipment"), discordgo.SecondaryButton),
		),
	}
}

func statsButtons(svc *charsvc.Service, lang string, userID int64) []discordgo.MessageComponent {
	res, _ := svc.Profile(userID)
	rows := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("character.btn_profile", lang), components.Encode("character", "profile"), discordgo.SecondaryButton),
			components.Button(i18n.T("character.btn_equipment", lang), components.Encode("character", "equipment"), discordgo.SecondaryButton),
		),
	}
	if res.SkillPoints > 0 {
		rows = append(rows, components.ActionRow(
			components.Button("STR +", components.Encode("character", "stat_up_str"), discordgo.SuccessButton),
			components.Button("DEX +", components.Encode("character", "stat_up_dex"), discordgo.SuccessButton),
			components.Button("INT +", components.Encode("character", "stat_up_int"), discordgo.SuccessButton),
			components.Button("VIT +", components.Encode("character", "stat_up_vit"), discordgo.SuccessButton),
			components.Button("LUK +", components.Encode("character", "stat_up_luk"), discordgo.SuccessButton),
		))
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
		components.Button(i18n.T("character.btn_profile", lang), components.Encode("character", "profile"), discordgo.SecondaryButton),
		components.Button(i18n.T("character.btn_stats", lang), components.Encode("character", "stats"), discordgo.SecondaryButton),
	}

	row2 := []discordgo.MessageComponent{}
	for _, slot := range []string{"weapon", "armor", "accessory", "trinket"} {
		if eqBySlot[slot] {
			label := i18n.T("character.btn_unequip", lang, map[string]any{"slot": slotDisplayName(slot, lang)})
			row2 = append(row2, components.Button(label, components.Encode("character", "unequip_"+slot), discordgo.DangerButton))
		} else {
			label := i18n.T("character.btn_equip", lang, map[string]any{"slot": slotDisplayName(slot, lang)})
			row2 = append(row2, components.Button(label, components.Encode("character", "equip_"+slot), discordgo.SuccessButton))
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
	case "weapon":
		return i18n.T("character.slot_weapon", lang)
	case "armor":
		return i18n.T("character.slot_armor", lang)
	case "accessory":
		return i18n.T("character.slot_accessory", lang)
	case "trinket":
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
