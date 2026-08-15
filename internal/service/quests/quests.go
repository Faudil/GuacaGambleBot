package quests

import (
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type QuestStepType int

const (
	StepDialogue QuestStepType = iota
	StepChoice
	StepActivity
	StepRequirement
	StepBossBattle
)

// MissingItem describes an item requirement that is not yet satisfied.
type MissingItem struct {
	ItemID string
	Needed int
	Have   int
}

// RequirementError carries structured information about which quest
// requirements are not yet satisfied, so handlers can show helpful guidance.
type RequirementError struct {
	MoneyNeeded    int
	MoneyHave      int
	MissingItems   []MissingItem
	NeedsHouse     bool
	PetLevelNeeded int
	PetLevelHave   int
}

func (e *RequirementError) Error() string {
	return "quest requirements not satisfied"
}

type QuestReward struct {
	Money         int
	XP            int
	Crowns        int
	ItemIDs       []string
	AchievementID string
}

type QuestStep struct {
	Type    QuestStepType
	TextKey string
	Rewards *QuestReward
	Extra   map[string]any
}

type QuestDef struct {
	ID       string
	Type     string
	NPCID    string
	TitleKey string
	DescKey  string
	Steps    []QuestStep
	RepReq   int
	Unlocks  []string
}

var QuestRegistry = map[string]*QuestDef{
	// --- Criminality: The Masked Shadow Falls ---
	// The branching choice is handled by the criminality cog.
	// These sub-quests are started after the player chooses their path.
	"masked_shadow_falls_hunter": {
		ID: "masked_shadow_falls_hunter", Type: "main", TitleKey: "quests.masked_shadow.title", DescKey: "quests.masked_shadow.desc_hunter",
		Steps: []QuestStep{
			{Type: StepDialogue, TextKey: "quests.masked_shadow.hunter_step0", Rewards: &QuestReward{Money: 200}},
			{Type: StepActivity, TextKey: "quests.masked_shadow.hunter_step1", Extra: map[string]any{"target_stat": "hunt_evidence", "target_count": 3}},
			{Type: StepBossBattle, TextKey: "quests.masked_shadow.hunter_step2", Extra: map[string]any{"boss_id": "tracker_trial"}},
			{Type: StepDialogue, TextKey: "quests.masked_shadow.hunter_step3", Rewards: &QuestReward{Money: 500, ItemIDs: []string{"hounds_cloak"}}},
		},
	},
	"masked_shadow_falls_shadow": {
		ID: "masked_shadow_falls_shadow", Type: "main", TitleKey: "quests.masked_shadow.title", DescKey: "quests.masked_shadow.desc_shadow",
		Steps: []QuestStep{
			{Type: StepDialogue, TextKey: "quests.masked_shadow.shadow_step0", Rewards: &QuestReward{Money: 200}},
			{Type: StepActivity, TextKey: "quests.masked_shadow.shadow_step1", Extra: map[string]any{"target_stat": "stealth_progress", "target_count": 1}},
			{Type: StepDialogue, TextKey: "quests.masked_shadow.shadow_step2"},
			{Type: StepDialogue, TextKey: "quests.masked_shadow.shadow_step3", Rewards: &QuestReward{Money: 500, ItemIDs: []string{"shadow_cowl"}}},
		},
	},
	"masked_shadow_falls_forgive": {
		ID: "masked_shadow_falls_forgive", Type: "main", TitleKey: "quests.masked_shadow.title", DescKey: "quests.masked_shadow.desc_forgive",
		Steps: []QuestStep{
			{Type: StepDialogue, TextKey: "quests.masked_shadow.forgive_step0"},
			{Type: StepDialogue, TextKey: "quests.masked_shadow.forgive_step1"},
		},
	},

	"tutorial": {
		ID: "tutorial", Type: "main", TitleKey: "quests.day0_welcome.title", DescKey: "quests.day0_welcome.description",
		Steps: []QuestStep{
			// Day 0 — Arrival
			{Type: StepDialogue, TextKey: "quests.day0_welcome.step0_dialogue", Rewards: &QuestReward{Money: 100}},
			// Day 1 — Mining + Fishing
			{Type: StepActivity, TextKey: "quests.day1_strata.step1_activity", Extra: map[string]any{"target_stat": "items_mined", "target_count": 2}},
			{Type: StepDialogue, TextKey: "quests.day1_strata.step2_transition", Rewards: &QuestReward{Money: 100, ItemIDs: []string{"worm", "worm"}}},
			{Type: StepActivity, TextKey: "quests.day1_strata.step3_activity", Extra: map[string]any{"target_stat": "items_fished", "target_count": 1}},
			{Type: StepDialogue, TextKey: "quests.day1_strata.step4_dialogue", Rewards: &QuestReward{Money: 100, ItemIDs: []string{"wheat_seed", "wheat_seed", "growth_elixir"}}},
			// Day 2 — Farming + Hunting + Pet Care
			{Type: StepActivity, TextKey: "quests.day2_alchemy.step1_activity", Extra: map[string]any{"target_stat": "items_farmed", "target_count": 2}},
			{Type: StepDialogue, TextKey: "quests.day2_alchemy.step2_transition"},
			{Type: StepDialogue, TextKey: "quests.day4_will.step0_dialogue", Rewards: &QuestReward{Money: 300, ItemIDs: []string{"forest_egg"}}},
			{Type: StepActivity, TextKey: "quests.day4_will.step1_activity", Extra: map[string]any{"target_stat": "items_hunted", "target_count": 2}},
			{Type: StepDialogue, TextKey: "quests.day4_will.step2_dialogue"},
			{Type: StepActivity, TextKey: "quests.day4_will.step3_activity", Extra: map[string]any{"target_stat": "pets_fed", "target_count": 1}},
			// Day 3 — Archeology (dig for the capsule)
			{Type: StepDialogue, TextKey: "quests.day4_will.step4_dialogue", Rewards: &QuestReward{Money: 300}},
			{Type: StepActivity, TextKey: "quests.day2_alchemy.step3_activity", Extra: map[string]any{"target_stat": "items_digged", "target_count": 2}},
			{Type: StepDialogue, TextKey: "quests.day2_alchemy.step4_dialogue", Rewards: &QuestReward{ItemIDs: []string{"wheat"}}},
			// Day 4 — Market (sell goods to fund a home)
			{Type: StepActivity, TextKey: "quests.day5_odds.step3_activity", Extra: map[string]any{"target_stat": "items_sold_market", "target_count": 1}},
			{Type: StepDialogue, TextKey: "quests.day5_odds.step4_dialogue", Rewards: &QuestReward{Money: 300}},
			// Day 5 — House + Bank
			{Type: StepRequirement, TextKey: "quests.day3_base.step0_req", Extra: map[string]any{"req_owns_house": true}},
			{Type: StepDialogue, TextKey: "quests.day3_base.step1_transition"},
			{Type: StepActivity, TextKey: "quests.day3_base.step2_activity", Extra: map[string]any{"target_stat": "bank_deposits", "target_count": 1}},
			// Day 6 — Casino
			{Type: StepDialogue, TextKey: "quests.day5_odds.step0_dialogue"},
			{Type: StepActivity, TextKey: "quests.day5_odds.step1_activity", Extra: map[string]any{"target_stat": "casino_games_played", "target_count": 2}},
			{Type: StepDialogue, TextKey: "quests.day5_odds.step2_transition"},
			// Day 7 — Community contribution
			{Type: StepDialogue, TextKey: "quests.day6_contribution.step0_dialogue"},
			{Type: StepRequirement, TextKey: "quests.day6_contribution.step1_req", Extra: map[string]any{"req_items": map[string]any{"iron_ore": 5, "wheat": 3}}},
			{Type: StepDialogue, TextKey: "quests.day6_contribution.step2_dialogue"},
			// Day 8 — The Undercroft (find the key)
			{Type: StepDialogue, TextKey: "quests.day7_delve.step0_dialogue"},
			{Type: StepActivity, TextKey: "quests.day7_delve.step1_activity", Extra: map[string]any{"target_stat": "delve_completions", "target_count": 1}},
			{Type: StepDialogue, TextKey: "quests.day7_delve.step2_dialogue", Rewards: &QuestReward{Money: 200}},
			// Day 9 — Guardian + finale
			{Type: StepDialogue, TextKey: "quests.day8_sprout.step0_event"},
			{Type: StepBossBattle, TextKey: "quests.day8_sprout.step1_boss", Extra: map[string]any{"boss_stage": 5}},
			{Type: StepDialogue, TextKey: "quests.day8_sprout.step2_transition"},
			{Type: StepDialogue, TextKey: "quests.day8_sprout.step4_dialogue", Rewards: &QuestReward{Money: 1000, ItemIDs: []string{"boss_trophy"}}},
		},
	},
	"daily_quest": {
		ID: "daily_quest", Type: "daily", TitleKey: "quests.daily_challenge.title", DescKey: "quests.daily_challenge.description",
		Steps: []QuestStep{
			{Type: StepActivity, TextKey: "quests.daily_challenge.active_quest"},
		},
	},
	"boss_league": {
		ID: "boss_league", Type: "main", TitleKey: "quests.boss_league.title", DescKey: "quests.boss_league.description",
		Steps: []QuestStep{
			{Type: StepDialogue, TextKey: "quests.boss_league.step0_intro"},
			{Type: StepBossBattle, TextKey: "quests.boss_league.step1_battle", Extra: map[string]any{"boss_stage": 0}, Rewards: &QuestReward{Money: 500, ItemIDs: []string{"spark_shard"}}},
			{Type: StepDialogue, TextKey: "quests.boss_league.step2_victory"},
			{Type: StepBossBattle, TextKey: "quests.boss_league.step3_battle", Extra: map[string]any{"boss_stage": 1}, Rewards: &QuestReward{Money: 1000, ItemIDs: []string{"stone_heart"}}},
			{Type: StepDialogue, TextKey: "quests.boss_league.step4_victory"},
			{Type: StepBossBattle, TextKey: "quests.boss_league.step5_battle", Extra: map[string]any{"boss_stage": 2}, Rewards: &QuestReward{Money: 2000, ItemIDs: []string{"storm_core"}}},
			{Type: StepDialogue, TextKey: "quests.boss_league.step6_victory"},
			{Type: StepBossBattle, TextKey: "quests.boss_league.step7_battle", Extra: map[string]any{"boss_stage": 3}, Rewards: &QuestReward{Money: 3500, ItemIDs: []string{"abyss_pearl"}}},
			{Type: StepDialogue, TextKey: "quests.boss_league.step8_victory"},
			{Type: StepBossBattle, TextKey: "quests.boss_league.step9_battle", Extra: map[string]any{"boss_stage": 4}, Rewards: &QuestReward{Money: 5000, ItemIDs: []string{"phoenix_crest", "boss_trophy"}}},
			{Type: StepDialogue, TextKey: "quests.boss_league.step10_victory"},
		},
	},
	// --- Optional side quests, unlocked by in-game events ---
	// Unlocked when the player loses to the tutorial's final boss (stage 5,
	// The Vault Guardian). Irian mentors the player and trains their pet so
	// the rematch is winnable. Template for future NPC quest lines: add the
	// quest here, link it to an NPC via NPCID, and register its unlock event
	// in BossLossUnlocks (or start it from wherever the event is detected).
	"irian_training": {
		ID: "irian_training", Type: "side", NPCID: "irian",
		TitleKey: "quests.irian_training.title", DescKey: "quests.irian_training.description",
		Steps: []QuestStep{
			{Type: StepDialogue, TextKey: "quests.irian_training.step0_intro", Rewards: &QuestReward{Money: 150}},
			{Type: StepActivity, TextKey: "quests.irian_training.step1_feed", Extra: map[string]any{"target_stat": "pets_fed", "target_count": 3}},
			{Type: StepDialogue, TextKey: "quests.irian_training.step2_feed_dialogue"},
			{Type: StepActivity, TextKey: "quests.irian_training.step3_hunt", Extra: map[string]any{"target_stat": "items_hunted", "target_count": 3}},
			{Type: StepRequirement, TextKey: "quests.irian_training.step4_level_req", Extra: map[string]any{"req_pet_level": 10}},
			{Type: StepDialogue, TextKey: "quests.irian_training.step5_reward", Rewards: &QuestReward{Money: 500, ItemIDs: []string{"warrior_stew", "stonebread", "zephyr_berries"}}},
		},
	},
}

// BossLossUnlock maps a boss stage to the side quest offered when the player
// loses against it. New character quest lines triggered by a boss defeat just
// need one entry here plus a QuestDef with a matching ID.
type BossLossUnlock struct {
	BossStage int
	QuestID   string
}

// BossLossUnlocks lists the quest lines unlocked by losing to a boss.
var BossLossUnlocks = []BossLossUnlock{
	// The Vault Guardian (stage 5) is the tutorial's final boss; losing to it
	// unlocks Irian's training quest line.
	{BossStage: 5, QuestID: "irian_training"},
}

// UnlockOnBossLoss starts the side quest registered for this boss stage if the
// player doesn't already have it active or completed. It returns the quest ID
// and whether it was newly started (false when nothing is registered or the
// player already has the quest).
func (s *Service) UnlockOnBossLoss(userID int64, bossStage int) (string, bool) {
	for _, u := range BossLossUnlocks {
		if u.BossStage != bossStage {
			continue
		}
		if err := s.StartQuest(userID, u.QuestID); err != nil {
			return "", false
		}
		return u.QuestID, true
	}
	return "", false
}

type QuestInfo struct {
	QuestID    string
	Title      string
	Status     string
	StepIndex  int
	Progress   int
	TotalSteps int
	CustomData map[string]any
}

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	svc := &Service{store: s, cfg: cfg}
	s.SetQuestAdvanceFn(svc.RecordActivityComplete)
	return svc
}

