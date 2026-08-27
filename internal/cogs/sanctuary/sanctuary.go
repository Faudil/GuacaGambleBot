package sanctuary

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
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	ps "guacagamblebot/internal/service/pets"
	sansvc "guacagamblebot/internal/service/sanctuary"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *sansvc.Service
	psvc  *ps.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	psvc := ps.New(s, cfg)
	c := &Cog{store: s, cfg: cfg, svc: sansvc.New(s, cfg, psvc), psvc: psvc}
	r.SlashWithOptions("sanctuary", "Manage or visit your pet sanctuary", []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        "user",
			Description: "Visit another player's sanctuary",
			Required:    false,
		},
	}, c.onSlash)
	r.Prefix("sanctuary", c.onPrefix)
	r.Component("sanctuary", "view", c.onView)
	r.Component("sanctuary", "retire", c.onRetireSelect)
	r.Component("sanctuary", "retire_pick", c.onRetirePick)
	r.Component("sanctuary", "recall", c.onRecallSelect)
	r.Component("sanctuary", "recall_pick", c.onRecallPick)
	r.Component("sanctuary", "pet_page", c.onPetPage)
	r.Component("sanctuary", "pet_search_open", c.onPetSearchOpen)
	r.Modal("sanctuary", "pet_search_submit", c.onPetSearchSubmit)
	r.Component("sanctuary", "collect", c.onCollect)
	r.Component("sanctuary", "upgrade", c.onUpgrade)
	r.Component("sanctuary", "complete", c.onComplete)
	r.Component("sanctuary", "showcase", c.onShowcaseSelect)
	r.Component("sanctuary", "showcase_select", c.onShowcaseSet)
	r.Component("sanctuary", "showcase_slot", c.onShowcaseAssign)
	r.Component("sanctuary", "fusion", c.onFusionMenu)
	r.Component("sanctuary", "fusion_rarity", c.onFusionRarity)
	r.Component("sanctuary", "fusion_pick", c.onFusionPick)
	r.Component("sanctuary", "fusion_confirm", c.onFusionConfirm)
	r.Component("sanctuary", "ascend", c.onAscendMenu)
	r.Component("sanctuary", "ascend_pick", c.onAscendPick)
}

func (c *Cog) onSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	targetID := int64(0)
	if len(i.ApplicationCommandData().Options) > 0 {
		user := i.ApplicationCommandData().Options[0].UserValue(nil)
		if user != nil {
			targetID = interaction.ToInt64(user.ID)
		}
	}
	if targetID == 0 {
		c.showOwnSanctuary(b, i, lang)
	} else {
		c.visitSanctuary(b, i, targetID, lang)
	}
}

func (c *Cog) onPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	embed, comps := c.buildSanctuaryEmbed(userID, lang)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) showOwnSanctuary(b *interaction.Bot, i *discordgo.InteractionCreate, lang string) {
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.buildSanctuaryEmbed(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) showOwnSanctuaryMessage(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message, userID int64, lang string) {
	embed, comps := c.buildSanctuaryEmbed(userID, lang)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) buildSanctuaryEmbed(userID int64, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	san, _ := c.svc.GetOrCreateSanctuary(userID)
	tier, used, max, _ := c.svc.GetSanctuaryInfo(userID)

	tierName := "Not Built"
	if t, ok := sansvc.SanctuaryTiers[tier]; ok {
		tierName = t.Name
	}
	biomeProgress := c.collectionProgress(userID)
	collectionStr := ""
	for _, biome := range ps.Biomes {
		info := biomeProgress[biome]
		bar := buildProgressBar(info.Have, info.Total, 10)
		biomeName := i18n.T("biomes."+biome, lang)
		collectionStr += fmt.Sprintf("%s %s: %s %d/%d\n", biomeEmoji(biome), biomeName, bar, info.Have, info.Total)
	}

	var statusLine string
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
		statusLine = fmt.Sprintf("🔧 **Upgrading** to %s%s", sansvc.SanctuaryTiers[tier+1].Name, remaining)
	}

	desc := fmt.Sprintf("**Tier:** %s (Lvl %d)\n**Capacity:** %d/%d pets\n\n", tierName, tier, used, max)
	if statusLine != "" {
		desc += statusLine + "\n\n"
	}
	desc += "**📊 Collection Progress:**\n" + collectionStr

	nextTier := ""
	if t, ok := sansvc.SanctuaryTiers[tier+1]; ok && san.UnderConstruction == nil {
		nextTier = fmt.Sprintf("\n\n**Next:** %s ($%d, %dh)", t.Name, t.Price, t.BuildHours)
	}

	embed := components.Embed(
		"🏡 **Pet Sanctuary**",
		desc+nextTier,
		0x2ecc71,
	)
	return embed, c.buildButtons(san, tier, used, max, lang, userID)
}

