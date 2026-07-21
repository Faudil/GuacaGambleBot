package quests

import (
	"time"

	"guacagamblebot/internal/config"
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

type QuestReward struct {
	Money         int
	XP            int
	Crowns        int
	ItemIDs       []string
	AchievementID string
}

type QuestStep struct {
	Type     QuestStepType
	TextKey  string
	Rewards  *QuestReward
	Extra    map[string]any
}

type QuestDef struct {
	ID          string
	Type        string
	NPCID       string
	TitleKey    string
	DescKey     string
	Steps       []QuestStep
	RepReq      int
	Unlocks     []string
}

var QuestRegistry = map[string]*QuestDef{
	"tutorial": {
		ID: "tutorial", Type: "main", TitleKey: "quests.day0_welcome.title", DescKey: "quests.day0_welcome.description",
		Steps: []QuestStep{
			{Type: StepDialogue, TextKey: "quests.day0_welcome.step0_dialogue", Rewards: &QuestReward{Money: 100}},
			{Type: StepActivity, TextKey: "quests.day1_strata.step1_activity", Extra: map[string]any{"target_stat": "items_mined", "target_count": 10}},
			{Type: StepDialogue, TextKey: "quests.day1_strata.step2_dialogue", Rewards: &QuestReward{Money: 200}},
			{Type: StepActivity, TextKey: "quests.day2_alchemy.step1_activity", Extra: map[string]any{"target_stat": "items_farmed", "target_count": 10}},
			{Type: StepDialogue, TextKey: "quests.day2_alchemy.step2_choice", Rewards: &QuestReward{ItemIDs: []string{"blé"}}},
		},
	},
	"daily_quest": {
		ID: "daily_quest", Type: "daily", TitleKey: "quests.daily_challenge.title", DescKey: "quests.daily_challenge.description",
		Steps: []QuestStep{
			{Type: StepActivity, TextKey: "quests.daily_challenge.active_quest"},
		},
	},
}

type QuestInfo struct {
	QuestID     string
	Title       string
	Status      string
	StepIndex   int
	Progress    int
	TotalSteps  int
	CustomData  map[string]any
}

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) GetQuestDef(id string) *QuestDef {
	return QuestRegistry[id]
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
		}
	}
	if nextIdx >= len(def.Steps) {
		return s.store.DB.Model(&model.UserQuest{}).
			Where("user_id = ? AND quest_id = ?", userID, questID).
			Updates(map[string]any{"status": "COMPLETED", "completed_at": time.Now()}).Error
	}
	return s.store.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = ?", userID, questID).
		Updates(map[string]any{"step_index": nextIdx, "progress_value": 0}).Error
}
