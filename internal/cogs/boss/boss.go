package boss

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/assets"
	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	bosssvc "guacagamblebot/internal/service/boss"
	charsvc "guacagamblebot/internal/service/character"
	petsvc "guacagamblebot/internal/service/pets"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
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
	r.Component("boss", "fight", c.onFightButton)
	r.Component("boss", "show", c.onShowButton)
}

func (c *Cog) onSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps, image := c.show(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		assets.ResponseThumbnail(discordgo.InteractionResponseChannelMessageWithSource, embed, comps, image))
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
		o, errEmbed, errComps := c.prepareFight(interaction.ToInt64(m.Author.ID), lang, "")
		if o == nil {
			_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
				Embeds:     []*discordgo.MessageEmbed{errEmbed},
				Components: errComps,
			})
			return
		}
		sent, _ := s.ChannelMessageSendComplex(m.ChannelID, assets.Send(o.spawn, nil, o.image))
		msgID := ""
		if sent != nil {
			msgID = sent.ID
		}
		go c.animateFight(nil, nil, s, m.ChannelID, msgID, o, lang)
	default:
		embed, _, image := c.show(interaction.ToInt64(m.Author.ID), lang)
		_, _ = s.ChannelMessageSendComplex(m.ChannelID, assets.SendThumbnail(embed, nil, image))
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
// Used as a fallback when a fight is triggered without an explicit quest id
// (e.g. the legacy /start boss button).
func (c *Cog) findBossBattleQuest(userID int64) (string, *questssvc.QuestDef, *model.UserQuest, *model.UserQuestData) {
	for _, qid := range []string{"boss_league", "arena_rival", "tutorial"} {
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

// hasBossStep reports whether a quest line ever puts the player in front of a
// boss, i.e. it's relevant to show on the /boss screen.
func hasBossStep(def *questssvc.QuestDef) bool {
	for _, step := range def.Steps {
		if step.Type == questssvc.StepBossBattle {
			return true
		}
	}
	return false
}

// bossQuestLine is one active, boss-related quest currently sitting on a step
// the player can act on from /boss (its narrative beat or its boss fight).
type bossQuestLine struct {
	qid     string
	def     *questssvc.QuestDef
	uqd     *model.UserQuestData
	stepIdx int
}

// findBossQuestLines returns every active quest whose current step is either
// the narrative lead-up to a boss (StepDialogue) or the boss fight itself
// (StepBossBattle), across every quest line that ever features a boss. This
// lets a player choose between several boss lines at once (e.g. the arena
// rival vs. the league) instead of only ever seeing one.
func (c *Cog) findBossQuestLines(userID int64) []bossQuestLine {
	infos, err := c.qsvc.GetAllActiveQuests(userID)
	if err != nil {
		return nil
	}
	var lines []bossQuestLine
	for _, info := range infos {
		def := c.qsvc.GetQuestDef(info.QuestID)
		if def == nil || !hasBossStep(def) {
			continue
		}
		_, uqd, err := c.qsvc.GetQuestProgress(userID, info.QuestID)
		if err != nil || uqd == nil || uqd.StepIndex >= len(def.Steps) {
			continue
		}
		switch def.Steps[uqd.StepIndex].Type {
		case questssvc.StepDialogue, questssvc.StepBossBattle:
			lines = append(lines, bossQuestLine{qid: info.QuestID, def: def, uqd: uqd, stepIdx: uqd.StepIndex})
		}
	}
	return lines
}

// show renders every boss quest line the player currently has active, each as
// its own field with its own action button, so they can choose which boss to
// pursue (e.g. the league boss vs. the arena rival) instead of only ever
// seeing one at a time. It returns the picture asset of the first quest line
// with a boss image (empty when none apply) so callers can upload it.
func (c *Cog) show(userID int64, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent, string) {
	if uq, _, err := c.qsvc.GetQuestProgress(userID, "boss_league"); err != nil || uq == nil {
		// boss_league not started yet: auto-start it once any pet hits level 20+.
		pets, _ := petsvc.New(c.store, c.cfg).GetPets(userID)
		for _, p := range pets {
			if p.Level >= 20 {
				_ = c.qsvc.StartQuest(userID, "boss_league")
				break
			}
		}
	}

	lines := c.findBossQuestLines(userID)
	if len(lines) == 0 {
		if uq, _, _ := c.qsvc.GetQuestProgress(userID, "boss_league"); uq != nil && uq.Status == "COMPLETED" {
			embed := components.Embed(i18n.T("boss_league.title", lang), i18n.T("boss_league.champion", lang), 0xf1c40f)
			return embed, nil, ""
		}
		embed := components.Embed(i18n.T("boss_league.title", lang), i18n.T("boss_league.locked", lang), 0x992d22)
		return embed, nil, ""
	}

	var fields []*discordgo.MessageEmbedField
	var comps []discordgo.MessageComponent
	var bossImage string

	for _, line := range lines {
		lineTitle := i18n.T(line.def.TitleKey, lang)
		stepStr := i18n.T("quests.step_progress", lang, map[string]any{
			"current": line.stepIdx + 1,
			"total":   len(line.def.Steps),
		})

		step := line.def.Steps[line.stepIdx]
		var desc string
		var btn discordgo.MessageComponent

		switch step.Type {
		case questssvc.StepDialogue:
			desc = i18n.T(step.TextKey, lang)
			btn = components.Button(i18n.T("quests.continue_label", lang),
				components.EncodeOwner(userID, "quest", "advance", line.qid),
				discordgo.SuccessButton)
		case questssvc.StepBossBattle:
			bossStage := questStepBossStage(line.def, line.stepIdx)
			var bossName string
			if bossStage >= 0 && bossStage < len(bosssvc.BossLeague) {
				boss := bosssvc.BossLeague[bossStage]
				bossName = boss.NameEN
				bossDesc := boss.DescEN
				if lang == "fr" {
					bossName = boss.NameFR
					bossDesc = boss.DescFR
				}
				desc = fmt.Sprintf("**%s**\n*%s*\n\n", bossName, bossDesc)
				statsTxt := fmt.Sprintf("Lvl %d | %s | HP: %d | ATK: %d | DEF: %d | SPD: %d",
					boss.Level, boss.Species, boss.HP, boss.Atk, boss.Defense, boss.Speed)
				desc += fmt.Sprintf("🐾 %s\n", statsTxt)
				if bossImage == "" {
					bossImage = boss.Image
				}
			} else {
				desc = i18n.T(step.TextKey, lang)
			}
			desc += "\n" + i18n.T("boss_league.fight_hint", lang)
			label := "⚔️ " + i18n.T("quests.boss_fight_btn", lang)
			if bossName != "" {
				label = "⚔️ " + bossName
			}
			btn = components.Button(label,
				components.EncodeOwner(userID, "boss", "fight", line.qid),
				discordgo.DangerButton)
		default:
			continue
		}

		fields = append(fields, components.Field(lineTitle+" ("+stepStr+")", desc, false))
		comps = append(comps, components.ActionRow(btn))
	}

	comps = append(comps, components.ActionRow(
		components.Button("🔄", components.EncodeOwner(userID, "boss", "show"), discordgo.SecondaryButton),
	))

	embed := components.Embed(i18n.T("boss_league.title", lang), "", 0x992d22)
	embed.Fields = fields
	return embed, comps, bossImage
}

func (c *Cog) onFightButton(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	qid := ""
	if len(rest) >= 2 {
		qid = rest[0]
	}
	o, errEmbed, errComps := c.prepareFight(userID, lang, qid)
	if o == nil {
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, errEmbed, errComps))
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		assets.Response(discordgo.InteractionResponseUpdateMessage, o.spawn, nil, o.image))
	go c.animateFight(b, i, nil, "", "", o, lang)
}

