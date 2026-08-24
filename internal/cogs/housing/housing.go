package housing

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	crtsvc "guacagamblebot/internal/service/crafting"
	furnituresvc "guacagamblebot/internal/service/furniture"
	housingsvc "guacagamblebot/internal/service/housing"
	researchsvc "guacagamblebot/internal/service/research"
	sansvc "guacagamblebot/internal/service/sanctuary"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	hsvc  *housingsvc.Service
	fsvc  *furnituresvc.Service
	rsvc  *researchsvc.Service
	ssvc  *sansvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	hsvc := housingsvc.New(s, cfg)
	fsvc := furnituresvc.New(s, cfg, hsvc)
	rsvc := researchsvc.New(s, cfg, fsvc)
	ssvc := sansvc.New(s, cfg)
	c := &Cog{store: s, cfg: cfg, hsvc: hsvc, fsvc: fsvc, rsvc: rsvc, ssvc: ssvc}
	r.Slash("house", "Gère ta maison (acheter, améliorer, collecter).", c.onSlashMenu)
	r.Slash("hs", "Gère ta maison (acheter, améliorer, collecter).", c.onSlashMenu)
	r.Prefix("house", c.onPrefixMenu)
	r.Prefix("hs", c.onPrefixMenu)
	r.Component("house", "show", c.onShow)
	r.Component("house", "buy", c.onBuy)
	r.Component("house", "collect", c.onCollect)
	r.Component("house", "houses", c.onHouses)
	r.Component("house", "switch", c.onSwitch)
	r.Component("house", "shop", c.onShop)
	r.Component("house", "tree", c.onTree)
	r.Component("house", "upgrade", c.onUpgrade)
	r.Component("house", "furniture", c.onFurniture)
	r.Component("house", "place", c.onPlace)
	r.Component("house", "remove", c.onRemove)
	r.Component("house", "remove_confirm", c.onRemoveConfirm)
	r.Component("house", "research_view", c.onResearchView)
	r.Component("house", "start_research", c.onStartResearch)
	r.Component("house", "complete_research", c.onCompleteResearch)
	r.Component("house", "sanctuary", c.onSanctuary)
	r.Component("house", "sanctuary_upgrade", c.onSanctuaryUpgrade)
	r.Component("house", "sanctuary_complete", c.onSanctuaryComplete)
	r.Component("house", "rest", c.onRest)
	r.Component("house", "tree_start", c.onTreeStart)
	r.Component("house", "tree_complete", c.onTreeComplete)
	r.Modal("house", "rename", c.onRename)
	r.Modal("house", "color", c.onColor)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.menuForUser(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	embed, comps := c.menuForUser(lang, userID)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) menuForUser(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	_, err := c.hsvc.GetHousing(userID)
	if err != nil {
		return c.shopMenu(lang, userID)
	}
	embed := components.Embed(
		i18n.T("housing.menu_title", lang),
		i18n.T("housing.menu_desc", lang),
		0xB9936C,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("housing.btn_show", lang), components.EncodeOwner(userID, "house", "show"), discordgo.PrimaryButton),
			components.Button(i18n.T("housing.btn_collect", lang), components.EncodeOwner(userID, "house", "collect"), discordgo.SuccessButton),
		),
		components.ActionRow(
			components.Button(i18n.T("housing.btn_tree", lang), components.EncodeOwner(userID, "house", "tree"), discordgo.SecondaryButton),
			components.Button(i18n.T("housing.btn_upgrade", lang), components.EncodeOwner(userID, "house", "upgrade"), discordgo.PrimaryButton),
		),
		components.ActionRow(
			components.Button(i18n.T("housing.btn_furniture", lang), components.EncodeOwner(userID, "house", "furniture"), discordgo.SecondaryButton),
			components.Button(i18n.T("housing.btn_sanctuary", lang), components.EncodeOwner(userID, "house", "sanctuary"), discordgo.SuccessButton),
		),
	}
	if furnituresvc.HasFurniture(c.store, userID, "bed") {
		comps = append(comps, components.ActionRow(
			components.Button(i18n.T("housing.btn_rest", lang), components.EncodeOwner(userID, "house", "rest"), discordgo.SecondaryButton),
		))
	}
	comps = append(comps, components.ActionRow(
		components.Button(i18n.T("housing.btn_houses", lang), components.EncodeOwner(userID, "house", "houses"), discordgo.SecondaryButton),
		components.Button(i18n.T("housing.btn_shop", lang), components.EncodeOwner(userID, "house", "shop"), discordgo.PrimaryButton),
	))
	return embed, comps
}

