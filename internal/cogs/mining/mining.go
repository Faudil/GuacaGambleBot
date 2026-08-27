package mining

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/service/character"
	invsvc "guacagamblebot/internal/service/inventory"
	jsvc "guacagamblebot/internal/service/journal"
	"guacagamblebot/internal/service/magnet"
	miningsvc "guacagamblebot/internal/service/mining"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
)

type userSession struct {
	depth           int
	riskUnits       int
	bag             []miningsvc.BagEntry
	toolID          string
	ghostVeilTurns  int
	riskMod         int
	riskTurns       int
	contract        *miningsvc.Contract
	eventCount      int
	charterDeclined bool
}

var sessions = map[int64]*userSession{}
var sessionsMu sync.RWMutex
var userLocks sync.Map // map[int64]*sync.Mutex

func getUserMu(userID int64) *sync.Mutex {
	v, _ := userLocks.LoadOrStore(userID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

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
	r.Component("mine", "stay", c.onStay)
	r.Component("mine", "event", c.onEventOption)
	r.Component("mine", "leave", c.onLeave)
	r.Component("mine", "contract_pick", c.onContractPick)
}

func (c *Cog) respond(b *interaction.Bot, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent) {
	if err := b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps)); err != nil {
		slog.Warn("mining interaction respond failed", "user", interaction.ToInt64(interaction.UserID(i)), "error", err)
	}
}

// loadSession returns the in-memory session for userID, restoring it from the
// DB after a bot restart. It never charges an entry.
func (c *Cog) loadSession(userID int64) *userSession {
	sessionsMu.Lock()
	sess, ok := sessions[userID]
	sessionsMu.Unlock()
	if ok {
		return sess
	}
	ps, err := c.svc.LoadSession(userID)
	if err != nil || ps == nil {
		return nil
	}
	sess = &userSession{
		depth:           ps.Depth,
		riskUnits:       ps.RiskUnits,
		bag:             ps.Bag,
		toolID:          ps.ToolID,
		ghostVeilTurns:  ps.GhostVeilTurns,
		riskMod:         ps.RiskMod,
		riskTurns:       ps.RiskTurns,
		contract:        ps.Contract,
		eventCount:      ps.EventCount,
		charterDeclined: ps.CharterDeclined,
	}
	sessionsMu.Lock()
	if current, exists := sessions[userID]; exists {
		sessionsMu.Unlock()
		return current
	}
	sessions[userID] = sess
	sessionsMu.Unlock()
	return sess
}

// ensureSession returns the player's active session (resuming persisted state
// after a restart) or starts a fresh expedition, charging one daily entry.
func (c *Cog) ensureSession(userID int64, toolID string) (*userSession, error) {
	if sess := c.loadSession(userID); sess != nil {
		return sess, nil
	}
	mu := getUserMu(userID)
	mu.Lock()
	defer mu.Unlock()
	sessionsMu.RLock()
	if sess, active := sessions[userID]; active {
		sessionsMu.RUnlock()
		return sess, nil
	}
	sessionsMu.RUnlock()
	// Check DB without holding global
	if ps, _ := c.svc.LoadSession(userID); ps != nil {
		// loadSession would have inserted; re-check map
		sessionsMu.RLock()
		if sess, ok := sessions[userID]; ok {
			sessionsMu.RUnlock()
			return sess, nil
		}
		sessionsMu.RUnlock()
		// If still not in map, create from persisted
		sess := &userSession{
			depth:           ps.Depth,
			riskUnits:       ps.RiskUnits,
			bag:             ps.Bag,
			toolID:          ps.ToolID,
			ghostVeilTurns:  ps.GhostVeilTurns,
			riskMod:         ps.RiskMod,
			riskTurns:       ps.RiskTurns,
			contract:        ps.Contract,
			eventCount:      ps.EventCount,
			charterDeclined: ps.CharterDeclined,
		}
		sessionsMu.Lock()
		sessions[userID] = sess
		sessionsMu.Unlock()
		return sess, nil
	}
	if err := c.svc.EnterMine(userID); err != nil {
		return nil, err
	}
	sess := &userSession{depth: 1, toolID: toolID}
	sessionsMu.Lock()
	sessions[userID] = sess
	sessionsMu.Unlock()
	c.persistSession(userID)
	return sess, nil
}

