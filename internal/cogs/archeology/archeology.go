package archeology

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	archsvc "guacagamblebot/internal/service/archeology"
	furnituresvc "guacagamblebot/internal/service/furniture"
	invsvc "guacagamblebot/internal/service/inventory"
	jsvc "guacagamblebot/internal/service/journal"
	npcsvc "guacagamblebot/internal/service/npcs"
	researchsvc "guacagamblebot/internal/service/research"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
)

var digSessions = map[int64]*digSession{}
var digSessionsMu sync.Mutex

// collectedTokens tracks the one-time collect tokens embedded in keep/sell
// buttons so a double-click or a retry after an interaction timeout can never
// award the same fossil twice.
var collectedTokens = map[string]time.Time{}
var collectedTokensMu sync.Mutex

const (
	collectTokenTTL = 24 * time.Hour
	collectTokenMax = 2048
)

// claimCollectToken marks a token as collected, returning false when it was
// already claimed (the fossil has been handled). The map is pruned on insert
// so it stays bounded.
func claimCollectToken(token string) bool {
	if token == "" {
		return true
	}
	collectedTokensMu.Lock()
	defer collectedTokensMu.Unlock()
	if len(collectedTokens) >= collectTokenMax {
		now := time.Now()
		for t, ts := range collectedTokens {
			if now.Sub(ts) > collectTokenTTL {
				delete(collectedTokens, t)
			}
		}
	}
	if _, ok := collectedTokens[token]; ok {
		return false
	}
	collectedTokens[token] = time.Now()
	return true
}

// releaseCollectToken undoes a claim when processing failed, so the player can
// retry.
func releaseCollectToken(token string) {
	if token == "" {
		return
	}
	collectedTokensMu.Lock()
	delete(collectedTokens, token)
	collectedTokensMu.Unlock()
}

type digSession struct {
	state   *archsvc.GameState
	pending *archsvc.DigResult
}

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *archsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	def := universe.Get(cfg.Universe)
	if def == nil {
		def = universe.Get("hoakhaven")
	}
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)
	c := &Cog{store: s, cfg: cfg, svc: archsvc.New(s, cfg, npcSvc)}
	r.Slash("dig", "Archaeology fossil excavation", c.onSlashMenu)
	r.Prefix("dig", c.onPrefixMenu)
	r.Prefix("arch", c.onPrefixMenu)
	r.Component("arch", "menu", c.onMenu)
	r.Component("arch", "site", c.onSiteSelect)
	r.Component("arch", "action", c.onAction)
	r.Component("arch", "event", c.onEventChoice)
	r.Component("arch", "post", c.onPostExtract)
	r.Component("arch", "dust", c.onDustMenu)
	r.Component("arch", "dustpick", c.onDustPick)
	r.Modal("arch", "grind", c.onGrindModal)
	r.Prefix("reanimate", c.onPrefixReanimate)
	r.Prefix("rl", c.onPrefixReanimateList)
	r.Prefix("reanimatelist", c.onPrefixReanimateList)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(interaction.UserID(i))
	digSessionsMu.Lock()
	delete(digSessions, userID)
	digSessionsMu.Unlock()
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.bureau(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	digSessionsMu.Lock()
	delete(digSessions, userID)
	digSessionsMu.Unlock()
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	embed, comps := c.bureau(lang, userID)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) onMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(interaction.UserID(i))
	digSessionsMu.Lock()
	delete(digSessions, userID)
	digSessionsMu.Unlock()
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.bureau(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) bureau(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	level := c.svc.GetArcheologistLevel(userID)
	xp, xpNext := c.svc.GetArcheologistXP(userID)

	xpBar := progressBar(xp, xpNext, 12)
	timeFlavor := timeOfDayFlavor(lang)

	mastery := c.svc.GetToolMastery(userID)
	masteryLines := ""
	for _, toolID := range []string{"dynamite", "hammer", "brush"} {
		m := mastery[toolID]
		titleI18n := ""
		if m.TitleID != "" {
			titleI18n = " " + i18n.T(m.TitleID, lang)
		}
		toolDisplay := i18n.T("arch_tool_"+toolID, lang)
		masteryLines += toolDisplay + ": " + strconv.Itoa(m.Uses) + " uses" + titleI18n + "\n"
	}

	journal, journalMax := c.svc.GetJournalProgress(userID)
	totalFossils := c.svc.GetTotalFossilDigs(userID)

	tipKey := dailyDigTip()

	_, remaining, _ := c.store.CheckGameLimit(userID, "dig", 10)

	desc := i18n.T("arch.bureau_desc", lang, map[string]any{
		"level":        level,
		"xpbar":        xpBar,
		"xp":           xp,
		"xpnext":       xpNext,
		"time":         timeFlavor,
		"mastery":      masteryLines,
		"journal":      journal,
		"jmax":         journalMax,
		"totalfossils": totalFossils,
		"remaining":    remaining,
		"tip":          i18n.T(tipKey, lang),
	})

	embed := components.Embed(
		i18n.T("arch.bureau_title", lang),
		desc,
		0x8B4513,
	)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("arch.bureau_footer", lang, map[string]any{"remaining": remaining}),
	}

	sites := c.svc.GetSiteInfo(userID)
	var btns []discordgo.MessageComponent
	for _, site := range sites {
		label := i18n.T(site.NameID, lang)
		info := ""
		if site.Cost > 0 {
			info = i18n.T("arch.site_cost_tag", lang, map[string]any{"cost": site.Cost})
		}
		if !site.Unlocked {
			info = i18n.T("arch.site_lvl_tag", lang, map[string]any{"level": site.MinLevel})
			if site.Cost > 0 {
				info = i18n.T("arch.site_cost_tag", lang, map[string]any{"cost": site.Cost}) + " " + info
			}
			label = "🔒 " + label
		}
		if info != "" {
			label += " " + info
		}
		style := discordgo.SecondaryButton
		if site.Unlocked {
			switch site.Key {
			case "riverbed":
				style = discordgo.SuccessButton
			case "cliffside":
				style = discordgo.PrimaryButton
			case "fault":
				style = discordgo.DangerButton
			case "ice_sheet":
				style = discordgo.PrimaryButton
			case "volcanic":
				style = discordgo.DangerButton
			}
		}
		btns = append(btns, components.Button(label, components.EncodeOwner(userID, "arch", "site", site.Key), style))
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(btns[:3]...),
	}
	if len(btns) > 3 {
		comps = append(comps, components.ActionRow(btns[3:]...))
	}
	comps = append(comps, components.ActionRow(
		components.Button(i18n.T("arch.dust_btn", lang), components.EncodeOwner(userID, "arch", "dust"), discordgo.SecondaryButton),
	))

	return embed, comps
}

