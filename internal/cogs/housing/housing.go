package housing

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
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
	r.Component("house", "tree", c.onTree)
	r.Component("house", "upgrade", c.onUpgrade)
	r.Component("house", "furniture", c.onFurniture)
	r.Component("house", "place", c.onPlace)
	r.Component("house", "remove", c.onRemove)
	r.Component("house", "research_view", c.onResearchView)
	r.Component("house", "start_research", c.onStartResearch)
	r.Component("house", "complete_research", c.onCompleteResearch)
	r.Component("house", "sanctuary", c.onSanctuary)
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
		return c.shopMenu(lang)
	}
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
		components.ActionRow(
			components.Button("🪑 Furniture", components.Encode("house", "furniture"), discordgo.SecondaryButton),
			components.Button("🏡 Sanctuary", components.Encode("house", "sanctuary"), discordgo.SuccessButton),
		),
	}
	return embed, comps
}

func (c *Cog) shopMenu(lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	desc := ""
	for _, ht := range housingsvc.Houses {
		desc += fmt.Sprintf("**%s** — $%d\n", i18n.T("housing.types."+ht.ID, lang), ht.Price)
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
		row = append(row, components.Button(i18n.T("housing.types."+ht.ID, lang), components.Encode("house", "buy", ht.ID), discordgo.PrimaryButton))
		if len(row) == 5 {
			comps = append(comps, components.ActionRow(row...))
			row = nil
		}
	}
	if len(row) > 0 {
		comps = append(comps, components.ActionRow(row...))
	}
	return embed, comps
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
	furnitureSlots := fmt.Sprintf("\n🪑 Furniture: %d/%d slots", c.fsvc.GetUsedSlots(userID), ht.FurnitureSlots)
	embed.Fields = append(embed.Fields, components.Field(i18n.T("housing.stats_label", lang),
		i18n.T("housing.stats", lang, map[string]any{"level": h.Level, "buffs": buffsText})+furnitureSlots, false))

	if h.UnderConstruction != nil && *h.UnderConstruction != "" {
		embed.Fields = append(embed.Fields, components.Field("🛠️ Construction",
			fmt.Sprintf("**%s** en cours...", *h.UnderConstruction), false))
	}

	_, comps := c.menuForUser(lang, userID)
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
	_, comps := c.menuForUser(lang, userID)
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
	embed := components.Embed("✅", i18n.T("housing.upgrade_success", lang, map[string]any{"level": "?"}), 0x2ecc71)
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
		price := 0
		if ht := housingsvc.Houses[houseType]; ht != nil {
			price = ht.Price
		}
		msg := i18n.T("housing.no_money", lang, map[string]any{"price": price})
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
	maxSlots := ht.FurnitureSlots

	desc := fmt.Sprintf("🪑 **Slots: %d/%d**", used, maxSlots)

	placedFurniture, _ := c.fsvc.GetPlaced(userID)
	if len(placedFurniture) > 0 {
		desc += "\n\n**Placed:**"
		for _, pf := range placedFurniture {
			fd := furnituresvc.FurnitureDefs[pf.FurnitureID]
			if fd == nil {
				continue
			}
			researchInfo := ""
			for _, rID := range fd.UnlocksResearch {
				if rd := researchsvc.ResearchDefs[rID]; rd != nil {
					researchInfo += fmt.Sprintf("\n  └ 🔬 %s", rd.Name)
				}
			}
			desc += fmt.Sprintf("\n%s %s (%s)%s", fd.Emoji, fd.Name, fd.SlotType, researchInfo)
		}
	}

	desc += "\n\n**Available:**"
	for _, fd := range furnituresvc.FurnitureDefs {
		if c.fsvc.IsPlaced(userID, fd.ID) {
			continue
		}
		costStr := fmt.Sprintf("$%d", fd.CostMoney)
		for itemID, qty := range fd.CostItems {
			costStr += fmt.Sprintf(", %dx %s", qty, itemID)
		}
		researchInfo := ""
		for _, rID := range fd.UnlocksResearch {
			if rd := researchsvc.ResearchDefs[rID]; rd != nil {
				researchInfo += fmt.Sprintf(" → 🔬 %s", rd.Name)
			}
		}
		desc += fmt.Sprintf("\n%s %s | %s (%s)%s", fd.Emoji, fd.Name, costStr, fd.SlotType, researchInfo)
	}

	embed := components.Embed("🪑 Furnitures", desc, 0xB9936C)

	var comps []discordgo.MessageComponent
	var row []discordgo.MessageComponent

	row = append(row, components.Button("🔙 Back", components.Encode("house", "show"), discordgo.SecondaryButton))
	row = append(row, components.Button("🔬 Research", components.Encode("house", "research_view"), discordgo.PrimaryButton))
	comps = append(comps, components.ActionRow(row...))

	// Place/Remove buttons
	actionRow := []discordgo.MessageComponent{}
	for _, pf := range placedFurniture {
		actionRow = append(actionRow, components.Button(fmt.Sprintf("❌ %s", pf.FurnitureID), components.Encode("house", "remove", pf.FurnitureID), discordgo.DangerButton))
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
		label := fmt.Sprintf("📦 %s", fd.ID)
		actionRow = append(actionRow, components.Button(label, components.Encode("house", "place", fd.ID), discordgo.SuccessButton))
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
		interaction.RespondError(b, i, lang, "housing.no_house")
		return
	}
	fd := furnituresvc.FurnitureDefs[furnitureID]
	name := furnitureID
	if fd != nil {
		name = fd.Name
	}
	embed := components.Embed("✅", fmt.Sprintf("Placed **%s** in your house!", name), 0x2ecc71)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onRemove(b *interaction.Bot, i *discordgo.InteractionCreate) {
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
	fd := furnituresvc.FurnitureDefs[furnitureID]
	name := furnitureID
	if fd != nil {
		name = fd.Name
	}
	embed := components.Embed("🗑️", fmt.Sprintf("Removed **%s** from your house.", name), 0xe67e22)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

// --- Research handlers ---

func (c *Cog) onResearchView(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(interaction.UserID(i))

	desc := ""

	activeList, _ := c.rsvc.GetActive(userID)
	if len(activeList) > 0 {
		desc += "**⏳ Active Research:**\n"
		for _, a := range activeList {
			rd := researchsvc.ResearchDefs[a.ResearchID]
			rName := a.ResearchID
			if rd != nil {
				rName = rd.Name
			}
			if a.FinishTime != nil {
				remaining := time.Until(*a.FinishTime)
				if remaining > 0 {
					h := int(remaining.Hours())
					m := int(remaining.Minutes()) % 60
					desc += fmt.Sprintf("• %s — ⏱ %dh %dm remaining\n", rName, h, m)
				} else {
					desc += fmt.Sprintf("• %s — ✅ **Ready to complete!**\n", rName)
				}
			}
		}
		desc += "\n"
	}

	completed, _ := c.rsvc.GetCompleted(userID)
	if len(completed) > 0 {
		desc += "**✅ Completed Research:**\n"
		for _, co := range completed {
			rd := researchsvc.ResearchDefs[co.ResearchID]
			rName := co.ResearchID
			if rd != nil {
				rName = rd.Name
			}
			desc += fmt.Sprintf("• %s\n", rName)
		}
		desc += "\n"
	}

	availCount := 0
	desc += "**📖 Available Research:**\n"
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
			desc += fmt.Sprintf("• %s (%dh) — %s\n", rd.Name, rd.TimeHours, costStr)
			availCount++
		}
	}
	if availCount == 0 {
		desc += "*(Place furniture to unlock research)*\n"
	}

	desc += "\n**🔒 Locked Research (place furniture first):**\n"
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
		fd := furnituresvc.FurnitureDefs[rd.RequiredFurniture]
		fName := rd.RequiredFurniture
		if fd != nil {
			fName = fd.Name
		}
		desc += fmt.Sprintf("• %s → need **%s**\n", rd.Name, fName)
		lockedCount++
	}
	if lockedCount == 0 {
		desc += "*(All research unlocked or in progress)*"
	}

	embed := components.Embed("🔬 Research Overview", desc, 0x1B5E20)

	var comps []discordgo.MessageComponent
	var row []discordgo.MessageComponent
	row = append(row, components.Button("🔙 Back", components.Encode("house", "furniture"), discordgo.SecondaryButton))

	// Active ready-to-complete buttons
	for _, a := range activeList {
		if a.FinishTime != nil && time.Now().After(*a.FinishTime) {
			rd := researchsvc.ResearchDefs[a.ResearchID]
			label := fmt.Sprintf("✅ %s", a.ResearchID)
			if rd != nil {
				label = fmt.Sprintf("✅ %s", rd.Name)
			}
			row = append(row, components.Button(label, components.Encode("house", "complete_research", a.ResearchID), discordgo.SuccessButton))
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
			label := fmt.Sprintf("📖 %s", rd.ID)
			actionRow = append(actionRow, components.Button(label, components.Encode("house", "start_research", rd.ID), discordgo.PrimaryButton))
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
	rName := researchID
	timeStr := ""
	if rd != nil {
		rName = rd.Name
		timeStr = fmt.Sprintf(" (%dh)", rd.TimeHours)
	}
	embed := components.Embed("🔬", fmt.Sprintf("Started research **%s**%s!", rName, timeStr), 0x2ecc71)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onCompleteResearch(b *interaction.Bot, i *discordgo.InteractionCreate) {
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
	rName := researchID
	if rd != nil {
		rName = rd.Name
	}
	embed := components.Embed("✅", fmt.Sprintf("Completed research **%s**! New recipes unlocked!", rName), 0x2ecc71)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) onSanctuary(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(interaction.UserID(i))
	san, _ := c.ssvc.GetOrCreateSanctuary(userID)
	tier := san.Tier
	used := c.ssvc.GetUsedSlots(userID)
	max := c.ssvc.GetMaxSlots(userID)
	tierName := "Not Built"
	if t, ok := sansvc.SanctuaryTiers[tier]; ok {
		tierName = t.Name
	}
	desc := fmt.Sprintf("**Tier:** %s (Lvl %d)\n**Capacity:** %d/%d pets\n\nUse `/sanctuary` to manage your sanctuary in detail!", tierName, tier, used, max)
	embed := components.Embed("🏡 Pet Sanctuary", desc, 0x2ecc71)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}
