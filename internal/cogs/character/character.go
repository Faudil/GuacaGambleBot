package character

import (
	"strconv"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	charsvc "guacagamblebot/internal/service/character"
	"guacagamblebot/internal/store"
)

// Cog implements the Character / Profile interface: a single menu with a Show
// button that displays the invoking user's profile.
type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *charsvc.Service
}

// Register wires the cog into the router under both slash and prefix triggers.
func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: charsvc.New(s, cfg)}
	r.Slash("character", "Affiche ton profil de joueur.", c.onSlashMenu)
	r.Prefix("character", c.onPrefixMenu)
	r.Component("character", "show", c.onShow)
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
		i18n.T("character.menu_title", lang),
		i18n.T("character.menu_desc", lang),
		0x9b59b6,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("character.btn_show", lang), components.Encode("character", "show"), discordgo.PrimaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onShow(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	res, err := c.svc.Profile(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "character.menu_desc")
		return
	}
	embed := components.Embed(i18n.T("character.title", lang), "", 0x9b59b6)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("character.wallet", lang), "$"+strconv.Itoa(res.Wallet), true),
		components.Field(i18n.T("character.bank", lang), "$"+strconv.Itoa(res.Bank), true),
		components.Field(i18n.T("character.crowns", lang), strconv.Itoa(res.Crowns), true),
		components.Field(i18n.T("character.achievements", lang), strconv.Itoa(res.AchCount), false),
	}
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}
