package duel

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/store"
)

func testService(t *testing.T) (*Service, *store.Store) {
	d, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "d.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(d))
	cfg := &config.Config{StartingBalance: 100, DailyAmount: 50}
	s := store.New(d, cfg)
	return New(s, cfg), s
}

func setupUser(t *testing.T, st *store.Store, id int64, bal int) {
	_, err := st.UpdateBalance(id, bal)
	require.NoError(t, err)
}

func TestDuelErrors(t *testing.T) {
	svc, st := testService(t)
	setupUser(t, st, 1, 1000)
	setupUser(t, st, 2, 500)

	_, err := svc.Duel(1, 1, 100)
	assert.ErrorIs(t, err, ErrSelf)

	_, err = svc.Duel(1, 2, -5)
	assert.ErrorIs(t, err, ErrAmount)

	_, err = svc.Duel(1, 2, 999999)
	assert.ErrorIs(t, err, ErrNoMoney)
}

func TestDuelBalanceChanges(t *testing.T) {
	svc, st := testService(t)
	setupUser(t, st, 1, 1000) // bal 1100 (100 start + 1000)
	setupUser(t, st, 2, 500)  // bal 600

	b1pre, _ := st.GetBalance(1)
	b2pre, _ := st.GetBalance(2)

	res, err := svc.Duel(1, 2, 100)
	require.NoError(t, err)

	b1post, _ := st.GetBalance(1)
	b2post, _ := st.GetBalance(2)

	delta1 := b1post - b1pre
	delta2 := b2post - b2pre

	if res.IsDraw {
		assert.Equal(t, 0, delta1)
		assert.Equal(t, 0, delta2)
	} else {
		// winner gains +100, loser loses -100
		if res.WinnerID == 1 {
			assert.Equal(t, 100, delta1)
			assert.Equal(t, -100, delta2)
		} else {
			assert.Equal(t, -100, delta1)
			assert.Equal(t, 100, delta2)
		}
	}
}

func TestDuelWinnerStatsRecorded(t *testing.T) {
	svc, st := testService(t)
	setupUser(t, st, 5, 1000)
	setupUser(t, st, 6, 500)

	res, err := svc.Duel(5, 6, 200)
	require.NoError(t, err)
	assert.Equal(t, 200, res.Amount)
	assert.Greater(t, res.Die1C, 0)
	assert.Greater(t, res.Die2C, 0)
	assert.Greater(t, res.Die1O, 0)
	assert.Greater(t, res.Die2O, 0)
}
