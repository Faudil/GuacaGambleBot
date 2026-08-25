package quests

import (
	"testing"

	"guacagamblebot/internal/model"

	"github.com/stretchr/testify/require"
)

func TestWorldSpawnRespectsSpace(t *testing.T) {
	svc, st := testService(t)
	db := st.DB
	orig := WorldQuestSpawns
	WorldQuestSpawns = []WorldQuestSpawn{
		{QuestID: "world_lost_tool", Weight: 10, SpawnChance: 1, TriggerStats: []string{"items_mined"}},
		{QuestID: "world_stray_seed", Weight: 10, SpawnChance: 1, TriggerStats: []string{"items_mined"}},
		{QuestID: "world_tide_gift", Weight: 10, SpawnChance: 1, TriggerStats: []string{"items_mined"}},
	}
	t.Cleanup(func() { WorldQuestSpawns = orig })

	// fill to 4
	require.NoError(t, st.CreateQuest(1, "tutorial"))
	require.NoError(t, st.CreateQuest(1, "boss_league"))
	require.NoError(t, st.CreateQuest(1, "world_lost_tool"))
	require.NoError(t, st.CreateQuest(1, "world_stray_seed"))
	require.Equal(t, 4, ActiveQuestCount(st, 1))
	require.True(t, CanSpawnWorldQuest(st, 1))
	id, ok := svc.TrySpawnWorldQuest(1, "items_mined")
	require.True(t, ok)
	require.Equal(t, "world_tide_gift", id)
	require.Equal(t, 5, ActiveQuestCount(st, 1))
	// at 5 should block
	require.False(t, CanSpawnWorldQuest(st, 1))
	id2, ok2 := svc.TrySpawnWorldQuest(1, "items_mined")
	require.False(t, ok2)
	require.Empty(t, id2)
	// manual StartQuest can still go beyond 5
	require.NoError(t, st.CreateQuest(1, "lost_warden"))
	require.Equal(t, 6, ActiveQuestCount(st, 1))
	// completed quests never respawn
	db.Model(&model.UserQuest{}).Where("user_id=? AND quest_id=?", 2, "world_lost_tool").Delete(&model.UserQuest{})
	require.NoError(t, st.CreateQuest(2, "world_lost_tool"))
	db.Model(&model.UserQuest{}).Where("user_id=? AND quest_id=?", 2, "world_lost_tool").Update("status", "COMPLETED")
	WorldQuestSpawns = []WorldQuestSpawn{{QuestID: "world_lost_tool", Weight: 10, SpawnChance: 1, TriggerStats: []string{"items_mined"}}}
	_, ok = svc.TrySpawnWorldQuest(2, "items_mined")
	require.False(t, ok, "completed quest must not respawn")
}

func TestWorldSpawnViaRecordActivity(t *testing.T) {
	svc2, st2 := testService(t)
	_ = svc2
	st := st2
	db := st.DB
	orig := WorldQuestSpawns
	WorldQuestSpawns = []WorldQuestSpawn{{QuestID: "world_lost_tool", Weight: 10, SpawnChance: 1, TriggerStats: []string{"items_mined"}}}
	t.Cleanup(func() { WorldQuestSpawns = orig })
	require.NoError(t, st.CreateQuest(3, "tutorial"))
	require.Equal(t, 1, ActiveQuestCount(st, 3))
	// RecordActivity should trigger world spawn hook
	require.NoError(t, st.RecordActivity(3, "items_mined", 1))
	require.Equal(t, 2, ActiveQuestCount(st, 3))
	var n int64
	db.Model(&model.QuestNotification{}).Where("user_id=?", 3).Count(&n)
	require.GreaterOrEqual(t, n, int64(1))
}
