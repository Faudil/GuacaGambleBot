package leaderboard

import (
	"strconv"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	lb "guacagamblebot/internal/service/leaderboard"
	"guacagamblebot/internal/store"
)

// Cog implements the Leaderboard "embed interface": a single menu listing the
// top 10 users by wallet balance with a Refresh button.
type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *lb.Service
}

// Register wires the cog into the router under both slash and prefix triggers.
func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: lb.New(s, cfg)}
	r.Slash("leaderboard", "Classement des plus riches joueurs.", c.onSlashMenu)
	r.Slash("lb", "Classement des plus riches joueurs.", c.onSlashMenu)
	r.Prefix("leaderboard", c.onPrefixMenu)
	r.Prefix("lb", c.onPrefixMenu)
	r.Component("leaderboard", "refresh", c.onRefresh)
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
		i18n.T("leaderboard.menu_title", lang),
		i18n.T("leaderboard.menu_desc", lang),
		0xf1c40f,
	)
	users, err := c.svc.Top(10)
	if err == nil {
		if len(users) == 0 {
			embed.Description = i18n.T("leaderboard.empty", lang)
		} else {
			lines := ""
			for rank, u := range users {
				lines += i18n.T("leaderboard.entry", lang, map[string]any{
					"rank":    rank + 1,
					"user":    interaction.Mention(u.UserID),
					"balance": strconv.Itoa(u.Balance),
				}) + "\n"
			}
			embed.Description = lines
		}
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("leaderboard.title", lang)}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("leaderboard.btn_refresh", lang), components.Encode("leaderboard", "refresh"), discordgo.PrimaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onRefresh(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}
