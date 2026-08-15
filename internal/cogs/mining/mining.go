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
	invsvc "guacagamblebot/internal/service/inventory"
	jsvc "guacagamblebot/internal/service/journal"
	miningsvc "guacagamblebot/internal/service/mining"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
)

type userSession struct {
	depth          int
	bag            []miningsvc.BagEntry
	toolID         string
	ghostVeilTurns int
	riskMod        int
	riskTurns      int
}

var sessions = map[int64]*userSession{}

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *miningsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	def := universe.Get(cfg.Universe)
	if def == nil {
		def = universe.Get("hoakhaven")
	}
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)
	c := &Cog{store: s, cfg: cfg, svc: miningsvc.New(s, cfg, npcSvc)}
	r.Slash("mine", "Mining expedition", c.onSlashMenu)
	r.Slash("m", "Mining expedition", c.onSlashMenu)
	r.Prefix("mine", c.onPrefixMenu)
	r.Prefix("m", c.onPrefixMenu)
	r.Component("mine", "tool_select", c.onToolSelect)
	r.Component("mine", "descend", c.onDescend)
	r.Component("mine", "event", c.onEventOption)
	r.Component("mine", "leave", c.onLeave)
}

func (c *Cog) respond(b *interaction.Bot, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent) {
	if err := b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps)); err != nil {
		slog.Warn("mining interaction respond failed", "user", interaction.ToInt64(interaction.UserID(i)), "error", err)
	}
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
			fmt.Sprintf("%s\n└ %s", i18n.T(t.LocaleDescKey(), lang), status), false,
		))
	}
	for _, t := range locked {
		embed.Fields = append(embed.Fields, components.Field(
			fmt.Sprintf("🔒 %s", i18n.T(t.LocaleNameKey(), lang)),
			fmt.Sprintf("%s\n└ %s", i18n.T(t.LocaleDescKey(), lang), i18n.T("mining.tool_locked", lang, map[string]any{"level": t.MinLevel})), false,
		))
	}

	var row []discordgo.MessageComponent
	for _, t := range owned {
		row = append(row, discordgo.Button{
			Label:    i18n.T(t.LocaleNameKey(), lang),
			CustomID: components.EncodeOwner(userID, "mine", "tool_select", t.ItemID),
			Style:    discordgo.PrimaryButton,
			Emoji:    &discordgo.ComponentEmoji{Name: t.Emoji()},
		})
	}
	comps := []discordgo.MessageComponent{}
	if len(row) > 0 {
		comps = append(comps, components.ActionRow(row...))
	}
	return embed, comps
}

func (c *Cog) onToolSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	toolID := ""
	if len(rest) > 0 {
		toolID = rest[0]
	}
	sessions[userID] = &userSession{depth: 1, toolID: toolID}
	embed, comps := c.mineEmbed(lang, userID, "")
	c.respond(b, i, embed, comps)
}

func (c *Cog) mineEmbed(lang string, userID int64, eventMsg string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	sess, ok := sessions[userID]
	if !ok {
		return components.Embed(i18n.T("mining.title", lang), i18n.T("mining.desc", lang), 0x4A90D9), nil
	}

	depth := sess.depth
	bagStr := c.bagString(sess.bag, lang)
	ti := miningsvc.GetToolInfo(sess.toolID)
	ml, _ := c.svc.GetMinerLevel(userID)
	levelReduc := int(float64(ml) * 1.5)

	riskNext := (depth-1)*5 - ti.RiskReduction - levelReduc + sess.riskMod
	if sess.ghostVeilTurns > 0 {
		riskNext -= 10
	}
	if riskNext < 0 {
		riskNext = 0
	}

	color := miningsvc.DepthColor(depth)
	flavorKey := miningsvc.DepthFlavorKey(depth)

	desc := i18n.T(flavorKey, lang)
	if eventMsg != "" {
		desc = eventMsg + "\n\n" + desc
	}

	lootText := i18n.T("mining.found_nothing", lang)
	if len(sess.bag) > 0 {
		last := sess.bag[len(sess.bag)-1]
		lootText = i18n.T("mining.found_item", lang, map[string]any{"item": localizedItemName(last.Name, lang)})
	}
	desc += "\n\n" + lootText

	if effects := c.effectsLine(sess, lang); effects != "" {
		desc += "\n" + effects
	}

	embed := components.Embed(
		fmt.Sprintf("⛏️ %s — %dm", i18n.T("mining.title", lang), depth),
		desc, color,
	)

	riskBar := progressBar(riskNext, 100, 10)
	riskColor := ""
	switch {
	case riskNext >= 70:
		riskColor = "🔴"
	case riskNext >= 40:
		riskColor = "🟡"
	default:
		riskColor = "🟢"
	}

	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(riskColor+" "+i18n.T("mining.status_field_risk", lang),
			i18n.T("mining.risk_bar", lang, map[string]any{"bar": riskBar, "pct": riskNext}), true),
		components.Field(i18n.T("mining.status_field_bag", lang), bagStr, false),
	}
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("%s %s · %s %d",
			ti.Emoji(), i18n.T(ti.LocaleNameKey(), lang),
			i18n.T("mining.miner_level_label", lang), ml),
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(
			discordgo.Button{
				Label:    "⛏️ " + i18n.T("mining.dig_label", lang),
				CustomID: components.EncodeOwner(userID, "mine", "descend"),
				Style:    discordgo.PrimaryButton,
			},
			discordgo.Button{
				Label:    i18n.T("mining.leave_label", lang),
				CustomID: components.EncodeOwner(userID, "mine", "leave"),
				Style:    discordgo.SuccessButton,
			},
		),
	}
	return embed, comps
}

