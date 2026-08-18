package hunt

import (
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	huntsvc "guacagamblebot/internal/service/hunt"
	invsvc "guacagamblebot/internal/service/inventory"
	jsvc "guacagamblebot/internal/service/journal"
	npcsvc "guacagamblebot/internal/service/npcs"
	petsvc "guacagamblebot/internal/service/pets"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *huntsvc.Service
	qsvc  *questssvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	def := universe.Get(cfg.Universe)
	if def == nil {
		def = universe.Get("hoakhaven")
	}
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)
	c := &Cog{store: s, cfg: cfg, svc: huntsvc.New(s, cfg, npcSvc), qsvc: questssvc.New(s, cfg)}
	r.Slash("hunt", "Pet hunting expedition", c.onSlashMenu)
	r.Prefix("hunt", c.onPrefixMenu)
	r.Component("hunt", "menu", c.onMenu)
	r.Component("hunt", "zone", c.onHuntZone)
}

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

func (c *Cog) menu(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	desc := ""
	pet, _ := petsvc.New(c.store, c.cfg).GetActivePet(userID)
	if pet != nil {
		desc = i18n.T("hunt.dashboard_desc", lang, map[string]any{"name": pet.Nickname, "lvl": pet.Level})
	} else {
		desc = i18n.T("hunt.no_pet", lang)
	}
	if maxPerDay := c.cfg.HuntMaxPerDay; maxPerDay > 0 {
		_, remaining, _ := c.store.CheckGameLimit(userID, "hunt", maxPerDay)
		desc += "\n\n" + i18n.T("hunt.remaining", lang, map[string]any{"remaining": remaining, "max": maxPerDay})
	}
	embed := components.Embed(
		i18n.T("hunt.dashboard_title", lang),
		desc,
		0x0000FF,
	)
	progress, _ := c.store.GetZoneProgress(userID)
	zones := sortedZoneKeys()
	for _, key := range zones {
		zone := huntsvc.Zones[key]
		name := i18n.T("hunt."+zone.Key, lang)
		rangeStr := i18n.T("hunt.level_range", lang, map[string]any{"min": zone.LevelMin, "max": zone.LevelMax})
		embed.Fields = append(embed.Fields, components.Field(zone.Emoji+" "+name, rangeStr, true))
	}
	opts := make([]discordgo.SelectMenuOption, 0, len(zones))
	for _, key := range zones {
		zone := huntsvc.Zones[key]
		name := i18n.T("hunt."+zone.Key, lang)
		rangeStr := i18n.T("hunt.level_range", lang, map[string]any{"min": zone.LevelMin, "max": zone.LevelMax})
		if !huntsvc.FirstZones[key] {
			access, _ := c.svc.HasZoneAccess(userID, key)
			if !access {
				req := huntsvc.ZoneUnlockRequirements[key]
				prev := huntsvc.Zones[req.Previous]
				prevName := i18n.T("hunt."+prev.Key, lang)
				rangeStr = i18n.T("hunt.zone_locked_progress", lang, map[string]any{
					"current":    progress[req.Previous],
					"required":   req.RequiredWins,
					"prev_emoji": prev.Emoji,
					"prev_name":  prevName,
				})
			}
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label:       zone.Emoji + " " + name,
			Value:       key,
			Description: rangeStr,
		})
	}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			discordgo.SelectMenu{
				CustomID:    components.EncodeOwner(userID, "hunt", "zone"),
				Placeholder: i18n.T("hunt.select_zone", lang),
				Options:     opts,
			},
		),
	}
	return embed, comps
}