func (c *Cog) buildButtons(san *model.UserSanctuary, tier, used, max int, lang string, userID int64) []discordgo.MessageComponent {
	var buttons []discordgo.MessageComponent

	if tier > 0 && san.UnderConstruction == nil {
		buttons = append(buttons,
			components.Button("📦 Collect", components.EncodeOwner(userID, "sanctuary", "collect"), discordgo.SuccessButton),
		)
	}

	if used > 0 {
		buttons = append(buttons,
			components.Button("🔙 Recall", components.EncodeOwner(userID, "sanctuary", "recall"), discordgo.PrimaryButton),
		)
	}

	if used < max && tier > 0 {
		buttons = append(buttons,
			components.Button("🏞️ Retire", components.EncodeOwner(userID, "sanctuary", "retire"), discordgo.SecondaryButton),
		)
	}

	if san.UnderConstruction != nil {
		if san.FinishTime != nil && time.Now().After(*san.FinishTime) {
			buttons = append(buttons,
				components.Button("✅ Complete", components.EncodeOwner(userID, "sanctuary", "complete"), discordgo.SuccessButton),
			)
		}
	} else if _, ok := sansvc.SanctuaryTiers[tier+1]; ok {
		buttons = append(buttons,
			components.Button("⬆️ Upgrade", components.EncodeOwner(userID, "sanctuary", "upgrade"), discordgo.PrimaryButton),
		)
	}

	if used > 0 {
		buttons = append(buttons,
			components.Button("⭐ Showcase", components.EncodeOwner(userID, "sanctuary", "showcase"), discordgo.SecondaryButton),
		)
	}

	if tier > 0 {
		buttons = append(buttons,
			components.Button("⚗️ Fusion", components.EncodeOwner(userID, "sanctuary", "fusion"), discordgo.PrimaryButton),
		)
	}
	if tier >= 2 {
		buttons = append(buttons,
			components.Button("👑 Ascend", components.EncodeOwner(userID, "sanctuary", "ascend"), discordgo.SuccessButton),
		)
	}

	if len(buttons) == 0 {
		if tier == 0 {
			buttons = append(buttons,
				components.Button("🏗️ Build", components.EncodeOwner(userID, "sanctuary", "upgrade"), discordgo.SuccessButton),
			)
		} else {
			buttons = append(buttons,
				components.Button("🔄 Refresh", components.EncodeOwner(userID, "sanctuary", "view"), discordgo.SecondaryButton),
			)
		}
	}

	// Discord caps an action row at 5 buttons; split into multiple rows so a
	// fully-built sanctuary (which can offer 7 actions at once) doesn't blow
	// the limit and get the whole interaction response rejected.
	var rows []discordgo.MessageComponent
	for len(buttons) > 0 {
		n := 5
		if n > len(buttons) {
			n = len(buttons)
		}
		rows = append(rows, components.ActionRow(buttons[:n]...))
		buttons = buttons[n:]
	}
	return rows
}

type biomeCollectionInfo struct {
	Have  int
	Total int
}

