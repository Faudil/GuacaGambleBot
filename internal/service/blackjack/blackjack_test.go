package blackjack

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDeck(t *testing.T) {
	svc := New()
	deck := svc.CreateDeck()
	assert.Equal(t, 52, len(deck))
}

func TestNewGame(t *testing.T) {
	svc := New()
	gs := svc.NewGame(1, 2, 100)
	assert.Equal(t, int64(1), gs.Player1ID)
	assert.Equal(t, int64(2), gs.Player2ID)
	assert.Equal(t, 100, gs.Amount)
	assert.Equal(t, int64(1), gs.Turn)
	assert.Equal(t, 2, len(gs.Hands[1].Cards))
	assert.Equal(t, 2, len(gs.Hands[2].Cards))
	assert.Equal(t, 48, len(gs.Deck))
}

func TestScore(t *testing.T) {
	h := &Hand{Cards: []Card{
		{Rank: "A", Val: 11},
		{Rank: "K", Val: 10},
	}}
	assert.Equal(t, 21, h.Score())
}

func TestScoreBust(t *testing.T) {
	h := &Hand{Cards: []Card{
		{Rank: "K", Val: 10},
		{Rank: "Q", Val: 10},
		{Rank: "5", Val: 5},
	}}
	assert.Equal(t, 25, h.Score())
}

func TestScoreAceAdjustment(t *testing.T) {
	h := &Hand{Cards: []Card{
		{Rank: "A", Val: 11},
		{Rank: "A", Val: 11},
		{Rank: "9", Val: 9},
	}}
	// 11 + 11 + 9 = 31 → adjust one ace → 11 + 1 + 9 = 21
	assert.Equal(t, 21, h.Score())
}

func TestHit(t *testing.T) {
	svc := New()
	gs := svc.NewGame(1, 2, 100)
	ok, _ := gs.Hit(1)
	assert.True(t, ok)
	assert.Equal(t, 3, len(gs.Hands[1].Cards))
	assert.Equal(t, 47, len(gs.Deck))
}

func TestHitNotYourTurn(t *testing.T) {
	svc := New()
	gs := svc.NewGame(1, 2, 100)
	ok, _ := gs.Hit(2)
	assert.False(t, ok) // not p2's turn
}

func TestStand(t *testing.T) {
	svc := New()
	gs := svc.NewGame(1, 2, 100)
	assert.True(t, gs.Stand(1))
	assert.Equal(t, int64(2), gs.Turn) // now p2's turn
}

func TestCheckGameOverBust(t *testing.T) {
	svc := New()
	gs := svc.NewGame(1, 2, 100)
	// Force p1 to have >21
	gs.Hands[1] = &Hand{Cards: []Card{
		{Rank: "K", Val: 10},
		{Rank: "Q", Val: 10},
		{Rank: "5", Val: 5},
	}}
	gs.Finished[1] = true
	wid, reason, draw, over := gs.CheckGameOver()
	assert.True(t, over)
	assert.False(t, draw)
	assert.Equal(t, int64(2), wid)
	assert.Equal(t, "bust_p1", reason)
}

func TestCheckGameOverDraw(t *testing.T) {
	svc := New()
	gs := svc.NewGame(1, 2, 100)
	gs.Hands[1] = &Hand{Cards: []Card{
		{Rank: "K", Val: 10},
		{Rank: "7", Val: 7},
	}}
	gs.Hands[2] = &Hand{Cards: []Card{
		{Rank: "Q", Val: 10},
		{Rank: "7", Val: 7},
	}}
	gs.Finished[1] = true
	gs.Finished[2] = true
	wid, reason, draw, over := gs.CheckGameOver()
	assert.True(t, over)
	assert.True(t, draw)
	assert.Equal(t, int64(0), wid)
	assert.Equal(t, "draw", reason)
}

func TestFormatHand(t *testing.T) {
	h := &Hand{Cards: []Card{
		{Rank: "A", Suit: "♠️", Val: 11},
		{Rank: "K", Suit: "♥️", Val: 10},
	}}
	s := h.Format()
	require.Contains(t, s, "A♠️")
	require.Contains(t, s, "K♥️")
}
