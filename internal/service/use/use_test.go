package use

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d := testutil.NewDB(t)
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

func TestUseHookGrantsFishCredit(t *testing.T) {
	svc, st := testService(t)
	require.NoError(t, st.AddItemRaw(st.DB, 1, "hook", 1))
	require.NoError(t, st.IncrementGameLimit(1, "fish"))
	require.NoError(t, st.IncrementGameLimit(1, "fish"))

	_, err := svc.Apply(1, "hook")
	require.NoError(t, err)

	ok, remaining, err := st.CheckGameLimit(1, "fish", 10)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 10, remaining, "2 uses minus 2 hook credits = 0 -> full limit")

	has, err := st.HasItem(1, "hook", 1)
	require.NoError(t, err)
	assert.False(t, has, "hook consumed from inventory")
}

// Deprecated alias – old name expected “cooldown” but hook now grants +2 fish uses.
func TestUseHookResetsFishCooldown(t *testing.T) { TestUseHookGrantsFishCredit(t) }

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
