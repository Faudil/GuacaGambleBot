package forge

import (
	"os"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	forgesvc "guacagamblebot/internal/service/forge"
	furnituresvc "guacagamblebot/internal/service/furniture"
	housingsvc "guacagamblebot/internal/service/housing"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

func TestMain(m *testing.M) {
	_ = i18n.Load("../../../locales")
	os.Exit(m.Run())
}

func testCog(t *testing.T) (*Cog, *furnituresvc.Service, *housingsvc.Service, *store.Store) {
	t.Helper()
	d := testutil.NewDB(t)
	cfg := &config.Config{StartingBalance: 1000, DailyAmount: 50}
	s := store.New(d, cfg)
	hsvc := housingsvc.New(s, cfg)
	fsvc := furnituresvc.New(s, cfg, hsvc)
	return &Cog{store: s, cfg: cfg, svc: forgesvc.New(s, cfg)}, fsvc, hsvc, s
}

func buyHouseAndForge(t *testing.T, fsvc *furnituresvc.Service, hsvc *housingsvc.Service, s *store.Store, userID int64) {
	t.Helper()
	_, err := s.UpdateBalance(userID, 1000000)
	require.NoError(t, err)
	require.NoError(t, hsvc.BuyHouse(userID, "brick_house"))
	for itemID, qty := range furnituresvc.FurnitureDefs["forge"].CostItems {
		require.NoError(t, s.AddItemRaw(s.DB, userID, itemID, qty))
	}
	require.NoError(t, fsvc.Place(userID, "forge"))
}

func TestMenuShowsStatusAndCounts(t *testing.T) {
	c, fsvc, hsvc, s := testCog(t)
	const uid = 1
	buyHouseAndForge(t, fsvc, hsvc, s, uid)

	_, err := s.CreateEquipment(uid, "stick", "Wooden Stick", "🪵", "common", "weapon",
		1, 2, 0, 0, 0, 0, []byte("[]"), "")
	require.NoError(t, err)

	embed, comps := c.menu("en", uid)
	require.NotNil(t, embed)
	desc := embed.Description

	assert.Contains(t, desc, "✅", "forge status must show the placed forge as ok")
	assert.Contains(t, desc, "Common: 1", "menu must count unequipped commons")
	require.Len(t, comps, 1)
	row, ok := comps[0].(discordgo.ActionsRow)
	require.True(t, ok)
	require.Len(t, row.Components, 2, "menu must offer Fuse and Scrap")
}

func TestMenuShowsLockedForgeWithoutFurniture(t *testing.T) {
	c, _, _, _ := testCog(t)
	const uid = 1

	embed, _ := c.menu("en", uid)
	require.NotNil(t, embed)
	desc := embed.Description
	assert.Contains(t, desc, "❌", "missing forge must be shown as locked")
}

func TestPieceSummaryShowsNameStatsAndAffixes(t *testing.T) {
	c, _, _, s := testCog(t)
	const uid = 1
	eq, err := s.CreateEquipment(uid, "fused_test", "Flaming Longsword of the Dying Star", "⚔️",
		"rare", "weapon", 10, 12, 4, 0, 2, 0,
		[]byte(`[{"id":"of_power","name":"of Power","stat":"str","value":5}]`), "")
	require.NoError(t, err)

	summary := c.pieceSummary(eq, "en")
	assert.Contains(t, summary, "Flaming Longsword of the Dying Star")
	assert.Contains(t, summary, "Lv 10")
	assert.Contains(t, summary, "STR+12")
	assert.Contains(t, summary, "of Power +5")
	assert.True(t, strings.Contains(summary, "Weapon"), "slot name must be shown")
}

func TestRarityNameTranslationsExist(t *testing.T) {
	for _, r := range []items.Rarity{
		items.RarityCommon, items.RarityUncommon, items.RarityRare,
		items.RarityEpic, items.RarityLegendary,
	} {
		name := rarityName(r, "en")
		assert.NotEqual(t, "forge.rarity_"+string(r), name, "english rarity name must be translated")
		nameFR := rarityName(r, "fr")
		assert.NotEqual(t, "forge.rarity_"+string(r), nameFR, "french rarity name must be translated")
	}
}

func TestMenuCountsOnlyUnequipped(t *testing.T) {
	c, fsvc, hsvc, s := testCog(t)
	const uid = 1
	buyHouseAndForge(t, fsvc, hsvc, s, uid)

	eq, err := s.CreateEquipment(uid, "stick", "Wooden Stick", "🪵", "common", "weapon",
		1, 2, 0, 0, 0, 0, []byte("[]"), "")
	require.NoError(t, err)
	require.NoError(t, s.EquipInstance(uid, eq.ID))

	embed, _ := c.menu("en", uid)
	assert.Contains(t, embed.Description, "Common: 0", "equipped items must not count toward fuse/scrap")
}

func TestMenuShowsFusionResearchStatus(t *testing.T) {
	c, fsvc, hsvc, s := testCog(t)
	const uid = 1
	buyHouseAndForge(t, fsvc, hsvc, s, uid)

	embed, _ := c.menu("en", uid)
	assert.Contains(t, embed.Description, "Fusion upgrades")
	assert.Contains(t, embed.Description, "🔬", "unresearched tiers must be marked as researchable")
	assert.Contains(t, embed.Description, "Fusion: Common → Uncommon")

	require.NoError(t, s.DB.Create(&model.UserResearch{
		UserID: uid, ResearchID: "fusion_common", Completed: true,
	}).Error)

	embed, _ = c.menu("en", uid)
	assert.Contains(t, embed.Description, "✅", "researched tier must be marked as unlocked")
}

func TestFusionStatusMarker(t *testing.T) {
	assert.Equal(t, "✅", fusionStatusMarker(nil))
	assert.Equal(t, "🔬", fusionStatusMarker(forgesvc.ErrResearchRequired))
	assert.Equal(t, "❌", fusionStatusMarker(forgesvc.ErrNoForge))
}
