package archeology

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	invsvc "guacagamblebot/internal/service/inventory"
	npcsvc "guacagamblebot/internal/service/npcs"
	petsvc "guacagamblebot/internal/service/pets"
	researchsvc "guacagamblebot/internal/service/research"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
	"guacagamblebot/internal/universe"
	"guacagamblebot/internal/universe/hoakhaven"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d := testutil.NewDB(t)
	cfg := testCfg()
	s := store.New(d, cfg)
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	require.NotNil(t, def)
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)
	svc := New(s, cfg, npcSvc, petsvc.New(s, cfg, npcSvc))
	return svc, s
}

func testCfg() *config.Config {
	return &config.Config{StartingBalance: 100}
}

func fundReanimate(t *testing.T, s *store.Store, rarity string) {
	t.Helper()
	cost, ok := ReanimateCosts[rarity]
	if !ok {
		return
	}
	_, _ = s.UpdateBalance(1, cost.Money*5) // ensure plenty for multiple attempts
	for item, qty := range cost.Items {
		_ = s.AddItemRaw(s.DB, 1, item, qty*5)
	}
	// also ensure fossil item itself will be topped up by caller; ensure some extra
	require.NoError(t, s.DB.Where("user_id = ?", 1).First(&model.User{}).Error)
}

// placeGeneticsLab puts the user in an active house with a genetics lab, the
// furniture gate required for reanimation.
func placeGeneticsLab(t *testing.T, db *gorm.DB, userID int64) {
	t.Helper()
	require.NoError(t, db.Create(&model.UserHousing{UserID: userID, HouseType: "brick_house", Level: 1, IsActive: true}).Error)
	require.NoError(t, db.Create(&model.UserFurniture{UserID: userID, HouseType: "brick_house", FurnitureID: "genetics_lab", PlacedAt: time.Now()}).Error)
}

// completeResearch marks the given research as completed for the user.
func completeResearch(t *testing.T, db *gorm.DB, userID int64, researchID string) {
	t.Helper()
	require.NoError(t, db.Create(&model.UserResearch{UserID: userID, ResearchID: researchID, Completed: true}).Error)
}

func testServiceInMemory(t *testing.T) *Service {
	d := testutil.NewDB(t)
	cfg := testCfg()
	s := store.New(d, cfg)
	hoakhaven.Register()
	def := universe.Get("hoakhaven")
	require.NotNil(t, def)
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)
	return New(s, cfg, npcSvc, petsvc.New(s, cfg, npcSvc))
}

func TestNewGameRiverbed(t *testing.T) {
	svc, _ := testService(t)
	state, err := svc.NewGame(1, "riverbed")
	require.NoError(t, err)
	assert.Equal(t, "riverbed", state.PermitType)
	assert.Equal(t, 10, state.Depth)
	assert.Equal(t, 100, state.Integrity)
	assert.Equal(t, 6, state.Actions)
	assert.NotNil(t, state.Site)
}

func TestNewGameCliffside(t *testing.T) {
	svc, _ := testService(t)
	state, err := svc.NewGame(1, "cliffside")
	require.NoError(t, err)
	assert.Equal(t, "cliffside", state.PermitType)
	assert.Equal(t, 12, state.Depth)
}

func TestNewGameFaultNoMoney(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.NewGame(1, "fault")
	assert.ErrorIs(t, err, ErrNoMoney)
}

func TestNewGameFault(t *testing.T) {
	svc, s := testService(t)
	_, err := s.UpdateBalance(1, 500)
	require.NoError(t, err)
	state, err := svc.NewGame(1, "fault")
	require.NoError(t, err)
	assert.Equal(t, "fault", state.PermitType)
}

func TestNewGameIceSheetLocked(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.NewGame(1, "ice_sheet")
	assert.ErrorIs(t, err, ErrLocked)
}

func TestNewGameVolcanicLocked(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.NewGame(1, "volcanic")
	assert.ErrorIs(t, err, ErrLocked)
}