func (c *Cog) collectionProgress(userID int64) map[string]biomeCollectionInfo {
	progress := make(map[string]biomeCollectionInfo)
	biomePets := make(map[string]map[string]bool)
	for name, pt := range ps.PetTypes {
		if biomePets[pt.Biome] == nil {
			biomePets[pt.Biome] = make(map[string]bool)
		}
		biomePets[pt.Biome][name] = true
	}
	var userPets []model.UserPet
	c.store.DB.Where("user_id = ?", userID).Find(&userPets)
	owned := make(map[string]bool)
	for _, p := range userPets {
		owned[p.PetType] = true
	}
	for biome, pets := range biomePets {
		have := 0
		for name := range pets {
			if owned[name] {
				have++
			}
		}
		progress[biome] = biomeCollectionInfo{Have: have, Total: len(pets)}
	}
	return progress
}

func (c *Cog) onView(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.buildSanctuaryEmbed(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onCollect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	loot, count, err := c.svc.CollectResources(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "sanctuary.collect_error")
		return
	}
	if count == 0 {
		interaction.RespondError(b, i, lang, "sanctuary.collect_empty")
		return
	}
	names := make([]string, 0, len(loot))
	for _, id := range loot {
		names = append(names, items.LocalizedName(id, lang))
	}
	itemList := joinUnique(names)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("sanctuary.collect_success", lang, map[string]any{"count": count, "items": itemList}),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) onUpgrade(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	san, _ := c.svc.GetOrCreateSanctuary(userID)
	targetTier := san.Tier + 1
	t, ok := sansvc.SanctuaryTiers[targetTier]
	if !ok {
		interaction.RespondError(b, i, lang, "sanctuary.max_tier")
		return
	}
	err := c.svc.StartConstruction(userID, targetTier)
	if err != nil {
		interaction.RespondError(b, i, lang, "sanctuary.upgrade_error")
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("sanctuary.upgrade_started", lang, map[string]any{"name": t.Name, "hours": t.BuildHours}),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) onComplete(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	err := c.svc.CompleteConstruction(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "sanctuary.complete_error")
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("sanctuary.complete_success", lang),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// ── Fusion (TradeUp) & Ascendancy ─────────────────────────────────────────

func (c *Cog) onFusionMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(interaction.UserID(i))
	desc := "**Fusion Lab** — Trade pets of same rarity for a higher one (instant).\n"
	for _, rarity := range []string{ps.RarityCommon, ps.RarityRare, ps.RarityEpic} {
		req, target := ps.TradeUpRarity(rarity)
		cost := sansvc.TradeUpCosts[rarity]
		mats := ""
		for k, v := range cost.Items {
			if mats != "" {
				mats += ", "
			}
			mats += fmt.Sprintf("%dx %s", v, k)
		}
		desc += fmt.Sprintf("\n**%s → %s** : %d pets + $%d + %s", rarity, target, req, cost.Money, mats)
	}
	embed := components.Embed("⚗️ Fusion Lab", desc, 0x9b59b6)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("Common→Rare (5)", components.EncodeOwner(userID, "sanctuary", "fusion_rarity", ps.RarityCommon), discordgo.PrimaryButton),
			components.Button("Rare→Epic (4)", components.EncodeOwner(userID, "sanctuary", "fusion_rarity", ps.RarityRare), discordgo.PrimaryButton),
			components.Button("Epic→Legendary (3)", components.EncodeOwner(userID, "sanctuary", "fusion_rarity", ps.RarityEpic), discordgo.DangerButton),
		),
		components.ActionRow(
			components.Button("🔙 Back", components.EncodeOwner(userID, "sanctuary", "view"), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onFusionRarity(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	rarity := rest[0]
	req, _ := ps.TradeUpRarity(rarity)
	if req == 0 {
		return
	}
	// Research gate
	if rid, ok := sansvc.TradeUpResearch[rarity]; ok {
		var r model.UserResearch
		if err := c.store.DB.Where("user_id = ? AND research_id = ? AND completed = ?", userID, rid, true).First(&r).Error; err != nil {
			_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{Content: "❌ Research required: " + rid, Flags: discordgo.MessageFlagsEphemeral},
			})
			return
		}
	}
	pets, _ := c.psvc.GetPets(userID)
	opts := []discordgo.SelectMenuOption{}
	for _, p := range pets {
		pt := ps.PetTypes[p.PetType]
		if pt == nil || pt.Rarity != rarity {
			continue
		}
		if p.IsActive || p.OnExpedition {
			continue
		}
		label := p.Nickname + " (" + p.PetType + ")"
		if len([]rune(label)) > 50 {
			label = string([]rune(label)[:50])
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label: label,
			Value: strconv.FormatInt(p.ID, 10),
			Emoji: &discordgo.ComponentEmoji{Name: pt.Emoji},
		})
		if len(opts) >= 25 {
			break
		}
	}
	if len(opts) < req {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("❌ Need %d %s pets (have eligible %d)", req, rarity, len(opts)), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	embed := components.Embed("⚗️ Select pets", fmt.Sprintf("Select exactly **%d %s** pets to fuse.", req, rarity), 0x9b59b6)
	sel := discordgo.SelectMenu{
		CustomID:    components.EncodeOwner(userID, "sanctuary", "fusion_pick", rarity),
		Placeholder: fmt.Sprintf("Pick %d pets", req),
		Options:     opts,
		MinValues:   &req,
		MaxValues:   req,
	}
	_ = b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, []discordgo.MessageComponent{components.ActionRow(sel)}))
}