func (c *Cog) onSiteSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	siteKey := "riverbed"
	if len(rest) > 0 {
		siteKey = rest[0]
	}

	state, err := c.svc.NewGame(userID, siteKey)
	if err != nil {
		errKey := "arch.error"
		switch err {
		case archsvc.ErrDigLimit:
			errKey = "arch.limit_reached"
		case archsvc.ErrNoMoney:
			errKey = "arch.no_money"
		case archsvc.ErrLocked:
			errKey = "arch.site_locked"
		case store.ErrInventoryFull:
			errKey = "inventory.full"
		}
		interaction.RespondError(b, i, lang, errKey)
		return
	}

	digSessionsMu.Lock()
	digSessions[userID] = &digSession{state: state}
	digSessionsMu.Unlock()
	c.showDigEmbed(b, i, lang, state, "")
}

func (c *Cog) onAction(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)

	digSessionsMu.Lock()
	sess, ok := digSessions[userID]
	digSessionsMu.Unlock()
	if !ok || sess.state == nil || sess.state.Finished {
		interaction.RespondError(b, i, lang, "arch.session_expired")
		return
	}

	if len(rest) < 1 {
		interaction.RespondError(b, i, lang, "arch.error")
		return
	}

	actionName := rest[0]

	if actionName == "scan" {
		outcome := c.svc.ApplyAction(sess.state, archsvc.ActionScan)
		if outcome.Finished {
			result := c.svc.Resolve(sess.state)
			sess.pending = result
			c.showResultEmbed(b, i, lang, result)
			return
		}
		c.showDigEmbed(b, i, lang, &outcome.State, i18n.T("arch.scan_feedback", lang))
		return
	}

	var act archsvc.ActionType
	switch actionName {
	case "continue":
		c.showDigEmbed(b, i, lang, sess.state, "")
		return
	case "dynamite":
		act = archsvc.ActionDynamite
	case "hammer":
		act = archsvc.ActionHammer
	case "brush":
		act = archsvc.ActionBrush
	default:
		interaction.RespondError(b, i, lang, "arch.error")
		return
	}

	outcome := c.svc.ApplyAction(sess.state, act)

	evt := c.svc.RollEvent(sess.state)
	if evt != nil {
		digSessionsMu.Lock()
		digSessions[userID] = &digSession{state: &outcome.State}
		digSessionsMu.Unlock()
		c.showEventEmbed(b, i, lang, evt, &outcome.State)
		return
	}

	if outcome.Finished {
		result := c.svc.Resolve(sess.state)
		sess.pending = result
		c.showResultEmbed(b, i, lang, result)
		return
	}

	c.showDigEmbed(b, i, lang, &outcome.State, digFeedback(lang, outcome, act))
}

