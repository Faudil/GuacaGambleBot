package fishing

import (
	"math/rand"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	fishingsvc "guacagamblebot/internal/service/fishing"
	invsvc "guacagamblebot/internal/service/inventory"
	loresvc "guacagamblebot/internal/service/lore"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
)

type fishSession struct {
	baitTier fishingsvc.BaitTier
	state    *fishingsvc.FishFightState
}

var sessions = map[int64]*fishSession{}

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *fishingsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	def := universe.Get(cfg.Universe)
	if def == nil {
		def = universe.Get("hoakhaven")
	}
	loreSvc := loresvc.New(s, cfg, def)
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)

	c := &Cog{store: s, cfg: cfg, svc: fishingsvc.New(s, cfg, loreSvc, npcSvc)}
	r.Slash("fish", "Fishing minigame", c.onSlashMenu)
	r.Slash("f", "Fishing minigame", c.onSlashMenu)
	r.Prefix("fish", c.onPrefixMenu)
	r.Prefix("f", c.onPrefixMenu)
	r.Component("fish", "menu", c.onMenu)
	r.Component("fish", "bait_common", c.onBaitSelect)
	r.Component("fish", "bait_rare", c.onBaitSelect)
	r.Component("fish", "bait_legendary", c.onBaitSelect)
	r.Component("fish", "spot_pond", c.onSpotSelect)
	r.Component("fish", "spot_river", c.onSpotSelect)
	r.Component("fish", "spot_ocean", c.onSpotSelect)
	r.Component("fish", "spot_lava", c.onSpotSelect)
	r.Component("fish", "reel", c.onFightAction)
	r.Component("fish", "pull", c.onFightAction)
	r.Component("fish", "rest", c.onFightAction)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.baitMenu(lang, interaction.ToInt64(interaction.UserID(i)))
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	embed, comps := c.baitMenu(lang, interaction.ToInt64(m.Author.ID))
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) onMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(interaction.UserID(i))
	delete(sessions, userID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.baitMenu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) baitMenu(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	freeOk, _ := c.svc.CanFreeCast(userID)
	hasCommon, _ := c.svc.HasBait(userID, fishingsvc.BaitCommon)
	hasRare, _ := c.svc.HasBait(userID, fishingsvc.BaitRare)
	hasLegendary, _ := c.svc.HasBait(userID, fishingsvc.BaitLegendary)

	var freeStatus string
	if freeOk {
		freeStatus = i18n.T("fishing.free_available", lang)
	} else {
		freeStatus = i18n.T("fishing.free_used", lang)
	}

	embed := components.Embed(
		i18n.T("fishing.session_title", lang),
		i18n.T("fishing.bait_desc", lang),
		0x008080,
	)

	fields := []*discordgo.MessageEmbedField{
		components.Field(i18n.T("fishing.common_bait_name", lang), i18n.T("fishing.common_bait_desc", lang)+"\n"+freeStatus, true),
	}
	if hasRare {
		fields = append(fields, components.Field(i18n.T("fishing.rare_bait_name", lang), i18n.T("fishing.rare_bait_desc", lang), true))
	}
	if hasLegendary {
		fields = append(fields, components.Field(i18n.T("fishing.legendary_bait_name", lang), i18n.T("fishing.legendary_bait_desc", lang), true))
	}
	embed.Fields = fields

	var comps []discordgo.MessageComponent
	if freeOk || hasCommon {
		comps = append(comps, components.ActionRow(
			components.Button(i18n.T("fishing.common_bait_btn", lang), components.Encode("fish", "bait_common"), discordgo.SuccessButton),
		))
	}
	if hasRare {
		comps = append(comps, components.ActionRow(
			components.Button(i18n.T("fishing.rare_bait_btn", lang), components.Encode("fish", "bait_rare"), discordgo.PrimaryButton),
		))
	}
	if hasLegendary {
		comps = append(comps, components.ActionRow(
			components.Button(i18n.T("fishing.legendary_bait_btn", lang), components.Encode("fish", "bait_legendary"), discordgo.DangerButton),
		))
	}
	return embed, comps
}