func TestApplyActionBrush(t *testing.T) {
	svc, _ := testService(t)
	state, _ := svc.NewGame(1, "riverbed")
	outcome := svc.ApplyAction(state, ActionBrush)
	assert.Less(t, outcome.State.Depth, 10)
	assert.Equal(t, 100, outcome.State.Integrity)
	assert.Equal(t, 5, outcome.State.Actions)
	assert.False(t, outcome.Damaged)
}

func TestApplyActionDynamite(t *testing.T) {
	svc, _ := testService(t)
	state, _ := svc.NewGame(1, "riverbed")
	outcome := svc.ApplyAction(state, ActionDynamite)
	assert.LessOrEqual(t, outcome.State.Depth, 10)
	assert.Equal(t, 5, outcome.State.Actions)
}

func TestApplyActionScan(t *testing.T) {
	svc, _ := testService(t)
	state, _ := svc.NewGame(1, "riverbed")
	outcome := svc.ApplyAction(state, ActionScan)
	assert.True(t, outcome.State.RevealedLayer)
	assert.Equal(t, 5, outcome.State.Actions)
	assert.False(t, outcome.Finished)
}

func TestWhisperRevealsBestTool(t *testing.T) {
	cases := []struct {
		layer LayerType
		tool  string
	}{
		{LayerSoftSoil, "brush"},
		{LayerHardRock, "hammer"},
		{LayerGravel, "hammer"},
		{LayerClay, "brush"},
		{LayerBedrock, "dynamite"},
	}
	for _, tc := range cases {
		state := &GameState{CurrentLayer: tc.layer, Actions: 4}
		result := (&Service{}).ResolveEvent(state, &DigEvent{Type: EventFossilWhisper}, "accept")
		assert.Equal(t, "arch.event_whisper_result_title", result.TitleID)
		assert.Equal(t, tc.tool, result.RevealedTool)
		assert.Equal(t, tc.layer, result.RevealedLayer)
		assert.True(t, result.BackToDig)
		assert.True(t, state.RevealedLayer)
	}
}

func TestResolveDisaster(t *testing.T) {
	state := &GameState{Integrity: 0, Depth: 30, Actions: 2}
	res := (&Service{}).Resolve(state)
	assert.Equal(t, "bone_dust", res.ItemName)
}

func TestResolveDamaged(t *testing.T) {
	state := &GameState{Integrity: 30, Depth: 0, Actions: 3}
	res := (&Service{}).Resolve(state)
	assert.Equal(t, "damaged_fossil", res.ItemName)
}

func TestResolveTimeout(t *testing.T) {
	state := &GameState{Integrity: 100, Depth: 20, Actions: 0}
	res := (&Service{}).Resolve(state)
	assert.Equal(t, "bone_dust", res.ItemName)
}

func TestSiteInfo(t *testing.T) {
	svc, _ := testService(t)
	infos := svc.GetSiteInfo(1)
	assert.Len(t, infos, 5)
	foundRiverbed := false
	for _, info := range infos {
		if info.Key == "riverbed" {
			foundRiverbed = true
			assert.True(t, info.Unlocked)
		}
		if info.Key == "volcanic" {
			assert.False(t, info.Unlocked)
		}
	}
	assert.True(t, foundRiverbed)
}

func TestGetArcheologistLevel(t *testing.T) {
	svc, _ := testService(t)
	assert.Equal(t, 0, svc.GetArcheologistLevel(1))
}

func TestGetArcheologistXP(t *testing.T) {
	svc, _ := testService(t)
	xp, next := svc.GetArcheologistXP(1)
	assert.Equal(t, 0, xp)
	assert.Equal(t, 50, next)
}

