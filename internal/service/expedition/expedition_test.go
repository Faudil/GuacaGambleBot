package expedition

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d := testutil.NewDB(t)
	cfg := &config.Config{}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func testPet() *model.UserPet {
	return &model.UserPet{
		ID: 1, UserID: 1, PetType: "Dragon", Nickname: "Draco",
		Level: 10, MaxHP: 200, HP: 200, Atk: 40, Defense: 30, Speed: 30,
		DGE: 20, ACC: 30, CritC: 15, CritD: 2.0, SpcC: 10,
	}
}

func TestGenerateExpedition(t *testing.T) {
	svc, _ := testService(t)
	pet := testPet()
	res := svc.Generate(pet, 2)
	assert.NotNil(t, res)
	assert.Greater(t, res.XP, 0)
	assert.NotEmpty(t, res.Log)
	assert.LessOrEqual(t, res.PetHP, pet.HP)
	assert.GreaterOrEqual(t, res.PetHP, 0)
}

func TestGenerateStructuredEvents(t *testing.T) {
	svc, _ := testService(t)
	pet := testPet()
	res := svc.Generate(pet, 1)
	require.NotEmpty(t, res.Log)

	for _, ev := range res.Log {
		require.Contains(t, []string{"exploration", "combat", "loot", "rest", "return"}, ev.Type, "event type must be one of the known categories")
		switch ev.Type {
		case "exploration":
			assert.NotEmpty(t, ev.Location, "exploration events must carry the location key")
			assert.NotEmpty(t, ev.Text, "exploration events must keep a fallback text")
		case "combat":
			assert.NotEmpty(t, ev.Enemy, "combat events must carry the enemy species")
			assert.Greater(t, ev.EnemyLevel, 0, "combat events must carry the enemy level")
			assert.Contains(t, []string{"win", "loss", "stalemate"}, ev.CombatResult)
		case "loot":
			assert.NotEmpty(t, ev.Item, "loot events must carry the item id")
		case "rest":
			assert.GreaterOrEqual(t, ev.Heal, 0, "rest events must carry the healed amount")
		case "return":
			assert.NotEmpty(t, ev.Text, "return events must keep a fallback text")
		}
	}
}

func TestStartRejectsKOPet(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.User{UserID: 1}).Error)
	ko := testPet()
	require.NoError(t, st.DB.Create(ko).Error)
	// KO pets are persisted via column updates (GORM skips zero values on
	// CREATE because of the hp default), mirroring the real heal/expedition flow.
	require.NoError(t, st.DB.Model(&model.UserPet{}).Where("id = ?", ko.ID).Update("hp", 0).Error)

	res := svc.Generate(testPet(), 1)
	exp, err := svc.Start(1, ko.ID, 1, res)
	assert.ErrorIs(t, err, ErrPetKO)
	assert.Nil(t, exp, "no expedition must be started for a K.O. pet")
}

func TestStartAndGetActive(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.User{UserID: 1}).Error)
	require.NoError(t, st.DB.Create(testPet()).Error)

	res := svc.Generate(testPet(), 2)
	exp, err := svc.Start(1, 1, 2, res)
	require.NoError(t, err)
	require.NotNil(t, exp)
	assert.Equal(t, int64(1), exp.UserID)
	assert.Equal(t, int64(1), exp.PetID)
	assert.False(t, exp.IsClaimed)

	// HP loss from combat events must be persisted on the pet.
	var pet model.UserPet
	require.NoError(t, st.DB.First(&pet, 1).Error)
	assert.Equal(t, res.PetHP, pet.HP)
	assert.True(t, pet.OnExpedition)

	active, err := svc.GetActive(1)
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, exp.ID, active.ID)
}

func TestClaim(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.User{UserID: 1}).Error)
	require.NoError(t, st.DB.Create(testPet()).Error)

	res := svc.Generate(testPet(), 1)
	exp, err := svc.Start(1, 1, 1, res)
	require.NoError(t, err)

	_, _, err = svc.Claim(exp)
	require.NoError(t, err)

	_, err = svc.GetActive(1)
	assert.Error(t, err)
}