func (c *Cog) onShowButton(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps, image := c.show(userID, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		assets.ResponseThumbnail(discordgo.InteractionResponseUpdateMessage, embed, comps, image))
}

// fightOutcome carries the frames and data needed to animate a boss fight.
type fightOutcome struct {
	spawn *discordgo.MessageEmbed
	final *discordgo.MessageEmbed
	comps []discordgo.MessageComponent
	turns []battle.BattleTurn
	image string // boss picture asset uploaded with the spawn/final frames

	petD  components.DisplayPet
	bossD components.DisplayPet
}

// prepareFight validates the boss fight, runs it, and prepares the animated
// frames. On failure it returns a nil outcome plus the error embed/buttons.
// qid names the quest line to fight; when empty it falls back to the legacy
// single-boss lookup (used by callers that predate multi-boss lines, e.g.
// /start's boss button).
func (c *Cog) prepareFight(userID int64, lang string, qid string) (*fightOutcome, *discordgo.MessageEmbed, []discordgo.MessageComponent) {
	backBtn := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("🔄", components.EncodeOwner(userID, "boss", "show"), discordgo.SecondaryButton),
		),
	}

	ok, _, err := c.store.CheckGameLimit(userID, "boss_fight", 5)
	if err != nil {
		return nil, components.Embed("❌", "Error checking limit.", 0xe74c3c), backBtn
	}
	if !ok {
		return nil, components.Embed("❌", i18n.T("economy.daily_footer", lang), 0xe74c3c), backBtn
	}

	var def *questssvc.QuestDef
	var uqd *model.UserQuestData
	if qid != "" {
		var uq *model.UserQuest
		uq, uqd, err = c.qsvc.GetQuestProgress(userID, qid)
		if err != nil || uq == nil || uq.Status != "ACTIVE" || uqd == nil {
			qid = ""
		} else {
			def = c.qsvc.GetQuestDef(qid)
			if def == nil {
				qid = ""
			}
		}
	}
	if qid == "" {
		qid, def, _, uqd = c.findBossBattleQuest(userID)
	}
	if qid == "" {
		return nil, components.Embed("❌", i18n.T("boss_league.locked", lang), 0xe74c3c), backBtn
	}

	stepIdx := uqd.StepIndex

	bossStage := questStepBossStage(def, stepIdx)
	if bossStage < 0 {
		return nil, components.Embed("❌", i18n.T("boss_league.no_battle_step", lang), 0xe74c3c), backBtn
	}
	if bossStage >= len(bosssvc.BossLeague) {
		return nil, components.Embed("❌", "Unknown boss stage.", 0xe74c3c), backBtn
	}

	pet, err := petsvc.New(c.store, c.cfg).GetActivePet(userID)
	if err != nil || pet == nil {
		return nil, components.Embed("❌", i18n.T("boss_league.no_pet", lang), 0xe74c3c), backBtn
	}

	if pet.HP <= 0 {
		return nil, components.Embed("❌", i18n.T("boss_league.pet_ko", lang, map[string]any{"name": pet.Nickname}), 0xe74c3c), backBtn
	}

	bossCfg := bosssvc.BossLeague[bossStage]
	bossPet := c.svc.CreateBossPet(bossCfg)

	userBP := c.petToBattlePet(pet)
	result := battle.Simulate(userBP, bossPet)
	// A lost fight leaves the pet wounded (or K.O.): persist the remaining HP
	// so the loss has consequences and the pet must be healed before the next
	// attempt. Wins keep the pet's pre-fight HP.
	if userBP.HP <= 0 {
		pet.HP = userBP.HP
	}
	_ = petsvc.New(c.store, c.cfg).UpdatePet(pet)
	_ = c.store.IncrementGameLimit(userID, "boss_fight")

	bossName := bossCfg.NameEN
	if lang == "fr" {
		bossName = bossCfg.NameFR
	}

	o := &fightOutcome{
		petD: components.DisplayPet{
			Name: pet.Nickname, Emoji: userBP.Emoji, Level: pet.Level,
			HP: userBP.MaxHP, MaxHP: userBP.MaxHP,
		},
		bossD: components.DisplayPet{
			Name: bossName, Emoji: bossPet.Emoji, Level: bossPet.Level,
			HP: bossPet.MaxHP, MaxHP: bossPet.MaxHP,
		},
		turns: result.Turns,
		image: bossCfg.Image,
	}
	o.comps = []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("⚔️ "+i18n.T("quests.boss_fight_btn", lang), components.EncodeOwner(userID, "boss", "fight", qid), discordgo.DangerButton),
			components.Button("🔄", components.EncodeOwner(userID, "boss", "show"), discordgo.SecondaryButton),
		),
	}
	o.spawn = c.bossRetroFrame(o, o.petD.MaxHP, o.bossD.MaxHP,
		[]string{i18n.T("boss_league.fight_intro", lang, map[string]any{
			"pet_emoji": o.petD.Emoji, "pet_name": o.petD.Name,
			"boss_name": o.bossD.Name, "boss_emoji": o.bossD.Emoji,
		})}, lang)
	assets.Image(o.spawn, o.image)

	if userBP.HP > 0 && bossPet.HP <= 0 {
		_, _ = c.svc.UpdateBalance(userID, bossCfg.RewardMoney)
		_ = achievement.IncrementStat(c.svc.DB(), userID, "pve_wins", 1)
		unlocks, _ := achievement.CheckAndUnlock(c.svc.DB(), userID)

		// Record boss victory in quest system (grants quest step rewards + trinket)
		_ = c.qsvc.RecordBossVictory(userID, bossStage)
		// Guaranteed boss trophy for any boss victory (farmable, per spec).
		_ = c.store.AddItemRaw(c.svc.DB(), userID, "boss_trophy", 1)
		// Per-stage stat so procedural daily quests can target the current boss.
		_ = c.store.RecordActivity(userID, "boss_stage_"+strconv.Itoa(bossStage), 1)

		o.final = c.bossRetroFrame(o, userBP.HP, bossPet.HP, result.Log, lang)
		o.final.Title = i18n.T("boss_league.victory", lang, map[string]any{"boss_name": o.bossD.Name})
		o.final.Color = 0x2ecc71

		if bossCfg.RewardMoney > 0 {
			o.final.Description += fmt.Sprintf("\n💵 +$%d", bossCfg.RewardMoney)
		}
		for item, qty := range bossCfg.RewardItems {
			o.final.Description += fmt.Sprintf("\n📦 %s x%d", items.LocalizedName(item, lang), qty)
		}
		o.final.Description += fmt.Sprintf("\n🏆 %s x1", items.LocalizedName("boss_trophy", lang))
		if bossCfg.XP > 0 {
			charLeveled, charLvl := charsvc.AddXP(c.store, userID, bossCfg.XP)
			o.final.Description += fmt.Sprintf("\n✨ +%d XP", bossCfg.XP)
			if charLeveled {
				o.final.Description += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": charLvl})
			}
		}

		// Check if quest completed
		_, uqd2, _ := c.qsvc.GetQuestProgress(userID, qid)
		if uqd2 != nil && uqd2.StepIndex >= len(def.Steps) {
			uq2, _, _ := c.qsvc.GetQuestProgress(userID, qid)
			if uq2 != nil && uq2.Status == "COMPLETED" {
				o.final.Description += "\n" + i18n.T("start.completed_desc", lang)
			}
		}

		if len(unlocks) > 0 {
			achStr := ""
			for _, a := range unlocks {
				achName := i18n.T("achievements."+a.ID+".name", lang)
				achStr += fmt.Sprintf("🎖️ %s (+%d Glory)\n", achName, a.Glory)
			}
			o.final.Fields = append(o.final.Fields, components.Field("🎖️ Achievements", achStr, false))
		}
	} else {
		o.final = c.bossRetroFrame(o, userBP.HP, bossPet.HP, result.Log, lang)
		o.final.Title = i18n.T("boss_league.defeat", lang, map[string]any{"pet_name": o.petD.Name, "boss_name": o.bossD.Name})
		o.final.Color = 0xe74c3c
		o.final.Description += "\n\n" + i18n.T("boss_league.try_again", lang)
		// A defeat may unlock an optional side quest line with an NPC mentor
		// (see questssvc.BossLossUnlocks). Notify the player and offer a
		// button that opens the mentor's NPC menu directly.
		if startedQuest, newly := c.qsvc.UnlockOnBossLoss(userID, bossStage); newly {
			if qDef := c.qsvc.GetQuestDef(startedQuest); qDef != nil && qDef.NPCID != "" {
				npcName := c.npcName(qDef.NPCID)
				o.final.Description += "\n\n" + i18n.T("boss_league.training_unlocked", lang, map[string]any{"npc": npcName})
				o.comps = append(o.comps, components.ActionRow(
					components.Button("💬 "+i18n.T("boss_league.talk_to_npc", lang, map[string]any{"npc": npcName}),
						components.EncodeOwner(userID, "npc", qDef.NPCID),
						discordgo.SuccessButton),
				))
			}
		}
	}
	assets.Image(o.final, o.image)
	return o, nil, nil
}

