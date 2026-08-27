package sanctuary_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	ps "guacagamblebot/internal/service/pets"
	sansvc "guacagamblebot/internal/service/sanctuary"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/testutil"
)

func newTestStore(t *testing.T) *store.Store {
	d := testutil.NewDB(t)
	return store.New(d, &config.Config{StartingBalance: 100})
}

func seedFusionPrereqs(t *testing.T, st *store.Store, userID int64) {
	t.Helper()
	require.NoError(t, st.DB.Create(&model.User{UserID: userID, Balance: 100000}).Error)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: userID, ItemID: "bone_dust", Quantity: 20}).Error)
	require.NoError(t, st.DB.Create(&model.Inventory{UserID: userID, ItemID: "coal", Quantity: 10}).Error)
	for i := 0; i < 5; i++ {
		require.NoError(t, st.DB.Create(&model.UserPet{
			UserID: userID, PetType: "Escargot", Nickname: "Snail",
			Level: 1, IsActive: false, OnExpedition: false,
		}).Error)
	}
}

// Pet fusion research is separate from the forge's gear fusion research
// (regression test for the bug where sanctuary.TradeUpResearch reused the
// forge's fusion_common/rare/epic IDs, so completing gear fusion research
// silently unlocked pet fusion too).
func TestTradeUpRequiresItsOwnResearchNotForges(t *testing.T) {
	userID := int64(700)
	st := newTestStore(t)
	psvc := ps.New(st, &config.Config{})
	svc := sansvc.New(st, &config.Config{}, psvc)

	seedFusionPrereqs(t, st, userID)

	var petIDs []int64
	var pets []model.UserPet
	require.NoError(t, st.DB.Where("user_id = ?", userID).Find(&pets).Error)
	for _, p := range pets {
		petIDs = append(petIDs, p.ID)
	}

	// No research completed at all: TradeUp must fail.
	_, err := svc.TradeUp(userID, petIDs)
	require.ErrorIs(t, err, sansvc.ErrFusionNoResearch)

	// Completing the FORGE's gear fusion research must NOT unlock pet fusion.
	require.NoError(t, st.DB.Create(&model.UserResearch{UserID: userID, ResearchID: "fusion_common", Completed: true}).Error)
	_, err = svc.TradeUp(userID, petIDs)
	require.ErrorIs(t, err, sansvc.ErrFusionNoResearch, "gear fusion research must not gate pet fusion")

	// Completing the sanctuary's OWN pet fusion research does unlock it.
	require.NoError(t, st.DB.Create(&model.UserResearch{UserID: userID, ResearchID: "pet_fusion_common", Completed: true}).Error)
	newPet, err := svc.TradeUp(userID, petIDs)
	require.NoError(t, err)
	require.NotNil(t, newPet)
}
