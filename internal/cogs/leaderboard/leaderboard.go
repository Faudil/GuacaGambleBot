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

// Cog implements the Leaderboard "embed interface": a category menu letting
// users browse the richest players and the biggest single casino wins.
type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *lb.Service
}

// Register wires the cog into the router under both slash and prefix triggers.
func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: lb.New(s, cfg)}
	r.Slash("leaderboard", "cmd.leaderboard.desc", c.onSlashMenu)
	r.Slash("lb", "cmd.leaderboard.desc", c.onSlashMenu)
	r.Prefix("leaderboard", c.onPrefixMenu)
	r.Prefix("lb", c.onPrefixMenu)
	r.Component("leaderboard", "category", c.onCategory)
	r.Component("leaderboard", "refresh", c.onRefresh)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(lang, "richest")
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	embed, comps := c.menu(lang, "richest")
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) onCategory(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	category := "richest"
	if vals := i.MessageComponentData().Values; len(vals) > 0 {
		category = vals[0]
	}
	embed, comps := c.menu(lang, category)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onRefresh(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	category := "richest"
	if len(rest) > 0 {
		category = rest[0]
	}
	embed, comps := c.menu(lang, category)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) menu(lang, category string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	var embed *discordgo.MessageEmbed
	switch category {
	case "slots", "coinflip":
		embed = c.recordsEmbed(lang, category)
	default:
		embed = c.wealthEmbed(lang)
	}

	categoryOpts := []discordgo.SelectMenuOption{
		{Label: i18n.T("leaderboard.cat_richest", lang), Value: "richest", Emoji: &discordgo.ComponentEmoji{Name: "🏆"}},
		{Label: i18n.T("leaderboard.cat_slots", lang), Value: "slots", Emoji: &discordgo.ComponentEmoji{Name: "🎰"}},
		{Label: i18n.T("leaderboard.cat_coinflip", lang), Value: "coinflip", Emoji: &discordgo.ComponentEmoji{Name: "🪙"}},
	}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			discordgo.SelectMenu{
				CustomID:    components.Encode("leaderboard", "category"),
				Placeholder: i18n.T("leaderboard.select_placeholder", lang),
				Options:     categoryOpts,
			},
		),
		components.ActionRow(
			components.Button(i18n.T("leaderboard.btn_refresh", lang), components.Encode("leaderboard", "refresh", category), discordgo.PrimaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) wealthEmbed(lang string) *discordgo.MessageEmbed {
	embed := components.Embed(
		i18n.T("leaderboard.menu_title", lang),
		i18n.T("leaderboard.menu_desc", lang),
		components.ColorReward,
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
	return embed
}

func (c *Cog) recordsEmbed(lang, game string) *discordgo.MessageEmbed {
	titleKey := "leaderboard.records_title_slots"
	if game == "coinflip" {
		titleKey = "leaderboard.records_title_coinflip"
	}
	embed := components.Embed(i18n.T(titleKey, lang), "", components.ColorReward)
	records, err := c.svc.TopWinRecords(game, 10)
	if err == nil {
		if len(records) == 0 {
			embed.Description = i18n.T("leaderboard.records_empty", lang)
		} else {
			lines := ""
			for rank, r := range records {
				lines += i18n.T("leaderboard.records_entry", lang, map[string]any{
					"rank":   rank + 1,
					"user":   interaction.Mention(r.UserID),
					"amount": strconv.Itoa(r.Amount),
				}) + "\n"
			}
			embed.Description = lines
		}
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("leaderboard.title", lang)}
	return embed
}
