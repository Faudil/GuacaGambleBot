package delve

import (
	"math"
	"math/rand"
)

const (
	// RecLevelBase is the recommended level for floor 1. Delve is a mid-game
	// activity: underleveled players are punished hard instead of being
	// hard-locked out.
	RecLevelBase = 12

	RecLevelPerFloor = 3
	RecLevelOffset   = 0

	UnderLeveledMult    = 0.30
	UnderLeveledXPBonus = 0.25
)

func RecommendedLevel(floor int) int {
	lv := RecLevelBase + RecLevelPerFloor*(floor-1) - RecLevelOffset
	if lv < 1 {
		return 1
	}
	return lv
}

type DangerInfo struct {
	Skulls      int
	RecLevel    int
	PlayerLevel int
	IsPunished  bool
}

func CalcDanger(floor int, playerLevel int) DangerInfo {
	rec := RecommendedLevel(floor)
	diff := rec - playerLevel
	skulls := 1
	if diff > 0 {
		skulls = 1 + int(math.Ceil(float64(diff)/3.0))
	}
	if skulls > 5 {
		skulls = 5
	}
	return DangerInfo{
		Skulls:      skulls,
		RecLevel:    rec,
		PlayerLevel: playerLevel,
		IsPunished:  playerLevel < rec,
	}
}

func LevelScalingMul(floor int, playerLevel int) float64 {
	rec := RecommendedLevel(floor)
	if playerLevel >= rec {
		return 1.0
	}
	return 1.0 + float64(rec-playerLevel)*UnderLeveledMult
}

func TrapDamage(floor int) int {
	return 8 + 4*floor
}

func MimicDamage(floor int) int {
	return 10 + 5*floor
}

func AmbushDamage(floor int) int {
	return 15 + 3*floor
}

func DisarmDC(floor int) int {
	return 11 + floor
}

func FleeDC(floor int) int {
	return 10 + floor
}

func CombatFleeDC(floor int) int {
	return 8 + floor
}

func MimicChance(floor int) int {
	c := 30 + 3*floor
	if c > 70 {
		c = 70
	}
	return c
}

func CorridorChance(floor int) int {
	return 15 + 2*floor
}

func BackfireChance(floor int) int {
	return 15 + 3*floor
}

func TorchBurnChance(floor int) int {
	return 20 + 2*floor
}

func EnemyCritChance(floor int) int {
	return 10 + floor
}

type CorridorEventType int

const (
	NoEvent CorridorEventType = iota
	CorridorTrap
	CorridorAmbush
	CorridorCollapse
	CorridorSpectral
	CorridorSporeCloud
	CorridorSteamVent
	CorridorWhispers
)

func RollCorridorEvent(zone string, floor int, dark bool, rng *rand.Rand) CorridorEventType {
	chance := CorridorChance(floor)
	if dark {
		chance *= 2
	}
	if rng.Intn(100) >= chance {
		return NoEvent
	}

	zoneEventChance := 15
	if rng.Intn(100) < zoneEventChance {
		switch zone {
		case "crypt":
			return CorridorSpectral
		case "fungal_wilds":
			return CorridorSporeCloud
		case "forge_district":
			return CorridorSteamVent
		case "abyss":
			return CorridorWhispers
		}
	}

	roll := rng.Intn(100)
	switch {
	case roll < 40:
		return CorridorTrap
	case roll < 70:
		return CorridorAmbush
	default:
		return CorridorCollapse
	}
}

var zoneBosses = []struct {
	Floor   int
	Name    string
	Emoji   string
	HPBonus int
	AtkMult float64
	DefMult float64
	Zone    string
}{
	{Floor: 3, Name: "Crypt Lord", Emoji: "👑", HPBonus: 100, AtkMult: 1.75, DefMult: 1.75, Zone: "crypt"},
	{Floor: 6, Name: "Spore Tyrant", Emoji: "🍄", HPBonus: 150, AtkMult: 1.75, DefMult: 1.75, Zone: "fungal_wilds"},
	{Floor: 9, Name: "Forge Master", Emoji: "⚒️", HPBonus: 200, AtkMult: 1.75, DefMult: 1.75, Zone: "forge_district"},
}

func BossForFloor(floor int) *struct {
	Floor   int
	Name    string
	Emoji   string
	HPBonus int
	AtkMult float64
	DefMult float64
	Zone    string
} {
	for _, b := range zoneBosses {
		if b.Floor == floor {
			return &b
		}
	}
	return nil
}

func CombatXP(floor int, playerLevel int) int {
	rec := RecommendedLevel(floor)
	base := 25 + 15*floor
	if playerLevel < rec {
		bonus := 1.0 + float64(rec-playerLevel)*UnderLeveledXPBonus
		base = int(float64(base) * bonus)
	}
	return base
}

func FloorClearXP(floor int) int {
	return 10 + 5*floor
}

func BossXP(floor int) int {
	return 100 + 30*floor
}

func MerchantPriceBase(floor int) int {
	return 25 + 5*floor
}

func PotionPrice(floor int) int {
	return 40 + 10*floor
}

func TorchPrice(floor int) int {
	return 25 + 5*floor
}

func MysteryCachePrice(floor int) int {
	return 60 + 20*floor
}

func ShrineDonateCost(floor int) int {
	return 20 + 10*floor
}

func ShrinePrayDC(floor int) int {
	return 10 + floor
}

func ShrineBacklash(floor int) int {
	return 8 + 2*floor
}

func ForceDoorDC(floor int) int {
	return 12 + floor
}

func SteamVentDamage(floor int) int {
	return 5 + 2*floor
}

func KeyPrice(floor int) int {
	return 25 + 5*floor
}

func PoisonPerRoom(floor int) int {
	return 5
}

func PoisonDuration() int {
	return 3
}

func KeyDropChance(floor int) int {
	return 15
}

func IntimidateDC(floor int) int {
	return 12 + floor
}

func ReinforcedChestChance(floor int) int {
	return 20
}
