package expedition

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	expeditionsvc "guacagamblebot/internal/service/expedition"
	jsvc "guacagamblebot/internal/service/journal"
	petsvc "guacagamblebot/internal/service/pets"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
)

// durationOptions lists the selectable expeditions, ordered from shortest to
// longest. The key localizes the button label, the value is the hour count
// carried in the custom_id.
var durationOptions = []struct{ Key, Hours string }{
	{"quick", "1"},
	{"trip", "4"},
	{"long", "8"},
	{"epic", "24"},
}

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *expeditionsvc.Service
	psvc  *petsvc.Service
	qsvc  *questssvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{
		store: s,
		cfg:   cfg,
		svc:   expeditionsvc.New(s, cfg),
		psvc:  petsvc.New(s, cfg),
		qsvc:  questssvc.New(s, cfg),
	}
	r.Slash("expedition", "Send your pet exploring", c.onSlashMenu)
	r.Slash("exp", "Send your pet exploring", c.onSlashMenu)
	r.Prefix("expedition", c.onPrefixMenu)
	r.Prefix("exp", c.onPrefixMenu)
	r.Component("expedition", "menu", c.onMenu)
	r.Component("expedition", "start", c.onStart)
	r.Component("expedition", "claim", c.onClaim)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.menuView(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	embed, comps := c.menuView(lang, userID)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

// onMenu re-renders the current view: the active expedition status when one is
// running, otherwise the launch menu. Used by the refresh and back buttons.
func (c *Cog) onMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.menuView(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// menuView dispatches to the launch menu or the active expedition status.
func (c *Cog) menuView(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	exp, err := c.svc.GetActive(userID)
	if err == nil && exp != nil {
		return c.statusView(lang, userID, exp)
	}
	return c.idleMenu(lang, userID)
}

// idleMenu renders the expedition dashboard: the pet card and one button per
// named duration. Buttons are disabled while the pet is K.O.
func (c *Cog) idleMenu(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	pet, _ := c.psvc.GetActivePet(userID)

	desc := i18n.T("expedition.menu_no_pet", lang)
	petKO := false
	if pet != nil {
		desc = i18n.T("expedition.menu_desc", lang, map[string]any{"pet": pet.Nickname})
		if pet.HP <= 0 {
			petKO = true
			desc += "\n\n" + i18n.T("expedition.ko_warning", lang, map[string]any{"name": pet.Nickname})
		}
	}

	embed := components.Embed(i18n.T("expedition.menu_title", lang), desc, 0x3498db)
	embed.Footer = &discordgo.MessageEmbedFooter{Text: i18n.T("expedition.menu_footer", lang)}
	if pet != nil {
		embed.Fields = []*discordgo.MessageEmbedField{
			components.Field(i18n.T("expedition.status_field_pet", lang), c.petCard(pet, lang), false),
		}
	}

	var row []discordgo.MessageComponent
	for _, opt := range durationOptions {
		row = append(row, components.ButtonDisabled(
			i18n.T("expedition.duration."+opt.Key, lang),
			components.EncodeOwner(userID, "expedition", "start", opt.Hours),
			discordgo.PrimaryButton,
			petKO,
		))
	}
	comps := []discordgo.MessageComponent{components.ActionRow(row...)}
	return embed, comps
}

// petCard renders the compact pet line shared by the menu and status views.
func (c *Cog) petCard(pet *model.UserPet, lang string) string {
	pt := petsvc.PetTypes[pet.PetType]
	emoji := "🐾"
	rarity := ""
	if pt != nil {
		emoji = pt.Emoji
		rarity = i18n.T("rarities."+pt.Rarity, lang)
	}
	return i18n.T("expedition.pet_card", lang, map[string]any{
		"emoji":  emoji,
		"name":   pet.Nickname,
		"level":  pet.Level,
		"rarity": rarity,
		"hp_bar": components.HPBar(pet.HP, pet.MaxHP),
		"hp":     pet.HP,
		"max_hp": pet.MaxHP,
	})
}

// statusView renders the active expedition: progress bar, ETA, adventure log,
// pet status and a refresh + claim button row (claim enabled once finished).
func (c *Cog) statusView(lang string, userID int64, exp *model.PetExpedition) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	pet, _ := c.psvc.GetPetByID(exp.PetID)
	petName := "Pet"
	if pet != nil {
		petName = pet.Nickname
	}

	now := time.Now()
	done := !now.Before(exp.EndTime)
	progress := 0
	if total := exp.EndTime.Sub(exp.StartTime).Seconds(); total > 0 {
		elapsed := now.Sub(exp.StartTime).Seconds()
		if elapsed < 0 {
			elapsed = 0
		}
		progress = int(elapsed / total * 100)
	}
	if progress > 100 {
		progress = 100
	}

	color := 0xf39c12
	if done {
		color = 0x2ecc71
	}

	desc := i18n.T("expedition.status_progress", lang, map[string]any{
		"bar":      progressBar(progress, 100, 10),
		"progress": strconv.Itoa(progress),
	})
	if done {
		desc += "\n" + i18n.T("expedition.status_ready", lang)
	} else {
		rem := exp.EndTime.Sub(now)
		h := int(rem.Hours())
		m := int(rem.Minutes()) % 60
		desc += "\n" + i18n.T("expedition.status_eta", lang, map[string]any{
			"end_time":  exp.EndTime.Format("15:04"),
			"remaining": i18n.T("expedition.time_format", lang, map[string]any{"hours": h, "minutes": m}),
		})
	}

	embed := components.Embed(
		i18n.T("expedition.status_title", lang, map[string]any{"pet": petName}),
		desc, color)

	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("expedition.status_field_log", lang), c.adventureLog(exp, pet, lang, 8), false),
	}
	if pet != nil {
		embed.Fields = append(embed.Fields, components.Field(i18n.T("expedition.status_field_pet", lang), c.petCard(pet, lang), false))
	}
	embed.Fields = append(embed.Fields, components.Field(i18n.T("expedition.status_field_rewards", lang),
		i18n.T("expedition.rewards_line", lang, map[string]any{
			"xp":    exp.RewardXP,
			"items": itemCount(exp.RewardItems),
		}), false))

	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("expedition.refresh_label", lang),
				components.EncodeOwner(userID, "expedition", "menu"), discordgo.SecondaryButton),
			components.ButtonDisabled(i18n.T("expedition.claim_label", lang),
				components.EncodeOwner(userID, "expedition", "claim"), discordgo.SuccessButton, !done),
		),
	}
	return embed, comps
}

