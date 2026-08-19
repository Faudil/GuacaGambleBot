package pets

import "math/rand"

const (
	RarityCommon    = "common"
	RarityRare      = "rare"
	RarityEpic      = "epic"
	RarityLegendary = "legendary"
)

const (
	BiomeForest   = "forest"
	BiomeCave     = "cave"
	BiomeDesert   = "desert"
	BiomeMountain = "mountain"
	BiomeOcean    = "ocean"
	BiomeTundra   = "tundra"
	BiomeVolcano  = "volcano"
)

var Biomes = []string{BiomeForest, BiomeCave, BiomeDesert, BiomeMountain, BiomeOcean, BiomeTundra, BiomeVolcano}

const (
	BonusHUNT = 0
	BonusFARM = 1
	BonusMINE = 2
	BonusFISH = 3
)

type PetType struct {
	Name      string
	Emoji     string
	Rarity    string
	Biome     string
	Bonus     int
	MaxHP     int
	Atk       int
	Defense   int
	Speed     int
	DGE       int
	ACC       int
	CritC     int
	CritD     float64
	SpcC      int
	Hatchable bool
}

var PetTypes = map[string]*PetType{
	// ── Forest 🌲 ──
	"Escargot":     {Name: "Escargot", Emoji: "🐌", Rarity: RarityCommon, Biome: BiomeForest, Bonus: BonusFARM, MaxHP: 60, Atk: 5, Defense: 15, Speed: 2, DGE: 0, ACC: 10, CritC: 5, CritD: 1.2, Hatchable: true},
	"Cochon":       {Name: "Cochon", Emoji: "🐷", Rarity: RarityCommon, Biome: BiomeForest, Bonus: BonusFARM, MaxHP: 60, Atk: 12, Defense: 8, Speed: 8, DGE: 2, ACC: 10, CritC: 5, CritD: 1.5, Hatchable: true},
	"Abeille":      {Name: "Abeille", Emoji: "🐝", Rarity: RarityCommon, Biome: BiomeForest, Bonus: BonusFARM, MaxHP: 25, Atk: 24, Defense: 3, Speed: 25, DGE: 25, ACC: 5, CritC: 10, CritD: 1.5, Hatchable: true},
	"Hérisson":     {Name: "Hérisson", Emoji: "🦔", Rarity: RarityCommon, Biome: BiomeForest, Bonus: BonusMINE, MaxHP: 28, Atk: 5, Defense: 26, Speed: 18, DGE: 5, ACC: 5, CritC: 5, CritD: 1.5, Hatchable: true},
	"Chouette":     {Name: "Chouette", Emoji: "🦉", Rarity: RarityCommon, Biome: BiomeForest, Bonus: BonusHUNT, MaxHP: 35, Atk: 15, Defense: 5, Speed: 25, DGE: 8, ACC: 20, CritC: 10, CritD: 1.5, Hatchable: true},
	"Méganeura":    {Name: "Méganeura", Emoji: "🦟", Rarity: RarityCommon, Biome: BiomeForest, Bonus: BonusHUNT, MaxHP: 25, Atk: 22, Defense: 4, Speed: 24, DGE: 22, ACC: 12, CritC: 12, CritD: 1.6, Hatchable: true},
	"Chien":        {Name: "Chien", Emoji: "🐶", Rarity: RarityRare, Biome: BiomeForest, Bonus: BonusHUNT, MaxHP: 65, Atk: 22, Defense: 10, Speed: 18, DGE: 8, ACC: 15, CritC: 10, CritD: 1.5, Hatchable: true},
	"Renard":       {Name: "Renard", Emoji: "🦊", Rarity: RarityRare, Biome: BiomeForest, Bonus: BonusMINE, MaxHP: 50, Atk: 20, Defense: 6, Speed: 25, DGE: 18, ACC: 20, CritC: 15, CritD: 1.6, Hatchable: true},
	"Archéoptéryx": {Name: "Archéoptéryx", Emoji: "🪶", Rarity: RarityRare, Biome: BiomeForest, Bonus: BonusHUNT, MaxHP: 50, Atk: 20, Defense: 8, Speed: 30, DGE: 18, ACC: 20, CritC: 15, CritD: 1.6, Hatchable: true},
	"Titanoboa":    {Name: "Titanoboa", Emoji: "🐍", Rarity: RarityEpic, Biome: BiomeForest, Bonus: BonusHUNT, MaxHP: 95, Atk: 30, Defense: 20, Speed: 15, DGE: 20, ACC: 18, CritC: 20, CritD: 1.7, Hatchable: true},
	"Licorne":      {Name: "Licorne", Emoji: "🦄", Rarity: RarityLegendary, Biome: BiomeForest, Bonus: BonusFARM, MaxHP: 100, Atk: 28, Defense: 20, Speed: 32, DGE: 20, ACC: 27, CritC: 12, CritD: 1.2, Hatchable: true},

	// ── Cave 🦇 ──
	"Souris":     {Name: "Souris", Emoji: "🐀", Rarity: RarityCommon, Biome: BiomeCave, Bonus: BonusMINE, MaxHP: 25, Atk: 12, Defense: 3, Speed: 25, DGE: 25, ACC: 5, CritC: 10, CritD: 1.5, Hatchable: true},
	"Taupe":      {Name: "Taupe", Emoji: "🦡", Rarity: RarityCommon, Biome: BiomeCave, Bonus: BonusMINE, MaxHP: 45, Atk: 14, Defense: 8, Speed: 8, DGE: 5, ACC: 25, CritC: 10, CritD: 2.0, Hatchable: true},
	"Putois":     {Name: "Putois", Emoji: "🦨", Rarity: RarityCommon, Biome: BiomeCave, Bonus: BonusFARM, MaxHP: 90, Atk: 15, Defense: 26, Speed: 12, DGE: 24, ACC: 20, CritC: 15, CritD: 1.7, Hatchable: true},
	"Ours":       {Name: "Ours", Emoji: "🐻", Rarity: RarityRare, Biome: BiomeCave, Bonus: BonusMINE, MaxHP: 90, Atk: 28, Defense: 18, Speed: 8, DGE: 2, ACC: 10, CritC: 5, CritD: 2.0, Hatchable: true},
	"Scorpion":   {Name: "Scorpion", Emoji: "🦂", Rarity: RarityRare, Biome: BiomeCave, Bonus: BonusHUNT, MaxHP: 40, Atk: 22, Defense: 10, Speed: 25, DGE: 15, ACC: 15, CritC: 25, CritD: 1.8, Hatchable: true},
	"Dimétrodon": {Name: "Dimétrodon", Emoji: "🦎", Rarity: RarityRare, Biome: BiomeCave, Bonus: BonusMINE, MaxHP: 80, Atk: 26, Defense: 16, Speed: 12, DGE: 5, ACC: 15, CritC: 10, CritD: 1.8, Hatchable: true},
	"Entelodon":  {Name: "Entelodon", Emoji: "🐗", Rarity: RarityEpic, Biome: BiomeCave, Bonus: BonusMINE, MaxHP: 100, Atk: 30, Defense: 15, Speed: 18, DGE: 8, ACC: 15, CritC: 15, CritD: 1.7, Hatchable: true},
	"Cerbère":    {Name: "Cerbère", Emoji: "🐺🐺🐺", Rarity: RarityLegendary, Biome: BiomeCave, Bonus: BonusHUNT, MaxHP: 90, Atk: 35, Defense: 20, Speed: 28, DGE: 15, ACC: 25, CritC: 25, CritD: 1.5, Hatchable: true},
	"Nidhögg":    {Name: "Nidhögg", Emoji: "🐍⚡", Rarity: RarityLegendary, Biome: BiomeCave, Bonus: BonusMINE, MaxHP: 110, Atk: 32, Defense: 18, Speed: 20, DGE: 20, ACC: 50, CritC: 18, CritD: 1.7, Hatchable: true},

	// ── Desert 🏜️ ──
	"Hamster":      {Name: "Hamster", Emoji: "🐹", Rarity: RarityCommon, Biome: BiomeDesert, Bonus: BonusFARM, MaxHP: 25, Atk: 5, Defense: 10, Speed: 25, DGE: 22, ACC: 5, CritC: 10, CritD: 1.5, Hatchable: true},
	"Bison":        {Name: "Bison", Emoji: "🦬", Rarity: RarityRare, Biome: BiomeDesert, Bonus: BonusFARM, MaxHP: 80, Atk: 10, Defense: 18, Speed: 25, DGE: 8, ACC: 15, CritC: 5, CritD: 1.6, Hatchable: true},
	"Doedicurus":   {Name: "Doedicurus", Emoji: "🦛", Rarity: RarityRare, Biome: BiomeDesert, Bonus: BonusMINE, MaxHP: 95, Atk: 20, Defense: 22, Speed: 8, DGE: 2, ACC: 10, CritC: 5, CritD: 1.5, Hatchable: true},
	"Chameau":      {Name: "Chameau", Emoji: "🐪", Rarity: RarityEpic, Biome: BiomeDesert, Bonus: BonusFARM, MaxHP: 120, Atk: 18, Defense: 20, Speed: 12, DGE: 5, ACC: 10, CritC: 5, CritD: 1.5, Hatchable: true},
	"Iguane":       {Name: "Iguane", Emoji: "🦎", Rarity: RarityEpic, Biome: BiomeDesert, Bonus: BonusHUNT, MaxHP: 60, Atk: 20, Defense: 20, Speed: 20, DGE: 15, ACC: 18, CritC: 18, CritD: 1.7, Hatchable: true},
	"Rhino":        {Name: "Rhino", Emoji: "🦏", Rarity: RarityEpic, Biome: BiomeDesert, Bonus: BonusMINE, MaxHP: 90, Atk: 26, Defense: 32, Speed: 12, DGE: 5, ACC: 10, CritC: 12, CritD: 1.7, Hatchable: true},
	"Lion":         {Name: "Lion", Emoji: "🦁", Rarity: RarityEpic, Biome: BiomeDesert, Bonus: BonusHUNT, MaxHP: 95, Atk: 35, Defense: 18, Speed: 18, DGE: 10, ACC: 22, CritC: 20, CritD: 1.7, Hatchable: true},
	"Kangourou":    {Name: "Kangourou", Emoji: "🦘", Rarity: RarityEpic, Biome: BiomeDesert, Bonus: BonusFARM, MaxHP: 65, Atk: 25, Defense: 15, Speed: 27, DGE: 18, ACC: 15, CritC: 15, CritD: 2.0, Hatchable: true},
	"Phorusrhacos": {Name: "Phorusrhacos", Emoji: "🦤", Rarity: RarityEpic, Biome: BiomeDesert, Bonus: BonusHUNT, MaxHP: 85, Atk: 34, Defense: 12, Speed: 28, DGE: 18, ACC: 22, CritC: 22, CritD: 1.8, Hatchable: true},

	// ── Mountain 🏔️ ──
	"Mouton":     {Name: "Mouton", Emoji: "🐑", Rarity: RarityCommon, Biome: BiomeMountain, Bonus: BonusFARM, MaxHP: 55, Atk: 8, Defense: 12, Speed: 10, DGE: 5, ACC: 5, CritC: 5, CritD: 1.5, Hatchable: true},
	"Cheval":     {Name: "Cheval", Emoji: "🐴", Rarity: RarityRare, Biome: BiomeMountain, Bonus: BonusFARM, MaxHP: 80, Atk: 18, Defense: 10, Speed: 30, DGE: 10, ACC: 10, CritC: 5, CritD: 1.5, Hatchable: true},
	"Gorille":    {Name: "Gorille", Emoji: "🦍", Rarity: RarityRare, Biome: BiomeMountain, Bonus: BonusFARM, MaxHP: 70, Atk: 22, Defense: 22, Speed: 12, DGE: 7, ACC: 12, CritC: 10, CritD: 1.6, Hatchable: true},
	"Ptéranodon": {Name: "Ptéranodon", Emoji: "🦇", Rarity: RarityRare, Biome: BiomeMountain, Bonus: BonusHUNT, MaxHP: 70, Atk: 24, Defense: 12, Speed: 25, DGE: 15, ACC: 25, CritC: 15, CritD: 1.7, Hatchable: true},
	"Aigle":      {Name: "Aigle", Emoji: "🦅", Rarity: RarityEpic, Biome: BiomeMountain, Bonus: BonusHUNT, MaxHP: 80, Atk: 30, Defense: 10, Speed: 35, DGE: 22, ACC: 25, CritC: 20, CritD: 2.0, Hatchable: true},
	"Panda":      {Name: "Panda", Emoji: "🐼", Rarity: RarityEpic, Biome: BiomeMountain, Bonus: BonusFARM, MaxHP: 110, Atk: 22, Defense: 15, Speed: 10, DGE: 8, ACC: 15, CritC: 10, CritD: 1.5, Hatchable: true},
	"Léopard":    {Name: "Léopard", Emoji: "🐆", Rarity: RarityEpic, Biome: BiomeMountain, Bonus: BonusHUNT, MaxHP: 75, Atk: 35, Defense: 12, Speed: 46, DGE: 16, ACC: 18, CritC: 20, CritD: 1.7, Hatchable: true},
	"Diplodocus": {Name: "Diplodocus", Emoji: "🦕", Rarity: RarityLegendary, Biome: BiomeMountain, Bonus: BonusFISH, MaxHP: 140, Atk: 20, Defense: 40, Speed: 15, DGE: 10, ACC: 20, CritC: 10, CritD: 1.2, Hatchable: true},

	// ── Ocean 🌊 ──
	"Grenouille":   {Name: "Grenouille", Emoji: "🐸", Rarity: RarityCommon, Biome: BiomeOcean, Bonus: BonusFISH, MaxHP: 30, Atk: 15, Defense: 4, Speed: 20, DGE: 10, ACC: 10, CritC: 8, CritD: 1.5, Hatchable: true},
	"Pélican":      {Name: "Pélican", Emoji: "🦤", Rarity: RarityCommon, Biome: BiomeOcean, Bonus: BonusFISH, MaxHP: 45, Atk: 12, Defense: 5, Speed: 18, DGE: 8, ACC: 15, CritC: 5, CritD: 1.5, Hatchable: true},
	"Canard":       {Name: "Canard", Emoji: "🦆", Rarity: RarityCommon, Biome: BiomeOcean, Bonus: BonusFISH, MaxHP: 30, Atk: 10, Defense: 8, Speed: 21, DGE: 9, ACC: 17, CritC: 5, CritD: 1.7, Hatchable: true},
	"Trilobite":    {Name: "Trilobite", Emoji: "🦀", Rarity: RarityCommon, Biome: BiomeOcean, Bonus: BonusFISH, MaxHP: 40, Atk: 10, Defense: 20, Speed: 5, DGE: 0, ACC: 10, CritC: 5, CritD: 1.5, Hatchable: true},
	"Ammonite":     {Name: "Ammonite", Emoji: "🐚", Rarity: RarityCommon, Biome: BiomeOcean, Bonus: BonusFISH, MaxHP: 35, Atk: 12, Defense: 15, Speed: 8, DGE: 5, ACC: 15, CritC: 5, CritD: 1.5, Hatchable: true},
	"Anomalocaris": {Name: "Anomalocaris", Emoji: "🦐", Rarity: RarityCommon, Biome: BiomeOcean, Bonus: BonusFISH, MaxHP: 30, Atk: 18, Defense: 6, Speed: 20, DGE: 15, ACC: 10, CritC: 12, CritD: 1.6, Hatchable: true},
	"Orthoceras":   {Name: "Orthoceras", Emoji: "🦑", Rarity: RarityCommon, Biome: BiomeOcean, Bonus: BonusFISH, MaxHP: 45, Atk: 15, Defense: 10, Speed: 10, DGE: 8, ACC: 15, CritC: 10, CritD: 1.6, Hatchable: true},
	"Pieuvre":      {Name: "Pieuvre", Emoji: "🐙", Rarity: RarityEpic, Biome: BiomeOcean, Bonus: BonusFISH, MaxHP: 100, Atk: 25, Defense: 15, Speed: 20, DGE: 25, ACC: 30, CritC: 15, CritD: 1.5, Hatchable: true},
	"Crocodile":    {Name: "Crocodile", Emoji: "🐊", Rarity: RarityEpic, Biome: BiomeOcean, Bonus: BonusFISH, MaxHP: 80, Atk: 30, Defense: 20, Speed: 18, DGE: 17, ACC: 25, CritC: 20, CritD: 2.0, Hatchable: true},
	"Dauphin":      {Name: "Dauphin", Emoji: "🐬", Rarity: RarityEpic, Biome: BiomeOcean, Bonus: BonusFISH, MaxHP: 100, Atk: 18, Defense: 15, Speed: 32, DGE: 22, ACC: 30, CritC: 20, CritD: 2.0, Hatchable: true},
	"Mosasaurus":   {Name: "Mosasaurus", Emoji: "🐊", Rarity: RarityEpic, Biome: BiomeOcean, Bonus: BonusFISH, MaxHP: 110, Atk: 32, Defense: 18, Speed: 22, DGE: 12, ACC: 20, CritC: 15, CritD: 1.8, Hatchable: true},
	"Mégalodon":    {Name: "Mégalodon", Emoji: "🦈", Rarity: RarityLegendary, Biome: BiomeOcean, Bonus: BonusFISH, MaxHP: 130, Atk: 35, Defense: 25, Speed: 18, DGE: 10, ACC: 20, CritC: 15, CritD: 1.5, Hatchable: true},
	"Kraken":       {Name: "Kraken", Emoji: "🦑", Rarity: RarityLegendary, Biome: BiomeOcean, Bonus: BonusFISH, MaxHP: 130, Atk: 25, Defense: 35, Speed: 18, DGE: 20, ACC: 10, CritC: 15, CritD: 1.5, Hatchable: true},
	"Bedawang":     {Name: "Bedawang", Emoji: "🐢🌳", Rarity: RarityLegendary, Biome: BiomeOcean, Bonus: BonusFARM, MaxHP: 200, Atk: 25, Defense: 40, Speed: 1, DGE: 0, ACC: 25, CritC: 10, CritD: 1.2, Hatchable: true},

	// ── Tundra ❄️ ──
	"Paresseux":          {Name: "Paresseux", Emoji: "🦥", Rarity: RarityCommon, Biome: BiomeTundra, Bonus: BonusFISH, MaxHP: 50, Atk: 15, Defense: 15, Speed: 2, DGE: 0, ACC: 10, CritC: 5, CritD: 1.2, Hatchable: true},
	"Singe":              {Name: "Singe", Emoji: "🐵", Rarity: RarityRare, Biome: BiomeTundra, Bonus: BonusFARM, MaxHP: 55, Atk: 22, Defense: 12, Speed: 28, DGE: 15, ACC: 15, CritC: 12, CritD: 1.5, Hatchable: true},
	"Ours polaire":       {Name: "Ours polaire", Emoji: "🐻‍❄️", Rarity: RarityRare, Biome: BiomeTundra, Bonus: BonusMINE, MaxHP: 100, Atk: 30, Defense: 20, Speed: 8, DGE: 2, ACC: 10, CritC: 5, CritD: 2.0, Hatchable: true},
	"Smilodon":           {Name: "Smilodon", Emoji: "🐯", Rarity: RarityRare, Biome: BiomeTundra, Bonus: BonusHUNT, MaxHP: 75, Atk: 30, Defense: 12, Speed: 28, DGE: 15, ACC: 18, CritC: 20, CritD: 1.8, Hatchable: true},
	"Mégalocéros":        {Name: "Mégalocéros", Emoji: "🦌", Rarity: RarityRare, Biome: BiomeTundra, Bonus: BonusFARM, MaxHP: 90, Atk: 18, Defense: 15, Speed: 20, DGE: 8, ACC: 12, CritC: 10, CritD: 1.6, Hatchable: true},
	"Rhinocéros laineux": {Name: "Rhinocéros laineux", Emoji: "🦏", Rarity: RarityEpic, Biome: BiomeTundra, Bonus: BonusMINE, MaxHP: 120, Atk: 28, Defense: 30, Speed: 12, DGE: 5, ACC: 12, CritC: 10, CritD: 1.6, Hatchable: true},
	"Mamouth":            {Name: "Mamouth", Emoji: "🦣", Rarity: RarityLegendary, Biome: BiomeTundra, Bonus: BonusMINE, MaxHP: 180, Atk: 20, Defense: 40, Speed: 10, DGE: 5, ACC: 10, CritC: 5, CritD: 1.5, Hatchable: true},
	"Fenrir":             {Name: "Fenrir", Emoji: "🐺⛓️", Rarity: RarityLegendary, Biome: BiomeTundra, Bonus: BonusMINE, MaxHP: 100, Atk: 40, Defense: 20, Speed: 30, DGE: 20, ACC: 20, CritC: 20, CritD: 1.5, Hatchable: true},
	"Ratatosk":           {Name: "Ratatosk", Emoji: "🐿️❄️", Rarity: RarityLegendary, Biome: BiomeTundra, Bonus: BonusMINE, MaxHP: 90, Atk: 30, Defense: 15, Speed: 40, DGE: 25, ACC: 30, CritC: 25, CritD: 2.0, Hatchable: true},

	// ── Volcano 🌋 ──
	"Fourmi":       {Name: "Fourmi", Emoji: "🐜", Rarity: RarityCommon, Biome: BiomeVolcano, Bonus: BonusMINE, MaxHP: 5, Atk: 5, Defense: 5, Speed: 5, DGE: 5, ACC: 5, CritC: 5, CritD: 1.2, Hatchable: true},
	"Chat":         {Name: "Chat", Emoji: "😼", Rarity: RarityRare, Biome: BiomeVolcano, Bonus: BonusFISH, MaxHP: 45, Atk: 25, Defense: 2, Speed: 35, DGE: 20, ACC: 10, CritC: 20, CritD: 1.8, Hatchable: true},
	"Tigre":        {Name: "Tigre", Emoji: "🐯", Rarity: RarityEpic, Biome: BiomeVolcano, Bonus: BonusMINE, MaxHP: 85, Atk: 35, Defense: 12, Speed: 32, DGE: 15, ACC: 20, CritC: 25, CritD: 2.0, Hatchable: true},
	"Dragon":       {Name: "Dragon", Emoji: "🐉", Rarity: RarityLegendary, Biome: BiomeVolcano, Bonus: BonusHUNT, MaxHP: 130, Atk: 35, Defense: 20, Speed: 20, DGE: 15, ACC: 25, CritC: 10, CritD: 1.2, Hatchable: true},
	"Tyrannosaure": {Name: "Tyrannosaure", Emoji: "🦖", Rarity: RarityLegendary, Biome: BiomeVolcano, Bonus: BonusHUNT, MaxHP: 120, Atk: 40, Defense: 20, Speed: 20, DGE: 15, ACC: 20, CritC: 15, CritD: 2.0, Hatchable: true},
	"Phoenix":      {Name: "Phoenix", Emoji: "🐦‍🔥", Rarity: RarityLegendary, Biome: BiomeVolcano, Bonus: BonusFARM, MaxHP: 200, Atk: 20, Defense: 15, Speed: 30, DGE: 25, ACC: 15, CritC: 15, CritD: 1.5, Hatchable: true},
}

