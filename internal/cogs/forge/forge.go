package forge

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
	"guacagamblebot/internal/model"
	forgesvc "guacagamblebot/internal/service/forge"
	furnituresvc "guacagamblebot/internal/service/furniture"
	"guacagamblebot/internal/store"
)

// maxMenuOptions caps select menu options at Discord's hard limit of 25.
const maxMenuOptions = 25

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *forgesvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: forgesvc.New(s, cfg)}
	r.Slash("forge", "Fuse 5 pieces of equipment into a higher rarity, or scrap them for resources.", c.onSlashMenu)
	r.Prefix("forge", c.onPrefixMenu)
	r.Component("forge", "menu", c.onMenu)
	r.Component("forge", "back", c.onBack)
	r.Component("forge", "fuse", c.onFuseRarity)
	r.Component("forge", "fuse_pick", c.onFuseItems)
	r.Component("forge", "fuse_confirm", c.onFuseConfirm)
	r.Component("forge", "scrap", c.onScrapRarity)
	r.Component("forge", "scrap_pick", c.onScrapItems)
	r.Component("forge", "scrap_confirm", c.onScrapConfirm)
}

// --- Entry points ---

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.menu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	embed, comps := c.menu(lang, userID)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

// --- Views ---

func (c *Cog) menu(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	hasForge := furnituresvc.HasFurniture(c.store, userID, "forge")
	hasArcane := furnituresvc.HasFurniture(c.store, userID, "arcane_forge")

	status := i18n.T("forge.status_forge", lang, map[string]any{"ok": boolCheck(hasForge)})
	status += "\n" + i18n.T("forge.status_arcane", lang, map[string]any{"ok": boolCheck(hasArcane)})

	desc := i18n.T("forge.menu_desc", lang) + "\n\n" + status + "\n\n**" + i18n.T("forge.gear_label", lang) + "**\n"
	for _, r := range forgesvc.RarityTiers {
		count := c.svc.UnequippedCount(userID, r)
		desc += fmt.Sprintf("%s %s: %d\n", rarityEmoji(r), rarityName(r, lang), count)
	}

	desc += "\n**" + i18n.T("forge.fusion_upgrades_label", lang) + "**\n"
	for idx := 0; idx < len(forgesvc.RarityTiers)-1; idx++ {
		from := forgesvc.RarityTiers[idx]
		to := forgesvc.RarityTiers[idx+1]
		marker := fusionStatusMarker(c.svc.CanFuse(userID, from))
		label := i18n.T("forge.rarity_to", lang, map[string]any{
			"from": rarityName(from, lang),
			"to":   rarityName(to, lang),
		})
		desc += fmt.Sprintf("%s %s — %s\n", marker, label, i18n.T("forge.research_"+string(from), lang))
	}

	embed := components.Embed(i18n.T("forge.menu_title", lang), desc, 0x8e6b23)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("forge.fuse_btn", lang), components.EncodeOwner(userID, "forge", "fuse"), discordgo.PrimaryButton),
			components.Button(i18n.T("forge.scrap_btn", lang), components.EncodeOwner(userID, "forge", "scrap"), discordgo.SecondaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.showMenu(b, i, discordgo.InteractionResponseUpdateMessage)
}

func (c *Cog) onBack(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.showMenu(b, i, discordgo.InteractionResponseUpdateMessage)
}

func (c *Cog) showMenu(b *interaction.Bot, i *discordgo.InteractionCreate, respType discordgo.InteractionResponseType) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.menu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(respType, embed, comps))
}

// --- Fusion flow: pick rarity, pick 5 items, fuse ---

func (c *Cog) onFuseRarity(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	var opts []discordgo.SelectMenuOption
	for idx := 0; idx < len(forgesvc.RarityTiers)-1; idx++ {
		from := forgesvc.RarityTiers[idx]
		to := forgesvc.RarityTiers[idx+1]
		lock := ""
		if err := c.svc.CanFuse(userID, from); err != nil {
			lock = " 🔒"
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label: i18n.T("forge.rarity_to", lang, map[string]any{
				"from": rarityName(from, lang),
				"to":   rarityName(to, lang),
			}) + lock,
			Value: string(from),
			Emoji: &discordgo.ComponentEmoji{Name: rarityEmoji(from)},
		})
	}

	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("forge.fuse_rarity_title", lang),
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				components.ActionRow(discordgo.SelectMenu{
					MenuType:    discordgo.StringSelectMenu,
					CustomID:    components.EncodeOwner(userID, "forge", "fuse_pick"),
					Placeholder: i18n.T("forge.fuse_rarity_placeholder", lang),
					Options:     opts,
				}),
			},
		},
	})
}