// RecordActivityComplete is called by the store when an activity step reaches
// its target. It advances the step with proper custom_data for the next step.
// Returns whether the quest fully completed (COMPLETED) and the i18n text key
// of the next step (empty when completed).
func (s *Service) RecordActivityComplete(userID int64, questID string) (bool, string, error) {
	def := QuestRegistry[questID]
	if def == nil {
		return false, "", nil
	}
	var uqd model.UserQuestData
	if err := s.store.DB.Where("user_id = ? AND quest_id = ?", userID, questID).First(&uqd).Error; err != nil {
		return false, "", err
	}
	if uqd.StepIndex >= len(def.Steps) {
		return false, "", nil
	}
	step := def.Steps[uqd.StepIndex]
	if step.Rewards != nil {
		r := step.Rewards
		if r.Money > 0 {
			if _, err := s.store.UpdateBalance(userID, r.Money); err != nil {
				return false, "", err
			}
		}
		for _, itemID := range r.ItemIDs {
			s.grantRewardItem(userID, itemID)
		}
	}
	nextIdx := uqd.StepIndex + 1
	if nextIdx >= len(def.Steps) {
		if err := s.store.DB.Model(&model.UserQuest{}).
			Where("user_id = ? AND quest_id = ?", userID, questID).
			Updates(map[string]any{"status": "COMPLETED", "completed_at": time.Now()}).Error; err != nil {
			return false, "", err
		}
		return true, "", nil
	}
	updates := map[string]any{"step_index": nextIdx, "progress_value": 0}
	sType := def.Steps[nextIdx].Type
	if sType == StepActivity || sType == StepBossBattle {
		if custom, err := json.Marshal(def.Steps[nextIdx].Extra); err == nil {
			updates["custom_data"] = string(custom)
		}
	} else {
		updates["custom_data"] = "{}"
	}
	if err := s.store.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = ?", userID, questID).
		Updates(updates).Error; err != nil {
		return false, "", err
	}
	return false, def.Steps[nextIdx].TextKey, nil
}

