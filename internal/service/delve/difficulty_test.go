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
		{1, 1}, {2, 4}, {3, 7}, {4, 10}, {5, 13},
		{6, 16}, {7, 19}, {8, 22}, {9, 25}, {10, 28},
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
	rec := RecommendedLevel(5)
	expected := 1.0 + float64(rec-1)*UnderLeveledMult
	if math.Abs(mul-expected) > 0.001 {
		t.Errorf("LevelScalingMul(5,1) = %f; want %f", mul, expected)
	}
	mulEq := LevelScalingMul(5, 20)
	if mulEq != 1.0 {
		t.Errorf("LevelScalingMul(5,20) = %f; want 1.0", mulEq)
	}
}

func TestDangerSkulls(t *testing.T) {
	d := CalcDanger(1, 1)
	if d.Skulls != 1 || d.IsPunished {
		t.Errorf("F1 Lv1: %+v", d)
	}
	d2 := CalcDanger(5, 1)
	if !d2.IsPunished || d2.Skulls < 3 {
		t.Errorf("F5 Lv1: %+v", d2)
	}
}

func TestMimicChanceCap(t *testing.T) {
	for f := 1; f <= 30; f++ {
		c := MimicChance(f)
		if c > 70 {
			t.Errorf("MimicChance(%d) = %d > 70", f, c)
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
	if TorchBurnChance(1) < 20 || TorchBurnChance(10) > 60 {
		t.Errorf("TorchBurnChance out of range")
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
