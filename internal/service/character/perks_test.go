package character

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
	"guacagamblebot/internal/store"
)

func perkTestStore(t *testing.T) *store.Store {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "perk.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	return store.New(d, &config.Config{StartingBalance: 100})
}

func TestRollPerkChoicesExcludesOwnedPassives(t *testing.T) {
	s := perkTestStore(t)
	char, err := s.EnsureCharacter(1)
	require.NoError(t, err)
	require.NoError(t, s.AddPassive(1, "perk_xp_boost"))
	char, err = s.GetCharacter(1)
	require.NoError(t, err)

	choices := RollPerkChoices(char)
	require.Len(t, choices, 3)
	for _, p := range choices {
		assert.NotEqual(t, "perk_xp_boost", p.ID, "owned passive must not be offered again")
	}
}

func TestApplyPerkInstant(t *testing.T) {
	s := perkTestStore(t)
	_, err := s.EnsureCharacter(1)
	require.NoError(t, err)
	_, _, err = s.AddCharacterXP(1, 300) // level 2 → 1 perk point
	require.NoError(t, err)

	desc, err := ApplyPerk(s, 1, "perk_str")
	require.NoError(t, err)
	assert.Contains(t, desc, "STR")

	c, err := s.GetCharacter(1)
	require.NoError(t, err)
	assert.Equal(t, 7, c.STR, "+2 STR from perk on top of base 5")
	assert.Equal(t, 0, c.PerkPoints, "perk point consumed")
}

func TestApplyPerkPassive(t *testing.T) {
	s := perkTestStore(t)
	_, err := s.EnsureCharacter(1)
	require.NoError(t, err)
	_, _, err = s.AddCharacterXP(1, 300)
	require.NoError(t, err)

	_, err = ApplyPerk(s, 1, "perk_mine_yield")
	require.NoError(t, err)
	assert.True(t, s.HasPassive(1, "perk_mine_yield"))
}

func TestApplyPerkNoPoints(t *testing.T) {
	s := perkTestStore(t)
	_, err := s.EnsureCharacter(1)
	require.NoError(t, err)

	_, err = ApplyPerk(s, 1, "perk_str")
	assert.ErrorIs(t, err, store.ErrNoPerkPoints)
}

func TestApplyPerkUnknown(t *testing.T) {
	s := perkTestStore(t)
	_, err := s.EnsureCharacter(1)
	require.NoError(t, err)
	_, _, err = s.AddCharacterXP(1, 300)
	require.NoError(t, err)

	_, err = ApplyPerk(s, 1, "perk_does_not_exist")
	assert.Error(t, err)
}

func TestApplyPerkGoldScalesWithLevel(t *testing.T) {
	s := perkTestStore(t)
	_, err := s.EnsureCharacter(1)
	require.NoError(t, err)
	_, _, err = s.AddCharacterXP(1, 900) // level 3 (300 + 600 XP)
	require.NoError(t, err)

	before, err := s.GetBalance(1)
	require.NoError(t, err)
	_, err = ApplyPerk(s, 1, "perk_gold")
	require.NoError(t, err)
	after, err := s.GetBalance(1)
	require.NoError(t, err)
	assert.Equal(t, 60, after-before, "gold = 20 x level 3")
}

var _ = model.UserCharacter{}
