package pets

import "math/rand"

const (
	RarityCommon   = "common"
	RarityRare     = "rare"
	RarityEpic     = "epic"
	RarityLegendary = "legendary"
)

const (
	BonusHUNT = 0
	BonusFARM = 1
	BonusMINE = 2
	BonusFISH = 3
)

type PetType struct {
	Name     string
	Emoji    string
	Rarity   string
	Bonus    int
	MaxHP    int
	Atk      int
	Defense  int
	Speed    int
	DGE      int
	ACC      int
	CritC    int
	CritD    float64
	SpcC     int
	Hatchable bool
}

var PetTypes = map[string]*PetType{
	"Escargot":    {Name: "Escargot", Emoji: "🐌", Rarity: RarityCommon, Bonus: BonusFARM, MaxHP: 60, Atk: 5, Defense: 15, Speed: 2, DGE: 0, ACC: 10, CritC: 5, CritD: 1.2, Hatchable: true},
	"Souris":      {Name: "Souris", Emoji: "🐀", Rarity: RarityCommon, Bonus: BonusMINE, MaxHP: 25, Atk: 12, Defense: 3, Speed: 25, DGE: 25, ACC: 5, CritC: 10, CritD: 1.5, Hatchable: true},
	"Cochon":      {Name: "Cochon", Emoji: "🐷", Rarity: RarityCommon, Bonus: BonusFARM, MaxHP: 60, Atk: 12, Defense: 8, Speed: 8, DGE: 2, ACC: 10, CritC: 5, CritD: 1.5, Hatchable: true},
	"Grenouille":  {Name: "Grenouille", Emoji: "🐸", Rarity: RarityCommon, Bonus: BonusFISH, MaxHP: 30, Atk: 15, Defense: 4, Speed: 20, DGE: 10, ACC: 10, CritC: 8, CritD: 1.5, Hatchable: true},
	"Taupe":       {Name: "Taupe", Emoji: "🦡", Rarity: RarityCommon, Bonus: BonusMINE, MaxHP: 45, Atk: 14, Defense: 8, Speed: 8, DGE: 5, ACC: 25, CritC: 10, CritD: 2.0, Hatchable: true},
	"Pélican":     {Name: "Pélican", Emoji: "🦤", Rarity: RarityCommon, Bonus: BonusFISH, MaxHP: 45, Atk: 12, Defense: 5, Speed: 18, DGE: 8, ACC: 15, CritC: 5, CritD: 1.5, Hatchable: true},
	"Mouton":      {Name: "Mouton", Emoji: "🐑", Rarity: RarityCommon, Bonus: BonusFARM, MaxHP: 55, Atk: 8, Defense: 12, Speed: 10, DGE: 5, ACC: 5, CritC: 5, CritD: 1.5, Hatchable: true},
	"Abeille":     {Name: "Abeille", Emoji: "🐝", Rarity: RarityCommon, Bonus: BonusFARM, MaxHP: 25, Atk: 24, Defense: 3, Speed: 25, DGE: 25, ACC: 5, CritC: 10, CritD: 1.5, Hatchable: true},
	"Chien":       {Name: "Chien", Emoji: "🐶", Rarity: RarityRare, Bonus: BonusHUNT, MaxHP: 65, Atk: 22, Defense: 10, Speed: 18, DGE: 8, ACC: 15, CritC: 10, CritD: 1.5, Hatchable: true},
	"Chat":        {Name: "Chat", Emoji: "😼", Rarity: RarityRare, Bonus: BonusFISH, MaxHP: 45, Atk: 25, Defense: 2, Speed: 35, DGE: 20, ACC: 10, CritC: 20, CritD: 1.8, Hatchable: true},
	"Cheval":      {Name: "Cheval", Emoji: "🐴", Rarity: RarityRare, Bonus: BonusFARM, MaxHP: 80, Atk: 18, Defense: 10, Speed: 30, DGE: 10, ACC: 10, CritC: 5, CritD: 1.5, Hatchable: true},
	"Renard":      {Name: "Renard", Emoji: "🦊", Rarity: RarityRare, Bonus: BonusMINE, MaxHP: 50, Atk: 20, Defense: 6, Speed: 25, DGE: 18, ACC: 20, CritC: 15, CritD: 1.6, Hatchable: true},
	"Singe":       {Name: "Singe", Emoji: "🐵", Rarity: RarityRare, Bonus: BonusFARM, MaxHP: 55, Atk: 22, Defense: 12, Speed: 28, DGE: 15, ACC: 15, CritC: 12, CritD: 1.5, Hatchable: true},
	"Ours":        {Name: "Ours", Emoji: "🐻", Rarity: RarityRare, Bonus: BonusMINE, MaxHP: 90, Atk: 28, Defense: 18, Speed: 8, DGE: 2, ACC: 10, CritC: 5, CritD: 2.0, Hatchable: true},
	"Chameau":     {Name: "Chameau", Emoji: "🐪", Rarity: RarityEpic, Bonus: BonusFARM, MaxHP: 120, Atk: 18, Defense: 20, Speed: 12, DGE: 5, ACC: 10, CritC: 5, CritD: 1.5, Hatchable: true},
	"Panda":       {Name: "Panda", Emoji: "🐼", Rarity: RarityEpic, Bonus: BonusFARM, MaxHP: 110, Atk: 22, Defense: 15, Speed: 10, DGE: 8, ACC: 15, CritC: 10, CritD: 1.5, Hatchable: true},
	"Tigre":       {Name: "Tigre", Emoji: "🐯", Rarity: RarityEpic, Bonus: BonusMINE, MaxHP: 85, Atk: 35, Defense: 12, Speed: 32, DGE: 15, ACC: 20, CritC: 25, CritD: 2.0, Hatchable: true},
	"Pieuvre":     {Name: "Pieuvre", Emoji: "🐙", Rarity: RarityEpic, Bonus: BonusFISH, MaxHP: 100, Atk: 25, Defense: 15, Speed: 20, DGE: 25, ACC: 30, CritC: 15, CritD: 1.5, Hatchable: true},
	"Dragon":      {Name: "Dragon", Emoji: "🐉", Rarity: RarityLegendary, Bonus: BonusHUNT, MaxHP: 130, Atk: 35, Defense: 20, Speed: 20, DGE: 15, ACC: 25, CritC: 10, CritD: 1.2, Hatchable: true},
	"Hamster":     {Name: "Hamster", Emoji: "🐹", Rarity: RarityCommon, Bonus: BonusFARM, MaxHP: 25, Atk: 5, Defense: 10, Speed: 25, DGE: 22, ACC: 5, CritC: 10, CritD: 1.5, Hatchable: true},
	"Fourmi":      {Name: "Fourmi", Emoji: "🐜", Rarity: RarityCommon, Bonus: BonusMINE, MaxHP: 5, Atk: 5, Defense: 5, Speed: 5, DGE: 5, ACC: 5, CritC: 5, CritD: 1.2, Hatchable: true},
	"Hérisson":    {Name: "Hérisson", Emoji: "🦔", Rarity: RarityCommon, Bonus: BonusMINE, MaxHP: 28, Atk: 5, Defense: 26, Speed: 18, DGE: 5, ACC: 5, CritC: 5, CritD: 1.5, Hatchable: true},
	"Canard":      {Name: "Canard", Emoji: "🦆", Rarity: RarityCommon, Bonus: BonusFISH, MaxHP: 30, Atk: 10, Defense: 8, Speed: 21, DGE: 9, ACC: 17, CritC: 5, CritD: 1.7, Hatchable: true},
	"Chouette":    {Name: "Chouette", Emoji: "🦉", Rarity: RarityCommon, Bonus: BonusHUNT, MaxHP: 35, Atk: 15, Defense: 5, Speed: 25, DGE: 8, ACC: 20, CritC: 10, CritD: 1.5, Hatchable: true},
	"Paresseux":   {Name: "Paresseux", Emoji: "🦥", Rarity: RarityCommon, Bonus: BonusFISH, MaxHP: 50, Atk: 15, Defense: 15, Speed: 2, DGE: 0, ACC: 10, CritC: 5, CritD: 1.2, Hatchable: true},
	"Kangourou":   {Name: "Kangourou", Emoji: "🦘", Rarity: RarityEpic, Bonus: BonusFARM, MaxHP: 65, Atk: 25, Defense: 15, Speed: 27, DGE: 18, ACC: 15, CritC: 15, CritD: 2.0, Hatchable: true},
	"Iguane":      {Name: "Iguane", Emoji: "🦎", Rarity: RarityEpic, Bonus: BonusHUNT, MaxHP: 60, Atk: 20, Defense: 20, Speed: 20, DGE: 15, ACC: 18, CritC: 18, CritD: 1.7, Hatchable: true},
	"Gorille":     {Name: "Gorille", Emoji: "🦍", Rarity: RarityRare, Bonus: BonusFARM, MaxHP: 70, Atk: 22, Defense: 22, Speed: 12, DGE: 7, ACC: 12, CritC: 10, CritD: 1.6, Hatchable: true},
	"Scorpion":    {Name: "Scorpion", Emoji: "🦂", Rarity: RarityRare, Bonus: BonusHUNT, MaxHP: 40, Atk: 22, Defense: 10, Speed: 25, DGE: 15, ACC: 15, CritC: 25, CritD: 1.8, Hatchable: true},
	"Bison":       {Name: "Bison", Emoji: "🦬", Rarity: RarityRare, Bonus: BonusFARM, MaxHP: 80, Atk: 10, Defense: 18, Speed: 25, DGE: 8, ACC: 15, CritC: 5, CritD: 1.6, Hatchable: true},
	"Aigle":       {Name: "Aigle", Emoji: "🦅", Rarity: RarityEpic, Bonus: BonusHUNT, MaxHP: 80, Atk: 30, Defense: 10, Speed: 35, DGE: 22, ACC: 25, CritC: 20, CritD: 2.0, Hatchable: true},
	"Rhino":       {Name: "Rhino", Emoji: "🦏", Rarity: RarityEpic, Bonus: BonusMINE, MaxHP: 90, Atk: 26, Defense: 32, Speed: 12, DGE: 5, ACC: 10, CritC: 12, CritD: 1.7, Hatchable: true},
	"Crocodile":   {Name: "Crocodile", Emoji: "🐊", Rarity: RarityEpic, Bonus: BonusFISH, MaxHP: 80, Atk: 30, Defense: 20, Speed: 18, DGE: 17, ACC: 25, CritC: 20, CritD: 2.0, Hatchable: true},
	"Putois":      {Name: "Putois", Emoji: "🦨", Rarity: RarityCommon, Bonus: BonusFARM, MaxHP: 90, Atk: 15, Defense: 26, Speed: 12, DGE: 24, ACC: 20, CritC: 15, CritD: 1.7, Hatchable: true},
	"Dauphin":     {Name: "Dauphin", Emoji: "🐬", Rarity: RarityEpic, Bonus: BonusFISH, MaxHP: 100, Atk: 18, Defense: 15, Speed: 32, DGE: 22, ACC: 30, CritC: 20, CritD: 2.0, Hatchable: true},
	"Léopard":     {Name: "Léopard", Emoji: "🐆", Rarity: RarityEpic, Bonus: BonusHUNT, MaxHP: 75, Atk: 35, Defense: 12, Speed: 46, DGE: 16, ACC: 18, CritC: 20, CritD: 1.7, Hatchable: true},
	"Lion":        {Name: "Lion", Emoji: "🦁", Rarity: RarityEpic, Bonus: BonusHUNT, MaxHP: 95, Atk: 35, Defense: 18, Speed: 18, DGE: 10, ACC: 22, CritC: 20, CritD: 1.7, Hatchable: true},
	"Ours polaire": {Name: "Ours polaire", Emoji: "🐻‍❄️", Rarity: RarityRare, Bonus: BonusMINE, MaxHP: 100, Atk: 30, Defense: 20, Speed: 8, DGE: 2, ACC: 10, CritC: 5, CritD: 2.0, Hatchable: true},
	"Tyrannosaure": {Name: "Tyrannosaure", Emoji: "🦖", Rarity: RarityLegendary, Bonus: BonusHUNT, MaxHP: 120, Atk: 40, Defense: 20, Speed: 20, DGE: 15, ACC: 20, CritC: 15, CritD: 2.0, Hatchable: true},
	"Diplodocus":  {Name: "Diplodocus", Emoji: "🦕", Rarity: RarityLegendary, Bonus: BonusFISH, MaxHP: 140, Atk: 20, Defense: 40, Speed: 15, DGE: 10, ACC: 20, CritC: 10, CritD: 1.2, Hatchable: true},
	"Mamouth":     {Name: "Mamouth", Emoji: "🦣", Rarity: RarityLegendary, Bonus: BonusMINE, MaxHP: 180, Atk: 20, Defense: 40, Speed: 10, DGE: 5, ACC: 10, CritC: 5, CritD: 1.5, Hatchable: true},
	"Mégalodon":   {Name: "Mégalodon", Emoji: "🦈", Rarity: RarityLegendary, Bonus: BonusFISH, MaxHP: 130, Atk: 35, Defense: 25, Speed: 18, DGE: 10, ACC: 20, CritC: 15, CritD: 1.5, Hatchable: true},
	"Kraken":      {Name: "Kraken", Emoji: "🦑", Rarity: RarityLegendary, Bonus: BonusFISH, MaxHP: 130, Atk: 25, Defense: 35, Speed: 18, DGE: 20, ACC: 10, CritC: 15, CritD: 1.5, Hatchable: true},
	"Licorne":     {Name: "Licorne", Emoji: "🦄", Rarity: RarityLegendary, Bonus: BonusFARM, MaxHP: 100, Atk: 28, Defense: 20, Speed: 32, DGE: 20, ACC: 27, CritC: 12, CritD: 1.2, Hatchable: true},
	"Phoenix":     {Name: "Phoenix", Emoji: "🐦‍🔥", Rarity: RarityLegendary, Bonus: BonusFARM, MaxHP: 200, Atk: 20, Defense: 15, Speed: 30, DGE: 25, ACC: 15, CritC: 15, CritD: 1.5, Hatchable: true},
	"Cerbère":     {Name: "Cerbère", Emoji: "🐺🐺🐺", Rarity: RarityLegendary, Bonus: BonusHUNT, MaxHP: 90, Atk: 35, Defense: 20, Speed: 28, DGE: 15, ACC: 25, CritC: 25, CritD: 1.5, Hatchable: true},
	"Fenrir":      {Name: "Fenrir", Emoji: "🐺⛓️", Rarity: RarityLegendary, Bonus: BonusMINE, MaxHP: 100, Atk: 40, Defense: 20, Speed: 30, DGE: 20, ACC: 20, CritC: 20, CritD: 1.5, Hatchable: true},
	"Ratatosk":    {Name: "Ratatosk", Emoji: "🐿️❄️", Rarity: RarityLegendary, Bonus: BonusMINE, MaxHP: 90, Atk: 30, Defense: 15, Speed: 40, DGE: 25, ACC: 30, CritC: 25, CritD: 2.0, Hatchable: true},
	"Nidhögg":     {Name: "Nidhögg", Emoji: "🐍⚡", Rarity: RarityLegendary, Bonus: BonusMINE, MaxHP: 110, Atk: 32, Defense: 18, Speed: 20, DGE: 20, ACC: 50, CritC: 18, CritD: 1.7, Hatchable: true},
	"Bedawang":    {Name: "Bedawang", Emoji: "🐢🌳", Rarity: RarityLegendary, Bonus: BonusFARM, MaxHP: 200, Atk: 25, Defense: 40, Speed: 1, DGE: 0, ACC: 25, CritC: 10, CritD: 1.2, Hatchable: true},
}

