package pets

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/model"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
)

func TestAllPetTypesHaveDamageType(t *testing.T) {
	for name, pt := range PetTypes {
		_, ok := battle.PetDamageType(name)
		assert.True(t, ok, "pet %q (%s) has no battle damage type", name, pt.Rarity)
	}
}

func TestPrehistoricPoolsRegistered(t *testing.T) {
	byRarity := map[string][]string{
		RarityCommon: PrehistoricPets.Common,
		RarityRare:   PrehistoricPets.Rare,
		RarityEpic:   PrehistoricPets.Epic,
	}
	for rarity, names := range byRarity {
		for _, name := range names {
			pt, ok := PetTypes[name]
			require.True(t, ok, "prehistoric pet %q must exist in the registry", name)
			assert.Equal(t, rarity, pt.Rarity, "prehistoric pet %q must be %s", name, rarity)
		}
	}
}

func TestRollPrehistoric(t *testing.T) {
	all := append(append([]string{}, PrehistoricPets.Common...), PrehistoricPets.Rare...)
	all = append(all, PrehistoricPets.Epic...)

	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		name := RollPrehistoric()
		_, ok := PetTypes[name]
		require.True(t, ok, "rolled unknown pet %q", name)
		assert.Contains(t, all, name)
		seen[name] = true
	}

	// Over many rolls every rarity must be reachable.
	for _, names := range [][]string{PrehistoricPets.Common, PrehistoricPets.Rare, PrehistoricPets.Epic} {
		hit := false
		for _, n := range names {
			if seen[n] {
				hit = true
				break
			}
		}
		assert.True(t, hit, "at least one %v-tier prehistoric pet must roll", names)
	}
}

func testService(t *testing.T) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "p.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestCreatePet(t *testing.T) {
	svc, _ := testService(t)
	pet, err := svc.CreatePet(1, "Dragon")
	require.NoError(t, err)
	require.NotNil(t, pet)
	assert.Equal(t, "Dragon", pet.PetType)
	assert.Equal(t, 130, pet.MaxHP)
	assert.Equal(t, 35, pet.Atk)
}

func TestGetPets(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.CreatePet(1, "Escargot")
	require.NoError(t, err)
	_, err = svc.CreatePet(1, "Souris")
	require.NoError(t, err)
	pets, err := svc.GetPets(1)
	require.NoError(t, err)
	assert.Len(t, pets, 2)
}

func TestGetFeedItemDef(t *testing.T) {
	feedable := map[string]string{
		"sardine": "speed", "trout": "speed", "carp": "speed", "salmon": "max_hp",
		"wheat": "acc", "oat": "acc", "corn": "acc",
		"carrot": "defense", "potato": "defense",
		"tomato": "atk", "pumpkin": "atk", "coffee_bean": "atk", "cocoa_bean": "atk",
		"coffee": "speed", "strawberry": "speed", "star_fruit": "speed",
		"golden_apple": "max_hp", "nova_fruit": "max_hp",
		"ghost_wheat": "acc", "prismatic_corn": "acc",
		"golden_potato": "defense", "golden_carrot": "defense",
		"blood_tomato": "atk", "cursed_pumpkin": "atk",
	}
	for id, stat := range feedable {
		def := GetFeedItemDef(id)
		require.NotNil(t, def, "expected %s to be feedable", id)
		assert.Equal(t, stat, def.Stat, "unexpected stat for %s", id)
		assert.Equal(t, float64(1), def.Amount, "unexpected amount for %s", id)
	}
	for _, id := range []string{"pebble", "coal", "iron_ore", "wheat_seed", "bow", "hook"} {
		assert.Nil(t, GetFeedItemDef(id), "expected %s to NOT be feedable", id)
	}
}