// npcName resolves an NPC's display name for the configured universe, falling
// back to the NPC id when the universe is unknown.
func (c *Cog) npcName(npcID string) string {
	def := universe.Get(c.cfg.Universe)
	if def == nil {
		def = universe.Get("hoakhaven")
	}
	if def != nil {
		if n, ok := def.NPCs[npcID]; ok {
			return n.Name
		}
	}
	return npcID
}

// animateFight replays the fight turns with live HP bars. When msgID is empty
// the frames edit the interaction response; when msgID is set they edit the
// given channel message (prefix path).
func (c *Cog) animateFight(b *interaction.Bot, i *discordgo.InteractionCreate, s *discordgo.Session, channelID, msgID string, o *fightOutcome, lang string) {
	if msgID == "" && b == nil {
		return
	}
	edit := func(frame *discordgo.MessageEmbed, comps []discordgo.MessageComponent) {
		if msgID == "" {
			// Discord drops attachments absent from an edit payload, so the
			// boss picture is only re-uploaded on frames that carry it (the
			// final result frame), never on the ~12 animation frames.
			if frame.Image != nil {
				_, _ = b.Session.InteractionResponseEdit(i.Interaction, assets.EditResponse(frame, comps, o.image))
				return
			}
			_, _ = b.Session.InteractionResponseEdit(i.Interaction, components.WebhookEditResponse(frame, comps))
		} else {
			data := &discordgo.MessageEdit{
				Channel: channelID,
				ID:      msgID,
				Embeds:  &[]*discordgo.MessageEmbed{frame},
			}
			if frame.Image != nil {
				if f := assets.File(o.image); f != nil {
					data.Files = []*discordgo.File{f}
				}
			}
			_, _ = s.ChannelMessageEditComplex(data)
		}
	}
	interaction.AnimateFight(
		o.turns,
		func(journal []string, t battle.BattleTurn) *discordgo.MessageEmbed {
			return c.bossRetroFrame(o, t.Pet1HP, t.Pet2HP, journal, lang)
		},
		edit,
		func(_ []string) { edit(o.final, o.comps) },
	)
}