func (c *Cog) effectsLine(sess *userSession, lang string) string {
	var parts []string
	if sess.ghostVeilTurns > 0 {
		parts = append(parts, "👻 "+i18n.T("mining.ghost_veil_active", lang, map[string]any{"turns": sess.ghostVeilTurns}))
	}
	if sess.riskTurns > 0 {
		sign := ""
		if sess.riskMod > 0 {
			sign = "+"
		}
		parts = append(parts, i18n.T("mining.effect_risk", lang,
			map[string]any{"sign": sign, "pct": sess.riskMod, "turns": sess.riskTurns}))
	}
	if len(parts) == 0 {
		return ""
	}
	return i18n.T("mining.effects_label", lang, map[string]any{"list": strings.Join(parts, " · ")})
}

func (c *Cog) eventEmbed(lang string, userID int64, ev *miningsvc.NarrativeEvent) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	if ev == nil || len(ev.Options) == 0 {
		return c.mineEmbed(lang, userID, "")
	}
	sess := sessions[userID]
	depthStr := ""
	if sess != nil {
		depthStr = fmt.Sprintf(" (%dm)", sess.depth)
	}

	emoji := map[miningsvc.EventRarity]string{
		"common":    "🟢",
		"rare":      "🔵",
		"legendary": "🌟",
	}
	label := map[miningsvc.EventRarity]string{
		"common":    "Common",
		"rare":      "Rare",
		"legendary": "Legendary",
	}

	titlePrefix := emoji[ev.Rarity]
	if titlePrefix == "" {
		titlePrefix = "🟢"
	}
	rarityLabel := label[ev.Rarity]
	if rarityLabel == "" {
		rarityLabel = "Common"
	}

	titleKey := "mining.ev_" + eventKeyID(ev.ID) + "_title"
	descKey := "mining.ev_" + eventKeyID(ev.ID) + "_desc"

	color := 0x9B59B6
	switch ev.Rarity {
	case "common":
		color = 0x4A90D9
	case "rare":
		color = 0x9B59B6
	case "legendary":
		color = 0xF1C40F
	}

	embed := components.Embed(
		fmt.Sprintf("%s %s %s", titlePrefix, i18n.T(titleKey, lang), depthStr),
		fmt.Sprintf("*%s*\n\n%s", rarityLabel, i18n.T(descKey, lang)),
		color,
	)

	var comps []discordgo.MessageComponent
	var row []discordgo.MessageComponent
	for i, opt := range ev.Options {
		btn := discordgo.Button{
			Label:    i18n.T(opt.Label, lang),
			CustomID: components.EncodeOwner(userID, "mine", "event", ev.ID, fmt.Sprint(i)),
			Style:    discordgo.PrimaryButton,
		}
		row = append(row, btn)
	}
	comps = append(comps, components.ActionRow(row...))

	return embed, comps
}

func eventKeyID(id string) string {
	return strings.ReplaceAll(id, "_legendary", "_leg")
}

func (c *Cog) onDescend(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	sess, ok := sessions[userID]
	if !ok {
		sess = &userSession{depth: 1}
		sessions[userID] = sess
	}

	gvt := sess.ghostVeilTurns
	res, err := c.svc.Descend(userID, sess.depth, sess.bag, sess.toolID, gvt)
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
		c.respond(b, i, components.Embed("💥 COLLAPSE!", msg, 0xFF0000), nil)
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

	eventMsg := c.buildEasterEggText(res.Event, lang)
	if res.LoreID != "" {
		loreTitle := miningsvc.LoreDisplayName(res.LoreID)
		loreMsg := i18n.T("mining.lore_discovery", lang, map[string]any{"title": loreTitle})
		if eventMsg != "" {
			eventMsg = loreMsg + "\n" + eventMsg
		} else {
			eventMsg = loreMsg
		}
	}

	embed, comps := c.mineEmbed(lang, userID, eventMsg)

	if res.NarrativeEvent != nil {
		eEmbed, eComps := c.eventEmbed(lang, userID, res.NarrativeEvent)
		embed = eEmbed
		comps = eComps
	}

	c.respond(b, i, embed, comps)
}