func (c *Cog) onFusionPick(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	data := i.MessageComponentData()
	_, _, rest := components.Decode(data.CustomID)
	if len(rest) == 0 {
		return
	}
	rarity := rest[0]
	ids := data.Values
	req, target := ps.TradeUpRarity(rarity)
	if len(ids) != req {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("❌ Need %d pets", req), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	cost := sansvc.TradeUpCosts[rarity]
	mats := ""
	for k, v := range cost.Items {
		if mats != "" {
			mats += ", "
		}
		mats += fmt.Sprintf("%dx %s", v, k)
	}
	embed := components.Embed("⚗️ Confirm Fusion", fmt.Sprintf("%d %s → 1 %s\nCost: $%d + %s\nRandom species of %s", req, rarity, target, cost.Money, mats, target), 0xf1c40f)
	joined := ""
	for idx, id := range ids {
		if idx > 0 {
			joined += ","
		}
		joined += id
	}
	_ = b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("✅ Confirm Fusion", components.EncodeOwner(userID, "sanctuary", "fusion_confirm", rarity, joined), discordgo.SuccessButton),
			components.Button("❌ Cancel", components.EncodeOwner(userID, "sanctuary", "view"), discordgo.SecondaryButton),
		),
	}))
	_ = lang
}

func (c *Cog) onFusionConfirm(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 2 {
		return
	}
	// rest[0]=rarity, rest[1]=comma joined ids
	idsStr := rest[1]
	parts := []string{}
	// custom_id may have been split by ::, so joined ids may be in separate rest entries if commas not escaped ; handle both
	if len(rest) > 2 {
		// if ids were split by Encode, they would be separate entries; reconstruct
		for _, p := range rest[1:] {
			parts = append(parts, p)
		}
		idsStr = ""
		for idx, p := range parts {
			if idx > 0 {
				idsStr += ","
			}
			idsStr += p
		}
		// fallback: values also carry ids if coming from previous select? not here
	}
	_ = idsStr
	idStrs := splitIDs(idsStr)
	var ids []int64
	for _, s := range idStrs {
		if s == "" {
			continue
		}
		if id, err := strconv.ParseInt(s, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		// try fallback from Values if button encode failed due to length limit
		data := i.MessageComponentData()
		for _, v := range data.Values {
			if id, err := strconv.ParseInt(v, 10, 64); err == nil {
				ids = append(ids, id)
			}
		}
	}
	newPet, err := c.svc.TradeUp(userID, ids)
	if err != nil {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ Fusion failed: " + err.Error(), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	pt := ps.PetTypes[newPet.PetType]
	emoji := "🐾"
	if pt != nil {
		emoji = pt.Emoji
	}
	embed := components.Embed("✨ Fusion Success!", fmt.Sprintf("%s **%s** (%s) created!", emoji, newPet.Nickname, newPet.PetType), 0x2ecc71)
	_ = b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))
	_ = lang
}