// digFeedback renders the per-action result line shown at the top of the dig
// screen so the player sees how much progress a tool made.
func digFeedback(lang string, outcome *archsvc.ActionOutcome, action archsvc.ActionType) string {
	integrity := ""
	if outcome.IntLoss > 0 {
		integrity = i18n.T("arch.fb_integrity", lang, map[string]any{"loss": outcome.IntLoss})
	}
	layer := ""
	if outcome.LayerShift {
		layer = i18n.T("arch.fb_layer", lang)
	}
	return i18n.T("arch.action_feedback", lang, map[string]any{
		"tool": i18n.T("arch_tool_"+string(action), lang), "depth": outcome.DepthRem,
		"integrity": integrity, "layer": layer,
	})
}

// collectToken extracts the one-time collect token from a keep/sell button
// payload (new-format buttons only; legacy payloads carry none).
func collectToken(rest []string) string {
	if len(rest) < 9 {
		return ""
	}
	return rest[7]
}

// decodeDigResult reconstructs a DigResult from the data payload embedded in a
// keep/sell button custom_id (rest after the action). It returns nil when the
// payload is absent or malformed, in which case the caller falls back to the
// in-memory session.
func decodeDigResult(rest []string) *archsvc.DigResult {
	if len(rest) < 7 {
		return nil
	}
	value, err1 := strconv.Atoi(rest[2])
	integrity, err2 := strconv.Atoi(rest[4])
	xp, err3 := strconv.Atoi(rest[5])
	qty, err4 := strconv.Atoi(rest[6])
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return nil
	}
	if rest[1] == "" || rest[3] == "" {
		return nil
	}
	return &archsvc.DigResult{
		ItemName:  rest[1],
		Value:     value,
		Quality:   rest[3],
		Integrity: integrity,
		XP:        xp,
		Quantity:  qty,
	}
}

