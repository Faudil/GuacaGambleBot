package farm

import (
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	farmsvc "guacagamblebot/internal/service/farm"
	"guacagamblebot/internal/items"
	invsvc "guacagamblebot/internal/service/inventory"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
)

var eventSessions = map[int64]*activeEvent{}

type activeEvent struct {
	Event   *farmsvc.Event
	ZoneKey string
	Blessed bool
}

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *farmsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	def := universe.Get(cfg.Universe)
	if def == nil {
		def = universe.Get("hoakhaven")
	}
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)
	c := &Cog{store: s, cfg: cfg, svc: farmsvc.New(s, cfg, npcSvc)}
	r.Slash("farm", "Farming minigame", c.onSlashMenu)
	r.Prefix("farm", c.onPrefixMenu)
	r.Prefix("fm", c.onPrefixMenu)
	r.Prefix("seedmaker", c.onSeedMakerPrefix)
	r.Prefix("semoir", c.onSeedMakerPrefix)
	r.Component("farm", "menu", c.onMenu)
	r.Component("farm", "zone", c.onZone)
	r.Component("farm", "plot", c.onPlot)
	r.Component("farm", "harvest", c.onHarvest)
	r.Component("farm", "seed_choose", c.onSeedChoose)
	r.Component("farm", "seedmaker", c.onSeedMaker)
	r.Component("farm", "seedmaker_choose", c.onSeedMakerChoose)
	r.Component("farm", "water", c.onWater)
	r.Component("farm", "fertilize", c.onFertilize)
	r.Component("farm", "inspect", c.onInspect)
	r.Component("farm", "inspect_main", c.onInspectFarm)
	r.Component("farm", "stats", c.onStats)
	r.Component("farm", "event", c.onEventChoice)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	delete(eventSessions, userID)
	embed, comps := c.menu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	delete(eventSessions, userID)
	embed, comps := c.menu(lang, userID)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) onMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(interaction.UserID(i))
	delete(eventSessions, userID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) menu(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	level := c.svc.GetFarmerLevel(userID)
	xp, xpNext := c.svc.GetFarmerXP(userID)
	active := c.svc.CountActivePlots(userID)
	maxTotal := c.svc.MaxTotalPlots(userID)
	cropNext, secs := c.svc.GetNextHarvest(userID)

	xpBar := progressBar(xp, xpNext, 12)

	timeFlavor := timeOfDayFlavor(lang)

	var nextHarvest string
	if secs > 0 {
		mins := secs / 60
		if mins < 1 {
			nextHarvest = i18n.T("farm.next_harvest_soon", lang)
		} else if mins < 60 {
			nextHarvest = i18n.T("farm.next_harvest_mins", lang, map[string]any{"item": displayCrop(cropNext), "min": mins})
		} else {
			hours := mins / 60
			nextHarvest = i18n.T("farm.next_harvest_hours", lang, map[string]any{"item": displayCrop(cropNext), "hours": hours})
		}
	} else {
		nextHarvest = i18n.T("farm.next_harvest_none", lang)
	}

	blessed := c.svc.HasBlessing(userID)
	blessStr := ""
	if blessed {
		blessStr = "\n" + i18n.T("farm.blessed_active", lang)
	}

	tipKey := dailyTip()

	desc := i18n.T("farm.main_desc", lang, map[string]any{
		"level":   level,
		"xpbar":   xpBar,
		"xp":      xp,
		"xpnext":  xpNext,
		"active":  active,
		"max":     maxTotal,
		"next":    nextHarvest,
		"blessed": blessStr,
		"time":    timeFlavor,
		"tip":     i18n.T(tipKey, lang),
	})

	embed := components.Embed(
		i18n.T("farm.main_title", lang),
		desc,
		0x006400,
	)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("farm.footer", lang),
	}

	zones := c.svc.GetAccessibleZones(userID)
	var btns []discordgo.MessageComponent
	for _, z := range zones {
		switch z {
		case "public":
			btns = append(btns, components.Button(i18n.T("farm.public_label", lang), components.EncodeOwner(userID, "farm", "zone", "public"), discordgo.SecondaryButton))
		case "veggie":
			btns = append(btns, components.Button(i18n.T("farm.veggie_label", lang), components.EncodeOwner(userID, "farm", "zone", "veggie"), discordgo.PrimaryButton))
		case "greenhouse":
			btns = append(btns, components.Button(i18n.T("farm.greenhouse_label", lang), components.EncodeOwner(userID, "farm", "zone", "greenhouse"), discordgo.SuccessButton))
		case "orchard":
			btns = append(btns, components.Button(i18n.T("farm.orchard_label", lang), components.EncodeOwner(userID, "farm", "zone", "orchard"), discordgo.DangerButton))
		}
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(btns...),
		components.ActionRow(
			components.Button(i18n.T("farm.inspect_main_btn", lang), components.EncodeOwner(userID, "farm", "inspect_main"), discordgo.SecondaryButton),
			components.Button(i18n.T("farm.seedmaker_btn", lang), components.EncodeOwner(userID, "farm", "seedmaker"), discordgo.SecondaryButton),
			components.Button(i18n.T("farm.stats_btn", lang), components.EncodeOwner(userID, "farm", "stats"), discordgo.SecondaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onZone(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	zoneKey := "public"
	if len(rest) > 0 {
		zoneKey = rest[0]
	}

	if !c.svc.HasZoneAccess(userID, zoneKey) {
		deedMap := map[string]string{
			"veggie":     "garden_plot",
			"greenhouse": "tropical_greenhouse",
			"orchard":    "enchanted_orchard",
		}
		deed := deedMap[zoneKey]
		embed := components.Embed(
			i18n.T("farm.no_land_title", lang),
			i18n.T("farm.no_land_desc", lang, map[string]any{"deed": items.DisplayName(deed)}),
			0xE74C3C,
		)
		back := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("farm.back", lang), components.EncodeOwner(userID, "farm", "menu"), discordgo.SecondaryButton),
			),
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, back))
		return
	}

	ok, _, err := c.store.CheckGameLimit(userID, "farm", 20)
	if err != nil || !ok {
		interaction.RespondError(b, i, lang, "farm.limit_reached")
		return
	}

	plots, err := c.svc.GetPlots(userID, zoneKey)
	if err != nil {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}

	evt := c.svc.RollEvent(userID, zoneKey, plots)
	if evt != nil {
		delete(eventSessions, userID)
		eventSessions[userID] = &activeEvent{
			Event:   evt,
			ZoneKey: zoneKey,
		}
		c.showEvent(b, i, lang, userID, evt)
		return
	}

	c.showZone(b, i, lang, zoneKey, userID, plots)
}

