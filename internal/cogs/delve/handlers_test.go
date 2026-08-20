package delve

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	delvesvc "guacagamblebot/internal/service/delve"
	"guacagamblebot/internal/store"
)

// merchantRT is a mock HTTP transport that records every interaction response
// body so the merchant flow can be asserted offline.
type merchantRT struct {
	mu     sync.Mutex
	bodies []string
}

func (r *merchantRT) RoundTrip(req *http.Request) (*http.Response, error) {
	body, _ := io.ReadAll(req.Body)
	r.mu.Lock()
	r.bodies = append(r.bodies, string(body))
	r.mu.Unlock()
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"id":"1"}`)),
		Header:     make(http.Header),
	}, nil
}

func (r *merchantRT) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.bodies...)
}

const merchantTestUser int64 = 7

func merchantTestCog(t *testing.T) (*Cog, *interaction.Bot, *merchantRT) {
	t.Helper()
	require.NoError(t, i18n.Load("../../../locales"))
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "delve.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{}
	st := store.New(d, cfg)
	c := &Cog{
		store:          st,
		cfg:            cfg,
		svc:            delvesvc.New(st, cfg),
		sessions:       make(map[int64]*model.DelveSession),
		merchantOffers: make(map[int64][]delvesvc.DelveItem),
		merchantExtra:  make(map[int64]map[string]int),
		riddles:        make(map[int64]riddleEntry),
	}
	sess, err := discordgo.New("test")
	require.NoError(t, err)
	rt := &merchantRT{}
	sess.Client = &http.Client{Transport: rt}
	return c, &interaction.Bot{Session: interaction.NewDeferringSession(sess)}, rt
}

func merchantBuyInteraction(customID string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:        "1",
		Token:     "tok",
		GuildID:   "1",
		ChannelID: "1",
		Member:    &discordgo.Member{User: &discordgo.User{ID: "7"}},
		Type:      discordgo.InteractionMessageComponent,
		Data:      discordgo.MessageComponentInteractionData{CustomID: customID},
	}}
}

func merchantOfferCommon() []delvesvc.DelveItem {
	return []delvesvc.DelveItem{
		{ID: "delve_test_common", Name: "Test Common", Emoji: "⚔️", Rarity: delvesvc.Common, Quantity: 1},
	}
}

// TestMerchantBuyChargesExactlyOnce is the regression test for the merchant
// that "does not respond": gold between 1x and 2x the listed price must buy
// the item for exactly the listed price, not be rejected by a duplicate
// gold check or charged twice.
func TestMerchantBuyChargesExactlyOnce(t *testing.T) {
	c, b, rt := merchantTestCog(t)
	s, err := c.svc.StartSession(merchantTestUser, 1, 1)
	require.NoError(t, err)
	// Floor 1 Common item: (0+1) * (25+5*1) = 30. 45 gold is < 2x price, which
	// used to make the duplicate charge check reject the purchase outright.
	s.Gold = 45
	c.saveSession(s)
	c.merchantOffers[merchantTestUser] = merchantOfferCommon()

	c.onMerchantBuy(b, merchantBuyInteraction("delve::merchant_buy::0"))

	require.Len(t, rt.snapshot(), 1, "a purchase must produce exactly one response")
	got, err := c.svc.GetSession(merchantTestUser)
	require.NoError(t, err)
	assert.Equal(t, 45-30, got.Gold, "gold must be charged exactly once")
	inv := c.svc.GetInventory(got)
	require.Len(t, inv, 1, "the bought item must be persisted")
	assert.Equal(t, "delve_test_common", inv[0].ID)
}

func TestMerchantBuyRejectsWhenGoldTooLow(t *testing.T) {
	c, b, rt := merchantTestCog(t)
	s, err := c.svc.StartSession(merchantTestUser, 1, 1)
	require.NoError(t, err)
	s.Gold = 10
	c.saveSession(s)
	c.merchantOffers[merchantTestUser] = merchantOfferCommon()

	c.onMerchantBuy(b, merchantBuyInteraction("delve::merchant_buy::0"))

	require.Len(t, rt.snapshot(), 1, "a rejected purchase must still answer the interaction")
	got, err := c.svc.GetSession(merchantTestUser)
	require.NoError(t, err)
	assert.Equal(t, 10, got.Gold, "a rejected purchase must not charge gold")
	assert.Empty(t, c.svc.GetInventory(got), "a rejected purchase must not grant the item")
}

func TestMerchantBuyAppliesHaggleDiscount(t *testing.T) {
	c, b, rt := merchantTestCog(t)
	s, err := c.svc.StartSession(merchantTestUser, 1, 1)
	require.NoError(t, err)
	s.Gold = 21
	c.saveSession(s)
	c.merchantOffers[merchantTestUser] = merchantOfferCommon()
	c.merchantExtra[merchantTestUser] = map[string]int{"haggle_discount": 1}

	c.onMerchantBuy(b, merchantBuyInteraction("delve::merchant_buy::0"))

	require.Len(t, rt.snapshot(), 1)
	got, err := c.svc.GetSession(merchantTestUser)
	require.NoError(t, err)
	assert.Equal(t, 21-30*70/100, got.Gold, "the haggle discount must apply to the single charge")
	inv := c.svc.GetInventory(got)
	require.Len(t, inv, 1)
	assert.Equal(t, "delve_test_common", inv[0].ID)
}

// TestMerchantHaggleKeepsShopOpen ensures haggling re-renders the shop instead
// of replacing it with the floor transition, so the discount/markup can be spent.
func TestMerchantHaggleKeepsShopOpen(t *testing.T) {
	c, b, rt := merchantTestCog(t)
	s, err := c.svc.StartSession(merchantTestUser, 1, 1)
	require.NoError(t, err)
	s.Gold = 100
	c.saveSession(s)

	c.onMerchantHaggle(b, merchantBuyInteraction("delve::merchant_haggle"))

	bodies := rt.snapshot()
	require.Len(t, bodies, 1)
	assert.Contains(t, bodies[0], "delve::merchant_buy::0", "the shop must stay open with buy buttons after haggling")
	got, err := c.svc.GetSession(merchantTestUser)
	require.NoError(t, err)
	assert.Equal(t, 100, got.Gold, "haggling must not charge gold")
}

// TestPuzzleSolveOpensValidModal is the regression test for the Solve button
// failing: the modal payload must respect Discord's limits (label <= 45 chars,
// placeholder <= 100 chars) or Discord rejects the whole response.
func TestPuzzleSolveOpensValidModal(t *testing.T) {
	c, b, rt := merchantTestCog(t)
	s, err := c.svc.StartSession(merchantTestUser, 1, 1)
	require.NoError(t, err)
	c.saveSession(s)

	c.onPuzzleSolve(b, merchantBuyInteraction("delve::puzzle_solve"))

	bodies := rt.snapshot()
	require.Len(t, bodies, 1, "Solve must produce exactly one response")
	var modal struct {
		Type int `json:"type"`
		Data struct {
			CustomID   string `json:"custom_id"`
			Title      string `json:"title"`
			Components []struct {
				Components []struct {
					Label       string `json:"label"`
					Placeholder string `json:"placeholder"`
				} `json:"components"`
			} `json:"components"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(bodies[0]), &modal))
	assert.Equal(t, int(discordgo.InteractionResponseModal), modal.Type, "Solve must open a modal")
	assert.Equal(t, "delve::puzzle_answer", modal.Data.CustomID)
	assert.NotEmpty(t, modal.Data.Title)
	require.Len(t, modal.Data.Components, 1)
	require.Len(t, modal.Data.Components[0].Components, 1)
	input := modal.Data.Components[0].Components[0]
	require.NotEmpty(t, input.Label, "the input label must not be a missing locale key")
	assert.LessOrEqual(t, len(input.Label), 45, "Discord caps text input labels at 45 characters")
	require.NotEmpty(t, input.Placeholder, "the question must not be a missing locale key")
	assert.LessOrEqual(t, len(input.Placeholder), 100, "Discord caps text input placeholders at 100 characters")
	assert.False(t, strings.HasPrefix(input.Label, "delve."), "a missing label key must not leak into the modal")
	assert.False(t, strings.HasPrefix(input.Placeholder, "delve."), "a missing question key must not leak into the modal")
}

