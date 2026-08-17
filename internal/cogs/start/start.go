package start

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
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
	"items_mined":         "⛏️ Mining",
	"items_farmed":        "🌾 Farming",
	"items_fished":        "🎣 Fishing",
	"items_hunted":        "⚔️ Hunting",
	"items_digged":        "🦴 Digging",
	"casino_games_played": "🎰 Casino",
	"bank_deposits":       "🏦 Bank",
	"items_sold_market":   "🏪 Market",
	"delve_completions":   "🏰 Delve",
	"pets_fed":            "🐾 Pet Care",
}

// activityCommands maps an activity target stat to the slash command the player
// should type to complete it, shown next to each quest step.
var activityCommands = map[string]string{
	"items_mined":         "/mine",
	"items_farmed":        "/farm",
	"items_fished":        "/fish",
	"items_hunted":        "/hunt",
	"items_digged":        "/dig",
	"casino_games_played": "/casino",
	"bank_deposits":       "/deposit",
	"items_sold_market":   "/market",
	"delve_completions":   "/delve",
	"pets_fed":            "/pets",
	"hunt_evidence":       "/crimhunt",
	"stealth_progress":    "/steal",
	"blackjack_won":       "/blackjack",
	"slots_won":           "/casino",
	"wagers_won":          "/bet",
}

func stepCommand(targetStat string) string {
	return activityCommands[targetStat]
}

// categoryCommands maps item categories to the slash command that produces them,
// used to hint how to gather requirement items.
var categoryCommands = map[items.Category]string{
	items.Mining:     "/mine",
	items.Farming:    "/farm",
	items.Fishing:    "/fish",
	items.Archeology: "/dig",
}

// itemCategoryCommand returns the slash command that produces the given item,
// or "" when the item isn't a gatherable resource.
func itemCategoryCommand(itemID string) string {
	it := items.Get(itemID)
	if it == nil {
		return ""
	}
	return categoryCommands[it.Category]
}

// requirementCommands builds the slash commands the player should type to
// fulfill a requirement step, deduplicated and in a stable order.
func requirementCommands(step questssvc.QuestStep) string {
	if step.Extra == nil {
		return ""
	}
	var cmds []string
	if _, ok := step.Extra["req_owns_house"]; ok {
		cmds = append(cmds, "/house")
	}
	if _, ok := step.Extra["req_pet_level"]; ok {
		cmds = append(cmds, "/pets", "/hunt")
	}
	if _, ok := step.Extra["req_money"]; ok {
		cmds = append(cmds, "/daily", "/market")
	}
	if reqItems, ok := step.Extra["req_items"].(map[string]any); ok {
		itemIDs := make([]string, 0, len(reqItems))
		for id := range reqItems {
			itemIDs = append(itemIDs, id)
		}
		sort.Strings(itemIDs)
		for _, itemID := range itemIDs {
			if cmd := itemCategoryCommand(itemID); cmd != "" {
				cmds = append(cmds, cmd)
			}
		}
	}
	seen := make(map[string]bool)
	var out []string
	for _, cmd := range cmds {
		if !seen[cmd] {
			seen[cmd] = true
			out = append(out, cmd)
		}
	}
	return strings.Join(out, " ")
}

// stepCommandHint returns the slash command(s) the player should type to make
// progress on the given step, or "" when the step has no command to type
// (dialogue and choice steps only need the Continue button).
func stepCommandHint(step questssvc.QuestStep) string {
	switch step.Type {
	case questssvc.StepActivity:
		targetStat, _ := step.Extra["target_stat"].(string)
		return stepCommand(targetStat)
	case questssvc.StepRequirement:
		return requirementCommands(step)
	case questssvc.StepBossBattle:
		return "/boss"
	default:
		return ""
	}
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
	s := fmt.Sprintf("%s %s %d/%d", label, bar, progress, targetCount)
	if cmd := stepCommand(targetStat); cmd != "" {
		s += fmt.Sprintf(" — `%s`", cmd)
	}
	return s
}

// stepButtonDisabled reports whether a quest step's action button should be
// greyed out because its objective is not completed yet.
func (c *Cog) stepButtonDisabled(userID int64, questID string, step questssvc.QuestStep) bool {
	switch step.Type {
	case questssvc.StepActivity:
		return !c.svc.IsActivityComplete(userID, questID)
	case questssvc.StepBossBattle:
		return true
	case questssvc.StepRequirement:
		return c.svc.CheckRequirement(userID, questID) != nil
	default:
		return false
	}
}

// completedDesc builds the tutorial completion message, appending the final
// step's rewards so the player sees what they earned.
func (c *Cog) completedDesc(lang string, def *questssvc.QuestDef) string {
	text := i18n.T("start.completed_desc", lang)
	if def != nil && len(def.Steps) > 0 {
		if rs := questssvc.RewardSummary(lang, def.Steps[len(def.Steps)-1].Rewards); rs != "" {
			text += "\n\n" + i18n.T("quests.completed_rewards", lang, map[string]any{"rewards": rs})
		}
	}
	return text
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
			if cmd := stepCommandHint(step); cmd != "" {
				text += "\n" + i18n.T("quests.step_command", lang, map[string]any{"command": cmd})
			}
		case questssvc.StepBossBattle:
			label = i18n.T("quests.activity_view_btn", lang)
			text = i18n.T("start.boss_prompt", lang)
			if cmd := stepCommandHint(step); cmd != "" {
				text += "\n" + i18n.T("quests.step_command", lang, map[string]any{"command": cmd})
			}
		}
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				discordgo.Button{
					Label:    label,
					CustomID: components.EncodeOwner(userID, "start", "continue", "tutorial"),
					Style:    discordgo.SuccessButton,
					Disabled: c.stepButtonDisabled(userID, "tutorial", step),
				},
			),
		}
		return components.Embed(i18n.T("start.begin_title", lang), text, 0x2ecc71), comps
	}

	if uq != nil && uq.Status == "COMPLETED" {
		def := c.svc.GetQuestDef("tutorial")
		return components.Embed(i18n.T("start.completed_title", lang), c.completedDesc(lang, def), 0x2ecc71), nil
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
					discordgo.Button{
						Label:    i18n.T("quests.activity_view_btn", lang),
						CustomID: components.EncodeOwner(userID, "start", "continue", questID),
						Style:    discordgo.SuccessButton,
						Disabled: true,
					},
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
				components.Embed(i18n.T("start.completed_title", lang), c.completedDesc(lang, def), 0x2ecc71), nil))
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
	} else if cmd := stepCommandHint(nextStep); cmd != "" {
		text += "\n" + i18n.T("quests.step_command", lang, map[string]any{"command": cmd})
	}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			discordgo.Button{
				Label:    btnLabel,
				CustomID: components.EncodeOwner(userID, "start", "continue", questID),
				Style:    discordgo.SuccessButton,
				Disabled: c.stepButtonDisabled(userID, questID, nextStep),
			},
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
			components.Embed("", text, 0x2ecc71), comps))
}
