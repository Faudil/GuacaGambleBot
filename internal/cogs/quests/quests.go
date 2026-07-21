package quests

import (
	"fmt"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *questssvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: questssvc.New(s, cfg)}
	r.Slash("quest", "Affiche tes quêtes actives.", c.onSlashMenu)
	r.Prefix("quest", c.onPrefixMenu)
	r.Prefix("q", c.onPrefixMenu)
	r.Component("quest", "show", c.onShow)
	r.Component("quest", "advance", c.onAdvance)
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
		i18n.T("quests.title", lang),
		i18n.T("quests.menu_desc", lang),
		0x2ecc71,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("quests.btn_show", lang), components.Encode("quest", "show"), discordgo.PrimaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onShow(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	quests, err := c.svc.GetAllActiveQuests(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "quests.title")
		return
	}
	if len(quests) == 0 {
		interaction.RespondError(b, i, lang, "quests.no_active")
		return
	}
	embed := components.Embed(i18n.T("quests.title", lang), "", 0x2ecc71)
	var desc string
	for _, q := range quests {
		def := c.svc.GetQuestDef(q.QuestID)
		title := q.QuestID
		if def != nil {
			title = i18n.T(def.TitleKey, lang)
		}
		desc += fmt.Sprintf("🔹 **%s** (`!q %s`)\n", title, q.QuestID)
	}
	embed.Description = desc
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onAdvance(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	questID := rest[0]
	if err := c.svc.AdvanceStep(userID, questID, ""); err != nil {
		interaction.RespondError(b, i, lang, "quests.title")
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
			components.Embed("✅", i18n.T("quests.continue_label", lang), 0x2ecc71), nil))
}