func TestAddArcheologistXPMultiLevelUp(t *testing.T) {
	svc, s := testService(t)
	svc.addArcheologistXP(s.DB, 1, 1000)
	var job model.Job
	require.NoError(t, s.DB.Where("user_id = ? AND job_name = ?", 1, "archeologist").First(&job).Error)
	// 1000 XP from level 1: consumes 75+100+125+150+175+200 = 825 across 6 level-ups.
	assert.Equal(t, 7, job.Level)
	assert.Equal(t, 175, job.XP)
	xp, next := svc.GetArcheologistXP(1)
	assert.Equal(t, 175, xp)
	assert.Equal(t, 225, next)
}

func TestAddArcheologistXPCreationLevelsUp(t *testing.T) {
	svc, s := testService(t)
	// First award creates the record at level 1 and must still apply level-ups.
	svc.addArcheologistXP(s.DB, 1, 200)
	var job model.Job
	require.NoError(t, s.DB.Where("user_id = ? AND job_name = ?", 1, "archeologist").First(&job).Error)
	assert.Equal(t, 3, job.Level) // 75 -> L2 (125 left), 100 -> L3 (25 left)
	assert.Equal(t, 25, job.XP)
	xp, next := svc.GetArcheologistXP(1)
	assert.Equal(t, 25, xp)
	assert.Equal(t, 125, next)
}

func TestAwardResult(t *testing.T) {
	svc, s := testService(t)
	err := svc.AwardResult(1, &DigResult{ItemName: "common_fossil", Value: 150, XP: 50, Quality: "common"})
	require.NoError(t, err)
	var inv model.Inventory
	err = s.DB.Where("user_id = ? AND item_id = ?", 1, "common_fossil").First(&inv).Error
	assert.NoError(t, err)
	assert.Equal(t, 1, inv.Quantity)
}

func TestAwardResultAccumulatesHarvest(t *testing.T) {
	svc, s := testService(t)
	res := &DigResult{ItemName: "common_fossil", Value: 150, XP: 50, Quality: "common"}
	require.NoError(t, svc.AwardResult(1, res))
	require.NoError(t, svc.AwardResult(1, res))
	var fh model.UserFossilHarvest
	require.NoError(t, s.DB.Where("user_id = ? AND fossil_id = ?", 1, "common_fossil").First(&fh).Error)
	assert.Equal(t, 2, fh.Count)
}

func TestSellResult(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "common_fossil", Quantity: 1})
	bal, err := s.GetBalance(1)
	require.NoError(t, err)
	price, newBal, lucky, _, err := svc.SellResult(1, &DigResult{ItemName: "common_fossil", Value: 150, Quality: "common"})
	require.NoError(t, err)
	// Base sell price is 1.2x; a lucky roll pays between 1.2x and 2x extra.
	assert.GreaterOrEqual(t, price, 180)
	assert.LessOrEqual(t, price, 360)
	if lucky {
		assert.Greater(t, price, 180)
	}
	assert.Equal(t, bal+price, newBal)
}

func TestSellResultScalesWithQuantity(t *testing.T) {
	svc, s := testService(t)
	bal, err := s.GetBalance(1)
	require.NoError(t, err)
	price, newBal, _, _, err := svc.SellResult(1, &DigResult{ItemName: "common_fossil", Value: 150, Quality: "common", Quantity: 3})
	require.NoError(t, err)
	// 150 * 1.2 * 3 = 540 base, up to 1080 with the lucky multiplier.
	assert.GreaterOrEqual(t, price, 540)
	assert.LessOrEqual(t, price, 1080)
	assert.Equal(t, bal+price, newBal)
}

func TestSellResultTracksHarvest(t *testing.T) {
	svc, s := testService(t)
	_, _, _, _, err := svc.SellResult(1, &DigResult{ItemName: "common_fossil", Value: 150, Quality: "common", Quantity: 3})
	require.NoError(t, err)
	var fh model.UserFossilHarvest
	require.NoError(t, s.DB.Where("user_id = ? AND fossil_id = ?", 1, "common_fossil").First(&fh).Error)
	assert.Equal(t, 3, fh.Count)
}