func (c *Cog) onEventChoice(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)

	digSessionsMu.Lock()
	sess, ok := digSessions[userID]
	digSessionsMu.Unlock()
	if !ok || sess.state == nil {
		c.onMenu(b, i)
		return
	}

	if len(rest) < 2 {
		c.showDigEmbed(b, i, lang, sess.state, "")
		return
	}

	actionVal := rest[0]

	evtType, _ := strconv.Atoi(rest[1])

	evt := &archsvc.DigEvent{Type: archsvc.EventType(evtType)}
	result := c.svc.ResolveEvent(sess.state, evt, actionVal)

	var desc string
	if result.RevealedTool != "" {
		desc = i18n.T(result.DescID, lang, map[string]any{
			"tool":  i18n.T("arch_tool_"+result.RevealedTool, lang),
			"layer": i18n.T(archsvc.GetLayerNameID(result.RevealedLayer), lang),
		})
	} else {
		desc = i18n.T(result.DescID, lang)
	}
	if result.CoinChange > 0 {
		desc += "\n\n" + i18n.T("arch.coins_gained", lang, map[string]any{"coins": result.CoinChange})
		c.store.UpdateBalance(userID, result.CoinChange)
	} else if result.CoinChange < 0 {
		desc += "\n\n" + i18n.T("arch.coins_spent", lang, map[string]any{"coins": -result.CoinChange})
		c.store.UpdateBalance(userID, result.CoinChange)
	}
	if result.ItemGiven != "" {
		c.svc.AwardResult(userID, &archsvc.DigResult{ItemName: result.ItemGiven, Value: 0, XP: 0})
		desc += "\n\n" + i18n.T("arch.item_received", lang, map[string]any{"item": items.LocalizedName(result.ItemGiven, lang), "qty": result.ItemQty})
	}

	if result.BackToDig && sess.state != nil && !sess.state.Finished {
		embed := components.Embed(i18n.T(result.TitleID, lang), desc, 0x006400)
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("arch.back_dig", lang), components.EncodeOwner(userID, "arch", "action", "continue"), discordgo.SecondaryButton),
			),
		}
		digSessionsMu.Lock()
		digSessions[userID] = &digSession{state: sess.state}
		digSessionsMu.Unlock()
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
		return
	}

	if sess.state.Finished {
		result := c.svc.Resolve(sess.state)
		sess.pending = result
		c.showResultEmbed(b, i, lang, result)
		return
	}

	embed := components.Embed(i18n.T(result.TitleID, lang), desc, 0x006400)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("arch.back_menu", lang), components.EncodeOwner(userID, "arch", "menu"), discordgo.SecondaryButton),
		),
	}
	digSessionsMu.Lock()
	delete(digSessions, userID)
	digSessionsMu.Unlock()
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onPostExtract(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)

	action := "keep"
	if len(rest) > 0 {
		action = rest[0]
	}

	token := ""
	res := decodeDigResult(rest)
	if res == nil {
		// Legacy buttons carry no result data; fall back to the in-memory
		// session so nothing is lost.
		digSessionsMu.Lock()
		sess, ok := digSessions[userID]
		delete(digSessions, userID)
		digSessionsMu.Unlock()
		if !ok || sess.pending == nil {
			interaction.RespondError(b, i, lang, "arch.session_expired")
			return
		}
		res = sess.pending
	} else {
		token = collectToken(rest)
		digSessionsMu.Lock()
		delete(digSessions, userID)
		digSessionsMu.Unlock()
	}

	if !claimCollectToken(token) {
		interaction.RespondError(b, i, lang, "arch.already_collected")
		return
	}

	var embed *discordgo.MessageEmbed
	var serr error
	switch action {
	case "sell":
		var price int
		var lucky bool
		var mult float64
		price, _, lucky, mult, serr = c.svc.SellResult(userID, res)
		if serr == nil {
			desc := i18n.T("arch.sold_desc", lang, map[string]any{"item": items.LocalizedName(res.ItemName, lang), "coins": price})
			if lucky {
				desc += "\n\n" + i18n.T("arch.sold_lucky", lang, map[string]any{"mult": fmt.Sprintf("%.2f", mult)})
			}
			embed = components.Embed(
				i18n.T("arch.sold_title", lang),
				desc,
				0xF1C40F,
			)
		}

	default:
		serr = c.svc.AwardResult(userID, res)
		if serr == nil {
			xpStr := ""
			if res.XP > 0 {
				xpStr = i18n.T("arch.xp_gained", lang, map[string]any{"xp": res.XP})
			}
			embed = components.Embed(
				i18n.T("arch.keep_title", lang),
				i18n.T("arch.keep_desc", lang, map[string]any{"item": items.LocalizedName(res.ItemName, lang), "xp": xpStr}),
				0x00FF00,
			)
		}
	}
	if serr != nil {
		releaseCollectToken(token)
		interaction.RespondError(b, i, lang, "arch.error")
		return
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("arch.back_menu", lang), components.EncodeOwner(userID, "arch", "menu"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))

	// Follow-ups only work after the interaction is acknowledged, and they run
	// outside the 3s response window, so the heavy quest/journal/achievement
	// queries no longer risk a Discord "did not respond in time".
	if n, ok := c.store.PopQuestNotification(userID); ok {
		interaction.SendQuestNotification(b, i, n, lang)
	}

	if text, dm := jsvc.SceneLine(c.store, userID, "archeology", lang); text != "" {
		interaction.SendJournalScene(b, i, text, dm)
	}
	unlocks, uerr := achievement.CheckAndUnlock(b.DB, userID)
	if uerr == nil && len(unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, unlocks)
	}
}

