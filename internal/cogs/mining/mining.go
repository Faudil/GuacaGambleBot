package mining

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	miningsvc "guacagamblebot/internal/service/mining"
	"guacagamblebot/internal/store"
)

type userSession struct {
	depth          int
	bag            []miningsvc.BagEntry
	toolID         string
	ghostVeilTurns int
}

var sessions = map[int64]*userSession{}

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *miningsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: miningsvc.New(s, cfg)}
	r.Slash("mine", "Mining expedition", c.onSlashMenu)
	r.Slash("m", "Mining expedition", c.onSlashMenu)
	r.Prefix("mine", c.onPrefixMenu)
	r.Prefix("m", c.onPrefixMenu)
	r.Component("mine", "tool_select", c.onToolSelect)
	r.Component("mine", "descend", c.onDescend)
	r.Component("mine", "leave", c.onLeave)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.toolSelection(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	embed, comps := c.toolSelection(lang, userID)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func localizedItemName(itemID, lang string) string {
	key := "mining.item_" + itemID
	loc := i18n.T(key, lang)
	if loc != key {
		return loc
	}
	return items.DisplayName(itemID)
}

func (c *Cog) toolSelection(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	level, err := c.svc.GetMinerLevel(userID)
	if err != nil {
		level = 1
	}

	embed := components.Embed(
		i18n.T("mining.tool_title", lang),
		i18n.T("mining.tool_desc", lang)+fmt.Sprintf("\n🧑‍🏭 **%s:** %d", i18n.T("mining.miner_level_label", lang), level),
		0x4A90D9,
	)

	owned := c.svc.OwnedTools(userID, level)
	locked := miningsvc.LockedTools(level)

	embed.Fields = []*discordgo.MessageEmbedField{}

	for _, t := range owned {
		status := i18n.T("mining.tool_owned", lang)
		if t.ItemID == "" {
			status = i18n.T("mining.tool_none", lang)
		}
		embed.Fields = append(embed.Fields, components.Field(
			fmt.Sprintf("%s %s", t.Emoji(), i18n.T(t.LocaleNameKey(), lang)),
			fmt.Sprintf("%s\n└ %s", i18n.T(t.LocaleDescKey(), lang), status),
			false,
		))
	}

	for _, t := range locked {
		embed.Fields = append(embed.Fields, components.Field(
			fmt.Sprintf("🔒 %s", i18n.T(t.LocaleNameKey(), lang)),
			fmt.Sprintf("%s\n└ %s", i18n.T(t.LocaleDescKey(), lang), i18n.T("mining.tool_locked", lang, map[string]any{"level": t.MinLevel})),
			false,
		))
	}

	var comps []discordgo.MessageComponent
	var row []discordgo.MessageComponent
	for _, t := range owned {
		btn := discordgo.Button{
			Label:    i18n.T(t.LocaleNameKey(), lang),
			CustomID: components.Encode("mine", "tool_select", t.ItemID),
			Style:    discordgo.PrimaryButton,
			Emoji:    &discordgo.ComponentEmoji{Name: t.Emoji()},
		}
		row = append(row, btn)
	}
	if len(row) > 0 {
		comps = append(comps, components.ActionRow(row...))
	}

	return embed, comps
}

func (c *Cog) onToolSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, action, rest := components.Decode(i.MessageComponentData().CustomID)
	_ = action

	toolID := ""
	if len(rest) > 0 {
		toolID = rest[0]
	}

	sessions[userID] = &userSession{
		depth:  1,
		toolID: toolID,
	}

	embed, comps := c.mineEmbed(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) mineEmbed(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	sess, ok := sessions[userID]
	if !ok {
		embed := components.Embed(
			i18n.T("mining.title", lang),
			i18n.T("mining.desc", lang),
			0x4A90D9,
		)
		return embed, nil
	}

	depth := sess.depth
	bagStr := c.bagString(sess.bag, lang)
	ti := miningsvc.GetToolInfo(sess.toolID)

	riskNext := (depth-1)*5 - ti.RiskReduction
	ml, _ := c.svc.GetMinerLevel(userID)
	levelReduc := int(float64(ml) * 1.5)
	riskNext -= levelReduc
	if sess.ghostVeilTurns > 0 {
		riskNext -= 10
	}
	if riskNext < 0 {
		riskNext = 0
	}

	barMax := 50
	if depth > barMax {
		barMax = depth + 5
	}

	color := miningsvc.DepthColor(depth)
	flavorKey := miningsvc.DepthFlavorKey(depth)

	embed := components.Embed(
		fmt.Sprintf("⛏️ %s — %s %dm", i18n.T("mining.title", lang), i18n.T("mining.status_field_depth", lang), depth),
		i18n.T(flavorKey, lang),
		color,
	)

	depthBar := progressBar(depth, barMax, 10)
	riskBar := progressBar(riskNext, 100, 10)

	lootText := i18n.T("mining.found_nothing", lang)
	var lastItem *miningsvc.MineItem
	if len(sess.bag) > 0 {
		lastItem = &miningsvc.MineItem{Name: sess.bag[len(sess.bag)-1].Name}
	}
	if lastItem != nil {
		lootText = i18n.T("mining.found_item", lang, map[string]any{
			"item": localizedItemName(lastItem.Name, lang),
		})
	}

	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("mining.status_field_depth", lang),
			i18n.T("mining.depth_bar", lang, map[string]any{
				"bar":   depthBar,
				"depth": depth,
			}), true),
		components.Field(i18n.T("mining.status_field_risk", lang),
			i18n.T("mining.risk_bar", lang, map[string]any{
				"bar": riskBar,
				"pct": riskNext,
			}), true),
		components.Field(i18n.T("mining.status_field_loot", lang), lootText, true),
		components.Field(i18n.T("mining.status_field_bag", lang), bagStr, false),
		components.Field(i18n.T("mining.status_field_tool", lang),
			fmt.Sprintf("%s %s", ti.Emoji(), i18n.T(ti.LocaleNameKey(), lang)), true),
	}

	if sess.ghostVeilTurns > 0 {
		embed.Fields = append(embed.Fields, components.Field(
			"👻 Ghostly Veil",
			i18n.T("mining.ghost_veil_active", lang, map[string]any{"turns": sess.ghostVeilTurns}),
			true,
		))
	}

	var comps []discordgo.MessageComponent

	choiceRow := []discordgo.MessageComponent{
			discordgo.Button{
				Label:    i18n.T("mining.choice_careful", lang),
				CustomID: components.Encode("mine", "descend", string(miningsvc.BranchCareful)),
				Style:    discordgo.SecondaryButton,
				Emoji:    &discordgo.ComponentEmoji{Name: "🛡️"},
			},
			discordgo.Button{
				Label:    i18n.T("mining.choice_aggressive", lang),
				CustomID: components.Encode("mine", "descend", string(miningsvc.BranchAggressive)),
				Style:    discordgo.DangerButton,
				Emoji:    &discordgo.ComponentEmoji{Name: "⚡"},
			},
		}
		comps = append(comps, components.ActionRow(choiceRow...))

		altRow := []discordgo.MessageComponent{
			discordgo.Button{
				Label:    i18n.T("mining.choice_veins", lang),
				CustomID: components.Encode("mine", "descend", string(miningsvc.BranchSearchVeins)),
				Style:    discordgo.SuccessButton,
				Emoji:    &discordgo.ComponentEmoji{Name: "🔍"},
			},
			discordgo.Button{
				Label:    i18n.T("mining.choice_rest", lang),
				CustomID: components.Encode("mine", "descend", string(miningsvc.BranchRest)),
				Style:    discordgo.SecondaryButton,
				Emoji:    &discordgo.ComponentEmoji{Name: "💤"},
			},
		}
		comps = append(comps, components.ActionRow(altRow...))

	exitRow := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    i18n.T("mining.leave_label", lang),
			CustomID: components.Encode("mine", "leave"),
			Style:    discordgo.SuccessButton,
			Emoji:    &discordgo.ComponentEmoji{Name: "🏃"},
		},
	}
	comps = append(comps, components.ActionRow(exitRow...))

	return embed, comps
}