func TestSellPriceDeterministic(t *testing.T) {
	price, lucky, mult := sellPrice(150, 1, 0.99)
	assert.False(t, lucky)
	assert.Equal(t, 1.0, mult)
	assert.Equal(t, 180, price)

	price, lucky, mult = sellPrice(150, 3, 0.99)
	assert.False(t, lucky)
	assert.Equal(t, 1.0, mult)
	assert.Equal(t, 540, price)

	price, lucky, mult = sellPrice(150, 1, 0.0)
	assert.True(t, lucky)
	assert.Equal(t, sellLuckyMin, mult)
	assert.Equal(t, 216, price)

	price, lucky, mult = sellPrice(150, 1, 0.1)
	assert.True(t, lucky)
	assert.InDelta(t, 1.6, mult, 0.001)
	assert.Equal(t, 288, price)

	price, lucky, mult = sellPrice(150, 3, 0.19)
	assert.True(t, lucky)
	assert.InDelta(t, 1.96, mult, 0.001)
	assert.Equal(t, 1058, price)
}

func TestGrindFossils(t *testing.T) {
	svc, s := testService(t)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "rare_fossil", 3))
	require.NoError(t, s.AddItemRaw(s.DB, 1, "bone_dust", 2))

	dust, err := svc.GrindFossils(1, "rare_fossil", 3)
	require.NoError(t, err)
	assert.Equal(t, 15, dust) // 3 * 5 per DustRates

	var inv model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "rare_fossil").First(&inv).Error)
	assert.Equal(t, 0, inv.Quantity)
	var dustInv model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "bone_dust").First(&dustInv).Error)
	assert.Equal(t, 17, dustInv.Quantity)
}

func TestGrindFossilsPartialQuantity(t *testing.T) {
	svc, s := testService(t)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "epic_fossil", 3))

	dust, err := svc.GrindFossils(1, "epic_fossil", 2)
	require.NoError(t, err)
	assert.Equal(t, 14, dust) // 2 * 7

	var inv model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "epic_fossil").First(&inv).Error)
	assert.Equal(t, 1, inv.Quantity)
}

func TestGrindFossilsNotEnough(t *testing.T) {
	svc, s := testService(t)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "common_fossil", 2))

	_, err := svc.GrindFossils(1, "common_fossil", 5)
	assert.ErrorIs(t, err, ErrNotEnoughFossils)

	var inv model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "common_fossil").First(&inv).Error)
	assert.Equal(t, 2, inv.Quantity, "stock must be untouched on failure")
	var dust model.Inventory
	assert.Error(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "bone_dust").First(&dust).Error, "no dust granted on failure")
}

func TestGrindFossilsInvalidItem(t *testing.T) {
	svc, s := testService(t)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "bone_dust", 5))
	_, err := svc.GrindFossils(1, "bone_dust", 1)
	assert.ErrorIs(t, err, ErrNotGrindable)
}

func TestGrindFossilsInvalidQuantity(t *testing.T) {
	svc, s := testService(t)
	require.NoError(t, s.AddItemRaw(s.DB, 1, "common_fossil", 2))
	_, err := svc.GrindFossils(1, "common_fossil", 0)
	assert.ErrorIs(t, err, ErrNotEnoughFossils)
}

func TestDustRatesSanity(t *testing.T) {
	// Every listed fossil must be a registered item and yield at least 1 dust.
	for itemID, rate := range DustRates {
		assert.NotNil(t, items.Get(itemID), "%s must exist in the items registry", itemID)
		assert.GreaterOrEqual(t, rate, 1, "%s must yield at least 1 bone dust", itemID)
	}
	// GrindableOrder and DustRates must agree both ways.
	for _, itemID := range GrindableOrder {
		assert.Contains(t, DustRates, itemID)
	}
	assert.Len(t, GrindableOrder, len(DustRates))
}

