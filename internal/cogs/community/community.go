package community

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	communitysvc "guacagamblebot/internal/service/community"
	"guacagamblebot/internal/store"
)

var buildingEmoji = map[string]string{
	"market":   "🛒",
	"bank":     "🏦",
	"hospital": "🏥",
	"statue":   "🗿",
}

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *communitysvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: communitysvc.New(s, cfg)}
	r.SlashWithOptions("community", "cmd.community.desc",
		[]*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionString, Name: "invest", Description: "Bâtiment à financer (market, bank, hospital, statue).", Required: false},
			{Type: discordgo.ApplicationCommandOptionString, Name: "resource", Description: "Ressource à donner (ex: money, pebble, wheat).", Required: false},
			{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Quantité à donner.", Required: false},
		}, c.onSlash)
	r.Prefix("community", c.onPrefix)
	r.Prefix("com", c.onPrefix)
	r.Component("community", "list", c.onList)
	r.Component("community", "inspect", c.onInspect)
	r.Component("community", "contribute", c.onContribute)
	r.Component("community", "stats", c.onStats)
	r.Modal("community", "invest_modal", c.onInvestModal)
}

func (c *Cog) onSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	building, resource, amount := "", "", 0
	for _, o := range i.ApplicationCommandData().Options {
		switch o.Name {
		case "invest":
			building = strings.TrimSpace(o.StringValue())
		case "resource":
			resource = strings.TrimSpace(o.StringValue())
		case "amount":
			amount = int(o.IntValue())
		}
	}
	if building != "" {
		if resource == "" {
			resource = "money"
		}
		if amount <= 0 {
			amount = 1
		}
		c.doInvest(b, i, lang, building, normalizeResource(resource), amount)
		return
	}
	embed, comps := c.listView(lang, interaction.ToInt64(i.GuildID))
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	fields := strings.Fields(m.Content)
	if len(fields) < 2 || fields[1] != "invest" {
		embed, comps := c.listView(lang, interaction.ToInt64(m.GuildID))
		_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: comps,
		})
		return
	}
	if len(fields) < 5 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("community.invest_usage", lang, map[string]any{"prefix": b.Prefix}))
		return
	}
	amount, err := strconv.Atoi(fields[4])
	if err != nil || amount <= 0 {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("community.invalid_amount", lang))
		return
	}
	userID := interaction.ToInt64(m.Author.ID)
	serverID := interaction.ToInt64(m.GuildID)
	res, err := c.svc.Invest(serverID, userID, fields[2], normalizeResource(fields[3]), amount)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, c.errorText(err, fields[2], fields[3], lang))
		return
	}
	embed := c.successEmbed(lang, serverID, fields[2], res, normalizeResource(fields[3]))
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func (c *Cog) onList(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.listView(lang, interaction.ToInt64(i.GuildID))
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onInspect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	serverID := interaction.ToInt64(i.GuildID)
	embed, comps := c.inspectView(lang, serverID, rest[0])
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onContribute(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 2 {
		return
	}
	building, resource := rest[0], rest[1]
	bName := c.buildingName(building, lang)
	customID := components.Encode("community", "invest_modal", building, resource)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: components.ModalResponse(customID,
			i18n.T("community.modal_title", lang, map[string]any{"name": bName}),
			components.TextInput("amount",
				i18n.T("community.modal_amount_label", lang), true,
				i18n.T("community.modal_amount_placeholder", lang),
				discordgo.TextInputShort, 1, 9)),
	})
}

func (c *Cog) onInvestModal(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_, _, rest := components.Decode(i.ModalSubmitData().CustomID)
	if len(rest) < 2 {
		return
	}
	building, resource := rest[0], rest[1]
	raw := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	amount, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || amount <= 0 {
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
				components.Embed("❌", i18n.T("community.invalid_amount", lang), components.ColorDanger), nil))
		return
	}
	serverID := interaction.ToInt64(i.GuildID)
	userID := interaction.ToInt64(interaction.UserID(i))
	res, err := c.svc.Invest(serverID, userID, building, resource, amount)
	if err != nil {
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
				components.Embed("❌", c.errorText(err, building, resource, lang), components.ColorDanger), nil))
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
			c.successEmbed(lang, serverID, building, res, resource), nil))
	if i.Interaction.Message != nil {
		embed, comps := c.inspectView(lang, serverID, building)
		_, _ = b.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    i.Interaction.Message.ChannelID,
			ID:         i.Interaction.Message.ID,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &comps,
		})
	}
}

