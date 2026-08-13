package hunt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/items"
)

// rollHuntGear must only offer gear at or below the player's level, and stay
// within 10 levels of it (never starter junk for high-level players).
func TestRollHuntGearLevelAppropriate(t *testing.T) {
	svc, _ := testService(t)

	seen := map[string]bool{}
	for range 500 {
		it, ok := svc.rollHuntGear(1, 12, 1.0)
		if !ok {
			continue
		}
		seen[it.ID] = true
		assert.LessOrEqual(t, it.MinLevel, 12, "gear must not exceed player level")
		assert.GreaterOrEqual(t, it.MinLevel, 3, "gear must be within 10 levels of player")
	}
	assert.NotEmpty(t, seen, "level 12 player should be able to drop gear")
}

// A level 1 player may still drop the common starter gear.
func TestRollHuntGearLevel1(t *testing.T) {
	svc, _ := testService(t)

	it, ok := svc.rollHuntGear(1, 1, 1.0)
	if ok {
		assert.LessOrEqual(t, it.MinLevel, 1)
		require.NotEmpty(t, it.ID)
	}
}

// grantGearInstance must produce a persistent equipment instance.
func TestGrantGearInstanceCreatesEquipment(t *testing.T) {
	svc, s := testService(t)

	bow := items.Get("hunters_bow")
	require.NotNil(t, bow)

	svc.grantGearInstance(1, *bow)

	eqs, err := s.GetAllUserEquipment(1)
	require.NoError(t, err)
	require.Len(t, eqs, 1)
	assert.Equal(t, "hunters_bow", eqs[0].BaseID)
	assert.Equal(t, "weapon", eqs[0].EquipSlot)
	assert.Equal(t, 10, eqs[0].MinLevel)
}
