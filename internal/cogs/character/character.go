package character

import (
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
	r.Component("character", "equip_form_weapon", c.onEquipForm("weapon"))
	r.Component("character", "equip_form_armor", c.onEquipForm("armor"))
	r.Component("character", "equip_form_accessory", c.onEquipForm("accessory"))
	r.Component("character", "unequip_weapon", c.onUnequip("weapon"))
	r.Component("character", "unequip_armor", c.onUnequip("armor"))
	r.Component("character", "unequip_accessory", c.onUnequip("accessory"))
	r.Modal("character", "equip_submit", c.onEquipSubmit)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed := profileEmbed(c.svc, lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, navButtons(lang)))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	embed := profileEmbed(c.svc, lang, interaction.ToInt64(m.Author.ID))
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: navButtons(lang),
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
		comps = navButtons(lang)
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

func (c *Cog) onEquipForm(slot string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
		name := slotDisplayName(slot, lang)
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: components.Encode("character", "equip_submit", slot),
				Title:    i18n.T("character.equip_modal_title", lang, map[string]any{"slot": name}),
				Components: []discordgo.MessageComponent{
					components.ActionRow(
						components.TextInput(
							"item_name",
							i18n.T("character.equip_item_label", lang),
							true,
							"",
							discordgo.TextInputShort,
							1,
							50,
						),
					),
				},
			},
		})
	}
}

func (c *Cog) onEquipSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	vals := interaction.ModalValues(i)
	itemName := strings.TrimSpace(vals["item_name"])
	if itemName == "" {
		return
	}

	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.ModalSubmitData().CustomID
	_, _, rest := components.Decode(cid)
	slot := ""
	if len(rest) > 0 {
		slot = rest[0]
	}

	it := items.Get(itemName)
	if it == nil {
		interaction.RespondError(b, i, lang, "character.equip_not_found")
		return
	}
	if it.EquipSlot != slot {
		interaction.RespondError(b, i, lang, "character.equip_wrong_slot")
		return
	}
	// Check inventory
	var qty int64
	c.store.DB.Model(&model.Inventory{}).
		Where("user_id = ? AND item_id = ? AND quantity > 0", userID, it.ID).
		Count(&qty)
	if qty == 0 {
		interaction.RespondError(b, i, lang, "character.equip_not_owned")
		return
	}
	_ = c.svc.EquipItem(userID, slot, it.ID)
	c.showView("equipment", b, i)
}

func (c *Cog) onUnequip(slot string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		userID := interaction.ToInt64(interaction.UserID(i))
		_ = c.svc.UnequipSlot(userID, slot)
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
		"🏦 %s: $%d | 🔒 %s: $%d\n🏆 %s: %d",
		i18n.T("character.wallet_label", lang), res.Wallet,
		i18n.T("character.bank_label", lang), res.Bank,
		i18n.T("character.achievements_label", lang), res.AchCount,
	)

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

	embed := components.Embed(
		i18n.T("character.stats_title", lang),
		sb.String(),
		0x9b59b6,
	)
	return embed
}

func equipmentEmbed(svc *charsvc.Service, lang string, userID int64) *discordgo.MessageEmbed {
	eq, _ := svc.GetEquipment(userID)

	slotLine := func(slot, itemID string) string {
		it := items.Get(itemID)
		if it == nil || itemID == "" {
			return fmt.Sprintf("❌ **%s:** %s\n", slotDisplayName(slot, lang), i18n.T("character.empty_slot", lang))
		}
		bonus := statStr(it)
		return fmt.Sprintf("✅ **%s:** %s %s\n", slotDisplayName(slot, lang), it.Emoji, it.Name) + bonus
	}

	desc := slotLine("weapon", eq["weapon"])
	desc += slotLine("armor", eq["armor"])
	desc += slotLine("accessory", eq["accessory"])

	embed := components.Embed(
		i18n.T("character.equipment_title", lang),
		desc,
		0x9b59b6,
	)
	return embed
}

// --- Button builders ---

func navButtons(lang string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("character.btn_profile", lang), components.Encode("character", "profile"), discordgo.PrimaryButton),
			components.Button(i18n.T("character.btn_stats", lang), components.Encode("character", "stats"), discordgo.PrimaryButton),
			components.Button(i18n.T("character.btn_equipment", lang), components.Encode("character", "equipment"), discordgo.PrimaryButton),
		),
	}
}

func statsButtons(svc *charsvc.Service, lang string, userID int64) []discordgo.MessageComponent {
	res, _ := svc.Profile(userID)
	rows := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("character.btn_profile", lang), components.Encode("character", "profile"), discordgo.SecondaryButton),
			components.Button(i18n.T("character.btn_stats", lang), components.Encode("character", "stats"), discordgo.PrimaryButton),
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
	eq, _ := svc.GetEquipment(userID)

	row1 := []discordgo.MessageComponent{
		components.Button(i18n.T("character.btn_profile", lang), components.Encode("character", "profile"), discordgo.SecondaryButton),
		components.Button(i18n.T("character.btn_stats", lang), components.Encode("character", "stats"), discordgo.SecondaryButton),
		components.Button(i18n.T("character.btn_equipment", lang), components.Encode("character", "equipment"), discordgo.PrimaryButton),
	}

	row2 := []discordgo.MessageComponent{}
	for _, slot := range []string{"weapon", "armor", "accessory"} {
		if eq[slot] != "" {
			label := i18n.T("character.btn_unequip", lang, map[string]any{"slot": slotDisplayName(slot, lang)})
			row2 = append(row2, components.Button(label, components.Encode("character", "unequip_"+slot), discordgo.DangerButton))
		} else {
			label := i18n.T("character.btn_equip", lang, map[string]any{"slot": slotDisplayName(slot, lang)})
			row2 = append(row2, components.Button(label, components.Encode("character", "equip_form_"+slot), discordgo.SuccessButton))
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