// grantRewardItem hands a quest reward to the player. Equipment pieces are
// turned into real UserEquipment instances (with rolled affixes); everything
// else lands in the regular inventory.
func (s *Service) grantRewardItem(userID int64, itemID string) {
	it := items.Get(itemID)
	if it != nil && it.EquipSlot != "" {
		rar := it.Rarity
		affixes := items.RollAffixes(rar, it.EquipSlot)
		var applied []items.AppliedAffix
		for _, a := range affixes {
			applied = append(applied, items.AppliedAffix{
				ID:    a.ID,
				Name:  a.Name,
				Stat:  a.Stat,
				Value: items.RollAffixValue(a),
			})
		}
		_, err := s.store.CreateEquipmentFromAffixes(userID, it.ID, it.Name, it.Emoji,
			string(rar), it.EquipSlot, it.MinLevel,
			it.StatSTR, it.StatDEX, it.StatINT, it.StatVIT, it.StatLUK,
			applied, it.SetID)
		if err != nil {
			return
		}
		return
	}
	s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
		DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + ?", 1)}),
	}).Create(&model.Inventory{UserID: userID, ItemID: itemID, Quantity: 1})
}

func (s *Service) GetQuestDef(id string) *QuestDef {
	return QuestRegistry[id]
}

// tutorialHuntStepIndex returns the index of the first tutorial step that
// requires a pet (the hunting activity), or -1 if not found.
func tutorialHuntStepIndex(def *QuestDef) int {
	for i, st := range def.Steps {
		if st.TextKey == "quests.day4_will.step1_activity" {
			return i
		}
	}
	return -1
}

