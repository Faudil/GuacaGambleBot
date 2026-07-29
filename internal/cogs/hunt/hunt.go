package hunt

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
	"guacagamblebot/internal/items"
	petsvc "guacagamblebot/internal/service/pets"
	huntsvc "guacagamblebot/internal/service/hunt"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *huntsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: huntsvc.New(s, cfg)}
	r.Slash("hunt", "Pet hunting expedition", c.onSlashMenu)
	r.Prefix("hunt", c.onPrefixMenu)
	r.Component("hunt", "menu", c.onMenu)
	r.Component("hunt", "zone", c.onHuntZone)
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
		i18n.T("hunt.dashboard_title", lang),
		i18n.T("hunt.dashboard_desc", lang, nil),
		0x0000FF,
	)
	for _, zone := range huntsvc.Zones {
		name := i18n.T("hunt."+zone.Key, lang)
		rangeStr := i18n.T("hunt.level_range", lang, map[string]any{"min": zone.LevelMin, "max": zone.LevelMax})
		embed.Fields = append(embed.Fields, components.Field(zone.Emoji+" "+name, rangeStr, true))
	}
	opts := make([]discordgo.SelectMenuOption, 0, len(huntsvc.Zones))
	for _, zone := range huntsvc.Zones {
		name := i18n.T("hunt."+zone.Key, lang)
		rangeStr := "Lvl " + itoa(zone.LevelMin) + "-" + itoa(zone.LevelMax)
		opts = append(opts, discordgo.SelectMenuOption{
			Label:       zone.Emoji + " " + name,
			Value:       zone.Key,
			Description: rangeStr,
		})
	}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			discordgo.SelectMenu{
				CustomID:    components.Encode("hunt", "zone"),
				Placeholder: i18n.T("hunt.select_zone", lang),
				Options:     opts,
			},
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
			interaction.RespondError(b, i, lang, "hunt.no_pet")
		case huntsvc.ErrPetKO:
			interaction.RespondError(b, i, lang, "hunt.pet_ko")
		default:
			interaction.RespondError(b, i, lang, "hunt.error")
		}
		return
	}

	psvc := petsvc.New(c.store, c.cfg)
	pet, _ := psvc.GetActivePet(userID)
	if pet != nil {
		ready, _ := c.store.CheckCooldown(userID, "pet_interaction", 180*time.Minute)
		if ready {
			ir := petsvc.MaybeTriggerInteraction(pet, "hunt")
			if ir != nil {
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
				if len(opts) > 0 {
					embed := components.Embed(
						i18n.T("pets.interact.title", lang, map[string]any{"name": pet.Nickname}),
						intro, 0x9b59b6)
					_, _ = b.Session.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
						Embeds: []*discordgo.MessageEmbed{embed},
						Components: []discordgo.MessageComponent{
							components.ActionRow(
								discordgo.SelectMenu{
									CustomID:    components.Encode("pets", "interact", strconv.FormatInt(pet.ID, 10)),
									Placeholder: i18n.T("pets.interact.placeholder", lang),
									Options:     opts,
								},
							),
						},
						Flags: discordgo.MessageFlagsEphemeral,
					})
				}
			}
		}
	}
	var artifactLeveled bool
	if pet != nil && res.XP > 0 {
		lvlRes := psvc.AddXP(pet, res.XP)
		if lvlRes.Leveled {
			res.LeveledUp = true
			res.NewLevel = lvlRes.NewLevel
		}
		if res.PlayerWon {
			psvc.AddBond(pet, 1)
			psvc.RecordHistory(pet, "hunt_win",
				"⚔️ **"+pet.Nickname+"** won a fight in the **"+zoneKey+"** zone and earned **"+itoa(res.XP)+" XP**.")
		} else {
			psvc.AddBond(pet, 1)
			psvc.RecordHistory(pet, "hunt_loss",
				"😰 **"+pet.Nickname+"** was defeated while hunting in the **"+zoneKey+"** zone...")
		}
		psvc.UpdatePet(pet)
		_, artifactLeveled, _ = psvc.AddArtifactXP(userID, petsvc.ArtifactHuntXP)
	}

	zone := huntsvc.Zones[zoneKey]
	zoneName := i18n.T("hunt."+zone.Key, lang)
	color := 0x00FF00
	desc := ""

	if res.PlayerWon {
		color = 0xFFD700
		desc = i18n.T("hunt.victory_msg", lang, map[string]any{"pet": "Your pet", "xp": res.XP})
		if len(res.Loot) > 0 {
			lootNames := make([]string, len(res.Loot))
			for i, loot := range res.Loot {
				lootNames[i] = items.DisplayName(loot)
			}
			desc += "\n\n" + i18n.T("hunt.loot_found", lang) + strings.Join(lootNames, ", ")
		}
	} else if res.EnemyWon {
		color = 0xFF0000
		desc = i18n.T("hunt.defeat_msg", lang, map[string]any{"pet": "Your pet", "xp": res.XP})
	} else {
		color = 0xFFA500
		desc = i18n.T("hunt.flee_msg", lang, map[string]any{"xp": res.XP})
	}

	if res.LeveledUp {
		desc += "\n\n" + i18n.T("hunt.level_up", lang, map[string]any{"pet": "Your pet", "level": res.NewLevel})
	}
	if artifactLeveled {
		desc += "\n\n" + i18n.T("pets.artifact.level_up", lang)
	}

	if qid := c.store.PopQuestCompleted(userID); qid != "" {
		desc += "\n\n" + questssvc.QuestCompletedMsg(qid, lang)
	}

	embed := components.Embed(
		i18n.T("hunt.expedition_title", lang, map[string]any{"emoji": zone.Emoji, "name": zoneName}),
		desc,
		color,
	)

	back := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("hunt.back", lang), components.Encode("hunt", "menu"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, back))

	unlocks, uerr := achievement.CheckAndUnlock(b.DB, userID)
	if uerr == nil && len(unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, unlocks)
	}
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