// onDustMenu opens the fossil grinder view: a select menu of the player's
// grindable fossils with their bone dust rates.
func (c *Cog) onDustMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.buildGrindView(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) buildGrindView(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	var rateLines []string
	for _, itemID := range archsvc.GrindableOrder {
		rateLines = append(rateLines, i18n.T("arch.dust_rate_line", lang, map[string]any{
			"item": items.LocalizedName(itemID, lang), "rate": archsvc.DustRates[itemID],
		}))
	}
	embed := components.Embed(
		i18n.T("arch.dust_title", lang),
		i18n.T("arch.dust_desc", lang, map[string]any{"rates": strings.Join(rateLines, "\n")}),
		0x8B4513,
	)

	var inv []model.Inventory
	c.store.DB.Where("user_id = ? AND quantity > 0", userID).Find(&inv)
	var opts []discordgo.SelectMenuOption
	for _, iv := range inv {
		rate, ok := archsvc.DustRates[iv.ItemID]
		if !ok {
			continue
		}
		label := fmt.Sprintf("%s (x%d)", items.LocalizedName(iv.ItemID, lang), iv.Quantity)
		if len(label) > 100 {
			label = label[:97] + "..."
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label:       label,
			Value:       iv.ItemID,
			Emoji:       &discordgo.ComponentEmoji{Name: "🦴"},
			Description: i18n.T("arch.dust_rate_desc", lang, map[string]any{"rate": rate}),
		})
	}
	if len(opts) == 0 {
		opts = append(opts, discordgo.SelectMenuOption{
			Label:       i18n.T("arch.dust_empty", lang),
			Value:       "_none",
			Description: i18n.T("arch.dust_empty_desc", lang),
			Default:     true,
		})
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(discordgo.SelectMenu{
			CustomID:    components.EncodeOwner(userID, "arch", "dustpick"),
			Placeholder: i18n.T("arch.dust_select_placeholder", lang),
			Options:     opts,
		}),
		components.ActionRow(
			components.Button(i18n.T("arch.back_menu", lang), components.EncodeOwner(userID, "arch", "menu"), discordgo.SecondaryButton),
		),
	}
	return embed, comps
}