func (c *Cog) onBaitSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, action, _ := components.Decode(cid)

	var tier fishingsvc.BaitTier
	switch action {
	case "bait_common":
		tier = fishingsvc.BaitCommon
	case "bait_rare":
		tier = fishingsvc.BaitRare
	case "bait_legendary":
		tier = fishingsvc.BaitLegendary
	default:
		return
	}

	if tier == fishingsvc.BaitCommon {
		free, _ := c.svc.CanFreeCast(userID)
		if !free {
			has, _ := c.svc.HasBait(userID, tier)
			if !has {
				interaction.RespondError(b, i, lang, "fishing.no_bait")
				return
			}
		}
	} else {
		has, _ := c.svc.HasBait(userID, tier)
		if !has {
			interaction.RespondError(b, i, lang, "fishing.no_bait")
			return
		}
	}

	sessions[userID] = &fishSession{baitTier: tier}
	embed, comps := c.spotMenu(lang, userID, tier)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) spotMenu(lang string, userID int64, tier fishingsvc.BaitTier) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	lavaUnlocked, _ := c.svc.LavaUnlocked(userID)
	tierName := baitTierName(lang, tier)

	embed := components.Embed(
		i18n.T("fishing.spot_title", lang),
		i18n.T("fishing.spot_desc", lang, map[string]any{"bait": tierName}),
		0x008080,
	)

	fields := []*discordgo.MessageEmbedField{
		components.Field(i18n.T("fishing.pond_field_name", lang), i18n.T("fishing.pond_field_value", lang), true),
		components.Field(i18n.T("fishing.river_field_name", lang), i18n.T("fishing.river_field_value", lang), true),
		components.Field(i18n.T("fishing.ocean_field_name", lang), i18n.T("fishing.ocean_field_value", lang), true),
	}
	if lavaUnlocked {
		fields = append(fields, components.Field(i18n.T("fishing.lava_field_name", lang), i18n.T("fishing.lava_field_value", lang), true))
	}
	embed.Fields = fields

	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("fishing.pond_label", lang), components.Encode("fish", "spot_pond"), discordgo.SuccessButton),
			components.Button(i18n.T("fishing.river_label", lang), components.Encode("fish", "spot_river"), discordgo.PrimaryButton),
			components.Button(i18n.T("fishing.ocean_label", lang), components.Encode("fish", "spot_ocean"), discordgo.DangerButton),
		),
	}
	if lavaUnlocked {
		comps = append(comps, components.ActionRow(
			components.Button(i18n.T("fishing.lava_label", lang), components.Encode("fish", "spot_lava"), discordgo.SecondaryButton),
		))
	}
	comps = append(comps, components.ActionRow(
		components.Button(i18n.T("fishing.back", lang), components.Encode("fish", "menu"), discordgo.SecondaryButton),
	))
	return embed, comps
}

func (c *Cog) onSpotSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, action, _ := components.Decode(cid)

	sess, ok := sessions[userID]
	if !ok {
		interaction.RespondError(b, i, lang, "fishing.error")
		return
	}

	biome := action[5:]

	if biome == "lava" {
		unlocked, _ := c.svc.LavaUnlocked(userID)
		if !unlocked {
			_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: i18n.T("fishing.lava_locked", lang),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
	}

	cd, err := c.svc.CheckCooldown(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "fishing.error")
		return
	}
	if cd > 0 {
		secs := int(cd.Seconds())
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: i18n.T("fishing.cooldown", lang, map[string]any{"time": itoa(secs)}),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	ok, _, err = c.store.CheckGameLimit(userID, "fish", 10)
	if err != nil {
		interaction.RespondError(b, i, lang, "fishing.error")
		return
	}
	if !ok {
		interaction.RespondError(b, i, lang, "fishing.limit_reached")
		return
	}

	if sess.baitTier == fishingsvc.BaitCommon {
		free, _ := c.svc.CanFreeCast(userID)
		if free {
			if err := c.svc.UseFreeCast(userID); err != nil {
				interaction.RespondError(b, i, lang, "fishing.error")
				return
			}
		} else {
			if err := c.svc.ConsumeBait(userID, sess.baitTier); err != nil {
				interaction.RespondError(b, i, lang, "fishing.error")
				return
			}
		}
	} else {
		if err := c.svc.ConsumeBait(userID, sess.baitTier); err != nil {
			interaction.RespondError(b, i, lang, "fishing.error")
			return
		}
	}

	bottleKey := c.svc.RollMessageBottle()
	if bottleKey != "" {
		res, err := c.svc.ResolveBottle(userID)
		if err != nil {
			interaction.RespondError(b, i, lang, "fishing.error")
			return
		}
		embed := c.bottleEmbed(bottleKey, res, lang)
		delete(sessions, userID)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))
		return
	}

	state := c.svc.GenerateFish(biome, sess.baitTier)
	sess.state = state

	embed, comps := c.fightEmbed(state, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) fightEmbed(state *fishingsvc.FishFightState, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	desc := fightFlavor(state, lang)

	tensionPct := state.Tension
	if tensionPct > 100 {
		tensionPct = 100
	}
	tensionBar := barString(tensionPct, 100, "▰", "▰", "▱")
	tensionLabel := i18n.T("fishing.tension_bar", lang, map[string]any{"bar": tensionBar, "pct": itoa(tensionPct)})

	staminaMax := state.Species.Stamina
	if state.Mutated {
		staminaMax = state.Species.Stamina
	}
	staminaCurrent := state.Stamina
	if staminaCurrent > staminaMax {
		staminaCurrent = staminaMax
	}
	staminaPct := 0
	if staminaMax > 0 {
		staminaPct = staminaCurrent * 100 / staminaMax
	}
	staminaBar := barString(staminaPct, 100, "🟩", "🟩", "⬜")

	var distStr string
	switch state.Distance {
	case 0:
		distStr = i18n.T("fishing.distance_far", lang)
	case 1:
		distStr = i18n.T("fishing.distance_close", lang)
	case 2:
		distStr = i18n.T("fishing.distance_caught", lang)
	}

	color := 0x008080
	if state.Golden {
		color = 0xFFD700
	} else if state.Mutated {
		color = 0x00FF00
	} else if state.Species.Secret == "ghost_carp" {
		color = 0xC0C0C0
	} else if state.Species.Secret == "cosmic_jellyfish" {
		color = 0x9B59B6
	}

	embed := components.Embed(
		i18n.T("fishing.fight_title", lang),
		desc,
		color,
	)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("fishing.tension_label", lang), tensionLabel, false),
		components.Field(i18n.T("fishing.stamina_label", lang), staminaBar, false),
		components.Field(i18n.T("fishing.distance_label", lang), distStr, false),
	}

	if state.LuckyBreak {
		embed.Fields = append(embed.Fields, components.Field("", i18n.T("fishing.lucky_break_active", lang), false))
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("fishing.reel_btn", lang), components.Encode("fish", "reel"), discordgo.PrimaryButton),
			components.Button(i18n.T("fishing.pull_btn", lang), components.Encode("fish", "pull"), discordgo.DangerButton),
			components.Button(i18n.T("fishing.rest_btn", lang), components.Encode("fish", "rest"), discordgo.SuccessButton),
		),
	}
	return embed, comps
}

