package expedition

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	expeditionsvc "guacagamblebot/internal/service/expedition"
	"guacagamblebot/internal/items"
	petsvc "guacagamblebot/internal/service/pets"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *expeditionsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: expeditionsvc.New(s, cfg)}
	r.Slash("expedition", "Expéditions de familiers", c.onSlash)
	r.Slash("exp", "Expéditions de familiers", c.onSlash)
	r.Prefix("expedition", c.onPrefix)
	r.Prefix("exp", c.onPrefix)
}

func (c *Cog) onSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	c.handle(i, b, lang)
}

func (c *Cog) onPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	parts := strings.Fields(m.Content)
	sub := ""
	duration := ""
	if len(parts) > 1 {
		sub = parts[1]
	}
	if len(parts) > 2 {
		duration = parts[2]
	}
	embed, comps := c.handleMessage(interaction.ToInt64(m.Author.ID), sub, duration, lang)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) handle(i *discordgo.InteractionCreate, b *interaction.Bot, lang string) {
	userID := interaction.ToInt64(interaction.UserID(i))
	options := i.ApplicationCommandData().Options
	sub := ""
	duration := ""
	for _, opt := range options {
		switch opt.Name {
		case "action":
			sub = opt.StringValue()
		case "duration":
			duration = opt.StringValue()
		}
	}
	embed, comps := c.handleMessage(userID, sub, duration, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) handleMessage(userID int64, sub, duration, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	switch sub {
	case "start":
		return c.start(userID, duration, lang)
	case "status":
		return c.status(userID, lang)
	case "claim":
		return c.claim(userID, lang)
	default:
		embed := components.Embed("🚀 "+i18n.T("expedition.help_title", lang),
			i18n.T("expedition.help_desc", lang)+"\n\n"+
				i18n.T("expedition.help_benefits", lang)+"\n"+
				i18n.T("expedition.help_benefits_list", lang)+"\n\n"+
				i18n.T("expedition.help_commands", lang)+"\n"+
				i18n.T("expedition.help_commands_list", lang),
			0x3498db)
		return embed, nil
	}
}

func durationHours(s string) int {
	switch s {
	case "1", "2":
		return 1
	case "4", "6":
		return 4
	case "8":
		return 8
	case "24":
		return 24
	}
	return 0
}

func (c *Cog) start(userID int64, durationStr string, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	dur := durationHours(durationStr)
	if dur == 0 {
		return components.Embed("❌", i18n.T("expedition.invalid_duration", lang), 0xe74c3c), nil
	}

	pet, err := petsvc.New(c.store, c.cfg).GetActivePet(userID)
	if err != nil || pet == nil {
		return components.Embed("❌", i18n.T("expedition.no_pet", lang), 0xe74c3c), nil
	}

	active, _ := c.svc.GetActive(userID)
	if active != nil {
		return components.Embed("❌", i18n.T("expedition.already_active", lang), 0xe74c3c), nil
	}

	res := c.svc.Generate(pet.Nickname, pet.Level, dur, lang)
	_, err = c.svc.Start(userID, pet.ID, dur, res)
	if err != nil {
		return components.Embed("❌", i18n.T("expedition.invalid_duration", lang), 0xe74c3c), nil
	}

	return components.Embed("🚀",
		i18n.T("expedition.start_success", lang, map[string]any{"pet": pet.Nickname, "duration": dur}),
		0x2ecc71), nil
}

func (c *Cog) status(userID int64, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	exp, err := c.svc.GetActive(userID)
	if err != nil || exp == nil {
		return components.Embed("❌", i18n.T("expedition.no_active", lang), 0xe74c3c), nil
	}

	now := time.Now()
	totalSec := exp.EndTime.Sub(exp.StartTime).Seconds()
	elapsed := now.Sub(exp.StartTime).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	progress := int((elapsed / totalSec) * 100)
	if progress > 100 {
		progress = 100
	}

	color := 0xf39c12
	if progress >= 100 {
		color = 0x2ecc71
	}

	embed := components.Embed(
		i18n.T("expedition.status_title", lang, map[string]any{"pet": "Pet"}),
		"", color)
	embed.Fields = []*discordgo.MessageEmbedField{
		components.Field(i18n.T("expedition.log_header", lang),
			i18n.T("expedition.status_desc", lang, map[string]any{"progress": progress, "end_time": exp.EndTime.Format("15:04")}),
			false),
	}

	if progress >= 100 {
		embed.Fields = append(embed.Fields, components.Field("✨ "+i18n.T("expedition.claim_ready", lang), "\u200b", false))
	} else {
		rem := exp.EndTime.Sub(now)
		h := int(rem.Hours())
		m := int(rem.Minutes()) % 60
		embed.Fields = append(embed.Fields, components.Field("⏳",
			i18n.T("expedition.time_format", lang, map[string]any{"hours": h, "minutes": m}),
			false))
	}

	return embed, nil
}

func (c *Cog) claim(userID int64, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	exp, err := c.svc.GetActive(userID)
	if err != nil || exp == nil {
		return components.Embed("❌", i18n.T("expedition.no_active", lang), 0xe74c3c), nil
	}

	if time.Now().Before(exp.EndTime) {
		rem := exp.EndTime.Sub(time.Now())
		h := int(rem.Hours())
		m := int(rem.Minutes()) % 60
		remStr := i18n.T("expedition.time_format", lang, map[string]any{"hours": h, "minutes": m})
		return components.Embed("❌", i18n.T("expedition.not_finished", lang, map[string]any{"remaining": remStr}), 0xe74c3c), nil
	}

	pet, err := petsvc.New(c.store, c.cfg).GetPetByID(exp.PetID)
	if err != nil || pet == nil {
		return components.Embed("❌", "Pet not found.", 0xe74c3c), nil
	}

	petSvc := petsvc.New(c.store, c.cfg)
	lvlRes := petSvc.AddXP(pet, exp.RewardXP)
	leveled := lvlRes.Leveled
	petSvc.AddBond(pet, 2)
	petSvc.RecordHistory(pet, "expedition",
		"🧭 **"+pet.Nickname+"** returned from an expedition with **"+itoa(exp.RewardXP)+" XP**!")
	_ = petSvc.UpdatePet(pet)

	var rawItems []string
	_ = json.Unmarshal([]byte(exp.RewardItems), &rawItems)

	lootStr := ""
	if len(rawItems) > 0 {
		counts := map[string]int{}
		for _, it := range rawItems {
			it = strings.TrimSpace(it)
			if it == "" {
				continue
			}
			counts[it]++
		}
		for itName, count := range counts {
			lootStr += "- " + itoa(count) + "x " + items.DisplayName(itName) + "\n"
		}
	} else {
		lootStr = i18n.T("expedition.no_items", lang)
	}

	_ = c.svc.Claim(exp)

	desc := i18n.T("expedition.claim_title", lang, map[string]any{"pet": pet.Nickname, "xp": exp.RewardXP, "items": "\n" + lootStr})
	if leveled {
		desc += "\n\n" + i18n.T("pets.play.level_up", lang, map[string]any{"name": pet.Nickname, "level": pet.Level})
	}

	return components.Embed("🎁", desc, 0xf1c40f), nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	return out
}
