package quests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	questssvc "guacagamblebot/internal/service/quests"
)

func TestStepCommandHintActivity(t *testing.T) {
	step := questssvc.QuestStep{Type: questssvc.StepActivity, Extra: map[string]any{"target_stat": "items_mined"}}
	assert.Equal(t, "/mine", stepCommandHint(step))
}

func TestStepCommandHintActivityUnknownStat(t *testing.T) {
	step := questssvc.QuestStep{Type: questssvc.StepActivity, Extra: map[string]any{"target_stat": "mystery_stat"}}
	assert.Equal(t, "", stepCommandHint(step))
}

func TestStepCommandHintRequirementHouse(t *testing.T) {
	step := questssvc.QuestStep{Type: questssvc.StepRequirement, Extra: map[string]any{"req_owns_house": true}}
	assert.Equal(t, "/house", stepCommandHint(step))
}

func TestStepCommandHintRequirementPetLevel(t *testing.T) {
	step := questssvc.QuestStep{Type: questssvc.StepRequirement, Extra: map[string]any{"req_pet_level": 10}}
	assert.Equal(t, "/pets /hunt", stepCommandHint(step))
}

func TestStepCommandHintRequirementItems(t *testing.T) {
	step := questssvc.QuestStep{Type: questssvc.StepRequirement, Extra: map[string]any{
		"req_items": map[string]any{"iron_ore": 5, "wheat": 3},
	}}
	assert.Equal(t, "/mine /farm", stepCommandHint(step))
}

func TestStepCommandHintRequirementCombined(t *testing.T) {
	step := questssvc.QuestStep{Type: questssvc.StepRequirement, Extra: map[string]any{
		"req_owns_house": true,
		"req_items":      map[string]any{"iron_ore": 2, "wheat": 3},
	}}
	assert.Equal(t, "/house /mine /farm", stepCommandHint(step))
}

func TestStepCommandHintRequirementUnknownItem(t *testing.T) {
	step := questssvc.QuestStep{Type: questssvc.StepRequirement, Extra: map[string]any{
		"req_items": map[string]any{"mystery_item": 1},
	}}
	assert.Equal(t, "", stepCommandHint(step))
}

func TestStepCommandHintRequirementNoExtra(t *testing.T) {
	step := questssvc.QuestStep{Type: questssvc.StepRequirement}
	assert.Equal(t, "", stepCommandHint(step))
}

func TestStepCommandHintBoss(t *testing.T) {
	step := questssvc.QuestStep{Type: questssvc.StepBossBattle, Extra: map[string]any{"boss_stage": 5}}
	assert.Equal(t, "/boss", stepCommandHint(step))
}

func TestStepCommandHintDialogue(t *testing.T) {
	assert.Equal(t, "", stepCommandHint(questssvc.QuestStep{Type: questssvc.StepDialogue}))
	assert.Equal(t, "", stepCommandHint(questssvc.QuestStep{Type: questssvc.StepChoice}))
}

func TestRequirementCommandsDeduplicates(t *testing.T) {
	step := questssvc.QuestStep{Type: questssvc.StepRequirement, Extra: map[string]any{
		"req_items": map[string]any{"iron_ore": 5, "coal": 3},
	}}
	assert.Equal(t, "/mine", requirementCommands(step))
}
