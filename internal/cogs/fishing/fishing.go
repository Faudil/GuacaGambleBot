package fishing

import (
	"strconv"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	fishingsvc "guacagamblebot/internal/service/fishing"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *fishingsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: fishingsvc.New(s, cfg)}
	r.Slash("fish", "Fishing minigame", c.onSlashMenu)
	r.Slash("f", "Fishing minigame", c.onSlashMenu)
	r.Prefix("fish", c.onPrefixMenu)
	r.Prefix("f", c.onPrefixMenu)
	r.Component("fish", "menu", c.onMenu)
	r.Component("fish", "pond", c.onCast)
	r.Component("fish", "river", c.onCast)
	r.Component("fish", "ocean", c.onCast)
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
		i18n.T("fishing.session_title", lang),
		i18n.T("fishing.session_desc", lang),
		0x008080,
	)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("fishing.pond_field_name", lang), i18n.T("fishing.pond_field_value", lang), true),
		components.Field(i18n.T("fishing.river_field_name", lang), i18n.T("fishing.river_field_value", lang), true),
		components.Field(i18n.T("fishing.ocean_field_name", lang), i18n.T("fishing.ocean_field_value", lang), true),
	}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("fishing.pond_label", lang), components.Encode("fish", "pond"), discordgo.SuccessButton),
			components.Button(i18n.T("fishing.river_label", lang), components.Encode("fish", "river"), discordgo.PrimaryButton),
			components.Button(i18n.T("fishing.ocean_label", lang), components.Encode("fish", "ocean"), discordgo.DangerButton),
		),
		components.ActionRow(
			components.Button(i18n.T("fishing.back", lang), components.Encode("fish", "menu"), discordgo.SecondaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onCast(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	cd, err := c.svc.CheckCooldown(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "fishing.error")
		return
	}
	if cd > 0 {
		secs := int(cd.Seconds())
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: i18n.T("fishing.cooldown", lang, map[string]any{"time": strconv.Itoa(secs)}),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	cid := i.MessageComponentData().CustomID
	_, action, _ := components.Decode(cid)

	res, err := c.svc.CastLine(userID, action)
	if err != nil {
		interaction.RespondError(b, i, lang, "fishing.error")
		return
	}

	embed := components.Embed(
		i18n.T("fishing.success_title", lang),
		i18n.T("fishing.success_desc", lang, map[string]any{
			"reaction": strconv.FormatFloat(res.Reaction, 'f', 3, 64),
			"item":     items.DisplayName(res.ItemName),
			"xp":       strconv.Itoa(res.XP),
		}),
		0xFFD700,
	)
	back := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("fishing.back", lang), components.Encode("fish", "menu"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, back))

	unlocks, uerr := achievement.CheckAndUnlock(b.DB, userID)
	if uerr == nil && len(unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, unlocks)
	}
}
