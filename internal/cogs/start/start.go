package start

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *questssvc.Service
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}

var activityLabels = map[string]string{
	"items_mined":          "⛏️ Mining",
	"items_farmed":         "🌾 Farming",
	"items_fished":         "🎣 Fishing",
	"items_hunted":         "⚔️ Hunting",
	"items_digged":         "🦴 Digging",
	"casino_games_played":  "🎰 Casino",
	"bank_deposits":        "🏦 Bank",
	"items_sold_market":    "🏪 Market",
	"delve_completions":    "🏰 Delve",
	"pets_fed":             "🐾 Pet Care",
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: questssvc.New(s, cfg)}
	r.Slash("start", "Begin your journey!", c.onSlash)
	r.Slash("begin", "Begin your journey!", c.onSlash)
	r.Prefix("start", c.onPrefix)
	r.Prefix("begin", c.onPrefix)
	r.Component("start", "continue", c.onContinue)
}

func progressBar(current, target int) string {
	const totalBlocks = 10
	if target <= 0 {
		return ""
	}
	filled := current * totalBlocks / target
	if filled > totalBlocks {
		filled = totalBlocks
	}
	empty := totalBlocks - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

func (c *Cog) onSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.buildJourneyResponse(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	embed, comps := c.buildJourneyResponse(lang, userID)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func activityProgressStr(step *questssvc.QuestStep, progress int) string {
	targetStat, _ := step.Extra["target_stat"].(string)
	targetCount := toInt(step.Extra["target_count"])
	label := targetStat
	if l, ok := activityLabels[targetStat]; ok {
		label = l
	}
	bar := progressBar(progress, targetCount)
	return fmt.Sprintf("%s %s %d/%d", label, bar, progress, targetCount)
}

func (c *Cog) buildJourneyResponse(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	_, err := c.store.EnsureCharacter(userID)
	if err != nil {
		return components.Embed("", i18n.T("start.error", lang), 0xe74c3c), nil
	}

	startErr := c.svc.StartQuest(userID, "tutorial")
	if startErr == nil {
		def := c.svc.GetQuestDef("tutorial")
		step := def.Steps[0]
		text := i18n.T(step.TextKey, lang)
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("start.continue_btn", lang), components.EncodeOwner(userID, "start", "continue", "tutorial"), discordgo.SuccessButton),
			),
		}
		return components.Embed(i18n.T("start.begin_title", lang), text, 0x2ecc71), comps
	}

	uq, uqd, err := c.svc.GetQuestProgress(userID, "tutorial")
	if err == nil && uq.Status == "ACTIVE" {
		def := c.svc.GetQuestDef("tutorial")
		stepIdx := 0
		if uqd != nil {
			stepIdx = uqd.StepIndex
		}
		step := def.Steps[stepIdx]
		text := i18n.T(step.TextKey, lang)
		label := i18n.T("start.continue_btn", lang)
		switch step.Type {
		case questssvc.StepActivity:
			progStr := activityProgressStr(&step, uqd.ProgressValue)
			text = i18n.T("start.activity_prompt", lang, map[string]any{"quest": i18n.T(def.TitleKey, lang)})
			text += "\n\n" + progStr
			label = i18n.T("quests.activity_view_btn", lang)
		case questssvc.StepRequirement:
			label = i18n.T("quests.req_button", lang)
		case questssvc.StepBossBattle:
			label = i18n.T("quests.activity_view_btn", lang)
			text = i18n.T("start.boss_prompt", lang)
		}
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(label, components.EncodeOwner(userID, "start", "continue", "tutorial"), discordgo.SuccessButton),
			),
		}
		return components.Embed(i18n.T("start.begin_title", lang), text, 0x2ecc71), comps
	}

	if uq != nil && uq.Status == "COMPLETED" {
		return components.Embed(i18n.T("start.completed_title", lang), i18n.T("start.already_completed", lang), 0x2ecc71), nil
	}

	return components.Embed("", i18n.T("start.error", lang), 0xe74c3c), nil
}