// onDustPick opens the grind amount modal for the selected fossil.
func (c *Cog) onDustPick(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	itemID := i.MessageComponentData().Values[0]
	if itemID == "_none" {
		interaction.RespondError(b, i, lang, "arch.dust_empty")
		return
	}
	if _, ok := archsvc.DustRates[itemID]; !ok {
		interaction.RespondError(b, i, lang, "arch.error")
		return
	}
	modal := components.ModalResponse(
		components.EncodeOwner(userID, "arch", "grind", itemID),
		i18n.T("arch.dust_modal_title", lang, map[string]any{"item": items.LocalizedName(itemID, lang)}),
		components.TextInput("amount",
			i18n.T("arch.dust_modal_label", lang, map[string]any{"rate": archsvc.DustRates[itemID]}), true, "1",
			discordgo.TextInputShort, 1, 5),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

// onGrindModal performs the grind and reports the bone dust gained.
func (c *Cog) onGrindModal(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.ModalSubmitData().CustomID)
	if len(rest) < 1 {
		interaction.RespondError(b, i, lang, "arch.error")
		return
	}
	itemID := rest[0]
	amountStr := strings.TrimSpace(interaction.ModalValues(i)["amount"])
	amount, err := strconv.Atoi(amountStr)
	if err != nil || amount <= 0 || amount > 99999 {
		interaction.RespondError(b, i, lang, "arch.dust_error_amount")
		return
	}

	dust, err := c.svc.GrindFossils(userID, itemID, amount)
	if err != nil {
		switch err {
		case archsvc.ErrNotEnoughFossils:
			interaction.RespondError(b, i, lang, "arch.dust_error_no_fossils")
		default:
			interaction.RespondError(b, i, lang, "arch.error")
		}
		return
	}

	embed := components.Embed(
		i18n.T("arch.dust_success_title", lang),
		i18n.T("arch.dust_success_desc", lang, map[string]any{
			"qty": amount, "item": items.LocalizedName(itemID, lang), "dust": dust,
		}),
		0x8B4513,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("arch.back_menu", lang), components.EncodeOwner(userID, "arch", "menu"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: comps,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) showDigEmbed(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, state *archsvc.GameState, feedback string) {
	userID := interaction.ToInt64(interaction.UserID(i))
	dug := state.MaxDepth - state.Depth
	if dug < 0 {
		dug = 0
	}
	if dug > state.MaxDepth {
		dug = state.MaxDepth
	}
	blocksFull := 0
	if state.MaxDepth > 0 {
		blocksFull = dug * 5 / state.MaxDepth
	}
	if blocksFull > 5 {
		blocksFull = 5
	}
	depthBar := ""
	for j := 0; j < blocksFull; j++ {
		depthBar += "🟫"
	}
	for j := blocksFull; j < 5; j++ {
		depthBar += "⬛"
	}

	intPct := float64(state.Integrity) / 100.0
	if intPct < 0 {
		intPct = 0
	}
	intBlocks := int(intPct * 5)
	intBar := ""
	for j := 0; j < intBlocks; j++ {
		intBar += "❤️"
	}
	for j := intBlocks; j < 5; j++ {
		intBar += "💔"
	}

	layerEmoji := archsvc.GetLayerEmoji(state.CurrentLayer)
	layerNameID := archsvc.GetLayerNameID(state.CurrentLayer)
	layerName := i18n.T(layerNameID, lang)

	desc := i18n.T("arch.dig_desc", lang, map[string]any{
		"site":      i18n.T(state.Site.NameID, lang),
		"layer":     layerEmoji + " " + layerName,
		"site_desc": i18n.T(state.Site.DescID, lang),
		"maxdepth":  state.MaxDepth,
	})

	if feedback != "" {
		desc = feedback + "\n\n" + desc
	}

	if state.RevealedLayer {
		effDyna := c.svc.GetToolEffectiveness(state, archsvc.ActionDynamite)
		effHamm := c.svc.GetToolEffectiveness(state, archsvc.ActionHammer)
		effBrush := c.svc.GetToolEffectiveness(state, archsvc.ActionBrush)
		desc += "\n\n" + i18n.T("arch.scan_result", lang, map[string]any{
			"dynamite_depth": effDyna["depth"], "dynamite_risk": effDyna["risk"],
			"hammer_depth": effHamm["depth"], "hammer_risk": effHamm["risk"],
			"brush_depth": effBrush["depth"], "brush_risk": effBrush["risk"],
		})
	}

	embed := components.Embed(
		i18n.T("arch.dig_title", lang),
		desc,
		state.Site.Color,
	)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("arch.depth_label", lang), i18n.T("arch.depth_value", lang, map[string]any{"bar": depthBar, "dug": dug, "max": state.MaxDepth, "left": state.Depth}), false),
		components.Field(i18n.T("arch.integrity_label", lang), intBar+" "+itoa(state.Integrity)+"%", false),
		components.Field(i18n.T("arch.actions_label", lang), "**"+itoa(state.Actions)+"**", true),
		components.Field(i18n.T("arch.layer_label", lang), layerEmoji+" "+layerName, true),
	}

	if state.CursedDebuff {
		embed.Fields = append(embed.Fields, components.Field("⚠️", i18n.T("arch.cursed_debuff", lang), false))
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("arch.dynamite_btn", lang), components.EncodeOwner(userID, "arch", "action", "dynamite"), discordgo.DangerButton),
			components.Button(i18n.T("arch.hammer_btn", lang), components.EncodeOwner(userID, "arch", "action", "hammer"), discordgo.PrimaryButton),
			components.Button(i18n.T("arch.brush_btn", lang), components.EncodeOwner(userID, "arch", "action", "brush"), discordgo.SuccessButton),
			components.Button(i18n.T("arch.scan_btn", lang), components.EncodeOwner(userID, "arch", "action", "scan"), discordgo.SecondaryButton),
		),
	}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) showEventEmbed(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, evt *archsvc.DigEvent, state *archsvc.GameState) {
	userID := interaction.ToInt64(interaction.UserID(i))
	desc := i18n.T(evt.DescID, lang)

	embed := components.Embed(
		i18n.T(evt.TitleID, lang),
		desc,
		0x9B59B6,
	)

	var btns []discordgo.MessageComponent
	for _, ch := range evt.Choices {
		style := discordgo.ButtonStyle(ch.Style)
		if style == 0 {
			style = discordgo.PrimaryButton
		}
		customID := components.EncodeOwner(userID, "arch", "event", ch.Value, itoa(int(evt.Type)))
		btns = append(btns, components.Button(i18n.T(ch.LabelID, lang), customID, style))
	}

	comps := []discordgo.MessageComponent{components.ActionRow(btns...)}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) showResultEmbed(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, res *archsvc.DigResult) {
	uid := interaction.ToInt64(interaction.UserID(i))
	color := 0x00FF00
	switch res.Quality {
	case "disaster":
		color = 0xE74C3C
	case "damaged":
		color = 0xF39C12
	case "shadow":
		color = 0x000000
	case "cursed":
		color = 0x8B0000
	case "living":
		color = 0x00CED1
	case "journal":
		color = 0x8B4513
	case "pure_dna":
		color = 0x9B59B6
	case "legendary":
		color = 0xFFD700
	case "epic":
		color = 0x9B59B6
	case "rare":
		color = 0x3498DB
	}

	desc := outcomeDesc(res, lang)

	embed := components.Embed(
		i18n.T("arch.result_title", lang),
		desc,
		color,
	)

	var btns []discordgo.MessageComponent
	if res.Quality != "disaster" && res.Quality != "damaged" {
		token := fmt.Sprintf("%x", rand.Uint32())
		resParts := []string{res.ItemName, itoa(res.Value), res.Quality, itoa(res.Integrity), itoa(res.XP), itoa(res.Quantity), token}
		keepCustomID := components.EncodeOwner(uid, append([]string{"arch", "post", "keep"}, resParts...)...)
		sellCustomID := components.EncodeOwner(uid, append([]string{"arch", "post", "sell"}, resParts...)...)
		btns = append(btns, components.Button(i18n.T("arch.keep_btn", lang), keepCustomID, discordgo.SuccessButton))
		btns = append(btns, components.Button(i18n.T("arch.sell_btn", lang), sellCustomID, discordgo.PrimaryButton))
	} else {
		c.svc.AwardResult(userID(interaction.UserID(i)), res)
	}

	comps := []discordgo.MessageComponent{}
	if len(btns) > 0 {
		comps = append(comps, components.ActionRow(btns...))
	}
	comps = append(comps, components.ActionRow(
		components.Button(i18n.T("arch.back_menu", lang), components.EncodeOwner(uid, "arch", "menu"), discordgo.SecondaryButton),
	))

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func outcomeDesc(res *archsvc.DigResult, lang string) string {
	itemName := items.LocalizedName(res.ItemName, lang)
	qtyStr := ""
	if res.Quantity > 1 {
		qtyStr = " x" + itoa(res.Quantity)
	}
	received := i18n.T("arch.received", lang, map[string]any{"item": itemName + qtyStr})
	switch res.Quality {
	case "disaster":
		return i18n.T("arch.disaster_msg", lang) + "\n" + received
	case "damaged":
		return i18n.T("arch.damaged_msg", lang) + "\n" + received
	case "shadow":
		return i18n.T("arch.shadow_msg", lang) + "\n" + received
	case "cursed":
		return i18n.T("arch.cursed_msg", lang) + "\n" + received
	case "living":
		return i18n.T("arch.living_msg", lang) + "\n" + received
	case "journal":
		return i18n.T("arch.journal_msg", lang) + "\n" + received
	default:
		return i18n.T("arch.success_msg", lang, map[string]any{"item": itemName + qtyStr, "quality": i18n.T("arch.quality_"+res.Quality, lang), "integrity": res.Integrity}) + "\n" + received
	}
}

func (c *Cog) onPrefixReanimate(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	parts := strings.Fields(m.Content)
	rarity := ""
	if len(parts) >= 2 {
		rarity = strings.ToLower(parts[1])
	}

	if rarity == "list" || rarity == "l" {
		c.sendReanimateList(b, s, lang, userID, m.ChannelID)
		return
	}

	if rarity == "" {
		_, _ = b.Session.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: i18n.T("arch.reanimate_cmd_usage", lang),
		})
		return
	}

	rarityMap := map[string]string{
		"common": "common", "commun": "common",
		"rare": "rare",
		"epic": "epic", "epique": "epic",
		"legendary": "legendary", "legendaire": "legendary",
		"pure": "pure_dna", "pur": "pure_dna", "dna": "pure_dna",
	}
	resolvedRarity, ok := rarityMap[rarity]
	if !ok {
		_, _ = b.Session.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: i18n.T("arch.reanimate_cmd_invalid", lang),
		})
		return
	}

	pool, ok := archsvc.ReanimatePools[resolvedRarity]
	if !ok {
		return
	}

	petName, success, err := c.svc.Reanimate(userID, resolvedRarity)
	if err != nil {
		content := i18n.T("arch.reanimate_cmd_no_fossils", lang, map[string]any{"count": 5, "item": items.LocalizedName(pool.ItemName, lang)})
		switch {
		case errors.Is(err, archsvc.ErrNoGeneticsLab):
			content = i18n.T("arch.reanimate_no_lab", lang)
		case errors.Is(err, archsvc.ErrResearchRequired):
			resName := archsvc.ReanimateResearch[resolvedRarity]
			if rd := researchsvc.ResearchDefs[resName]; rd != nil {
				resName = rd.Name
			}
			content = i18n.T("arch.reanimate_no_research", lang, map[string]any{
				"research": resName,
				"rarity":   i18n.T("arch.quality_"+resolvedRarity, lang),
			})
		}
		_, _ = b.Session.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: content,
		})
		return
	}

	if success {
		embed := components.Embed(
			i18n.T("arch.reanimate_success_title", lang),
			i18n.T("arch.reanimate_success_desc", lang, map[string]any{"pet": petName}),
			0x9B59B6,
		)
		_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}})
	} else {
		embed := components.Embed(
			i18n.T("arch.reanimate_fail_title", lang),
			i18n.T("arch.reanimate_fail_desc", lang),
			0xE74C3C,
		)
		_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}})
	}
}