func splitIDs(s string) []string {
	out := []string{}
	cur := ""
	for _, ch := range s {
		if ch == ',' {
			out = append(out, cur)
			cur = ""
		} else {
			cur += string(ch)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func (c *Cog) onAscendMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	active, err := c.psvc.GetActivePet(userID)
	if err != nil || active == nil {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: i18n.T("pets.transcend.no_pet", lang), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	if active.Level < 20 {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: i18n.T("pets.transcend.level_low", lang, map[string]any{"name": active.Nickname}), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	if active.TranscendLockedUntil != nil && time.Now().Before(*active.TranscendLockedUntil) {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("❌ %s is locked until %s", active.Nickname, active.TranscendLockedUntil.Format(time.RFC822)), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	pets, _ := c.psvc.GetPets(userID)
	opts := []discordgo.SelectMenuOption{}
	for _, p := range pets {
		if p.ID == active.ID {
			continue
		}
		if p.PetType != active.PetType {
			continue
		}
		if p.OnExpedition {
			continue
		}
		pt := ps.PetTypes[p.PetType]
		emoji := "🐾"
		if pt != nil {
			emoji = pt.Emoji
		}
		label := p.Nickname + " Lvl " + strconv.Itoa(p.Level)
		if len([]rune(label)) > 50 {
			label = string([]rune(label)[:50])
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label: label,
			Value: strconv.FormatInt(p.ID, 10),
			Emoji: &discordgo.ComponentEmoji{Name: emoji},
		})
		if len(opts) >= 25 {
			break
		}
	}
	if len(opts) == 0 {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: fmt.Sprintf("❌ No sacrifice of same species (%s) found.", active.PetType), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	embed := components.Embed("👑 Ascend — Transcend", fmt.Sprintf("Active: **%s** (%s) Lvl %d TRS %d\nSacrifice same species to gain +1 TRS and lock 24h.", active.Nickname, active.PetType, active.Level, active.TrsLvl), 0x9b59b6)
	sel := discordgo.SelectMenu{
		CustomID:    components.EncodeOwner(userID, "sanctuary", "ascend_pick"),
		Placeholder: "Choose sacrifice",
		Options:     opts,
	}
	_ = b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, []discordgo.MessageComponent{components.ActionRow(sel)}))
}

func (c *Cog) onAscendPick(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		return
	}
	sacID, _ := strconv.ParseInt(data.Values[0], 10, 64)
	active, err := c.svc.Transcend(userID, sacID)
	if err != nil {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ Transcend failed: " + err.Error(), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	embed := components.Embed("✨ Transcendence!", i18n.T("pets.transcend.success", lang, map[string]any{"name": active.Nickname, "level": active.TrsLvl}), 0x9b59b6)
	embed.Footer = &discordgo.MessageEmbedFooter{Text: "Locked for 24h"}
	_ = b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, nil))
}

// ── Pet picker: shared paginated + searchable pet selector ────────────────
//
// Retire, Recall, and Showcase all need "pick one of my pets" with the same
// shape: a select menu (capped at Discord's 25-option limit) plus Prev/Next
// paging and a name-or-species search, so a player with a hundred pets can
// still reach any of them. The three menus share this one implementation;
// only the source pet list, labels, and destination pick-action differ.

const petPageSize = 25

// petPickerKind identifies which picker flow a page/search interaction
// belongs to, since "pet_page" and "pet_search_*" are shared components.
type petPickerKind string

const (
	pickerRetire   petPickerKind = "retire"
	pickerRecall   petPickerKind = "recall"
	pickerShowcase petPickerKind = "showcase"
)

type petPickerSpec struct {
	title          string
	color          int
	promptKey      string
	placeholderKey string
	noneKey        string
	pickAction     string
	fetch          func(c *Cog, userID int64) ([]model.UserPet, error)
	label          func(p model.UserPet) (label, emoji string)
}

func petLabel(p model.UserPet) (string, string) {
	pt := ps.PetTypes[p.PetType]
	emoji := "🐾"
	if pt != nil {
		emoji = pt.Emoji
	}
	return p.Nickname, emoji
}

var petPickers = map[petPickerKind]petPickerSpec{
	pickerRetire: {
		title: "🏞️ Retire a Pet", color: 0x2ecc71,
		promptKey: "sanctuary.retire_prompt", placeholderKey: "sanctuary.retire_placeholder",
		noneKey: "sanctuary.no_active_pets", pickAction: "retire_pick",
		fetch: func(c *Cog, userID int64) ([]model.UserPet, error) { return c.psvc.GetActiveRosterPets(userID) },
		label: petLabel,
	},
	pickerRecall: {
		title: "🔙 Recall a Pet", color: 0x3498db,
		promptKey: "sanctuary.recall_prompt", placeholderKey: "sanctuary.recall_placeholder",
		noneKey: "sanctuary.no_sanctuary_pets", pickAction: "recall_pick",
		fetch: func(c *Cog, userID int64) ([]model.UserPet, error) { return c.psvc.GetSanctuaryPets(userID) },
		label: petLabel,
	},
	pickerShowcase: {
		title: "⭐ Showcase Pets", color: 0xf1c40f,
		promptKey: "sanctuary.showcase_prompt", placeholderKey: "sanctuary.showcase_placeholder",
		noneKey: "sanctuary.no_sanctuary_pets", pickAction: "showcase_select",
		fetch: func(c *Cog, userID int64) ([]model.UserPet, error) { return c.psvc.GetSanctuaryPets(userID) },
		label: func(p model.UserPet) (string, string) {
			label, emoji := petLabel(p)
			if p.ShowcaseSlot > 0 {
				label += " [Slot " + itoa(p.ShowcaseSlot) + "]"
			}
			return label, emoji
		},
	},
}

// filterPets keeps pets whose nickname or species contains query (case-insensitive).
func filterPets(pets []model.UserPet, query string) []model.UserPet {
	if query == "" {
		return pets
	}
	q := strings.ToLower(query)
	out := make([]model.UserPet, 0, len(pets))
	for _, p := range pets {
		if strings.Contains(strings.ToLower(p.Nickname), q) || strings.Contains(strings.ToLower(p.PetType), q) {
			out = append(out, p)
		}
	}
	return out
}

// pagePets slices pets into the requested page (clamped in range) of petPageSize.
func pagePets(pets []model.UserPet, page int) (pageItems []model.UserPet, totalPages, clamped int) {
	totalPages = (len(pets) + petPageSize - 1) / petPageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if page < 0 {
		page = 0
	}
	if page > totalPages-1 {
		page = totalPages - 1
	}
	start := page * petPageSize
	end := start + petPageSize
	if start > len(pets) {
		start = len(pets)
	}
	if end > len(pets) {
		end = len(pets)
	}
	return pets[start:end], totalPages, page
}

func (c *Cog) renderPetPicker(b *interaction.Bot, i *discordgo.InteractionCreate, kind petPickerKind, page int, query string) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	spec, ok := petPickers[kind]
	if !ok {
		return
	}
	all, err := spec.fetch(c, userID)
	if err != nil || len(all) == 0 {
		interaction.RespondError(b, i, lang, spec.noneKey)
		return
	}
	filtered := filterPets(all, query)
	pageItems, totalPages, page := pagePets(filtered, page)

	opts := make([]discordgo.SelectMenuOption, 0, len(pageItems))
	for _, p := range pageItems {
		label, emoji := spec.label(p)
		if len([]rune(label)) > 100 {
			label = string([]rune(label)[:100])
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label: label,
			Value: strconv.FormatInt(p.ID, 10),
			Emoji: &discordgo.ComponentEmoji{Name: emoji},
		})
	}

	prompt := i18n.T(spec.promptKey, lang) + fmt.Sprintf("\n\n**Page %d/%d** · %d pet(s)", page+1, totalPages, len(filtered))
	if query != "" {
		prompt += fmt.Sprintf("\n🔍 Filter: `%s`", query)
	}
	embed := components.Embed(spec.title, prompt, spec.color)

	var rows []discordgo.MessageComponent
	if len(opts) > 0 {
		rows = append(rows, components.ActionRow(discordgo.SelectMenu{
			CustomID:    components.EncodeOwner(userID, "sanctuary", spec.pickAction),
			Placeholder: i18n.T(spec.placeholderKey, lang),
			Options:     opts,
		}))
	}
	rows = append(rows, components.ActionRow(
		components.ButtonDisabled("◀ Prev",
			components.EncodeOwner(userID, "sanctuary", "pet_page", string(kind), strconv.Itoa(page-1), query),
			discordgo.SecondaryButton, page <= 0),
		components.Button("🔍 Search",
			components.EncodeOwner(userID, "sanctuary", "pet_search_open", string(kind)),
			discordgo.PrimaryButton),
		components.ButtonDisabled("Next ▶",
			components.EncodeOwner(userID, "sanctuary", "pet_page", string(kind), strconv.Itoa(page+1), query),
			discordgo.SecondaryButton, page >= totalPages-1),
	))
	_ = b.Session.InteractionRespond(i.Interaction, components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, rows))
}