var RarityXP = map[string]float64{
	RarityCommon:    0.5,
	RarityRare:      1.0,
	RarityEpic:      1.5,
	RarityLegendary: 2.0,
}

type PersonalityTrait struct {
	Name        string
	Emoji       string
	Rarity      string // "common" or "uncommon"
	Description string
}

var PersonalityTraits = map[string]*PersonalityTrait{
	"brave":       {Name: "Brave", Emoji: "⚔️", Rarity: "common", Description: "Bold and fearless in battle."},
	"playful":     {Name: "Playful", Emoji: "🎾", Rarity: "common", Description: "Full of energy and always up for fun."},
	"grumpy":      {Name: "Grumpy", Emoji: "😤", Rarity: "common", Description: "Easily annoyed but deeply loyal."},
	"curious":     {Name: "Curious", Emoji: "🔍", Rarity: "common", Description: "Always exploring and finding new things."},
	"gentle":      {Name: "Gentle", Emoji: "🤲", Rarity: "common", Description: "Soft-hearted and caring."},
	"timid":       {Name: "Timid", Emoji: "😰", Rarity: "common", Description: "Skittish but quick on its feet."},
	"fierce":      {Name: "Fierce", Emoji: "🔥", Rarity: "uncommon", Description: "Aggressive and relentless in combat."},
	"sleepy":      {Name: "Sleepy", Emoji: "💤", Rarity: "uncommon", Description: "Always napping, but surprisingly resilient."},
	"loyal":       {Name: "Loyal", Emoji: "❤️", Rarity: "uncommon", Description: "Devoted companion that never wavers."},
	"mischievous": {Name: "Mischievous", Emoji: "😈", Rarity: "uncommon", Description: "Loves pranks and causing harmless trouble."},
}

