package boss

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	bosssvc "guacagamblebot/internal/service/boss"
	petsvc "guacagamblebot/internal/service/pets"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *bosssvc.Service
	qsvc  *questssvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: bosssvc.New(s, cfg), qsvc: questssvc.New(s, cfg)}
	r.Slash("boss", "Boss League - Combattez des boss", c.onSlash)
	r.Prefix("boss", c.onPrefix)
	r.Prefix("league", c.onPrefix)
	r.Prefix("bl", c.onPrefix)
}

func (c *Cog) onSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.show(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	parts := strings.Fields(m.Content)
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}
	switch sub {
	case "fight":
		embed := c.fight(interaction.ToInt64(m.Author.ID), lang)
		_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
		})
	default:
		embed, _ := c.show(interaction.ToInt64(m.Author.ID), lang)
		_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
		})
	}
}

// questStepBossStage returns the boss_stage from the quest step's Extra, or -1.
func questStepBossStage(def *questssvc.QuestDef, stepIdx int) int {
	if stepIdx < 0 || stepIdx >= len(def.Steps) {
		return -1
	}
	step := def.Steps[stepIdx]
	if step.Type != questssvc.StepBossBattle {
		return -1
	}
	stage, _ := step.Extra["boss_stage"].(int)
	return stage
}

// findBossBattleQuest finds any active quest whose current step is a StepBossBattle.
// It checks boss_league first, then falls back to any other active quest.
func (c *Cog) findBossBattleQuest(userID int64) (string, *questssvc.QuestDef, *model.UserQuest, *model.UserQuestData) {
	for _, qid := range []string{"boss_league", "tutorial"} {
		uq, uqd, err := c.qsvc.GetQuestProgress(userID, qid)
		if err != nil || uq == nil || uq.Status != "ACTIVE" || uqd == nil {
			continue
		}
		def := c.qsvc.GetQuestDef(qid)
		if def == nil || uqd.StepIndex >= len(def.Steps) {
			continue
		}
		if def.Steps[uqd.StepIndex].Type == questssvc.StepBossBattle {
			return qid, def, uq, uqd
		}
	}
	return "", nil, nil, nil
}

func (c *Cog) show(userID int64, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	qid, def, uq, uqd := c.findBossBattleQuest(userID)
	if qid == "" {
		// Try to auto-start boss_league if any pet is level 20+
		pets, _ := petsvc.New(c.store, c.cfg).GetPets(userID)
		hasLv20 := false
		for _, p := range pets {
			if p.Level >= 20 {
				hasLv20 = true
				break
			}
		}
		if hasLv20 {
			_ = c.qsvc.StartQuest(userID, "boss_league")
			qid, def, uq, uqd = c.findBossBattleQuest(userID)
		}
	}

	if qid == "" {
		embed := components.Embed(
			i18n.T("boss_league.title", lang),
			i18n.T("boss_league.locked", lang),
			0x992d22,
		)
		return embed, nil
	}
	stepIdx := 0
	if uqd != nil {
		stepIdx = uqd.StepIndex
	}

	title := i18n.T(def.TitleKey, lang)
	stepStr := i18n.T("quests.step_progress", lang, map[string]any{
		"current": stepIdx + 1,
		"total":   len(def.Steps),
	})

	if uq.Status == "COMPLETED" {
		if qid == "boss_league" {
			embed := components.Embed(title, i18n.T("boss_league.champion", lang), 0xf1c40f)
			return embed, nil
		}
		embed := components.Embed(title, "✅ Quest completed!", 0x2ecc71)
		return embed, nil
	}

	step := def.Steps[stepIdx]
	var desc string
	var btns []discordgo.MessageComponent

	switch step.Type {
	case questssvc.StepDialogue:
		desc = i18n.T(step.TextKey, lang)
		btns = append(btns,
			components.Button(i18n.T("quests.continue_label", lang),
				components.Encode("quest", "advance", qid),
				discordgo.SuccessButton),
		)
	case questssvc.StepBossBattle:
		bossStage := questStepBossStage(def, stepIdx)
		if bossStage >= 0 && bossStage < len(bosssvc.BossLeague) {
			boss := bosssvc.BossLeague[bossStage]
			bossName := boss.NameEN
			bossDesc := boss.DescEN
			if lang == "fr" {
				bossName = boss.NameFR
				bossDesc = boss.DescFR
			}
			desc = fmt.Sprintf("**%s**\n*%s*\n\n", bossName, bossDesc)
			statsTxt := fmt.Sprintf("Lvl %d | %s | HP: %d | ATK: %d | DEF: %d | SPD: %d",
				boss.Level, boss.Species, boss.HP, boss.Atk, boss.Defense, boss.Speed)
			desc += fmt.Sprintf("🐾 %s\n", statsTxt)
		} else {
			desc = i18n.T(step.TextKey, lang)
		}
		desc += "\n" + i18n.T("boss_league.fight_hint", lang)
		btns = append(btns,
			components.Button(i18n.T("quests.activity_view_btn", lang),
				components.Encode("quest", "advance", qid),
				discordgo.SuccessButton),
		)
	}

	btns = append(btns,
		components.Button("🔄", components.Encode("quest", "show"), discordgo.SecondaryButton),
	)

	embed := components.Embed(
		title+" ("+stepStr+")",
		desc,
		0x992d22,
	)

	var comps []discordgo.MessageComponent
	for _, btn := range btns {
		comps = append(comps, components.ActionRow(btn))
	}
	return embed, comps
}

