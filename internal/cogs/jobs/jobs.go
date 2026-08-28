package jobs

import (
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	jobssvc "guacagamblebot/internal/service/jobs"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *jobssvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: jobssvc.New(s, cfg)}
	r.Slash("jobs", "cmd.jobs.desc", c.onSlashMenu)
	r.Slash("job", "cmd.jobs.desc", c.onSlashMenu)
	r.Prefix("jobs", c.onPrefixMenu)
	r.Prefix("level", c.onPrefixMenu)
	r.Prefix("jobstats", c.onPrefixMenu)
	r.Prefix("jb", c.onPrefixMenu)
	r.Component("jobs", "show", c.onShow)
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
	embed := components.Embed(
		i18n.T("jobs.menu_title", lang),
		i18n.T("jobs.menu_desc", lang),
		components.ColorWarning,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("jobs.btn_show", lang), components.EncodeOwner(userID, "jobs", "show"), discordgo.PrimaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onShow(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	res, err := c.svc.GetJobs(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "jobs.menu_desc")
		return
	}
	embed := components.Embed(
		i18n.T("jobs.title", lang, map[string]any{"user": interaction.DisplayName(b.Session, i.GuildID, i.Member, userID)}),
		i18n.T("jobs.footer", lang, map[string]any{"total": res.TotalLevel}),
		components.ColorWarning,
	)
	for _, j := range res.Jobs {
		pct := 0
		if j.Next > 0 {
			pct = j.XP * 100 / j.Next
		}
		filled := pct / 10
		if filled > 10 {
			filled = 10
		}
		bar := strings.Repeat("🟩", filled) + strings.Repeat("🟥", 10-filled)
		name := i18n.T("jobs."+j.Name, lang)
		val := i18n.T("jobs.xp_line", lang, map[string]any{
			"level": j.Level, "xp": j.XP, "next": j.Next, "bar": bar,
		})
		embed.Fields = append(embed.Fields, components.Field(name, val, false))
	}
	_, comps := c.menu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}
