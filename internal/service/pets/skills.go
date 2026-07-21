package pets

import (
	"math/rand"

	"guacagamblebot/internal/battle"
)

type PetSkill struct {
	ID          string
	Name        string
	Description string
	Emoji       string
	MinRarity   string // minimum rarity to be offered
	Category    string // "battle" or "utility"
	BattleApply func(owner, opponent *battle.BattlePet)
}

var AllPetSkills = map[string]*PetSkill{
	// ─── Battle Skills (Common) ────────────────────────────
	"iron_shell": {
		ID: "iron_shell", Name: "Iron Shell",
		Description: "+20% Defense in battle.",
		Emoji: "🛡️", MinRarity: RarityCommon, Category: "battle",
		BattleApply: func(owner, _ *battle.BattlePet) {
			owner.Defense = int(float64(owner.Defense) * 1.20)
		},
	},
	"keen_edge": {
		ID: "keen_edge", Name: "Keen Edge",
		Description: "+15% Attack in battle.",
		Emoji: "⚔️", MinRarity: RarityCommon, Category: "battle",
		BattleApply: func(owner, _ *battle.BattlePet) {
			owner.Atk = int(float64(owner.Atk) * 1.15)
		},
	},
	"evasive": {
		ID: "evasive", Name: "Evasive",
		Description: "+20% Dodge in battle.",
		Emoji: "💨", MinRarity: RarityCommon, Category: "battle",
		BattleApply: func(owner, _ *battle.BattlePet) {
			owner.DGE = int(float64(owner.DGE) * 1.20)
		},
	},

	// ─── Battle Skills (Rare) ──────────────────────────────
	"last_stand": {
		ID: "last_stand", Name: "Last Stand",
		Description: "Deals 2x damage when below 25% HP.",
		Emoji: "🔥", MinRarity: RarityRare, Category: "battle",
		BattleApply: func(owner, _ *battle.BattlePet) {
			owner.PerkInt["last_stand"] = 1
		},
	},
	"regeneration": {
		ID: "regeneration", Name: "Regeneration",
		Description: "Heals 5% Max HP every 3 turns.",
		Emoji: "💚", MinRarity: RarityRare, Category: "battle",
		BattleApply: func(owner, _ *battle.BattlePet) {
			owner.PerkInt["regeneration"] = 0 // turn counter
		},
	},
	"counter": {
		ID: "counter", Name: "Counter",
		Description: "30% chance to reflect 50% of incoming damage.",
		Emoji: "🔄", MinRarity: RarityRare, Category: "battle",
		BattleApply: func(owner, _ *battle.BattlePet) {
			owner.PerkInt["counter"] = 1
		},
	},

	// ─── Battle Skills (Epic) ──────────────────────────────
	"piercing_strike": {
		ID: "piercing_strike", Name: "Piercing Strike",
		Description: "Attacks ignore 40% of target's Defense.",
		Emoji: "🗡️", MinRarity: RarityEpic, Category: "battle",
		BattleApply: func(owner, _ *battle.BattlePet) {
			owner.PerkInt["piercing"] = 1
		},
	},
	"berserker": {
		ID: "berserker", Name: "Berserker",
		Description: "+3% crit chance per 10% HP lost.",
		Emoji: "😤", MinRarity: RarityEpic, Category: "battle",
		BattleApply: func(owner, _ *battle.BattlePet) {
			owner.PerkInt["berserker"] = 1
		},
	},
	"thornmail": {
		ID: "thornmail", Name: "Thornmail",
		Description: "Always reflects thorn damage (no probability check).",
		Emoji: "🌵", MinRarity: RarityEpic, Category: "battle",
		BattleApply: func(owner, _ *battle.BattlePet) {
			owner.PerkInt["thornmail"] = 1
		},
	},

	// ─── Battle Skills (Legendary) ─────────────────────────
	"phoenix_rebirth": {
		ID: "phoenix_rebirth", Name: "Phoenix Rebirth",
		Description: "Once per battle, revive with 30% HP when KO'd.",
		Emoji: "🦅", MinRarity: RarityLegendary, Category: "battle",
		BattleApply: func(owner, _ *battle.BattlePet) {
			owner.PerkInt["rebirth"] = 1
		},
	},
	"dragon_fury": {
		ID: "dragon_fury", Name: "Dragon's Fury",
		Description: "First attack deals 2x damage.",
		Emoji: "🐉", MinRarity: RarityLegendary, Category: "battle",
		BattleApply: func(owner, _ *battle.BattlePet) {
			owner.PerkInt["dragon_fury"] = 1
		},
	},

	// ─── Utility Skills ────────────────────────────────────
	"keen_senses": {
		ID: "keen_senses", Name: "Keen Senses",
		Description: "+25% loot from hunting.",
		Emoji: "👁️", MinRarity: RarityCommon, Category: "utility",
	},
	"prospector": {
		ID: "prospector", Name: "Prospector",
		Description: "+25% yield from mining.",
		Emoji: "⛏️", MinRarity: RarityCommon, Category: "utility",
	},
	"green_thumb": {
		ID: "green_thumb", Name: "Green Thumb",
		Description: "Farming grows 20% faster.",
		Emoji: "🌱", MinRarity: RarityCommon, Category: "utility",
	},
	"angler": {
		ID: "angler", Name: "Angler",
		Description: "+25% fish quality/quantity.",
		Emoji: "🎣", MinRarity: RarityCommon, Category: "utility",
	},
	"scavenger": {
		ID: "scavenger", Name: "Scavenger",
		Description: "+25% expedition rewards.",
		Emoji: "🎒", MinRarity: RarityRare, Category: "utility",
	},
	"treasure_hunter": {
		ID: "treasure_hunter", Name: "Treasure Hunter",
		Description: "+5% rare discovery chance.",
		Emoji: "💎", MinRarity: RarityEpic, Category: "utility",
	},
	"mentor": {
		ID: "mentor", Name: "Mentor",
		Description: "Active pet gains +10% XP from all sources.",
		Emoji: "📚", MinRarity: RarityRare, Category: "utility",
	},
}

func SkillsByRarity(rarity string) []*PetSkill {
	var pool []*PetSkill
	for _, sk := range AllPetSkills {
		if rarityOrder[sk.MinRarity] <= rarityOrder[rarity] {
			pool = append(pool, sk)
		}
	}
	return pool
}

func RandomSkillOptions(rarity string, count int) []*PetSkill {
	pool := SkillsByRarity(rarity)
	if len(pool) <= count {
		return pool
	}
	// Shuffle and pick
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	return pool[:count]
}

func RandomBattleSkills(rarity string, count int) []*PetSkill {
	var pool []*PetSkill
	for _, sk := range AllPetSkills {
		if sk.Category == "battle" && rarityOrder[sk.MinRarity] <= rarityOrder[rarity] {
			pool = append(pool, sk)
		}
	}
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	if len(pool) > count {
		return pool[:count]
	}
	return pool
}

func RandomUtilitySkills(rarity string, count int) []*PetSkill {
	var pool []*PetSkill
	for _, sk := range AllPetSkills {
		if sk.Category == "utility" && rarityOrder[sk.MinRarity] <= rarityOrder[rarity] {
			pool = append(pool, sk)
		}
	}
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	if len(pool) > count {
		return pool[:count]
	}
	return pool
}
