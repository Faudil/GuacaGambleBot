package roulette

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGame(t *testing.T) {
	g := NewGame(1, 100)
	assert.Equal(t, int64(1), g.LeaderID)
	assert.Equal(t, 100, g.EntryFee)
	assert.False(t, g.IsActive)
	assert.Equal(t, 6, len(g.Cylinder))
}

func TestAddPlayer(t *testing.T) {
	g := NewGame(1, 100)
	assert.Equal(t, 1, len(g.Players)) // leader auto-added
	err := g.AddPlayer(2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(g.Players))

	err = g.AddPlayer(2)
	assert.ErrorIs(t, err, ErrAlreadyJoined)
}

func TestStartNotLeader(t *testing.T) {
	g := NewGame(1, 100)
	g.AddPlayer(3)
	err := g.Start(2)
	assert.ErrorIs(t, err, ErrNotLeader)
}

func TestStartMinPlayers(t *testing.T) {
	g := NewGame(1, 100)
	// Only leader is in the game (1 player)
	err := g.Start(1)
	assert.ErrorIs(t, err, ErrMinPlayers)
}

func TestStartSuccess(t *testing.T) {
	g := NewGame(1, 100)
	g.AddPlayer(2)
	err := g.Start(1)
	assert.NoError(t, err)
	assert.True(t, g.IsActive)
	assert.Equal(t, 200, g.Pot)
	assert.Equal(t, 2, len(g.Alive))
}

func TestCurrentPlayer(t *testing.T) {
	g := NewGame(1, 100)
	g.AddPlayer(2)
	_ = g.Start(1)
	cp := g.CurrentPlayer()
	assert.NotNil(t, cp)
	assert.Equal(t, int64(1), cp.UserID)
}

func TestTriggerNotYourTurn(t *testing.T) {
	g := NewGame(1, 100)
	g.AddPlayer(2)
	_ = g.Start(1)
	_, result, _, _ := g.Trigger(2)
	assert.Equal(t, "", result)
}

func TestTriggerClick(t *testing.T) {
	g := NewGame(1, 100)
	g.Cylinder = []bool{false, false, false, false, false, false}
	g.AddPlayer(2)
	_ = g.Start(1)

	ok, result, sur, share := g.Trigger(1)
	assert.True(t, ok)
	assert.Equal(t, "click", result)
	assert.Nil(t, sur)
	assert.Equal(t, 0, share)
	assert.Equal(t, 1, g.TurnIndex)
}

func TestTriggerDead(t *testing.T) {
	g := NewGame(1, 100)
	g.Cylinder = []bool{true, false, false, false, false, false}
	g.AddPlayer(2)
	_ = g.Start(1)

	ok, result, sur, share := g.Trigger(1)
	assert.True(t, ok)
	assert.Equal(t, "dead", result)
	assert.Equal(t, 1, len(sur))
	assert.Equal(t, 200, share)
	assert.False(t, g.IsActive)
}