func TestGetCraftedFeedItemDef(t *testing.T) {
	crafted := map[string]FeedItemDef{
		"lucky_roast":        {Stat: "crit_c", Amount: 1},
		"thunder_steak":      {Stat: "crit_d", Amount: 0.1},
		"heart_stew":         {Stat: "max_hp", Amount: 2},
		"fatalist_elixir":    {Stat: "crit_c", Amount: 2},
		"ruin_tonic":         {Stat: "crit_d", Amount: 0.2},
		"vitality_elixir":    {Stat: "max_hp", Amount: 4},
		"dragon_chili":       {Stat: "atk", Amount: 1},
		"iron_loaf":          {Stat: "defense", Amount: 1},
		"storm_porridge":     {Stat: "speed", Amount: 1},
		"falcon_pie":         {Stat: "acc", Amount: 1},
		"clover_salad":       {Stat: "crit_c", Amount: 1},
		"volcano_ribs":       {Stat: "crit_d", Amount: 0.1},
		"giant_noodles":      {Stat: "max_hp", Amount: 2},
		"skull_elixir":       {Stat: "atk", Amount: 2},
		"bastion_tonic":      {Stat: "defense", Amount: 2},
		"tempest_draught":    {Stat: "speed", Amount: 2},
		"seer_elixir":        {Stat: "acc", Amount: 2},
		"gamblers_tonic":     {Stat: "crit_c", Amount: 2},
		"annihilator_elixir": {Stat: "crit_d", Amount: 0.2},
		"colossus_draught":   {Stat: "max_hp", Amount: 4},
	}
	for id, want := range crafted {
		def := GetFeedItemDef(id)
		require.NotNil(t, def, "expected %s to be feedable", id)
		assert.Equal(t, want.Stat, def.Stat, "unexpected stat for %s", id)
		assert.Equal(t, want.Amount, def.Amount, "unexpected amount for %s", id)
	}
}

func TestFeedSameFoodTwice(t *testing.T) {
	svc, _ := testService(t)
	pet, err := svc.CreatePet(1, "Escargot")
	require.NoError(t, err)
	baseAtk := pet.Atk

	for i := 0; i < 2; i++ {
		fed, err := svc.FeedPet(pet, GetFeedItemDef("warrior_stew"))
		require.NoError(t, err)
		assert.True(t, fed)
	}
	assert.Equal(t, baseAtk+2, pet.Atk, "the same meal must stack when fed twice")
	assert.Equal(t, 2, pet.FoodEaten)
}

func TestFeedPetStats(t *testing.T) {
	svc, _ := testService(t)
	pet, err := svc.CreatePet(1, "Escargot")
	require.NoError(t, err)
	baseHP := pet.MaxHP
	baseCritC := pet.CritC
	baseCritD := pet.CritD

	fed, err := svc.FeedPet(pet, GetFeedItemDef("lucky_roast"))
	require.NoError(t, err)
	assert.True(t, fed)
	assert.Equal(t, baseCritC+1, pet.CritC)

	fed, err = svc.FeedPet(pet, GetFeedItemDef("thunder_steak"))
	require.NoError(t, err)
	assert.True(t, fed)
	assert.Equal(t, baseCritD+0.1, pet.CritD)

	fed, err = svc.FeedPet(pet, GetFeedItemDef("heart_stew"))
	require.NoError(t, err)
	assert.True(t, fed)
	assert.Equal(t, baseHP+2, pet.MaxHP)
	assert.Equal(t, baseHP+2, pet.HP)

	fed, err = svc.FeedPet(pet, GetFeedItemDef("vitality_elixir"))
	require.NoError(t, err)
	assert.True(t, fed)
	assert.Equal(t, baseHP+6, pet.MaxHP)
	assert.Equal(t, baseHP+6, pet.HP)
}

func TestAddXPLevelUp(t *testing.T) {
	svc, _ := testService(t)
	pet, err := svc.CreatePet(1, "Escargot")
	require.NoError(t, err)
	res := svc.AddXP(pet, 1000)
	assert.True(t, res.Leveled)
	assert.Greater(t, pet.Level, 1)
}

func TestUpdateElo(t *testing.T) {
	svc, _ := testService(t)
	p1, _ := svc.CreatePet(1, "Dragon")
	p2, _ := svc.CreatePet(2, "Souris")
	p1.Level = 10
	p2.Level = 10
	p1.Elo = 1000
	p2.Elo = 1000
	d1, d2 := svc.UpdateElo(p1, p2, 1.0)
	assert.NotEqual(t, 0, d1)
	assert.NotEqual(t, 0, d2)
	assert.Greater(t, p1.Elo, 1000)
	assert.Less(t, p2.Elo, 1000)
}