func (c *Cog) onFuseItems(b *interaction.Bot, i *discordgo.InteractionCreate) {
	data := i.Interaction.MessageComponentData()
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	if len(data.Values) == 0 {
		return
	}
	rarity := items.Rarity(data.Values[0])

	var rows []model.UserEquipment
	if err := c.store.DB.Where("user_id = ? AND is_equipped = ? AND rarity = ?",
		userID, false, string(rarity)).Find(&rows).Error; err != nil {
		interaction.RespondError(b, i, lang, "forge.err_generic")
		return
	}
	if len(rows) < forgesvc.FuseCount {
		interaction.RespondError(b, i, lang, "forge.err_need_five")
		return
	}

	sort.SliceStable(rows, func(a, b int) bool {
		if rows[a].MinLevel != rows[b].MinLevel {
			return rows[a].MinLevel > rows[b].MinLevel
		}
		return rows[a].Name < rows[b].Name
	})

	opts := make([]discordgo.SelectMenuOption, 0, min(len(rows), maxMenuOptions))
	for _, eq := range rows {
		if len(opts) >= maxMenuOptions {
			break
		}
		label := fmt.Sprintf("[%s] %s", eq.Rarity, eq.Name)
		if len(label) > 100 {
			label = label[:100]
		}
		desc := statSummaryLine(eq.StatSTR, eq.StatDEX, eq.StatINT, eq.StatVIT, eq.StatLUK)
		if len(desc) > 100 {
			desc = desc[:100]
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label:       label,
			Value:       fmt.Sprintf("%d", eq.ID),
			Description: desc,
			Emoji:       &discordgo.ComponentEmoji{Name: rarityEmoji(items.Rarity(eq.Rarity))},
		})
	}

	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("forge.fuse_pick_title", lang, map[string]any{"rarity": rarityName(rarity, lang)}),
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				components.ActionRow(discordgo.SelectMenu{
					MenuType:    discordgo.StringSelectMenu,
					CustomID:    components.EncodeOwner(userID, "forge", "fuse_confirm", string(rarity)),
					Placeholder: i18n.T("forge.fuse_pick_placeholder", lang),
					MinValues:   intPtr(forgesvc.FuseCount),
					MaxValues:   forgesvc.FuseCount,
					Options:     opts,
				}),
			},
		},
	})
}

func (c *Cog) onFuseConfirm(b *interaction.Bot, i *discordgo.InteractionCreate) {
	data := i.Interaction.MessageComponentData()
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(data.CustomID)
	if len(rest) < 1 {
		return
	}
	rarity := items.Rarity(rest[0])

	var ids []uint
	for _, v := range data.Values {
		id, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, uint(id))
	}
	if len(ids) != forgesvc.FuseCount {
		interaction.RespondError(b, i, lang, "forge.err_need_five")
		return
	}

	eq, err := c.svc.Fuse(userID, rarity, ids)
	if err != nil {
		if errors.Is(err, forgesvc.ErrResearchRequired) {
			interaction.RespondError(b, i, lang, "forge.err_research_required", map[string]any{
				"research": i18n.T("forge.research_"+string(rarity), lang),
			})
			return
		}
		interaction.RespondError(b, i, lang, c.errorKey(err))
		return
	}

	desc := i18n.T("forge.fuse_result_desc", lang, map[string]any{
		"from": rarityName(rarity, lang),
		"item": c.pieceSummary(eq, lang),
	})
	embed := components.Embed(i18n.T("forge.fuse_result_title", lang), desc, 0x8e6b23)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("forge.back_btn", lang), components.EncodeOwner(userID, "forge", "back"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// --- Scrap flow: pick rarity, pick item, scrap ---

func (c *Cog) onScrapRarity(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	var opts []discordgo.SelectMenuOption
	for _, r := range forgesvc.RarityTiers {
		opts = append(opts, discordgo.SelectMenuOption{
			Label: fmt.Sprintf("%s (%d)", rarityName(r, lang), c.svc.UnequippedCount(userID, r)),
			Value: string(r),
			Emoji: &discordgo.ComponentEmoji{Name: rarityEmoji(r)},
		})
	}

	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("forge.scrap_rarity_title", lang),
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				components.ActionRow(discordgo.SelectMenu{
					MenuType:    discordgo.StringSelectMenu,
					CustomID:    components.EncodeOwner(userID, "forge", "scrap_pick"),
					Placeholder: i18n.T("forge.scrap_rarity_placeholder", lang),
					Options:     opts,
				}),
			},
		},
	})
}

