package furniture

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	housingsvc "guacagamblebot/internal/service/housing"
	"guacagamblebot/internal/store"
)

func testService(t *testing.T) (*Service, *housingsvc.Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "furn.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
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