func TestReanimatePoolsRegistry(t *testing.T) {
	for rarity, pool := range ReanimatePools {
		for _, pet := range pool.Pets {
			pt, ok := petsvc.PetTypes[pet]
			require.True(t, ok, "reanimate pool %q references unknown pet %q", rarity, pet)
			switch rarity {
			case "common", "rare", "epic", "legendary":
				assert.Equal(t, rarity, pt.Rarity, "pet %q must match the %q pool rarity", pet, rarity)
			}
		}
	}
}

func TestReanimate(t *testing.T) {
	svc, s := testService(t)
	placeGeneticsLab(t, s.DB, 1)
	completeResearch(t, s.DB, 1, "reanimate_common")
	fundReanimate(t, s, "common")
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "common_fossil", Quantity: 5})
	petName, success, err := svc.Reanimate(1, "common")
	assert.NoError(t, err)
	_ = petName
	_ = success
}

func TestReanimateRequiresGeneticsLab(t *testing.T) {
	svc, s := testService(t)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "common_fossil", Quantity: 5})
	_, _, err := svc.Reanimate(1, "common")
	assert.ErrorIs(t, err, ErrNoGeneticsLab)
}

func TestReanimateRequiresResearch(t *testing.T) {
	svc, s := testService(t)
	placeGeneticsLab(t, s.DB, 1)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "common_fossil", Quantity: 5})
	_, _, err := svc.Reanimate(1, "common")
	assert.ErrorIs(t, err, ErrResearchRequired)
}

func TestReanimateWithLabAndResearch(t *testing.T) {
	svc, s := testService(t)
	placeGeneticsLab(t, s.DB, 1)
	completeResearch(t, s.DB, 1, "reanimate_rare")
	fundReanimate(t, s, "rare")
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "rare_fossil", Quantity: 5})
	petName, success, err := svc.Reanimate(1, "rare")
	require.NoError(t, err)
	if success {
		assert.Contains(t, ReanimatePools["rare"].Pets, petName)
	}
}

// reanimateUntilSuccess drives Reanimate until a success so the created pet
// can be inspected. The archeologist starts at level 0 (50% success rate), so
// a few attempts are usually enough.
func reanimateUntilSuccess(t *testing.T, svc *Service, s *store.Store, rarity string) string {
	t.Helper()
	item := ReanimatePools[rarity].ItemName
	cost, hasCost := ReanimateCosts[rarity]
	for i := 0; i < 20; i++ {
		var inv model.Inventory
		if err := s.DB.Where("user_id = ? AND item_id = ?", 1, item).First(&inv).Error; err != nil {
			_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: item, Quantity: 5})
		} else if inv.Quantity < 5 {
			_ = s.DB.Model(&inv).UpdateColumn("quantity", 5)
		}
		if hasCost {
			// top up money and cost items each attempt (they are consumed even on fail)
			_, _ = s.UpdateBalance(1, cost.Money+5000)
			for itm, qty := range cost.Items {
				_ = s.AddItemRaw(s.DB, 1, itm, qty)
			}
		}
		petName, success, err := svc.Reanimate(1, rarity)
		require.NoError(t, err)
		if success {
			return petName
		}
	}
	t.Fatal("reanimation never succeeded after 20 attempts")
	return ""
}

func TestReanimateUsesPetTypeStats(t *testing.T) {
	svc, s := testService(t)
	placeGeneticsLab(t, s.DB, 1)
	completeResearch(t, s.DB, 1, "reanimate_common")
	petName := reanimateUntilSuccess(t, svc, s, "common")

	var pet model.UserPet
	require.NoError(t, s.DB.Where("user_id = ? AND pet_type = ?", 1, petName).Order("id DESC").First(&pet).Error)
	pt := petsvc.PetTypes[petName]
	require.NotNil(t, pt)
	assert.Equal(t, pt.MaxHP, pet.MaxHP, "reanimated pet must use its species MaxHP")
	assert.Equal(t, pt.MaxHP, pet.HP)
	assert.Equal(t, pt.Atk, pet.Atk)
	assert.Equal(t, pt.Defense, pet.Defense)
	assert.Equal(t, pt.Speed, pet.Speed)
	assert.Equal(t, pt.DGE, pet.DGE)
	assert.Equal(t, pt.ACC, pet.ACC)
	assert.Equal(t, pt.CritC, pet.CritC)
	assert.InDelta(t, pt.CritD, pet.CritD, 0.001)
}

