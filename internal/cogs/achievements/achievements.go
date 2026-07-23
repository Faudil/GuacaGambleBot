package achievements

import (
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	achievementsvc "guacagamblebot/internal/service/achievements"
	"guacagamblebot/internal/store"
)

// Cog implements the Achievements menu: a single view listing the invoking
// user's achievements.
type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *achievementsvc.Service
}

// Register wires the cog into the router under both slash and prefix triggers.
func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: achievementsvc.New(s, cfg)}
	r.Slash("achievements", "Voir vos succès et récompenses.", c.onSlashMenu)
	r.Slash("ach", "Voir vos succès et récompenses.", c.onSlashMenu)
	r.Prefix("achievements", c.onPrefixMenu)
	r.Prefix("ach", c.onPrefixMenu)
	r.Component("achievements", "show", c.onShow)
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
		i18n.T("achievements.list_title", lang),
		i18n.T("achievements.menu_desc", lang),
		0xf1c40f,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("achievements.btn_show", lang), components.Encode("achievements", "show"), discordgo.PrimaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onShow(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	views, err := c.svc.List(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "achievements.empty")
		return
	}

	title := i18n.T("achievements.list_title", lang)
	embed := components.Embed(title, "", 0xf1c40f)

	if len(views) == 0 {
		embed.Description = i18n.T("achievements.empty", lang)
	} else {
		lines := make([]string, 0, len(views))
		for _, v := range views {
			status := i18n.T("achievements.locked", lang)
			if v.Unlocked {
				status = i18n.T("achievements.unlocked", lang)
			}
			name := i18n.T("achievements."+v.ID+".name", lang)
			lines = append(lines, i18n.T("achievements.entry", lang, map[string]any{
				"emoji":  v.Emoji,
				"name":   name,
				"glory":  v.Glory,
				"status": status,
			}))
		}
		embed.Description = strings.Join(lines, "\n")
	}

	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}