func (c *Cog) showZone(b *interaction.Bot, i *discordgo.InteractionCreate, lang, zoneKey string, userID int64, plots []farmsvc.PlotInfo) {
	zoneName := zoneDisplayName(zoneKey, lang)
	zoneFlavor := zoneFlavorText(zoneKey, lang)

	embed := components.Embed(zoneName, zoneFlavor, zoneColor(zoneKey))

	blessed := c.svc.HasBlessing(userID) && c.svc.GetBlessingZone(userID) == zoneKey
	if blessed {
		embed.Description += "\n\n" + i18n.T("farm.blessed_zone_active", lang)
	}

	for _, p := range plots {
		fieldVal := c.plotField(&p, lang)
		plotLabel := i18n.T("farm.plot_number", lang, map[string]any{"n": p.PlotIndex + 1})
		embed.Fields = append(embed.Fields, components.Field(plotLabel, fieldVal, false))
	}

	var comps []discordgo.MessageComponent
	var row []discordgo.MessageComponent

	for _, p := range plots {
		if p.ItemName == "" {
			row = append(row, components.Button(
				i18n.T("farm.plot_empty_btn", lang, map[string]any{"n": p.PlotIndex + 1}),
				components.EncodeOwner(userID, "farm", "plot", zoneKey, strconv.Itoa(p.PlotIndex)),
				discordgo.SecondaryButton,
			))
		} else if p.Ready {
			row = append(row, components.Button(
				i18n.T("farm.plot_ready_btn", lang, map[string]any{"item": plotDisplayName(&p)}),
				components.EncodeOwner(userID, "farm", "harvest", zoneKey, strconv.Itoa(p.PlotIndex)),
				discordgo.SuccessButton,
			))
		} else {
			btnLabel := plotDisplayName(&p) + " " + strconv.Itoa(p.Progress) + "%"
			row = append(row, components.Button(
				btnLabel,
				components.EncodeOwner(userID, "farm", "inspect", zoneKey, strconv.Itoa(p.PlotIndex)),
				discordgo.PrimaryButton,
			))
		}
	}

	comps = append(comps, components.ActionRow(row...))
	comps = append(comps, components.ActionRow(
		components.Button(i18n.T("farm.back", lang), components.EncodeOwner(userID, "farm", "menu"), discordgo.SecondaryButton),
	))

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) showEvent(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, userID int64, evt *farmsvc.Event) {
	var desc string
	if evt.Type == farmsvc.EventMerchant {
		merchantName, _ := evt.Data["merchant"].(string)
		desc = i18n.T(evt.Description, lang, map[string]any{"merchant": i18n.T(merchantName, lang)})
	} else if evt.Type == farmsvc.EventPest {
		plotIdx, _ := evt.Data["plot"].(int)
		desc = i18n.T(evt.Description, lang, map[string]any{"plot": plotIdx + 1})
	} else {
		desc = i18n.T(evt.Description, lang)
	}

	embed := components.Embed(
		i18n.T(evt.Title, lang),
		desc,
		eventColor(evt.Type),
	)

	var btns []discordgo.MessageComponent
	for _, ch := range evt.Choices {
		style := discordgo.ButtonStyle(ch.Style)
		if style == 0 {
			style = discordgo.PrimaryButton
		}
		var label string
		if strings.Contains(ch.CustomID, "buy") {
			parts := strings.Split(ch.CustomID, "::")
			if len(parts) >= 6 {
				price, _ := strconv.Atoi(parts[5])
				label = i18n.T(ch.Label, lang, map[string]any{"price": price})
			} else {
				label = i18n.T(ch.Label, lang)
			}
		} else {
			label = i18n.T(ch.Label, lang)
		}
		customID := components.EncodeOwner(userID, strings.Split(ch.CustomID, "::")...)
		btns = append(btns, components.Button(label, customID, style))
	}

	comps := []discordgo.MessageComponent{components.ActionRow(btns...)}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onEventChoice(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	parts := strings.Split(cid, "::")

	session, ok := eventSessions[userID]
	if !ok || session.Event == nil {
		c.onMenu(b, i)
		return
	}

	evt := session.Event
	var choice string
	if len(parts) >= 4 {
		choice = parts[3]
	} else {
		choice = "leave"
	}

	result := c.svc.ResolveEvent(userID, evt, choice)

	desc := i18n.T(result.Description, lang)
	if result.CoinChange > 0 {
		desc += "\n\n" + i18n.T("farm.coins_gained", lang, map[string]any{"coins": result.CoinChange})
	} else if result.CoinChange < 0 {
		desc += "\n\n" + i18n.T("farm.coins_spent", lang, map[string]any{"coins": -result.CoinChange})
	}
	if result.ItemGiven != "" {
		desc += "\n\n" + i18n.T("farm.item_received", lang, map[string]any{"item": items.DisplayName(result.ItemGiven), "qty": result.ItemQty})
	}

	embed := components.Embed(
		i18n.T(result.Title, lang),
		desc,
		0x006400,
	)

	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("farm.back", lang), components.EncodeOwner(userID, "farm", "menu"), discordgo.SecondaryButton),
		),
	}

	delete(eventSessions, userID)

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onPlot(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 2 {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}
	zoneKey := rest[0]
	plotIdx := rest[1]

	var options []discordgo.SelectMenuOption
	for _, seed := range farmsvc.RegularSeedNames {
		if !c.svc.HasItem(userID, seed) {
			continue
		}
		var growTime int
		for _, s := range farmsvc.Seeds {
			if s.Name == seed {
				growTime = s.GrowTimeSec
				break
			}
		}
		desc := i18n.T("farm.seed_option_desc", lang, map[string]any{"time": growTime / 60, "price": seedPrice(seed)})
		options = append(options, discordgo.SelectMenuOption{
			Label:       items.DisplayName(seed),
			Value:       seed,
			Description: desc,
			Emoji:       &discordgo.ComponentEmoji{Name: "🌱"},
		})
	}

	hasMysterious := c.svc.HasItem(userID, "mysterious_seed")
	if hasMysterious {
		options = append(options, discordgo.SelectMenuOption{
			Label:       i18n.T("farm.plant_mysterious_btn", lang),
			Value:       "mysterious_seed",
			Description: i18n.T("farm.mysterious_seed_desc_option", lang),
			Emoji:       &discordgo.ComponentEmoji{Name: "🔮"},
		})
	}

	if len(options) == 0 {
		interaction.RespondError(b, i, lang, "farm.no_seeds")
		return
	}

	menu := discordgo.SelectMenu{
		CustomID:    components.EncodeOwner(userID, "farm", "seed_choose", zoneKey, plotIdx),
		Placeholder: i18n.T("farm.choose_seed_placeholder", lang),
		Options:     options,
	}

	embed := components.Embed(
		i18n.T("farm.choose_seed", lang),
		"",
		0x00FF00,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(menu),
		components.ActionRow(
			components.Button(i18n.T("farm.back", lang), components.EncodeOwner(userID, "farm", "zone", zoneKey), discordgo.SecondaryButton),
		),
	}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onSeedChoose(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 2 {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}
	zoneKey := rest[0]
	plotIdx, _ := strconv.Atoi(rest[1])

	data := i.MessageComponentData()
	if len(data.Values) < 1 {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}
	seedName := data.Values[0]

	if seedName != "mysterious_seed" && !c.svc.HasItem(userID, seedName) {
		interaction.RespondError(b, i, lang, "farm.no_seeds")
		return
	}

	var growTime int
	if seedName == "mysterious_seed" {
		growTime = 1800
	} else {
		for _, s := range farmsvc.Seeds {
			if s.Name == seedName {
				growTime = s.GrowTimeSec
				break
			}
		}
	}

	err := c.svc.Plant(userID, zoneKey, plotIdx, seedName, growTime)
	if err != nil {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}

	if seedName == "mysterious_seed" {
		embed := components.Embed(
			i18n.T("farm.planted_mysterious_title", lang),
			i18n.T("farm.planted_mysterious_desc", lang),
			0x9B59B6,
		)
		back := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("farm.back", lang), components.EncodeOwner(userID, "farm", "menu"), discordgo.SecondaryButton),
			),
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, back))
		return
	}

	mins := growTime / 60
	embed := components.Embed(
		i18n.T("farm.planted_title", lang),
		i18n.T("farm.planted_desc", lang, map[string]any{
			"item": items.DisplayName(seedName),
			"time": mins,
		}),
		0x00FF00,
	)
	back := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("farm.back", lang), components.EncodeOwner(userID, "farm", "menu"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, back))
}