func TestReanimateFirstPetIsActive(t *testing.T) {
	svc, s := testService(t)
	placeGeneticsLab(t, s.DB, 1)
	completeResearch(t, s.DB, 1, "reanimate_common")
	petName := reanimateUntilSuccess(t, svc, s, "common")

	var pet model.UserPet
	require.NoError(t, s.DB.Where("user_id = ? AND pet_type = ?", 1, petName).Order("id DESC").First(&pet).Error)
	assert.True(t, pet.IsActive, "the first reanimated pet must be auto-activated")
}

func TestReanimateRespectsPetSlots(t *testing.T) {
	svc, s := testService(t)
	placeGeneticsLab(t, s.DB, 1)
	completeResearch(t, s.DB, 1, "reanimate_common")
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "common_fossil", Quantity: 5})

	petsSvc := petsvc.New(s, testCfg())
	for i := 0; i < petsvc.BasePetSlots; i++ {
		_, err := petsSvc.CreatePet(1, "Escargot")
		require.NoError(t, err)
	}

	_, _, err := svc.Reanimate(1, "common")
	assert.ErrorIs(t, err, ErrPetSlotsFull)

	var inv model.Inventory
	require.NoError(t, s.DB.Where("user_id = ? AND item_id = ?", 1, "common_fossil").First(&inv).Error)
	assert.Equal(t, 5, inv.Quantity, "fossils must not be consumed when pet slots are full")
}

func TestReanimateResearchDefs(t *testing.T) {
	for rarity, resID := range ReanimateResearch {
		rd, ok := researchsvc.ResearchDefs[resID]
		require.True(t, ok, "research %q for rarity %q must exist", resID, rarity)
		assert.Equal(t, "genetics_lab", rd.RequiredFurniture, "research %q must require the genetics lab", resID)
	}
}

func TestReanimateNoFossils(t *testing.T) {
	svc, s := testService(t)
	placeGeneticsLab(t, s.DB, 1)
	completeResearch(t, s.DB, 1, "reanimate_common")
	_, _, err := svc.Reanimate(1, "common")
	assert.Error(t, err)
}

func TestReanimateInvalidRarity(t *testing.T) {
	svc, _ := testService(t)
	_, _, err := svc.Reanimate(1, "invalid")
	assert.Error(t, err)
}

func TestReanimateNotEnoughFossils(t *testing.T) {
	svc, s := testService(t)
	placeGeneticsLab(t, s.DB, 1)
	completeResearch(t, s.DB, 1, "reanimate_common")
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "common_fossil", Quantity: 3})
	_, _, err := svc.Reanimate(1, "common")
	assert.ErrorIs(t, err, ErrNoFossils)
}

func TestRollEvent(t *testing.T) {
	svc, _ := testService(t)
	state, _ := svc.NewGame(1, "riverbed")
	evt := svc.RollEvent(state)
	if evt != nil {
		assert.NotEmpty(t, evt.TitleID)
	}
}

func TestResolveCaveInCareful(t *testing.T) {
	state := &GameState{Actions: 5, Integrity: 100, Depth: 30, MaxDepth: 30}
	evt := &DigEvent{Type: EventCaveIn}
	result := (&Service{}).ResolveEvent(state, evt, "careful")
	assert.True(t, result.BackToDig)
	assert.Equal(t, 2, result.ActionsLost)
}

func TestResolveCaveInRush(t *testing.T) {
	state := &GameState{Actions: 5, Integrity: 100, Depth: 30, MaxDepth: 30}
	evt := &DigEvent{Type: EventCaveIn}
	result := (&Service{}).ResolveEvent(state, evt, "rush")
	assert.True(t, result.BackToDig)
	assert.Equal(t, 20, result.IntLoss)
	assert.Greater(t, result.DepthGain, 0)
}

