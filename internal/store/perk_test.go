package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrantGameLimitCredit(t *testing.T) {
	s := newStore(t)

	require.NoError(t, s.IncrementGameLimit(1, "mine_descend"))
	require.NoError(t, s.IncrementGameLimit(1, "mine_descend"))

	require.NoError(t, s.GrantGameLimitCredit(1, "mine_descend", 3))
	ok, remaining, err := s.CheckGameLimit(1, "mine_descend", 10)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 10, remaining, "3 credits must cancel the 2 uses and never go below zero")

	// Granting credits with no usage today is a no-op.
	require.NoError(t, s.GrantGameLimitCredit(1, "slots", 2))
	ok, remaining, err = s.CheckGameLimit(1, "slots", 10)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 10, remaining)
}

func TestResetGameLimit(t *testing.T) {
	s := newStore(t)

	require.NoError(t, s.IncrementGameLimit(1, "slots"))
	require.NoError(t, s.IncrementGameLimit(1, "slots"))
	ok, remaining, err := s.CheckGameLimit(1, "slots", 10)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 8, remaining, "2 of 10 used")

	require.NoError(t, s.ResetGameLimit(1, "slots"))
	ok, remaining, err = s.CheckGameLimit(1, "slots", 10)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 10, remaining, "limit fully refreshed")
}

func TestPerkPointsAndPassives(t *testing.T) {
	s := newStore(t)

	// Level up once: +1 perk point, +2 skill points (300 XP for level 1→2).
	leveled, lvl, err := s.AddCharacterXP(1, 300)
	require.NoError(t, err)
	assert.True(t, leveled)
	assert.Equal(t, 2, lvl)

	points, err := s.GetPerkPoints(1)
	require.NoError(t, err)
	assert.Equal(t, 1, points)

	require.NoError(t, s.AddPassive(1, "perk_xp_boost"))
	assert.True(t, s.HasPassive(1, "perk_xp_boost"))
	assert.False(t, s.HasPassive(1, "perk_mine_yield"))
	// Re-adding a passive is idempotent.
	require.NoError(t, s.AddPassive(1, "perk_xp_boost"))

	require.NoError(t, s.DecrementPerkPoints(1))
	assert.Equal(t, ErrNoPerkPoints, s.DecrementPerkPoints(1))

	c, err := s.GetCharacter(1)
	require.NoError(t, err)
	assert.Contains(t, c.Passives, "perk_xp_boost")
	assert.Equal(t, 0, c.PerkPoints)
}

func TestQuickLearnerDoublesXP(t *testing.T) {
	s := newStore(t)

	require.NoError(t, s.SetActiveBuff(1, "quick_learner"))
	_, lvl, err := s.AddCharacterXP(1, 150)
	require.NoError(t, err)
	// 150 * 2 = 300 XP → exactly level 2 (needs 300 for level 1→2).
	assert.Equal(t, 2, lvl)

	// Buff is consumed: a plain 100 XP award no longer doubles and stays level 2.
	_, lvl, err = s.AddCharacterXP(1, 100)
	require.NoError(t, err)
	assert.Equal(t, 2, lvl)
}

func TestXpBoostPassive(t *testing.T) {
	s := newStore(t)

	require.NoError(t, s.AddPassive(1, "perk_xp_boost"))
	// 100 * 1.05 = 105 XP → still level 1.
	_, lvl, err := s.AddCharacterXP(1, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, lvl)

	c, err := s.GetCharacter(1)
	require.NoError(t, err)
	assert.Equal(t, 105, c.XP)
}
