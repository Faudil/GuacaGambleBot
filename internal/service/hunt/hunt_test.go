package hunt

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	invsvc "guacagamblebot/internal/service/inventory"
	jobssvc "guacagamblebot/internal/service/jobs"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
	"guacagamblebot/internal/universe"
	"guacagamblebot/internal/universe/hoakhaven"
)

func testService(t *testing.T) (*Service, *store.Store) {
	return testServiceWithCfg(t, &config.Config{StartingBalance: 100})
}

func testServiceWithCfg(t *testing.T, cfg *config.Config) (*Service, *store.Store) {
	d := testutil.NewDB(t)
	if cfg == nil {
		cfg = &config.Config{StartingBalance: 100}
	}
	s := store.New(d, cfg)
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	require.NotNil(t, def)
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)
	svc := New(s, cfg, npcSvc)
	return svc, s
}

func addActivePet(t *testing.T, s *store.Store, userID int64) {
	t.Helper()
	_ = s.DB.Create(&model.UserPet{
		UserID:   userID,
		PetType:  "Chien",
		Nickname: "Buddy",
		Level:    5,
		XP:       0,
		MaxHP:    100,
		HP:       100,
		Atk:      15,
		Defense:  8,
		Speed:    10,
		IsActive: true,
	})
}

func TestExecuteHuntNoPet(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.ExecuteHunt(1, "forest")
	assert.ErrorIs(t, err, ErrNoPet)
}

func TestExecuteHuntEasy(t *testing.T) {
	svc, s := testService(t)
	addActivePet(t, s, 1)
	res, err := svc.ExecuteHunt(1, "forest")
	require.NoError(t, err)
	assert.True(t, res.PlayerWon || res.EnemyWon)
	assert.NotEmpty(t, res.Turns)
	assert.Greater(t, res.XP, 0)
}

func TestExecuteHuntInvalidZone(t *testing.T) {
	svc, s := testService(t)
	addActivePet(t, s, 1)
	_, err := svc.ExecuteHunt(1, "nonexistent")
	assert.Error(t, err)
}

func TestHuntThornmail(t *testing.T) {
	svc, s := testService(t)
	pet := &model.UserPet{
		UserID:   1,
		PetType:  "Chien",
		Nickname: "Prickly",
		Level:    5,
		MaxHP:    200,
		HP:       200,
		Atk:      8,
		Defense:  50,
		Speed:    2,
		IsActive: true,
	}
	require.NoError(t, s.DB.Create(pet).Error)
	require.NoError(t, s.DB.Create(&model.UserPetSkill{PetID: pet.ID, Slot: 1, SkillID: "thornmail"}).Error)

	res, err := svc.ExecuteHunt(1, "forest")
	require.NoError(t, err)
	require.NotEmpty(t, res.Turns)

	msgs := make([]string, 0, len(res.Turns))
	for _, turn := range res.Turns {
		msgs = append(msgs, turn.Msg)
	}
	assert.Contains(t, strings.Join(msgs, "\n"), "thorn damage", "thornmail must reflect damage back at the enemy")
}

func TestHuntGrantsHunterJobXP(t *testing.T) {
	svc, s := testService(t)
	addActivePet(t, s, 1)
	res, err := svc.ExecuteHunt(1, "forest")
	require.NoError(t, err)

	var job model.Job
	require.NoError(t, s.DB.Where("user_id = ? AND job_name = ?", 1, "hunter").First(&job).Error,
		"a hunt must create the hunter job row")
	assert.GreaterOrEqual(t, job.Level, 1)
	total := job.XP
	for lvl := 1; lvl < job.Level; lvl++ {
		total += jobssvc.XPForLevel(lvl)
	}
	assert.Equal(t, res.XP, total, "hunter job XP must match the hunt's activity XP")
}

func TestHuntGrantsHunterJobXPMultiple(t *testing.T) {
	svc, s := testServiceWithCfg(t, &config.Config{
		StartingBalance:     100,
		HuntMaxPerDay:       100,
		HuntCooldownSeconds: 0,
	})
	addActivePet(t, s, 1)
	_ = s.DB.Create(&model.Job{UserID: 1, JobName: "hunter", Level: 2, XP: 10})

	_, err := svc.ExecuteHunt(1, "forest")
	require.NoError(t, err)

	var job model.Job
	require.NoError(t, s.DB.Where("user_id = ? AND job_name = ?", 1, "hunter").First(&job).Error)
	assert.GreaterOrEqual(t, job.Level, 2, "existing hunter job must keep its level")
	// The leftover XP may be 0 when the hunt lands exactly on a level-up
	// boundary, so compare total invested XP instead.
	totalXP := job.XP
	for l := 2; l < job.Level; l++ {
		totalXP += jobssvc.XPForLevel(l)
	}
	assert.Greater(t, totalXP, 10, "existing hunter job must gain XP")
}

