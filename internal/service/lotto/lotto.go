package lotto

import (
	"errors"
	"math/rand"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
	"guacagamblebot/internal/store"
)

var (
	ErrNoMoney      = errors.New("insufficient funds")
	ErrInvalidNum   = errors.New("number must be between 1 and 100")
	ErrLimit        = errors.New("daily limit reached")
)

type TicketResult struct {
	Win        bool
	Number     int
	WinningNum int
	Jackpot    int
	AddedValue int
	NewJackpot int
	Unlocks    []*achievement.Achievement
	LeveledUp  bool
	NewLevel   int
}

type JackpotInfo struct {
	Jackpot       int
	WinningNumber int
}

type Service struct {
	store        *store.Store
	cfg          *config.Config
	TicketPrice  int
	DailyIncrease int
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{
		store:         s,
		cfg:           cfg,
		TicketPrice:   20,
		DailyIncrease: 300,
	}
}

func (s *Service) ensureServerState(serverID int64) error {
	return s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "server_id"}},
		DoNothing: true,
	}).Create(&model.ServerLottoState{
		ServerID:      serverID,
		WinningNumber: rand.Intn(100) + 1,
		Jackpot:       s.cfg.BaseJackpot,
	}).Error
}

func (s *Service) getState(serverID int64) (*model.ServerLottoState, error) {
	if err := s.ensureServerState(serverID); err != nil {
		return nil, err
	}
	var state model.ServerLottoState
	if err := s.store.DB.Where("server_id = ?", serverID).First(&state).Error; err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *Service) BuyTicket(userID, serverID int64, number int) (*TicketResult, error) {
	if number < 1 || number > 100 {
		return nil, ErrInvalidNum
	}
	bal, err := s.store.GetBalance(userID)
	if err != nil {
		return nil, err
	}
	if bal < s.TicketPrice {
		return nil, ErrNoMoney
	}
	ok, _, err := s.store.CheckGameLimit(userID, "lotto", 3)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrLimit
	}

	if _, err := s.store.UpdateBalance(userID, -s.TicketPrice); err != nil {
		return nil, err
	}
	if err := s.store.IncrementGameLimit(userID, "lotto"); err != nil {
		return nil, err
	}
	_ = achievement.IncrementStat(s.store.DB, userID, "lotto_participations", 1)

	state, err := s.getState(serverID)
	if err != nil {
		return nil, err
	}

	res := &TicketResult{
		Number:      number,
		WinningNum:  state.WinningNumber,
		Jackpot:     state.Jackpot,
		AddedValue:  s.TicketPrice,
		NewJackpot:  state.Jackpot + s.TicketPrice,
	}

	if err := s.store.DB.Model(&model.ServerLottoState{}).
		Where("server_id = ?", serverID).
		UpdateColumn("jackpot", gorm.Expr("jackpot + ?", s.TicketPrice)).Error; err != nil {
		return nil, err
	}

	if number == state.WinningNumber {
		res.Win = true
		_ = achievement.IncrementStat(s.store.DB, userID, "lotto_won", 1)
		if _, err := s.store.UpdateBalance(userID, state.Jackpot); err != nil {
			return nil, err
		}
		newNum := rand.Intn(100) + 1
		newJackpot := s.cfg.BaseJackpot
		if err := s.store.DB.Model(&model.ServerLottoState{}).
			Where("server_id = ?", serverID).
			Updates(map[string]any{
				"winning_number": newNum,
				"jackpot":        newJackpot,
			}).Error; err != nil {
			return nil, err
		}
		res.NewJackpot = newJackpot
	} else {
		res.NewJackpot = state.Jackpot + s.TicketPrice
	}

	leveled, lvl := charsvc.AddXP(s.store, userID, 10)
	res.LeveledUp = leveled
	res.NewLevel = lvl
	res.Unlocks, _ = achievement.CheckAndUnlock(s.store.DB, userID)
	return res, nil
}

func (s *Service) Jackpot(serverID int64) (*JackpotInfo, error) {
	state, err := s.getState(serverID)
	if err != nil {
		return nil, err
	}
	return &JackpotInfo{Jackpot: state.Jackpot, WinningNumber: state.WinningNumber}, nil
}

func (s *Service) TryDailyBonus(serverID int64) (bool, error) {
	state, err := s.getState(serverID)
	if err != nil {
		return false, err
	}
	today := time.Now().Format("2006-01-02")
	if state.LastBonusDate == today {
		return false, nil
	}
	if err := s.store.DB.Model(&model.ServerLottoState{}).
		Where("server_id = ?", serverID).
		Updates(map[string]any{
			"jackpot":        gorm.Expr("jackpot + ?", s.DailyIncrease),
			"last_bonus_date": today,
		}).Error; err != nil {
		return false, err
	}
	return true, nil
}