func (c *Cog) onFightAction(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, action, _ := components.Decode(cid)

	sess, ok := sessions[userID]
	if !ok || sess.state == nil || sess.state.Escaped || sess.state.Stamina <= 0 {
		if !ok {
			embed, comps := c.baitMenu(lang, userID)
			_ = b.Session.InteractionRespond(i.Interaction,
				components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
		} else {
			delete(sessions, userID)
			embed, comps := c.baitMenu(lang, userID)
			_ = b.Session.InteractionRespond(i.Interaction,
				components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
		}
		return
	}

	result := c.svc.ApplyAction(sess.state, action)

	if result.Caught {
		res, err := c.svc.ResolveCatch(userID, sess.state)
		delete(sessions, userID)
		if err != nil {
			interaction.RespondError(b, i, lang, "fishing.error")
			return
		}
		embed := c.catchEmbed(res, lang)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))
		return
	}

	if result.Escaped {
		res, err := c.svc.ResolveEscape(userID)
		delete(sessions, userID)
		if err != nil {
			interaction.RespondError(b, i, lang, "fishing.error")
			return
		}
		embed := c.escapeEmbed(res, lang)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))
		return
	}

	if result.LuckyBreak {
		embed, comps := c.fightEmbed(sess.state, lang)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
		return
	}

	embed, comps := c.fightEmbed(sess.state, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) catchEmbed(res *fishingsvc.FightResolve, lang string) *discordgo.MessageEmbed {
	var name string
	if res.Golden {
		name = "✦ " + i18n.T("fishing.golden_prefix", lang) + " " + res.ItemName
	} else if res.Mutated {
		name = "🧪 " + i18n.T("fishing.mutated_prefix", lang) + " " + res.ItemName
	} else if res.Secret == "ghost_carp" {
		name = "👻 " + res.ItemName
	} else if res.Secret == "cosmic_jellyfish" {
		name = "🌌 " + res.ItemName
	} else {
		name = res.ItemName
	}

	milestone := ""
	switch {
	case res.Weight >= 40000:
		milestone = "\n" + i18n.T("fishing.milestone_epic", lang)
	case res.Weight >= 10000:
		milestone = "\n" + i18n.T("fishing.milestone_legendary", lang)
	case res.Weight >= 2000:
		milestone = "\n" + i18n.T("fishing.milestone_huge", lang)
	case res.Weight >= 500:
		milestone = "\n" + i18n.T("fishing.milestone_large", lang)
	case res.Weight >= 100:
		milestone = "\n" + i18n.T("fishing.milestone_decent", lang)
	}

	loreText := ""
	if res.LoreID != "" {
		loreText = "\n\n📜 " + i18n.T("fishing.lore_found", lang)
	}

	color := 0xFFD700
	if res.Mutated {
		color = 0x00FF00
	} else if res.Secret != "" {
		color = 0x9B59B6
	}

	embed := components.Embed(
		i18n.T("fishing.caught_title", lang),
		i18n.T("fishing.caught_desc", lang, map[string]any{
			"name":   name,
			"weight": itoa(res.Weight),
			"size":   itoa(res.Size),
			"xp":     itoa(res.XP),
		})+milestone+loreText,
		color,
	)
	return embed
}

