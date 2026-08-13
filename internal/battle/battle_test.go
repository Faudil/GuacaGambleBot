package battle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimulate(t *testing.T) {
	p1 := &BattlePet{
		ID: 1, Nickname: "Alpha", Emoji: "🐉",
		Level: 10, HP: 100, MaxHP: 100,
		Atk: 30, Defense: 15, Speed: 20,
		DGE: 10, ACC: 20, CritC: 15, CritD: 1.5, SpcC: 5,
	}
	p2 := &BattlePet{
		ID: 2, Nickname: "Beta", Emoji: "🐲",
		Level: 10, HP: 100, MaxHP: 100,
		Atk: 25, Defense: 10, Speed: 25,
		DGE: 15, ACC: 20, CritC: 10, CritD: 1.5, SpcC: 5,
	}

	result := Simulate(p1, p2)
	assert.NotNil(t, result)
	assert.Len(t, result.Log, min(10, len(result.Log)))
	assert.True(t, result.WinnerID == 1 || result.WinnerID == 2)
	assert.NotEmpty(t, result.Turns, "every battle must record at least one turn")
	last := result.Turns[len(result.Turns)-1]
	if result.WinnerID == 1 {
		assert.Greater(t, last.Pet1HP, 0)
		assert.LessOrEqual(t, last.Pet2HP, 0)
	} else {
		assert.LessOrEqual(t, last.Pet1HP, 0)
		assert.Greater(t, last.Pet2HP, 0)
	}
}

func TestSimulateKO(t *testing.T) {
	p1 := &BattlePet{
		ID: 1, Nickname: "Strong", Emoji: "💪",
		Level: 20, HP: 500, MaxHP: 500,
		Atk: 100, Defense: 50, Speed: 50,
		DGE: 10, ACC: 30, CritC: 20, CritD: 2.0, SpcC: 10,
	}
	p2 := &BattlePet{
		ID: 2, Nickname: "Weak", Emoji: "🐁",
		Level: 1, HP: 20, MaxHP: 20,
		Atk: 5, Defense: 2, Speed: 5,
		DGE: 5, ACC: 5, CritC: 5, CritD: 1.2, SpcC: 0,
	}

	result := Simulate(p1, p2)
	assert.Equal(t, int64(1), result.WinnerID)
	assert.True(t, result.Pet1HP > 0)
	assert.True(t, result.Pet2HP <= 0)
}

func TestHealFull(t *testing.T) {
	p := &BattlePet{HP: 50, MaxHP: 100, Defense: 20}
	p.defenseMalus = 10
	p.stunnedTurns = 2
	p.healFull()
	assert.Equal(t, 100, p.HP)
	assert.Equal(t, 0, p.defenseMalus)
	assert.Equal(t, 0, p.stunnedTurns)
}

func TestSimulatePreserveHP(t *testing.T) {
	p1 := &BattlePet{
		ID: 1, Nickname: "Hurt", Emoji: "🐉", PetType: "Dragon",
		Level: 10, HP: 30, MaxHP: 100,
		Atk: 30, Defense: 15, Speed: 20,
		DGE: 10, ACC: 20, CritC: 15, CritD: 1.5, SpcC: 10,
	}
	p2 := &BattlePet{
		ID: 2, Nickname: "Fresh", Emoji: "🐲", PetType: "Phoenix",
		Level: 10, HP: 100, MaxHP: 100,
		Atk: 25, Defense: 10, Speed: 25,
		DGE: 15, ACC: 20, CritC: 10, CritD: 1.5, SpcC: 10,
	}

	SimulatePreserveHP(p1, p2)
	assert.GreaterOrEqual(t, p1.HP, 0)
	assert.LessOrEqual(t, p1.HP, 30, "injured pet must not be healed to full by SimulatePreserveHP")
	assert.LessOrEqual(t, p2.HP, 100)
}
