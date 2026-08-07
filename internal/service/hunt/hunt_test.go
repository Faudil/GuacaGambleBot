package hunt

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/model"
	invsvc "guacagamblebot/internal/service/inventory"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
	"guacagamblebot/internal/universe/hoakhaven"
)

func testService(t *testing.T) (*Service, *store.Store) {
	return testServiceWithCfg(t, &config.Config{StartingBalance: 100})
}

func testServiceWithCfg(t *testing.T, cfg *config.Config) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "h.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
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
	assert.NotEmpty(t, res.Log)
	assert.Greater(t, res.XP, 0)
}

func TestExecuteHuntInvalidZone(t *testing.T) {
	svc, s := testService(t)
	addActivePet(t, s, 1)
	_, err := svc.ExecuteHunt(1, "nonexistent")
	assert.Error(t, err)
}

func TestNewEnemy(t *testing.T) {
	e := NewEnemy("forest")
	assert.NotEmpty(t, e.Name)
	assert.Greater(t, e.HP, 0)
	assert.Greater(t, e.Level, 0)
}

func TestBattleLogTracksHP(t *testing.T) {
	svc, s := testService(t)
	addActivePet(t, s, 1)
	res, err := svc.ExecuteHunt(1, "forest")
	require.NoError(t, err)
	require.NotEmpty(t, res.Log)

	assert.NotEmpty(t, res.EnemyName)
	assert.NotEmpty(t, res.EnemyEmoji)
	assert.Greater(t, res.EnemyLevel, 0)
	assert.Equal(t, res.PetStartHP, res.Log[0].PetHP, "first entry must record the starting pet HP")

	prevPetHP, prevEnemyHP := res.PetStartHP, res.EnemyMaxHP
	for _, e := range res.Log {
		assert.LessOrEqual(t, e.EnemyHP, prevEnemyHP, "enemy HP must never increase")
		assert.LessOrEqual(t, e.PetHP, prevPetHP, "pet HP must never increase")
		assert.GreaterOrEqual(t, e.EnemyHP, 0)
		assert.GreaterOrEqual(t, e.PetHP, 0)
		prevPetHP, prevEnemyHP = e.PetHP, e.EnemyHP
	}

	last := res.Log[len(res.Log)-1]
	if res.PlayerWon {
		assert.Equal(t, 0, last.EnemyHP)
	} else if res.EnemyWon {
		assert.Equal(t, 0, last.PetHP)
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
		assert.NotEmpty(t, enemy.Name)
		if isBoss {
			assert.Equal(t, Zones["forest"].Boss.Name, enemy.Name)
		} else {
			assert.Contains(t, []string{"Slime Gluant", "Sanglier Sauvage"}, enemy.Name)
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