// adventureLog renders the stored expedition events through i18n, falling back
// to the raw stored text for legacy rows. Only the most recent events are kept
// so the field stays readable.
func (c *Cog) adventureLog(exp *model.PetExpedition, pet *model.UserPet, lang string, max int) string {
	var events []expeditionsvc.ExpeditionEvent
	if err := json.Unmarshal([]byte(exp.Log), &events); err != nil || len(events) == 0 {
		return i18n.T("expedition.no_events", lang)
	}
	lines := make([]string, 0, len(events))
	for _, ev := range events {
		lines = append(lines, c.eventLine(ev, pet, lang))
	}
	if len(lines) > max {
		lines = lines[len(lines)-max:]
	}
	return strings.Join(lines, "\n")
}

// eventLine localizes a single expedition event from its structured fields.
func (c *Cog) eventLine(ev expeditionsvc.ExpeditionEvent, pet *model.UserPet, lang string) string {
	petName := "Pet"
	if pet != nil {
		petName = pet.Nickname
	}
	params := map[string]any{"pet": petName}
	switch ev.Type {
	case "exploration":
		if ev.Location == "" {
			return ev.Text
		}
		params["location"] = i18n.T("expedition.locations."+ev.Location, lang)
		params["xp"] = ev.XP
		return i18n.T("expedition.events.exploration", lang, params)
	case "combat":
		if ev.Enemy == "" {
			return ev.Text
		}
		params["enemy"] = ev.Enemy
		if ev.CombatResult == "win" {
			params["xp"] = ev.XP
			return i18n.T("expedition.events.combat_win", lang, params)
		}
		if ev.CombatResult == "loss" {
			return i18n.T("expedition.events.combat_ko", lang, params)
		}
		return i18n.T("expedition.events.combat_loss", lang, params)
	case "loot":
		if ev.Item == "" {
			return ev.Text
		}
		params["item"] = items.LocalizedName(ev.Item, lang)
		return i18n.T("expedition.events.loot", lang, params)
	case "rest":
		if ev.Heal > 0 {
			params["hp"] = ev.Heal
			return i18n.T("expedition.events.rest_heal", lang, params)
		}
		return i18n.T("expedition.events.rest", lang, params)
	case "return":
		return i18n.T("expedition.events.return_home", lang, params)
	}
	return ev.Text
}

