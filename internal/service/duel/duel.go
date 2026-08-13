package duel

import (
	"errors"
	"math/rand"
	"time"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/store"
)

var (
	ErrNoMoney     = errors.New("insufficient funds")
	ErrSelf        = errors.New("cannot duel yourself")
	ErrBot         = errors.New("cannot duel a bot")
	ErrAmount      = errors.New("amount must be positive")
	ErrDuelLimit   = errors.New("duel daily limit reached")
	ErrDuelCD      = errors.New("wait before dueling again")
)

type DuelResult struct {
	ChallengerID  int64
	OpponentID    int64
	Amount        int
	Die1C         int
	Die2C         int
	TotalC        int
	Die1O         int
	Die2O         int
	TotalO        int
	WinnerID      int64
	IsDraw        bool
	UnlocksC      []*achievement.Achievement
	UnlocksO      []*achievement.Achievement
}

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) Duel(challengerID, opponentID int64, amount int) (*DuelResult, error) {
	ok, _, err := s.store.CheckGameLimit(challengerID, "duel", 20)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrDuelLimit
	}

	ready, err := s.store.CheckCooldown(challengerID, "duel", 30*time.Second)
	if err != nil {
		return nil, err
	}
	if !ready {
		return nil, ErrDuelCD
	}

	if challengerID == opponentID {
		return nil, ErrSelf
	}
	if amount <= 0 {
		return nil, ErrAmount
	}
	cb, err := s.store.GetBalance(challengerID)
	if err != nil {
		return nil, err
	}
	if cb < amount {
		return nil, ErrNoMoney
	}
	ob, err := s.store.GetBalance(opponentID)
	if err != nil {
		return nil, err
	}
	if ob < amount {
		return nil, ErrNoMoney
	}

	d1c := rand.Intn(6) + 1
	d2c := rand.Intn(6) + 1
	tc := d1c + d2c
	d1o := rand.Intn(6) + 1
	d2o := rand.Intn(6) + 1
	to := d1o + d2o

	res := &DuelResult{
		ChallengerID: challengerID,
		OpponentID:   opponentID,
		Amount:       amount,
		Die1C:        d1c, Die2C: d2c, TotalC: tc,
		Die1O:        d1o, Die2O: d2o, TotalO: to,
	}

	if tc > to {
		res.WinnerID = challengerID
		if _, _, err := s.store.Transfer(opponentID, challengerID, amount); err != nil {
			return nil, mapMoneyErr(err)
		}
		if err := achievement.IncrementStat(s.store.DB, challengerID, "pvp_wins", 1); err != nil {
			return nil, err
		}
		if err := achievement.IncrementStat(s.store.DB, opponentID, "pvp_losses", 1); err != nil {
			return nil, err
		}
	} else if to > tc {
		res.WinnerID = opponentID
		if _, _, err := s.store.Transfer(challengerID, opponentID, amount); err != nil {
			return nil, mapMoneyErr(err)
		}
		if err := achievement.IncrementStat(s.store.DB, opponentID, "pvp_wins", 1); err != nil {
			return nil, err
		}
		if err := achievement.IncrementStat(s.store.DB, challengerID, "pvp_losses", 1); err != nil {
			return nil, err
		}
	} else {
		res.IsDraw = true
	}

	_ = s.store.IncrementGameLimit(challengerID, "duel")
	_ = s.store.SetCooldown(challengerID, "duel")

	res.UnlocksC, _ = achievement.CheckAndUnlock(s.store.DB, challengerID)
	res.UnlocksO, _ = achievement.CheckAndUnlock(s.store.DB, opponentID)
	return res, nil
}

func mapMoneyErr(err error) error {
	if errors.Is(err, store.ErrInsufficientFunds) {
		return ErrNoMoney
	}
	return err
}