func RandomPersonality() string {
	// Common personalities: 70% chance, uncommon: 30%
	common := make([]string, 0)
	uncommon := make([]string, 0)
	for id, t := range PersonalityTraits {
		if t.Rarity == "common" {
			common = append(common, id)
		} else {
			uncommon = append(uncommon, id)
		}
	}
	if rand.Float64() < 0.70 {
		return common[rand.Intn(len(common))]
	}
	return uncommon[rand.Intn(len(uncommon))]
}

func RollGacha(targetRarity string, biome string) string {
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
	possible := petsByBiomeAndRarity(biome, targetRarity)
	if len(possible) == 0 {
		rarityTiers := []string{RarityCommon, RarityRare, RarityEpic, RarityLegendary}
		startIdx := -1
		for i, r := range rarityTiers {
			if r == targetRarity {
				startIdx = i
				break
			}
		}
		if startIdx >= 0 {
			for i := startIdx + 1; i < len(rarityTiers); i++ {
				possible = petsByBiomeAndRarity(biome, rarityTiers[i])
				if len(possible) > 0 {
					break
				}
			}
		}
		if len(possible) == 0 && startIdx > 0 {
			for i := startIdx - 1; i >= 0; i-- {
				possible = petsByBiomeAndRarity(biome, rarityTiers[i])
				if len(possible) > 0 {
					break
				}
			}
		}
	}
	if len(possible) == 0 {
		for name, pt := range PetTypes {
			if pt.Biome == biome {
				possible = append(possible, name)
			}
		}
	}
	if len(possible) == 0 {
		for name := range PetTypes {
			possible = append(possible, name)
		}
	}
	if len(possible) == 0 {
		return "Escargot"
	}
	return possible[rand.Intn(len(possible))]
}

