package archeology

import (
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	archsvc "guacagamblebot/internal/service/archeology"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/store"
)

var digSessions = map[int64]*digSession{}

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
	c := &Cog{store: s, cfg: cfg, svc: archsvc.New(s, cfg)}
	r.Slash("dig", "Archaeology fossil excavation", c.onSlashMenu)
	r.Prefix("dig", c.onPrefixMenu)
	r.Prefix("arch", c.onPrefixMenu)
	r.Component("arch", "menu", c.onMenu)
	r.Component("arch", "site", c.onSiteSelect)
	r.Component("arch", "action", c.onAction)
	r.Component("arch", "event", c.onEventChoice)
	r.Component("arch", "post", c.onPostExtract)
	r.Prefix("reanimate", c.onPrefixReanimate)
	r.Prefix("rl", c.onPrefixReanimateList)
	r.Prefix("reanimatelist", c.onPrefixReanimateList)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(interaction.UserID(i))
	delete(digSessions, userID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.bureau(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	delete(digSessions, userID)
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	embed, comps := c.bureau(lang, userID)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) onMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(interaction.UserID(i))
	delete(digSessions, userID)
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

	desc := i18n.T("arch.bureau_desc", lang, map[string]any{
		"level":    level,
		"xpbar":    xpBar,
		"xp":       xp,
		"xpnext":   xpNext,
		"time":     timeFlavor,
		"mastery":  masteryLines,
		"journal":  journal,
		"jmax":     journalMax,
		"totalfossils": totalFossils,
		"tip":      i18n.T(tipKey, lang),
	})

	_, remaining, _ := c.store.CheckGameLimit(userID, "dig", 10)

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
		if !site.Unlocked {
			label = "🔒 " + label
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
		btns = append(btns, components.Button(label, components.Encode("arch", "site", site.Key), style))
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(btns[:3]...),
	}
	if len(btns) > 3 {
		comps = append(comps, components.ActionRow(btns[3:]...))
	}

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
		}
		interaction.RespondError(b, i, lang, errKey)
		return
	}

	digSessions[userID] = &digSession{state: state}
	c.showDigEmbed(b, i, lang, state)
}

