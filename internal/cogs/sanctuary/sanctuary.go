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
	c := &Cog{store: s, cfg: cfg, svc: sansvc.New(s, cfg), psvc: ps.New(s, cfg)}
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