// onStart launches the expedition of the selected duration, guarded by the pet
// being alive and no expedition already running.
func (c *Cog) onStart(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	dur := durationHours(rest)
	if dur == 0 {
		interaction.RespondError(b, i, lang, "expedition.invalid_duration")
		return
	}

	pet, err := c.psvc.GetActivePet(userID)
	if err != nil || pet == nil {
		if granted, _ := c.qsvc.EnsureTutorialEgg(userID); granted {
			interaction.RespondError(b, i, lang, "quests.tutorial_egg_granted")
		} else {
			interaction.RespondError(b, i, lang, "expedition.no_pet")
		}
		return
	}
	if pet.HP <= 0 {
		c.respondErrorParams(b, i, lang, "expedition.pet_ko", map[string]any{"name": pet.Nickname})
		return
	}

	active, _ := c.svc.GetActive(userID)
	if active != nil {
		interaction.RespondError(b, i, lang, "expedition.already_active")
		return
	}

	res := c.svc.Generate(pet, dur)
	if _, err := c.svc.Start(userID, pet.ID, dur, res); err != nil {
		if errors.Is(err, expeditionsvc.ErrPetKO) {
			c.respondErrorParams(b, i, lang, "expedition.pet_ko", map[string]any{"name": pet.Nickname})
			return
		}
		if errors.Is(err, store.ErrInventoryFull) {
			interaction.RespondError(b, i, lang, "inventory.full")
			return
		}
		slog.Error("expedition: failed to start", "user_id", userID, "pet_id", pet.ID, "error", err)
		interaction.RespondError(b, i, lang, "expedition.error")
		return
	}

	exp, err := c.svc.GetActive(userID)
	if err != nil || exp == nil {
		interaction.RespondError(b, i, lang, "expedition.error")
		return
	}
	embed, comps := c.statusView(lang, userID, exp)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// onClaim collects the expedition rewards once the pet is back, then surfaces
// the quest/journal/achievement follow-ups like the other activity cogs.
func (c *Cog) onClaim(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	exp, err := c.svc.GetActive(userID)
	if err != nil || exp == nil {
		interaction.RespondError(b, i, lang, "expedition.no_active")
		return
	}
	if time.Now().Before(exp.EndTime) {
		rem := exp.EndTime.Sub(time.Now())
		h := int(rem.Hours())
		m := int(rem.Minutes()) % 60
		remStr := i18n.T("expedition.time_format", lang, map[string]any{"hours": h, "minutes": m})
		c.respondErrorParams(b, i, lang, "expedition.not_finished", map[string]any{"remaining": remStr})
		return
	}

	pet, err := c.psvc.GetPetByID(exp.PetID)
	if err != nil || pet == nil {
		interaction.RespondError(b, i, lang, "expedition.error")
		return
	}

	petSvc := c.psvc
	lvlRes := petSvc.AddXP(pet, exp.RewardXP)
	leveled := lvlRes.Leveled
	petSvc.AddBond(pet, 2)
	petSvc.RecordHistory(pet, "expedition",
		"🧭 **"+pet.Nickname+"** returned from an expedition with **"+strconv.Itoa(exp.RewardXP)+" XP**!")
	if err := petSvc.UpdatePet(pet); err != nil {
		slog.Error("expedition: failed to save pet after expedition", "user_id", userID, "pet_id", pet.ID, "error", err)
	}

	lootStr := c.lootString(exp.RewardItems, lang)

	charLeveled, charNewLevel, _ := c.svc.Claim(exp)
	_, artLeveled, _ := petSvc.AddArtifactXP(userID, petsvc.ArtifactExpeditionXP)

	embed, comps := c.claimView(userID, pet, exp, leveled, charLeveled, charNewLevel, artLeveled, lootStr, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))

	if n, ok := c.store.PopQuestNotification(userID); ok {
		interaction.SendQuestNotification(b, i, n, lang)
	}
	if text, dm := jsvc.SceneLine(c.store, userID, "expedition", lang); text != "" {
		interaction.SendJournalScene(b, i, text, dm)
	}
	if unlocks, uerr := achievement.CheckAndUnlock(b.DB, userID); uerr == nil && len(unlocks) > 0 {
		interaction.SendAchievements(b, i, lang, unlocks)
	}
	c.maybePetInteraction(b, i, pet, lang)
}

