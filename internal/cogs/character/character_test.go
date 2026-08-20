package character

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

func TestMain(m *testing.M) {
	_ = i18n.Load("../../../locales")
	os.Exit(m.Run())
}

func testStore(t *testing.T) *store.Store {
	t.Helper()
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "character_cog.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	return store.New(d, &config.Config{StartingBalance: 100})
}

func mustWeapon(t *testing.T, s *store.Store, userID int64, name, rarity string, minLevel int) model.UserEquipment {
	t.Helper()
	eq, err := s.CreateEquipment(userID, name, name, "⚔️", rarity, "weapon", minLevel, 2, 0, 0, 0, 0, []byte("[]"), "")
	require.NoError(t, err)
	return *eq
}

func TestEquipSelectOptionsCapsAtDiscordLimit(t *testing.T) {
	s := testStore(t)
	var rows []model.UserEquipment
	for i := 0; i < 40; i++ {
		rows = append(rows, mustWeapon(t, s, 1, "Iron Sword "+string(rune('A'+i%26)), "common", 1))
	}

	opts := equipSelectOptions(rows, 1, "en")
	require.Len(t, opts, maxEquipMenuOptions, "select menu must never exceed Discord's 25-option cap")
	assert.Len(t, opts, 25)

	seen := map[string]bool{}
	for _, o := range opts {
		assert.False(t, seen[o.Value], "each option must appear once")
		seen[o.Value] = true
		assert.LessOrEqual(t, len(o.Label), 100, "option label must respect Discord's limit")
	}
}

func TestEquipSelectOptionsUsableFirst(t *testing.T) {
	s := testStore(t)
	locked := mustWeapon(t, s, 1, "Dragon Slayer Sword", "legendary", 20)
	usable := mustWeapon(t, s, 1, "Wooden Stick", "common", 1)
	rows := []model.UserEquipment{locked, usable}

	opts := equipSelectOptions(rows, 1, "en")
	require.Len(t, opts, 2)
	assert.Equal(t, "[common] Wooden Stick", opts[0].Label, "usable weapon must come first even when it is a worse rarity")
	assert.Equal(t, "[legendary] Dragon Slayer Sword", opts[1].Label)
	assert.Contains(t, opts[1].Description, "Lv 20", "locked option must show the required level")
}

func TestEquipSelectOptionsSortsByRarityAndLevel(t *testing.T) {
	s := testStore(t)
	common := mustWeapon(t, s, 1, "Wooden Stick", "common", 1)
	rare := mustWeapon(t, s, 1, "Hunter's Bow", "rare", 10)
	epic := mustWeapon(t, s, 1, "Enchanted Blade", "epic", 15)
	rows := []model.UserEquipment{common, rare, epic}

	opts := equipSelectOptions(rows, 20, "en")
	require.Len(t, opts, 3)
	assert.Equal(t, "[epic] Enchanted Blade", opts[0].Label)
	assert.Equal(t, "[rare] Hunter's Bow", opts[1].Label)
	assert.Equal(t, "[common] Wooden Stick", opts[2].Label)
}

func TestEquipSelectOptionsLabelTruncation(t *testing.T) {
	s := testStore(t)
	long := "Blade of the Endless Sundered Veil Beyond the Reach of Mortal Names"
	eq, err := s.CreateEquipment(1, "long_blade", long, "⚔️", "legendary", "weapon", 20,
		0, 0, 0, 0, 0, []byte("[]"), "")
	require.NoError(t, err)

	opts := equipSelectOptions([]model.UserEquipment{*eq}, 20, "en")
	require.Len(t, opts, 1)
	assert.Equal(t, "[legendary] "+long, opts[0].Label, "short labels keep the rarity prefix")

	veryLong := long + " and the Eternal Winter That Never Truly Ends"
	eq2, err := s.CreateEquipment(1, "long_blade_2", veryLong, "⚔️", "legendary", "weapon", 20,
		0, 0, 0, 0, 0, []byte("[]"), "")
	require.NoError(t, err)
	opts = equipSelectOptions([]model.UserEquipment{*eq2}, 20, "en")
	require.Len(t, opts, 1)
	assert.LessOrEqual(t, len(opts[0].Label), 100, "labels over 100 chars must be truncated")
	assert.Equal(t, 100, len(opts[0].Label))
}

func TestEquipSelectOptionsSetsMenuEmoji(t *testing.T) {
	s := testStore(t)
	eq := mustWeapon(t, s, 1, "Wooden Stick", "common", 1)
	opts := equipSelectOptions([]model.UserEquipment{eq}, 1, "en")
	require.Len(t, opts, 1)
	assert.Equal(t, "⬜", opts[0].Emoji.Name)
	require.IsType(t, discordgo.SelectMenuOption{}, opts[0])
}