func (c *Cog) onPetPage(b *interaction.Bot, i *discordgo.InteractionCreate) {
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) < 2 {
		return
	}
	page, _ := strconv.Atoi(rest[1])
	query := ""
	if len(rest) > 2 {
		query = rest[2]
	}
	c.renderPetPicker(b, i, petPickerKind(rest[0]), page, query)
}

func (c *Cog) onPetSearchOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	kind := string(pickerRetire)
	if len(rest) > 0 {
		kind = rest[0]
	}
	modal := components.ModalResponse(
		components.EncodeOwner(userID, "sanctuary", "pet_search_submit", kind),
		i18n.T("sanctuary.search_modal_title", lang),
		components.TextInput("query", i18n.T("sanctuary.search_input_label", lang), false,
			i18n.T("sanctuary.search_input_placeholder", lang), discordgo.TextInputShort, 0, 30),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onPetSearchSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	_, _, rest := components.Decode(i.ModalSubmitData().CustomID)
	kind := pickerRetire
	if len(rest) > 0 {
		kind = petPickerKind(rest[0])
	}
	query := strings.ReplaceAll(strings.TrimSpace(interaction.ModalValues(i)["query"]), "::", " ")
	c.renderPetPicker(b, i, kind, 0, query)
}

func (c *Cog) onRetireSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.renderPetPicker(b, i, pickerRetire, 0, "")
}