// claimView renders the reward screen: XP earned, level-ups and the loot.
func (c *Cog) claimView(userID int64, pet *model.UserPet, exp *model.PetExpedition,
	leveled, charLeveled bool, charNewLevel int, artLeveled bool, lootStr, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	desc := i18n.T("expedition.claim_desc", lang, map[string]any{"pet": pet.Nickname, "xp": exp.RewardXP})
	if leveled {
		desc += "\n\n" + i18n.T("expedition.level_up", lang, map[string]any{"pet": pet.Nickname, "level": pet.Level})
	}
	if charLeveled {
		desc += "\n\n" + i18n.T("character.level_up", lang, map[string]any{"level": charNewLevel})
	}
	if artLeveled {
		desc += "\n\n" + i18n.T("expedition.artifact_leveled", lang)
	}

	embed := components.Embed(i18n.T("expedition.claim_title", lang), desc, 0xf1c40f)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("expedition.claim_field_loot", lang), lootStr, false),
	}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("expedition.menu_label", lang),
				components.EncodeOwner(userID, "expedition", "menu"), discordgo.SecondaryButton),
		),
	}
	return embed, comps
}

// lootString renders the collected item counts, localized by language.
func (c *Cog) lootString(raw string, lang string) string {
	var rawItems []string
	_ = json.Unmarshal([]byte(raw), &rawItems)

	counts := map[string]int{}
	for _, it := range rawItems {
		it = strings.TrimSpace(it)
		if it == "" {
			continue
		}
		counts[it]++
	}
	if len(counts) == 0 {
		return i18n.T("expedition.no_items", lang)
	}
	lines := make([]string, 0, len(counts))
	for itName, count := range counts {
		lines = append(lines, "- "+strconv.Itoa(count)+"x "+items.LocalizedName(itName, lang))
	}
	return strings.Join(lines, "\n")
}

// itemCount returns how many individual items were found, for the rewards
// preview on the status screen.
func itemCount(raw string) int {
	var rawItems []string
	_ = json.Unmarshal([]byte(raw), &rawItems)
	count := 0
	for _, it := range rawItems {
		if strings.TrimSpace(it) != "" {
			count++
		}
	}
	return count
}

// maybePetInteraction triggers a pet personality interaction when one is ready.
func (c *Cog) maybePetInteraction(b *interaction.Bot, i *discordgo.InteractionCreate, pet *model.UserPet, lang string) {
	if pet == nil {
		return
	}
	userID := interaction.ToInt64(interaction.UserID(i))
	ready, _ := c.store.CheckCooldown(userID, "pet_interaction", 180*time.Minute)
	if !ready {
		return
	}
	ir := petsvc.MaybeTriggerInteraction(pet, "expedition")
	if ir == nil {
		return
	}
	intro := i18n.T(ir.IntroKey(pet.Personality), lang)
	if intro == ir.IntroKey(pet.Personality) {
		intro = i18n.T(ir.GenericIntroKey(), lang)
	}
	opts := make([]discordgo.SelectMenuOption, 0, len(ir.Choices))
	for _, ch := range ir.Choices {
		label := i18n.T(ch.ChoiceLabelKey(), lang)
		if label == ch.ChoiceLabelKey() {
			label = ch.ID
		}
		opts = append(opts, discordgo.SelectMenuOption{
			Label: ch.Emoji + " " + label,
			Value: ch.ID,
		})
	}
	if len(opts) == 0 {
		return
	}
	embed := components.Embed(
		i18n.T("pets.interact.title", lang, map[string]any{"name": pet.Nickname}),
		intro, 0x9b59b6)
	_, _ = b.Session.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			components.ActionRow(
				discordgo.SelectMenu{
					CustomID:    components.EncodeOwner(userID, "pets", "interact", strconv.FormatInt(pet.ID, 10)),
					Placeholder: i18n.T("pets.interact.placeholder", lang),
					Options:     opts,
				},
			),
		},
		Flags: discordgo.MessageFlagsEphemeral,
	})
}

// respondErrorParams replies with an ephemeral error message using i18n params.
func (c *Cog) respondErrorParams(b *interaction.Bot, i *discordgo.InteractionCreate, lang, key string, params map[string]any) {
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: i18n.T(key, lang, params),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// durationHours maps the hour value carried in the custom_id to the expedition
// length in hours, or 0 when unknown.
func durationHours(rest []string) int {
	if len(rest) == 0 {
		return 0
	}
	switch rest[0] {
	case "1":
		return 1
	case "4":
		return 4
	case "8":
		return 8
	case "24":
		return 24
	}
	return 0
}

// progressBar renders a 10-segment █░ bar.
func progressBar(value, max, segments int) string {
	if max <= 0 {
		return strings.Repeat("░", segments)
	}
	filled := value * segments / max
	if filled > segments {
		filled = segments
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", segments-filled)
}
