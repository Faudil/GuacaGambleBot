package expedition

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/model"
	expeditionsvc "guacagamblebot/internal/service/expedition"
	petsvc "guacagamblebot/internal/service/pets"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
)

func TestMain(m *testing.M) {
	_ = i18n.Load("../../../locales")
	os.Exit(m.Run())
}

func testCog(t *testing.T) *Cog {
	t.Helper()
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "expedition_cog.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{}
	s := store.New(d, cfg)
	return &Cog{
		store: s,
		cfg:   cfg,
		svc:   expeditionsvc.New(s, cfg),
		psvc:  petsvc.New(s, cfg),
		qsvc:  questssvc.New(s, cfg),
	}
}

func createPet(t *testing.T, s *store.Store, hp int) *model.UserPet {
	t.Helper()
	pet := &model.UserPet{
		UserID:   1,
		PetType:  "Chien",
		Nickname: "Rex",
		Level:    7,
		MaxHP:    100,
		HP:       hp,
		IsActive: true,
	}
	require.NoError(t, s.DB.Create(pet).Error)
	return pet
}

// durationButtons extracts the start buttons from the menu action row.
func durationButtons(t *testing.T, comps []discordgo.MessageComponent) []discordgo.Button {
	t.Helper()
	require.Len(t, comps, 1)
	row, ok := comps[0].(discordgo.ActionsRow)
	require.True(t, ok)
	require.Len(t, row.Components, len(durationOptions))
	buttons := make([]discordgo.Button, 0, len(row.Components))
	for _, comp := range row.Components {
		btn, ok := comp.(discordgo.Button)
		require.True(t, ok)
		buttons = append(buttons, btn)
	}
	return buttons
}

func TestMenuShowsDurationButtons(t *testing.T) {
	cog := testCog(t)
	createPet(t, cog.store, 100)

	embed, comps := cog.menuView("en", 1)
	assert.NotEmpty(t, embed.Description)
	assert.Contains(t, embed.Description, "Rex")

	buttons := durationButtons(t, comps)
	for i, btn := range buttons {
		require.NotEmpty(t, btn.Label, "duration button %d must be labeled", i)
		assert.False(t, btn.Disabled, "duration buttons must be enabled for a healthy pet")
		assert.Equal(t, discordgo.PrimaryButton, btn.Style)
		_, _, rest := componentsDecode(btn.CustomID)
		require.Len(t, rest, 1, "start button must carry the hour count")
		require.NotEqual(t, 0, durationHours(rest), "start button %q must carry a valid duration", btn.Label)
	}
}

func TestMenuDisabledWhenPetKO(t *testing.T) {
	cog := testCog(t)
	pet := createPet(t, cog.store, 100)
	// KO pets are persisted via column updates (GORM skips zero values on
	// CREATE because of the hp default).
	require.NoError(t, cog.store.DB.Model(&model.UserPet{}).Where("id = ?", pet.ID).Update("hp", 0).Error)

	embed, comps := cog.menuView("en", 1)
	assert.Contains(t, embed.Description, "K.O.")
	for _, btn := range durationButtons(t, comps) {
		assert.True(t, btn.Disabled, "duration buttons must be disabled while the pet is K.O.")
	}
}

func TestMenuRoutesToStatusWhenActive(t *testing.T) {
	cog := testCog(t)
	pet := createPet(t, cog.store, 80)
	now := time.Now()
	require.NoError(t, cog.store.DB.Create(&model.PetExpedition{
		UserID:    1,
		PetID:     pet.ID,
		StartTime: now.Add(-30 * time.Minute),
		EndTime:   now.Add(90 * time.Minute),
		RewardXP:  250,
		Log:       `[{"time":15,"type":"exploration","text":"fallback","location":"forest","xp":120}]`,
	}).Error)

	embed, comps := cog.menuView("en", 1)
	assert.Contains(t, embed.Title, "Rex")
	assert.Contains(t, embed.Description, "%")

	require.Len(t, comps, 1)
	row := comps[0].(discordgo.ActionsRow)
	require.Len(t, row.Components, 2)
	refresh, claim := row.Components[0].(discordgo.Button), row.Components[1].(discordgo.Button)
	assert.False(t, refresh.Disabled, "refresh must stay enabled while the expedition is running")
	assert.True(t, claim.Disabled, "claim must be disabled while the expedition is running")
	assert.Equal(t, "expedition::menu", splitCustomID(refresh.CustomID))
}