func petsByBiomeAndRarity(biome string, rarity string) []string {
	var out []string
	for name, pt := range PetTypes {
		if pt.Biome == biome && pt.Rarity == rarity {
			out = append(out, name)
		}
	}
	return out
}

// PrehistoricPets are the fossil-themed species reanimated from fossils and
// hatched from fossilized eggs.
var PrehistoricPets = struct {
	Common []string
	Rare   []string
	Epic   []string
}{
	Common: []string{"Trilobite", "Ammonite", "Anomalocaris", "Orthoceras", "Méganeura"},
	Rare:   []string{"Archéoptéryx", "Ptéranodon", "Dimétrodon", "Smilodon", "Mégalocéros", "Doedicurus"},
	Epic:   []string{"Mosasaurus", "Titanoboa", "Phorusrhacos", "Rhinocéros laineux", "Entelodon"},
}

// RollPrehistoric rolls a prehistoric pet across all rarities: mostly common,
// with a good chance of rare and a small chance of epic.
func RollPrehistoric() string {
	r := rand.Float64()
	switch {
	case r < 0.60:
		return PrehistoricPets.Common[rand.Intn(len(PrehistoricPets.Common))]
	case r < 0.90:
		return PrehistoricPets.Rare[rand.Intn(len(PrehistoricPets.Rare))]
	default:
		return PrehistoricPets.Epic[rand.Intn(len(PrehistoricPets.Epic))]
	}
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
