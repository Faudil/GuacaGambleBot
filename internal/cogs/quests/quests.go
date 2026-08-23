package quests

import (
	"errors"
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
	"guacagamblebot/internal/universe"
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

// requirementErrorDesc builds a helpful message explaining which quest
// requirements are missing and how to obtain them.
func (c *Cog) requirementErrorDesc(reqErr *questssvc.RequirementError, lang string) string {
	var lines []string
	if reqErr.NeedsHouse {
		lines = append(lines, i18n.T("quests.req_missing_house", lang))
	}
	if reqErr.MoneyNeeded > 0 {
		lines = append(lines, i18n.T("quests.req_missing_money", lang, map[string]any{
			"needed": reqErr.MoneyNeeded,
			"have":   reqErr.MoneyHave,
		}))
	}
	for _, m := range reqErr.MissingItems {
		lines = append(lines, i18n.T("quests.req_missing_item", lang, map[string]any{
			"item":   items.LocalizedName(m.ItemID, lang),
			"have":   m.Have,
			"needed": m.Needed,
		}))
	}
	if reqErr.PetLevelNeeded > 0 {
		lines = append(lines, i18n.T("quests.req_missing_pet_level", lang, map[string]any{
			"level": reqErr.PetLevelNeeded,
			"have":  reqErr.PetLevelHave,
		}))
	}
	if reqErr.ArtifactLevelNeeded > 0 {
		lines = append(lines, i18n.T("quests.req_missing_artifact_level", lang, map[string]any{
			"level": reqErr.ArtifactLevelNeeded,
			"have":  reqErr.ArtifactLevelHave,
		}))
	}
	if reqErr.ArtifactPointsNeeded > 0 {
		lines = append(lines, i18n.T("quests.req_missing_artifact_points", lang, map[string]any{
			"needed": reqErr.ArtifactPointsNeeded,
			"have":   reqErr.ArtifactPointsHave,
		}))
	}
	desc := strings.Join(lines, "\n")
	desc += "\n\n" + i18n.T("quests.req_farm_hint", lang)
	return desc
}

var activityLabels = map[string]string{
	"items_mined":            "⛏️ Mining",
	"items_farmed":           "🌾 Farming",
	"items_fished":           "🎣 Fishing",
	"items_hunted":           "⚔️ Hunting",
	"items_digged":           "🦴 Digging",
	"casino_games_played":    "🎰 Casino",
	"bank_deposits":          "🏦 Bank",
	"items_sold_market":      "🏪 Market",
	"delve_completions":      "🏰 Delve",
	"delve_floors_cleared":   "🏰 Delve",
	"zone_bosses_defeated":   "👑 Zone Boss",
	"expedition_completions": "🐾 Expedition",
	"pets_fed":               "🐾 Pet Care",
	"blackjack_won":          "🃏 Blackjack",
	"wagers_won":             "🎲 Wager",
}

// activityCommands maps an activity target stat to the slash command the player
// should type to complete it, shown next to each quest step.
var activityCommands = map[string]string{
	"items_mined":            "/mine",
	"items_farmed":           "/farm",
	"items_fished":           "/fish",
	"items_hunted":           "/hunt",
	"items_digged":           "/dig",
	"casino_games_played":    "/casino",
	"bank_deposits":          "/deposit",
	"items_sold_market":      "/market",
	"delve_completions":      "/delve",
	"delve_floors_cleared":   "/delve",
	"zone_bosses_defeated":   "/delve",
	"expedition_completions": "/expedition",
	"pets_fed":               "/pets",
	"hunt_evidence":          "/crimhunt",
	"stealth_progress":       "/steal",
	"blackjack_won":          "/blackjack",
	"slots_won":              "/casino",
	"wagers_won":             "/bet",
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
	if _, ok := step.Extra["req_artifact_level"]; ok {
		cmds = append(cmds, "/artifact", "/hunt")
	}
	if _, ok := step.Extra["req_artifact_points_spent"]; ok {
		cmds = append(cmds, "/artifact")
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

// stepButtonDisabled reports whether a quest step's action button should be
// greyed out because its objective is not completed yet. Dialogue and choice
// steps are always continuable.
func (c *Cog) stepButtonDisabled(userID int64, q questssvc.QuestInfo, step questssvc.QuestStep) bool {
	switch step.Type {
	case questssvc.StepActivity:
		return !c.svc.IsActivityComplete(userID, q.QuestID)
	case questssvc.StepBossBattle:
		return true
	case questssvc.StepRequirement:
		return c.svc.CheckRequirement(userID, q.QuestID) != nil
	default:
		return false
	}
}

// completedText builds the quest completion message, listing the rewards of
// the final step so the player sees what they earned.
func (c *Cog) completedText(lang string, def *questssvc.QuestDef) string {
	text := i18n.T("quests.completed_msg", lang, map[string]any{"title": i18n.T(def.TitleKey, lang)})
	if len(def.Steps) == 0 {
		return text
	}
	if rs := c.rewardSummary(lang, def.Steps[len(def.Steps)-1].Rewards); rs != "" {
		text += "\n\n" + i18n.T("quests.completed_rewards", lang, map[string]any{"rewards": rs})
	}
	return text
}

// rewardSummary renders a step's rewards as a single display string, or ""
// when the step grants nothing.
func (c *Cog) rewardSummary(lang string, r *questssvc.QuestReward) string {
	return questssvc.RewardSummary(lang, r)
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: questssvc.New(s, cfg)}
	r.Slash("quest", "View your active quests and progress.", c.onSlash)
	r.Prefix("quest", c.onPrefix)
	r.Prefix("q", c.onPrefix)
	r.Component("quest", "show", c.onShow)
	r.Component("quest", "advance", c.onAdvance)
}

// nextStepText builds the text shown for a quest step after advancing, appending
// the command to type when the step requires one.
func (c *Cog) nextStepText(lang string, def *questssvc.QuestDef, step questssvc.QuestStep) string {
	text := i18n.T(step.TextKey, lang)
	if step.Type == questssvc.StepActivity {
		targetStat, _ := step.Extra["target_stat"].(string)
		label := targetStat
		if l, ok := activityLabels[targetStat]; ok {
			label = l
		}
		text = i18n.T("quests.activity_intro", lang, map[string]any{"label": label, "quest": i18n.T(def.TitleKey, lang)})
	}
	if cmd := stepCommandHint(step); cmd != "" {
		text += "\n" + i18n.T("quests.step_command", lang, map[string]any{"command": cmd})
	}
	return text
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
	if err != nil {
		return components.Embed(i18n.T("quests.title", lang), i18n.T("quests.no_active", lang), 0x2ecc71), nil
	}

	granted, _ := c.svc.EnsureTutorialEgg(userID)

	var desc string
	var btns []discordgo.MessageComponent

	if granted {
		desc += i18n.T("quests.tutorial_egg_granted", lang) + "\n\n"
	} else if c.svc.HasUnhatchedTutorialEgg(userID) {
		desc += i18n.T("quests.tutorial_egg_hint", lang) + "\n\n"
	}

	// Post-tutorial questline directory: suggested next, available questlines
	// and locked ones with their unlock hints.
	hubDesc, hubBtns := c.questlineHubSection(lang, userID)
	desc += hubDesc
	btns = append(btns, hubBtns...)

	if len(quests) == 0 && strings.TrimSpace(desc) == "" {
		return components.Embed(i18n.T("quests.title", lang), i18n.T("quests.no_active", lang), 0x2ecc71), nil
	}

	for _, q := range quests {
		def := c.svc.GetQuestDef(q.QuestID)
		if def == nil {
			continue
		}
		title := i18n.T(def.TitleKey, lang)

		// Procedural daily quests are rendered from their recipe (custom data)
		// and managed in /daily — see the economy cog.
		if q.QuestID == "daily_quest" {
			desc += c.dailyQuestLine(lang, q) + "\n\n"
			continue
		}

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
			textKey := ""
			if step.Extra != nil {
				if s, ok := step.Extra["target_stat"].(string); ok {
					targetStat = s
				}
				targetCount = toInt(step.Extra["target_count"])
			} else if q.CustomData != nil {
				if s, ok := q.CustomData["target_stat"].(string); ok {
					targetStat = s
				}
				targetCount = toInt(q.CustomData["target_count"])
				if tk, ok := q.CustomData["text_key"].(string); ok {
					textKey = tk
				}
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
			if textKey != "" {
				desc += i18n.T(textKey, lang) + "\n"
			} else if targetCount == 0 {
				text := i18n.T(step.TextKey, lang)
				if len(text) > 1024 {
					text = text[:1024] + "..."
				}
				desc += text + "\n"
			}
			if progStr != "" {
				desc += progStr + "\n"
			}
			if cmd := stepCommand(targetStat); cmd != "" {
				desc += i18n.T("quests.step_command", lang, map[string]any{"command": cmd}) + "\n"
			}
			btnLabel = i18n.T("quests.activity_view_btn", lang)
		case questssvc.StepDialogue, questssvc.StepChoice:
			btnLabel = i18n.T("quests.continue_label", lang)
			text := i18n.T(step.TextKey, lang)
			if len(text) > 1024 {
				text = text[:1024] + "..."
			}
			desc += text + "\n"
		case questssvc.StepRequirement:
			btnLabel = i18n.T("quests.req_button", lang)
			text := i18n.T(step.TextKey, lang)
			if len(text) > 1024 {
				text = text[:1024] + "..."
			}
			desc += text + "\n"
			if cmd := stepCommandHint(step); cmd != "" {
				desc += i18n.T("quests.step_command", lang, map[string]any{"command": cmd}) + "\n"
			}
		case questssvc.StepBossBattle:
			btnLabel = i18n.T("quests.activity_view_btn", lang)
			text := i18n.T(step.TextKey, lang)
			if len(text) > 1024 {
				text = text[:1024] + "..."
			}
			desc += text + "\n"
			if cmd := stepCommandHint(step); cmd != "" {
				desc += i18n.T("quests.step_command", lang, map[string]any{"command": cmd}) + "\n"
			}
		default:
			btnLabel = i18n.T("quests.continue_label", lang)
		}

		btnLabel = title + " — " + btnLabel
		btns = append(btns, discordgo.Button{
			Label:    btnLabel,
			CustomID: components.EncodeOwner(userID, "quest", "advance", q.QuestID),
			Style:    discordgo.SuccessButton,
			Disabled: c.stepButtonDisabled(userID, q, step),
		})
		desc += "\n"
	}

	btns = append(btns, components.Button("🔄", components.EncodeOwner(userID, "quest", "show"), discordgo.SecondaryButton))

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

// dailyQuestLine renders the daily quest compactly for /quest. The full view
// with steps and the delivery button lives in /daily (economy cog).
func (c *Cog) dailyQuestLine(lang string, q questssvc.QuestInfo) string {
	cd := q.CustomData
	if cd == nil {
		return i18n.T("quests.daily.title", lang)
	}
	title := i18n.T("quests.daily.title", lang)
	if tk, ok := cd["title_key"].(string); ok && tk != "" {
		title = i18n.T(tk, lang)
	}
	stepsAny, _ := cd["steps"].([]any)
	if len(stepsAny) == 0 {
		return "📅 " + title
	}
	if q.StepIndex >= len(stepsAny) {
		return "✅ " + title
	}
	st, _ := stepsAny[q.StepIndex].(map[string]any)
	line := ""
	if tk, ok := st["text_key"].(string); ok && tk != "" {
		vars := map[string]any{"n": toInt(st["count"])}
		if kind, _ := st["kind"].(string); kind == "turnin" {
			if itemsMap, ok := st["items"].(map[string]any); ok {
				for itemID, qty := range itemsMap {
					vars["n"] = toInt(qty)
					vars["item"] = items.LocalizedName(itemID, lang)
				}
			}
		} else if zone, ok := st["zone"].(string); ok && zone != "" {
			vars["zone"] = i18n.T("hunt."+zone+"_zone", lang)
		}
		line = i18n.T(tk, lang, vars)
	}
	prog := ""
	if kind, _ := st["kind"].(string); kind == "activity" {
		if target := toInt(st["count"]); target > 0 {
			prog = fmt.Sprintf(" (%d/%d)", q.Progress, target)
		}
	}
	stepStr := i18n.T("quests.step_progress", lang, map[string]any{
		"current": q.StepIndex + 1,
		"total":   len(stepsAny),
	})
	return fmt.Sprintf("📅 **%s** (%s) — %s%s", title, stepStr, line, prog)
}

// questlineHubSection builds the post-tutorial questline directory shown at
// the top of /quest: a suggested-next breadcrumb, the available questlines
// (each with a button that opens the offering NPC's menu) and the locked ones
// with their unlock hint. It returns empty when the tutorial is not done or
// every questline is started or completed.
func (c *Cog) questlineHubSection(lang string, userID int64) (string, []discordgo.MessageComponent) {
	if !questssvc.TutorialCompleted(c.store, userID) {
		return "", nil
	}
	avail := questssvc.AvailableQuestlines(c.store, userID)
	locked := questssvc.LockedQuestlines(c.store, userID)
	if len(avail) == 0 && len(locked) == 0 {
		return "", nil
	}

	var desc string
	var btns []discordgo.MessageComponent

	if sug := questssvc.SuggestedNext(c.store, userID); sug != nil {
		npc := c.npcData(sug.NPCID)
		desc += i18n.T("quests.hub_suggested", lang, map[string]any{
			"emoji": npc.Emoji, "npc": npc.Name, "title": i18n.T(sug.TitleKey, lang),
		}) + "\n\n"
	}

	if len(avail) > 0 {
		desc += i18n.T("quests.hub_available", lang) + "\n"
		for _, d := range avail {
			npc := c.npcData(d.NPCID)
			desc += fmt.Sprintf("%s **%s** — %s\n", npc.Emoji, i18n.T(d.TitleKey, lang), npc.Name)
			btns = append(btns, components.Button(
				i18n.T("quests.talk_to_npc", lang, map[string]any{"npc": npc.Name}),
				components.EncodeOwner(userID, "npc", d.NPCID),
				discordgo.PrimaryButton))
		}
		desc += "\n"
	}

	if len(locked) > 0 {
		desc += i18n.T("quests.hub_locked", lang) + "\n"
		for _, d := range locked {
			desc += fmt.Sprintf("🔒 **%s** — %s\n", i18n.T(d.TitleKey, lang), c.unlockHint(lang, d))
		}
		desc += "\n"
	}

	return desc, btns
}

// npcData resolves an NPC's display data for the configured universe, falling
// back to the NPC id when the universe is unknown.
func (c *Cog) npcData(npcID string) *universe.NPCData {
	def := universe.Get(c.cfg.Universe)
	if def == nil {
		def = universe.Get("hoakhaven")
	}
	if def != nil {
		if n, ok := def.NPCs[npcID]; ok {
			return n
		}
	}
	return &universe.NPCData{ID: npcID, Name: npcID, Emoji: "❔"}
}

// unlockHint renders why a locked questline is not yet available.
func (c *Cog) unlockHint(lang string, d *questssvc.QuestDef) string {
	npc := c.npcData(d.NPCID)
	switch {
	case d.HintKey != "":
		return i18n.T(d.HintKey, lang, map[string]any{"npc": npc.Name})
	case d.PathReq == "hunter":
		return i18n.T("quests.unlock_hint.hunter", lang, map[string]any{"npc": npc.Name})
	case d.PathReq == "shadow":
		return i18n.T("quests.unlock_hint.shadow", lang, map[string]any{"npc": npc.Name})
	case d.BossReq > 0:
		return i18n.T("quests.unlock_hint.boss", lang, map[string]any{"stage": d.BossReq})
	case d.RepReq > 0:
		return i18n.T("quests.unlock_hint.affinity", lang, map[string]any{"npc": npc.Name, "level": d.RepReq})
	}
	return ""
}

// completionResponse renders a quest's completion message. Completing the
// tutorial swaps the plain message for a "what now" epilogue that points at
// the first questline and the /quest hub.
func (c *Cog) completionResponse(lang string, def *questssvc.QuestDef, userID int64) (string, []discordgo.MessageComponent) {
	if def.ID != "tutorial" {
		return c.completedText(lang, def), nil
	}
	text := i18n.T("quests.tutorial_epilogue", lang)
	if sug := questssvc.SuggestedNext(c.store, userID); sug != nil {
		npc := c.npcData(sug.NPCID)
		text += "\n\n" + i18n.T("quests.hub_suggested", lang, map[string]any{
			"emoji": npc.Emoji, "npc": npc.Name, "title": i18n.T(sug.TitleKey, lang),
		})
	}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("📜 "+i18n.T("quests.view_questlines_btn", lang),
				components.EncodeOwner(userID, "quest", "show"),
				discordgo.SuccessButton),
		),
	}
	return text, comps
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
	if currStep.Type == questssvc.StepBossBattle {
		embed, comps := c.buildQuestEmbed(lang, userID)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
		return
	}
	if currStep.Type == questssvc.StepActivity {
		if c.svc.IsActivityComplete(userID, questID) {
			if err := c.svc.AdvanceStep(userID, questID, ""); err != nil {
				interaction.RespondError(b, i, lang, "quests.title")
				return
			}
			uq2, uqd2, _ := c.svc.GetQuestProgress(userID, questID)
			if uq2.Status == "COMPLETED" {
				text, comps := c.completionResponse(lang, def, userID)
				_ = b.Session.InteractionRespond(i.Interaction,
					components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
						components.Embed("✅", text, 0x2ecc71), comps))
				return
			}
			nextIdx := 0
			if uqd2 != nil {
				nextIdx = uqd2.StepIndex
			}
			nextStep := def.Steps[nextIdx]
			text := c.nextStepText(lang, def, nextStep)
			title := i18n.T(def.TitleKey, lang) + " — " + i18n.T("quests.step_progress", lang, map[string]any{
				"current": nextIdx + 1,
				"total":   len(def.Steps),
			})
			btnLabel := i18n.T(def.TitleKey, lang) + " — " + i18n.T("quests.continue_label", lang)
			comps := []discordgo.MessageComponent{
				components.ActionRow(
					discordgo.Button{
						Label:    btnLabel,
						CustomID: components.EncodeOwner(userID, "quest", "advance", questID),
						Style:    discordgo.SuccessButton,
						Disabled: c.stepButtonDisabled(userID, questssvc.QuestInfo{QuestID: questID}, nextStep),
					},
				),
				components.ActionRow(
					components.Button("🔄", components.EncodeOwner(userID, "quest", "show"), discordgo.SecondaryButton),
				),
			}
			_ = b.Session.InteractionRespond(i.Interaction,
				components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
					components.Embed(title, text, 0x2ecc71), comps))
			return
		} else {
			embed, comps := c.buildQuestEmbed(lang, userID)
			_ = b.Session.InteractionRespond(i.Interaction,
				components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
			return
		}
	}
	if currStep.Type == questssvc.StepRequirement {
		reqTitle := "❌ " + i18n.T(def.TitleKey, lang) + " — " + i18n.T("quests.step_progress", lang, map[string]any{
			"current": stepIdx + 1,
			"total":   len(def.Steps),
		})
		err := c.svc.FulfillRequirement(userID, questID)
		if err != nil {
			var reqErr *questssvc.RequirementError
			if errors.As(err, &reqErr) {
				_ = b.Session.InteractionRespond(i.Interaction,
					components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
						components.Embed(reqTitle, c.requirementErrorDesc(reqErr, lang), 0xe74c3c), nil))
				return
			}
			_ = b.Session.InteractionRespond(i.Interaction,
				components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
					components.Embed(reqTitle, i18n.T("quests.req_missing", lang), 0xe74c3c), nil))
			return
		}
		uq2, uqd2, _ := c.svc.GetQuestProgress(userID, questID)
		if uq2.Status == "COMPLETED" {
			text, comps := c.completionResponse(lang, def, userID)
			_ = b.Session.InteractionRespond(i.Interaction,
				components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
					components.Embed("✅", text, 0x2ecc71), comps))
			return
		}
		nextStep := def.Steps[uqd2.StepIndex]
		text := c.nextStepText(lang, def, nextStep)
		doneTitle := "✅ " + i18n.T(def.TitleKey, lang) + " — " + i18n.T("quests.step_progress", lang, map[string]any{
			"current": uqd2.StepIndex + 1,
			"total":   len(def.Steps),
		})
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
				components.Embed(doneTitle, i18n.T("quests.req_done", lang)+"\n\n"+text, 0x2ecc71),
				[]discordgo.MessageComponent{
					components.ActionRow(
						discordgo.Button{
							Label:    i18n.T("quests.continue_label", lang),
							CustomID: components.EncodeOwner(userID, "quest", "advance", questID),
							Style:    discordgo.SuccessButton,
							Disabled: c.stepButtonDisabled(userID, questssvc.QuestInfo{QuestID: questID}, nextStep),
						},
					),
				}))
		return
	}

	if err := c.svc.AdvanceStep(userID, questID, ""); err != nil {
		interaction.RespondError(b, i, lang, "quests.title")
		return
	}

	uq2, uqd2, _ := c.svc.GetQuestProgress(userID, questID)
	if uq2.Status == "COMPLETED" {
		text, comps := c.completionResponse(lang, def, userID)
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
				components.Embed("✅", text, 0x2ecc71), comps))
		return
	}

	nextIdx := 0
	if uqd2 != nil {
		nextIdx = uqd2.StepIndex
	}
	nextStep := def.Steps[nextIdx]
	text := c.nextStepText(lang, def, nextStep)

	btnLabel := i18n.T(def.TitleKey, lang) + " " + i18n.T("quests.continue_label", lang)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			discordgo.Button{
				Label:    btnLabel,
				CustomID: components.EncodeOwner(userID, "quest", "advance", questID),
				Style:    discordgo.SuccessButton,
				Disabled: c.stepButtonDisabled(userID, questssvc.QuestInfo{QuestID: questID}, nextStep),
			},
		),
		components.ActionRow(
			components.Button("🔄", components.EncodeOwner(userID, "quest", "show"), discordgo.SecondaryButton),
		),
	}

	title := i18n.T(def.TitleKey, lang) + " " + i18n.T("quests.step_progress", lang, map[string]any{
		"current": nextIdx + 1,
		"total":   len(def.Steps),
	})
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage,
			components.Embed(title, text, 0x2ecc71), comps))
}
