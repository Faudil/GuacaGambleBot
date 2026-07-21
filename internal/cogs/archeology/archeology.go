package archeology

import (
	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	archsvc "guacagamblebot/internal/service/archeology"
	"guacagamblebot/internal/store"
)

type userGame struct {
	state *archsvc.GameState
}

var games = map[int64]*userGame{}

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *archsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: archsvc.New(s, cfg)}
	r.Slash("dig", "Archeology fossil excavation", c.onSlashMenu)
	r.Prefix("dig", c.onPrefixMenu)
	r.Component("arch", "safe", c.onSafePermit)
	r.Component("arch", "faille", c.onFaillePermit)
	r.Component("arch", "dynamite", c.onAction)
	r.Component("arch", "hammer", c.onAction)
	r.Component("arch", "brush", c.onAction)
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
		i18n.T("archeology.bureau_title", lang),
		i18n.T("archeology.bureau_desc", lang)+
			i18n.T("archeology.safe_desc", lang)+
			i18n.T("archeology.fault_desc", lang),
		0x0000FF,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("archeology.safe_permit_label", lang), components.Encode("arch", "safe"), discordgo.SuccessButton),
			components.Button(i18n.T("archeology.fault_permit_label", lang), components.Encode("arch", "faille"), discordgo.DangerButton),
		),
	}
	return embed, comps
}

func (c *Cog) onSafePermit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	c.startGame(b, i, userID, "safe", lang)
}

func (c *Cog) onFaillePermit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	c.startGame(b, i, userID, "faille", lang)
}

func (c *Cog) startGame(b *interaction.Bot, i *discordgo.InteractionCreate, userID int64, permitType, lang string) {
	state, err := c.svc.NewGame(userID, permitType)
	if err != nil {
		interaction.RespondError(b, i, lang, "archeology.no_money_permit")
		return
	}
	games[userID] = &userGame{state: state}

	embed, comps := c.gameEmbed(state, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) gameEmbed(state *archsvc.GameState, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	depthPct := float64(state.Depth) / 50.0
	if depthPct < 0 {
		depthPct = 0
	}
	blocksFull := int((1.0 - depthPct) * 5)
	depthBar := ""
	for i := 0; i < blocksFull; i++ {
		depthBar += "🟫"
	}
	for i := blocksFull; i < 5; i++ {
		depthBar += "⬛"
	}

	intPct := float64(state.Integrity) / 100.0
	if intPct < 0 {
		intPct = 0
	}
	intBlocks := int(intPct * 5)
	intBar := ""
	for i := 0; i < intBlocks; i++ {
		intBar += "❤️"
	}
	for i := intBlocks; i < 5; i++ {
		intBar += "💔"
	}

	embed := components.Embed(
		i18n.T("archeology.mini_game_title", lang),
		"",
		0xB8860B,
	)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("archeology.depth_remaining", lang), depthBar+" "+itoa(state.Depth)+" cm", false),
		components.Field(i18n.T("archeology.integrity", lang), intBar+" "+itoa(state.Integrity)+"%", false),
		components.Field(i18n.T("archeology.actions_remaining", lang), "**"+itoa(state.Actions)+"**", false),
	}
	permitName := i18n.T("archeology.safe_site", lang)
	if state.PermitType == "faille" {
		permitName = i18n.T("archeology.fault_site", lang)
	}
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("archeology.permit_footer", lang, map[string]any{"name": permitName})}

	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("archeology.dynamite_label", lang), components.Encode("arch", "dynamite"), discordgo.DangerButton),
			components.Button(i18n.T("archeology.hammer_label", lang), components.Encode("arch", "hammer"), discordgo.PrimaryButton),
			components.Button(i18n.T("archeology.brush_label", lang), components.Encode("arch", "brush"), discordgo.SuccessButton),
		),
	}
	return embed, comps
}

func (c *Cog) onAction(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	g, ok := games[userID]
	if !ok || g.state.Finished {
		interaction.RespondError(b, i, lang, "archeology.error")
		return
	}

	cid := i.MessageComponentData().CustomID
	_, action, _ := components.Decode(cid)

	var act archsvc.ActionType
	switch action {
	case "dynamite":
		act = archsvc.ActionDynamite
	case "hammer":
		act = archsvc.ActionHammer
	case "brush":
		act = archsvc.ActionBrush
	}

	outcome := c.svc.ApplyAction(g.state, act)

	if outcome.Finished {
		res := c.svc.Resolve(g.state)
		delete(games, userID)

		embed, _ := c.gameEmbed(&outcome.State, lang)
		embed.Description = outcomeMsg(res, lang)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))
		return
	}

	embed, comps := c.gameEmbed(&outcome.State, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func outcomeMsg(res *archsvc.DigResult, lang string) string {
	switch res.ItemName {
	case "Poussière d'os":
		return i18n.T("archeology.disaster_msg", lang) + "\n" + i18n.T("archeology.received", lang, map[string]any{"item": res.ItemName})
	case "Fossile Abîmé":
		return i18n.T("archeology.damaged_msg", lang) + "\n" + i18n.T("archeology.received", lang, map[string]any{"item": res.ItemName})
	case "ADN Pur":
		return i18n.T("archeology.perfect_msg", lang, map[string]any{"item": res.ItemName})
	case "Fragment Légendaire":
		return i18n.T("archeology.legendary_msg", lang, map[string]any{"item": res.ItemName, "integrity": 100})
	default:
		return i18n.T("archeology.success_msg", lang, map[string]any{"item": res.ItemName, "integrity": 100})
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