func (c *Cog) onSeedMaker(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	var options []discordgo.SelectMenuOption
	for _, crop := range farmsvc.Crops {
		qty := c.svc.GetItemQuantity(userID, crop.Name)
		if qty < 1 {
			continue
		}
		options = append(options, discordgo.SelectMenuOption{
			Label: items.DisplayName(crop.Name),
			Value: crop.Name,
			Description: i18n.T("farm.seedmaker_option_desc", lang, map[string]any{
				"qty": qty,
			}),
			Emoji: &discordgo.ComponentEmoji{Name: "🌱"},
		})
	}

	if len(options) == 0 {
		interaction.RespondError(b, i, lang, "farm.seedmaker_no_crops")
		return
	}

	menu := discordgo.SelectMenu{
		CustomID:    components.EncodeOwner(userID, "farm", "seedmaker_choose"),
		Placeholder: i18n.T("farm.seedmaker_choose_placeholder", lang),
		Options:     options,
	}
	comps := []discordgo.MessageComponent{
		components.ActionRow(menu),
		components.ActionRow(
			components.Button(i18n.T("farm.back", lang), components.EncodeOwner(userID, "farm", "menu"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
			components.Embed(i18n.T("farm.seedmaker_title", lang), i18n.T("farm.seedmaker_desc", lang), 0x006400), comps))
}

func (c *Cog) onSeedMakerChoose(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	data := i.MessageComponentData()
	if len(data.Values) < 1 {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}
	cropName := data.Values[0]

	if !c.svc.HasItem(userID, cropName) {
		interaction.RespondError(b, i, lang, "farm.seedmaker_no_crops")
		return
	}

	seedID, qty, err := c.svc.ConvertToSeeds(userID, cropName)
	if err != nil {
		interaction.RespondError(b, i, lang, "farm.seedmaker_no_crops")
		return
	}

	embed := components.Embed(
		i18n.T("farm.seedmaker_result_title", lang),
		i18n.T("farm.seedmaker_result_desc", lang, map[string]any{
			"crop": items.DisplayName(cropName),
			"qty":  qty,
			"seed": items.DisplayName(seedID),
		}),
		0x00FF00,
	)
	back := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("farm.back", lang), components.EncodeOwner(userID, "farm", "menu"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, back))
}

func (c *Cog) onSeedMakerPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)

	parts := strings.Fields(m.Content)
	if len(parts) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("farm.seedmaker_cmd_usage", lang))
		return
	}

	cropName := resolveCropName(parts[1])
	if cropName == "" {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("farm.seedmaker_invalid_crop", lang, map[string]any{"item": parts[1]}))
		return
	}

	qty := 1
	if len(parts) >= 3 {
		qty, _ = strconv.Atoi(parts[2])
	}
	if qty < 1 {
		qty = 1
	}
	if qty > 10 {
		qty = 10
	}

	owned := c.svc.GetItemQuantity(userID, cropName)
	if owned < qty {
		qty = owned
	}
	if qty < 1 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("farm.seedmaker_no_crops", lang))
		return
	}

	totalSeeds := 0
	var seedID string
	for i := 0; i < qty; i++ {
		sid, n, err := c.svc.ConvertToSeeds(userID, cropName)
		if err != nil {
			break
		}
		seedID = sid
		totalSeeds += n
	}
	if totalSeeds < 1 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("farm.seedmaker_no_crops", lang))
		return
	}

	embed := components.Embed(
		i18n.T("farm.seedmaker_result_title", lang),
		i18n.T("farm.seedmaker_result_desc", lang, map[string]any{
			"crop": items.DisplayName(cropName),
			"qty":  totalSeeds,
			"seed": items.DisplayName(seedID),
		}),
		0x00FF00,
	)
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func resolveCropName(raw string) string {
	lower := strings.ToLower(raw)
	for _, crop := range farmsvc.Crops {
		if strings.EqualFold(crop.Name, lower) || strings.EqualFold(items.DisplayName(crop.Name), lower) {
			return crop.Name
		}
	}
	return ""
}

