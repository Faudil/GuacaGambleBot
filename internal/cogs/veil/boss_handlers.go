package veil

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	veilsvc "guacagamblebot/internal/service/veil"
)

func (c *Cog) handleBossAction(b *interaction.Bot, i *discordgo.InteractionCreate, action string, lang string) {
	userID := interaction.ToInt64(i.Member.User.ID)
	raid := c.getRaid(userID)
	if raid == nil {
		c.errorEphemeral(b, i, i18n.T("veil.encounter.not_in_raid", lang))
		return
	}

	mapped := c.mapBossAction(action)

	switch mapped {
	case "attack":
		c.startMinigame(b, i, userID, "attack", lang)
	case "heal":
		c.startMinigame(b, i, userID, "heal", lang)
	case "protect":
		c.startMinigame(b, i, userID, "protect", lang)
	default:
		c.recordAndCheckResolution(b, i, raid, userID, mapped, 0, lang)
	}
}

func (c *Cog) recordAndCheckResolution(b *interaction.Bot, i *discordgo.InteractionCreate, raid *model.VeilRaid, userID int64, action string, value int, lang string) {
	c.mu.Lock()
	if c.turnActions[raid.ID] == nil {
		c.turnActions[raid.ID] = map[int64]string{}
	}
	key := action
	if value > 0 {
		key = action + ":" + strconv.Itoa(value)
	}
	c.turnActions[raid.ID][userID] = key
	acted := len(c.turnActions[raid.ID])
	needed := len(veilsvc.GetParticipantsWith(raid))
	c.mu.Unlock()

	if acted < needed {
		c.errorEphemeral(b, i, i18n.T("veil.encounter.action_registered", lang, map[string]any{"acted": acted, "needed": needed}))
		return
	}

	c.finishBossTurn(b, i, raid, lang)
}

func (c *Cog) finishBossTurn(b *interaction.Bot, i *discordgo.InteractionCreate, raid *model.VeilRaid, lang string) {
	c.mu.Lock()
	actions := c.turnActions[raid.ID]
	delete(c.turnActions, raid.ID)
	c.mu.Unlock()

	actionMap := map[int64]string{}
	for uid, key := range actions {
		actionMap[uid] = key
	}

	res := veilsvc.ResolveBossTurn(raid, actionMap, lang)

	if res.BossDefeated {
		c.svc.EndRaid(raid, "completed")
		c.mu.Lock()
		delete(c.activeRaids, raid.ID)
		c.mu.Unlock()

		shards, chronicles, drops := veilsvc.GenerateRewards(raid, lang)
		desc := i18n.T("veil.rewards.victory_desc", lang)

		for _, d := range drops {
			desc += fmt.Sprintf("\n%s **%s** — <@%d>\n└ %s\n", d.Emoji, d.ItemName, d.UserID, d.StatsStr)
			for _, a := range d.Affixes {
				statLabel := formatStatLabel(a.Stat)
				desc += fmt.Sprintf("└ %s (+%d %s)\n", a.Name, a.Value, statLabel)
			}
		}

		for uid, s := range shards {
			desc += i18n.T("veil.rewards.shard_line", lang, map[string]any{"user": strconv.FormatInt(uid, 10), "shards": s})
		}
		for uid, chr := range chronicles {
			desc += i18n.T("veil.rewards.chronicle_line", lang, map[string]any{"user": strconv.FormatInt(uid, 10), "chronicle": chr})
		}

		embed := &discordgo.MessageEmbed{
			Title:       i18n.T("veil.rewards.victory_title", lang),
			Description: desc,
			Color:       components.ColorReward,
		}
		c.respond(b, i, embed, nil)
		return
	}

	if res.PhaseChanged > 0 {
		raid.BossPhase = res.PhaseChanged
		if res.PhaseChanged == 2 {
			c.svc.Store().SaveVeilRaid(raid)
			c.respond(b, i, veilsvc.RenderAnchorEmbed(lang), veilsvc.AnchorComponents())
		} else {
			r := veilsvc.StartBossPhase(raid, res.PhaseChanged, lang)
			r.Embed.Description = res.Desc + "\n\n" + r.Embed.Description
			c.svc.Store().SaveVeilRaid(raid)
			c.mu.Lock()
			delete(c.turnActions, raid.ID)
			c.mu.Unlock()
			c.respondBoss(b, i, raid, r.Embed, r.Comps)
		}
		return
	}

	c.svc.Store().SaveVeilRaid(raid)
	embed := veilsvc.RenderBossEmbed(raid, res.Desc, lang)
	c.respondBoss(b, i, raid, embed, nil)
}

func (c *Cog) mapBossAction(action string) string {
	switch action {
	case "boss_atk1", "boss_atk2", "boss_atk3":
		return "attack"
	case "boss_add":
		return "attack_add"
	case "boss_heal2", "boss_heal3":
		return "heal"
	case "boss_prot2", "boss_prot3":
		return "protect"
	case "stabilize":
		return "stabilize"
	}
	return action
}

func (c *Cog) handleAnchor(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	raid := c.getRaid(userID)
	if raid == nil {
		c.errorEphemeral(b, i, i18n.T("veil.encounter.not_in_raid", lang))
		return
	}
	c.errorEphemeral(b, i, i18n.T("veil.boss.anchor_done", lang))
}

func formatStatLabel(stat string) string {
	switch stat {
	case "str":
		return "STR"
	case "dex":
		return "DEX"
	case "int":
		return "INT"
	case "vit":
		return "VIT"
	case "luk":
		return "LUK"
	}
	return stat
}
