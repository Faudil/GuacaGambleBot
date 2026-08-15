package roulette

import (
	"errors"
	"math/rand"
)

var (
	ErrNoMoney       = errors.New("insufficient funds")
	ErrNotLeader     = errors.New("only the leader can start")
	ErrMinPlayers    = errors.New("need at least 2 players")
	ErrNotYourTurn   = errors.New("not your turn")
	ErrAlreadyJoined = errors.New("already joined")
)

type Player struct {
	UserID int64
}

type Game struct {
	Players    []Player
	Alive      []Player
	EntryFee   int
	Pot        int
	Cylinder   []bool
	TurnIndex  int
	IsActive   bool
	LeaderID   int64
	LuckyBreak bool
}

func NewGame(leaderID int64, entryFee int) *Game {
	cyl := make([]bool, 6)
	cyl[rand.Intn(6)] = true
	return &Game{
		Players:  []Player{{UserID: leaderID}},
		EntryFee: entryFee,
		Cylinder: cyl,
		LeaderID: leaderID,
		IsActive: false,
	}
}

func (g *Game) AddPlayer(userID int64) error {
	for _, p := range g.Players {
		if p.UserID == userID {
			return ErrAlreadyJoined
		}
	}
	g.Players = append(g.Players, Player{UserID: userID})
	return nil
}

func (g *Game) Start(userID int64) error {
	if userID != g.LeaderID {
		return ErrNotLeader
	}
	if len(g.Players) < 2 {
		return ErrMinPlayers
	}
	g.Alive = make([]Player, len(g.Players))
	copy(g.Alive, g.Players)
	g.Pot = len(g.Players) * g.EntryFee
	g.IsActive = true
	g.TurnIndex = 0
	return nil
}

func (g *Game) CurrentPlayer() *Player {
	if len(g.Alive) == 0 {
		return nil
	}
	return &g.Alive[g.TurnIndex%len(g.Alive)]
}

func (g *Game) Trigger(userID int64) (bool, string, []Player, int) {
	if g.CurrentPlayer() == nil || g.CurrentPlayer().UserID != userID {
		return false, "", nil, 0
	}
	bullet := g.Cylinder[0]
	g.Cylinder = g.Cylinder[1:]
	if bullet {
		if g.LuckyBreak {
			// The trigger player's lucky_break buff deflects the bullet.
			g.LuckyBreak = false
			g.TurnIndex++
			return true, "click", nil, 0
		}
		// Player dies
		survivors := make([]Player, 0, len(g.Alive)-1)
		for _, p := range g.Alive {
			if p.UserID != userID {
				survivors = append(survivors, p)
			}
		}
		share := 0
		if len(survivors) > 0 {
			share = g.Pot / len(survivors)
		}
		g.IsActive = false
		return true, "dead", survivors, share
	}
	g.TurnIndex++
	return true, "click", nil, 0
}
