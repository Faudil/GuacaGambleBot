package farm

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
	invsvc "guacagamblebot/internal/service/inventory"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
	"guacagamblebot/internal/universe/hoakhaven"
)

func newFarmEventService(t *testing.T) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "fe.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100}
	s := store.New(d, cfg)
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	require.NotNil(t, def)
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)
	return New(s, cfg, npcSvc), s
}

func TestRollBuriedMagnetRequiresOwnedMagnet(t *testing.T) {
	svc, st := newFarmEventService(t)

	evt := svc.rollBuriedMagnet(1, "public")
	assert.Nil(t, evt, "no event without a magnet")

	require.NoError(t, st.AddItemRaw(st.DB, 1, "magnet", 1))
	evt = svc.rollBuriedMagnet(1, "public")
	require.NotNil(t, evt, "event appears when a magnet is owned")
	assert.Equal(t, EventBuriedMagnet, evt.Type)
	assert.Len(t, evt.Choices, 4)
}

func TestResolveBuriedMagnetConsumesAndGrantsOre(t *testing.T) {
	svc, st := newFarmEventService(t)
	require.NoError(t, st.AddItemRaw(st.DB, 1, "electric_magnet", 1))

	evt := &Event{
		Type:      EventBuriedMagnet,
		ZoneKey:   "public",
		PlotIndex: -1,
	}
	res := svc.ResolveEvent(1, evt, "electric_magnet")
	require.NotNil(t, res)
	assert.NotEmpty(t, res.Items, "should grant ore")
	assert.Equal(t, "farm.event_buried_win_title", res.Title)

	has, err := st.HasItem(1, "electric_magnet", 1)
	require.NoError(t, err)
	assert.False(t, has, "magnet consumed")

	var inv model.Inventory
	err = st.DB.Where("user_id = ? AND item_id IN ?", 1, []string{"platinum", "emerald", "rough_diamond", "ancient_alloy", "kethari_crystal", "primordial_geode"}).First(&inv).Error
	require.NoError(t, err, "granted ore present in inventory")
}

func TestResolveBuriedMagnetWithoutMagnet(t *testing.T) {
	svc, _ := newFarmEventService(t)
	evt := &Event{Type: EventBuriedMagnet, ZoneKey: "public"}
	res := svc.ResolveEvent(1, evt, "magnet")
	require.NotNil(t, res)
	assert.Equal(t, "farm.event_buried_no_magnet_title", res.Title)
	assert.Nil(t, res.Items)
}