func (c *Cog) onHarvest(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 2 {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}
	zoneKey := rest[0]
	plotIdx, _ := strconv.Atoi(rest[1])

	var msg string
	blessed := c.svc.HasBlessing(userID)

	hasMysteriousSeed := c.svc.RollMysteriousSeed(userID)

	res, err := c.svc.Harvest(userID, zoneKey, plotIdx)
	if err != nil {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}

	if blessed && c.svc.GetBlessingZone(userID) == zoneKey {
		c.svc.ConsumeBlessing(userID)
		res.Quantity *= 2
		res.XP *= 2
		res.Value *= 2
		msg = i18n.T("farm.harvest_blessed", lang)
	}

	loot := i18n.T("farm.harvest_msg", lang, map[string]any{"qty": res.Quantity, "item": items.DisplayName(res.CropName)})
	if res.Mutated {
		flavorKey := c.svc.GetMutationFlavor(res.CropName)
		if flavorKey == "" {
			flavorKey = "farm.mutation_generic"
		}
		mutDesc := i18n.T(flavorKey, lang)
		loot += "\n" + mutDesc
	}

	valueStr := strconv.Itoa(res.Value)
	desc := i18n.T("farm.success_desc", lang, map[string]any{
		"loot":  loot,
		"value": valueStr,
		"xp":    res.XP,
	})
	if res.LeveledUp {
		desc += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": res.NewLevel})
	}
	if msg != "" {
		desc = msg + "\n" + desc
	}

	if c.svc.CheckGoldenCarrot(userID) && zoneKey == "veggie" {
		_ = c.svc.AddItem(userID, "golden_carrot", 1)
		desc += "\n\n" + i18n.T("farm.secret_golden_carrot", lang)
	}

	embed := components.Embed(
		i18n.T("farm.harvest_title", lang),
		desc,
		0x00FF00,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("farm.back", lang), components.EncodeOwner(userID, "farm", "menu"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))

	if n, ok := c.store.PopQuestNotification(userID); ok {
		interaction.SendQuestNotification(b, i, n, lang)
	}

	unlocks, uerr := achievement.CheckAndUnlock(b.DB, userID)
	if uerr == nil && len(unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, unlocks)
	}

	if hasMysteriousSeed {
		_ = c.svc.AddItem(userID, "mysterious_seed", 1)
		seedEmbed := components.Embed(
			i18n.T("farm.event_mysterious_seed_title", lang),
			i18n.T("farm.event_mysterious_seed_desc", lang),
			0x9B59B6,
		)
		_, _ = b.Session.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{seedEmbed},
			Flags:  discordgo.MessageFlagsEphemeral,
		})
	}

	if zoneKey == "public" && plotIdx == 1 && rand.Float64() < c.svc.GetScarecrowChance() {
		_ = c.svc.AddItem(userID, "scarecrow_charm", 1)
		scareEmbed := components.Embed(
			i18n.T("farm.scarecrow_title", lang),
			i18n.T("farm.scarecrow_desc", lang),
			0xFFD700,
		)
		_, _ = b.Session.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{scareEmbed},
			Flags:  discordgo.MessageFlagsEphemeral,
		})
	}
}

