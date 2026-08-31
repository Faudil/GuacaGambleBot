package furniture

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	housingsvc "guacagamblebot/internal/service/housing"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

func testService(t *testing.T) (*Service, *housingsvc.Service, *store.Store) {
	d := testutil.NewDB(t)
	cfg := &config.Config{StartingBalance: 1000, DailyAmount: 50}
	s := store.New(d, cfg)
	hsvc := housingsvc.New(s, cfg)
	return New(s, cfg, hsvc), hsvc, s
}

func buyHouse(t *testing.T, hsvc *housingsvc.Service, s *store.Store, userID int64, houseType string) {
	t.Helper()
	_, err := s.UpdateBalance(userID, 1000000)
	require.NoError(t, err)
	require.NoError(t, hsvc.BuyHouse(userID, houseType))
}

func fundForWorkbench(t *testing.T, s *store.Store, userID int64) {
	t.Helper()
	_, err := s.UpdateBalance(userID, 100000)
	require.NoError(t, err)
	require.NoError(t, s.AddItemRaw(s.DB, userID, "pebble", 100))
	require.NoError(t, s.AddItemRaw(s.DB, userID, "iron_ore", 100))
}

// TestGeneticsLabDefined guards the reported bug that the genetics lab furniture
// was "missing" from the house UI: the definition must exist and unlock research.
func TestGeneticsLabDefined(t *testing.T) {
	fd, ok := FurnitureDefs["genetics_lab"]
	require.True(t, ok, "genetics_lab must be defined so it shows in the house furniture UI")
	assert.NotEmpty(t, fd.Emoji)
	assert.NotEmpty(t, fd.Name)
	assert.Contains(t, fd.UnlocksResearch, "dna_research")
}

// TestFurnitureScopedPerHouse verifies that furniture belongs to the active
// house: switching houses shows an empty furniture list and the same item can be
// placed in a different house.
func TestFurnitureScopedPerHouse(t *testing.T) {
	fsvc, hsvc, s := testService(t)
	const uid = 1

	buyHouse(t, hsvc, s, uid, "wooden_shack")
	buyHouse(t, hsvc, s, uid, "brick_house") // brick_house becomes active
	fundForWorkbench(t, s, uid)

	require.NoError(t, fsvc.Place(uid, "workbench"))
	assert.True(t, fsvc.IsPlaced(uid, "workbench"))
	assert.Equal(t, 1, fsvc.GetUsedSlots(uid))
	placed, err := fsvc.GetPlaced(uid)
	require.NoError(t, err)
	require.Len(t, placed, 1)
	assert.Equal(t, "brick_house", placed[0].HouseType)

	// Switching houses: the other house starts empty and can host the same item.
	require.NoError(t, hsvc.SwitchHouse(uid, "wooden_shack"))
	assert.False(t, fsvc.IsPlaced(uid, "workbench"))
	assert.Equal(t, 0, fsvc.GetUsedSlots(uid))
	placed, err = fsvc.GetPlaced(uid)
	require.NoError(t, err)
	assert.Len(t, placed, 0)

	require.NoError(t, fsvc.Place(uid, "workbench"))
	assert.True(t, fsvc.IsPlaced(uid, "workbench"))

	// Remove only touches the active house.
	require.NoError(t, fsvc.Remove(uid, "workbench"))
	assert.False(t, fsvc.IsPlaced(uid, "workbench"))

	// The other house still owns its own copy.
	require.NoError(t, hsvc.SwitchHouse(uid, "brick_house"))
	assert.True(t, fsvc.IsPlaced(uid, "workbench"))
}

func TestPlaceDuplicateSameHouseFails(t *testing.T) {
	fsvc, hsvc, s := testService(t)
	const uid = 1

	buyHouse(t, hsvc, s, uid, "brick_house")
	fundForWorkbench(t, s, uid)

	require.NoError(t, fsvc.Place(uid, "workbench"))
	err := fsvc.Place(uid, "workbench")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already placed")
}