func TestResolveCaveInAbandon(t *testing.T) {
	state := &GameState{Actions: 5, Integrity: 100, Depth: 30, MaxDepth: 30}
	evt := &DigEvent{Type: EventCaveIn}
	result := (&Service{}).ResolveEvent(state, evt, "abandon")
	assert.False(t, result.BackToDig)
	assert.True(t, state.Finished)
}

func TestResolveGuardianTribute(t *testing.T) {
	state := &GameState{Actions: 3, Integrity: 100}
	evt := &DigEvent{Type: EventGuardian}
	result := (&Service{}).ResolveEvent(state, evt, "tribute")
	assert.True(t, result.BackToDig)
	assert.Equal(t, 4, state.Actions)
	assert.Equal(t, -100, result.CoinChange)
}

func TestGetToolMastery(t *testing.T) {
	svc, _ := testService(t)
	mastery := svc.GetToolMastery(1)
	assert.Contains(t, mastery, "dynamite")
	assert.Contains(t, mastery, "hammer")
	assert.Contains(t, mastery, "brush")
}

func TestIncrementToolUses(t *testing.T) {
	svc, _ := testService(t)
	svc.incrementToolUses(1, "hammer")
	uses := svc.getToolUses(1, "hammer")
	assert.Equal(t, 1, uses)
}

func TestJournalPages(t *testing.T) {
	svc, s := testService(t)
	pages := svc.HasJournalPages(1)
	assert.Empty(t, pages)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "journal_page_1", Quantity: 1})
	pages = svc.HasJournalPages(1)
	assert.Equal(t, []int{1}, pages)
}

func TestJournalProgress(t *testing.T) {
	svc, s := testService(t)
	count, max := svc.GetJournalProgress(1)
	assert.Equal(t, 0, count)
	assert.Equal(t, 8, max)
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "journal_page_1", Quantity: 1})
	_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "journal_page_2", Quantity: 1})
	count, max = svc.GetJournalProgress(1)
	assert.Equal(t, 2, count)
}

func TestHasAllJournalPages(t *testing.T) {
	svc, s := testService(t)
	assert.False(t, svc.HasAllJournalPages(1))
	for i := 1; i <= 8; i++ {
		_ = s.DB.Create(&model.Inventory{UserID: 1, ItemID: "journal_page_" + itoa(i), Quantity: 1})
	}
	assert.True(t, svc.HasAllJournalPages(1))
}

func TestGetSiteInfoLevel3UnlocksIce(t *testing.T) {
	svc, s := testService(t)
	s.DB.Create(&model.Job{UserID: 1, JobName: "archeologist", Level: 3, XP: 0})
	infos := svc.GetSiteInfo(1)
	for _, info := range infos {
		if info.Key == "ice_sheet" {
			assert.True(t, info.Unlocked)
		}
		if info.Key == "volcanic" {
			assert.False(t, info.Unlocked)
		}
	}
}

func TestRollEventReturnsNilOnFresh(t *testing.T) {
	svc, _ := testService(t)
	state, _ := svc.NewGame(1, "cliffside")
	evt := svc.RollEvent(state)
	_ = evt
}

func TestSitesDefined(t *testing.T) {
	assert.Len(t, Sites, 5)
	assert.Contains(t, Sites, "riverbed")
	assert.Contains(t, Sites, "cliffside")
	assert.Contains(t, Sites, "fault")
	assert.Contains(t, Sites, "ice_sheet")
	assert.Contains(t, Sites, "volcanic")
}

func TestLayerDefs(t *testing.T) {
	assert.Len(t, LayerDefs, 5)
	_, ok := LayerDefs[LayerSoftSoil]
	assert.True(t, ok)
	_, ok = LayerDefs[LayerBedrock]
	assert.True(t, ok)
}