func (c *Cog) onStats(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed := c.statsEmbed(b.Session, lang, interaction.ToInt64(i.GuildID),
		interaction.ToInt64(interaction.UserID(i)), i.Member)
	comps := c.listButtons(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) listView(lang string, serverID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := components.Embed(
		i18n.T("community.list_title", lang),
		i18n.T("community.menu_desc", lang),
		components.ColorSuccess,
	)
	projects, _ := c.svc.GetAllProjects(serverID)
	for _, p := range projects {
		bName := c.buildingName(p.Key, lang)
		bDesc := c.buildingDesc(p.Key, lang)
		info := fmt.Sprintf("**%s** (Lvl %d/%d)\n_%s_\n", bName, p.Level, p.MaxLevel, bDesc)
		if p.Bonuses != nil {
			for k, v := range p.Bonuses {
				info += fmt.Sprintf("✅ %s\n", i18n.T("community.bonus_"+k, lang, map[string]any{"val": v}))
			}
		} else {
			info += "❌ " + i18n.T("community.no_bonus_yet", lang) + "\n"
		}
		info += c.progressSummary(lang, p.Progress)
		embed.Fields = append(embed.Fields, components.Field("\u200b", info, false))
	}
	comps := c.listButtons(lang)
	return embed, comps
}

func (c *Cog) inspectView(lang string, serverID int64, building string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	info, err := c.svc.GetProjectInfo(serverID, building)
	if err != nil || info == nil {
		return components.Embed("❌", i18n.T("community.not_found", lang, map[string]any{"name": building}), components.ColorDanger),
			c.listButtons(lang)
	}
	bName := c.buildingName(building, lang)
	bDesc := c.buildingDesc(building, lang)
	embed := components.Embed(
		i18n.T("community.inspect_title", lang, map[string]any{
			"name": bName, "level": info.Level + 1,
		}),
		bDesc+"\n\n"+i18n.T("community.cur_level", lang, map[string]any{"level": info.Level, "max": info.MaxLevel}),
		components.ColorInfo,
	)
	if info.Level >= info.MaxLevel {
		embed.Description += "\n\n" + i18n.T("community.maxed", lang)
	}
	for _, p := range info.Progress {
		pct := 0.0
		if p.Required > 0 {
			pct = float64(p.Contributed) / float64(p.Required)
		}
		if pct > 1 {
			pct = 1
		}
		bar := progressBar(pct)
		embed.Fields = append(embed.Fields, components.Field(
			fmt.Sprintf("%s %s", c.resourceEmoji(p.Resource), c.resourceName(p.Resource, lang)),
			fmt.Sprintf("%s **%d/%d** (%d%%)", bar, p.Contributed, p.Required, int(pct*100)),
			false,
		))
	}
	if info.Bonuses != nil {
		bonusStr := ""
		for k, v := range info.Bonuses {
			bonusStr += fmt.Sprintf("✅ %s\n", i18n.T("community.bonus_"+k, lang, map[string]any{"val": v}))
		}
		embed.Fields = append(embed.Fields, components.Field(i18n.T("community.bonuses_title", lang), strings.TrimRight(bonusStr, "\n"), false))
	}
	return embed, c.inspectButtons(lang, info)
}

func (c *Cog) progressSummary(lang string, progress []communitysvc.ResourceProgress) string {
	line := ""
	done := 0
	total := 0
	for _, p := range progress {
		done += p.Contributed
		total += p.Required
	}
	if total > 0 {
		pct := float64(done) / float64(total)
		if pct > 1 {
			pct = 1
		}
		line += fmt.Sprintf("%s **%d/%d** (%d%%)\n", progressBar(pct), done, total, int(pct*100))
	}
	for _, p := range progress {
		line += fmt.Sprintf("%s %s %s\n", c.resourceEmoji(p.Resource),
			c.resourceName(p.Resource, lang),
			i18n.T("community.progress_fraction", lang,
				map[string]any{"current": p.Contributed, "required": p.Required}))
	}
	return line
}

func (c *Cog) listButtons(lang string) []discordgo.MessageComponent {
	row := make([]discordgo.MessageComponent, 0, len(communitysvc.BuildingOrder)+1)
	for _, key := range communitysvc.BuildingOrder {
		row = append(row, components.Button(
			buildingEmoji[key]+" "+c.buildingName(key, lang),
			components.Encode("community", "inspect", key),
			discordgo.SecondaryButton,
		))
	}
	row = append(row, components.Button(i18n.T("community.btn_stats", lang),
		components.Encode("community", "stats"), discordgo.SuccessButton))
	return []discordgo.MessageComponent{components.ActionRow(row...)}
}

func (c *Cog) inspectButtons(lang string, info *communitysvc.BuildingInfo) []discordgo.MessageComponent {
	row := make([]discordgo.MessageComponent, 0, len(info.Progress)+1)
	for _, p := range info.Progress {
		label := c.resourceEmoji(p.Resource) + " " + c.resourceName(p.Resource, lang)
		disabled := info.Level >= info.MaxLevel || p.Full()
		if p.Full() {
			label = "✅ " + label
		}
		row = append(row, components.ButtonDisabled(label,
			components.Encode("community", "contribute", info.Key, p.Resource),
			discordgo.PrimaryButton, disabled))
	}
	row = append(row, components.Button(i18n.T("community.btn_back", lang),
		components.Encode("community", "list"), discordgo.SecondaryButton))
	rows := []discordgo.MessageComponent{components.ActionRow(row...)}
	return rows
}

func (c *Cog) statsEmbed(s interaction.Session, lang string, serverID, userID int64, member *discordgo.Member) *discordgo.MessageEmbed {
	stats, _ := c.svc.GetUserStats(userID, serverID)
	embed := components.Embed(
		i18n.T("community.stats_title", lang, map[string]any{"user": interaction.DisplayName(s, fmt.Sprintf("%d", serverID), member, userID)}),
		"",
		components.ColorArcane,
	)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("community.res_money", lang), fmt.Sprintf("%d", stats.TotalMoneyInvested), true),
		components.Field(i18n.T("community.res_items", lang), fmt.Sprintf("%d", stats.TotalItemsInvested), true),
	}
	top, err := c.svc.GetTopContributors(serverID, 5)
	if err == nil && len(top) > 0 {
		lines := make([]string, 0, len(top))
		for i, t := range top {
			lines = append(lines, fmt.Sprintf("**%d.** %s — %d", i+1,
				interaction.DisplayName(s, fmt.Sprintf("%d", serverID), member, t.UserID), t.Total))
		}
		embed.Fields = append(embed.Fields, components.Field(i18n.T("community.top_contributors", lang),
			strings.Join(lines, "\n"), false))
	}
	return embed
}

