package quests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"guacagamblebot/internal/model"
)

// grantRewardItem must turn equipment rewards into real UserEquipment
// instances instead of plain inventory rows.
func TestGrantRewardItemCreatesEquipmentInstance(t *testing.T) {
	svc, st := testService(t)

	svc.grantRewardItem(1, "spark_shard")

	var eqs []model.UserEquipment
	require.NoError(t, st.DB.Where("user_id = ?", 1).Find(&eqs).Error)
	require.Len(t, eqs, 1, "gear reward must create an equipment instance")
	assert.Equal(t, "spark_shard", eqs[0].BaseID)
	assert.Equal(t, "trinket", eqs[0].EquipSlot)
	assert.Equal(t, 15, eqs[0].MinLevel)
	assert.Equal(t, "epic", eqs[0].Rarity)

	var inv []model.Inventory
	require.NoError(t, st.DB.Where("user_id = ?", 1).Find(&inv).Error)
	assert.Len(t, inv, 0, "no plain inventory row for gear rewards")
}

// grantRewardItem must keep non-equipment rewards as inventory rows.
func TestGrantRewardItemNonGearStaysInventory(t *testing.T) {
	svc, st := testService(t)

	svc.grantRewardItem(1, "coal")

	var eqs []model.UserEquipment
	require.NoError(t, st.DB.Where("user_id = ?", 1).Find(&eqs).Error)
	assert.Len(t, eqs, 0)

	has, err := st.HasItem(1, "coal", 1)
	require.NoError(t, err)
	assert.True(t, has)
}