func (c *Cog) onRetirePick(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		return
	}
	petID, err := strconv.ParseInt(data.Values[0], 10, 64)
	if err != nil {
		return
	}
	if err := c.svc.RetirePet(userID, petID); err != nil {
		interaction.RespondError(b, i, lang, "sanctuary.retire_error")
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("sanctuary.retire_success", lang),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) onRecallSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.renderPetPicker(b, i, pickerRecall, 0, "")
}

func (c *Cog) onRecallPick(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		return
	}
	petID, err := strconv.ParseInt(data.Values[0], 10, 64)
	if err != nil {
		return
	}
	if err := c.svc.RecallPet(userID, petID); err != nil {
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ Could not recall: " + err.Error(), Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T("sanctuary.recall_success", lang),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) onShowcaseSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.renderPetPicker(b, i, pickerShowcase, 0, "")
}

func (c *Cog) onShowcaseSet(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		return
	}
	_, _ = strconv.ParseInt(data.Values[0], 10, 64)

	slotOpts := make([]discordgo.SelectMenuOption, 0, 6)
	slotOpts = append(slotOpts, discordgo.SelectMenuOption{
		Label: "Remove from showcase",
		Value: "0",
	})
	for i := 1; i <= 5; i++ {
		slotOpts = append(slotOpts, discordgo.SelectMenuOption{
			Label: "Slot " + itoa(i),
			Value: itoa(i),
		})
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
			components.Embed("⭐ Choose Showcase Slot", i18n.T("sanctuary.showcase_slot_prompt", lang), 0xf1c40f),
			[]discordgo.MessageComponent{
				components.ActionRow(
					discordgo.SelectMenu{
						CustomID:    components.EncodeOwner(userID, "sanctuary", "showcase_slot", data.Values[0]),
						Placeholder: "Select a showcase slot...",
						Options:     slotOpts,
					},
				),
			}))
}