func (c *Cog) onDescend(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)

	choice := miningsvc.BranchCareful
	if len(rest) > 0 {
		switch rest[0] {
		case string(miningsvc.BranchCareful):
			choice = miningsvc.BranchCareful
		case string(miningsvc.BranchAggressive):
			choice = miningsvc.BranchAggressive
		case string(miningsvc.BranchSearchVeins):
			choice = miningsvc.BranchSearchVeins
		case string(miningsvc.BranchRest):
			choice = miningsvc.BranchRest
		}
	}

	sess, ok := sessions[userID]
	if !ok {
		sess = &userSession{depth: 1}
		sessions[userID] = sess
	}

	gvt := sess.ghostVeilTurns

	res, err := c.svc.Descend(userID, sess.depth, sess.bag, choice, sess.toolID, gvt)
	if err != nil {
		if errors.Is(err, miningsvc.ErrMineLimit) {
			interaction.RespondError(b, i, lang, "mining.limit_reached")
		} else {
			slog.Error("mining descend failed", "user", userID, "error", err)
			interaction.RespondError(b, i, lang, "mining.error")
		}
		return
	}

	if res.Collapsed {
		delete(sessions, userID)
		bagStr := c.bagString(res.Bag, lang)
		msg := i18n.T("mining.collapse_msg", lang, map[string]any{"items": bagStr})
		if len(res.Bag) == 0 {
			msg = i18n.T("mining.collapse_empty", lang)
		}
		embed := components.Embed(
			"💥 COLLAPSE!",
			msg,
			0xFF0000,
		)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))
		return
	}

	sess.depth++
	sess.bag = res.Bag

	if sess.ghostVeilTurns > 0 {
		sess.ghostVeilTurns--
	}
	if res.Event != nil && res.Event.Buff == miningsvc.GhostVeilBuffID() {
		sess.ghostVeilTurns = 3
	}

	embed, comps := c.mineEmbed(lang, userID)

	eventText := c.buildEventText(res, lang)
	if eventText != "" {
		embed.Description = eventText + "\n\n" + embed.Description
	}

	if res.LoreID != "" {
		loreTitle := miningsvc.LoreDisplayName(res.LoreID)
		loreMsg := i18n.T("mining.lore_discovery", lang, map[string]any{"title": loreTitle})
		if embed.Footer == nil {
			embed.Footer = &discordgo.MessageEmbedFooter{}
		}
		embed.Footer.Text = loreMsg
	}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onLeave(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	sess, ok := sessions[userID]
	if !ok {
		sess = &userSession{}
	}

	res, err := c.svc.LeaveMine(userID, sess.bag, sess.toolID)
	if err != nil {
		slog.Error("mining leave failed", "user", userID, "error", err)
		interaction.RespondError(b, i, lang, "mining.error")
		return
	}
	delete(sessions, userID)

	if len(res.Bag) > 0 {
		embed := components.Embed(
			"✅ Expedition Complete!",
			i18n.T("mining.success_msg", lang, map[string]any{
				"bag": c.bagString(res.Bag, lang),
				"xp":  res.XP,
			}),
			0x00FF00,
		)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))
	} else {
		embed := components.Embed(
			"😐 Expedition Over",
			i18n.T("mining.empty_msg", lang, map[string]any{"xp": res.XP}),
			0xC0C0C0,
		)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))
	}

	if len(res.Unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, res.Unlocks)
	}
}

