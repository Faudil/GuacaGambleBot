package veil

import (
	"math/rand"
	"sort"
	"strings"

	"guacagamblebot/internal/i18n"
)

type AttackGambleState struct {
	Multiplier int
	Failed     bool
	Locked     bool
}

func NewAttackGamble() *AttackGambleState {
	return &AttackGambleState{Multiplier: 1}
}

func PressMorePower(state *AttackGambleState) (mult int, failed bool) {
	state.Multiplier++
	odds := []int{0, 10, 25, 45, 70, 90, 100}
	chance := 100
	if state.Multiplier < len(odds) {
		chance = odds[state.Multiplier]
	}
	if rand.Intn(100) < chance {
		state.Failed = true
		return state.Multiplier, true
	}
	return state.Multiplier, false
}

func LockDamage(state *AttackGambleState) int {
	state.Locked = true
	if state.Failed {
		return 0
	}
	return state.Multiplier
}

func GambleOdds(mult int) int {
	odds := []int{0, 10, 25, 45, 70, 90, 100}
	if mult < len(odds) {
		return odds[mult]
	}
	return 100
}

type DiceState struct {
	Dice    [5]int
	Rerolls int
	Target  int64
	Picked  [5]bool
}

func NewDiceState() *DiceState {
	return &DiceState{Rerolls: 2}
}

func RollDice(state *DiceState) {
	for i := range state.Dice {
		state.Dice[i] = rand.Intn(6) + 1
	}
}

func RerollDie(state *DiceState, index int) {
	if state.Rerolls <= 0 || index < 0 || index >= 5 {
		return
	}
	state.Dice[index] = rand.Intn(6) + 1
	state.Rerolls--
}

func EvaluateDiceHand(dice [5]int, lang string) (handName string, healPercent int) {
	counts := map[int]int{}
	for _, d := range dice {
		counts[d]++
	}

	pairs := 0
	triples := 0
	quads := 0
	for _, c := range counts {
		switch c {
		case 2:
			pairs++
		case 3:
			triples++
		case 4:
			quads++
		case 5:
			return i18n.T("veil.minigame.hand_five_kind", lang), 100
		}
	}

	if quads > 0 {
		return i18n.T("veil.minigame.hand_four_kind", lang), 80
	}
	if triples > 0 && pairs > 0 {
		return i18n.T("veil.minigame.hand_full_house", lang), 60
	}

	sorted := make([]int, 5)
	copy(sorted, dice[:])
	sort.Ints(sorted)
	isStraight := true
	for i := 1; i < 5; i++ {
		if sorted[i] != sorted[i-1]+1 {
			isStraight = false
			break
		}
	}
	if isStraight {
		return i18n.T("veil.minigame.hand_straight", lang), 50
	}
	if triples > 0 {
		return i18n.T("veil.minigame.hand_three_kind", lang), 40
	}
	if pairs == 2 {
		return i18n.T("veil.minigame.hand_two_pair", lang), 30
	}
	if pairs == 1 {
		return i18n.T("veil.minigame.hand_one_pair", lang), 20
	}
	return i18n.T("veil.minigame.hand_high_card", lang), 10
}

func DiceHandDescription(state *DiceState, lang string) string {
	handName, healPercent := EvaluateDiceHand(state.Dice, lang)
	sb := &strings.Builder{}
	for _, d := range state.Dice {
		emoji := map[int]string{1: "1️⃣", 2: "2️⃣", 3: "3️⃣", 4: "4️⃣", 5: "5️⃣", 6: "6️⃣"}
		sb.WriteString(emoji[d] + " ")
	}
	sb.WriteString(i18n.T("veil.minigame.dice_hand", lang, map[string]any{"hand": handName, "pct": healPercent}))
	if state.Rerolls > 0 {
		sb.WriteString(i18n.T("veil.minigame.dice_rerolls", lang, map[string]any{"count": state.Rerolls}))
	}
	return sb.String()
}

type ShieldState struct {
	Intensity int
	Confirmed bool
}

func NewShieldState() *ShieldState {
	return &ShieldState{Intensity: 5}
}

func AdjustShieldIntensity(state *ShieldState, delta int) int {
	state.Intensity += delta
	if state.Intensity < 1 {
		state.Intensity = 1
	}
	if state.Intensity > 10 {
		state.Intensity = 10
	}
	return state.Intensity
}

func ShieldRisk(intensity int) int {
	if intensity <= 5 {
		return 0
	}
	return (intensity - 5) * 20
}

func ShieldValue(intensity int) int {
	return intensity * 35
}

func ConfirmShield(state *ShieldState) (shieldVal int, backlash int) {
	state.Confirmed = true
	val := ShieldValue(state.Intensity)
	risk := ShieldRisk(state.Intensity)
	if rand.Intn(100) < risk {
		backlash = val * 30 / 100
		return val, backlash
	}
	return val, 0
}

func ShieldDescription(state *ShieldState, lang string) string {
	return i18n.T("veil.minigame.shield_desc", lang, map[string]any{
		"intensity": state.Intensity,
		"risk":      ShieldRisk(state.Intensity),
		"val":       ShieldValue(state.Intensity),
	})
}


