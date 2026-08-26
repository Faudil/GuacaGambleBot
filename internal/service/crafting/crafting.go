package crafting

import (
	"encoding/json"
	"errors"
	"math"
	"math/rand"

	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
	furnituresvc "guacagamblebot/internal/service/furniture"
	"guacagamblebot/internal/store"
)

var (
	ErrNoRecipe         = errors.New("recipe not found")
	ErrNoLevel          = errors.New("level too low")
	ErrNoIngredients    = errors.New("missing ingredients")
	ErrResearchRequired = errors.New("research required")
)

type Recipe struct {
	Result            string
	Ingredients       map[string]int
	LevelRequired     int
	XP                int
	RequiredResearch  string // primary research gate (rarity or set)
	RequiredResearch2 string // secondary research gate (e.g. set needs both rarity + set research)
	IsEquipment       bool   // if true, creates UserEquipment instance instead of Inventory entry
}

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

// LevelRequired only gates recipes without a research requirement: recipes
// behind RequiredResearch/RequiredResearch2 are unlocked solely by completing
// that research (their LevelRequired is 1 so the level check never blocks
// them). The crafter level still gates the basic, research-free recipes and
// keeps the crafting XP progression meaningful.
var Recipes = map[string]Recipe{
	"beer":                {Result: "beer", Ingredients: map[string]int{"wheat": 3}, LevelRequired: 1, XP: 20},
	"coffee":              {Result: "coffee", Ingredients: map[string]int{"coffee_bean": 3}, LevelRequired: 1, XP: 20},
	"scratch_ticket":      {Result: "scratch_ticket", Ingredients: map[string]int{"coal": 1, "pebble": 1}, LevelRequired: 1, XP: 20},
	"fortune_cookie":      {Result: "fortune_cookie", Ingredients: map[string]int{"wheat": 2, "strawberry": 1}, LevelRequired: 2, XP: 30},
	"fertilizer":          {Result: "fertilizer", Ingredients: map[string]int{"rotten_plant": 3, "coal": 1}, LevelRequired: 1, XP: 15, RequiredResearch: "advanced_botany"},
	"forget_potion":       {Result: "forget_potion", Ingredients: map[string]int{"rotten_plant": 2, "pufferfish": 1}, LevelRequired: 1, XP: 30, RequiredResearch: "scroll_magic"},
	"bow":                 {Result: "bow", Ingredients: map[string]int{"oat": 2, "pebble": 2}, LevelRequired: 1, XP: 25, RequiredResearch: "tool_crafting"},
	"rusty_magnet":        {Result: "rusty_magnet", Ingredients: map[string]int{"iron_ore": 2, "pebble": 4}, LevelRequired: 1, XP: 30, RequiredResearch: "tool_crafting"},
	"hook":                {Result: "hook", Ingredients: map[string]int{"iron_ore": 1, "silver_ore": 1}, LevelRequired: 1, XP: 35, RequiredResearch: "tool_crafting"},
	"identity_scroll":     {Result: "identity_scroll", Ingredients: map[string]int{"rotten_plant": 2, "silver_ore": 1}, LevelRequired: 1, XP: 35, RequiredResearch: "scroll_magic"},
	"magnet":              {Result: "magnet", Ingredients: map[string]int{"iron_ore": 5, "copper_ore": 2, "silver_ore": 1}, LevelRequired: 1, XP: 50, RequiredResearch: "magnetism"},
	"rigged_coin":         {Result: "rigged_coin", Ingredients: map[string]int{"gold_nugget": 1, "pebble": 2, "coal": 1}, LevelRequired: 1, XP: 55, RequiredResearch: "game_theory"},
	"casino_token":        {Result: "casino_token", Ingredients: map[string]int{"gold_nugget": 1, "silver_ore": 1}, LevelRequired: 1, XP: 50, RequiredResearch: "game_theory"},
	"garden_plot":         {Result: "garden_plot", Ingredients: map[string]int{"gold_nugget": 2, "pebble": 20}, LevelRequired: 1, XP: 80, RequiredResearch: "advanced_botany"},
	"electric_magnet":     {Result: "electric_magnet", Ingredients: map[string]int{"platinum": 5, "copper_ore": 10, "gold_nugget": 1}, LevelRequired: 1, XP: 80, RequiredResearch: "magnetism"},
	"tropical_greenhouse": {Result: "tropical_greenhouse", Ingredients: map[string]int{"gold_nugget": 5, "platinum": 2}, LevelRequired: 1, XP: 120, RequiredResearch: "advanced_botany"},
	"vip_ticket":          {Result: "vip_ticket", Ingredients: map[string]int{"rough_diamond": 3, "platinum": 2}, LevelRequired: 1, XP: 150, RequiredResearch: "game_theory"},
	"enchanted_orchard":   {Result: "enchanted_orchard", Ingredients: map[string]int{"rough_diamond": 2, "emerald": 2}, LevelRequired: 1, XP: 250, RequiredResearch: "advanced_botany"},
	"volcano_egg":         {Result: "volcano_egg", Ingredients: map[string]int{"rough_diamond": 1, "golden_apple": 1, "pure_dna": 1, "bone_dust": 10}, LevelRequired: 1, XP: 300, RequiredResearch: "dna_research"},
	"mutagen":             {Result: "mutagen", Ingredients: map[string]int{"mutated_flesh": 3, "silver_ore": 1}, LevelRequired: 1, XP: 30, RequiredResearch: "mutagen_synthesis"},

	// --- Pet food (tier 1) ---
	"warrior_stew":   {Result: "warrior_stew", Ingredients: map[string]int{"sardine": 3, "wheat": 2}, LevelRequired: 1, XP: 30},
	"stonebread":     {Result: "stonebread", Ingredients: map[string]int{"pebble": 3, "potato": 2}, LevelRequired: 1, XP: 30},
	"zephyr_berries": {Result: "zephyr_berries", Ingredients: map[string]int{"trout": 3, "carrot": 2}, LevelRequired: 1, XP: 30},
	"hunters_soup":   {Result: "hunters_soup", Ingredients: map[string]int{"carp": 3, "tomato": 2}, LevelRequired: 1, XP: 30},
	"lucky_roast":    {Result: "lucky_roast", Ingredients: map[string]int{"carp": 3, "pumpkin": 2}, LevelRequired: 1, XP: 30},
	"thunder_steak":  {Result: "thunder_steak", Ingredients: map[string]int{"salmon": 3, "potato": 2}, LevelRequired: 1, XP: 35},
	"heart_stew":     {Result: "heart_stew", Ingredients: map[string]int{"salmon": 2, "oat": 3}, LevelRequired: 1, XP: 30},
	"dragon_chili":   {Result: "dragon_chili", Ingredients: map[string]int{"trout": 3, "corn": 2}, LevelRequired: 1, XP: 35},
	"iron_loaf":      {Result: "iron_loaf", Ingredients: map[string]int{"coal": 2, "oat": 3}, LevelRequired: 1, XP: 30},
	"storm_porridge": {Result: "storm_porridge", Ingredients: map[string]int{"sardine": 3, "coffee_bean": 2}, LevelRequired: 1, XP: 35},
	"falcon_pie":     {Result: "falcon_pie", Ingredients: map[string]int{"wheat": 3, "pumpkin": 2}, LevelRequired: 1, XP: 35},
	"clover_salad":   {Result: "clover_salad", Ingredients: map[string]int{"corn": 3, "cocoa_bean": 1}, LevelRequired: 1, XP: 35},
	"volcano_ribs":   {Result: "volcano_ribs", Ingredients: map[string]int{"coal": 3, "salmon": 2}, LevelRequired: 1, XP: 35},
	"giant_noodles":  {Result: "giant_noodles", Ingredients: map[string]int{"wheat": 3, "trout": 2}, LevelRequired: 1, XP: 30},

	// --- Pet potions (tier 2, pet_nutrition research) ---
	"berserker_elixir":   {Result: "berserker_elixir", Ingredients: map[string]int{"shark": 2, "golden_apple": 1, "emerald": 1}, LevelRequired: 3, XP: 150, RequiredResearch: "pet_nutrition"},
	"adamant_tonic":      {Result: "adamant_tonic", Ingredients: map[string]int{"iron_ore": 5, "star_fruit": 1, "silver_ore": 3}, LevelRequired: 3, XP: 150, RequiredResearch: "pet_nutrition"},
	"gale_draught":       {Result: "gale_draught", Ingredients: map[string]int{"swordfish": 2, "nova_fruit": 1, "gold_nugget": 3}, LevelRequired: 3, XP: 150, RequiredResearch: "pet_nutrition"},
	"oracles_insight":    {Result: "oracles_insight", Ingredients: map[string]int{"pufferfish": 2, "blood_tomato": 1, "rough_diamond": 1}, LevelRequired: 3, XP: 150, RequiredResearch: "pet_nutrition"},
	"fatalist_elixir":    {Result: "fatalist_elixir", Ingredients: map[string]int{"swordfish": 2, "ghost_wheat": 1, "emerald": 1}, LevelRequired: 3, XP: 150, RequiredResearch: "pet_nutrition"},
	"ruin_tonic":         {Result: "ruin_tonic", Ingredients: map[string]int{"magma_carp": 2, "golden_carrot": 1, "platinum": 2}, LevelRequired: 3, XP: 150, RequiredResearch: "pet_nutrition"},
	"vitality_elixir":    {Result: "vitality_elixir", Ingredients: map[string]int{"whale": 1, "star_fruit": 2, "gold_nugget": 3}, LevelRequired: 3, XP: 150, RequiredResearch: "pet_nutrition"},
	"skull_elixir":       {Result: "skull_elixir", Ingredients: map[string]int{"lava_serpent": 2, "cursed_pumpkin": 1, "rough_diamond": 1}, LevelRequired: 3, XP: 150, RequiredResearch: "pet_nutrition"},
	"bastion_tonic":      {Result: "bastion_tonic", Ingredients: map[string]int{"iron_ore": 5, "golden_potato": 1, "platinum": 2}, LevelRequired: 3, XP: 150, RequiredResearch: "pet_nutrition"},
	"tempest_draught":    {Result: "tempest_draught", Ingredients: map[string]int{"shark": 2, "prismatic_corn": 1, "gold_nugget": 3}, LevelRequired: 3, XP: 150, RequiredResearch: "pet_nutrition"},
	"seer_elixir":        {Result: "seer_elixir", Ingredients: map[string]int{"pufferfish": 2, "ghost_wheat": 1, "silver_ore": 3}, LevelRequired: 3, XP: 150, RequiredResearch: "pet_nutrition"},
	"gamblers_tonic":     {Result: "gamblers_tonic", Ingredients: map[string]int{"gold_nugget": 3, "golden_apple": 1, "emerald": 1}, LevelRequired: 3, XP: 150, RequiredResearch: "pet_nutrition"},
	"annihilator_elixir": {Result: "annihilator_elixir", Ingredients: map[string]int{"magma_carp": 2, "blood_tomato": 1, "platinum": 2}, LevelRequired: 3, XP: 150, RequiredResearch: "pet_nutrition"},
	"colossus_draught":   {Result: "colossus_draught", Ingredients: map[string]int{"whale": 2, "golden_apple": 2, "gold_nugget": 3}, LevelRequired: 3, XP: 150, RequiredResearch: "pet_nutrition"},

	// --- Common equipment (equip_common) ---
	"craft_stick":         {Result: "stick", Ingredients: map[string]int{"wheat": 2, "pebble": 1}, LevelRequired: 1, XP: 15, RequiredResearch: "equip_common", IsEquipment: true},
	"craft_leather_armor": {Result: "leather_armor", Ingredients: map[string]int{"iron_ore": 3, "coal": 2}, LevelRequired: 1, XP: 30, RequiredResearch: "equip_common", IsEquipment: true},

	// --- Uncommon equipment (equip_uncommon) ---
	"craft_iron_mace":    {Result: "iron_mace", Ingredients: map[string]int{"iron_ore": 5, "coal": 3}, LevelRequired: 1, XP: 30, RequiredResearch: "equip_uncommon", IsEquipment: true},
	"craft_lucky_charm":  {Result: "lucky_charm", Ingredients: map[string]int{"gold_nugget": 1, "emerald": 1}, LevelRequired: 1, XP: 30, RequiredResearch: "equip_uncommon", IsEquipment: true},
	"craft_fishing_rod":  {Result: "fishing_rod", Ingredients: map[string]int{"wheat": 5, "iron_ore": 3}, LevelRequired: 1, XP: 35, RequiredResearch: "equip_uncommon", IsEquipment: true},
	"craft_miner_helmet": {Result: "miner_helmet", Ingredients: map[string]int{"iron_ore": 5, "coal": 5}, LevelRequired: 1, XP: 35, RequiredResearch: "equip_uncommon", IsEquipment: true},

	// --- Rare equipment (equip_rare) ---
	"craft_hunters_bow":   {Result: "hunters_bow", Ingredients: map[string]int{"iron_ore": 8, "silver_ore": 3}, LevelRequired: 1, XP: 50, RequiredResearch: "equip_rare", IsEquipment: true},
	"craft_hunter_cloak":  {Result: "hunter_cloak", Ingredients: map[string]int{"coal": 5, "silver_ore": 5}, LevelRequired: 1, XP: 50, RequiredResearch: "equip_rare", IsEquipment: true},
	"craft_golden_ring":   {Result: "golden_ring", Ingredients: map[string]int{"gold_nugget": 3, "emerald": 2}, LevelRequired: 1, XP: 60, RequiredResearch: "equip_rare", IsEquipment: true},
	"craft_crystal_staff": {Result: "crystal_staff", Ingredients: map[string]int{"gold_nugget": 3, "rough_diamond": 2}, LevelRequired: 1, XP: 60, RequiredResearch: "equip_rare", IsEquipment: true},

	// --- Epic equipment (equip_epic) ---
	"craft_enchanted_robe": {Result: "enchanted_robe", Ingredients: map[string]int{"platinum": 3, "rough_diamond": 2}, LevelRequired: 1, XP: 80, RequiredResearch: "equip_epic", IsEquipment: true},
	"craft_ancient_amulet": {Result: "ancient_amulet", Ingredients: map[string]int{"platinum": 5, "emerald": 3}, LevelRequired: 1, XP: 90, RequiredResearch: "equip_epic", IsEquipment: true},

	// --- Dragon Slayer Set 🐉 (forge, equip_rare) — mining + boss resources ---
	"craft_dragon_slayer_sword":    {Result: "dragon_slayer_sword", Ingredients: map[string]int{"rough_diamond": 5, "ancient_alloy": 5, "platinum": 10, "boss_trophy": 1}, LevelRequired: 10, XP: 200, RequiredResearch: "set_dragon_slayer", RequiredResearch2: "equip_rare", IsEquipment: true},
	"craft_dragon_slayer_armor":    {Result: "dragon_slayer_armor", Ingredients: map[string]int{"rough_diamond": 5, "ancient_alloy": 5, "platinum": 10, "boss_trophy": 1}, LevelRequired: 10, XP: 200, RequiredResearch: "set_dragon_slayer", RequiredResearch2: "equip_rare", IsEquipment: true},
	"craft_dragon_slayer_ring":     {Result: "dragon_slayer_ring", Ingredients: map[string]int{"rough_diamond": 5, "ancient_alloy": 5, "platinum": 10, "boss_trophy": 1}, LevelRequired: 10, XP: 200, RequiredResearch: "set_dragon_slayer", RequiredResearch2: "equip_rare", IsEquipment: true},
	"craft_dragon_slayer_talisman": {Result: "dragon_slayer_talisman", Ingredients: map[string]int{"rough_diamond": 5, "ancient_alloy": 5, "platinum": 10, "boss_trophy": 1}, LevelRequired: 10, XP: 200, RequiredResearch: "set_dragon_slayer", RequiredResearch2: "equip_rare", IsEquipment: true},

	// --- Shadow Stalker Set 🌑 (forge, equip_rare) — archeology resources ---
	"craft_shadow_stalker_blade":  {Result: "shadow_stalker_blade", Ingredients: map[string]int{"shadow_fossil": 3, "cursed_artifact": 5, "purified_relic": 3, "legendary_fragment": 3}, LevelRequired: 10, XP: 200, RequiredResearch: "set_shadow_stalker", RequiredResearch2: "equip_rare", IsEquipment: true},
	"craft_shadow_stalker_cloak":  {Result: "shadow_stalker_cloak", Ingredients: map[string]int{"shadow_fossil": 3, "cursed_artifact": 5, "purified_relic": 3, "legendary_fragment": 3}, LevelRequired: 10, XP: 200, RequiredResearch: "set_shadow_stalker", RequiredResearch2: "equip_rare", IsEquipment: true},
	"craft_shadow_stalker_amulet": {Result: "shadow_stalker_amulet", Ingredients: map[string]int{"shadow_fossil": 3, "cursed_artifact": 5, "purified_relic": 3, "legendary_fragment": 3}, LevelRequired: 10, XP: 200, RequiredResearch: "set_shadow_stalker", RequiredResearch2: "equip_rare", IsEquipment: true},
	"craft_shadow_stalker_charm":  {Result: "shadow_stalker_charm", Ingredients: map[string]int{"shadow_fossil": 3, "cursed_artifact": 5, "purified_relic": 3, "legendary_fragment": 3}, LevelRequired: 10, XP: 200, RequiredResearch: "set_shadow_stalker", RequiredResearch2: "equip_rare", IsEquipment: true},

	// --- Arcane Weaver Set 🔮 (arcane_forge, equip_epic) — dimensional + exotic resources ---
	"craft_arcane_weaver_staff": {Result: "arcane_weaver_staff", Ingredients: map[string]int{"dimensional_shard": 5, "pure_dna": 3, "nova_fruit": 3, "lava_serpent": 5}, LevelRequired: 10, XP: 300, RequiredResearch: "set_arcane_weaver", RequiredResearch2: "equip_epic", IsEquipment: true},
	"craft_arcane_weaver_robe":  {Result: "arcane_weaver_robe", Ingredients: map[string]int{"dimensional_shard": 5, "pure_dna": 3, "nova_fruit": 3, "lava_serpent": 5}, LevelRequired: 10, XP: 300, RequiredResearch: "set_arcane_weaver", RequiredResearch2: "equip_epic", IsEquipment: true},
	"craft_arcane_weaver_crown": {Result: "arcane_weaver_crown", Ingredients: map[string]int{"dimensional_shard": 5, "pure_dna": 3, "nova_fruit": 3, "lava_serpent": 5}, LevelRequired: 10, XP: 300, RequiredResearch: "set_arcane_weaver", RequiredResearch2: "equip_epic", IsEquipment: true},
	"craft_arcane_weaver_orb":   {Result: "arcane_weaver_orb", Ingredients: map[string]int{"dimensional_shard": 5, "pure_dna": 3, "nova_fruit": 3, "lava_serpent": 5}, LevelRequired: 10, XP: 300, RequiredResearch: "set_arcane_weaver", RequiredResearch2: "equip_epic", IsEquipment: true},
}

