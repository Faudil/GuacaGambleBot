package npcs

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "npcs.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func TestGetNPCData(t *testing.T) {
	svc, _ := testService(t)
	n := svc.GetNPCData("elara")
	require.NotNil(t, n)
	assert.Equal(t, "Elara", n.Name)
	assert.Equal(t, "🌿", n.Emoji)
}

func TestGetNPCDataMissing(t *testing.T) {
	svc, _ := testService(t)
	assert.Nil(t, svc.GetNPCData("nonexistent_npc"))
}

func TestGetAllNPCMeta(t *testing.T) {
	svc, _ := testService(t)
	all := svc.GetAllNPCMeta()
	assert.Len(t, all, len(NPCs))
}

func TestGetReputationCreatesDefault(t *testing.T) {
	svc, _ := testService(t)
	rep, err := svc.GetReputation(1, "elara")
	require.NoError(t, err)
	assert.Equal(t, int64(1), rep.UserID)
	assert.Equal(t, "elara", rep.NPCID)
	assert.Equal(t, 0, rep.Reputation)
	assert.Equal(t, 1, rep.Level)
}

func TestAddReputation(t *testing.T) {
	svc, _ := testService(t)
	added, err := svc.AddReputation(1, "elara", 50)
	require.NoError(t, err)
	assert.Equal(t, 50, added)

	rep, err := svc.GetReputation(1, "elara")
	require.NoError(t, err)
	assert.Equal(t, 50, rep.Reputation)
	assert.Equal(t, 1, rep.Level)
}

func TestAddReputationTriggersLevelUp(t *testing.T) {
	svc, _ := testService(t)
	added, err := svc.AddReputation(1, "elara", 100)
	require.NoError(t, err)
	assert.Equal(t, 100, added)

	rep, err := svc.GetReputation(1, "elara")
	require.NoError(t, err)
	assert.Equal(t, 0, rep.Reputation) // 100 - 100 = 0 after level up
	assert.Equal(t, 2, rep.Level)
}

func TestAddReputationExcessOverLevelUp(t *testing.T) {
	svc, _ := testService(t)
	added, err := svc.AddReputation(1, "elara", 250)
	require.NoError(t, err)
	assert.Equal(t, 250, added)

	rep, err := svc.GetReputation(1, "elara")
	require.NoError(t, err)
	assert.Equal(t, 150, rep.Reputation) // 250 - 100 (lvl1) = 150, not enough for lvl3 (needs 200)
	assert.Equal(t, 2, rep.Level)
}

func TestAddReputationRespectsDailyCap(t *testing.T) {
	svc, st := testService(t)
	today := time.Now().Format("2006-01-02")
	require.NoError(t, st.DB.Create(&model.UserNPCDailyRep{
		UserID: 1, NPCID: "elara", DateStr: today, Amount: 480,
	}).Error)

	added, err := svc.AddReputation(1, "elara", 50)
	require.NoError(t, err)
	assert.Equal(t, 20, added)
}

func TestGetAllReputations(t *testing.T) {
	svc, _ := testService(t)
	reps, err := svc.GetAllReputations(1)
	require.NoError(t, err)
	assert.Empty(t, reps)
}

func TestRankUp(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.UserNPCReputation{
		UserID: 1, NPCID: "elara", Reputation: 150, Level: 1,
	}).Error)

	err := svc.RankUp(1, "elara")
	require.NoError(t, err)

	rep, err := svc.GetReputation(1, "elara")
	require.NoError(t, err)
	assert.Equal(t, 2, rep.Level)
	assert.Equal(t, 0, rep.Reputation)
}

func TestRankUpNotEnough(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.UserNPCReputation{
		UserID: 1, NPCID: "elara", Reputation: 50, Level: 1,
	}).Error)

	err := svc.RankUp(1, "elara")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enough reputation")
}

func TestGetBonuses(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.DB.Create(&model.UserNPCReputation{
		UserID: 1, NPCID: "gamblebot", Reputation: 0, Level: 3,
	}).Error)
	require.NoError(t, st.DB.Create(&model.UserNPCReputation{
		UserID: 1, NPCID: "thorek", Reputation: 0, Level: 2,
	}).Error)

	b := svc.GetBonuses(1)
	assert.Equal(t, 6.0, b.ShopDiscount)       // level 3 * 2
	assert.Equal(t, 4, b.MiningRiskReduction)   // level 2 * 2
}

func TestRankName(t *testing.T) {
	assert.Equal(t, "Inconnu", RankName(0))
	assert.Equal(t, "Inconnu", RankName(1))
	assert.Equal(t, "Connaissance", RankName(2))
	assert.Equal(t, "Partenaire", RankName(10))
}

func TestGetDailyRepCap(t *testing.T) {
	cap := GetDailyRepCap()
	assert.Equal(t, 500, cap.Flat)
	assert.Equal(t, 0, cap.PerLevel)
}