// TestRiddleQuestionsFitModalLimits checks every riddle in both locales stays
// within Discord's modal limits, so any future locale edit is caught here.
func TestRiddleQuestionsFitModalLimits(t *testing.T) {
	require.NoError(t, i18n.Load("../../../locales"))
	for _, lang := range []string{"en", "fr"} {
		label := i18n.T("delve.riddle.modal_label", lang)
		require.NotEqual(t, "delve.riddle.modal_label", label, "modal label must exist in %s", lang)
		assert.LessOrEqual(t, len(label), 45, "modal label in %s must fit Discord's label limit", lang)
		for _, id := range riddlePool {
			q := i18n.T("delve.riddle."+id+".question", lang)
			require.NotEqual(t, "delve.riddle."+id+".question", q, "question for %q must exist in %s", id, lang)
			assert.LessOrEqual(t, len(q), 100, "question %q in %s must fit Discord's placeholder limit", id, lang)
		}
	}
}

// TestFloorLeaveShowsRunSummary ensures leaving the delve replies with a run
// summary (stats, loot, deeds) and ends the session.
func TestFloorLeaveShowsRunSummary(t *testing.T) {
	c, b, rt := merchantTestCog(t)
	s, err := c.svc.StartSession(merchantTestUser, 1, 1)
	require.NoError(t, err)
	s.Gold = 120
	s.RoomsCleared = 3
	s.Torches = 1
	s.Keys = 2
	s.Potions = 0
	c.svc.AddItem(s, merchantOfferCommon()[0])
	c.svc.AddFlag(s, "solved_riddle")
	c.saveSession(s)
	c.store.SetCooldown(merchantTestUser, "journal_ambient")

	c.onFloorLeave(b, merchantBuyInteraction("delve::floor_leave"))

	bodies := rt.snapshot()
	require.Len(t, bodies, 1, "leaving must produce exactly one response")
	lang := c.store.GetLanguage(1)
	assert.Contains(t, bodies[0], i18n.T("delve.summary.title", lang), "the reply must use the summary title")
	assert.Contains(t, bodies[0], i18n.T("delve.summary.gold", lang, map[string]any{"gold": "120"}), "the summary must show collected gold")
	assert.Contains(t, bodies[0], "Test Common", "the summary must list the looted item")
	assert.Contains(t, bodies[0], i18n.T("delve.flags.solved_riddle", lang), "the summary must list the riddle deed")
	got, err := c.svc.GetSession(merchantTestUser)
	require.NoError(t, err)
	assert.Nil(t, got, "the delve session must be ended after leaving")
}

