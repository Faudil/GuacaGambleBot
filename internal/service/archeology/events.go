package archeology

import (
	"math/rand"
)

func (s *Service) RollEvent(state *GameState) *DigEvent {
	r := rand.Float64()

	if r < 0.01 && state.Depth <= 0 {
		return s.rollFossilEgg()
	}

	if r < 0.06 {
		return s.rollBuriedTreasure()
	}

	if r < 0.14 && state.Integrity > 30 {
		return s.rollGuardian()
	}

	if r < 0.22 && state.Depth < state.MaxDepth/2 {
		return s.rollCaveIn()
	}

	if r < 0.32 {
		return s.rollFossilWhisper(state)
	}

	return nil
}

func (s *Service) rollFossilWhisper(state *GameState) *DigEvent {
	return &DigEvent{
		Type:    EventFossilWhisper,
		TitleID: "arch.event_whisper_title",
		DescID:  "arch.event_whisper_desc",
		Data:    map[string]any{"layer": GetLayerNameID(state.CurrentLayer), "tool": bestToolForLayer(state.CurrentLayer)},
		Choices: []EventChoice{
			{LabelID: "arch.event_whisper_accept", Value: "accept", Style: 3},
		},
	}
}

func bestToolForLayer(layer LayerType) string {
	switch layer {
	case LayerSoftSoil:
		return "brush"
	case LayerHardRock:
		return "hammer"
	case LayerGravel:
		return "hammer"
	case LayerClay:
		return "brush"
	case LayerBedrock:
		return "dynamite"
	}
	return "hammer"
}

func (s *Service) rollCaveIn() *DigEvent {
	return &DigEvent{
		Type:    EventCaveIn,
		TitleID: "arch.event_cavein_title",
		DescID:  "arch.event_cavein_desc",
		Choices: []EventChoice{
			{LabelID: "arch.event_cavein_careful", Value: "careful", Style: 3},
			{LabelID: "arch.event_cavein_rush", Value: "rush", Style: 1},
			{LabelID: "arch.event_cavein_abandon", Value: "abandon", Style: 2},
		},
	}
}

func (s *Service) rollGuardian() *DigEvent {
	return &DigEvent{
		Type:    EventGuardian,
		TitleID: "arch.event_guardian_title",
		DescID:  "arch.event_guardian_desc",
		Choices: []EventChoice{
			{LabelID: "arch.event_guardian_tribute", Value: "tribute", Style: 3},
			{LabelID: "arch.event_guardian_press", Value: "press", Style: 1},
			{LabelID: "arch.event_guardian_retreat", Value: "retreat", Style: 2},
		},
	}
}

func (s *Service) rollBuriedTreasure() *DigEvent {
	coins := 50 + rand.Intn(151)
	return &DigEvent{
		Type:    EventBuriedTreasure,
		TitleID: "arch.event_treasure_title",
		DescID:  "arch.event_treasure_desc",
		Data:    map[string]any{"coins": coins},
		Choices: []EventChoice{
			{LabelID: "arch.event_treasure_dig", Value: "dig", Style: 3},
			{LabelID: "arch.event_treasure_ignore", Value: "ignore", Style: 2},
		},
	}
}

func (s *Service) rollFossilEgg() *DigEvent {
	return &DigEvent{
		Type:    EventFossilEgg,
		TitleID: "arch.event_egg_title",
		DescID:  "arch.event_egg_desc",
		Choices: []EventChoice{
			{LabelID: "arch.event_egg_take", Value: "take", Style: 3},
		},
	}
}

func (s *Service) ResolveEvent(state *GameState, evt *DigEvent, choice string) *EventResult {
	switch evt.Type {
	case EventFossilWhisper:
		state.RevealedLayer = true
		return &EventResult{
			TitleID:       "arch.event_whisper_result_title",
			DescID:        "arch.event_whisper_result_desc",
			RevealedTool:  bestToolForLayer(state.CurrentLayer),
			RevealedLayer: state.CurrentLayer,
			BackToDig:     true,
		}
	case EventCaveIn:
		return s.resolveCaveIn(state, choice)
	case EventGuardian:
		return s.resolveGuardian(state, choice)
	case EventBuriedTreasure:
		return s.resolveTreasure(state, choice)
	case EventFossilEgg:
		return &EventResult{
			TitleID:   "arch.event_egg_result_title",
			DescID:    "arch.event_egg_result_desc",
			ItemGiven: "fossilized_egg",
			ItemQty:   1,
			BackToDig: true,
		}
	}
	return &EventResult{TitleID: "arch.event_default_title", DescID: "arch.event_default_desc", BackToDig: true}
}

func (s *Service) resolveCaveIn(state *GameState, choice string) *EventResult {
	switch choice {
	case "careful":
		state.Actions -= 2
		if state.Actions <= 0 {
			state.Finished = true
		}
		return &EventResult{
			TitleID:     "arch.event_cavein_careful_title",
			DescID:      "arch.event_cavein_careful_desc",
			ActionsLost: 2,
			BackToDig:   true,
		}
	case "rush":
		state.Integrity -= 20
		if state.Integrity <= 0 {
			state.Finished = true
		}
		freeDepth := state.MaxDepth / 4
		state.Depth -= freeDepth
		if state.Depth <= 0 {
			state.Finished = true
		}
		return &EventResult{
			TitleID:   "arch.event_cavein_rush_title",
			DescID:    "arch.event_cavein_rush_desc",
			IntLoss:   20,
			DepthGain: freeDepth,
			BackToDig: true,
		}
	case "abandon":
		state.Actions = 0
		state.Finished = true
		return &EventResult{
			TitleID:   "arch.event_cavein_abandon_title",
			DescID:    "arch.event_cavein_abandon_desc",
			BackToDig: false,
		}
	}
	return nil
}

func (s *Service) resolveGuardian(state *GameState, choice string) *EventResult {
	switch choice {
	case "tribute":
		state.Actions++
		return &EventResult{
			TitleID:    "arch.event_guardian_tribute_title",
			DescID:     "arch.event_guardian_tribute_desc",
			CoinChange: -100,
			BackToDig:  true,
		}
	case "press":
		state.Integrity -= 25
		if state.Integrity <= 0 {
			state.Finished = true
		}
		return &EventResult{
			TitleID:   "arch.event_guardian_press_title",
			DescID:    "arch.event_guardian_press_desc",
			IntLoss:   25,
			BackToDig: true,
		}
	case "retreat":
		return &EventResult{
			TitleID:   "arch.event_guardian_retreat_title",
			DescID:    "arch.event_guardian_retreat_desc",
			BackToDig: true,
		}
	}
	return nil
}

func (s *Service) resolveTreasure(state *GameState, choice string) *EventResult {
	switch choice {
	case "dig":
		state.Actions--
		if state.Actions <= 0 {
			state.Finished = true
		}
		coins, _ := evtDataCoins(state)
		return &EventResult{
			TitleID:     "arch.event_treasure_dig_title",
			DescID:      "arch.event_treasure_dig_desc",
			CoinChange:  coins,
			ActionsLost: 1,
			BackToDig:   true,
		}
	case "ignore":
		return &EventResult{
			TitleID:   "arch.event_treasure_ignore_title",
			DescID:    "arch.event_treasure_ignore_desc",
			BackToDig: true,
		}
	}
	return nil
}

func evtDataCoins(state *GameState) (int, bool) {
	return 50 + rand.Intn(151), true
}