func TestClaimRecordsExpeditionCompletion(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.User{UserID: 1}).Error)
	require.NoError(t, st.DB.Create(testPet()).Error)

	// A quest with a matching activity step must see the completion.
	require.NoError(t, st.DB.Create(&model.UserQuest{
		UserID: 1, QuestID: "chronicler_legend", Status: "ACTIVE",
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserQuestData{
		UserID: 1, QuestID: "chronicler_legend", StepIndex: 2,
		ProgressValue: 0, CustomData: `{"target_stat":"expedition_completions","target_count":2}`,
	}).Error)

	claim := func() {
		t.Helper()
		// The expedition sim may leave the pet knocked out; heal it so the
		// second expedition is accepted regardless of the random outcome.
		require.NoError(t, st.DB.Model(&model.UserPet{}).Where("id = ?", 1).
			Update("hp", testPet().MaxHP).Error)
		res := svc.Generate(testPet(), 1)
		exp, err := svc.Start(1, 1, 1, res)
		require.NoError(t, err)
		_, _, err = svc.Claim(exp)
		require.NoError(t, err)
	}

	claim()
	claim()

	var qd model.UserQuestData
	require.NoError(t, st.DB.Where("user_id = ? AND quest_id = ?", 1, "chronicler_legend").First(&qd).Error)
	assert.Equal(t, 2, qd.ProgressValue, "two claimed expeditions must satisfy the step")
}

// TestGenerateStopsWhenKO verifies that a pet knocked out mid-adventure stops
// simulating and gets a final return-home event instead of continuing.
func TestGenerateStopsWhenKO(t *testing.T) {
	svc, _ := testService(t)
	weak := testPet()
	weak.HP = 1

	for i := 0; i < 50; i++ {
		res := svc.Generate(weak, 4)
		if res.PetHP > 0 {
			continue
		}
		last := res.Log[len(res.Log)-1]
		assert.Equalf(t, "return", last.Type, "a KO'd pet must end its log with a return event (run %d)", i)
		assert.Equal(t, "loss", res.Log[len(res.Log)-2].CombatResult, "the return must follow the KO combat")
		assert.True(t, res.ReturnAt > 0, "a KO'd pet must carry an early return time (run %d)", i)
		for j, ev := range res.Log {
			if ev.Type == "return" {
				assert.Equalf(t, len(res.Log)-1, j, "the return event may only be the last one (run %d)", i)
			}
		}
	}
}

// TestStartTruncatesEndTimeWhenKO verifies that an expedition whose pet was
// knocked out ends at the return time instead of the full requested window.
func TestStartTruncatesEndTimeWhenKO(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.User{UserID: 1}).Error)
	require.NoError(t, st.DB.Create(testPet()).Error)

	res := &ExpeditionResult{
		Log: []ExpeditionEvent{
			{Time: 60, Type: "combat", CombatResult: "loss", Text: "KO"},
			{Time: 60, Type: "return", Text: "Home"},
		},
		XP:       10,
		PetHP:    0,
		ReturnAt: 60 * time.Minute,
	}
	exp, err := svc.Start(1, 1, 4, res)
	require.NoError(t, err)
	require.NotNil(t, exp)

	assert.Equal(t, 60*time.Minute, exp.EndTime.Sub(exp.StartTime),
		"a KO'd pet must return home early instead of at the full duration")

	var pet model.UserPet
	require.NoError(t, st.DB.First(&pet, 1).Error)
	assert.Equal(t, 0, pet.HP, "the pet must come back knocked out")
	assert.True(t, pet.OnExpedition)

	// A full-duration run must not be truncated.
	require.NoError(t, st.DB.Model(&model.UserPet{}).Where("id = ?", 1).
		Update("hp", testPet().MaxHP).Error)
	full, err := svc.Start(1, 1, 4, &ExpeditionResult{
		Log:   []ExpeditionEvent{{Time: 240, Type: "exploration", Text: "ok"}},
		XP:    10,
		PetHP: testPet().MaxHP,
	})
	require.NoError(t, err)
	assert.Equal(t, 4*time.Hour, full.EndTime.Sub(full.StartTime))
}

// TestRestHealsPet verifies that rest events restore a rolled amount of HP.
func TestRestHealsPet(t *testing.T) {
	svc, _ := testService(t)
	pet := testPet()
	pet.HP = pet.MaxHP / 2
	found := false
	for i := 0; i < 200; i++ {
		res := svc.Generate(pet, 1)
		for _, ev := range res.Log {
			if ev.Type == "rest" {
				found = true
				assert.Greater(t, ev.Heal, 0, "a rest on a wounded pet must heal some HP")
				assert.LessOrEqual(t, res.PetHP, pet.MaxHP, "healing must never exceed max HP")
				break
			}
		}
		if found {
			break
		}
	}
	assert.True(t, found, "a rest event must occur within the samples")
}