func (c *Cog) onContinue(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	questID := rest[0]

	def := c.svc.GetQuestDef(questID)
	if def == nil {
		return
	}

	uq, uqd, err := c.svc.GetQuestProgress(userID, questID)
	if err != nil || uq.Status != "ACTIVE" {
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
				components.Embed("", i18n.T("start.error", lang), 0xe74c3c), nil))
		return
	}

	stepIdx := 0
	if uqd != nil {
		stepIdx = uqd.StepIndex
	}

	currStep := def.Steps[stepIdx]
	switch currStep.Type {
	case questssvc.StepActivity:
		if c.svc.IsActivityComplete(userID, questID) {
			if err := c.svc.AdvanceStep(userID, questID, ""); err != nil {
				_ = b.Session.InteractionRespond(i.Interaction,
					components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
						components.Embed("", i18n.T("start.error", lang), 0xe74c3c), nil))
				return
			}
		} else {
			uq2, uqd2, _ := c.svc.GetQuestProgress(userID, questID)
			text := i18n.T("start.activity_prompt", lang, map[string]any{"quest": i18n.T(def.TitleKey, lang)})
			if uq2.Status == "ACTIVE" && uqd2 != nil {
				text += "\n\n" + activityProgressStr(&currStep, uqd2.ProgressValue)
			}
			comps := []discordgo.MessageComponent{
				components.ActionRow(
					components.Button(i18n.T("quests.activity_view_btn", lang), components.EncodeOwner(userID, "start", "continue", questID), discordgo.SuccessButton),
				),
			}
			_ = b.Session.InteractionRespond(i.Interaction,
				components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
					components.Embed(i18n.T("start.begin_title", lang), text, 0x2ecc71), comps))
			return
		}

	case questssvc.StepRequirement:
		err := c.svc.FulfillRequirement(userID, questID)
		if err != nil {
			text := i18n.T(def.TitleKey, lang) + "\n\n❌ " + err.Error()
			comps := []discordgo.MessageComponent{
				components.ActionRow(
					components.Button(i18n.T("quests.req_button", lang), components.EncodeOwner(userID, "start", "continue", questID), discordgo.SuccessButton),
				),
			}
			_ = b.Session.InteractionRespond(i.Interaction,
				components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
					components.Embed(i18n.T("start.begin_title", lang), text, 0xe74c3c), comps))
			return
		}

	case questssvc.StepBossBattle:
		text := i18n.T("start.boss_prompt", lang)
		if currStep.Extra != nil {
			if bs, ok := currStep.Extra["boss_stage"].(int); ok {
				_ = bs
			}
		}
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button("⚔️ "+i18n.T("start.boss_fight_btn", lang), components.EncodeOwner(userID, "boss", "fight"), discordgo.DangerButton),
			),
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
				components.Embed(i18n.T("start.begin_title", lang), text, 0x992d22), comps))
		return

	default:
		if err := c.svc.AdvanceStep(userID, questID, ""); err != nil {
			_ = b.Session.InteractionRespond(i.Interaction,
				components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
					components.Embed("", i18n.T("start.error", lang), 0xe74c3c), nil))
			return
		}
	}

	uq2, uqd2, _ := c.svc.GetQuestProgress(userID, questID)
	if uq2.Status == "COMPLETED" {
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
				components.Embed(i18n.T("start.completed_title", lang), i18n.T("start.completed_desc", lang), 0x2ecc71), nil))
		return
	}

	nextIdx := 0
	if uqd2 != nil {
		nextIdx = uqd2.StepIndex
	}
	nextStep := def.Steps[nextIdx]
	text := i18n.T(nextStep.TextKey, lang)
	btnLabel := i18n.T("start.continue_btn", lang)
	if nextStep.Type == questssvc.StepActivity {
		text = i18n.T("start.activity_prompt", lang, map[string]any{"quest": i18n.T(def.TitleKey, lang)})
		progress := 0
		if uqd2 != nil {
			progress = uqd2.ProgressValue
		}
		text += "\n\n" + activityProgressStr(&nextStep, progress)
		btnLabel = i18n.T("quests.activity_view_btn", lang)
	}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(btnLabel, components.EncodeOwner(userID, "start", "continue", questID), discordgo.SuccessButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
			components.Embed("", text, 0x2ecc71), comps))
}
