package mining

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	miningsvc "guacagamblebot/internal/service/mining"
	"guacagamblebot/internal/store"
)

type userSession struct {
	depth    int
	bag      []miningsvc.BagEntry
	riskReduc int
}

var sessions = map[int64]*userSession{}

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *miningsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: miningsvc.New(s, cfg)}
	r.Slash("mine", "Mining expedition", c.onSlashMenu)
	r.Slash("m", "Mining expedition", c.onSlashMenu)
	r.Prefix("mine", c.onPrefixMenu)
	r.Prefix("m", c.onPrefixMenu)
	r.Component("mine", "descend", c.onDescend)
	r.Component("mine", "leave", c.onLeave)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	sessions[userID] = &userSession{depth: 1, riskReduc: 0}
	embed, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	sessions[userID] = &userSession{depth: 1, riskReduc: 0}
	embed, comps := c.menu(lang)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) menu(lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := components.Embed(
		i18n.T("mining.title", lang),
		i18n.T("mining.desc", lang),
		0x0000FF,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("mining.dig_label", lang), components.Encode("mine", "descend"), discordgo.PrimaryButton),
			components.Button(i18n.T("mining.leave_label", lang), components.Encode("mine", "leave"), discordgo.SuccessButton),
		),
	}
	return embed, comps
}

func (c *Cog) onDescend(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	sess, ok := sessions[userID]
	if !ok {
		sess = &userSession{depth: 1}
		sessions[userID] = sess
	}

	res, err := c.svc.Descend(userID, sess.depth, sess.bag, sess.riskReduc)
	if err != nil {
		if errors.Is(err, miningsvc.ErrMineLimit) {
			interaction.RespondError(b, i, lang, "mining.limit_reached")
		} else {
			slog.Error("mining descend failed", "user", userID, "error", err)
			interaction.RespondError(b, i, lang, "mining.error")
		}
		return
	}

	if res.Collapsed {
		delete(sessions, userID)
		embed := components.Embed(
			i18n.T("mining.title", lang),
			i18n.T("mining.collapse_msg", lang, map[string]any{"items": bagString(res.Bag, lang)}),
			0xFF0000,
		)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))
		return
	}

	sess.depth++
	sess.bag = res.Bag

	riskNext := (sess.depth - 1) * 5
	if riskNext < 0 {
		riskNext = 0
	}
	embed := components.Embed(
		i18n.T("mining.title", lang),
		i18n.T("mining.status", lang, map[string]any{
			"depth": sess.depth,
			"item":  items.DisplayName(res.Item.Name),
			"bag":   bagString(res.Bag, lang),
			"risk":  riskNext,
		}),
		0x0000FF,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("mining.dig_label", lang), components.Encode("mine", "descend"), discordgo.PrimaryButton),
			components.Button(i18n.T("mining.leave_label", lang), components.Encode("mine", "leave"), discordgo.SuccessButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onLeave(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	sess, ok := sessions[userID]
	if !ok {
		sess = &userSession{depth: 1}
	}

	res, err := c.svc.LeaveMine(userID, sess.bag)
	if err != nil {
		slog.Error("mining leave failed", "user", userID, "error", err)
		interaction.RespondError(b, i, lang, "mining.error")
		return
	}
	delete(sessions, userID)

	if len(res.Bag) > 0 {
		embed := components.Embed(
			i18n.T("mining.title", lang),
			i18n.T("mining.success_msg", lang, map[string]any{
				"bag": bagString(res.Bag, lang),
				"xp":  res.XP,
			}),
			0x00FF00,
		)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))
	} else {
		embed := components.Embed(
			i18n.T("mining.title", lang),
			i18n.T("mining.empty_msg", lang, map[string]any{"xp": res.XP}),
			0xC0C0C0,
		)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))
	}

	if len(res.Unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, res.Unlocks)
	}
}

func bagString(bag []miningsvc.BagEntry, lang string) string {
	if len(bag) == 0 {
		return i18n.T("mining.nothing", lang)
	}
	parts := make([]string, len(bag))
	for i, e := range bag {
		displayName := items.DisplayName(e.Name)
		if e.Count > 1 {
			parts[i] = displayName + " x" + itoa(e.Count)
		} else {
			parts[i] = displayName
		}
	}
	return strings.Join(parts, ", ")
}

func itoa(n int) string {
	return strings.TrimSpace(strings.Replace(strings.Replace(
		strings.TrimSpace(string(rune(n+'0'))), "\x00", "", -1), "\x00", "", -1))
}