func TestClaimEnabledWhenFinished(t *testing.T) {
	cog := testCog(t)
	pet := createPet(t, cog.store, 80)
	now := time.Now()
	require.NoError(t, cog.store.DB.Create(&model.PetExpedition{
		UserID:    1,
		PetID:     pet.ID,
		StartTime: now.Add(-3 * time.Hour),
		EndTime:   now.Add(-1 * time.Minute),
		RewardXP:  250,
		Log:       `[]`,
	}).Error)

	_, comps := cog.menuView("en", 1)
	row := comps[0].(discordgo.ActionsRow)
	claim := row.Components[1].(discordgo.Button)
	assert.False(t, claim.Disabled, "claim must be enabled once the expedition is over")
}

func TestClaimViewOffersMenuBack(t *testing.T) {
	cog := testCog(t)
	pet := createPet(t, cog.store, 100)
	exp := &model.PetExpedition{
		UserID:      1,
		PetID:       pet.ID,
		RewardXP:    250,
		RewardItems: `["pebble","pebble"]`,
	}

	embed, comps := cog.claimView(1, pet, exp, false, false, 0, false, "loot", "en")
	assert.Contains(t, embed.Description, "250")
	assert.Equal(t, i18n.T("expedition.claim_title", "en"), embed.Title)

	require.Len(t, comps, 1)
	row := comps[0].(discordgo.ActionsRow)
	back := row.Components[0].(discordgo.Button)
	assert.Equal(t, "expedition::menu", splitCustomID(back.CustomID))
}

func TestLootStringCountsAndLocalizes(t *testing.T) {
	cog := testCog(t)
	loot := cog.lootString(`["pebble","pebble","iron_ore"]`, "fr")
	assert.Contains(t, loot, "2x")
	assert.Contains(t, loot, "1x")
	assert.NotContains(t, loot, "{")
	assert.NotEmpty(t, cog.lootString(`[]`, "fr"), "empty loot must render a friendly message")
}

func TestAdventureLogLocalizedWithLegacyFallback(t *testing.T) {
	cog := testCog(t)
	pet := createPet(t, cog.store, 100)

	log := `[
		{"time":15,"type":"exploration","text":"legacy text","location":"forest","xp":120},
		{"time":45,"type":"combat","text":"old","enemy":"Renard","enemy_level":6,"combat_result":"win","xp":300},
		{"time":60,"type":"loot","text":"old","item":"pebble"},
		{"time":90,"type":"rest","text":"old"}
	]`
	exp := &model.PetExpedition{UserID: 1, PetID: pet.ID, Log: log}

	// French: structured events must render through i18n.
	fr := cog.adventureLog(exp, pet, "fr", 8)
	assert.Contains(t, fr, "la Forêt Mystique", "exploration must localize the location")
	assert.Contains(t, fr, "Renard", "combat must keep the enemy species name")
	assert.Contains(t, fr, "300", "combat must include the XP")
	assert.NotContains(t, fr, "legacy text", "structured events must not fall back to raw text")

	// Legacy row: only raw text stored, structured fields empty.
	legacy := &model.PetExpedition{UserID: 1, PetID: pet.ID, Log: `[{"time":15,"type":"exploration","text":"raw legacy line","xp":0}]`}
	raw := cog.adventureLog(legacy, pet, "fr", 8)
	assert.Contains(t, raw, "raw legacy line")
}

func TestAdventureLogEmpty(t *testing.T) {
	cog := testCog(t)
	pet := createPet(t, cog.store, 100)
	exp := &model.PetExpedition{UserID: 1, PetID: pet.ID, Log: `[]`}
	assert.NotEmpty(t, cog.adventureLog(exp, pet, "en", 8))
}

// helpers -------------------------------------------------------------

func componentsDecode(customID string) (string, string, []string) {
	parts := strings.Split(customID, "::")
	return parts[0], parts[1], parts[2 : len(parts)-1]
}

func splitCustomID(customID string) string {
	domain, action, _ := componentsDecode(customID)
	return domain + "::" + action
}
