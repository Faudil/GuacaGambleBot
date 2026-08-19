package use

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/store"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "use.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	s := store.New(d, &config.Config{StartingBalance: 100})
	return New(s, &config.Config{StartingBalance: 100}), s
}

func TestUseNotOwned(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.Apply(1, "beer")
	assert.ErrorIs(t, err, ErrNotOwned)
}

func TestUseNotUsable(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.AddItemRaw(st.DB, 1, "coal", 1))
	_, err := svc.Apply(1, "coal")
	assert.ErrorIs(t, err, ErrNotUsable)
}

func TestIsUsable(t *testing.T) {
	assert.True(t, IsUsable("beer"))
	assert.True(t, IsUsable("electric_magnet"))
	assert.False(t, IsUsable("coal"))
	assert.False(t, IsUsable("diamond_sword"))
	assert.False(t, IsUsable(""))
}

func TestUseBeerConsumesItem(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.AddItemRaw(st.DB, 1, "beer", 1))
	require.NoError(t, st.IncrementGameLimit(1, "mine_descend"))
	require.NoError(t, st.IncrementGameLimit(1, "mine_descend"))

	desc, err := svc.Apply(1, "beer")
	require.NoError(t, err)
	assert.Contains(t, desc, "mine")

	ok, remaining, err := st.CheckGameLimit(1, "mine_descend", 10)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 10, remaining, "2 uses minus 2 beer credits = 0 -> full limit")

	has, err := st.HasItem(1, "beer", 1)
	require.NoError(t, err)
	assert.False(t, has, "beer consumed from inventory")
}

func TestUseRiggedCoinSetsBuff(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.AddItemRaw(st.DB, 1, "rigged_coin", 1))

	_, err := svc.Apply(1, "rigged_coin")
	require.NoError(t, err)

	ok, _ := st.HasActiveBuff(1, "rigged_coin")
	assert.True(t, ok)
}

func TestUseHookResetsFishCooldown(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.AddItemRaw(st.DB, 1, "hook", 1))
	require.NoError(t, st.SetCooldown(1, "fish"))

	_, err := svc.Apply(1, "hook")
	require.NoError(t, err)

	ready, err := st.CheckCooldown(1, "fish", 24*3600*1e9)
	require.NoError(t, err)
	assert.True(t, ready, "cooldown was cleared")
}

func TestUseScratchTicketPaysOut(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.AddItemRaw(st.DB, 1, "scratch_ticket", 1))

	before, err := st.GetBalance(1)
	require.NoError(t, err)
	desc, err := svc.Apply(1, "scratch_ticket")
	require.NoError(t, err)
	assert.Contains(t, desc, "Scratch")

	after, err := st.GetBalance(1)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, after, before, "payout is never negative")
}

func TestUseMagnetsConsumeAndPullOre(t *testing.T) {
	for _, itemID := range []string{"rusty_magnet", "magnet", "electric_magnet"} {
		svc, st := testService(t)
		require.NoError(t, st.AddItemRaw(st.DB, 1, itemID, 1))

		desc, err := svc.Apply(1, itemID)
		require.NoError(t, err)
		assert.Contains(t, desc, "Magnet")

		has, err := st.HasItem(1, itemID, 1)
		require.NoError(t, err)
		assert.False(t, has, "%s consumed from inventory", itemID)
	}
}

func TestUseMagnetGrantsMarketableOre(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.AddItemRaw(st.DB, 1, "magnet", 1))

	_, err := svc.Apply(1, "magnet")
	require.NoError(t, err)

	var invs []struct {
		ItemID string
	}
	require.NoError(t, st.DB.Table("inventory").
		Where("user_id = ? AND item_id IN ?", 1, []string{"silver_ore", "gold_nugget", "platinum", "emerald", "rough_diamond"}).
		Find(&invs).Error)
	assert.NotEmpty(t, invs, "magnet should pull ore into the inventory")
}