func (c *Cog) onScrapItems(b *interaction.Bot, i *discordgo.InteractionCreate) {
	data := i.Interaction.MessageComponentData()
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	if len(data.Values) == 0 {
		return
	}
	rarity := items.Rarity(data.Values[0])

	var rows []model.UserEquipment
	if err := c.store.DB.Where("user_id = ? AND is_equipped = ? AND rarity = ?",
		userID, false, string(rarity)).Find(&rows).Error; err != nil {
		interaction.RespondError(b, i, lang, "forge.err_generic")
		return
	}
	if len(rows) == 0 {
		interaction.RespondError(b, i, lang, "forge.err_no_items")
		return
	}

	sort.SliceStable(rows, func(a, b int) bool {
		if rows[a].MinLevel != rows[b].MinLevel {
			return rows[a].MinLevel > rows[b].MinLevel
		}
		return rows[a].Name < rows[b].Name
	})

	opts := make([]discordgo.SelectMenuOption, 0, min(len(rows), maxMenuOptions))
	for _, eq := range rows {
		if len(opts) >= maxMenuOptions {
			break
		}
		label := fmt.Sprintf("[%s] %s", eq.Rarity, eq.Name)
		if len(label) > 100 {
			label = label[:100]
		}
		desc := statSummaryLine(eq.StatSTR, eq.StatDEX, eq.StatINT, eq.StatVIT, eq.StatLUK)
		if len(desc) > 100 {
			desc = desc[:100]
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label:       label,
			Value:       fmt.Sprintf("%d", eq.ID),
			Description: desc,
			Emoji:       &discordgo.ComponentEmoji{Name: rarityEmoji(items.Rarity(eq.Rarity))},
		})
	}

	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("forge.scrap_pick_title", lang, map[string]any{"rarity": rarityName(rarity, lang)}),
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				components.ActionRow(discordgo.SelectMenu{
					MenuType:    discordgo.StringSelectMenu,
					CustomID:    components.EncodeOwner(userID, "forge", "scrap_confirm", string(rarity)),
					Placeholder: i18n.T("forge.scrap_pick_placeholder", lang),
					Options:     opts,
				}),
			},
		},
	})
}

func (c *Cog) onScrapConfirm(b *interaction.Bot, i *discordgo.InteractionCreate) {
	data := i.Interaction.MessageComponentData()
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	if len(data.Values) == 0 {
		return
	}
	equipID, err := strconv.ParseUint(data.Values[0], 10, 64)
	if err != nil {
		return
	}

	rewards, err := c.svc.Scrap(userID, uint(equipID))
	if err != nil {
		interaction.RespondError(b, i, lang, c.errorKey(err))
		return
	}

	var parts []string
	for itemID, qty := range rewards {
		parts = append(parts, fmt.Sprintf("%s x%d", items.LocalizedName(itemID, lang), qty))
	}
	sort.Strings(parts)

	desc := i18n.T("forge.scrap_result_desc", lang, map[string]any{
		"resources": "• " + strings.Join(parts, "\n• "),
	})
	embed := components.Embed(i18n.T("forge.scrap_result_title", lang), desc, 0x8e6b23)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("forge.back_btn", lang), components.EncodeOwner(userID, "forge", "back"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// --- Error mapping ---

func (c *Cog) errorKey(err error) string {
	switch err {
	case forgesvc.ErrNoForge:
		return "forge.err_no_forge"
	case forgesvc.ErrNeedArcaneForge:
		return "forge.err_need_arcane_forge"
	case forgesvc.ErrResearchRequired:
		return "forge.err_research_required"
	case forgesvc.ErrNeedFive:
		return "forge.err_need_five"
	case forgesvc.ErrNotOwned:
		return "forge.err_not_owned"
	case forgesvc.ErrEquippedItem:
		return "forge.err_equipped"
	case forgesvc.ErrWrongRarity:
		return "forge.err_wrong_rarity"
	case forgesvc.ErrNoItems:
		return "forge.err_no_items"
	case forgesvc.ErrUnknownItem:
		return "forge.err_unknown_item"
	}
	return "forge.err_generic"
}

// fusionStatusMarker renders the per-tier fusion upgrade status: researched,
// research available but not completed, or furniture missing.
func fusionStatusMarker(err error) string {
	switch err {
	case nil:
		return "✅"
	case forgesvc.ErrResearchRequired:
		return "🔬"
	}
	return "❌"
}

// --- Helpers ---

func (c *Cog) pieceSummary(eq *model.UserEquipment, lang string) string {
	line := fmt.Sprintf("%s **%s** %s\n", rarityEmoji(items.Rarity(eq.Rarity)), eq.Name, eq.Emoji)
	line += fmt.Sprintf("`%s` · %s\n", i18n.T("forge.lvl_lbl", lang, map[string]any{"level": eq.MinLevel}), slotName(eq.EquipSlot, lang))
	stats := statSummaryLine(eq.StatSTR, eq.StatDEX, eq.StatINT, eq.StatVIT, eq.StatLUK)
	if stats != "" {
		line += stats
	}
	var affixes []items.AppliedAffix
	if err := json.Unmarshal([]byte(eq.Affixes), &affixes); err == nil && len(affixes) > 0 {
		var parts []string
		for _, a := range affixes {
			parts = append(parts, fmt.Sprintf("%s +%d", a.Name, a.Value))
		}
		line += "└ " + strings.Join(parts, ", ") + "\n"
	}
	return line
}

func rarityName(r items.Rarity, lang string) string {
	return i18n.T("forge.rarity_"+string(r), lang)
}

func rarityEmoji(r items.Rarity) string {
	switch r {
	case items.RarityCommon:
		return "⬜"
	case items.RarityUncommon:
		return "🟩"
	case items.RarityRare:
		return "🔵"
	case items.RarityEpic:
		return "🟣"
	case items.RarityLegendary:
		return "🟠"
	}
	return "⬜"
}

func slotName(slot, lang string) string {
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

func boolCheck(ok bool) string {
	if ok {
		return "✅"
	}
	return "❌"
}

func intPtr(n int) *int {
	return &n
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