// EnsureTutorialEgg repairs players who reached the tutorial hunting/feeding
// steps before the Mystery Egg reward was moved earlier in the quest — leaving
// them stuck without a pet. Grants the egg once if the player is on or past the
// hunting step and has no pet and no egg. Returns true when an egg was granted.
func (s *Service) EnsureTutorialEgg(userID int64) (bool, error) {
	def := QuestRegistry["tutorial"]
	if def == nil {
		return false, nil
	}
	var uqd model.UserQuestData
	err := s.store.DB.Where("user_id = ? AND quest_id = ?", userID, "tutorial").First(&uqd).Error
	if err != nil {
		return false, nil
	}
	if uqd.StepIndex >= len(def.Steps) {
		return false, nil
	}

	// First step that requires a pet (hunting activity).
	huntIdx := tutorialHuntStepIndex(def)
	if huntIdx < 0 || uqd.StepIndex < huntIdx {
		return false, nil
	}

	// Already owns a pet?
	var petCount int64
	if err := s.store.DB.Model(&model.UserPet{}).Where("user_id = ?", userID).Count(&petCount).Error; err == nil && petCount > 0 {
		return false, nil
	}
	// Already has the egg?
	var inv model.Inventory
	err = s.store.DB.Where("user_id = ? AND item_id = ?", userID, "forest_egg").First(&inv).Error
	if err == nil && inv.Quantity > 0 {
		return false, nil
	}

	err = s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
		DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + ?", 1)}),
	}).Create(&model.Inventory{UserID: userID, ItemID: "forest_egg", Quantity: 1}).Error
	if err != nil {
		return false, err
	}
	slog.Info("quests: granted tutorial Mystery Egg to stuck player", "user", userID, "step", uqd.StepIndex)
	return true, nil
}