// persistSession writes the current in-memory session to the DB so a restart
// cannot lose the expedition.
func (c *Cog) persistSession(userID int64) {
	sessionsMu.Lock()
	sess, ok := sessions[userID]
	var ps *miningsvc.PersistedSession
	if ok {
		ps = &miningsvc.PersistedSession{
			Depth:           sess.depth,
			RiskUnits:       sess.riskUnits,
			ToolID:          sess.toolID,
			GhostVeilTurns:  sess.ghostVeilTurns,
			RiskMod:         sess.riskMod,
			RiskTurns:       sess.riskTurns,
			Bag:             append([]miningsvc.BagEntry(nil), sess.bag...),
			Contract:        sess.contract,
			EventCount:      sess.eventCount,
			CharterDeclined: sess.charterDeclined,
		}
	}
	sessionsMu.Unlock()
	if ps == nil {
		return
	}
	_ = c.svc.SaveSession(userID, ps)
}

func (c *Cog) sessionError(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, userID int64, err error) {
	switch {
	case errors.Is(err, miningsvc.ErrMineLimit):
		interaction.RespondError(b, i, lang, "mining.limit_reached")
	case errors.Is(err, store.ErrInventoryFull):
		interaction.RespondError(b, i, lang, "inventory.full")
	default:
		slog.Error("mining session failed", "user", userID, "error", err)
		interaction.RespondError(b, i, lang, "mining.error")
	}
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	remaining, err := c.svc.RemainingEntries(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "mining.error")
		return
	}
	if remaining <= 0 {
		interaction.RespondError(b, i, lang, "mining.limit_reached")
		return
	}
	embed, comps := c.toolSelection(lang, userID, remaining)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	remaining, err := c.svc.RemainingEntries(userID)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("mining.error", lang))
		return
	}
	if remaining <= 0 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("mining.limit_reached", lang))
		return
	}
	embed, comps := c.toolSelection(lang, userID, remaining)
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
	return items.LocalizedName(itemID, lang)
}

