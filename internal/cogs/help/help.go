package help

import (
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/store"
)

// Help categories. main.go files every cog under one of these when it wires the
// router, so this list is the whole taxonomy of the bot in one place.
const (
	CatStart      = "start"
	CatEconomy    = "economy"
	CatCasino     = "casino"
	CatRPG        = "rpg"
	CatActivities = "activities"
	CatPets       = "pets"
	CatWorld      = "world"
	CatAdmin      = "admin"
)

// categories fixes the display order and the icon of each category. Their names
// are localised through "help.cat.<key>".
var categories = []struct{ Key, Emoji string }{
	{CatStart, "🚀"},
	{CatEconomy, "💰"},
	{CatCasino, "🎰"},
	{CatRPG, "⚔️"},
	{CatActivities, "🎣"},
	{CatPets, "🐾"},
	{CatWorld, "🌍"},
	{CatAdmin, "🛠️"},
}

type Cog struct {
	store  *store.Store
	cfg    *config.Config
	router *interaction.Router
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, router: r}
	r.Slash("help", "cmd.help.desc", c.onSlash)
	r.Prefix("help", c.onPrefix)
	r.Prefix("h", c.onPrefix)
	r.Component("help", "category", c.onCategory)
	r.Component("help", "home", c.onHome)
}

func (c *Cog) lang(guildID string) string {
	return c.store.GetLanguage(interaction.ToInt64(guildID))
}

func (c *Cog) onSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.lang(i.GuildID)
	embed, comps := c.overview(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.lang(m.GuildID)
	embed, comps := c.overview(lang)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) onCategory(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.lang(i.GuildID)
	key := ""
	if vals := i.MessageComponentData().Values; len(vals) > 0 {
		key = vals[0]
	}
	embed, comps := c.categoryView(lang, key)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onHome(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.lang(i.GuildID)
	embed, comps := c.overview(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// overview lists every category with the commands it holds, so a new player can
// see the whole game on one screen before drilling into a section.
func (c *Cog) overview(lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := components.Embed(
		i18n.T("help.title", lang),
		i18n.T("help.desc", lang, map[string]any{"prefix": c.cfg.Prefix}),
		components.ColorBrand,
	)
	total := 0
	for _, cat := range categories {
		groups := c.router.Catalog(cat.Key)
		if len(groups) == 0 {
			continue
		}
		total += len(groups)
		names := make([]string, 0, len(groups))
		for _, g := range groups {
			names = append(names, "`/"+g.Name+"`")
		}
		embed.Fields = append(embed.Fields, components.Field(
			cat.Emoji+" "+i18n.T("help.cat."+cat.Key, lang),
			strings.Join(names, " "),
			false,
		))
	}
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("help.footer", lang, map[string]any{"count": total}),
	}
	return embed, c.controls(lang, "")
}

// categoryView details one category: every command, its aliases and what it does.
func (c *Cog) categoryView(lang, key string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	groups := c.router.Catalog(key)
	if len(groups) == 0 {
		return c.overview(lang)
	}
	emoji := ""
	for _, cat := range categories {
		if cat.Key == key {
			emoji = cat.Emoji
		}
	}
	var b strings.Builder
	for _, g := range groups {
		b.WriteString("`/" + g.Name + "`")
		if g.HasPrefix {
			b.WriteString(" · `" + c.cfg.Prefix + g.Name + "`")
		}
		if len(g.Aliases) > 0 {
			sort.Strings(g.Aliases)
			b.WriteString(" _(" + strings.Join(g.Aliases, ", ") + ")_")
		}
		b.WriteString("\n" + i18n.T(g.DescKey, lang) + "\n\n")
	}
	embed := components.Embed(
		emoji+" "+i18n.T("help.cat."+key, lang),
		strings.TrimRight(b.String(), "\n"),
		components.ColorBrand,
	)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("help.cat_footer", lang, map[string]any{"count": len(groups)}),
	}
	return embed, c.controls(lang, key)
}

// controls builds the category picker plus a way back to the overview.
func (c *Cog) controls(lang, selected string) []discordgo.MessageComponent {
	opts := make([]discordgo.SelectMenuOption, 0, len(categories))
	for _, cat := range categories {
		if len(c.router.Catalog(cat.Key)) == 0 {
			continue
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label:   i18n.T("help.cat."+cat.Key, lang),
			Value:   cat.Key,
			Emoji:   &discordgo.ComponentEmoji{Name: cat.Emoji},
			Default: cat.Key == selected,
		})
	}
	return []discordgo.MessageComponent{
		components.ActionRow(discordgo.SelectMenu{
			CustomID:    components.Encode("help", "category"),
			Placeholder: i18n.T("help.select_placeholder", lang),
			Options:     opts,
		}),
		components.ActionRow(components.Button(
			i18n.T("help.btn_overview", lang),
			components.Encode("help", "home"),
			discordgo.SecondaryButton,
		)),
	}
}