// HasUnhatchedTutorialEgg reports whether the player is on or past the tutorial
// hunting step, has no pets, and is still holding the unhatched Mystery Egg —
// i.e. they are stuck and need to hatch it to progress.
func (s *Service) HasUnhatchedTutorialEgg(userID int64) bool {
	def := QuestRegistry["tutorial"]
	if def == nil {
		return false
	}
	var uqd model.UserQuestData
	if err := s.store.DB.Where("user_id = ? AND quest_id = ?", userID, "tutorial").First(&uqd).Error; err != nil {
		return false
	}
	huntIdx := tutorialHuntStepIndex(def)
	if huntIdx < 0 || uqd.StepIndex < huntIdx {
		return false
	}
	var petCount int64
	if err := s.store.DB.Model(&model.UserPet{}).Where("user_id = ?", userID).Count(&petCount).Error; err == nil && petCount > 0 {
		return false
	}
	var inv model.Inventory
	err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, "forest_egg").First(&inv).Error
	return err == nil && inv.Quantity > 0
}

func (s *Service) GetAllActiveQuests(userID int64) ([]QuestInfo, error) {
	var uqs []model.UserQuest
	if err := s.store.DB.Where("user_id = ? AND status = 'ACTIVE'", userID).Find(&uqs).Error; err != nil {
		return nil, err
	}
	var out []QuestInfo
	for _, uq := range uqs {
		def := QuestRegistry[uq.QuestID]
		if def == nil {
			continue
		}
		var data model.UserQuestData
		qd := QuestInfo{
			QuestID: uq.QuestID, Title: uq.QuestID, Status: uq.Status,
			TotalSteps: len(def.Steps),
		}
		if err := s.store.DB.Where("user_id = ? AND quest_id = ?", userID, uq.QuestID).First(&data).Error; err == nil {
			qd.StepIndex = data.StepIndex
			qd.Progress = data.ProgressValue
			json.Unmarshal([]byte(data.CustomData), &qd.CustomData)
		}
		out = append(out, qd)
	}
	return out, nil
}