func (c *Cog) onMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.menu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onHuntZone(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	data := i.MessageComponentData()
	zoneKey := ""
	if len(data.Values) > 0 {
		zoneKey = data.Values[0]
	}
	if zoneKey == "" {
		_, zoneKey, _ = components.Decode(data.CustomID)
	}

	res, err := c.svc.ExecuteHunt(userID, zoneKey)
	if err != nil {
		switch err {
		case huntsvc.ErrNoPet:
			if granted, _ := c.qsvc.EnsureTutorialEgg(userID); granted {
				interaction.RespondError(b, i, lang, "quests.tutorial_egg_granted")
			} else {
				interaction.RespondError(b, i, lang, "hunt.no_pet")
			}
		case huntsvc.ErrPetKO:
			interaction.RespondError(b, i, lang, "hunt.pet_ko")
		case huntsvc.ErrHuntLimit:
			c.respondErrorParams(b, i, lang, "hunt.limit_reached", map[string]any{"max": c.huntMaxPerDay()})
		case huntsvc.ErrHuntCooldown:
			c.respondErrorParams(b, i, lang, "hunt.cooldown", map[string]any{"seconds": c.huntCooldownSeconds()})
		case huntsvc.ErrZoneLocked:
			req, ok := huntsvc.ZoneUnlockRequirements[zoneKey]
			if !ok {
				interaction.RespondError(b, i, lang, "hunt.error")
				break
			}
			prev := huntsvc.Zones[req.Previous]
			c.respondErrorParams(b, i, lang, "hunt.zone_locked_error", map[string]any{
				"required":   req.RequiredWins,
				"prev_emoji": prev.Emoji,
				"prev_name":  i18n.T("hunt."+prev.Key, lang),
			})
		case store.ErrInventoryFull:
			interaction.RespondError(b, i, lang, "inventory.full")
		default:
			slog.Error("hunt failed", "user", userID, "zone", zoneKey, "error", err)
			interaction.RespondError(b, i, lang, "hunt.error")
		}
		return
	}

	psvc := petsvc.New(c.store, c.cfg)
	pet, _ := psvc.GetActivePet(userID)

	var artifactLeveled bool
	var unlockedZone string
	if res.PlayerWon {
		if next, err := c.svc.RecordHuntWin(userID, zoneKey); err == nil && next != "" {
			unlockedZone = next
		}
		if res.IsBoss {
			_, _ = c.svc.RecordZoneBossKill(userID, zoneKey)
		}
	}
	if pet != nil && res.XP > 0 {
		lvlRes := psvc.AddXP(pet, res.XP)
		if lvlRes.Leveled {
			res.LeveledUp = true
			res.NewLevel = lvlRes.NewLevel
		}
		if res.PlayerWon {
			psvc.AddBond(pet, 1)
			psvc.RecordHistory(pet, "hunt_win",
				i18n.T("hunt.history_win", lang, map[string]any{
					"pet":  pet.Nickname,
					"zone": i18n.T("hunt."+huntsvc.Zones[zoneKey].Key, lang),
					"xp":   res.XP,
				}))
		} else {
			psvc.AddBond(pet, 1)
			psvc.RecordHistory(pet, "hunt_loss",
				i18n.T("hunt.history_loss", lang, map[string]any{
					"pet":  pet.Nickname,
					"zone": i18n.T("hunt."+huntsvc.Zones[zoneKey].Key, lang),
				}))
		}
		psvc.UpdatePet(pet)
		_, artifactLeveled, _ = psvc.AddArtifactXP(userID, petsvc.ArtifactHuntXP)
	}

	zone := huntsvc.Zones[zoneKey]
	zoneName := i18n.T("hunt."+zone.Key, lang)

	petEmoji := "🐾"
	if pet != nil {
		if pt := petsvc.PetTypes[pet.PetType]; pt != nil {
			petEmoji = pt.Emoji
		}
	}
	enemyName := i18n.T("hunt.enemies."+res.EnemyName, lang)
	if enemyName == "hunt.enemies."+res.EnemyName {
		enemyName = res.EnemyName
	}
	if res.IsBoss {
		enemyName += " " + i18n.T("hunt.boss_tag", lang)
	} else {
		enemyName += i18n.T("hunt.wild_suffix", lang)
	}
	petName := i18n.T("hunt.your_pet", lang)
	if pet != nil {
		petName = pet.Nickname
	}

	back := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("hunt.back", lang), components.EncodeOwner(userID, "hunt", "menu"), discordgo.SecondaryButton),
		),
	}

	if len(res.Turns) == 0 {
		desc := c.resultDesc(res, petName, lang, artifactLeveled, userID, unlockedZone)
		emb := components.Embed(
			i18n.T("hunt.expedition_title", lang, map[string]any{"emoji": zone.Emoji, "name": zoneName}),
			desc, 0xFFA500,
		)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, emb, back))
		if n, ok := c.store.PopQuestNotification(userID); ok {
			interaction.SendQuestNotification(b, i, n, lang)
		}

		if text, dm := jsvc.SceneLine(c.store, userID, "hunt", lang); text != "" {
			interaction.SendJournalScene(b, i, text, dm)
		}
		return
	}

	// Spawn frame: retro layout with full HP bars.
	petD := components.DisplayPet{
		Name: petName, Emoji: petEmoji, Level: pet.Level,
		HP: res.PetStartHP, MaxHP: res.PetMaxHP,
	}
	enemyD := components.DisplayPet{
		Name: enemyName, Emoji: res.EnemyEmoji, Level: res.EnemyLevel,
		HP: res.EnemyMaxHP, MaxHP: res.EnemyMaxHP,
	}
	spawn := c.huntRetroFrame(petD, enemyD,
		[]string{i18n.T("hunt.enemy_spawn", lang, map[string]any{"name": enemyName})},
		zone.Emoji, zoneName, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, spawn, nil))

	// Animate the fight, then show the result.
	go interaction.AnimateFight(
		res.Turns,
		func(journal []string, t battle.BattleTurn) *discordgo.MessageEmbed {
			petD.HP = t.Pet1HP
			enemyD.HP = t.Pet2HP
			petD.IsKO = t.Pet1HP <= 0
			enemyD.IsKO = t.Pet2HP <= 0
			return c.huntRetroFrame(petD, enemyD, journal, zone.Emoji, zoneName, lang)
		},
		func(frame *discordgo.MessageEmbed, comps []discordgo.MessageComponent) {
			_, _ = b.Session.InteractionResponseEdit(i.Interaction, components.WebhookEditResponse(frame, comps))
		},
		func(journal []string) {
			footerKey := "hunt.end_footer"
			color := 0xFFA500
			if res.PlayerWon {
				color = 0xFFD700
				footerKey = "hunt.victory_footer"
			} else if res.EnemyWon {
				color = 0xFF0000
				footerKey = "hunt.defeat_footer"
			}
			desc := c.resultDesc(res, petName, lang, artifactLeveled, userID, unlockedZone)
			petD.HP = res.PetHP
			enemyD.HP = res.EnemyHP
			petD.IsKO = res.PetHP <= 0
			enemyD.IsKO = res.EnemyHP <= 0
			final := c.huntRetroFrame(petD, enemyD, journal, zone.Emoji, zoneName, lang)
			final.Color = color
			final.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T(footerKey, lang)}
			final.Description = final.Description + "\n\n" + desc
			_, _ = b.Session.InteractionResponseEdit(i.Interaction, components.WebhookEditResponse(final, back))

			if n, ok := c.store.PopQuestNotification(userID); ok {

				interaction.SendQuestNotification(b, i, n, lang)
			}

			if text, dm := jsvc.SceneLine(c.store, userID, "hunt", lang); text != "" {
				interaction.SendJournalScene(b, i, text, dm)
			}
			c.maybePetInteraction(b, i, pet, lang)
			unlocks, uerr := achievement.CheckAndUnlock(b.DB, userID)
			if uerr == nil && len(unlocks) > 0 {
				interaction.SendAchievements(b, i, lang, unlocks)
			}
		},
	)
}

