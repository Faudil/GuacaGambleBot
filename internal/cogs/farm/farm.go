package farm

import (
	"strconv"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	farmsvc "guacagamblebot/internal/service/farm"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *farmsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: farmsvc.New(s, cfg)}
	r.Slash("farm", "Farming minigame", c.onSlashMenu)
	r.Prefix("farm", c.onPrefixMenu)
	r.Component("farm", "menu", c.onMenu)
	r.Component("farm", "zone", c.onZone)
	r.Component("farm", "plot", c.onPlot)
	r.Component("farm", "harvest", c.onHarvest)
	r.Component("farm", "seed", c.onSeedPick)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	embed, comps := c.menu(lang)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) menu(lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := components.Embed(
		i18n.T("farm.map_title", lang),
		i18n.T("farm.map_desc", lang),
		0x006400,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("farm.public_label", lang), components.Encode("farm", "zone", "public"), discordgo.SecondaryButton),
			components.Button(i18n.T("farm.veggie_label", lang), components.Encode("farm", "zone", "veggie"), discordgo.PrimaryButton),
		),
		components.ActionRow(
			components.Button(i18n.T("farm.greenhouse_label", lang), components.Encode("farm", "zone", "greenhouse"), discordgo.SuccessButton),
			components.Button(i18n.T("farm.orchard_label", lang), components.Encode("farm", "zone", "orchard"), discordgo.DangerButton),
		),
	}
	return embed, comps
}

func (c *Cog) onMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
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

	plots, err := c.svc.GetPlots(userID, zoneKey)
	if err != nil {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}

	zoneName := zoneDisplayName(zoneKey, lang)
	embed := components.Embed(
		zoneName,
		i18n.T("farm.zone_welcome", lang),
		0xFFD700,
	)

	var comps []discordgo.MessageComponent
	for _, p := range plots {
		if p.ItemName == "" {
			comps = append(comps, components.Button(
				i18n.T("farm.plot_empty", lang, map[string]any{"idx": p.PlotIndex + 1}),
				components.Encode("farm", "plot", zoneKey, strconv.Itoa(p.PlotIndex)),
				discordgo.SecondaryButton,
			))
		} else if p.Ready {
			comps = append(comps, components.Button(
				i18n.T("farm.plot_ready", lang, map[string]any{"item": p.ItemName}),
				components.Encode("farm", "harvest", zoneKey, strconv.Itoa(p.PlotIndex)),
				discordgo.SuccessButton,
			))
		} else {
			comps = append(comps, components.Button(
				i18n.T("farm.plot_growing", lang, map[string]any{"item": p.ItemName, "pc": p.Progress}),
				components.Encode("farm", "none", ""),
				discordgo.PrimaryButton,
			))
		}
	}
	comps = append(comps,
		components.Button(i18n.T("farm.back", lang), components.Encode("farm", "menu"), discordgo.SecondaryButton),
	)

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed,
			[]discordgo.MessageComponent{components.ActionRow(comps...)}))
}

func (c *Cog) onPlot(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 2 {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}
	zoneKey := rest[0]
	plotIdx, _ := strconv.Atoi(rest[1])

	var seeds []string
	for _, s := range farmsvc.Seeds {
		seeds = append(seeds, s.Name)
	}

	embed := components.Embed(
		i18n.T("farm.choose_seed", lang),
		"",
		0x00FF00,
	)
	var btns []discordgo.MessageComponent
	for _, seed := range seeds {
		btns = append(btns, components.Button(seed,
			components.Encode("farm", "seed", zoneKey, rest[1], seed),
			discordgo.SecondaryButton))
	}
	btns = append(btns, components.Button(i18n.T("farm.back", lang),
		components.Encode("farm", "zone", zoneKey), discordgo.SecondaryButton))

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed,
			[]discordgo.MessageComponent{components.ActionRow(btns...)}))
	_ = plotIdx
}

func (c *Cog) onSeedPick(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 3 {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}
	zoneKey := rest[0]
	plotIdx, _ := strconv.Atoi(rest[1])
	seedName := rest[2]

	var growTime int
	for _, s := range farmsvc.Seeds {
		if s.Name == seedName {
			growTime = s.GrowTimeSec
			break
		}
	}

	err := c.svc.Plant(userID, zoneKey, plotIdx, seedName, growTime)
	if err != nil {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}

	embed := components.Embed(
		i18n.T("farm.planted_title", lang),
		i18n.T("farm.planted_desc", lang, map[string]any{
			"item": seedName,
			"time": growTime / 60,
		}),
		0x00FF00,
	)
	back := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("farm.back", lang), components.Encode("farm", "zone", zoneKey), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, back))
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

	res, err := c.svc.Harvest(userID, zoneKey, plotIdx)
	if err != nil {
		interaction.RespondError(b, i, lang, "farm.error")
		return
	}

	msg := i18n.T("farm.harvest_msg", lang, map[string]any{"qty": res.Quantity, "item": res.CropName})
	embed := components.Embed(
		i18n.T("farm.success_desc", lang, map[string]any{
			"loot": msg,
			"value": 0,
			"xp":   res.XP,
		}),
		"",
		0x00FF00,
	)
	back := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("farm.back", lang), components.Encode("farm", "zone", zoneKey), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, back))

	unlocks, uerr := achievement.CheckAndUnlock(b.DB, userID)
	if uerr == nil && len(unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, unlocks)
	}
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