func (s *Service) GetQuestProgress(userID int64, questID string) (*model.UserQuest, *model.UserQuestData, error) {
	var uq model.UserQuest
	if err := s.store.DB.Where("user_id = ? AND quest_id = ?", userID, questID).First(&uq).Error; err != nil {
		return nil, nil, err
	}
	var uqd model.UserQuestData
	if err := s.store.DB.Where("user_id = ? AND quest_id = ?", userID, questID).First(&uqd).Error; err != nil {
		return &uq, nil, nil
	}
	return &uq, &uqd, nil
}

func (s *Service) IsActivityComplete(userID int64, questID string) bool {
	def := QuestRegistry[questID]
	if def == nil {
		return false
	}
	_, uqd, err := s.GetQuestProgress(userID, questID)
	if err != nil || uqd == nil {
		return false
	}
	if uqd.StepIndex >= len(def.Steps) {
		return false
	}
	step := def.Steps[uqd.StepIndex]
	if step.Type != StepActivity {
		return false
	}
	var cd map[string]any
	if err := json.Unmarshal([]byte(uqd.CustomData), &cd); err != nil {
		return false
	}
	targetCount, _ := cd["target_count"].(float64)
	return uqd.ProgressValue >= int(targetCount)
}

func (s *Service) AdvanceStep(userID int64, questID string, choiceID string) error {
	def := QuestRegistry[questID]
	if def == nil {
		return nil
	}
	var uqd model.UserQuestData
	if err := s.store.DB.Where("user_id = ? AND quest_id = ?", userID, questID).First(&uqd).Error; err != nil {
		return err
	}
	nextIdx := uqd.StepIndex + 1
	if uqd.StepIndex < len(def.Steps) {
		step := def.Steps[uqd.StepIndex]
		if step.Rewards != nil {
			r := step.Rewards
			if r.Money > 0 {
				if _, err := s.store.UpdateBalance(userID, r.Money); err != nil {
					return err
				}
			}
			for _, itemID := range r.ItemIDs {
				s.grantRewardItem(userID, itemID)
			}
		}
	}
	if nextIdx >= len(def.Steps) {
		return s.store.DB.Model(&model.UserQuest{}).
			Where("user_id = ? AND quest_id = ?", userID, questID).
			Updates(map[string]any{"status": "COMPLETED", "completed_at": time.Now()}).Error
	}
	updates := map[string]any{"step_index": nextIdx, "progress_value": 0}
	if nextIdx < len(def.Steps) {
		sType := def.Steps[nextIdx].Type
		if sType == StepActivity || sType == StepBossBattle {
			if custom, err := json.Marshal(def.Steps[nextIdx].Extra); err == nil {
				updates["custom_data"] = string(custom)
			}
		}
	}
	slog.Info("AdvanceStep",
		"user_id", userID,
		"quest_id", questID,
		"nextIdx", nextIdx,
		"nextStepType", def.Steps[nextIdx].Type,
		"nextStepExtra", def.Steps[nextIdx].Extra,
		"updates", updates,
	)
	return s.store.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = ?", userID, questID).
		Updates(updates).Error
}