// huntRetroFrame renders one retro RPG battle frame for a hunt encounter.
func (c *Cog) huntRetroFrame(petD, enemyD components.DisplayPet, journal []string, zoneEmoji, zoneName, lang string) *discordgo.MessageEmbed {
	return components.FightFrameEmbed(
		i18n.T("hunt.expedition_title", lang, map[string]any{"emoji": zoneEmoji, "name": zoneName}),
		petD, enemyD,
		components.FightLabelsFor(lang, i18n.T("hunt.vs", lang)),
		journal,
	)
}

// resultDesc builds the post-fight summary (loot, XP, level-ups, quest note).
func (c *Cog) resultDesc(res *huntsvc.BattleResult, petName, lang string, artifactLeveled bool, userID int64, unlockedZone string) string {
	desc := ""
	if res.PlayerWon {
		desc = i18n.T("hunt.victory_msg", lang, map[string]any{"pet": petName, "xp": res.XP})
		if res.IsBoss {
			desc += i18n.T("hunt.boss_defeated", lang)
		}
		if len(res.Loot) > 0 {
			lootNames := make([]string, len(res.Loot))
			for i, loot := range res.Loot {
				lootNames[i] = items.LocalizedName(loot, lang)
			}
			desc += "\n\n" + i18n.T("hunt.loot_found", lang) + strings.Join(lootNames, ", ")
		} else {
			desc += i18n.T("hunt.no_loot", lang)
		}
	} else if res.EnemyWon {
		desc = i18n.T("hunt.defeat_msg", lang, map[string]any{"pet": petName, "xp": res.XP})
	} else {
		desc = i18n.T("hunt.flee_msg", lang, map[string]any{"xp": res.XP})
	}

	if res.LeveledUp {
		desc += "\n\n" + i18n.T("hunt.level_up", lang, map[string]any{"pet": petName, "level": res.NewLevel})
	}
	if res.CharLeveledUp {
		desc += "\n\n" + i18n.T("character.level_up", lang, map[string]any{"level": res.CharNewLevel})
	}
	if artifactLeveled {
		desc += "\n\n" + i18n.T("artifact.level_up", lang)
	}
	if unlockedZone != "" {
		uz := huntsvc.Zones[unlockedZone]
		desc += "\n\n" + i18n.T("hunt.zone_unlocked", lang, map[string]any{
			"emoji": uz.Emoji,
			"name":  i18n.T("hunt."+uz.Key, lang),
		})
	}
	return desc
}

