package use

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	invsvc "guacagamblebot/internal/service/inventory"
	usesvc "guacagamblebot/internal/service/use"
	"guacagamblebot/internal/store"
)

const maxMenuOptions = 25

var errNoUsableItems = errors.New("no usable items in inventory")

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *usesvc.Service
	inv   *invsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: usesvc.New(s, cfg), inv: invsvc.New(s, cfg)}
	r.Slash("use", "Use a consumable item from your inventory.", c.onSlashUse)
	r.Prefix("use", c.onPrefixUse)
	r.Component("use", "pick", c.onPickItem)
}

func (c *Cog) onSlashUse(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	embed, comps, err := c.usableMenu(lang, userID)
	if err != nil {
		interaction.RespondError(b, i, lang, c.errorKey(err))
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixUse(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	fields := strings.Fields(m.Content)
	if len(fields) < 2 {
		c.sendPrefixMenu(s, m.ChannelID, lang, interaction.ToInt64(m.Author.ID))
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

func (c *Cog) sendPrefixMenu(s *discordgo.Session, channelID, lang string, userID int64) {
	embed, comps, err := c.usableMenu(lang, userID)
	if err != nil {
		_, _ = s.ChannelMessageSend(channelID, c.errorText(err, lang))
		return
	}
	_, _ = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

// usableMenu builds the embed and select menu of usable items owned by the
// user. It returns an error when the user has no usable items.
func (c *Cog) usableMenu(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent, error) {
	result, err := c.inv.GetInventory(userID)
	if err != nil {
		return nil, nil, err
	}

	options := make([]discordgo.SelectMenuOption, 0, maxMenuOptions)
	for _, e := range result.Entries {
		if e.EquipInfo != nil || e.Item == nil || !usesvc.IsUsable(e.Item.ID) {
			continue
		}
		if len(options) >= maxMenuOptions {
			break
		}
		label := fmt.Sprintf("%s x%d", items.LocalizedName(e.Item.Name, lang), e.Quantity)
		if len(label) > 100 {
			label = label[:97] + "..."
		}
		options = append(options, discordgo.SelectMenuOption{
			Label: label,
			Value: e.Item.ID,
			Emoji: &discordgo.ComponentEmoji{Name: e.Item.Emoji},
		})
	}

	if len(options) == 0 {
		return nil, nil, errNoUsableItems
	}

	embed := components.Embed(
		i18n.T("use.title", lang),
		i18n.T("use.choose", lang),
		0x2ecc71,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(discordgo.SelectMenu{
			CustomID:    components.EncodeOwner(userID, "use", "pick"),
			Placeholder: i18n.T("use.select_placeholder", lang),
			Options:     options,
		}),
	}
	return embed, comps, nil
}

func (c *Cog) onPickItem(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		interaction.RespondError(b, i, lang, "use.not_usable")
		return
	}
	userID := interaction.ToInt64(interaction.UserID(i))
	c.doUse(b, i, lang, userID, data.Values[0])
}

func (c *Cog) doUse(b *interaction.Bot, i *discordgo.InteractionCreate, lang string, userID int64, itemName string) {
	desc, err := c.svc.Apply(userID, itemName)
	if err != nil {
		interaction.RespondError(b, i, lang, c.errorKey(err))
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
			components.Embed(i18n.T("use.title", lang), desc, 0x2ecc71), nil))
}

func (c *Cog) errorText(err error, lang string) string {
	return i18n.T(c.errorKey(err), lang)
}

func (c *Cog) errorKey(err error) string {
	if errors.Is(err, usesvc.ErrNotOwned) {
		return "use.not_owned"
	}
	if errors.Is(err, errNoUsableItems) {
		return "use.none"
	}
	return "use.not_usable"
}