func TestPlaceRequiresHouse(t *testing.T) {
	fsvc, _, s := testService(t)
	const uid = 1

	fundForWorkbench(t, s, uid)
	err := fsvc.Place(uid, "workbench")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "don't own a house")
}

// TestCardboardBoxHasNoFurnitureSlots guards the tier gate: the cheapest house
// must have zero furniture slots so furniture availability grows from there.
func TestCardboardBoxHasNoFurnitureSlots(t *testing.T) {
	ht := housingsvc.Houses["cardboard_box"]
	require.NotNil(t, ht)
	assert.Zero(t, ht.FurnitureSlots)
}

func TestPlaceFailsOnZeroSlotHouse(t *testing.T) {
	fsvc, hsvc, s := testService(t)
	const uid = 1

	buyHouse(t, hsvc, s, uid, "cardboard_box")
	fundForWorkbench(t, s, uid)

	err := fsvc.Place(uid, "workbench")
	require.ErrorIs(t, err, ErrNoFurnitureSlots)
}

func TestSlotsAt(t *testing.T) {
	ht := housingsvc.Houses["brick_house"]
	require.NotNil(t, ht)
	// base 4, +1 per level, capped at housingsvc.MaxFurnitureSlots.
	assert.Equal(t, 4, ht.SlotsAt(1))
	assert.Equal(t, 5, ht.SlotsAt(2))
	assert.Equal(t, 8, ht.SlotsAt(5))
	assert.Equal(t, housingsvc.MaxFurnitureSlots, ht.SlotsAt(99))

	// cardboard has no slots at any level.
	cb := housingsvc.Houses["cardboard_box"]
	assert.Equal(t, 0, cb.SlotsAt(1))
}

func TestEffectValueAndHasFurniture(t *testing.T) {
	fsvc, hsvc, s := testService(t)
	const uid = 1

	// No house → no effects.
	assert.Equal(t, 0.0, EffectValue(s, uid, "farm_yield"))
	assert.False(t, HasFurniture(s, uid, "greenhouse_kit"))

	buyHouse(t, hsvc, s, uid, "cardboard_box")
	buyHouse(t, hsvc, s, uid, "brick_house") // active
	_, err := s.UpdateBalance(uid, 100000)
	require.NoError(t, err)
	require.NoError(t, s.AddItemRaw(s.DB, uid, "rotten_plant", 100))
	require.NoError(t, s.AddItemRaw(s.DB, uid, "wheat", 100))
	require.NoError(t, s.AddItemRaw(s.DB, uid, "gold_nugget", 100))
	require.NoError(t, fsvc.Place(uid, "greenhouse_kit"))

	assert.True(t, HasFurniture(s, uid, "greenhouse_kit"))
	assert.Equal(t, 0.10, EffectValue(s, uid, "farm_yield"))
	assert.Equal(t, 0.0, EffectValue(s, uid, "craft_cost"))

	// Effects are scoped to the active house.
	require.NoError(t, hsvc.SwitchHouse(uid, "cardboard_box"))
	assert.False(t, HasFurniture(s, uid, "greenhouse_kit"))
	assert.Equal(t, 0.0, EffectValue(s, uid, "farm_yield"))
}

func TestPlaceFailsWhenSizeExceedsSlots(t *testing.T) {
	fsvc, hsvc, s := testService(t)
	const uid = 1

	// wooden_shack: 2 slots base. Workbench (1) fits, genetics lab (3) does not.
	buyHouse(t, hsvc, s, uid, "wooden_shack")
	fundForWorkbench(t, s, uid)
	require.NoError(t, fsvc.Place(uid, "workbench"))

	_, err := s.UpdateBalance(uid, 100000)
	require.NoError(t, err)
	require.NoError(t, s.AddItemRaw(s.DB, uid, "epic_fossil", 50))
	require.NoError(t, s.AddItemRaw(s.DB, uid, "bone_dust", 100))
	require.NoError(t, s.AddItemRaw(s.DB, uid, "emerald", 50))

	err = fsvc.Place(uid, "genetics_lab")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no free slots")
}