// TestVaultKeyShowsRunSummary ensures the tutorial vault-key ending also
// carries the run summary.
func TestVaultKeyShowsRunSummary(t *testing.T) {
	c, b, rt := merchantTestCog(t)
	s, err := c.svc.StartSession(merchantTestUser, 1, 1)
	require.NoError(t, err)
	s.Gold = 60
	s.RoomsCleared = 1
	c.svc.AddItem(s, merchantOfferCommon()[0])
	c.saveSession(s)
	c.store.SetCooldown(merchantTestUser, "journal_ambient")

	c.onKeyTake(b, merchantBuyInteraction("delve::key_take"))

	bodies := rt.snapshot()
	require.Len(t, bodies, 1)
	lang := c.store.GetLanguage(1)
	assert.Contains(t, bodies[0], i18n.T("delve.summary.gold", lang, map[string]any{"gold": "60"}))
	assert.Contains(t, bodies[0], "Test Common")
	got, err := c.svc.GetSession(merchantTestUser)
	require.NoError(t, err)
	assert.Nil(t, got, "the delve session must be ended after taking the vault key")
}

// TestDeathEmbedsRunSummary ensures the fallen embed carries the run summary
// (kept loot and stats) and the session is marked fallen.
func TestDeathEmbedsRunSummary(t *testing.T) {
	c, b, rt := merchantTestCog(t)
	s, err := c.svc.StartSession(merchantTestUser, 1, 1)
	require.NoError(t, err)
	s.Gold = 100
	s.HP = 0
	s.RoomsCleared = 2
	soulbound := merchantOfferCommon()[0]
	soulbound.IsSoulbound = true
	c.svc.AddItem(s, soulbound)
	c.saveSession(s)
	lang := c.store.GetLanguage(1)

	c.applyFallenPenalties(b, merchantBuyInteraction("delve::combat_flee"), s, merchantTestUser, lang)

	bodies := rt.snapshot()
	require.Len(t, bodies, 1)
	assert.Contains(t, bodies[0], i18n.T("delve.handler.death_fallen_title", lang), "the death embed must keep its fallen title")
	assert.Contains(t, bodies[0], "Test Common", "the summary must list the kept loot")
	assert.Contains(t, bodies[0], i18n.T("delve.summary.title", lang), "the death embed must carry the run summary")
	got, err := c.svc.GetSession(merchantTestUser)
	require.NoError(t, err)
	assert.Equal(t, "fallen", got.Status, "the session must be marked fallen")
}