func (c *Cog) onShowcaseAssign(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	data := i.MessageComponentData()
	_, _, rest := components.Decode(data.CustomID)
	if len(rest) == 0 || len(data.Values) == 0 {
		return
	}
	petID, _ := strconv.ParseInt(rest[0], 10, 64)
	slot, _ := strconv.Atoi(data.Values[0])
	err := c.svc.SetShowcase(userID, petID, slot)
	if err != nil {
		interaction.RespondError(b, i, lang, "sanctuary.showcase_error")
		return
	}
	embed, comps := c.buildSanctuaryEmbed(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) visitSanctuary(b *interaction.Bot, i *discordgo.InteractionCreate, targetID int64, lang string) {
	embed := c.buildVisitEmbed(targetID, lang)
	if embed == nil {
		interaction.RespondError(b, i, lang, "sanctuary.no_sanctuary")
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, nil))
}

func (c *Cog) visitSanctuaryMessage(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message, targetID int64, lang string) {
	embed := c.buildVisitEmbed(targetID, lang)
	if embed == nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("sanctuary.no_sanctuary", lang))
		return
	}
	_, _ = s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func (c *Cog) buildVisitEmbed(targetID int64, lang string) *discordgo.MessageEmbed {
	san, err := c.svc.GetSanctuary(targetID)
	if err != nil {
		return nil
	}
	tier := san.Tier
	if tier == 0 {
		return nil
	}
	tierName := "Unknown"
	if t, ok := sansvc.SanctuaryTiers[tier]; ok {
		tierName = t.Name
	}
	var showcase []model.UserPet
	c.store.DB.Where("user_id = ? AND in_sanctuary = ? AND showcase_slot > ?", targetID, true, 0).
		Order("showcase_slot ASC").Find(&showcase)
	showcaseStr := ""
	if len(showcase) > 0 {
		for _, p := range showcase {
			pt := ps.PetTypes[p.PetType]
			emoji := "🐾"
			if pt != nil {
				emoji = pt.Emoji
			}
			rarity := i18n.T("rarities."+pt.Rarity, lang)
			showcaseStr += fmt.Sprintf("%s **%s** (%s)\n", emoji, p.Nickname, rarity)
		}
	} else {
		showcaseStr = i18n.T("sanctuary.no_showcase", lang)
	}
	var count int64
	c.store.DB.Model(&model.UserPet{}).Where("user_id = ? AND in_sanctuary = ?", targetID, true).Count(&count)
	progress := c.collectionProgress(targetID)
	totalHave := 0
	totalAll := 0
	for _, info := range progress {
		totalHave += info.Have
		totalAll += info.Total
	}
	desc := fmt.Sprintf("**%s** (Lvl %d)\n**Pets:** %d\n**Collection:** %d/%d species\n\n**⭐ Showcase:**\n%s",
		tierName, tier, int(count), totalHave, totalAll, showcaseStr)
	embed := components.Embed(
		i18n.T("sanctuary.visit_title", lang, map[string]any{"user": "<@" + strconv.FormatInt(targetID, 10) + ">"}),
		desc,
		0x2ecc71,
	)
	return embed
}

func biomeEmoji(biome string) string {
	switch biome {
	case "forest":
		return "🌲"
	case "cave":
		return "🦇"
	case "desert":
		return "🏜️"
	case "mountain":
		return "🏔️"
	case "ocean":
		return "🌊"
	case "tundra":
		return "❄️"
	case "volcano":
		return "🌋"
	}
	return "🌍"
}

func buildProgressBar(current, total, size int) string {
	if total == 0 {
		return "[" + repeat("░", size) + "]"
	}
	filled := current * size / total
	if filled > size {
		filled = size
	}
	bar := "["
	for i := 0; i < size; i++ {
		if i < filled {
			bar += "▓"
		} else {
			bar += "░"
		}
	}
	bar += "]"
	return bar
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func joinUnique(items []string) string {
	seen := make(map[string]bool)
	out := ""
	first := true
	for _, item := range items {
		if seen[item] {
			continue
		}
		seen[item] = true
		if !first {
			out += ", "
		}
		out += item
		first = false
	}
	return out
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