func (c *Cog) fight(userID int64, lang string) *discordgo.MessageEmbed {
	ok, _, err := c.store.CheckGameLimit(userID, "boss_fight", 5)
	if err != nil {
		return components.Embed("❌", "Error checking limit.", 0xe74c3c)
	}
	if !ok {
		return components.Embed("❌", i18n.T("economy.daily_footer", lang), 0xe74c3c)
	}

	qid, def, _, uqd := c.findBossBattleQuest(userID)
	if qid == "" {
		return components.Embed("❌", i18n.T("boss_league.locked", lang), 0xe74c3c)
	}

	stepIdx := uqd.StepIndex

	bossStage := questStepBossStage(def, stepIdx)
	if bossStage < 0 {
		return components.Embed("❌", i18n.T("boss_league.no_battle_step", lang), 0xe74c3c)
	}
	if bossStage >= len(bosssvc.BossLeague) {
		return components.Embed("❌", "Unknown boss stage.", 0xe74c3c)
	}

	pet, err := petsvc.New(c.store, c.cfg).GetActivePet(userID)
	if err != nil || pet == nil {
		return components.Embed("❌", i18n.T("boss_league.no_pet", lang), 0xe74c3c)
	}

	if pet.HP <= 0 {
		return components.Embed("❌", i18n.T("boss_league.pet_ko", lang, map[string]any{"name": pet.Nickname}), 0xe74c3c)
	}

	bossCfg := bosssvc.BossLeague[bossStage]
	bossPet := c.svc.CreateBossPet(bossCfg)

	userBP := petToBattlePet(pet)
	battle.Simulate(userBP, bossPet)
	_ = petsvc.New(c.store, c.cfg).UpdatePet(pet)
	_ = c.store.IncrementGameLimit(userID, "boss_fight")

	if userBP.HP > 0 && bossPet.HP <= 0 {
		_, _ = c.svc.UpdateBalance(userID, bossCfg.RewardMoney)
		_ = achievement.IncrementStat(c.svc.DB(), userID, "pve_wins", 1)
		unlocks, _ := achievement.CheckAndUnlock(c.svc.DB(), userID)

		// Record boss victory in quest system (grants quest step rewards + trinket)
		_ = c.qsvc.RecordBossVictory(userID, bossStage)

		bossName := bossCfg.NameEN
		if lang == "fr" {
			bossName = bossCfg.NameFR
		}

		desc := fmt.Sprintf("🏆 **Victory!** You defeated **%s**!\n\n", bossName)
		if bossCfg.RewardMoney > 0 {
			desc += fmt.Sprintf("💵 +$%d\n", bossCfg.RewardMoney)
		}
		for item, qty := range bossCfg.RewardItems {
			desc += fmt.Sprintf("📦 %s x%d\n", items.DisplayName(item), qty)
		}

		// Check if quest completed
		_, uqd2, _ := c.qsvc.GetQuestProgress(userID, qid)
		if uqd2 != nil && uqd2.StepIndex >= len(def.Steps) {
			uq2, _, _ := c.qsvc.GetQuestProgress(userID, qid)
			if uq2 != nil && uq2.Status == "COMPLETED" {
				desc += "\n" + i18n.T("start.completed_desc", lang)
			}
		}

		embed := components.Embed(i18n.T("boss_league.victory", lang, map[string]any{"boss_name": bossName}), desc, 0x2ecc71)
		if len(unlocks) > 0 {
			achStr := ""
			for _, a := range unlocks {
				achName := i18n.T("achievements."+a.ID+".name", lang)
				achStr += fmt.Sprintf("🎖️ %s (+%d Glory)\n", achName, a.Glory)
			}
			embed.Fields = append(embed.Fields, components.Field("🎖️ Achievements", achStr, false))
		}
		return embed
	}

	bossName := bossCfg.NameEN
	if lang == "fr" {
		bossName = bossCfg.NameFR
	}
	return components.Embed(
		i18n.T("boss_league.defeat", lang, map[string]any{"pet_name": pet.Nickname, "boss_name": bossName}),
		"Train your pet and try again!",
		0xe74c3c,
	)
}

func petToBattlePet(pet *model.UserPet) *battle.BattlePet {
	pt := petsvc.PetTypes[pet.PetType]
	emoji := "🐾"
	if pt != nil {
		emoji = pt.Emoji
	}
	return &battle.BattlePet{
		ID: pet.ID, Nickname: pet.Nickname, Emoji: emoji,
		Level: pet.Level, HP: pet.HP, MaxHP: pet.MaxHP,
		Atk: pet.Atk, Defense: pet.Defense, Speed: pet.Speed,
		DGE: pet.DGE, ACC: pet.ACC, CritC: pet.CritC, CritD: pet.CritD, SpcC: pet.SpcC,
	}
}


