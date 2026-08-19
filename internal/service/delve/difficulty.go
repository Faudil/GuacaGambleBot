package delve

import (
	"math"
	"math/rand"
)

const (
	// RecLevelBase is the recommended level for floor 1. Delve is meant to
	// accompany players from their first steps to endgame: floor 1 is safe
	// for a brand-new character, floor 10 (the Veil gate) sits around
	// level 19, and the abyss spans the rest of the level range.
	RecLevelBase = 1

	RecLevelPerFloor = 2
	RecLevelOffset   = 0

	// UnderleveledMult is the enemy stat penalty per level the player is
	// below the recommended level. It is capped at UnderLeveledMaxDiff
	// levels so deep underleveled runs stay harsh but survivable instead
	// of scaling without bound.
	UnderLeveledMult    = 0.15
	UnderLeveledMaxDiff = 5

	UnderLeveledXPBonus   = 0.20
	UnderLeveledMaxXPDiff = 5
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
	diff := rec - playerLevel
	if diff > UnderLeveledMaxDiff {
		diff = UnderLeveledMaxDiff
	}
	return 1.0 + float64(diff)*UnderLeveledMult
}

func TrapDamage(floor int) int {
	return 6 + 3*floor
}

func MimicDamage(floor int) int {
	return 8 + 4*floor
}

func AmbushDamage(floor int) int {
	return 12 + 2*floor
}

// CollapseDamage is the damage taken when a corridor collapse hits while the
// player has no torch to brace with.
func CollapseDamage(floor int) int {
	return 8 + 2*floor
}

// RiftGazeDamage is the backlash from failing to understand a rift.
func RiftGazeDamage(floor int) int {
	return 8 + 2*floor
}

// SacrificeHPCost is the permanent max HP price paid at a blood altar.
func SacrificeHPCost(floor int) int {
	return 12 + 4*floor
}

// GardenHealAmount is the heal granted by harvesting a garden.
func GardenHealAmount(floor int) int {
	return 12 + 4*floor
}

// ForgeScavengeGold is the gold granted by scavenging a forge.
func ForgeScavengeGold(floor int) int {
	return 25 + 8*floor
}

// PotionHealAmount is the fixed heal of a delve potion.
func PotionHealAmount() int {
	return 30
}

// RiddleFailDamage is the damage taken on a wrong riddle answer.
func RiddleFailDamage() int {
	return 10
}

// RescueTorchCost is the HP lost by a rescuer who has no torch to spare.
func RescueTorchCost() int {
	return 10
}

// BossStatMult is the stat multiplier applied to zone boss encounters.
func BossStatMult() float64 {
	return 1.75
}

// EliteStatMult is the stat multiplier applied to elite (rift) encounters.
func EliteStatMult() float64 {
	return 1.5
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
	c := 25 + 2*floor
	if c > 50 {
		c = 50
	}
	return c
}

func CorridorChance(floor int) int {
	c := 12 + floor
	if c > 40 {
		c = 40
	}
	return c
}

func BackfireChance(floor int) int {
	c := 12 + 2*floor
	if c > 50 {
		c = 50
	}
	return c
}

func TorchBurnChance(floor int) int {
	c := 15 + floor
	if c > 45 {
		c = 45
	}
	return c
}

func EnemyCritChance(floor int) int {
	c := 5 + floor/2
	if c > 25 {
		c = 25
	}
	return c
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
	CorridorBridge
	CorridorMist
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
	case roll < 35:
		return CorridorTrap
	case roll < 60:
		return CorridorAmbush
	case roll < 80:
		return CorridorCollapse
	case roll < 90:
		return CorridorBridge
	default:
		return CorridorMist
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
	base := 30 + 25*floor
	if playerLevel < rec {
		diff := rec - playerLevel
		if diff > UnderLeveledMaxXPDiff {
			diff = UnderLeveledMaxXPDiff
		}
		bonus := 1.0 + float64(diff)*UnderLeveledXPBonus
		base = int(float64(base) * bonus)
	}
	return base
}

func FloorClearXP(floor int) int {
	return 15 + 8*floor
}

func BossXP(floor int) int {
	return 150 + 40*floor
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