func (s *Service) RecordBossVictory(userID int64, bossStage int) error {
	var uqs []model.UserQuest
	if err := s.store.DB.Where("user_id = ? AND status = 'ACTIVE'", userID).Find(&uqs).Error; err != nil {
		return err
	}
	for _, uq := range uqs {
		def := QuestRegistry[uq.QuestID]
		if def == nil {
			continue
		}
		var uqd model.UserQuestData
		if err := s.store.DB.Where("user_id = ? AND quest_id = ?", userID, uq.QuestID).First(&uqd).Error; err != nil {
			continue
		}
		if uqd.StepIndex >= len(def.Steps) {
			continue
		}
		step := def.Steps[uqd.StepIndex]
		if step.Type != StepBossBattle {
			continue
		}
		stage, ok := step.Extra["boss_stage"].(int)
		if !ok || stage != bossStage {
			continue
		}
		slog.Info("RecordBossVictory: advancing quest",
			"user_id", userID, "quest_id", uq.QuestID,
			"boss_stage", bossStage, "step", uqd.StepIndex)
		return s.AdvanceStep(userID, uq.QuestID, "")
	}
	return nil
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

// RequirementChecker validates one requirement declared in a step's Extra map
// (keyed by the extra key, e.g. "req_pet_level") and records what is missing
// into reqErr. New requirement types = add one checker here plus a line in
// the display code (see internal/cogs/quests/quests.go).
type RequirementChecker func(s *Service, userID int64, extra map[string]any, reqErr *RequirementError)

var requirementCheckers = map[string]RequirementChecker{
	"req_money":      checkMoneyRequirement,
	"req_items":      checkItemsRequirement,
	"req_owns_house": checkHouseRequirement,
	"req_pet_level":  checkPetLevelRequirement,
}

func checkMoneyRequirement(s *Service, userID int64, extra map[string]any, reqErr *RequirementError) {
	money := toInt(extra["req_money"])
	if money <= 0 {
		return
	}
	var user model.User
	if err := s.store.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
		return
	}
	if user.Balance < money {
		reqErr.MoneyNeeded = money
		reqErr.MoneyHave = user.Balance
	}
}

func checkItemsRequirement(s *Service, userID int64, extra map[string]any, reqErr *RequirementError) {
	items, ok := extra["req_items"].(map[string]any)
	if !ok {
		return
	}
	for itemID, qtyAny := range items {
		qty := toInt(qtyAny)
		if qty <= 0 {
			continue
		}
		var inv model.Inventory
		err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, itemID).First(&inv).Error
		have := 0
		if err == nil {
			have = inv.Quantity
		}
		if have < qty {
			reqErr.MissingItems = append(reqErr.MissingItems, MissingItem{
				ItemID: itemID,
				Needed: qty,
				Have:   have,
			})
		}
	}
}

func checkHouseRequirement(s *Service, userID int64, extra map[string]any, reqErr *RequirementError) {
	ownsHouse, ok := extra["req_owns_house"].(bool)
	if !ok || !ownsHouse {
		return
	}
	var housing model.UserHousing
	if err := s.store.DB.Where("user_id = ?", userID).First(&housing).Error; err != nil {
		reqErr.NeedsHouse = true
	}
}

// checkPetLevelRequirement verifies the level of the player's active pet
// against the "req_pet_level" extra value. The level is not consumed by
// FulfillRequirement — it acts as a training milestone.
func checkPetLevelRequirement(s *Service, userID int64, extra map[string]any, reqErr *RequirementError) {
	needed := toInt(extra["req_pet_level"])
	if needed <= 0 {
		return
	}
	var pet model.UserPet
	if err := s.store.DB.Where("user_id = ? AND is_active = ?", userID, true).First(&pet).Error; err != nil {
		reqErr.PetLevelNeeded = needed
		return
	}
	if pet.Level < needed {
		reqErr.PetLevelNeeded = needed
		reqErr.PetLevelHave = pet.Level
	}
}

