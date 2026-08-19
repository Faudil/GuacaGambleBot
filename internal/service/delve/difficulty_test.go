package delve

import (
	"math"
	"math/rand"
	"testing"
)

func TestRecommendedLevel(t *testing.T) {
	tests := []struct {
		floor int
		want  int
	}{
		{1, 1}, {2, 3}, {3, 5}, {4, 7}, {5, 9},
		{6, 11}, {7, 13}, {8, 15}, {9, 17}, {10, 19},
		{15, 29}, {26, 51}, {50, 99},
	}
	for _, tt := range tests {
		got := RecommendedLevel(tt.floor)
		if got != tt.want {
			t.Errorf("RecommendedLevel(%d) = %d; want %d", tt.floor, got, tt.want)
		}
	}
}

func TestLevelScalingMul(t *testing.T) {
	mul := LevelScalingMul(5, 1)
	expected := 1.0 + float64(UnderLeveledMaxDiff)*UnderLeveledMult
	if math.Abs(mul-expected) > 0.001 {
		t.Errorf("LevelScalingMul(5,1) = %f; want %f", mul, expected)
	}
	mulEq := LevelScalingMul(5, RecommendedLevel(5))
	if mulEq != 1.0 {
		t.Errorf("LevelScalingMul(5,%d) = %f; want 1.0", RecommendedLevel(5), mulEq)
	}
}

func TestLevelScalingMulCapped(t *testing.T) {
	// Being far under the recommended level must not scale without bound:
	// the penalty is capped at UnderLeveledMaxDiff levels.
	max := 1.0 + float64(UnderLeveledMaxDiff)*UnderLeveledMult
	for f := 1; f <= 60; f++ {
		m := LevelScalingMul(f, 1)
		if m > max+0.001 {
			t.Errorf("LevelScalingMul(%d,1) = %f; want <= %f (capped)", f, m, max)
		}
	}
}

func TestDangerSkulls(t *testing.T) {
	// Delve accompanies players from their first levels: a level 1 player
	// on floor 1 is right at home.
	d := CalcDanger(1, 1)
	if d.Skulls != 1 || d.IsPunished {
		t.Errorf("F1 Lv1 should be calm (1 skull, not punished): %+v", d)
	}
	// At the recommended level the danger drops to baseline.
	dRec := CalcDanger(1, RecommendedLevel(1))
	if dRec.Skulls != 1 || dRec.IsPunished {
		t.Errorf("F1 rec level should be calm: %+v", dRec)
	}
	// Being deep underleveled is still clearly dangerous.
	d2 := CalcDanger(5, 1)
	if !d2.IsPunished || d2.Skulls < 3 {
		t.Errorf("F5 Lv1: %+v", d2)
	}
}

func TestMimicChanceCap(t *testing.T) {
	for f := 1; f <= 30; f++ {
		c := MimicChance(f)
		if c > 50 {
			t.Errorf("MimicChance(%d) = %d > 50", f, c)
		}
	}
}

func TestChanceCaps(t *testing.T) {
	for f := 1; f <= 60; f++ {
		if c := CorridorChance(f); c > 40 {
			t.Errorf("CorridorChance(%d) = %d > 40", f, c)
		}
		if c := BackfireChance(f); c > 50 {
			t.Errorf("BackfireChance(%d) = %d > 50", f, c)
		}
		if c := EnemyCritChance(f); c > 25 {
			t.Errorf("EnemyCritChance(%d) = %d > 25", f, c)
		}
	}
}

func TestCombatXP(t *testing.T) {
	xp := CombatXP(1, 1)
	if xp <= 0 {
		t.Errorf("CombatXP(1,1) = %d; want >0", xp)
	}
	xpHigh := CombatXP(1, 10)
	if xpHigh > xp {
		t.Errorf("CombatXP(1,10) = %d; should be <= CombatXP(1,1)=%d", xpHigh, xp)
	}
	// Deeper floors award more XP than shallow ones at the same level.
	if CombatXP(10, 1) <= xp {
		t.Errorf("CombatXP(10,1) = %d; should be > CombatXP(1,1)=%d", CombatXP(10, 1), xp)
	}
}

func TestRollCorridorEventDistribution(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	events := 0
	for range 1000 {
		if RollCorridorEvent("crypt", 1, false, rng) != NoEvent {
			events++
		}
	}
	if events < 50 || events > 300 {
		t.Errorf("corridor events out of expected range: %d/1000", events)
	}
}

func TestTorchBurnChance(t *testing.T) {
	if TorchBurnChance(1) < 15 || TorchBurnChance(10) > 45 {
		t.Errorf("TorchBurnChance out of range")
	}
	for f := 1; f <= 60; f++ {
		if c := TorchBurnChance(f); c > 45 {
			t.Errorf("TorchBurnChance(%d) = %d > 45", f, c)
		}
	}
}

func TestBossForFloor(t *testing.T) {
	if BossForFloor(3) == nil {
		t.Error("BossForFloor(3) should exist")
	}
	if BossForFloor(4) != nil {
		t.Error("BossForFloor(4) should not exist")
	}
	if BossForFloor(9) == nil {
		t.Error("BossForFloor(9) should exist")
	}
}

func TestMerchantPrices(t *testing.T) {
	if MerchantPriceBase(1) < 1 {
		t.Error("MerchantPriceBase(1) too low")
	}
	if PotionPrice(5) <= PotionPrice(1) {
		t.Error("potion prices should scale with floor")
	}
}

func TestFloorRarityBonus(t *testing.T) {
	if floorRarityBonus(1) != 0 {
		t.Errorf("floorRarityBonus(1) = %f; want 0", floorRarityBonus(1))
	}
	if floorRarityBonus(30) > 0.15 {
		t.Errorf("floorRarityBonus(30) = %f; want capped at 0.15", floorRarityBonus(30))
	}
	if floorRarityBonus(10) <= floorRarityBonus(5) {
		t.Errorf("floorRarityBonus should grow with floor")
	}
}

func TestRollRarityShiftsWithFloor(t *testing.T) {
	// The share of high-tier loot must increase with depth: on floor 1 the
	// roll mirrors the base distribution, while floor 13+ shifts toward
	// Epic/Legendary.
	count := func(floor int, pick func(Rarity) bool) float64 {
		ok := 0
		const n = 20000
		for range n {
			if pick(rollRarity(floor, 0)) {
				ok++
			}
		}
		return float64(ok) / n
	}

	shallowEpic := count(1, func(r Rarity) bool { return r >= Epic })
	deepEpic := count(15, func(r Rarity) bool { return r >= Epic })
	if deepEpic <= shallowEpic+0.05 {
		t.Errorf("epic+ share floor 15 (%f) should exceed floor 1 (%f) by a clear margin", deepEpic, shallowEpic)
	}

	shallowCommon := count(1, func(r Rarity) bool { return r == Common })
	deepCommon := count(15, func(r Rarity) bool { return r == Common })
	if deepCommon >= shallowCommon {
		t.Errorf("common share floor 15 (%f) should drop below floor 1 (%f)", deepCommon, shallowCommon)
	}
}
