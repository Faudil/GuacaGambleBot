package pets

import (
	"fmt"
	"math/rand"

	"guacagamblebot/internal/model"
)

type InteractionChoice struct {
	ID         string
	Emoji      string
	BondReward int
	XPReward   int
	ItemReward string
}

type InteractionDef struct {
	ID          string
	Triggers    []string
	Chance      float64
	CooldownM   int
	Personality string // "" means any
	Choices     []InteractionChoice
}

type InteractionResult struct {
	ID      string
	Choices []InteractionChoice
}

type ChoiceReward struct {
	BondReward int
	XPReward   int
	ItemReward string
}

var interactionPool = []InteractionDef{
	{
		ID: "play_time", Triggers: []string{"feed", "idle"}, Chance: 0.15, CooldownM: 180,
		Choices: []InteractionChoice{
			{ID: "fetch", Emoji: "🎾", BondReward: 3, XPReward: 10},
			{ID: "tug", Emoji: "🪢", BondReward: 4, XPReward: 15, ItemReward: "pebble"},
			{ID: "ignore", Emoji: "🤲", BondReward: 2},
		},
	},
	{
		ID: "snack_time", Triggers: []string{"feed", "hunt"}, Chance: 0.12, CooldownM: 180,
		Choices: []InteractionChoice{
			{ID: "feed_treat", Emoji: "🍖", BondReward: 3, XPReward: 5},
			{ID: "share_meal", Emoji: "🍲", BondReward: 5, XPReward: 10},
			{ID: "cook", Emoji: "🍳", BondReward: 4, XPReward: 20, ItemReward: "tomato"},
		},
	},
	{
		ID: "explore_together", Triggers: []string{"expedition", "idle"}, Chance: 0.12, CooldownM: 240,
		Choices: []InteractionChoice{
			{ID: "explore", Emoji: "🧭", BondReward: 4, XPReward: 25},
			{ID: "follow", Emoji: "🐾", BondReward: 3, XPReward: 15, ItemReward: "coal"},
			{ID: "rest", Emoji: "😴", BondReward: 2},
		},
	},
	{
		ID: "grooming", Triggers: []string{"feed", "idle"}, Chance: 0.10, CooldownM: 240,
		Choices: []InteractionChoice{
			{ID: "brush", Emoji: "🪥", BondReward: 4, XPReward: 5},
			{ID: "bath", Emoji: "🛁", BondReward: 5, XPReward: 10, ItemReward: "sardine"},
			{ID: "massage", Emoji: "💆", BondReward: 3},
		},
	},
	{
		ID: "training", Triggers: []string{"battle", "hunt"}, Chance: 0.10, CooldownM: 240,
		Choices: []InteractionChoice{
			{ID: "spar", Emoji: "⚔️", BondReward: 4, XPReward: 30},
			{ID: "teach", Emoji: "📖", BondReward: 3, XPReward: 20},
			{ID: "praise", Emoji: "👏", BondReward: 5},
		},
	},
	{
		ID: "rescue", Triggers: []string{"battle", "hunt"}, Chance: 0.05, CooldownM: 480,
		Choices: []InteractionChoice{
			{ID: "stand_together", Emoji: "🤝", BondReward: 8, XPReward: 40, ItemReward: "rough_diamond"},
			{ID: "investigate", Emoji: "🔦", BondReward: 6, XPReward: 30},
			{ID: "retreat", Emoji: "🏃", BondReward: 4, XPReward: 10},
		},
	},
}

// InteractionIntroKey returns the i18n key for an interaction's intro text.
// If a personality-specific variant exists, it returns that; otherwise the generic one.
func (ir *InteractionResult) IntroKey(personality string) string {
	specificKey := fmt.Sprintf("pets.interact.%s.intro.%s", ir.ID, personality)
	// The i18n system handles fallback — if the specific key doesn't exist,
	// it will fall back to the generic key. We just request the specific key.
	return specificKey
}

// GenericIntroKey returns the base (non-personality) i18n key.
func (ir *InteractionResult) GenericIntroKey() string {
	return fmt.Sprintf("pets.interact.%s.intro", ir.ID)
}

// ChoiceLabel returns the i18n key for a choice label.
func (ic *InteractionChoice) ChoiceLabelKey() string {
	return fmt.Sprintf("pets.interact.%s.choices.%s", findInteractionID(ic.ID), ic.ID)
}

// ChoiceDetailKey returns the i18n key for a choice detail.
func (ic *InteractionChoice) ChoiceDetailKey() string {
	return fmt.Sprintf("pets.interact.%s.choices.%s.detail", findInteractionID(ic.ID), ic.ID)
}

func findInteractionID(choiceID string) string {
	for _, in := range interactionPool {
		for _, c := range in.Choices {
			if c.ID == choiceID {
				return in.ID
			}
		}
	}
	return ""
}

// MaybeTriggerInteraction checks probability and returns an interaction if triggered.
func MaybeTriggerInteraction(pet *model.UserPet, trigger string) *InteractionResult {
	var candidates []InteractionDef
	for _, in := range interactionPool {
		for _, t := range in.Triggers {
			if t == trigger && (in.Personality == "" || in.Personality == pet.Personality) {
				candidates = append(candidates, in)
				break
			}
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	chosen := candidates[rand.Intn(len(candidates))]
	if rand.Float64() > chosen.Chance {
		return nil
	}
	return &InteractionResult{
		ID:      chosen.ID,
		Choices: chosen.Choices,
	}
}

// ResolveInteraction returns the rewards for a given choice.
func ResolveInteraction(choiceID string) *ChoiceReward {
	for _, in := range interactionPool {
		for _, c := range in.Choices {
			if c.ID == choiceID {
				return &ChoiceReward{
					BondReward: c.BondReward,
					XPReward:   c.XPReward,
					ItemReward: c.ItemReward,
				}
			}
		}
	}
	return nil
}