func (c *Cog) toolSelection(lang string, userID int64, remaining int) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	level, err := c.svc.GetMinerLevel(userID)
	if err != nil {
		level = 1
	}
	embed := components.Embed(
		i18n.T("mining.tool_title", lang),
		i18n.T("mining.tool_desc", lang)+
			fmt.Sprintf("\n🧑‍🏭 **%s:** %d", i18n.T("mining.miner_level_label", lang), level)+
			fmt.Sprintf("\n⛏️ %s", i18n.T("mining.entries_remaining", lang, map[string]any{"count": remaining})),
		0x4A90D9,
	)
	owned := c.svc.OwnedTools(userID, level)
	locked := miningsvc.LockedTools(level)

	embed.Fields = []*discordgo.MessageEmbedField{}
	for _, t := range owned {
		status := i18n.T("mining.tool_owned", lang)
		if t.ItemID == "" {
			status = i18n.T("mining.tool_none", lang)
		} else if dur := c.svc.ToolDurability(userID, t.ItemID); dur > 0 {
			status = fmt.Sprintf("%s · %d/%d ⚒️", i18n.T("mining.tool_owned", lang), dur, t.Durability)
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
	if _, err := c.ensureSession(userID, toolID); err != nil {
		c.sessionError(b, i, lang, userID, err)
		return
	}
	embed, comps := c.mineEmbed(lang, userID, "", nil)
	c.respond(b, i, embed, comps)
}

var charterOffers = map[int64][]miningsvc.Contract{}
var charterMu sync.Mutex

func (c *Cog) charterEmbed(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	contracts := miningsvc.RollContracts()
	charterMu.Lock()
	charterOffers[userID] = contracts
	charterMu.Unlock()
	embed := components.Embed(
		i18n.T("mining.charter_title", lang),
		i18n.T("mining.charter_desc", lang),
		0xF1C40F,
	)
	for idx, ct := range contracts {
		titleKey := fmt.Sprintf("mining.contract_%s_title", string(ct.Type))
		descKey := fmt.Sprintf("mining.contract_%s_desc", string(ct.Type))
		title := i18n.T(titleKey, lang)
		if title == titleKey {
			title = string(ct.Type)
		}
		desc := i18n.T(descKey, lang, map[string]any{"target": ct.Target, "credits": ct.RewardCredits, "xp": ct.RewardXP})
		if desc == descKey {
			desc = fmt.Sprintf("Target %d", ct.Target)
		}
		embed.Fields = append(embed.Fields, components.Field(
			fmt.Sprintf("%d. %s", idx+1, title),
			desc, false,
		))
	}
	var row []discordgo.MessageComponent
	for idx := range contracts {
		row = append(row, discordgo.Button{
			Label:    fmt.Sprintf("#%d", idx+1),
			CustomID: components.EncodeOwner(userID, "mine", "contract_pick", fmt.Sprint(idx)),
			Style:    discordgo.PrimaryButton,
		})
	}
	row = append(row, discordgo.Button{
		Label:    i18n.T("mining.charter_skip", lang),
		CustomID: components.EncodeOwner(userID, "mine", "contract_pick", "skip"),
		Style:    discordgo.SecondaryButton,
	})
	comps := []discordgo.MessageComponent{components.ActionRow(row...)}
	return embed, comps
}

func (c *Cog) onContractPick(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	choice := ""
	if len(rest) > 0 {
		choice = rest[0]
	}
	mu := getUserMu(userID)
	mu.Lock()
	defer mu.Unlock()
	sessionsMu.Lock()
	sess, ok := sessions[userID]
	if !ok {
		sessionsMu.Unlock()
		interaction.RespondError(b, i, lang, "mining.error")
		return
	}
	if choice == "skip" {
		sess.contract = nil
		sess.charterDeclined = true
	} else {
		idx := 0
		fmt.Sscanf(choice, "%d", &idx)
		charterMu.Lock()
		pool, has := charterOffers[userID]
		charterMu.Unlock()
		if !has || idx < 0 || idx >= len(pool) {
			pool = miningsvc.RollContracts()
		}
		if idx >= 0 && idx < len(pool) {
			ct := pool[idx]
			sess.contract = &ct
		}
	}
	charterMu.Lock()
	delete(charterOffers, userID)
	charterMu.Unlock()
	sessionsMu.Unlock()
	c.persistSession(userID)
	embed, comps := c.mineEmbed(lang, userID, "", nil)
	c.respond(b, i, embed, comps)
}

func (c *Cog) mineEmbed(lang string, userID int64, eventMsg string, found *miningsvc.MineItem) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	sessionsMu.Lock()
	sess, ok := sessions[userID]
	if !ok {
		sessionsMu.Unlock()
		return components.Embed(i18n.T("mining.title", lang), i18n.T("mining.desc", lang), 0x4A90D9), nil
	}
	sessCopy := *sess
	sessionsMu.Unlock()
	sess = &sessCopy

	depth := sess.depth
	bagStr := c.bagString(sess.bag, lang)
	ti := miningsvc.GetToolInfo(sess.toolID)
	ml, _ := c.svc.GetMinerLevel(userID)

	riskNext := c.svc.RiskFor(userID, sess.riskUnits, sess.toolID, sess.ghostVeilTurns, sess.riskMod)

	color := miningsvc.DepthColor(depth)
	flavorKey := miningsvc.DepthFlavorKey(depth)

	desc := i18n.T(flavorKey, lang)
	if eventMsg != "" {
		desc = eventMsg + "\n\n" + desc
	}

	// found reports what was gained by the action that triggered this render
	// (e.g. this dig's roll), not the bag's running total — a merge into an
	// existing stack must show the delta just gained, not the cumulative count.
	lootText := i18n.T("mining.found_nothing", lang)
	if found != nil {
		if found.Count > 1 {
			lootText = i18n.T("mining.found_item_count", lang, map[string]any{"item": localizedItemName(found.Name, lang), "count": found.Count})
		} else {
			lootText = i18n.T("mining.found_item", lang, map[string]any{"item": localizedItemName(found.Name, lang)})
		}
	} else if len(sess.bag) > 0 {
		last := sess.bag[len(sess.bag)-1]
		if last.Count > 1 {
			lootText = i18n.T("mining.found_item_count", lang, map[string]any{"item": localizedItemName(last.Name, lang), "count": last.Count})
		} else {
			lootText = i18n.T("mining.found_item", lang, map[string]any{"item": localizedItemName(last.Name, lang)})
		}
	}
	desc += "\n\n" + lootText

	if effects := c.effectsLine(sess, lang); effects != "" {
		desc += "\n" + effects
	}
	if sess.contract != nil {
		prog, target := miningsvc.ContractProgress(sess.contract, sess.depth, sess.bag, sess.depth-1, sess.eventCount)
		done := ""
		if prog >= target {
			done = " ✅"
		}
		titleKey := fmt.Sprintf("mining.contract_%s_title", string(sess.contract.Type))
		ctTitle := i18n.T(titleKey, lang)
		if ctTitle == titleKey {
			ctTitle = string(sess.contract.Type)
		}
		desc += "\n" + i18n.T("mining.contract_progress", lang, map[string]any{"title": ctTitle, "prog": prog, "target": target, "done": done})
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
	if ti.ItemID != "" {
		if dur := c.svc.ToolDurability(userID, ti.ItemID); dur > 0 {
			embed.Footer.Text += fmt.Sprintf(" · %d/%d ⚒️", dur, ti.Durability)
		}
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(
			discordgo.Button{
				Label:    "⛏️ " + i18n.T("mining.dig_label", lang),
				CustomID: components.EncodeOwner(userID, "mine", "descend"),
				Style:    discordgo.PrimaryButton,
			},
			discordgo.Button{
				Label:    i18n.T("mining.stay_label", lang),
				CustomID: components.EncodeOwner(userID, "mine", "stay"),
				Style:    discordgo.SecondaryButton,
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

// decayTurns decrements the event risk effect counter, clearing the modifier
// once its turn budget is spent.
func decayTurns(sess *userSession) {
	if sess.riskTurns > 0 {
		sess.riskTurns--
		if sess.riskTurns <= 0 {
			sess.riskMod = 0
		}
	}
}

func (c *Cog) eventEmbed(lang string, userID int64, ev *miningsvc.NarrativeEvent) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	if ev != nil && ev.ID == miningsvc.NPCCharterEventID {
		return c.charterEmbed(lang, userID)
	}
	if ev == nil || len(ev.Options) == 0 {
		return c.mineEmbed(lang, userID, "", nil)
	}
	sessionsMu.Lock()
	sess, _ := sessions[userID]
	depthStr := ""
	if sess != nil {
		depthStr = fmt.Sprintf(" (%dm)", sess.depth)
	}
	sessionsMu.Unlock()

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
		if opt.Effect != nil && opt.Effect.RequireItem != "" {
			has, _ := c.store.HasItem(userID, opt.Effect.RequireItem, 1)
			if !has {
				continue
			}
		}
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

// onDescend digs and, on success, moves one depth deeper — the full risk
// increase per dig.
func (c *Cog) onDescend(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.performDig(b, i, true)
}

// onStay digs again at the current depth instead of going deeper — the
// player trades loot-tier progress for only half the risk increase of a
// normal descend.
func (c *Cog) onStay(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.performDig(b, i, false)
}

// performDig runs one dig action. When advance is true (a normal descend) a
// successful dig moves the player one depth deeper and the collapse risk
// grows by a full step (+2 risk units, i.e. +5%). When advance is false (the
// "stay" option) depth is unchanged and risk only grows by half a step (+1
// risk unit) — the same dig, loot and events, just without pressing deeper.
func (c *Cog) performDig(b *interaction.Bot, i *discordgo.InteractionCreate, advance bool) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	if _, err := c.ensureSession(userID, ""); err != nil {
		c.sessionError(b, i, lang, userID, err)
		return
	}

	mu := getUserMu(userID)
	mu.Lock()
	defer mu.Unlock()

	// Snapshot session fields under read lock
	sessionsMu.RLock()
	sess, ok := sessions[userID]
	if !ok {
		sess = &userSession{depth: 1}
		sessionsMu.RUnlock()
		sessionsMu.Lock()
		sessions[userID] = sess
		sessionsMu.Unlock()
		sessionsMu.RLock()
		sess = sessions[userID]
	}
	depth := sess.depth
	riskUnits := sess.riskUnits
	bagCopy := append([]miningsvc.BagEntry(nil), sess.bag...)
	toolID := sess.toolID
	gvt := sess.ghostVeilTurns
	riskMod := sess.riskMod
	hasContract := sess.contract != nil
	charterDeclined := sess.charterDeclined
	sessionsMu.RUnlock()

	res, err := c.svc.Descend(userID, depth, riskUnits, bagCopy, toolID, gvt, riskMod, hasContract, charterDeclined)
	if err != nil {
		slog.Error("mining descend failed", "user", userID, "error", err)
		interaction.RespondError(b, i, lang, "mining.error")
		return
	}

	if res.Collapsed {
		sessionsMu.Lock()
		delete(sessions, userID)
		sessionsMu.Unlock()
		charterMu.Lock()
		delete(charterOffers, userID)
		charterMu.Unlock()
		_ = c.svc.DeleteSession(userID)
		bagStr := c.bagString(res.Bag, lang)
		msg := i18n.T("mining.collapse_msg", lang, map[string]any{"items": bagStr})
		if len(res.Bag) == 0 {
			msg = i18n.T("mining.collapse_empty", lang)
		}
		c.respond(b, i, components.Embed(i18n.T("mining.collapse_title", lang), msg, 0xFF0000), nil)
		return
	}

	// Apply successful dig updates under write lock
	var brokeToolID string
	var sessForContract *userSession
	sessionsMu.Lock()
	if s, ok := sessions[userID]; ok {
		if advance {
			s.depth++
			s.riskUnits += 2
		} else {
			s.riskUnits++
		}
		s.bag = res.Bag
		if s.ghostVeilTurns > 0 {
			s.ghostVeilTurns--
		}
		decayTurns(s)
		if res.Event != nil && res.Event.Buff == miningsvc.GhostVeilBuffID() {
			s.ghostVeilTurns = 3
		}
		if res.ToolBroke {
			brokeToolID = s.toolID
			s.toolID = ""
		}
		if res.NarrativeEvent != nil {
			s.eventCount++
		}
		sessForContract = s
	}
	sessionsMu.Unlock()
	// persist after releasing per-user lock? Use helper (short lock)
	c.persistSession(userID)

	eventMsg := c.buildEasterEggText(res.Event, lang)
	if brokeToolID != "" {
		brokeMsg := i18n.T("mining.tool_broke", lang, map[string]any{"tool": localizedItemName(brokeToolID, lang)})
		if eventMsg != "" {
			eventMsg = brokeMsg + "\n\n" + eventMsg
		} else {
			eventMsg = brokeMsg
		}
	}
	if res.LoreID != "" {
		loreTitle := miningsvc.LoreDisplayName(res.LoreID)
		loreMsg := i18n.T("mining.lore_discovery", lang, map[string]any{"title": loreTitle})
		if eventMsg != "" {
			eventMsg = loreMsg + "\n" + eventMsg
		} else {
			eventMsg = loreMsg
		}
	}
	if sessForContract != nil {
		if contractDoneMsg := c.contractDoneMessage(sessForContract, lang); contractDoneMsg != "" {
			if eventMsg != "" {
				eventMsg = eventMsg + "\n\n" + contractDoneMsg
			} else {
				eventMsg = contractDoneMsg
			}
		}
	}

	embed, comps := c.mineEmbed(lang, userID, eventMsg, res.Item)

	if res.NarrativeEvent != nil {
		eEmbed, eComps := c.eventEmbed(lang, userID, res.NarrativeEvent)
		embed = eEmbed
		comps = eComps
	}

	c.respond(b, i, embed, comps)
}

func (c *Cog) contractDoneMessage(sess *userSession, lang string) string {
	if sess == nil || sess.contract == nil {
		return ""
	}
	if miningsvc.ContractCompleted(sess.contract, sess.depth, sess.bag, sess.depth-1, sess.eventCount) {
		titleKey := fmt.Sprintf("mining.contract_%s_title", string(sess.contract.Type))
		ctTitle := i18n.T(titleKey, lang)
		if ctTitle == titleKey {
			ctTitle = string(sess.contract.Type)
		}
		return i18n.T("mining.contract_done", lang, map[string]any{"title": ctTitle, "credits": sess.contract.RewardCredits})
	}
	return ""
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

	if _, err := c.ensureSession(userID, ""); err != nil {
		c.sessionError(b, i, lang, userID, err)
		return
	}

	mu := getUserMu(userID)
	mu.Lock()
	defer mu.Unlock()

	sessionsMu.RLock()
	sess, ok := sessions[userID]
	if !ok {
		sess = &userSession{depth: 1}
		sessionsMu.RUnlock()
		sessionsMu.Lock()
		sessions[userID] = sess
		sessionsMu.Unlock()
	} else {
		// copy needed fields for ApplyEventOption without holding write lock
		sessionsMu.RUnlock()
	}
	// snapshot for ApplyEventOption
	sessionsMu.RLock()
	depthSnap := sess.depth
	bagSnap := append([]miningsvc.BagEntry(nil), sess.bag...)
	sessionsMu.RUnlock()

	eff := c.svc.ApplyEventOption(eventID, optionIdx, depthSnap, bagSnap)

	if eff.RepairTool > 0 {
		sessionsMu.RLock()
		tid := sess.toolID
		sessionsMu.RUnlock()
		_ = c.svc.RepairTool(userID, tid, eff.RepairTool)
	}

	if eff.ConsumeItem != "" {
		has, err := c.store.HasItem(userID, eff.ConsumeItem, 1)
		if err != nil || !has {
			interaction.RespondError(b, i, lang, "mining.event_no_magnet")
			return
		}
		if err := c.store.RemoveInventoryItem(userID, eff.ConsumeItem, 1); err != nil {
			slog.Error("mining event consume failed", "user", userID, "item", eff.ConsumeItem, "error", err)
			interaction.RespondError(b, i, lang, "mining.error")
			return
		}
		for _, id := range magnet.EventPull(eff.ConsumeItem) {
			eff.Items = append(eff.Items, miningsvc.BagEntry{Name: id, Count: 1})
		}
	}

	msg := ""
	if eff.Message != "" {
		if idx := strings.Index(eff.Message, "|WINLINE:"); idx != -1 {
			base := eff.Message[:idx]
			extra := eff.Message[idx+9:]
			msg = i18n.T(base, lang)
			if msg == base && base != "mining.event_none" {
				msg = base
			}
			msg = msg + "\n" + extra
		} else {
			msg = i18n.T(eff.Message, lang)
		}
	}

	// Apply bag/effects under write lock
	sessionsMu.Lock()
	sess, ok = sessions[userID]
	if !ok {
		sessionsMu.Unlock()
		interaction.RespondError(b, i, lang, "mining.error")
		return
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
		sess.riskTurns += eff.RiskTurns
	}
	if eff.DepthGain != 0 {
		sess.depth += eff.DepthGain
	}
	sess.eventCount++
	contractMsg := c.contractDoneMessage(sess, lang)
	if contractMsg != "" {
		if msg != "" {
			msg = msg + "\n\n" + contractMsg
		} else {
			msg = contractMsg
		}
	}
	// capture ForceLeave path
	forceLeave := eff.ForceLeave
	var contractReward *miningsvc.Contract
	var contractDone bool
	var bagForLeave []miningsvc.BagEntry
	var toolForLeave string
	if forceLeave {
		contractReward = sess.contract
		contractDone = contractReward != nil && miningsvc.ContractCompleted(contractReward, sess.depth, sess.bag, sess.depth-1, sess.eventCount)
		bagForLeave = append([]miningsvc.BagEntry(nil), sess.bag...)
		toolForLeave = sess.toolID
		delete(sessions, userID)
	}
	if !forceLeave {
		sessionsMu.Unlock()
		c.persistSession(userID)
		embed, comps := c.mineEmbed(lang, userID, msg, nil)
		c.respond(b, i, embed, comps)
		return
	}
	sessionsMu.Unlock()
	charterMu.Lock()
	delete(charterOffers, userID)
	charterMu.Unlock()
	res, err := c.svc.LeaveMine(userID, bagForLeave, toolForLeave)
	_ = c.svc.DeleteSession(userID)
	if err != nil {
		if errors.Is(err, store.ErrInventoryFull) {
			interaction.RespondError(b, i, lang, "inventory.full")
			return
		}
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
	if contractDone {
		_, _ = c.store.UpdateBalance(userID, contractReward.RewardCredits)
		_, _ = character.AddXP(c.store, userID, contractReward.RewardXP)
		resultMsg += "\n" + i18n.T("mining.contract_reward_msg", lang, map[string]any{"credits": contractReward.RewardCredits, "xp": contractReward.RewardXP})
	}
	if msg != "" {
		resultMsg = msg + "\n\n" + resultMsg
	}
	c.respond(b, i, components.Embed(i18n.T("mining.expedition_complete_title", lang), resultMsg, 0x00FF00), nil)
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

func (c *Cog) onLeave(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	sess := c.loadSession(userID)
	if sess == nil {
		c.respond(b, i, components.Embed(
			i18n.T("mining.title", lang), i18n.T("mining.no_session", lang), 0x4A90D9,
		), nil)
		return
	}

	mu := getUserMu(userID)
	mu.Lock()
	defer mu.Unlock()

	sessionsMu.Lock()
	// re-fetch to ensure not already cleared by concurrent leave
	if s, ok := sessions[userID]; ok {
		sess = s
	} else {
		sessionsMu.Unlock()
		c.respond(b, i, components.Embed(
			i18n.T("mining.title", lang), i18n.T("mining.no_session", lang), 0x4A90D9,
		), nil)
		return
	}
	contractReward := sess.contract
	contractDone := contractReward != nil && miningsvc.ContractCompleted(contractReward, sess.depth, sess.bag, sess.depth-1, sess.eventCount)
	bagCopy := append([]miningsvc.BagEntry(nil), sess.bag...)
	toolCopy := sess.toolID
	delete(sessions, userID)
	sessionsMu.Unlock()
	charterMu.Lock()
	delete(charterOffers, userID)
	charterMu.Unlock()
	res, err := c.svc.LeaveMine(userID, bagCopy, toolCopy)
	if err != nil {
		// restore session on inventory-full so player can retry after freeing space
		if errors.Is(err, store.ErrInventoryFull) {
			sessionsMu.Lock()
			sessions[userID] = sess
			sessionsMu.Unlock()
			interaction.RespondError(b, i, lang, "inventory.full")
			return
		}
		// restore on generic error as well to avoid losing loot
		sessionsMu.Lock()
		sessions[userID] = sess
		sessionsMu.Unlock()
		slog.Error("mining leave failed", "user", userID, "error", err)
		interaction.RespondError(b, i, lang, "mining.error")
		return
	}
	_ = c.svc.DeleteSession(userID)
	if contractDone {
		_, _ = c.store.UpdateBalance(userID, contractReward.RewardCredits)
		_, _ = character.AddXP(c.store, userID, contractReward.RewardXP)
	}

	title, color := c.leaveResultDisplay(res, lang)
	if contractDone {
		title += "\n" + i18n.T("mining.contract_reward_msg", lang, map[string]any{"credits": contractReward.RewardCredits, "xp": contractReward.RewardXP})
	}

	c.respond(b, i, components.Embed(i18n.T("mining.expedition_complete_title", lang), title, color), nil)

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

// leaveResultDisplay renders the exit embed title and color from a leave
// result. The empty case must still pass {xp} so the placeholder never leaks.
func (c *Cog) leaveResultDisplay(res *miningsvc.LeaveResult, lang string) (string, int) {
	title := i18n.T("mining.empty_msg", lang, map[string]any{"xp": res.XP})
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
	return title, color
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