func TestNewEnemy(t *testing.T) {
	e := NewEnemy("forest")
	assert.NotEmpty(t, e.Nickname)
	assert.Greater(t, e.HP, 0)
	assert.Greater(t, e.Level, 0)
}

func TestBattleLogTracksHP(t *testing.T) {
	svc, s := testService(t)
	addActivePet(t, s, 1)
	res, err := svc.ExecuteHunt(1, "forest")
	require.NoError(t, err)
	require.NotEmpty(t, res.Turns)

	assert.NotEmpty(t, res.EnemyName)
	assert.NotEmpty(t, res.EnemyEmoji)
	assert.Greater(t, res.EnemyLevel, 0)
	assert.LessOrEqual(t, res.Turns[0].Pet1HP, res.PetStartHP, "first entry must not exceed the starting pet HP")

	prevPetHP, prevEnemyHP := res.PetStartHP, res.EnemyMaxHP
	for _, turn := range res.Turns {
		assert.LessOrEqual(t, turn.Pet1HP, prevPetHP, "pet HP must never increase")
		assert.LessOrEqual(t, turn.Pet2HP, prevEnemyHP, "enemy HP must never increase")
		assert.GreaterOrEqual(t, turn.Pet1HP, 0)
		assert.GreaterOrEqual(t, turn.Pet2HP, 0)
		prevPetHP, prevEnemyHP = turn.Pet1HP, turn.Pet2HP
	}

	last := res.Turns[len(res.Turns)-1]
	assert.Equal(t, res.PetHP, last.Pet1HP, "last entry must record the final pet HP")
	assert.Equal(t, res.EnemyHP, last.Pet2HP, "last entry must record the final enemy HP")
	if res.PlayerWon {
		assert.Equal(t, 0, last.Pet2HP)
	} else if res.EnemyWon {
		assert.Equal(t, 0, last.Pet1HP)
	}
}

func TestExecuteHuntDailyLimit(t *testing.T) {
	svc, s := testServiceWithCfg(t, &config.Config{
		StartingBalance:     100,
		HuntMaxPerDay:       3,
		HuntCooldownSeconds: 0,
	})
	addActivePet(t, s, 1)

	for i := 0; i < 3; i++ {
		// Heal the pet between hunts so the daily limit is the only constraint.
		_ = s.DB.Model(&model.UserPet{}).Where("user_id = ?", int64(1)).Update("hp", 100000).Error
		_, err := svc.ExecuteHunt(1, "forest")
		require.NoError(t, err, "hunt %d should succeed within the daily limit", i+1)
	}
	_, err := svc.ExecuteHunt(1, "forest")
	assert.ErrorIs(t, err, ErrHuntLimit, "4th hunt must be rejected by the daily limit")
}

func TestExecuteHuntCooldown(t *testing.T) {
	svc, s := testServiceWithCfg(t, &config.Config{
		StartingBalance:     100,
		HuntMaxPerDay:       100,
		HuntCooldownSeconds: 600,
	})
	addActivePet(t, s, 1)

	_, err := svc.ExecuteHunt(1, "forest")
	require.NoError(t, err)
	_, err = svc.ExecuteHunt(1, "forest")
	assert.ErrorIs(t, err, ErrHuntCooldown, "immediate second hunt must be blocked by the cooldown")
}