func (c *Cog) escapeEmbed(res *fishingsvc.FightResolve, lang string) *discordgo.MessageEmbed {
	return components.Embed(
		i18n.T("fishing.escaped_title", lang),
		i18n.T("fishing.escaped_desc", lang, map[string]any{"xp": itoa(res.XP)}),
		0xFF0000,
	)
}

func (c *Cog) bottleEmbed(bottleKey string, res *fishingsvc.FightResolve, lang string) *discordgo.MessageEmbed {
	desc := i18n.T(bottleKey, lang)
	if res.LoreID != "" {
		desc += "\n\n📜 " + i18n.T("fishing.lore_found", lang)
	}
	desc += "\n\n" + i18n.T("fishing.bottle_xp", lang, map[string]any{"xp": itoa(res.XP)})
	return components.Embed(
		i18n.T("fishing.bottle_title", lang),
		desc,
		0x8B4513,
	)
}

func baitTierName(lang string, tier fishingsvc.BaitTier) string {
	switch tier {
	case fishingsvc.BaitCommon:
		return i18n.T("fishing.common_bait_name", lang)
	case fishingsvc.BaitRare:
		return i18n.T("fishing.rare_bait_name", lang)
	case fishingsvc.BaitLegendary:
		return i18n.T("fishing.legendary_bait_name", lang)
	}
	return ""
}

func fightFlavor(state *fishingsvc.FishFightState, lang string) string {
	if state.Species.Secret == "ghost_carp" {
		return i18n.T("fishing.flavor_ghost_carp", lang)
	}
	if state.Species.Secret == "cosmic_jellyfish" {
		return i18n.T("fishing.flavor_cosmic_jellyfish", lang)
	}

	hour := time.Now().UTC().Hour()
	var timeFlavor string
	if hour < 6 || hour >= 19 {
		timeFlavor = i18n.T("fishing.flavor_night", lang)
	} else if hour >= 6 && hour < 18 {
		timeFlavor = i18n.T("fishing.flavor_day", lang)
	} else {
		timeFlavor = i18n.T("fishing.flavor_dusk", lang)
	}

	if state.Mutated {
		return i18n.T("fishing.flavor_mutated", lang) + "\n" + timeFlavor
	}
	if state.Golden {
		return i18n.T("fishing.flavor_golden", lang) + "\n" + timeFlavor
	}

	pool := flavorTexts(state.Species.Strength, lang)
	base := pool[rand.Intn(len(pool))]
	return base + "\n" + timeFlavor
}

func flavorTexts(strength int, lang string) []string {
	if strength >= 8 {
		return []string{
			i18n.T("fishing.flavor_high_1", lang),
			i18n.T("fishing.flavor_high_2", lang),
			i18n.T("fishing.flavor_high_3", lang),
		}
	}
	if strength >= 5 {
		return []string{
			i18n.T("fishing.flavor_mid_1", lang),
			i18n.T("fishing.flavor_mid_2", lang),
			i18n.T("fishing.flavor_mid_3", lang),
		}
	}
	if strength >= 2 {
		return []string{
			i18n.T("fishing.flavor_low_1", lang),
			i18n.T("fishing.flavor_low_2", lang),
			i18n.T("fishing.flavor_low_3", lang),
		}
	}
	return []string{
		i18n.T("fishing.flavor_trash_1", lang),
		i18n.T("fishing.flavor_trash_2", lang),
		i18n.T("fishing.flavor_trash_3", lang),
	}
}

func barString(current, max int, fillChar, partialChar, emptyChar string) string {
	total := 10
	filled := current * total / max
	if filled > total {
		filled = total
	}
	if filled < 0 {
		filled = 0
	}
	var b strings.Builder
	for i := 0; i < filled; i++ {
		b.WriteString(fillChar)
	}
	for i := filled; i < total; i++ {
		b.WriteString(emptyChar)
	}
	return b.String()
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