func (c *Cog) onWater(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 2 {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}
	zoneKey := rest[0]
	plotIdx, _ := strconv.Atoi(rest[1])

	err := c.svc.Water(userID, zoneKey, plotIdx)
	if err != nil {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}

	embed := components.Embed(
		i18n.T("farm.water_title", lang),
		i18n.T("farm.water_desc", lang),
		0x3498DB,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("farm.back", lang), components.EncodeOwner(userID, "farm", "menu"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onFertilize(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 2 {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}
	zoneKey := rest[0]
	plotIdx, _ := strconv.Atoi(rest[1])

	err := c.svc.Fertilize(userID, zoneKey, plotIdx)
	if err != nil {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}

	embed := components.Embed(
		i18n.T("farm.fertilize_title", lang),
		i18n.T("farm.fertilize_desc", lang),
		0xE67E22,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("farm.back", lang), components.EncodeOwner(userID, "farm", "menu"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onInspect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 2 {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}
	zoneKey := rest[0]
	plotIdx, _ := strconv.Atoi(rest[1])

	plots, err := c.svc.GetPlots(userID, zoneKey)
	if err != nil || plotIdx >= len(plots) {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}
	p := plots[plotIdx]
	if p.ItemName == "" {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}

	elapsed := time.Since(p.PlantTime).Seconds()
	remaining := p.GrowTime - int(elapsed)
	if remaining < 0 {
		remaining = 0
	}
	remMins := remaining / 60
	remSecs := remaining % 60

	var cropDesc string
	if p.Mysterious {
		cropDesc = i18n.T("farm.inspect_mysterious", lang)
	} else {
		item := items.Get(p.ItemName)
		if item != nil {
			cropDesc = item.Description
		}
	}

	progressPct := p.Progress
	if progressPct > 100 {
		progressPct = 100
	}

	desc := i18n.T("farm.inspect_desc", lang, map[string]any{
		"item":     plotDisplayName(&p),
		"cropdesc": cropDesc,
		"bar":      progressBar(progressPct, 100, 10),
		"pct":      progressPct,
		"mins":     remMins,
		"secs":     remSecs,
		"watered":  waterStr(p.Watered, lang),
	})

	embed := components.Embed(
		i18n.T("farm.inspect_title", lang, map[string]any{"n": p.PlotIndex + 1}),
		desc,
		0x008080,
	)

	var btns []discordgo.MessageComponent
	if !p.Ready {
		if !p.Watered {
			btns = append(btns, components.Button(
				i18n.T("farm.water_btn", lang),
				components.EncodeOwner(userID, "farm", "water", zoneKey, rest[1]),
				discordgo.PrimaryButton,
			))
		}
		if c.svc.HasItem(userID, "fertilizer") {
			btns = append(btns, components.Button(
				i18n.T("farm.fertilize_btn", lang),
				components.EncodeOwner(userID, "farm", "fertilize", zoneKey, rest[1]),
				discordgo.DangerButton,
			))
		}
	}
	btns = append(btns, components.Button(
		i18n.T("farm.back", lang),
		components.EncodeOwner(userID, "farm", "zone", zoneKey),
		discordgo.SecondaryButton,
	))

	comps := []discordgo.MessageComponent{components.ActionRow(btns...)}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onInspectFarm(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	zones := c.svc.GetAccessibleZones(userID)

	var desc string
	for _, z := range zones {
		plots, _ := c.svc.GetPlots(userID, z)
		zName := zoneDisplayName(z, lang)
		active := 0
		var details []string
		for _, p := range plots {
			if p.ItemName != "" {
				active++
				status := ""
				if p.Ready {
					status = "✅ " + i18n.T("farm.status_ready", lang)
				} else {
					status = "🌱 " + strconv.Itoa(p.Progress) + "%"
				}
				if p.Watered {
					status += " 💧"
				}
				if p.Mysterious {
					status += " 🔮"
				}
				details = append(details, plotDisplayName(&p)+" "+status)
			}
		}

		zoneLine := i18n.T("farm.inspect_zone_line", lang, map[string]any{
			"zone":   zName,
			"active": active,
			"total":  farmsvc.PlotsPerZone,
		})
		desc += zoneLine + "\n"
		for _, d := range details {
			desc += "  " + d + "\n"
		}
		desc += "\n"
	}

	if desc == "" {
		desc = i18n.T("farm.inspect_empty", lang)
	}

	embed := components.Embed(
		i18n.T("farm.inspect_main_title", lang),
		desc,
		0x008080,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("farm.back", lang), components.EncodeOwner(userID, "farm", "menu"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onStats(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	expertise := c.svc.GetExpertise(userID)

	var desc string
	if len(expertise) == 0 {
		desc = i18n.T("farm.stats_empty", lang)
	} else {
		for _, e := range expertise {
			title := ""
			if e.Title != "" {
				title = " " + i18n.T(e.Title, lang)
			}
			desc += i18n.T("farm.stats_line", lang, map[string]any{
				"item":      items.DisplayName(e.CropName),
				"harvested": e.Harvested,
				"title":     title,
			}) + "\n"
		}
	}

	embed := components.Embed(
		i18n.T("farm.stats_title", lang),
		desc,
		0x9B59B6,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("farm.back", lang), components.EncodeOwner(userID, "farm", "menu"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func plotDisplayName(p *farmsvc.PlotInfo) string {
	if p.Mysterious {
		return "🔮 " + i18n.T("farm.mysterious_label", "en")
	}
	if p.Ready {
		for _, s := range farmsvc.Seeds {
			if s.Name == p.ItemName {
				return items.DisplayName(s.Crop.Name)
			}
		}
	}
	return items.DisplayName(p.ItemName)
}

func displayCrop(cropName string) string {
	if cropName == "" {
		return "???"
	}
	return items.DisplayName(cropName)
}

func zoneDisplayName(key, lang string) string {
	switch key {
	case "public":
		return i18n.T("farm.public_name", lang)
	case "veggie":
		return i18n.T("farm.veggie_name", lang)
	case "greenhouse":
		return i18n.T("farm.greenhouse_name", lang)
	case "orchard":
		return i18n.T("farm.orchard_name", lang)
	}
	return key
}

func zoneColor(key string) int {
	switch key {
	case "public":
		return 0x2ECC71
	case "veggie":
		return 0x27AE60
	case "greenhouse":
		return 0x1ABC9C
	case "orchard":
		return 0x9B59B6
	}
	return 0x006400
}

func zoneFlavorText(key, lang string) string {
	switch key {
	case "public":
		return i18n.T("farm.zone_public_desc", lang)
	case "veggie":
		return i18n.T("farm.zone_veggie_desc", lang)
	case "greenhouse":
		return i18n.T("farm.zone_greenhouse_desc", lang)
	case "orchard":
		return i18n.T("farm.zone_orchard_desc", lang)
	}
	return ""
}

func eventColor(et farmsvc.EventType) int {
	switch et {
	case farmsvc.EventPest:
		return 0xE74C3C
	case farmsvc.EventMerchant:
		return 0xF39C12
	case farmsvc.EventBlessing:
		return 0x2ECC71
	case farmsvc.EventCropCircles:
		return 0x9B59B6
	}
	return 0x006400
}

func (c *Cog) plotField(p *farmsvc.PlotInfo, lang string) string {
	if p.ItemName == "" {
		return i18n.T("farm.plot_empty_desc", lang)
	}
	if p.Ready {
		w := ""
		if p.Watered {
			w = " 💧"
		}
		return i18n.T("farm.plot_ready_desc", lang, map[string]any{"item": plotDisplayName(p)}) + w
	}
	bar := progressBar(p.Progress, 100, 8)
	w := ""
	if p.Watered {
		w = " 💧"
	}
	extra := ""
	if p.Mysterious {
		extra += " 🔮"
	}
	return i18n.T("farm.plot_growing_desc", lang, map[string]any{"item": plotDisplayName(p), "bar": bar, "pct": p.Progress}) + w + extra
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
		return i18n.T("farm.time_night", lang)
	case hour < 12:
		return i18n.T("farm.time_morning", lang)
	case hour < 18:
		return i18n.T("farm.time_afternoon", lang)
	default:
		return i18n.T("farm.time_evening", lang)
	}
}

func waterStr(watered bool, lang string) string {
	if watered {
		return i18n.T("farm.watered_yes", lang)
	}
	return i18n.T("farm.watered_no", lang)
}

func seedPrice(seedName string) int {
	for _, s := range farmsvc.Seeds {
		if s.Name == seedName {
			return s.Price
		}
	}
	return 0
}

func dailyTip() string {
	tips := []string{
		"farm.tip_1",
		"farm.tip_2",
		"farm.tip_3",
		"farm.tip_4",
		"farm.tip_5",
		"farm.tip_6",
		"farm.tip_7",
		"farm.tip_8",
	}
	day := time.Now().UTC().YearDay()
	return tips[day%len(tips)]
}
