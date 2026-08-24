package sanctuary

import (
	"fmt"
	"strconv"
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
	r.Component("sanctuary", "recall", c.onRecallSelect)
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

	return []discordgo.MessageComponent{components.ActionRow(buttons...)}
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
		if len(label) > 50 {
			label = label[:50]
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
		if len(label) > 50 {
			label = label[:50]
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

func (c *Cog) onRetireSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	pets, err := c.psvc.GetActiveRosterPets(userID)
	if err != nil || len(pets) == 0 {
		interaction.RespondError(b, i, lang, "sanctuary.no_active_pets")
		return
	}
	opts := make([]discordgo.SelectMenuOption, 0, len(pets))
	for _, p := range pets {
		pt := ps.PetTypes[p.PetType]
		emoji := "🐾"
		if pt != nil {
			emoji = pt.Emoji
		}
		label := p.Nickname
		if len(label) > 50 {
			label = label[:50]
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label: label,
			Value: strconv.FormatInt(p.ID, 10),
			Emoji: &discordgo.ComponentEmoji{Name: emoji},
		})
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
			components.Embed("🏞️ Retire a Pet", i18n.T("sanctuary.retire_prompt", lang), 0x2ecc71),
			[]discordgo.MessageComponent{
				components.ActionRow(
					discordgo.SelectMenu{
						CustomID:    components.EncodeOwner(userID, "sanctuary", "retire"),
						Placeholder: i18n.T("sanctuary.retire_placeholder", lang),
						Options:     opts,
					},
				),
			}))
}

func (c *Cog) onRecallSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	pets, err := c.psvc.GetSanctuaryPets(userID)
	if err != nil || len(pets) == 0 {
		interaction.RespondError(b, i, lang, "sanctuary.no_sanctuary_pets")
		return
	}
	opts := make([]discordgo.SelectMenuOption, 0, len(pets))
	for _, p := range pets {
		pt := ps.PetTypes[p.PetType]
		emoji := "🐾"
		if pt != nil {
			emoji = pt.Emoji
		}
		label := p.Nickname
		if len(label) > 50 {
			label = label[:50]
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label: label,
			Value: strconv.FormatInt(p.ID, 10),
			Emoji: &discordgo.ComponentEmoji{Name: emoji},
		})
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
			components.Embed("🔙 Recall a Pet", i18n.T("sanctuary.recall_prompt", lang), 0x3498db),
			[]discordgo.MessageComponent{
				components.ActionRow(
					discordgo.SelectMenu{
						CustomID:    components.EncodeOwner(userID, "sanctuary", "recall"),
						Placeholder: i18n.T("sanctuary.recall_placeholder", lang),
						Options:     opts,
					},
				),
			}))
}

func (c *Cog) onShowcaseSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	pets, err := c.psvc.GetSanctuaryPets(userID)
	if err != nil || len(pets) == 0 {
		interaction.RespondError(b, i, lang, "sanctuary.no_sanctuary_pets")
		return
	}
	opts := make([]discordgo.SelectMenuOption, 0, len(pets))
	for _, p := range pets {
		pt := ps.PetTypes[p.PetType]
		emoji := "🐾"
		if pt != nil {
			emoji = pt.Emoji
		}
		status := ""
		if p.ShowcaseSlot > 0 {
			status = " [Slot " + itoa(p.ShowcaseSlot) + "]"
		}
		label := p.Nickname + status
		if len(label) > 50 {
			label = label[:50]
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label: label,
			Value: strconv.FormatInt(p.ID, 10),
			Emoji: &discordgo.ComponentEmoji{Name: emoji},
		})
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
			components.Embed("⭐ Showcase Pets", i18n.T("sanctuary.showcase_prompt", lang), 0xf1c40f),
			[]discordgo.MessageComponent{
				components.ActionRow(
					discordgo.SelectMenu{
						CustomID:    components.EncodeOwner(userID, "sanctuary", "showcase_select"),
						Placeholder: i18n.T("sanctuary.showcase_placeholder", lang),
						Options:     opts,
					},
				),
			}))
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