func (c *Cog) onAction(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, action, rest := components.Decode(cid)

	sess, ok := digSessions[userID]
	if !ok || sess.state == nil || sess.state.Finished {
		interaction.RespondError(b, i, lang, "arch.error")
		return
	}

	if action == "scan" {
		outcome := c.svc.ApplyAction(sess.state, archsvc.ActionScan)
		if outcome.Finished {
			result := c.svc.Resolve(sess.state)
			sess.pending = result
			c.showResultEmbed(b, i, lang, result)
			return
		}
		c.showDigEmbed(b, i, lang, &outcome.State)
		return
	}

	var act archsvc.ActionType
	switch action {
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

	_ = rest
	outcome := c.svc.ApplyAction(sess.state, act)

	evt := c.svc.RollEvent(sess.state)
	if evt != nil {
		digSessions[userID] = &digSession{state: &outcome.State}
		c.showEventEmbed(b, i, lang, evt, &outcome.State)
		return
	}

	if outcome.Finished {
		result := c.svc.Resolve(sess.state)
		sess.pending = result
		c.showResultEmbed(b, i, lang, result)
		return
	}

	c.showDigEmbed(b, i, lang, &outcome.State)
}

func (c *Cog) onEventChoice(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	parts := strings.Split(cid, "::")

	sess, ok := digSessions[userID]
	if !ok || sess.state == nil {
		c.onMenu(b, i)
		return
	}

	if len(parts) < 4 {
		c.showDigEmbed(b, i, lang, sess.state)
		return
	}

	actionVal := parts[3]

	var evtType int
	if len(parts) >= 5 {
		evtType, _ = strconv.Atoi(parts[4])
	}

	evt := &archsvc.DigEvent{Type: archsvc.EventType(evtType)}
	result := c.svc.ResolveEvent(sess.state, evt, actionVal)

	desc := i18n.T(result.DescID, lang)
	if result.CoinChange > 0 {
		desc += "\n\n" + i18n.T("arch.coins_gained", lang, map[string]any{"coins": result.CoinChange})
		c.store.UpdateBalance(userID, result.CoinChange)
	} else if result.CoinChange < 0 {
		desc += "\n\n" + i18n.T("arch.coins_spent", lang, map[string]any{"coins": -result.CoinChange})
		c.store.UpdateBalance(userID, result.CoinChange)
	}
	if result.ItemGiven != "" {
		c.svc.AwardResult(userID, &archsvc.DigResult{ItemName: result.ItemGiven, Value: 0, XP: 0})
		desc += "\n\n" + i18n.T("arch.item_received", lang, map[string]any{"item": items.DisplayName(result.ItemGiven), "qty": result.ItemQty})
	}

	if result.BackToDig && sess.state != nil && !sess.state.Finished {
		embed := components.Embed(i18n.T(result.TitleID, lang), desc, 0x006400)
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("arch.back_dig", lang), components.Encode("arch", "action", "continue"), discordgo.SecondaryButton),
			),
		}
		digSessions[userID] = &digSession{state: sess.state}
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
			components.Button(i18n.T("arch.back_menu", lang), components.Encode("arch", "menu"), discordgo.SecondaryButton),
		),
	}
	delete(digSessions, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onPostExtract(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)

	sess, ok := digSessions[userID]
	if !ok || sess.pending == nil {
		interaction.RespondError(b, i, lang, "arch.error")
		return
	}

	res := sess.pending
	action := "keep"
	if len(rest) > 0 {
		action = rest[0]
	}

	var embed *discordgo.MessageEmbed
	switch action {
	case "sell":
		bal, err := c.svc.SellResult(userID, res)
		if err != nil {
			interaction.RespondError(b, i, lang, "arch.error")
			return
		}
		embed = components.Embed(
			i18n.T("arch.sold_title", lang),
			i18n.T("arch.sold_desc", lang, map[string]any{"item": items.DisplayName(res.ItemName), "coins": bal}),
			0xF1C40F,
		)

	default:
		if err := c.svc.AwardResult(userID, res); err != nil {
			interaction.RespondError(b, i, lang, "arch.error")
			return
		}
		xpStr := ""
		if res.XP > 0 {
			xpStr = i18n.T("arch.xp_gained", lang, map[string]any{"xp": res.XP})
		}
		embed = components.Embed(
			i18n.T("arch.keep_title", lang),
			i18n.T("arch.keep_desc", lang, map[string]any{"item": items.DisplayName(res.ItemName), "xp": xpStr}),
			0x00FF00,
		)
	}

	unlocks, uerr := achievement.CheckAndUnlock(b.DB, userID)
	if uerr == nil && len(unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, unlocks)
	}

	delete(digSessions, userID)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("arch.back_menu", lang), components.Encode("arch", "menu"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) showDigEmbed(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, state *archsvc.GameState) {
	depthPct := float64(state.Depth) / float64(state.MaxDepth)
	if depthPct < 0 {
		depthPct = 0
	}
	blocksFull := int((1.0 - depthPct) * 5)
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
		"site":  i18n.T(state.Site.NameID, lang),
		"layer": layerEmoji + " " + layerName,
	})

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
		components.Field(i18n.T("arch.depth_label", lang), depthBar+" "+itoa(state.Depth)+"/"+itoa(state.MaxDepth)+"cm", false),
		components.Field(i18n.T("arch.integrity_label", lang), intBar+" "+itoa(state.Integrity)+"%", false),
		components.Field(i18n.T("arch.actions_label", lang), "**"+itoa(state.Actions)+"**", true),
		components.Field(i18n.T("arch.layer_label", lang), layerEmoji+" "+layerName, true),
	}

	if state.CursedDebuff {
		embed.Fields = append(embed.Fields, components.Field("⚠️", i18n.T("arch.cursed_debuff", lang), false))
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("arch.dynamite_btn", lang), components.Encode("arch", "action", "dynamite"), discordgo.DangerButton),
			components.Button(i18n.T("arch.hammer_btn", lang), components.Encode("arch", "action", "hammer"), discordgo.PrimaryButton),
			components.Button(i18n.T("arch.brush_btn", lang), components.Encode("arch", "action", "brush"), discordgo.SuccessButton),
			components.Button(i18n.T("arch.scan_btn", lang), components.Encode("arch", "action", "scan"), discordgo.SecondaryButton),
		),
	}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) showEventEmbed(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, evt *archsvc.DigEvent, state *archsvc.GameState) {
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
		customID := components.Encode("arch", "event", ch.Value, itoa(int(evt.Type)))
		btns = append(btns, components.Button(i18n.T(ch.LabelID, lang), customID, style))
	}

	comps := []discordgo.MessageComponent{components.ActionRow(btns...)}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) showResultEmbed(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, res *archsvc.DigResult) {
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
		btns = append(btns, components.Button(i18n.T("arch.keep_btn", lang), components.Encode("arch", "post", "keep"), discordgo.SuccessButton))
		btns = append(btns, components.Button(i18n.T("arch.sell_btn", lang), components.Encode("arch", "post", "sell"), discordgo.PrimaryButton))
	} else {
		c.svc.AwardResult(userID(interaction.UserID(i)), res)
	}

	comps := []discordgo.MessageComponent{}
	if len(btns) > 0 {
		comps = append(comps, components.ActionRow(btns...))
	}
	comps = append(comps, components.ActionRow(
		components.Button(i18n.T("arch.back_menu", lang), components.Encode("arch", "menu"), discordgo.SecondaryButton),
	))

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func outcomeDesc(res *archsvc.DigResult, lang string) string {
	itemName := items.DisplayName(res.ItemName)
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
		return i18n.T("arch.success_msg", lang, map[string]any{"item": itemName + qtyStr, "quality": i18n.T("quality_"+res.Quality, lang), "integrity": res.Integrity}) + "\n" + received
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
		_, _ = b.Session.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: i18n.T("arch.reanimate_cmd_no_fossils", lang, map[string]any{"count": 5, "item": items.DisplayName(pool.ItemName)}),
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

	desc := ""
	for rarity, pool := range archsvc.ReanimatePools {
		count := c.svc.GetFossilCount(userID, pool.ItemName)
		rarityName := i18n.T("quality_"+rarity, lang)
		desc += i18n.T("arch.reanimate_list_line", lang, map[string]any{
			"rarity": rarityName,
			"count":  count,
			"item":   items.DisplayName(pool.ItemName),
		}) + "\n"
	}

	if desc == "" {
		desc = i18n.T("arch.reanimate_list_empty", lang)
	}

	embed := components.Embed(
		i18n.T("arch.reanimate_list_title", lang),
		desc,
		0x9B59B6,
	)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{Embeds: []*discordgo.MessageEmbed{embed}})
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