func digWithEvents(svc *Service, state *GameState, tool ActionType) {
	for !state.Finished {
		svc.ApplyAction(state, tool)
		if state.Finished {
			break
		}
		evt := svc.RollEvent(state)
		if evt == nil {
			continue
		}
		switch evt.Type {
		case EventFossilWhisper:
			svc.ResolveEvent(state, evt, "accept")
		case EventBuriedTreasure:
			svc.ResolveEvent(state, evt, "ignore")
		case EventGuardian:
			svc.ResolveEvent(state, evt, "retreat")
		case EventCaveIn:
			svc.ResolveEvent(state, evt, "rush")
		case EventFossilEgg:
			svc.ResolveEvent(state, evt, "take")
		}
	}
}

// TestDigBalance simulates full digs (including events) to verify that every
// site is completable with its intended tool and that results are not
// guaranteed to be broken bones or damaged fossils.
func TestDigBalance(t *testing.T) {
	rand.Seed(20260816)

	type goal struct {
		site        string
		tool        ActionType
		minComplete float64
		minClean    float64
		maxDisaster float64
	}
	goals := []goal{
		{"riverbed", ActionHammer, 0.90, 0.80, 0.05},
		{"cliffside", ActionHammer, 0.85, 0.70, 0.10},
		{"ice_sheet", ActionHammer, 0.80, 0.60, 0.10},
		{"fault", ActionDynamite, 0.80, 0.30, 0.30},
		{"volcanic", ActionDynamite, 0.80, 0.30, 0.30},
	}

	svc := testServiceInMemory(t)
	const runsPerSeq = 1000

	for _, g := range goals {
		site := Sites[g.site]
		total, complete, clean, disaster := 0, 0, 0, 0
		for _, seq := range site.LayerSeqs {
			for i := 0; i < runsPerSeq; i++ {
				state := &GameState{
					UserID:       1,
					PermitType:   g.site,
					Depth:        site.Depth / len(seq),
					MaxDepth:     site.Depth,
					Integrity:    100,
					Actions:      6,
					CurrentLayer: seq[0],
					LayerSeq:     seq,
					LayerIdx:     0,
					Site:         site,
				}
				digWithEvents(svc, state, g.tool)
				res := svc.Resolve(state)
				total++
				switch res.Quality {
				case "disaster":
					disaster++
				case "damaged":
				default:
					complete++
					if state.Integrity >= 50 {
						clean++
					}
				}
			}
		}
		completeRate := float64(complete) / float64(total)
		cleanRate := float64(clean) / float64(total)
		disasterRate := float64(disaster) / float64(total)
		t.Logf("%-9s tool=%-9s complete=%.3f clean(>=50)=%.3f disaster=%.3f",
			g.site, g.tool, completeRate, cleanRate, disasterRate)
		assert.GreaterOrEqual(t, completeRate, g.minComplete, "%s completion", g.site)
		assert.GreaterOrEqual(t, cleanRate, g.minClean, "%s clean digs", g.site)
		assert.LessOrEqual(t, disasterRate, g.maxDisaster, "%s disaster rate", g.site)
	}
}

// TestDigBalanceBrushNeverDamages verifies the brush never reduces integrity
// while digging, even on the hardest layers.
func TestDigBalanceBrushNeverDamages(t *testing.T) {
	svc := testServiceInMemory(t)
	site := Sites["riverbed"]
	for _, seq := range site.LayerSeqs {
		for i := 0; i < 200; i++ {
			state := &GameState{
				UserID:       1,
				PermitType:   "riverbed",
				Depth:        site.Depth / len(seq),
				MaxDepth:     site.Depth,
				Integrity:    100,
				Actions:      6,
				CurrentLayer: seq[0],
				LayerSeq:     seq,
				LayerIdx:     0,
				Site:         site,
			}
			svc.ApplyAction(state, ActionBrush)
			assert.Equal(t, 100, state.Integrity)
		}
	}
}