var ClassicPool = []string{
	"Escargot", "Souris", "Cochon", "Grenouille", "Taupe", "Pélican", "Mouton", "Abeille",
	"Chien", "Chat", "Cheval", "Renard", "Singe", "Ours",
	"Chameau", "Panda", "Tigre", "Pieuvre",
	"Dragon",
}

var RarityXP = map[string]float64{
	RarityCommon:   0.5,
	RarityRare:     1.0,
	RarityEpic:     1.5,
	RarityLegendary: 2.0,
}

var RarityFoodCapacity = map[string]int{
	RarityCommon:   5,
	RarityRare:     4,
	RarityEpic:     3,
	RarityLegendary: 2,
}

func RollGacha(targetRarity string) string {
	if targetRarity == "" {
		r := rand.Float64()
		switch {
		case r < 0.05:
			targetRarity = RarityLegendary
		case r < 0.15:
			targetRarity = RarityEpic
		case r < 0.40:
			targetRarity = RarityRare
		default:
			targetRarity = RarityCommon
		}
	}
	possible := make([]string, 0)
	for _, name := range ClassicPool {
		if pt, ok := PetTypes[name]; ok && pt.Rarity == targetRarity {
			possible = append(possible, name)
		}
	}
	if len(possible) == 0 {
		for name, pt := range PetTypes {
			if pt.Rarity == targetRarity {
				possible = append(possible, name)
			}
		}
	}
	if len(possible) == 0 {
		return "Escargot"
	}
	return possible[rand.Intn(len(possible))]
}

var rarityOrder = map[string]int{RarityCommon: 0, RarityRare: 1, RarityEpic: 2, RarityLegendary: 3}

func TradeUpRarity(rarity string) (int, string) {
	switch rarity {
	case RarityCommon:
		return 5, RarityRare
	case RarityRare:
		return 4, RarityEpic
	case RarityEpic:
		return 3, RarityLegendary
	}
	return 0, ""
}

func RarityBonus(rarity string) string {
	switch rarity {
	case RarityCommon:
		return "common"
	case RarityRare:
		return "rare"
	case RarityEpic:
		return "epic"
	case RarityLegendary:
		return "legendary"
	}
	return "common"
}