func (c *Cog) shopMenu(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	owned := map[string]bool{}
	houses, _ := c.hsvc.ListHouses(userID)
	for _, h := range houses {
		owned[h.HouseType] = true
	}
	desc := ""
	for _, ht := range housingsvc.Houses {
		name := i18n.T("housing.types."+ht.ID, lang)
		if owned[ht.ID] {
			name = "✅ " + name + " " + i18n.T("housing.owned", lang)
		}
		desc += fmt.Sprintf("**%s** — $%d\n", name, ht.Price)
		for _, buff := range ht.Buffs {
			desc += fmt.Sprintf("  └ %s\n", buff)
		}
		desc += "\n"
	}
	embed := components.Embed(
		i18n.T("housing.shop_title", lang),
		i18n.T("housing.shop_desc", lang)+"\n\n"+desc,
		0xB9936C,
	)
	var comps []discordgo.MessageComponent
	var row []discordgo.MessageComponent
	for _, ht := range housingsvc.Houses {
		row = append(row, components.ButtonDisabled(i18n.T("housing.types."+ht.ID, lang), components.EncodeOwner(userID, "house", "buy", ht.ID), discordgo.PrimaryButton, owned[ht.ID]))
		if len(row) == 5 {
			comps = append(comps, components.ActionRow(row...))
			row = nil
		}
	}
	if len(row) > 0 {
		comps = append(comps, components.ActionRow(row...))
	}
	if len(owned) > 0 {
		comps = append(comps, components.ActionRow(
			components.Button(i18n.T("housing.btn_back", lang), components.EncodeOwner(userID, "house", "show"), discordgo.SecondaryButton),
		))
	}
	return embed, comps
}