// maybePetInteraction triggers a pet personality interaction when one is ready.
func (c *Cog) maybePetInteraction(b *interaction.Bot, i *discordgo.InteractionCreate, pet *model.UserPet, lang string) {
	if pet == nil {
		return
	}
	userID := interaction.ToInt64(interaction.UserID(i))
	ready, _ := c.store.CheckCooldown(userID, "pet_interaction", 180*time.Minute)
	if !ready {
		return
	}
	ir := petsvc.MaybeTriggerInteraction(pet, "hunt")
	if ir == nil {
		return
	}
	intro := i18n.T(ir.IntroKey(pet.Personality), lang)
	if intro == ir.IntroKey(pet.Personality) {
		intro = i18n.T(ir.GenericIntroKey(), lang)
	}
	opts := make([]discordgo.SelectMenuOption, 0, len(ir.Choices))
	for _, ch := range ir.Choices {
		label := i18n.T(ch.ChoiceLabelKey(), lang)
		if label == ch.ChoiceLabelKey() {
			label = ch.ID
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label: ch.Emoji + " " + label,
			Value: ch.ID,
		})
	}
	if len(opts) == 0 {
		return
	}
	embed := components.Embed(
		i18n.T("pets.interact.title", lang, map[string]any{"name": pet.Nickname}),
		intro, 0x9b59b6)
	_, _ = b.Session.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			components.ActionRow(
				discordgo.SelectMenu{
					CustomID:    components.EncodeOwner(userID, "pets", "interact", strconv.FormatInt(pet.ID, 10)),
					Placeholder: i18n.T("pets.interact.placeholder", lang),
					Options:     opts,
				},
			),
		},
		Flags: discordgo.MessageFlagsEphemeral,
	})
}

// huntMaxPerDay returns the configured daily hunt limit (0 means no limit).
func (c *Cog) huntMaxPerDay() int {
	if c.cfg != nil {
		return c.cfg.HuntMaxPerDay
	}
	return 0
}

// huntCooldownSeconds returns the configured cooldown between hunts.
func (c *Cog) huntCooldownSeconds() int {
	if c.cfg != nil {
		return c.cfg.HuntCooldownSeconds
	}
	return 0
}

// respondErrorParams replies with an ephemeral error message using i18n params.
func (c *Cog) respondErrorParams(b *interaction.Bot, i *discordgo.InteractionCreate, lang, key string, params map[string]any) {
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T(key, lang, params),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	return out
}

// sortedZoneKeys returns the hunt zone map keys ordered by ascending level range.
func sortedZoneKeys() []string {
	keys := make([]string, 0, len(huntsvc.Zones))
	for k := range huntsvc.Zones {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b string) int {
		return huntsvc.Zones[a].LevelMin - huntsvc.Zones[b].LevelMin
	})
	return keys
}