func TestHasZoneAccess(t *testing.T) {
	svc, _ := testService(t)
	for _, zone := range []string{"forest", "cave", "desert"} {
		ok, err := svc.HasZoneAccess(1, zone)
		require.NoError(t, err)
		assert.True(t, ok, "first zones must be accessible without unlocking: %s", zone)
	}
	ok, err := svc.HasZoneAccess(1, "mountain")
	require.NoError(t, err)
	assert.False(t, ok, "mountain must be locked before the desert requirement is met")

	require.NoError(t, svc.store.UnlockZone(1, "mountain"))
	ok, err = svc.HasZoneAccess(1, "mountain")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestRecordHuntWinUnlocksNextZone(t *testing.T) {
	svc, _ := testService(t)

	for i := 0; i < 2; i++ {
		unlocked, err := svc.RecordHuntWin(1, "desert")
		require.NoError(t, err)
		assert.Empty(t, unlocked, "mountain must not unlock before 3 desert wins (win %d)", i+1)
	}
	unlocked, err := svc.RecordHuntWin(1, "desert")
	require.NoError(t, err)
	assert.Equal(t, "mountain", unlocked, "3rd desert win must unlock mountain")

	ok, err := svc.HasZoneAccess(1, "mountain")
	require.NoError(t, err)
	assert.True(t, ok)

	unlocked, err = svc.RecordHuntWin(1, "desert")
	require.NoError(t, err)
	assert.Empty(t, unlocked, "extra desert wins must not re-unlock mountain")

	for i := 0; i < 2; i++ {
		_, err := svc.RecordHuntWin(1, "mountain")
		require.NoError(t, err)
	}
	unlocked, err = svc.RecordHuntWin(1, "mountain")
	require.NoError(t, err)
	assert.Equal(t, "ocean", unlocked, "3rd mountain win must unlock ocean")

	ok, err = svc.HasZoneAccess(1, "ocean")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestExecuteHuntLockedZone(t *testing.T) {
	svc, s := testService(t)
	addActivePet(t, s, 1)
	_, err := svc.ExecuteHunt(1, "mountain")
	assert.ErrorIs(t, err, ErrZoneLocked)
}

func TestZonesHaveBosses(t *testing.T) {
	for key, zone := range Zones {
		assert.NotEmpty(t, zone.Boss.Name, "zone %s must define a boss", key)
		assert.NotEmpty(t, zone.Boss.Emoji, "zone %s boss must have an emoji", key)
		assert.Greater(t, zone.Boss.HP, 0, "zone %s boss must have HP", key)
	}
}

func TestNewZoneEncounter(t *testing.T) {
	for i := 0; i < 50; i++ {
		enemy, isBoss := NewZoneEncounter("forest")
		require.NotNil(t, enemy)
		assert.Greater(t, enemy.HP, 0)
		assert.NotEmpty(t, enemy.Nickname)
		if isBoss {
			assert.Equal(t, Zones["forest"].Boss.Name, enemy.Nickname)
		} else {
			assert.Contains(t, []string{"Slime Gluant", "Sanglier Sauvage"}, enemy.Nickname)
		}
	}
}

func TestRecordZoneBossKill(t *testing.T) {
	svc, _ := testService(t)
	kills, err := svc.RecordZoneBossKill(1, "forest")
	require.NoError(t, err)
	assert.Equal(t, 1, kills, "first boss kill must report count 1")

	kills, err = svc.RecordZoneBossKill(1, "forest")
	require.NoError(t, err)
	assert.Equal(t, 2, kills, "second boss kill must report count 2")

	progress, err := svc.store.GetZoneProgress(1)
	require.NoError(t, err)
	assert.Equal(t, 0, progress["forest"], "boss kills must not affect win progress")
}

func TestExecuteHuntRecordsZoneStat(t *testing.T) {
	svc, s := testServiceWithCfg(t, &config.Config{StartingBalance: 100, HuntMaxPerDay: 10})
	addActivePet(t, s, 1)

	// A daily quest waiting on a forest hunt step must advance on a hunt.
	recipe := store.DailyRecipe{
		Requestor: "thorek", TitleKey: "quests.daily.thorek.title", IntroKey: "quests.daily.thorek.intro",
		Steps: []store.DailyStep{
			{Kind: store.DailyStepActivity, Stat: "hunt_forest", Count: 2, TextKey: "quests.daily.step.hunt_zone"},
			{Kind: store.DailyStepTurnIn, Items: map[string]int{"coal": 1}, TextKey: "quests.daily.thorek.turnin_coal"},
		},
		Reward: store.DailyReward{Money: 100},
	}
	data, err := json.Marshal(recipe)
	require.NoError(t, err)
	require.NoError(t, s.StartDailyQuest(1, string(data)))

	_, err = svc.ExecuteHunt(1, "forest")
	require.NoError(t, err)

	var d model.UserQuestData
	require.NoError(t, s.DB.Where("user_id = 1 AND quest_id = 'daily_quest'").First(&d).Error)
	assert.Equal(t, 1, d.ProgressValue, "forest hunt must count towards the zone step")

	// A hunt in another zone must not count.
	require.NoError(t, s.DB.Model(&model.UserPet{}).
		Where("user_id = 1").Update("hp", gorm.Expr("max_hp")).Error, "heal the pet between hunts")
	_, err = svc.ExecuteHunt(1, "cave")
	require.NoError(t, err)
	require.NoError(t, s.DB.Where("user_id = 1 AND quest_id = 'daily_quest'").First(&d).Error)
	assert.Equal(t, 1, d.ProgressValue, "cave hunt must not advance a forest objective")
}

func TestEggDropChancesScaleDownByProgression(t *testing.T) {
	zoneOrder := []string{"forest", "cave", "desert", "mountain", "ocean", "tundra", "volcano"}
	var prev float64 = 1.0
	for _, key := range zoneOrder {
		zone, ok := Zones[key]
		require.True(t, ok, "zone %q must exist", key)
		eggChance := 0.0
		for _, loot := range zone.LootTable {
			if strings.HasSuffix(loot.Item, "_egg") {
				eggChance = loot.Chance
				break
			}
		}
		assert.Greater(t, eggChance, 0.0, "zone %q must drop its biome egg", key)
		assert.LessOrEqual(t, eggChance, prev,
			"later zones must not drop eggs more often (zone %q)", key)
		prev = eggChance
	}
}