// onShop shows the real estate agency so an existing owner can buy another house.
func (c *Cog) onShop(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.shopMenu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// onHouses lists every owned house with buttons to switch the active one.
func (c *Cog) onHouses(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	houses, err := c.hsvc.ListHouses(userID)
	if err != nil || len(houses) == 0 {
		interaction.RespondError(b, i, lang, "housing.no_house")
		return
	}
	desc := ""
	for _, h := range houses {
		houseName := h.HouseType
		if ht := housingsvc.Houses[h.HouseType]; ht != nil {
			houseName = i18n.T("housing.types."+h.HouseType, lang)
		}
		badge := ""
		if h.IsActive {
			badge = " " + i18n.T("housing.active", lang)
		}
		desc += fmt.Sprintf("%s **%s** (Lvl %d)%s\n", "🏠", houseName, h.Level, badge)
	}
	embed := components.Embed(
		i18n.T("housing.houses_title", lang),
		i18n.T("housing.houses_desc", lang)+"\n\n"+desc,
		0xB9936C,
	)
	var comps []discordgo.MessageComponent
	row := []discordgo.MessageComponent{
		components.Button(i18n.T("housing.btn_back", lang), components.EncodeOwner(userID, "house", "show"), discordgo.SecondaryButton),
	}
	for _, h := range houses {
		houseName := h.HouseType
		if ht := housingsvc.Houses[h.HouseType]; ht != nil {
			houseName = i18n.T("housing.types."+h.HouseType, lang)
		}
		row = append(row, components.ButtonDisabled("🏠 "+houseName, components.EncodeOwner(userID, "house", "switch", h.HouseType), discordgo.PrimaryButton, h.IsActive))
		if len(row) == 5 {
			comps = append(comps, components.ActionRow(row...))
			row = nil
		}
	}
	if len(row) > 0 {
		comps = append(comps, components.ActionRow(row...))
	}
	comps = append(comps, components.ActionRow(
		components.Button(i18n.T("housing.btn_shop", lang), components.EncodeOwner(userID, "house", "shop"), discordgo.PrimaryButton),
	))
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// onSwitch makes another owned house the active one.
func (c *Cog) onSwitch(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 1 {
		return
	}
	houseType := rest[0]
	if err := c.hsvc.SwitchHouse(userID, houseType); err != nil {
		interaction.RespondError(b, i, lang, "housing.no_house")
		return
	}
	houseName := i18n.T("housing.types."+houseType, lang)
	embed := components.Embed("🏠", i18n.T("housing.switch_success", lang, map[string]any{"house": houseName}), 0x2ecc71)
	_, comps := c.menuForUser(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onShow(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	h, err := c.hsvc.GetHousing(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "housing.no_house")
		return
	}
	ht := housingsvc.Houses[h.HouseType]
	if ht == nil {
		interaction.RespondError(b, i, lang, "housing.no_house")
		return
	}
	title := i18n.T("housing.title", lang, map[string]any{"user": interaction.DisplayName(b.Session, i.GuildID, i.Member, userID)})
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

	collectInfo, _ := c.hsvc.GetCollectInfo(userID)
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
	statsText := i18n.T("housing.stats", lang, map[string]any{"level": h.Level, "buffs": buffsText})
	if maxSlots := ht.SlotsAt(h.Level); maxSlots > 0 {
		statsText += "\n🪑 " + i18n.T("housing.furniture_slots", lang, map[string]any{"used": c.fsvc.GetUsedSlots(userID), "max": maxSlots})
	}
	effects := c.activeEffects(userID, lang)
	if len(effects) > 0 {
		statsText += "\n✨ **Loadout:**\n" + strings.Join(effects, "\n")
	}
	if furnituresvc.HasFurniture(c.store, userID, "bed") {
		if ok, _, err := c.store.CheckGameLimit(userID, "sleep", 1); err == nil && ok {
			statsText += "\n" + i18n.T("housing.rest_status_ready", lang)
		} else {
			statsText += "\n" + i18n.T("housing.rest_status_done", lang)
		}
	}
	embed.Fields = append(embed.Fields, components.Field(i18n.T("housing.stats_label", lang), statsText, false))

	if h.UnderConstruction != nil && *h.UnderConstruction != "" {
		uc := *h.UnderConstruction
		ucName := uc
		if upg := housingsvc.UpgradesTree[uc]; upg != nil {
			ucName = upg.Name
		}
		status := fmt.Sprintf("**%s** %s", ucName, i18n.T("housing.tree_in_progress", lang))
		if h.FinishTime != nil && time.Now().After(*h.FinishTime) {
			status = fmt.Sprintf("**%s** %s", ucName, i18n.T("housing.tree_ready", lang))
		}
		embed.Fields = append(embed.Fields, components.Field("🛠️ Construction", status, false))
	}

	_, comps := c.menuForUser(lang, userID)
	if h.UnderConstruction != nil && *h.UnderConstruction != "" && h.FinishTime != nil && time.Now().After(*h.FinishTime) {
		ucName := *h.UnderConstruction
		if upg := housingsvc.UpgradesTree[*h.UnderConstruction]; upg != nil {
			ucName = upg.Name
		}
		comps = append(comps, components.ActionRow(
			components.Button("✅ "+ucName, components.EncodeOwner(userID, "house", "tree_complete", *h.UnderConstruction), discordgo.SuccessButton),
		))
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onCollect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	income, items, err := c.hsvc.Collect(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "housing.nothing_to_collect")
		return
	}
	msg := fmt.Sprintf("💰 **Collected!** +$%d", income)
	if len(items) > 0 {
		msg += "\n📦 **Resources:** " + strings.Join(items, ", ")
	}
	embed := components.Embed("📦 Collect", msg, 0x2ecc71)
	_, comps := c.menuForUser(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onTree(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed := components.Embed(
		i18n.T("housing.tree_title", lang),
		i18n.T("housing.tree_desc", lang),
		0x1B5E20,
	)

	owned := map[string]bool{}
	var upgrades []model.UserHousingUpgrade
	c.store.DB.Where("user_id = ?", userID).Find(&upgrades)
	for _, u := range upgrades {
		owned[u.UpgradeID] = true
	}

	underConstruction := ""
	constructionReady := false
	h, herr := c.hsvc.GetHousing(userID)
	if herr == nil && h.UnderConstruction != nil && *h.UnderConstruction != "" {
		underConstruction = *h.UnderConstruction
		if h.FinishTime != nil && time.Now().After(*h.FinishTime) {
			constructionReady = true
		}
	}

	for _, upg := range housingsvc.UpgradesTree {
		itemsReq := ""
		for item, qty := range upg.CostItems {
			itemsReq += fmt.Sprintf("%dx %s ", qty, item)
		}
		status := ""
		switch {
		case owned[upg.ID]:
			status = "✅ " + i18n.T("housing.tree_owned", lang)
		case underConstruction == upg.ID && constructionReady:
			status = "✅ " + i18n.T("housing.tree_ready", lang)
		case underConstruction == upg.ID:
			status = "⏳ " + i18n.T("housing.tree_in_progress", lang)
		case upg.Requires != "" && !owned[upg.Requires]:
			req := upg.Requires
			if rd := housingsvc.UpgradesTree[req]; rd != nil {
				req = rd.Name
			}
			status = "🔒 " + i18n.T("housing.tree_requires", lang, map[string]any{"upgrade": req})
		}
		embed.Fields = append(embed.Fields, components.Field(
			fmt.Sprintf("%s (%s) %s", upg.Name, upg.Branch, status),
			fmt.Sprintf("💰 $%d\n📦 %s\n⏱ %dh\n*%s*", upg.CostMoney, itemsReq, upg.TimeHours, upg.BonusDesc),
			false,
		))
	}

	var comps []discordgo.MessageComponent
	row := []discordgo.MessageComponent{
		components.Button(i18n.T("housing.btn_back", lang), components.EncodeOwner(userID, "house", "show"), discordgo.SecondaryButton),
	}
	if constructionReady {
		name := underConstruction
		if upg := housingsvc.UpgradesTree[underConstruction]; upg != nil {
			name = upg.Name
		}
		row = append(row, components.Button("✅ "+name, components.EncodeOwner(userID, "house", "tree_complete", underConstruction), discordgo.SuccessButton))
	}
	comps = append(comps, components.ActionRow(row...))

	actionRow := []discordgo.MessageComponent{}
	for _, upg := range housingsvc.UpgradesTree {
		if owned[upg.ID] || underConstruction != "" {
			continue
		}
		if upg.Requires != "" && !owned[upg.Requires] {
			continue
		}
		actionRow = append(actionRow, components.Button("📖 "+upg.ID, components.EncodeOwner(userID, "house", "tree_start", upg.ID), discordgo.PrimaryButton))
		if len(actionRow) == 5 {
			comps = append(comps, components.ActionRow(actionRow...))
			actionRow = nil
		}
	}
	if len(actionRow) > 0 {
		comps = append(comps, components.ActionRow(actionRow...))
	}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onUpgrade(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	err := c.hsvc.UpgradeLevel(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "housing.max_level")
		return
	}
	level := "?"
	if h, herr := c.hsvc.GetHousing(userID); herr == nil {
		level = strconv.Itoa(h.Level)
	}
	embed := components.Embed("✅", i18n.T("housing.upgrade_success", lang, map[string]any{"level": level}), 0x2ecc71)
	_, comps := c.menuForUser(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onBuy(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 1 {
		return
	}
	houseType := rest[0]
	if err := c.hsvc.BuyHouse(userID, houseType); err != nil {
		var msg string
		switch {
		case errors.Is(err, housingsvc.ErrAlreadyOwned):
			msg = i18n.T("housing.already_owned", lang)
		case errors.Is(err, housingsvc.ErrNotEnoughMoney):
			price := 0
			if ht := housingsvc.Houses[houseType]; ht != nil {
				price = ht.Price
			}
			msg = i18n.T("housing.no_money", lang, map[string]any{"price": price})
		default:
			msg = i18n.T("housing.buy_failed", lang, map[string]any{"error": err.Error()})
		}
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	houseName := i18n.T("housing.types."+houseType, lang)
	embed := components.Embed("🎉", i18n.T("housing.buy_success", lang, map[string]any{"house": houseName}), 0x2ecc71)
	_, comps := c.menuForUser(lang, userID)
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
	if err := c.hsvc.Rename(userID, name); err != nil {
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
	if err := c.hsvc.SetColor(userID, hex); err != nil {
		interaction.RespondError(b, i, lang, "housing.no_house")
		return
	}
	embed := components.Embed("✅", i18n.T("housing.color_success", lang, map[string]any{"hex": "#" + hex}), 0x2ecc71)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

// --- Furniture handlers ---

func (c *Cog) onFurniture(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	h, err := c.hsvc.GetHousing(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "housing.no_house")
		return
	}
	ht := housingsvc.Houses[h.HouseType]
	if ht == nil {
		interaction.RespondError(b, i, lang, "housing.no_house")
		return
	}

	used := c.fsvc.GetUsedSlots(userID)
	maxSlots := ht.SlotsAt(h.Level)

	if maxSlots == 0 {
		houseName := i18n.T("housing.types."+h.HouseType, lang)
		embed := components.Embed(i18n.T("housing.furniture_title", lang),
			i18n.T("housing.furniture_house_none", lang, map[string]any{"house": houseName}),
			0xB9936C)
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("housing.btn_back", lang), components.EncodeOwner(userID, "house", "show"), discordgo.SecondaryButton),
				components.Button(i18n.T("housing.btn_research", lang), components.EncodeOwner(userID, "house", "research_view"), discordgo.PrimaryButton),
			),
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
		return
	}

	desc := "🪑 **" + i18n.T("housing.furniture_slots", lang, map[string]any{"used": used, "max": maxSlots}) + "**"

	placedFurniture, _ := c.fsvc.GetPlaced(userID)
	if len(placedFurniture) > 0 {
		desc += "\n\n**" + i18n.T("housing.furniture_section_placed", lang) + ":**"
		for _, pf := range placedFurniture {
			fd := furnituresvc.FurnitureDefs[pf.FurnitureID]
			if fd == nil {
				continue
			}
			effectInfo := ""
			if eff := furnitureEffect(pf.FurnitureID, lang); eff != "" {
				effectInfo = fmt.Sprintf("\n  └ ✨ %s", eff)
			}
			researchInfo := ""
			for _, rID := range fd.UnlocksResearch {
				researchInfo += fmt.Sprintf("\n  └ 🔬 %s", researchName(rID, lang))
			}
			desc += fmt.Sprintf("\n%s %s (%d slot)%s%s", fd.Emoji, furnitureName(pf.FurnitureID, lang), fd.Slots, effectInfo, researchInfo)
		}
	}

	desc += "\n\n**" + i18n.T("housing.furniture_section_available", lang) + ":**"
	for _, fd := range furnituresvc.FurnitureDefs {
		if c.fsvc.IsPlaced(userID, fd.ID) {
			continue
		}
		costStr := fmt.Sprintf("$%d", fd.CostMoney)
		for itemID, qty := range fd.CostItems {
			costStr += fmt.Sprintf(", %dx %s", qty, itemID)
		}
		effectInfo := ""
		if eff := furnitureEffect(fd.ID, lang); eff != "" {
			effectInfo = fmt.Sprintf("\n  └ ✨ %s", eff)
		}
		researchInfo := ""
		for _, rID := range fd.UnlocksResearch {
			researchInfo += fmt.Sprintf("\n  └ 🔬 %s", researchName(rID, lang))
		}
		desc += fmt.Sprintf("\n%s %s | %s (%d slot)%s%s", fd.Emoji, furnitureName(fd.ID, lang), costStr, fd.Slots, effectInfo, researchInfo)
	}

	embed := components.Embed(i18n.T("housing.furniture_title", lang), desc, 0xB9936C)

	var comps []discordgo.MessageComponent
	var row []discordgo.MessageComponent

	row = append(row, components.Button(i18n.T("housing.btn_back", lang), components.EncodeOwner(userID, "house", "show"), discordgo.SecondaryButton))
	row = append(row, components.Button(i18n.T("housing.btn_research", lang), components.EncodeOwner(userID, "house", "research_view"), discordgo.PrimaryButton))
	comps = append(comps, components.ActionRow(row...))

	// Place/Remove buttons
	actionRow := []discordgo.MessageComponent{}
	for _, pf := range placedFurniture {
		actionRow = append(actionRow, components.Button(fmt.Sprintf("❌ %s", furnitureName(pf.FurnitureID, lang)), components.EncodeOwner(userID, "house", "remove", pf.FurnitureID), discordgo.DangerButton))
		if len(actionRow) == 5 {
			comps = append(comps, components.ActionRow(actionRow...))
			actionRow = nil
		}
	}
	if len(actionRow) > 0 {
		comps = append(comps, components.ActionRow(actionRow...))
		actionRow = nil
	}

	for _, fd := range furnituresvc.FurnitureDefs {
		if c.fsvc.IsPlaced(userID, fd.ID) {
			continue
		}
		label := fmt.Sprintf("📦 %s", furnitureName(fd.ID, lang))
		actionRow = append(actionRow, components.Button(label, components.EncodeOwner(userID, "house", "place", fd.ID), discordgo.SuccessButton))
		if len(actionRow) == 5 {
			comps = append(comps, components.ActionRow(actionRow...))
			actionRow = nil
		}
	}
	if len(actionRow) > 0 {
		comps = append(comps, components.ActionRow(actionRow...))
	}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// activeEffects lists the passive effects of the furniture placed in the user's
// active house, for display in the house view.
func (c *Cog) activeEffects(userID int64, lang string) []string {
	placed, err := c.fsvc.GetPlaced(userID)
	if err != nil {
		return nil
	}
	var out []string
	for _, pf := range placed {
		if fd := furnituresvc.FurnitureDefs[pf.FurnitureID]; fd != nil {
			eff := furnitureEffect(pf.FurnitureID, lang)
			if eff == "" && len(fd.Effects) > 0 {
				eff = fd.Effects[0].Description
			}
			if eff != "" {
				out = append(out, fmt.Sprintf("%s %s", fd.Emoji, eff))
			}
		}
	}
	return out
}

// furnitureName resolves the localized furniture name, falling back to the
// English name from the furniture catalog when a locale key is missing.
func furnitureName(id string, lang string) string {
	if fd := furnituresvc.FurnitureDefs[id]; fd != nil {
		key := "housing.furnitures." + id
		if name := i18n.T(key, lang); name != key {
			return name
		}
		return fd.Name
	}
	return id
}

// furnitureEffect resolves the localized effect description for a furniture.
func furnitureEffect(id string, lang string) string {
	key := "housing.furniture_effects." + id
	if val := i18n.T(key, lang); val != key {
		return val
	}
	if fd := furnituresvc.FurnitureDefs[id]; fd != nil && len(fd.Effects) > 0 {
		return fd.Effects[0].Description
	}
	return ""
}

// researchName resolves the localized research name, falling back to the
// English name from the research catalog when a locale key is missing.
func researchName(id string, lang string) string {
	if rd := researchsvc.ResearchDefs[id]; rd != nil {
		key := "housing.researches." + id
		if name := i18n.T(key, lang); name != key {
			return name
		}
		return rd.Name
	}
	return id
}

func (c *Cog) onPlace(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 1 {
		return
	}
	furnitureID := rest[0]

	if err := c.fsvc.Place(userID, furnitureID); err != nil {
		var msg string
		houseName := furnitureID
		if h, herr := c.hsvc.GetHousing(userID); herr == nil {
			houseName = i18n.T("housing.types."+h.HouseType, lang)
		}
		switch {
		case errors.Is(err, furnituresvc.ErrNoFurnitureSlots):
			msg = i18n.T("housing.furniture_house_none", lang, map[string]any{"house": houseName})
		case strings.Contains(err.Error(), "already placed"):
			msg = i18n.T("housing.furniture_already_placed", lang)
		case strings.Contains(err.Error(), "not enough money"):
			msg = i18n.T("housing.no_money", lang, map[string]any{"price": 0})
		default:
			msg = err.Error()
		}
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	embed := components.Embed("✅", i18n.T("housing.furniture_place_success", lang, map[string]any{"name": furnitureName(furnitureID, lang)}), 0x2ecc71)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onRemove(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 1 {
		return
	}
	furnitureID := rest[0]

	fd := furnituresvc.FurnitureDefs[furnitureID]
	emoji := "❌"
	slots := 0
	if fd != nil {
		emoji = fd.Emoji
		slots = fd.Slots
	}

	embed := components.Embed(
		i18n.T("housing.furniture_remove_confirm_title", lang),
		i18n.T("housing.furniture_remove_confirm_desc", lang, map[string]any{
			"emoji": emoji, "name": furnitureName(furnitureID, lang), "slots": slots,
		}),
		0xe74c3c,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("housing.furniture_remove_confirm_btn", lang), components.EncodeOwner(userID, "house", "remove_confirm", furnitureID), discordgo.DangerButton),
			components.Button(i18n.T("housing.furniture_remove_cancel_btn", lang), components.EncodeOwner(userID, "house", "furniture"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onRemoveConfirm(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 1 {
		return
	}
	furnitureID := rest[0]

	if err := c.fsvc.Remove(userID, furnitureID); err != nil {
		embed := components.Embed("❌", err.Error(), 0xe74c3c)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
		return
	}
	embed := components.Embed("🗑️", i18n.T("housing.furniture_remove_success", lang, map[string]any{"name": furnitureName(furnitureID, lang)}), 0xe67e22)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onRest(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	if err := c.fsvc.Rest(userID); err != nil {
		var msg string
		switch {
		case errors.Is(err, furnituresvc.ErrNoBed):
			msg = i18n.T("housing.rest_no_bed", lang)
		case errors.Is(err, furnituresvc.ErrAlreadySlept):
			msg = i18n.T("housing.rest_already", lang)
		default:
			msg = err.Error()
		}
		embed := components.Embed("❌", msg, 0xe74c3c)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
		return
	}
	embed := components.Embed("🛏️", i18n.T("housing.rest_success", lang), 0x2ecc71)
	_, comps := c.menuForUser(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// --- Research handlers ---

func (c *Cog) onResearchView(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	desc := ""

	activeList, _ := c.rsvc.GetActive(userID)
	if len(activeList) > 0 {
		desc += "**⏳ " + i18n.T("housing.research_active", lang) + ":**\n"
		for _, a := range activeList {
			rName := researchName(a.ResearchID, lang)
			if a.FinishTime != nil {
				remaining := time.Until(*a.FinishTime)
				if remaining > 0 {
					h := int(remaining.Hours())
					m := int(remaining.Minutes()) % 60
					desc += fmt.Sprintf("• %s — ⏱ %s\n", rName, i18n.T("housing.research_time_remaining", lang, map[string]any{"h": h, "m": m}))
				} else {
					desc += fmt.Sprintf("• %s — ✅ **%s**\n", rName, i18n.T("housing.research_ready", lang))
				}
			}
		}
		desc += "\n"
	}

	completed, _ := c.rsvc.GetCompleted(userID)
	if len(completed) > 0 {
		desc += "**✅ " + i18n.T("housing.research_completed", lang) + ":**\n"
		for _, co := range completed {
			rName := researchName(co.ResearchID, lang)
			desc += fmt.Sprintf("• %s\n", rName)
		}
		desc += "\n"
	}

	availCount := 0
	desc += "**📖 " + i18n.T("housing.research_available", lang) + ":**\n"
	for _, rd := range researchsvc.ResearchDefs {
		if c.rsvc.IsCompleted(userID, rd.ID) {
			continue
		}
		isActive := false
		for _, a := range activeList {
			if a.ResearchID == rd.ID {
				isActive = true
				break
			}
		}
		if isActive {
			continue
		}

		if c.fsvc.IsPlaced(userID, rd.RequiredFurniture) {
			costStr := fmt.Sprintf("$%d", rd.CostMoney)
			for itemID, qty := range rd.CostItems {
				costStr += fmt.Sprintf(", %dx %s", qty, itemID)
			}
			desc += fmt.Sprintf("• %s (%dh) — %s\n", researchName(rd.ID, lang), rd.TimeHours, costStr)
			availCount++
		}
	}
	if availCount == 0 {
		desc += "*(" + i18n.T("housing.research_empty_available", lang) + ")*\n"
	}

	desc += "\n**🔒 " + i18n.T("housing.research_locked", lang) + ":**\n"
	lockedCount := 0
	for _, rd := range researchsvc.ResearchDefs {
		if c.fsvc.IsPlaced(userID, rd.RequiredFurniture) || c.rsvc.IsCompleted(userID, rd.ID) {
			continue
		}
		isActive := false
		for _, a := range activeList {
			if a.ResearchID == rd.ID {
				isActive = true
				break
			}
		}
		if isActive {
			continue
		}
		desc += fmt.Sprintf("• %s → %s\n", researchName(rd.ID, lang), i18n.T("housing.research_needs_furniture", lang, map[string]any{"furniture": furnitureName(rd.RequiredFurniture, lang)}))
		lockedCount++
	}
	if lockedCount == 0 {
		desc += "*(" + i18n.T("housing.research_empty_locked", lang) + ")*"
	}

	embed := components.Embed(i18n.T("housing.research_title", lang), desc, 0x1B5E20)

	var comps []discordgo.MessageComponent
	var row []discordgo.MessageComponent
	row = append(row, components.Button(i18n.T("housing.btn_back", lang), components.EncodeOwner(userID, "house", "furniture"), discordgo.SecondaryButton))

	// Active ready-to-complete buttons
	for _, a := range activeList {
		if a.FinishTime != nil && time.Now().After(*a.FinishTime) {
			label := fmt.Sprintf("✅ %s", researchName(a.ResearchID, lang))
			row = append(row, components.Button(label, components.EncodeOwner(userID, "house", "complete_research", a.ResearchID), discordgo.SuccessButton))
		}
	}
	comps = append(comps, components.ActionRow(row...))

	// Start research buttons
	actionRow := []discordgo.MessageComponent{}
	for _, rd := range researchsvc.ResearchDefs {
		if c.rsvc.IsCompleted(userID, rd.ID) {
			continue
		}
		isActive := false
		for _, a := range activeList {
			if a.ResearchID == rd.ID {
				isActive = true
				break
			}
		}
		if isActive {
			continue
		}
		if c.fsvc.IsPlaced(userID, rd.RequiredFurniture) {
			label := fmt.Sprintf("📖 %s", researchName(rd.ID, lang))
			actionRow = append(actionRow, components.Button(label, components.EncodeOwner(userID, "house", "start_research", rd.ID), discordgo.PrimaryButton))
			if len(actionRow) == 5 {
				comps = append(comps, components.ActionRow(actionRow...))
				actionRow = nil
			}
		}
	}
	if len(actionRow) > 0 {
		comps = append(comps, components.ActionRow(actionRow...))
	}

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onStartResearch(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 1 {
		return
	}
	researchID := rest[0]

	if err := c.rsvc.Start(userID, researchID); err != nil {
		embed := components.Embed("❌", err.Error(), 0xe74c3c)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
		return
	}
	rd := researchsvc.ResearchDefs[researchID]
	rName := researchName(researchID, lang)
	timeHours := 0
	if rd != nil {
		timeHours = rd.TimeHours
	}
	embed := components.Embed("🔬", i18n.T("housing.research_started", lang, map[string]any{"name": rName, "time": timeHours}), 0x2ecc71)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onCompleteResearch(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 1 {
		return
	}
	researchID := rest[0]

	if err := c.rsvc.Complete(userID, researchID); err != nil {
		embed := components.Embed("❌", err.Error(), 0xe74c3c)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
		return
	}
	rd := researchsvc.ResearchDefs[researchID]
	rName := researchName(researchID, lang)
	msg := i18n.T("housing.research_completed_msg", lang, map[string]any{"name": rName})
	if rd != nil && len(rd.UnlocksRecipes) > 0 {
		names := make([]string, 0, len(rd.UnlocksRecipes))
		for _, key := range rd.UnlocksRecipes {
			if recipe, ok := crtsvc.Recipes[key]; ok {
				names = append(names, items.LocalizedName(recipe.Result, lang))
			}
		}
		if len(names) > 0 {
			msg += " " + strings.Join(names, ", ")
		}
	}
	embed := components.Embed("✅", msg, 0x2ecc71)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onTreeStart(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	if len(rest) < 1 {
		return
	}
	upgradeID := rest[0]

	if err := c.hsvc.StartConstruction(userID, upgradeID); err != nil {
		var msg string
		switch {
		case strings.Contains(err.Error(), "not enough money"):
			msg = i18n.T("housing.no_money", lang, map[string]any{"price": 0})
		case strings.Contains(err.Error(), "requires"):
			req := upgradeID
			if upg := housingsvc.UpgradesTree[upgradeID]; upg != nil {
				req = upg.Requires
			}
			if rd := housingsvc.UpgradesTree[req]; rd != nil {
				req = rd.Name
			}
			msg = i18n.T("housing.tree_requires", lang, map[string]any{"upgrade": req})
		case strings.Contains(err.Error(), "already owned"):
			msg = i18n.T("housing.tree_already", lang)
		case strings.Contains(err.Error(), "in progress"):
			msg = i18n.T("housing.tree_in_progress", lang)
		default:
			msg = err.Error()
		}
		embed := components.Embed("❌", msg, 0xe74c3c)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
		return
	}
	name := upgradeID
	hours := 0
	if upg := housingsvc.UpgradesTree[upgradeID]; upg != nil {
		name = upg.Name
		hours = upg.TimeHours
	}
	embed := components.Embed("🏗️", i18n.T("housing.tree_started", lang, map[string]any{"name": name, "hours": hours}), 0x2ecc71)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onTreeComplete(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cid := i.MessageComponentData().CustomID
	_, _, rest := components.Decode(cid)
	upgradeID := ""
	if len(rest) > 0 {
		upgradeID = rest[0]
	}

	if err := c.hsvc.CompleteConstruction(userID); err != nil {
		var msg string
		switch {
		case strings.Contains(err.Error(), "not finished"):
			msg = i18n.T("housing.tree_not_ready", lang)
		case strings.Contains(err.Error(), "no construction"):
			msg = i18n.T("housing.tree_no_construction", lang)
		default:
			msg = err.Error()
		}
		embed := components.Embed("❌", msg, 0xe74c3c)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
		return
	}
	name := upgradeID
	if upg := housingsvc.UpgradesTree[upgradeID]; upg != nil {
		name = upg.Name
	}
	embed := components.Embed("✅", i18n.T("housing.tree_completed", lang, map[string]any{"name": name}), 0x2ecc71)
	_, comps := c.menuForUser(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onSanctuary(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.buildHouseSanctuaryEmbed(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) buildHouseSanctuaryEmbed(userID int64, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	san, _ := c.ssvc.GetOrCreateSanctuary(userID)
	tier := san.Tier
	used := c.ssvc.GetUsedSlots(userID)
	max := c.ssvc.GetMaxSlots(userID)
	tierName := "Not Built"
	if t, ok := sansvc.SanctuaryTiers[tier]; ok {
		tierName = t.Name
	}
	houseMaxTier := c.ssvc.GetHouseMaxTier(userID)
	houseMaxSlots := c.ssvc.GetHouseMaxSlots(userID)
	houseName := "No House"
	if h, err := c.hsvc.GetHousing(userID); err == nil {
		if ht := housingsvc.Houses[h.HouseType]; ht != nil {
			houseName = i18n.T("housing.types."+h.HouseType, lang)
		}
	}
	desc := fmt.Sprintf("**Tier:** %s (Lvl %d)\n**Capacity:** %d/%d pets\n", tierName, tier, used, max)
	if houseMaxTier > 0 {
		desc += fmt.Sprintf("**House Limit:** %s → max Tier %d (%d slots)\n", houseName, houseMaxTier, houseMaxSlots)
	} else {
		desc += fmt.Sprintf("**House Limit:** %s → no sanctuary allowed (buy a house)\n", houseName)
	}
	if san.UnderConstruction != nil {
		remaining := ""
		if san.FinishTime != nil {
			d := time.Until(*san.FinishTime)
			if d > 0 {
				remaining = fmt.Sprintf(" (%dh remaining)", int(d.Hours()))
			} else {
				remaining = " (ready to complete!)"
			}
		}
		desc += fmt.Sprintf("\n🔧 **Upgrading** to %s%s\n", sansvc.SanctuaryTiers[tier+1].Name, remaining)
	} else if tier < houseMaxTier {
		if next, ok := sansvc.SanctuaryTiers[tier+1]; ok {
			mats := ""
			for item, qty := range next.Materials {
				if mats != "" {
					mats += ", "
				}
				mats += fmt.Sprintf("%dx %s", qty, item)
			}
			desc += fmt.Sprintf("\n**Next:** %s — $%d + %s (%dh)\n", next.Name, next.Price, mats, next.BuildHours)
		}
	} else if tier >= houseMaxTier && houseMaxTier > 0 {
		desc += "\n*Max tier for your house reached — buy a bigger house to expand further.*\n"
	}
	desc += "\nUse buttons below to manage sanctuary."

	embed := components.Embed("🏡 Pet Sanctuary", desc, 0x2ecc71)

	var comps []discordgo.MessageComponent
	var buttons []discordgo.MessageComponent
	// Upgrade / Complete button
	if san.UnderConstruction != nil {
		if san.FinishTime != nil && time.Now().After(*san.FinishTime) {
			buttons = append(buttons, components.Button("✅ Complete", components.EncodeOwner(userID, "house", "sanctuary_complete"), discordgo.SuccessButton))
		}
	} else if tier < houseMaxTier {
		if _, ok := sansvc.SanctuaryTiers[tier+1]; ok {
			buttons = append(buttons, components.Button("⬆️ Upgrade Sanctuary", components.EncodeOwner(userID, "house", "sanctuary_upgrade"), discordgo.SuccessButton))
		}
	}
	if len(buttons) > 0 {
		comps = append(comps, components.ActionRow(buttons...))
	}
	// Link to detailed sanctuary cog
	comps = append(comps, components.ActionRow(
		components.Button(i18n.T("housing.btn_back", lang), components.EncodeOwner(userID, "house", "show"), discordgo.SecondaryButton),
	))
	return embed, comps
}

func (c *Cog) onSanctuaryUpgrade(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	san, _ := c.ssvc.GetOrCreateSanctuary(userID)
	nextTier := san.Tier + 1
	if nextTier == 1 && san.Tier == 0 && c.ssvc.GetHouseMaxTier(userID) == 0 {
		interaction.RespondError(b, i, lang, "housing.no_house")
		return
	}
	if err := c.ssvc.StartConstruction(userID, nextTier); err != nil {
		msg := err.Error()
		if errors.Is(err, sansvc.ErrHouseSanctuaryCap) {
			msg = i18n.T("housing.sanctuary_house_max", lang)
		}
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ " + msg, Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	embed, comps := c.buildHouseSanctuaryEmbed(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onSanctuaryComplete(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	if err := c.ssvc.CompleteConstruction(userID); err != nil {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ " + err.Error(), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	embed, comps := c.buildHouseSanctuaryEmbed(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}