func (s *Service) CheckRequirement(userID int64, questID string) error {
	def := QuestRegistry[questID]
	if def == nil {
		return errors.New("quest not found")
	}
	var uqd model.UserQuestData
	if err := s.store.DB.Where("user_id = ? AND quest_id = ?", userID, questID).First(&uqd).Error; err != nil {
		return err
	}
	if uqd.StepIndex >= len(def.Steps) {
		return errors.New("quest already completed")
	}
	step := def.Steps[uqd.StepIndex]
	if step.Type != StepRequirement {
		return errors.New("current step is not a requirement")
	}

	extra := step.Extra
	if extra == nil {
		return errors.New("no requirement data")
	}

	reqErr := &RequirementError{}
	for key, checker := range requirementCheckers {
		if _, ok := extra[key]; ok {
			checker(s, userID, extra, reqErr)
		}
	}

	if reqErr.MoneyNeeded > 0 || len(reqErr.MissingItems) > 0 || reqErr.NeedsHouse || reqErr.PetLevelNeeded > 0 {
		return reqErr
	}

	return nil
}

func (s *Service) FulfillRequirement(userID int64, questID string) error {
	def := QuestRegistry[questID]
	if def == nil {
		return errors.New("quest not found")
	}
	var uqd model.UserQuestData
	if err := s.store.DB.Where("user_id = ? AND quest_id = ?", userID, questID).First(&uqd).Error; err != nil {
		return err
	}
	if uqd.StepIndex >= len(def.Steps) {
		return errors.New("quest already completed")
	}
	step := def.Steps[uqd.StepIndex]
	if step.Type != StepRequirement {
		return errors.New("current step is not a requirement")
	}

	if err := s.CheckRequirement(userID, questID); err != nil {
		return err
	}

	extra := step.Extra

	// Deduct money
	if money := toInt(extra["req_money"]); money > 0 {
		if err := s.store.DB.Model(&model.User{}).
			Where("user_id = ?", userID).
			UpdateColumn("balance", gorm.Expr("balance - ?", money)).Error; err != nil {
			return err
		}
	}

	// Deduct items
	if items, ok := extra["req_items"].(map[string]any); ok {
		for itemID, qtyAny := range items {
			qty := toInt(qtyAny)
			if qty <= 0 {
				continue
			}
			if err := s.store.DB.Model(&model.Inventory{}).
				Where("user_id = ? AND item_id = ?", userID, itemID).
				UpdateColumn("quantity", gorm.Expr("quantity - ?", qty)).Error; err != nil {
				return err
			}
		}
	}

	return s.AdvanceStep(userID, questID, "")
}

// StartQuest begins a quest for the user if not already active or completed.
func (s *Service) StartQuest(userID int64, questID string) error {
	def := QuestRegistry[questID]
	if def == nil {
		return errors.New("quest not found")
	}
	var existing model.UserQuest
	err := s.store.DB.Where("user_id = ? AND quest_id = ?", userID, questID).First(&existing).Error
	if err == nil {
		if existing.Status == "ACTIVE" {
			return errors.New("quest already active")
		}
		return errors.New("quest already completed")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return s.store.CreateQuest(userID, questID)
}

// QuestCompletedMsg returns a localized string to notify the user that a quest
// was completed and they can use /quest to check it. Returns empty if questID
// is empty or the quest definition is not found.
func QuestCompletedMsg(questID string, lang string) string {
	if questID == "" {
		return ""
	}
	def := QuestRegistry[questID]
	if def == nil {
		return ""
	}
	title := i18n.T(def.TitleKey, lang)
	return i18n.T("quests.completed_activity_msg", lang, map[string]any{"title": title})
}

// QuestNotificationMsg returns a localized string to notify the user about a
// quest event surfaced by RecordActivity: either a full quest completion or a
// step advancement (pointing at the next objective).
func QuestNotificationMsg(n store.QuestNotification, lang string) string {
	if n.QuestID == "" {
		return ""
	}
	if n.Completed {
		return QuestCompletedMsg(n.QuestID, lang)
	}
	def := QuestRegistry[n.QuestID]
	if def == nil {
		return ""
	}
	title := i18n.T(def.TitleKey, lang)
	next := ""
	if n.NextStepKey != "" {
		next = i18n.T(n.NextStepKey, lang)
	}
	return i18n.T("quests.step_advanced", lang, map[string]any{"title": title, "next": next})
}
