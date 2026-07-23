package housing

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	housingsvc "guacagamblebot/internal/service/housing"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *housingsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: housingsvc.New(s, cfg)}
	r.Slash("house", "Gère ta maison (acheter, améliorer, collecter).", c.onSlashMenu)
	r.Slash("hs", "Gère ta maison (acheter, améliorer, collecter).", c.onSlashMenu)
	r.Prefix("house", c.onPrefixMenu)
	r.Prefix("hs", c.onPrefixMenu)
	r.Component("house", "show", c.onShow)
	r.Component("house", "collect", c.onCollect)
	r.Component("house", "tree", c.onTree)
	r.Component("house", "upgrade", c.onUpgrade)
	r.Modal("house", "rename", c.onRename)
	r.Modal("house", "color", c.onColor)
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
		i18n.T("housing.menu_title", lang),
		i18n.T("housing.menu_desc", lang),
		0xB9936C,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("housing.btn_show", lang), components.Encode("house", "show"), discordgo.PrimaryButton),
			components.Button(i18n.T("housing.btn_collect", lang), components.Encode("house", "collect"), discordgo.SuccessButton),
		),
		components.ActionRow(
			components.Button(i18n.T("housing.btn_tree", lang), components.Encode("house", "tree"), discordgo.SecondaryButton),
			components.Button(i18n.T("housing.btn_upgrade", lang), components.Encode("house", "upgrade"), discordgo.PrimaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onShow(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	h, err := c.svc.GetHousing(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "housing.no_house")
		return
	}
	ht := housingsvc.Houses[h.HouseType]
	if ht == nil {
		interaction.RespondError(b, i, lang, "housing.no_house")
		return
	}
	title := i18n.T("housing.title", lang, map[string]any{"user": interaction.Mention(userID)})
	if h.CustomName != nil && *h.CustomName != "" {
		title = fmt.Sprintf("🏠 %s", *h.CustomName)
	}
	color := ht.Color
	if h.CustomColor != nil && *h.CustomColor != "" {
		if c, err := strconv.ParseInt(*h.CustomColor, 16, 64); err == nil {
			color = int(c)
		}
	}
	houseName := i18n.T("housing.types."+h.HouseType, lang)
	embed := components.Embed(title, fmt.Sprintf("**%s** (Lvl %d)", houseName, h.Level), color)

	collectInfo, _ := c.svc.GetCollectInfo(userID)
	if collectInfo != nil {
		incomeText := fmt.Sprintf("💰 **$%d** pending\n", collectInfo.Income)
		if len(collectInfo.Items) > 0 {
			for _, item := range collectInfo.Items {
				parts := strings.SplitN(item, ":", 2)
				if len(parts) == 2 {
					incomeText += fmt.Sprintf("• %s: `x%s`\n", parts[0], parts[1])
				}
			}
		}
		embed.Fields = append(embed.Fields, components.Field(i18n.T("housing.pending_rewards", lang), incomeText, false))
	}

	buffsText := strings.Join(ht.Buffs, "\n")
	embed.Fields = append(embed.Fields, components.Field(i18n.T("housing.stats_label", lang),
		i18n.T("housing.stats", lang, map[string]any{"level": h.Level, "buffs": buffsText}), false))

	if h.UnderConstruction != nil && *h.UnderConstruction != "" {
		embed.Fields = append(embed.Fields, components.Field("🛠️ Construction",
			fmt.Sprintf("**%s** en cours...", *h.UnderConstruction), false))
	}

	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onCollect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	income, items, err := c.svc.Collect(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "housing.nothing_to_collect")
		return
	}
	msg := fmt.Sprintf("💰 **Collected!** +$%d", income)
	if len(items) > 0 {
		msg += "\n📦 **Resources:** " + strings.Join(items, ", ")
	}
	embed := components.Embed("📦 Collect", msg, 0x2ecc71)
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onTree(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed := components.Embed(
		i18n.T("housing.tree_title", lang),
		i18n.T("housing.tree_desc", lang),
		0x1B5E20,
	)
	for _, upg := range housingsvc.UpgradesTree {
		itemsReq := ""
		for item, qty := range upg.CostItems {
			itemsReq += fmt.Sprintf("%dx %s ", qty, item)
		}
		embed.Fields = append(embed.Fields, components.Field(
			fmt.Sprintf("%s (%s)", upg.Name, upg.Branch),
			fmt.Sprintf("💰 $%d\n📦 %s\n⏱ %dh\n*%s*", upg.CostMoney, itemsReq, upg.TimeHours, upg.BonusDesc),
			false,
		))
	}
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onUpgrade(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	err := c.svc.UpgradeLevel(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "housing.max_level")
		return
	}
	embed := components.Embed("✅", i18n.T("housing.upgrade_success", lang, map[string]any{"level": "?"}), 0x2ecc71)
	_, comps := c.menu(lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onRename(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	vals := interaction.ModalValues(i)
	name := strings.TrimSpace(vals["name"])
	if len(name) > 32 {
		interaction.RespondError(b, i, lang, "housing.no_house")
		return
	}
	if err := c.svc.Rename(userID, name); err != nil {
		interaction.RespondError(b, i, lang, "housing.no_house")
		return
	}
	embed := components.Embed("✅", i18n.T("housing.rename_success", lang, map[string]any{"name": name}), 0x2ecc71)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onColor(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	vals := interaction.ModalValues(i)
	hex := strings.TrimSpace(vals["hex"])
	hex = strings.TrimPrefix(hex, "#")
	if _, err := strconv.ParseInt(hex, 16, 64); err != nil || len(hex) != 6 {
		interaction.RespondError(b, i, lang, "housing.no_house")
		return
	}
	if err := c.svc.SetColor(userID, hex); err != nil {
		interaction.RespondError(b, i, lang, "housing.no_house")
		return
	}
	embed := components.Embed("✅", i18n.T("housing.color_success", lang, map[string]any{"hex": "#" + hex}), 0x2ecc71)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}