func (c *Cog) onPrefixReanimateList(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	c.sendReanimateList(b, s, lang, userID, m.ChannelID)
}

func (c *Cog) sendReanimateList(b *interaction.Bot, s *discordgo.Session, lang string, userID int64, channelID string) {
	hasLab := furnituresvc.HasFurniture(c.store, userID, "genetics_lab")

	desc := ""
	for rarity, pool := range archsvc.ReanimatePools {
		count := c.svc.GetFossilCount(userID, pool.ItemName)
		rarityName := i18n.T("arch.quality_"+rarity, lang)
		line := i18n.T("arch.reanimate_list_line", lang, map[string]any{
			"rarity": rarityName,
			"count":  count,
			"item":   items.LocalizedName(pool.ItemName, lang),
		})
		if !hasLab {
			line += " " + i18n.T("arch.reanimate_list_lab", lang)
		} else {
			resID := archsvc.ReanimateResearch[rarity]
			if !c.researchCompleted(userID, resID) {
				resName := resID
				if rd := researchsvc.ResearchDefs[resID]; rd != nil {
					resName = rd.Name
				}
				line += " " + i18n.T("arch.reanimate_list_research", lang, map[string]any{"name": resName})
			} else {
				line += " ✅"
			}
		}
		desc += line + "\n"
	}

	if desc == "" {
		desc = i18n.T("arch.reanimate_list_empty", lang)
	}

	embed := components.Embed(
		i18n.T("arch.reanimate_list_title", lang),
		desc,
		0x9B59B6,
	)
	_, _ = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}})
}

func (c *Cog) researchCompleted(userID int64, researchID string) bool {
	var r model.UserResearch
	return c.store.DB.Where("user_id = ? AND research_id = ? AND completed = ?", userID, researchID, true).First(&r).Error == nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func progressBar(current, max, width int) string {
	if max <= 0 {
		return strings.Repeat("░", width)
	}
	filled := current * width / max
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func timeOfDayFlavor(lang string) string {
	hour := time.Now().UTC().Hour()
	switch {
	case hour < 6:
		return i18n.T("arch.time_night", lang)
	case hour < 12:
		return i18n.T("arch.time_morning", lang)
	case hour < 18:
		return i18n.T("arch.time_afternoon", lang)
	default:
		return i18n.T("arch.time_evening", lang)
	}
}

func dailyDigTip() string {
	tips := []string{
		"arch.tip_1", "arch.tip_2", "arch.tip_3", "arch.tip_4",
		"arch.tip_5", "arch.tip_6", "arch.tip_7", "arch.tip_8",
	}
	day := time.Now().UTC().YearDay()
	return tips[day%len(tips)]
}

func userID(id string) int64 {
	n, _ := strconv.ParseInt(id, 10, 64)
	return n
}