func TestAddXPAutoStartsArenaRival(t *testing.T) {
	svc, st := testService(t)
	pet, err := svc.CreatePet(1, "Dragon")
	require.NoError(t, err)

	// Tutorial not completed: the quest must not start.
	res := svc.AddXP(pet, 10000)
	require.True(t, res.Leveled)
	require.GreaterOrEqual(t, pet.Level, 5)
	var q model.UserQuest
	err = st.DB.Where("user_id = ? AND quest_id = ?", 1, "arena_rival").First(&q).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "quest must stay locked until the tutorial is completed")

	// Tutorial completed: the next XP gain starts the quest.
	require.NoError(t, st.DB.Create(&model.UserQuest{UserID: 1, QuestID: "tutorial", Status: "COMPLETED"}).Error)
	svc.AddXP(pet, 1)
	require.NoError(t, st.DB.Where("user_id = ? AND quest_id = ?", 1, "arena_rival").First(&q).Error)
	assert.Equal(t, "ACTIVE", q.Status)

	// Further XP gains must not create duplicate quest rows.
	svc.AddXP(pet, 1)
	var count int64
	st.DB.Model(&model.UserQuest{}).Where("user_id = ? AND quest_id = ?", 1, "arena_rival").Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestArtifactActivityAdvancesArenaRival(t *testing.T) {
	svc, st := testService(t)
	_ = questssvc.New(st, &config.Config{}) // wires the quest advance hook

	require.NoError(t, st.CreateQuest(1, "arena_rival"))
	custom, _ := json.Marshal(map[string]any{"target_stat": "artifact_leveled", "target_count": 1})
	require.NoError(t, st.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = ?", 1, "arena_rival").
		Updates(map[string]any{"step_index": 5, "custom_data": string(custom)}).Error)

	// Leveling the artifact must advance the quest to the perk step.
	_, leveled, err := svc.AddArtifactXP(1, 50)
	require.NoError(t, err)
	require.True(t, leveled)
	var uqd model.UserQuestData
	require.NoError(t, st.DB.Where("user_id = ? AND quest_id = ?", 1, "arena_rival").First(&uqd).Error)
	assert.Equal(t, 6, uqd.StepIndex, "artifact level-up must advance to the perk step")

	// Spending the point must advance the quest to the boss step.
	custom2, _ := json.Marshal(map[string]any{"target_stat": "artifact_point_spent", "target_count": 1})
	require.NoError(t, st.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = ?", 1, "arena_rival").
		Updates(map[string]any{"step_index": 6, "progress_value": 0, "custom_data": string(custom2)}).Error)

	_, err = svc.LevelArtifactStat(1, 0)
	require.NoError(t, err)
	require.NoError(t, st.DB.Where("user_id = ? AND quest_id = ?", 1, "arena_rival").First(&uqd).Error)
	assert.Equal(t, 7, uqd.StepIndex, "spending the artifact point must advance to the boss step")
}

func TestRollGacha(t *testing.T) {
	name := RollGacha("", "forest")
	assert.NotEmpty(t, name)
	_, ok := PetTypes[name]
	assert.True(t, ok)
}

func TestRollGachaLegendary(t *testing.T) {
	name := RollGacha(RarityLegendary, "forest")
	pt, ok := PetTypes[name]
	require.True(t, ok)
	assert.Equal(t, RarityLegendary, pt.Rarity)
}

