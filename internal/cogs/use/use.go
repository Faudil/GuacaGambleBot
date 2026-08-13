package use

import (
	"errors"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	usesvc "guacagamblebot/internal/service/use"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *usesvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: usesvc.New(s, cfg)}
	r.SlashWithOptions("use", "Use a consumable item from your inventory.",
		[]*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "item", Description: "The item to use.", Required: true},
		}, c.onSlashUse)
	r.Prefix("use", c.onPrefixUse)
}

func (c *Cog) onSlashUse(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	opts := i.ApplicationCommandData().Options
	itemName := strings.ToLower(opts[0].StringValue())
	c.doUse(b, i, lang, itemName, true)
}

func (c *Cog) onPrefixUse(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	fields := strings.Fields(m.Content)
	if len(fields) < 2 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("use.not_usable", lang))
		return
	}
	itemName := strings.ToLower(fields[1])
	userID := interaction.ToInt64(m.Author.ID)

	desc, err := c.svc.Apply(userID, itemName)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, c.errorText(err, lang))
		return
	}
	_, _ = s.ChannelMessageSend(m.ChannelID, desc)
}

func (c *Cog) doUse(b *interaction.Bot, i *discordgo.InteractionCreate, lang, itemName string, channelSource bool) {
	userID := interaction.ToInt64(interaction.UserID(i))
	desc, err := c.svc.Apply(userID, itemName)
	if err != nil {
		interaction.RespondError(b, i, lang, c.errorKey(err))
		return
	}
	responseType := discordgo.InteractionResponseUpdateMessage
	if channelSource {
		responseType = discordgo.InteractionResponseChannelMessageWithSource
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(responseType,
			components.Embed(i18n.T("use.title", lang), desc, 0x2ecc71), nil))
}

func (c *Cog) errorText(err error, lang string) string {
	return i18n.T(c.errorKey(err), lang)
}

func (c *Cog) errorKey(err error) string {
	if errors.Is(err, usesvc.ErrNotOwned) {
		return "use.not_owned"
	}
	return "use.not_usable"
}
