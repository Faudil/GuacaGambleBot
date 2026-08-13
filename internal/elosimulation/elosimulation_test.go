package elosimulation

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

func TestToBattlePetConversion(t *testing.T) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "p.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	st := store.New(d, &config.Config{})

	p := &model.UserPet{
		ID:       42,
		Nickname: "TestPet",
		Level:    10,
		MaxHP:    200,
		HP:       150,
		Atk:      30,
		Defense:  20,
		Speed:    15,
		DGE:      10,
		ACC:      5,
		CritC:    8,
		CritD:    2.0,
		SpcC:     5,
	}
	bp := toBattlePet(st, p)
	assert.Equal(t, p.ID, bp.ID)
	assert.Equal(t, p.Nickname, bp.Nickname)
	assert.Equal(t, p.Level, bp.Level)
	assert.Equal(t, p.MaxHP, bp.MaxHP)
	assert.Equal(t, p.HP, bp.HP)
	assert.Equal(t, p.Atk, bp.Atk)
	assert.Equal(t, p.Defense, bp.Defense)
	assert.Equal(t, p.Speed, bp.Speed)
	assert.Equal(t, p.DGE, bp.DGE)
	assert.Equal(t, p.ACC, bp.ACC)
	assert.Equal(t, p.CritC, bp.CritC)
	assert.Equal(t, p.CritD, bp.CritD)
	assert.Equal(t, p.SpcC, bp.SpcC)
}