// TestGrantRunLoot covers the three non-equipment reward kinds (gold, heal,
// misc item) applied by the room loot helper.
func TestGrantRunLoot(t *testing.T) {
	c, _, _ := merchantTestCog(t)
	s, err := c.svc.StartSession(merchantTestUser, 1, 1)
	require.NoError(t, err)
	lang := c.store.GetLanguage(1)

	t.Run("gold", func(t *testing.T) {
		s.Gold = 10
		text := c.grantRunLoot(s, &delvesvc.LootResult{Gold: 42}, lang)
		assert.Equal(t, 10+42, s.Gold, "gold finds must credit the run's gold")
		assert.Equal(t, i18n.T("delve.loot.gold_found", lang, map[string]any{"gold": "42"}), text)
		assert.Empty(t, c.svc.GetInventory(s))
	})

	t.Run("heal", func(t *testing.T) {
		s.HP = 50
		text := c.grantRunLoot(s, &delvesvc.LootResult{Heal: 40}, lang)
		assert.Equal(t, 90, s.HP, "heals must restore HP")
		assert.Equal(t, i18n.T("delve.loot.heal_found", lang, map[string]any{"hp": "40"}), text)
	})

	t.Run("heal capped at max", func(t *testing.T) {
		s.HP = s.MaxHP - 5
		c.grantRunLoot(s, &delvesvc.LootResult{Heal: 40}, lang)
		assert.Equal(t, s.MaxHP, s.HP, "heals must not exceed max HP")
	})

	t.Run("misc item", func(t *testing.T) {
		s.Inventory = "[]"
		item := delvesvc.DelveItem{ID: "depth_shard", Name: "Depth Shard", Emoji: "💎", Rarity: delvesvc.Rare, Quantity: 3}
		text := c.grantRunLoot(s, &delvesvc.LootResult{Item: item}, lang)
		inv := c.svc.GetInventory(s)
		require.Len(t, inv, 1)
		assert.Equal(t, "depth_shard", inv[0].ID)
		assert.Equal(t, 3, inv[0].Quantity)
		assert.Contains(t, text, delvesvc.DelveItemName(item, lang))
	})

	t.Run("nil", func(t *testing.T) {
		assert.Equal(t, "", c.grantRunLoot(s, nil, lang))
	})
}
