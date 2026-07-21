package blackjack

import (
	"errors"
	"math/rand"
)

var (
	ErrNoMoney  = errors.New("insufficient funds")
	ErrOpponent = errors.New("invalid opponent")
	ErrAmount   = errors.New("invalid amount")
)

type Card struct {
	Suit string
	Rank string
	Val  int
}

type Hand struct {
	Cards []Card
}

func (h *Hand) Score() int {
	score := 0
	aces := 0
	for _, c := range h.Cards {
		score += c.Val
		if c.Rank == "A" {
			aces++
		}
	}
	for score > 21 && aces > 0 {
		score -= 10
		aces--
	}
	return score
}

func (h *Hand) Format() string {
	s := ""
	for _, c := range h.Cards {
		s += "[" + c.Rank + c.Suit + "] "
	}
	return s
}

func (h *Hand) Add(c Card) {
	h.Cards = append(h.Cards, c)
}

type GameState struct {
	Player1ID int64
	Player2ID int64
	Amount    int
	Deck      []Card
	Hands     map[int64]*Hand
	Turn      int64
	Finished  map[int64]bool
}

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) CreateDeck() []Card {
	suits := []string{"♠️", "♥️", "♦️", "♣️"}
	ranks := []string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}
	values := map[string]int{
		"2": 2, "3": 3, "4": 4, "5": 5, "6": 6, "7": 7, "8": 8,
		"9": 9, "10": 10, "J": 10, "Q": 10, "K": 10, "A": 11,
	}
	deck := make([]Card, 0, 52)
	for _, suit := range suits {
		for _, rank := range ranks {
			deck = append(deck, Card{Suit: suit, Rank: rank, Val: values[rank]})
		}
	}
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	return deck
}

func (s *Service) NewGame(p1, p2 int64, amount int) *GameState {
	deck := s.CreateDeck()
	gs := &GameState{
		Player1ID: p1,
		Player2ID: p2,
		Amount:    amount,
		Deck:      deck,
		Hands:     map[int64]*Hand{},
		Turn:      p1,
		Finished:  map[int64]bool{},
	}
	gs.Hands[p1] = &Hand{Cards: []Card{gs.draw(), gs.draw()}}
	gs.Hands[p2] = &Hand{Cards: []Card{gs.draw(), gs.draw()}}
	return gs
}

func (gs *GameState) draw() Card {
	c := gs.Deck[0]
	gs.Deck = gs.Deck[1:]
	return c
}

func (gs *GameState) Hit(playerID int64) (bool, bool) {
	if gs.Turn != playerID || gs.Finished[playerID] {
		return false, false
	}
	c := gs.draw()
	gs.Hands[playerID].Add(c)
	score := gs.Hands[playerID].Score()
	if score > 21 {
		gs.Finished[playerID] = true
		return true, true
	}
	return true, false
}

func (gs *GameState) Stand(playerID int64) bool {
	if gs.Turn != playerID || gs.Finished[playerID] {
		return false
	}
	gs.Finished[playerID] = true
	if playerID == gs.Player1ID {
		gs.Turn = gs.Player2ID
	} else {
		gs.Turn = gs.Player1ID
	}
	return true
}

func (gs *GameState) CheckGameOver() (winnerID int64, reason string, isDraw bool, over bool) {
	s1 := gs.Hands[gs.Player1ID].Score()
	s2 := gs.Hands[gs.Player2ID].Score()

	if s1 > 21 {
		return gs.Player2ID, "bust_p1", false, true
	}
	if s2 > 21 {
		return gs.Player1ID, "bust_p2", false, true
	}
	if gs.Finished[gs.Player1ID] && gs.Finished[gs.Player2ID] {
		over = true
		if s1 > s2 {
			return gs.Player1ID, "beat", false, true
		} else if s2 > s1 {
			return gs.Player2ID, "beat", false, true
		} else {
			return 0, "draw", true, true
		}
	}
	return 0, "", false, false
}