// bossRetroFrame renders one retro RPG battle frame: pet vs boss with colored
// HP bars and the combat journal.
func (c *Cog) bossRetroFrame(o *fightOutcome, petHP, bossHP int, journal []string, lang string) *discordgo.MessageEmbed {
	o.petD.HP = petHP
	o.bossD.HP = bossHP
	o.petD.IsKO = petHP <= 0
	o.bossD.IsKO = bossHP <= 0
	return components.FightFrameEmbed(
		i18n.T("boss_league.title", lang),
		o.petD, o.bossD,
		components.FightLabelsFor(lang, i18n.T("hunt.vs", lang)),
		journal,
	)
}

func (c *Cog) petToBattlePet(pet *model.UserPet) *battle.BattlePet {
	pt := petsvc.PetTypes[pet.PetType]
	emoji := "🐾"
	if pt != nil {
		emoji = pt.Emoji
	}
	var skills []model.UserPetSkill
	c.store.DB.Where("pet_id = ?", pet.ID).Find(&skills)
	skillIDs := make([]string, 0, len(skills))
	for _, s := range skills {
		skillIDs = append(skillIDs, s.SkillID)
	}
	return &battle.BattlePet{
		ID: pet.ID, Nickname: pet.Nickname, Emoji: emoji, PetType: pet.PetType,
		Level: pet.Level, HP: pet.HP, MaxHP: pet.MaxHP,
		Atk: pet.Atk, Defense: pet.Defense, Speed: pet.Speed,
		DGE: pet.DGE, ACC: pet.ACC, CritC: pet.CritC, CritD: pet.CritD, SpcC: pet.SpcC,
		Skills: skillIDs,
	}
}
