package achievements

import (
	"math"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/logger"
	achievementsvc "guacagamblebot/internal/service/achievements"
	"guacagamblebot/internal/store"
)

// pageSize bounds each achievements list page. A single embed listing all
// achievements (95+) would exceed Discord's 4096-character description limit
// and get rejected, leaving the interaction unanswered.
const pageSize = 25

// Cog implements the Achievements menu: a paginated view listing the invoking
// user's achievements.
type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *achievementsvc.Service
}

// Register wires the cog into the router under both slash and prefix triggers.
func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: achievementsvc.New(s, cfg)}
	r.Slash("achievements", "cmd.achievements.desc", c.onSlashMenu)
	r.Slash("ach", "cmd.achievements.desc", c.onSlashMenu)
	r.Prefix("achievements", c.onPrefixMenu)
	r.Prefix("ach", c.onPrefixMenu)
	r.Component("achievements", "show", c.onShow)
	r.Component("achievements", "nav", c.onNav)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.menu(lang, userID)
	if err := b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps)); err != nil {
		logger.Log().Error("achievements: failed to send menu",
			"error", err,
			"user", interaction.UserID(i),
			"guild", i.GuildID,
		)
	}
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	embed, comps := c.menu(lang, userID)
	if _, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	}); err != nil {
		logger.Log().Error("achievements: failed to send prefix menu",
			"error", err,
			"user", m.Author.ID,
			"guild", m.GuildID,
		)
	}
}

func (c *Cog) menu(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := components.Embed(
		i18n.T("achievements.list_title", lang),
		i18n.T("achievements.menu_desc", lang),
		components.ColorReward,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("achievements.btn_show", lang), components.EncodeOwner(userID, "achievements", "show"), discordgo.PrimaryButton),
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
	embed, comps := c.listView(lang, userID, views, 1)
	if err := b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps)); err != nil {
		logger.Log().Error("achievements: failed to send list",
			"error", err,
			"user", interaction.UserID(i),
			"guild", i.GuildID,
		)
	}
}

// onNav handles the pagination buttons of the achievements list. The requested
// page travels inside the custom_id; stale presses on an old embed are clamped
// to the current page count.
func (c *Cog) onNav(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	dir := ""
	page := 1
	if len(rest) >= 2 {
		dir = rest[0]
		if p, err := strconv.Atoi(rest[1]); err == nil {
			page = p
		}
	}
	views, err := c.svc.List(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "achievements.empty")
		return
	}
	switch dir {
	case "prev":
		page--
	case "next":
		page++
	}
	embed, comps := c.listView(lang, userID, views, page)
	if err := b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps)); err != nil {
		logger.Log().Error("achievements: failed to send page",
			"error", err,
			"user", interaction.UserID(i),
			"guild", i.GuildID,
		)
	}
}

// listView renders one page of the achievement list with prev/page/next
// navigation. Views must arrive in a stable order (the service sorts them by
// glory) so pages do not shuffle between renders.
func (c *Cog) listView(lang string, userID int64, views []achievementsvc.View, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := components.Embed(i18n.T("achievements.list_title", lang), "", components.ColorReward)
	if len(views) == 0 {
		embed.Description = i18n.T("achievements.empty", lang)
	} else {
		totalPages := max(1, int(math.Ceil(float64(len(views))/float64(pageSize))))
		page = min(max(page, 1), totalPages)
		start := (page - 1) * pageSize
		end := min(start+pageSize, len(views))
		lines := make([]string, 0, pageSize)
		for _, v := range views[start:end] {
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
	return embed, c.listComponents(lang, userID, len(views), page)
}

// listComponents builds the pagination row shown under the list: prev/page/next
// plus a show button that resets to page one. Every button is owner-gated.
func (c *Cog) listComponents(lang string, userID int64, total, page int) []discordgo.MessageComponent {
	totalPages := max(1, int(math.Ceil(float64(total)/float64(pageSize))))
	prevBtn := discordgo.Button{
		Label:    i18n.T("achievements.nav_prev", lang),
		CustomID: components.EncodeOwner(userID, "achievements", "nav", "prev", strconv.Itoa(page)),
		Style:    discordgo.SecondaryButton,
		Disabled: page <= 1,
	}
	pageBtn := discordgo.Button{
		Label:    i18n.T("achievements.nav_page", lang, map[string]any{"page": page, "total": totalPages}),
		CustomID: "_disabled",
		Style:    discordgo.SecondaryButton,
		Disabled: true,
	}
	nextBtn := discordgo.Button{
		Label:    i18n.T("achievements.nav_next", lang),
		CustomID: components.EncodeOwner(userID, "achievements", "nav", "next", strconv.Itoa(page)),
		Style:    discordgo.SecondaryButton,
		Disabled: page >= totalPages,
	}
	showBtn := components.Button(i18n.T("achievements.btn_show", lang), components.EncodeOwner(userID, "achievements", "show"), discordgo.PrimaryButton)
	return []discordgo.MessageComponent{
		components.ActionRow(showBtn),
		components.ActionRow(prevBtn, pageBtn, nextBtn),
	}
}