func TestEveryBiomeEggReachesAllRarities(t *testing.T) {
	rarityWeights := map[string]float64{
		RarityCommon:    60,
		RarityRare:      25,
		RarityEpic:      10,
		RarityLegendary: 5,
	}
	for _, biome := range Biomes {
		t.Run(biome, func(t *testing.T) {
			// Every biome must have at least one pet per rarity tier, or eggs
			// could never roll that tier.
			for rarity := range rarityWeights {
				require.NotEmpty(t, petsByBiomeAndRarity(biome, rarity),
					"biome %q has no %s pets — its egg can never roll that tier", biome, rarity)
			}

			// The flattened pool must reproduce the fixed 60/25/10/5 odds
			// regardless of roster composition.
			names, weights, total := eggGachaPool(biome)
			require.Len(t, names, len(weights))
			require.Greater(t, total, 0.0)
			byTier := map[string]float64{}
			for i, name := range names {
				byTier[PetTypes[name].Rarity] += weights[i]
			}
			for rarity, want := range rarityWeights {
				assert.InDelta(t, want, byTier[rarity], 0.001, "biome %q tier %q weight", biome, rarity)
			}

			// Over many rolls every rarity must be reachable.
			seen := map[string]bool{}
			for i := 0; i < 3000; i++ {
				name := RollGacha("", biome)
				pt, ok := PetTypes[name]
				require.True(t, ok, "rolled unknown pet %q", name)
				assert.Equal(t, biome, pt.Biome, "egg from %q must hatch a %q pet", biome, biome)
				seen[pt.Rarity] = true
			}
			for rarity := range rarityWeights {
				assert.True(t, seen[rarity], "biome %q never rolled %s in 3000 tries", biome, rarity)
			}
		})
	}
}

func TestCreatePetAutoActivatesFirst(t *testing.T) {
	svc, _ := testService(t)
	first, err := svc.CreatePet(1, "Dragon")
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.True(t, first.IsActive, "first pet should be auto-activated")

	second, err := svc.CreatePet(1, "Escargot")
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.False(t, second.IsActive, "second pet must not steal activation")

	active, err := svc.GetActivePet(1)
	require.NoError(t, err)
	assert.Equal(t, first.ID, active.ID)
}

func TestSetActivePetExclusive(t *testing.T) {
	svc, _ := testService(t)
	a, err := svc.CreatePet(1, "Dragon")
	require.NoError(t, err)
	b, err := svc.CreatePet(1, "Escargot")
	require.NoError(t, err)
	require.True(t, a.IsActive)
	require.False(t, b.IsActive)

	require.NoError(t, svc.SetActivePet(1, b.ID, 0))

	active, err := svc.GetActivePet(1)
	require.NoError(t, err)
	assert.Equal(t, b.ID, active.ID, "switching activation must be exclusive")

	pets, err := svc.GetPets(1)
	require.NoError(t, err)
	for _, p := range pets {
		if p.ID == b.ID {
			assert.True(t, p.IsActive)
		} else {
			assert.False(t, p.IsActive)
		}
	}
}

func TestHealCost(t *testing.T) {
	assert.Equal(t, 1, HealCost(1, 0))
	assert.Equal(t, 1, HealCost(2, 0))
	assert.Equal(t, 50, HealCost(100, 0))
	assert.Equal(t, 25, HealCost(100, 50))
	assert.Equal(t, 45, HealCost(100, 10))
	assert.Equal(t, 0, HealCost(100, 100), "100% discount must make the heal free")
	assert.Equal(t, 1, HealCost(100, 99))
}

func TestHealPetGuards(t *testing.T) {
	svc, s := testService(t)
	pet, err := svc.CreatePet(1, "Dragon")
	require.NoError(t, err)

	err = svc.HealPet(pet, 10)
	assert.ErrorIs(t, err, ErrPetAlreadyFullHP, "full HP must be rejected")

	pet.HP = 10
	_, err = s.UpdateBalance(1, 100)
	require.NoError(t, err)
	require.NoError(t, svc.UpdatePet(pet))

	require.NoError(t, svc.HealPet(pet, 10))
	assert.Equal(t, pet.MaxHP, pet.HP)

	bal, err := s.GetBalance(1)
	require.NoError(t, err)
	assert.Equal(t, 190, bal, "heal must deduct exactly the paid cost")

	pet.HP = 5
	require.NoError(t, svc.UpdatePet(pet))
	err = svc.HealPet(pet, 100000)
	assert.ErrorIs(t, err, ErrInsufficientFunds)
	assert.Equal(t, 5, pet.HP, "HP must not change on failure")

	pet.HP = 5
	require.NoError(t, svc.UpdatePet(pet))
	require.NoError(t, svc.HealPet(pet, 0), "free heal (100% discount) must work")
	assert.Equal(t, pet.MaxHP, pet.HP)
}
