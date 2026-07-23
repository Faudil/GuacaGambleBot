package quests

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
	"items_mined":  "⛏️ Mining",
	"items_farmed": "🌾 Farming",
	"items_fished": "🎣 Fishing",
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: questssvc.New(s, cfg)}
	r.Slash("quest", "View your active quests and progress.", c.onSlash)
	r.Prefix("quest", c.onPrefix)
	r.Prefix("q", c.onPrefix)
	r.Component("quest", "show", c.onShow)
	r.Component("quest", "advance", c.onAdvance)
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

func (c *Cog) buildQuestEmbed(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	quests, err := c.svc.GetAllActiveQuests(userID)
	if err != nil || len(quests) == 0 {
		return components.Embed(i18n.T("quests.title", lang), i18n.T("quests.no_active", lang), 0x2ecc71), nil
	}

	var desc string
	var btns []discordgo.MessageComponent

	for _, q := range quests {
		def := c.svc.GetQuestDef(q.QuestID)
		if def == nil {
			continue
		}
		title := i18n.T(def.TitleKey, lang)
		stepStr := i18n.T("quests.step_progress", lang, map[string]any{
			"current": q.StepIndex + 1,
			"total":   q.TotalSteps,
		})

		desc += fmt.Sprintf("**%s** (%s)\n", title, stepStr)

		step := def.Steps[q.StepIndex]
		var btnLabel string
		switch step.Type {
		case questssvc.StepActivity:
			targetStat := ""
			targetCount := 0
			if step.Extra != nil {
				if s, ok := step.Extra["target_stat"].(string); ok {
					targetStat = s
				}
				targetCount = toInt(step.Extra["target_count"])
			}
			if targetCount == 0 && q.Progress > 0 {
				targetCount = q.Progress
			}
			label := targetStat
			if l, ok := activityLabels[targetStat]; ok {
				label = l
			}
			progStr := ""
			if targetCount > 0 {
				progStr = i18n.T("quests.step_activity_progress", lang, map[string]any{
					"current":  q.Progress,
					"target":   targetCount,
					"activity": label,
				})
				bar := progressBar(q.Progress, targetCount)
				if bar != "" {
					progStr += "\n" + bar
				}
			}
			if progStr != "" {
				desc += progStr + "\n"
			}
			btnLabel = i18n.T("quests.activity_view_btn", lang)
		case questssvc.StepDialogue, questssvc.StepChoice:
			btnLabel = i18n.T("quests.continue_label", lang)
			text := i18n.T(step.TextKey, lang)
			if len(text) > 200 {
				text = text[:200] + "..."
			}
			desc += text + "\n"
		default:
			btnLabel = i18n.T("quests.continue_label", lang)
		}

		btns = append(btns, components.Button(btnLabel, components.Encode("quest", "advance", q.QuestID), discordgo.SuccessButton))
		desc += "\n"
	}

	btns = append(btns, components.Button("🔄", components.Encode("quest", "show"), discordgo.SecondaryButton))

	var comps []discordgo.MessageComponent
	for len(btns) > 0 {
		n := 5
		if len(btns) < n {
			n = len(btns)
		}
		comps = append(comps, components.ActionRow(btns[:n]...))
		btns = btns[n:]
	}

	return components.Embed(i18n.T("quests.title", lang), desc, 0x2ecc71), comps
}

func (c *Cog) onSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.buildQuestEmbed(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	embed, comps := c.buildQuestEmbed(lang, userID)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) onShow(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.buildQuestEmbed(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onAdvance(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	questID := rest[0]

	def := c.svc.GetQuestDef(questID)
	if def == nil {
		interaction.RespondError(b, i, lang, "quests.title")
		return
	}

	uq, uqd, err := c.svc.GetQuestProgress(userID, questID)
	if err != nil || uq.Status != "ACTIVE" {
		interaction.RespondError(b, i, lang, "quests.no_active")
		return
	}

	stepIdx := 0
	if uqd != nil {
		stepIdx = uqd.StepIndex
	}

	currStep := def.Steps[stepIdx]
	if currStep.Type == questssvc.StepActivity ||
		currStep.Type == questssvc.StepRequirement ||
		currStep.Type == questssvc.StepBossBattle {
		interaction.RespondError(b, i, lang, "quests.complete_objective_first")
		return
	}

	if err := c.svc.AdvanceStep(userID, questID, ""); err != nil {
		interaction.RespondError(b, i, lang, "quests.title")
		return
	}

	uq2, uqd2, _ := c.svc.GetQuestProgress(userID, questID)
	if uq2.Status == "COMPLETED" {
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
				components.Embed("✅", i18n.T("quests.completed_msg", lang, map[string]any{"title": i18n.T(def.TitleKey, lang)}), 0x2ecc71), nil))
		return
	}

	nextIdx := 0
	if uqd2 != nil {
		nextIdx = uqd2.StepIndex
	}
	nextStep := def.Steps[nextIdx]
	text := i18n.T(nextStep.TextKey, lang)
	if nextStep.Type == questssvc.StepActivity {
		targetStat := ""
		if s, ok := nextStep.Extra["target_stat"].(string); ok {
			targetStat = s
		}
		label := targetStat
		if l, ok := activityLabels[targetStat]; ok {
			label = l
		}
		text = i18n.T("quests.activity_intro", lang, map[string]any{"label": label, "quest": i18n.T(def.TitleKey, lang)})
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("quests.continue_label", lang), components.Encode("quest", "advance", questID), discordgo.SuccessButton),
		),
		components.ActionRow(
			components.Button("🔄", components.Encode("quest", "show"), discordgo.SecondaryButton),
		),
	}

	title := i18n.T(def.TitleKey, lang) + " — " + i18n.T("quests.step_progress", lang, map[string]any{
		"current": nextIdx + 1,
		"total":   len(def.Steps),
	})
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
			components.Embed(title, text, 0x2ecc71), comps))
}
