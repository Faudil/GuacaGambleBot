package community

import (
	"fmt"
	"math"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	communitysvc "guacagamblebot/internal/service/community"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *communitysvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: communitysvc.New(s, cfg)}
	r.Slash("community", "Gère les projets communautaires du serveur.", c.onSlashMenu)
	r.Prefix("community", c.onPrefixMenu)
	r.Prefix("com", c.onPrefixMenu)
	r.Component("community", "list", c.onList)
	r.Component("community", "inspect", c.onInspect)
	r.Component("community", "stats", c.onStats)
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
		i18n.T("community.list_title", lang),
		i18n.T("community.menu_desc", lang),
		0x2ecc71,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("community.btn_list", lang), components.Encode("community", "list"), discordgo.PrimaryButton),
			components.Button(i18n.T("community.btn_inspect", lang), components.Encode("community", "inspect"), discordgo.SecondaryButton),
			components.Button(i18n.T("community.btn_stats", lang), components.Encode("community", "stats"), discordgo.SuccessButton),
		),
	}
	return embed, comps
}

func (c *Cog) onList(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	serverID := interaction.ToInt64(i.GuildID)
	projects, _ := c.svc.GetAllProjects(serverID)
	embed := components.Embed(i18n.T("community.list_title", lang), "", 0x2ecc71)
	for _, p := range projects {
		bName := i18n.T("community.building_"+p.Key+"_name", lang)
		bDesc := i18n.T("community.building_"+p.Key+"_desc", lang)
		info := fmt.Sprintf("**%s** (Lvl %d/%d)\n_%s_\n", bName, p.Level, p.MaxLevel, bDesc)
		if p.Bonuses != nil {
			for k, v := range p.Bonuses {
				info += fmt.Sprintf("✅ %s\n", i18n.T("community.bonus_"+k, lang, map[string]any{"val": v}))
			}
		} else {
			info += "❌ " + i18n.T("community.no_bonus_yet", lang) + "\n"
		}
		embed.Fields = append(embed.Fields, components.Field("\u200b", info, false))
	}
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onInspect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed := components.Embed(i18n.T("community.inspect_title", lang, map[string]any{"name": "?", "level": "?"}), "", 0x3498db)
	projects, _ := c.svc.GetAllProjects(interaction.ToInt64(i.GuildID))
	for _, p := range projects {
		if p.Costs == nil {
			continue
		}
		for res, required := range p.Costs {
			resName := i18n.T("community.res_money", lang)
			if res != "money" {
				resName = res
			}
			pct := 0.0
			if required > 0 {
				pct = 0
			}
			bar := progressBar(pct)
			embed.Fields = append(embed.Fields, components.Field(
				fmt.Sprintf("%s (0/%d)", resName, required),
				fmt.Sprintf("%s %d%%", bar, int(pct*100)),
				false,
			))
		}
	}
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onStats(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	serverID := interaction.ToInt64(i.GuildID)
	stats, _ := c.svc.GetUserStats(userID, serverID)
	embed := components.Embed(
		i18n.T("community.stats_title", lang, map[string]any{"user": interaction.Mention(userID)}),
		"",
		0x9b59b6,
	)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("community.res_money", lang), fmt.Sprintf("%d", stats.TotalMoneyInvested), true),
		components.Field("Items", fmt.Sprintf("%d", stats.TotalItemsInvested), true),
	}
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func progressBar(pct float64) string {
	filled := int(math.Round(pct * 10))
	if filled > 10 {
		filled = 10
	}
	out := ""
	for i := 0; i < filled; i++ {
		out += "🟦"
	}
	for i := filled; i < 10; i++ {
		out += "⬜"
	}
	return out
}
