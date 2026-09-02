// Package glossary implements the /glossary command: a player codex listing
// every item and equipment base in the game, showing which ones the player
// has discovered (bought, crafted, mined, fished, hunted, or looted) and
// where to get the ones they haven't yet.
package glossary

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	glossarysvc "guacagamblebot/internal/service/glossary"
	"guacagamblebot/internal/store"
)

// itemsPerPage keeps each category page's embed description well under
// Discord's 4096-char limit, and each catalog page's item count needs no
// pagination cap tied to the 25-option/25-field limits since rows are plain
// description text, not fields or select options.
const itemsPerPage = 15

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *glossarysvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: glossarysvc.New(s)}
	r.Slash("glossary", "cmd.glossary.desc", c.onSlash)
	r.Prefix("glossary", c.onPrefix)
	r.Prefix("codex", c.onPrefix)
	r.Component("glossary", "cat", c.onCategorySelect)
	r.Component("glossary", "nav", c.onNav)
	r.Component("glossary", "item", c.onItemDetail)
	r.Component("glossary", "back", c.onBack)
}

func (c *Cog) onSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.buildOverview(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	embed, comps := c.buildOverview(userID, lang)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) buildOverview(userID int64, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	progress, totalD, totalT, err := c.svc.AllProgress(userID)
	if err != nil {
		return components.Embed("", i18n.T("glossary.error", lang), components.ColorDanger), nil
	}

	desc := i18n.T("glossary.overview_header", lang, map[string]any{"discovered": totalD, "total": totalT}) + "\n\n"
	var selectOpts []discordgo.SelectMenuOption
	for _, cat := range glossarysvc.Categories {
		p := progress[cat]
		label := i18n.T("inventory.category_"+string(cat), lang)
		desc += fmt.Sprintf("%s — %d/%d\n", label, p.D, p.T)
		selectOpts = append(selectOpts, discordgo.SelectMenuOption{
			Label:       label,
			Value:       string(cat),
			Description: fmt.Sprintf("%d/%d", p.D, p.T),
		})
	}

	embed := components.Embed(i18n.T("glossary.title", lang), desc, components.ColorIdle)
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("glossary.footer", lang)}

	sel := discordgo.SelectMenu{
		CustomID:    components.Encode("glossary", "cat"),
		Placeholder: i18n.T("glossary.category_placeholder", lang),
		Options:     selectOpts,
	}
	return embed, []discordgo.MessageComponent{components.ActionRow(&sel)}
}

func (c *Cog) buildCategoryPage(userID int64, lang string, cat items.Category, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	discovered, _ := c.svc.Discovered(userID)
	entries := items.ItemsByCategory(cat)

	totalPages := (len(entries) + itemsPerPage - 1) / itemsPerPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	label := i18n.T("inventory.category_"+string(cat), lang)
	d, t, _ := c.svc.CategoryProgress(userID, cat)
	desc := fmt.Sprintf("%s — %d/%d\n\n", label, d, t)

	start := page * itemsPerPage
	end := start + itemsPerPage
	if end > len(entries) {
		end = len(entries)
	}

	var rowOpts []discordgo.SelectMenuOption
	for _, it := range entries[start:end] {
		if discovered[it.ID] {
			desc += fmt.Sprintf("%s **%s**\n", it.Emoji, items.LocalizedName(it.ID, lang))
			rowOpts = append(rowOpts, discordgo.SelectMenuOption{
				Label: items.LocalizedName(it.ID, lang),
				Value: it.ID,
				Emoji: &discordgo.ComponentEmoji{Name: it.Emoji},
			})
		} else {
			desc += fmt.Sprintf("❓ **%s**\n", i18n.T("glossary.locked", lang))
		}
	}

	embed := components.Embed(i18n.T("glossary.title", lang)+" — "+label, desc, components.ColorIdle)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: i18n.T("glossary.page_footer", lang, map[string]any{"page": page + 1, "total": totalPages}),
	}

	var comps []discordgo.MessageComponent
	if len(rowOpts) > 0 {
		sel := discordgo.SelectMenu{
			CustomID:    components.Encode("glossary", "item", string(cat), strconv.Itoa(page)),
			Placeholder: i18n.T("glossary.item_placeholder", lang),
			Options:     rowOpts,
		}
		comps = append(comps, components.ActionRow(&sel))
	}

	navRow := []discordgo.MessageComponent{
		components.ButtonDisabled(i18n.T("inventory.nav_prev", lang),
			components.Encode("glossary", "nav", string(cat), strconv.Itoa(page-1)), discordgo.SecondaryButton, page <= 0),
		components.ButtonDisabled(i18n.T("inventory.nav_next", lang),
			components.Encode("glossary", "nav", string(cat), strconv.Itoa(page+1)), discordgo.SecondaryButton, page >= totalPages-1),
		components.Button(i18n.T("glossary.back", lang), components.Encode("glossary", "back"), discordgo.SecondaryButton),
	}
	comps = append(comps, components.ActionRow(navRow...))

	return embed, comps
}

func (c *Cog) buildItemDetail(userID int64, lang string, cat items.Category, page int, itemID string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	it := items.Get(itemID)
	if it == nil {
		return components.Embed("", i18n.T("glossary.error", lang), components.ColorDanger), nil
	}

	discoveredAt, ok := c.store.ItemDiscoveredAt(userID, itemID)
	desc := items.LocalizedDescription(itemID, lang) + "\n\n"
	if ok {
		desc += i18n.T("glossary.discovered_on", lang, map[string]any{"date": discoveredAt.Format("2006-01-02")}) + "\n\n"
	}
	desc += "**" + i18n.T("glossary.how_to_get", lang) + "**\n"
	for _, src := range glossarysvc.AcquisitionSources(itemID, lang) {
		desc += "• " + src + "\n"
	}

	title := fmt.Sprintf("%s %s", it.Emoji, items.LocalizedName(itemID, lang))
	embed := components.Embed(title, desc, components.ColorIdle)

	backBtn := components.Button(i18n.T("glossary.back", lang),
		components.Encode("glossary", "nav", string(cat), strconv.Itoa(page)), discordgo.SecondaryButton)
	return embed, []discordgo.MessageComponent{components.ActionRow(backBtn)}
}

func (c *Cog) onCategorySelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cat := items.Category(i.MessageComponentData().Values[0])
	embed, comps := c.buildCategoryPage(userID, lang, cat, 0)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onNav(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 2 {
		return
	}
	cat := items.Category(rest[0])
	page, _ := strconv.Atoi(rest[1])
	embed, comps := c.buildCategoryPage(userID, lang, cat, page)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onItemDetail(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 2 {
		return
	}
	cat := items.Category(rest[0])
	page, _ := strconv.Atoi(rest[1])
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return
	}
	embed, comps := c.buildItemDetail(userID, lang, cat, page, values[0])
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onBack(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.buildOverview(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}