func (c *Cog) onEventOption(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 2 {
		interaction.RespondError(b, i, lang, "mining.error")
		return
	}
	eventID := rest[0]
	optionIdx := 0
	fmt.Sscanf(rest[1], "%d", &optionIdx)

	sess, ok := sessions[userID]
	if !ok {
		sess = &userSession{depth: 1}
		sessions[userID] = sess
	}

	eff := c.svc.ApplyEventOption(eventID, optionIdx, sess.depth, sess.bag)

	msg := ""
	if eff.Message != "" {
		msg = i18n.T(eff.Message, lang)
	}

	for _, it := range eff.Items {
		found := false
		for i, e := range sess.bag {
			if e.Name == it.Name {
				sess.bag[i].Count += it.Count
				found = true
				break
			}
		}
		if !found {
			sess.bag = append(sess.bag, miningsvc.BagEntry{Name: it.Name, Count: it.Count})
		}
	}

	if eff.RemoveItem == "random" && len(sess.bag) > 0 {
		idx := 0
		for i, e := range sess.bag {
			_ = i
			_ = e
			idx = i
		}
		if len(sess.bag) > 1 {
			idx = 0
		}
		_ = idx
		if len(sess.bag) > 0 {
			ri := 0
			sess.bag[ri].Count--
			if sess.bag[ri].Count <= 0 {
				sess.bag = append(sess.bag[:ri], sess.bag[ri+1:]...)
			}
		}
	}

	if eff.RiskTurns > 0 {
		sess.riskMod += eff.RiskMod
		sess.riskTurns = eff.RiskTurns
	}
	if eff.DepthGain != 0 {
		sess.depth += eff.DepthGain
	}
	if eff.ForceLeave {
		delete(sessions, userID)
		res, err := c.svc.LeaveMine(userID, sess.bag, sess.toolID)
		if err != nil {
			slog.Error("mining leave after event failed", "user", userID, "error", err)
			interaction.RespondError(b, i, lang, "mining.error")
			return
		}
		resultMsg := i18n.T("mining.success_msg", lang, map[string]any{
			"bag": c.bagString(res.Bag, lang), "xp": res.XP,
		})
		if res.LeveledUp {
			resultMsg += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": res.NewLevel})
		}
		if msg != "" {
			resultMsg = msg + "\n\n" + resultMsg
		}
		c.respond(b, i, components.Embed("✅ Expedition Complete!", resultMsg, 0x00FF00), nil)
		if n, ok := c.store.PopQuestNotification(userID); ok {
			interaction.SendQuestNotification(b, i, n, lang)
		}

		if text, dm := jsvc.SceneLine(c.store, userID, "mining", lang); text != "" {
			interaction.SendJournalScene(b, i, text, dm)
		}
		if len(res.Unlocks) > 0 {
			interaction.SendAchievements(b, i, lang, res.Unlocks)
		}
		return
	}

	embed, comps := c.mineEmbed(lang, userID, msg)
	c.respond(b, i, embed, comps)
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

	title := i18n.T("mining.empty_msg", lang)
	color := 0xC0C0C0
	if len(res.Bag) > 0 {
		title = i18n.T("mining.success_msg", lang, map[string]any{
			"bag": c.bagString(res.Bag, lang), "xp": res.XP,
		})
		color = 0x00FF00
	}
	if res.LeveledUp {
		title += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": res.NewLevel})
	}

	c.respond(b, i, components.Embed("✅ Expedition Complete!", title, color), nil)

	if n, ok := c.store.PopQuestNotification(userID); ok {

		interaction.SendQuestNotification(b, i, n, lang)
	}

	if text, dm := jsvc.SceneLine(c.store, userID, "mining", lang); text != "" {
		interaction.SendJournalScene(b, i, text, dm)
	}
	if len(res.Unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, res.Unlocks)
	}
}

func (c *Cog) buildEasterEggText(ev *miningsvc.MiningEvent, lang string) string {
	if ev == nil {
		return ""
	}
	switch ev.Type {
	case "hidden_chamber":
		return i18n.T("mining.event_hidden_chamber", lang, map[string]any{
			"items": c.bagString(ev.Items, lang),
		})
	case "ghost_miner":
		return i18n.T("mining.event_ghost_miner", lang)
	case "ancient_forge":
		return i18n.T("mining.event_ancient_forge", lang, map[string]any{
			"items": c.bagString(ev.Items, lang),
		})
	case "whispering_runes":
		return i18n.T("mining.event_whispering_runes", lang, map[string]any{
			"items": c.bagString(ev.Items, lang),
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
		n := localizedItemName(e.Name, lang)
		if e.Count > 1 {
			parts[i] = n + " x" + fmt.Sprint(e.Count)
		} else {
			parts[i] = n
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
