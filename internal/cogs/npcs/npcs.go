package npcs

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *npcsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: npcsvc.New(s, cfg)}
	r.Slash("npc", "Interagis avec les personnages du village.", c.onSlashMenu)
	r.Prefix("npc", c.onPrefixMenu)
	r.Component("npc", "back", c.onBack)
	for _, n := range npcsvc.NPCs {
		id := n.ID
		r.Component("npc", id, c.makeNPCSelect(id))
		r.Component("npc", "talk_"+id, c.makeTalk(id))
	}
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(lang, b, i)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	embed := components.Embed(i18n.T("npcs.list_title", lang), "", 0x3498db)
	allNPCs := c.svc.GetAllNPCMeta()
	var desc string
	for _, npc := range allNPCs {
		desc += fmt.Sprintf("%s **%s**\n", npc.Emoji, npc.Name)
	}
	embed.Description = desc
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
}

func (c *Cog) menu(lang string, b *interaction.Bot, i *discordgo.InteractionCreate) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	userID := interaction.ToInt64(interaction.UserID(i))
	embed := components.Embed(i18n.T("npcs.list_title", lang), "", 0x3498db)
	var desc string
	allReps, _ := c.svc.GetAllReputations(userID)
	repMap := map[string]*model.UserNPCReputation{}
	for _, r := range allReps {
		repMap[r.NPCID] = &r
	}
	allNPCs := c.svc.GetAllNPCMeta()
	for _, npc := range allNPCs {
		rep := repMap[npc.ID]
		lvl := 1
		points := 0
		if rep != nil {
			lvl = rep.Level
			points = rep.Reputation
		}
		nextLvl := 100 * lvl
		rankName := npcsvc.RankName(lvl)
		pct := 0
		if nextLvl > 0 {
			pct = points * 100 / nextLvl
		}
		filled := pct / 10
		if filled > 10 {
			filled = 10
		}
		bar := strings.Repeat("🟩", filled) + strings.Repeat("🟥", 10-filled)
		desc += fmt.Sprintf("%s **%s** (%s)\nLvl %d - %s (%d/%d)\n\n", npc.Emoji, npc.Name, rankName, lvl, bar, points, nextLvl)
	}
	embed.Description = desc
	var comps []discordgo.MessageComponent
	for _, npc := range allNPCs {
		comps = append(comps, components.ActionRow(
			components.Button(
				i18n.T("npcs.talk_button", lang, map[string]any{"name": npc.Name}),
				components.Encode("npc", npc.ID),
				discordgo.SecondaryButton,
			),
		))
	}
	return embed, comps
}

func (c *Cog) makeNPCSelect(npcID string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
		userID := interaction.ToInt64(interaction.UserID(i))
		npcData := c.svc.GetNPCData(npcID)
		if npcData == nil {
			interaction.RespondError(b, i, lang, "npcs.list_title")
			return
		}
		rep, _ := c.svc.GetReputation(userID, npcID)
		lvl := rep.Level
		points := rep.Reputation
		nextLvl := 100 * lvl
		rankName := npcsvc.RankName(lvl)
		pct := 0
		if nextLvl > 0 {
			pct = points * 100 / nextLvl
		}
		filled := pct / 10
		if filled > 10 {
			filled = 10
		}
		bar := strings.Repeat("🟩", filled) + strings.Repeat("🟥", 10-filled)
		greeting := npcData.Greetings[0]
		if lvl >= 2 && len(npcData.Greetings) > 1 {
			greeting = npcData.Greetings[1]
		}
		if lvl >= 3 && len(npcData.Greetings) > 2 {
			greeting = npcData.Greetings[2]
		}
		embed := components.Embed(
			fmt.Sprintf("%s %s", npcData.Emoji, npcData.Name),
			fmt.Sprintf("*%s*\n\n**Affinité** : %s (Lvl %d)\n%s (%d/%d)", greeting, rankName, lvl, bar, points, nextLvl),
			npcData.Color,
		)
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("npcs.topic_bio", lang), components.Encode("npc", "talk_"+npcID), discordgo.PrimaryButton),
			),
			components.ActionRow(
				components.Button("↩️", components.Encode("npc", "back"), discordgo.SecondaryButton),
			),
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
	}
}

func (c *Cog) makeTalk(npcID string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
		npcData := c.svc.GetNPCData(npcID)
		if npcData == nil {
			return
		}
		embed := components.Embed(
			fmt.Sprintf("%s %s - %s", npcData.Emoji, npcData.Name, i18n.T("npcs.topic_bio", lang)),
			npcData.Description,
			npcData.Color,
		)
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button("↩️", components.Encode("npc", npcID), discordgo.SecondaryButton),
			),
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
	}
}

func (c *Cog) onBack(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(lang, b, i)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}