func (s *Service) GetCrafterLevel(userID int64) int {
	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "crafter").First(&job).Error; err != nil {
		return 1
	}
	return job.Level
}

func (s *Service) Craft(userID int64, recipeKey string, amount int) (bool, int, error) {
	recipe, ok := Recipes[recipeKey]
	if !ok {
		return false, 0, ErrNoRecipe
	}
	level := s.GetCrafterLevel(userID)
	if level < recipe.LevelRequired {
		return false, 0, ErrNoLevel
	}
	if !s.isResearchCompleted(userID, recipe.RequiredResearch) {
		return false, 0, ErrResearchRequired
	}
	if !s.isResearchCompleted(userID, recipe.RequiredResearch2) {
		return false, 0, ErrResearchRequired
	}

	intMult := charsvc.GetINTBonus(s.store, userID)
	charXP := int(float64(recipe.XP*amount) * intMult)
	totalXP := recipe.XP * amount

	effectiveAmount := amount
	ingMultiplier := 1.0
	if charsvc.HasBuff(s.store, userID, "efficiency") {
		ingMultiplier = 0.5
		charsvc.ConsumeBuff(s.store, userID, "efficiency")
	}

	// A Workbench placed in the active house cuts ingredient costs.
	ingMultiplier *= 1 - furnituresvc.EffectValue(s.store, userID, "craft_cost")

	if charsvc.HasBuff(s.store, userID, "perfect_forge") {
		charsvc.ConsumeBuff(s.store, userID, "perfect_forge")
		effectiveAmount = amount * 2
	}

	free, err := s.store.FreeSlots(s.store.DB, userID)
	if err != nil {
		return false, 0, err
	}
	if free < effectiveAmount {
		return false, 0, store.ErrInventoryFull
	}

	if err := s.store.DB.Transaction(func(tx *gorm.DB) error {
		for ing, qty := range recipe.Ingredients {
			req := max(1, int(float64(qty*amount)*ingMultiplier))
			var inv model.Inventory
			if err := tx.Where("user_id = ? AND item_id = ? AND quantity >= ?", userID, ing, req).First(&inv).Error; err != nil {
				return ErrNoIngredients
			}
		}
		for ing, qty := range recipe.Ingredients {
			req := max(1, int(float64(qty*amount)*ingMultiplier))
			if err := tx.Model(&model.Inventory{}).
				Where("user_id = ? AND item_id = ?", userID, ing).
				UpdateColumn("quantity", gorm.Expr("quantity - ?", req)).Error; err != nil {
				return err
			}
		}

		if recipe.IsEquipment {
			// Create UserEquipment instances for each crafted piece
			base := items.Get(recipe.Result)
			if base == nil {
				return ErrNoRecipe
			}
			for i := 0; i < effectiveAmount; i++ {
				rar := upgradedRarity(base.Rarity,
					furnituresvc.EffectValue(s.store, userID, "equip_quality"),
					furnituresvc.EffectValue(s.store, userID, "equip_legendary"))
				affixes := items.RollAffixes(rar, base.EquipSlot)
				var applied []items.AppliedAffix
				for _, a := range affixes {
					applied = append(applied, items.AppliedAffix{
						ID:    a.ID,
						Name:  a.Name,
						Stat:  a.Stat,
						Value: items.RollAffixValue(a),
					})
				}
				totalSTR := base.StatSTR
				totalDEX := base.StatDEX
				totalINT := base.StatINT
				totalVIT := base.StatVIT
				totalLUK := base.StatLUK
				for _, a := range applied {
					switch a.Stat {
					case "str":
						totalSTR += a.Value
					case "dex":
						totalDEX += a.Value
					case "int":
						totalINT += a.Value
					case "vit":
						totalVIT += a.Value
					case "luk":
						totalLUK += a.Value
					}
				}
				affixData, _ := json.Marshal(applied)
				_, err := s.store.CreateEquipmentTx(tx, userID, base.ID, base.Name, base.Emoji,
					string(rar), base.EquipSlot, base.MinLevel,
					totalSTR, totalDEX, totalINT, totalVIT, totalLUK,
					affixData, base.SetID)
				if err != nil {
					return err
				}
			}
		} else {
			// Standard item: add to inventory
			if err := s.store.AddItemRaw(tx, userID, recipe.Result, effectiveAmount); err != nil {
				return err
			}
		}

		tx.Where("user_id = ? AND job_name = ?", userID, "crafter").
			FirstOrCreate(&model.Job{UserID: userID, JobName: "crafter", Level: 1, XP: 0})
		if err := tx.Model(&model.Job{}).
			Where("user_id = ? AND job_name = ?", userID, "crafter").
			UpdateColumn("xp", gorm.Expr("xp + ?", totalXP)).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false, 0, err
	}

	leveled, lvl := charsvc.AddXP(s.store, userID, charXP)
	// Craft activity stat, used by procedural daily quests ("craft N items").
	_ = s.store.RecordActivity(userID, "items_crafted", amount)
	return leveled, lvl, nil
}

func (s *Service) LevelUpCheck(userID int64) (bool, int) {
	var job model.Job
	if err := s.store.DB.Where("user_id = ? AND job_name = ?", userID, "crafter").First(&job).Error; err != nil {
		return false, 1
	}
	xpNeeded := job.Level * 100
	if job.XP >= xpNeeded {
		newLevel := job.Level + 1
		newXP := job.XP - xpNeeded
		s.store.DB.Model(&job).Where("user_id = ? AND job_name = ?", userID, "crafter").
			Updates(map[string]any{"level": newLevel, "xp": newXP})
		return true, newLevel
	}
	return false, job.Level
}

func (s *Service) isResearchCompleted(userID int64, researchID string) bool {
	if researchID == "" {
		return true
	}
	var r model.UserResearch
	err := s.store.DB.Where("user_id = ? AND research_id = ? AND completed = ?", userID, researchID, true).First(&r).Error
	return err == nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func floor(f float64) int {
	return int(math.Floor(f))
}

var rarityTiers = []items.Rarity{
	items.RarityCommon,
	items.RarityUncommon,
	items.RarityRare,
	items.RarityEpic,
	items.RarityLegendary,
}

// upgradedRarity rolls whether a crafted piece comes out one tier higher than
// its base rarity. qualityChance applies at every tier; legendaryChance adds an
// extra chance specifically for the epic → legendary upgrade (Arcane Forge).
func upgradedRarity(base items.Rarity, qualityChance, legendaryChance float64) items.Rarity {
	chance := qualityChance
	if base == items.RarityEpic {
		chance += legendaryChance
	}
	if chance <= 0 {
		return base
	}
	idx := -1
	for i, r := range rarityTiers {
		if r == base {
			idx = i
			break
		}
	}
	if idx < 0 || idx >= len(rarityTiers)-1 {
		return base
	}
	if rand.Float64() < chance {
		return rarityTiers[idx+1]
	}
	return base
}