func (c *Cog) successEmbed(lang string, serverID int64, building string, res *communitysvc.InvestResult, resource string) *discordgo.MessageEmbed {
	bName := c.buildingName(building, lang)
	msg := i18n.T("community.invest_success", lang, map[string]any{
		"amount": res.Invested, "res": c.resourceName(resource, lang), "building": bName,
	})
	embed := components.Embed("✅", msg, components.ColorSuccess)
	if res.LeveledUp {
		embed.Title = i18n.T("community.level_up_title", lang)
		embed.Description = i18n.T("community.level_up_desc", lang, map[string]any{
			"name": bName, "level": res.NewLevel,
		})
	}
	return embed
}

func (c *Cog) doInvest(b *interaction.Bot, i *discordgo.InteractionCreate, lang, building, resource string, amount int) {
	serverID := interaction.ToInt64(i.GuildID)
	userID := interaction.ToInt64(interaction.UserID(i))
	res, err := c.svc.Invest(serverID, userID, building, resource, amount)
	if err != nil {
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
				components.Embed("❌", c.errorText(err, building, resource, lang), components.ColorDanger), nil))
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource,
			c.successEmbed(lang, serverID, building, res, resource), nil))
}

func (c *Cog) errorText(err error, building, resource, lang string) string {
	switch {
	case errors.Is(err, communitysvc.ErrInvalidAmount):
		return i18n.T("community.invalid_amount", lang)
	case errors.Is(err, communitysvc.ErrBuildingNotFound):
		return i18n.T("community.not_found", lang, map[string]any{"name": building})
	case errors.Is(err, communitysvc.ErrMaxLevel):
		return i18n.T("community.max_level", lang, map[string]any{"name": c.buildingName(building, lang)})
	case errors.Is(err, communitysvc.ErrResourceNotNeeded):
		return i18n.T("community.res_not_needed", lang, map[string]any{"res": c.resourceName(resource, lang)})
	case errors.Is(err, communitysvc.ErrResourceFull):
		return i18n.T("community.res_already_full", lang, map[string]any{"res": c.resourceName(resource, lang)})
	case errors.Is(err, communitysvc.ErrNotEnoughMoney):
		return i18n.T("community.not_enough_money", lang)
	case errors.Is(err, communitysvc.ErrNotEnoughItems):
		return i18n.T("community.not_enough_items", lang)
	default:
		return "❌ " + err.Error()
	}
}

func (c *Cog) buildingName(key, lang string) string {
	return i18n.T("community.building_"+key+"_name", lang)
}

func (c *Cog) buildingDesc(key, lang string) string {
	return i18n.T("community.building_"+key+"_desc", lang)
}

func (c *Cog) resourceName(resource, lang string) string {
	if resource == communitysvc.MoneyKey {
		return i18n.T("community.res_money", lang)
	}
	return items.LocalizedName(resource, lang)
}

// normalizeResource resolves user input (display name, ID, or "money") to the
// canonical resource key used by the service.
func normalizeResource(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "money" || raw == "argent" || raw == "$" {
		return communitysvc.MoneyKey
	}
	if it := items.Get(raw); it != nil {
		return it.ID
	}
	return raw
}

func (c *Cog) resourceEmoji(resource string) string {
	if resource == communitysvc.MoneyKey {
		return "💰"
	}
	it := items.Get(resource)
	if it != nil && it.Emoji != "" {
		return it.Emoji
	}
	return "📦"
}

func progressBar(pct float64) string {
	filled := int(math.Round(pct * 10))
	if filled > 10 {
		filled = 10
	}
	out := ""
	for i := 0; i < filled; i++ {
		out += "🟦"
	}
	for i := filled; i < 10; i++ {
		out += "⬜"
	}
	return out
}