func (c *Cog) buildEventText(res *miningsvc.DescendResult, lang string) string {
	if res.Event == nil {
		return ""
	}
	switch res.Event.Type {
	case "hidden_chamber":
		return i18n.T("mining.event_hidden_chamber", lang, map[string]any{
			"items": c.bagString(res.Event.Items, lang),
		})
	case "ghost_miner":
		return i18n.T("mining.event_ghost_miner", lang)
	case "ancient_forge":
		return i18n.T("mining.event_ancient_forge", lang, map[string]any{
			"items": c.bagString(res.Event.Items, lang),
		})
	case "whispering_runes":
		return i18n.T("mining.event_whispering_runes", lang, map[string]any{
			"items": c.bagString(res.Event.Items, lang),
		})
	}
	return ""
}

func (c *Cog) bagString(bag []miningsvc.BagEntry, lang string) string {
	if len(bag) == 0 {
		return i18n.T("mining.nothing", lang)
	}
	parts := make([]string, len(bag))
	for i, e := range bag {
		displayName := localizedItemName(e.Name, lang)
		if e.Count > 1 {
			parts[i] = displayName + " x" + fmt.Sprint(e.Count)
		} else {
			parts[i] = displayName
		}
	}
	return strings.Join(parts, ", ")
}

func progressBar(value, max, segments int) string {
	if max <= 0 {
		return strings.Repeat("░", segments)
	}
	filled := value * segments / max
	if filled > segments {
		filled = segments
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", segments-filled)
}