func TestGetUsedSlotsSumsSizes(t *testing.T) {
	fsvc, hsvc, s := testService(t)
	const uid = 1

	buyHouse(t, hsvc, s, uid, "mansion") // 6 base slots
	fundForWorkbench(t, s, uid)
	require.NoError(t, s.AddItemRaw(s.DB, uid, "iron_ore", 100))
	require.NoError(t, s.AddItemRaw(s.DB, uid, "coal", 100))
	require.NoError(t, fsvc.Place(uid, "workbench")) // 1 slot
	require.NoError(t, fsvc.Place(uid, "forge"))     // 2 slots
	assert.Equal(t, 3, fsvc.GetUsedSlots(uid))
	assert.Equal(t, 6, fsvc.GetMaxSlots(uid))
}

func fundForBed(t *testing.T, s *store.Store, userID int64) {
	t.Helper()
	_, err := s.UpdateBalance(userID, 100000)
	require.NoError(t, err)
	require.NoError(t, s.AddItemRaw(s.DB, userID, "iron_ore", 100))
	require.NoError(t, s.AddItemRaw(s.DB, userID, "platinum", 100))
	require.NoError(t, s.AddItemRaw(s.DB, userID, "wheat", 100))
}

func TestBedDefined(t *testing.T) {
	fd, ok := FurnitureDefs["bed"]
	require.True(t, ok, "bed must be defined so it shows in the house furniture UI")
	assert.NotEmpty(t, fd.Emoji)
	assert.Equal(t, 2, fd.Slots)
}

func TestRestRequiresBed(t *testing.T) {
	fsvc, hsvc, s := testService(t)
	const uid = 1
	buyHouse(t, hsvc, s, uid, "brick_house")

	err := fsvc.Rest(uid)
	assert.ErrorIs(t, err, ErrNoBed)
}

func TestRestResetsCasinoLimits(t *testing.T) {
	fsvc, hsvc, s := testService(t)
	const uid = 1
	buyHouse(t, hsvc, s, uid, "brick_house")
	fundForBed(t, s, uid)
	require.NoError(t, fsvc.Place(uid, "bed"))

	// Exhaust the slots limit.
	for i := 0; i < 10; i++ {
		require.NoError(t, s.IncrementGameLimit(uid, "slots"))
	}
	ok, _, err := s.CheckGameLimit(uid, "slots", 10)
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, fsvc.Rest(uid))
	ok, _, err = s.CheckGameLimit(uid, "slots", 10)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestRestHalfRefunds(t *testing.T) {
	fsvc, hsvc, s := testService(t)
	const uid = 1
	buyHouse(t, hsvc, s, uid, "brick_house")
	fundForBed(t, s, uid)
	require.NoError(t, fsvc.Place(uid, "bed"))

	// Use all 20 farm uses: a rest refunds half of the daily capacity.
	for i := 0; i < 20; i++ {
		require.NoError(t, s.IncrementGameLimit(uid, "farm"))
	}
	require.NoError(t, fsvc.Rest(uid))

	_, remaining, err := s.CheckGameLimit(uid, "farm", 20)
	require.NoError(t, err)
	assert.Equal(t, 10, remaining)
}

func TestRestOncePerDay(t *testing.T) {
	fsvc, hsvc, s := testService(t)
	const uid = 1
	buyHouse(t, hsvc, s, uid, "brick_house")
	fundForBed(t, s, uid)
	require.NoError(t, fsvc.Place(uid, "bed"))

	require.NoError(t, fsvc.Rest(uid))
	err := fsvc.Rest(uid)
	assert.ErrorIs(t, err, ErrAlreadySlept)
}
