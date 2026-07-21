package betting

import (
	"errors"
	"strconv"

	"gorm.io/gorm"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

var (
	ErrNotFound    = errors.New("bet not found")
	ErrClosed      = errors.New("bet is closed")
	ErrFrozen      = errors.New("bet is frozen")
	ErrNoMoney     = errors.New("insufficient funds")
	ErrNotCreator  = errors.New("only the creator can do this")
	ErrInvalidOpt  = errors.New("invalid option, choose a or b")
)

type OddsResult struct {
	BetID       int64
	Description string
	Status      string
	Option1     string
	Option2     string
	Pool1       int
	Pool2       int
	Total       int
	Odds1       string
	Odds2       string
	Winner      string
}

type CloseResult struct {
	TotalPool    int
	WinningPool  int
	Multiplier   float64
	WinningOpt   string
	WagerResults []WagerResult
}

type WagerResult struct {
	UserID int64
	Amount int
	Won    bool
}

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) CreateBet(creatorID int64, description, option1, option2 string) (int64, error) {
	bet := model.Bet{
		CreatorID:   creatorID,
		Description: description,
		Option1:     option1,
		Option2:     option2,
		Status:      "OPEN",
	}
	if err := s.store.DB.Create(&bet).Error; err != nil {
		return 0, err
	}
	return bet.ID, nil
}

func (s *Service) getBet(betID int64) (*model.Bet, error) {
	var bet model.Bet
	if err := s.store.DB.First(&bet, betID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &bet, nil
}

func (s *Service) PlaceBet(userID, betID int64, choice string, amount int) error {
	if choice != "a" && choice != "b" {
		return ErrInvalidOpt
	}
	bet, err := s.getBet(betID)
	if err != nil {
		return err
	}
	if bet.Status == "CLOSE" {
		return ErrClosed
	}
	if bet.Status == "FROZEN" {
		return ErrFrozen
	}
	bal, err := s.store.GetBalance(userID)
	if err != nil {
		return err
	}
	if bal < amount {
		return ErrNoMoney
	}
	if _, err := s.store.UpdateBalance(userID, -amount); err != nil {
		return err
	}
	return s.store.DB.Create(&model.Wager{
		BetID:  betID,
		UserID: userID,
		Option: choice,
		Amount: amount,
	}).Error
}

func (s *Service) CloseBet(creatorID, betID int64, winningOption string) (*CloseResult, error) {
	if winningOption != "a" && winningOption != "b" {
		return nil, ErrInvalidOpt
	}
	bet, err := s.getBet(betID)
	if err != nil {
		return nil, err
	}
	if bet.CreatorID != creatorID {
		return nil, ErrNotCreator
	}
	if bet.Status == "CLOSE" {
		return nil, ErrClosed
	}

	var wagers []model.Wager
	if err := s.store.DB.Where("bet_id = ?", betID).Find(&wagers).Error; err != nil {
		return nil, err
	}

	totalPool := 0
	winningPool := 0
	for _, w := range wagers {
		totalPool += w.Amount
		if w.Option == winningOption {
			winningPool += w.Amount
		}
	}

	res := &CloseResult{
		TotalPool:   totalPool,
		WinningPool: winningPool,
		WinningOpt:  winningOption,
	}

	if winningPool > 0 {
		multiplier := float64(totalPool) / float64(winningPool)
		res.Multiplier = multiplier

		for _, w := range wagers {
			if w.Option == winningOption {
				payout := int(float64(w.Amount) * multiplier)
				if _, err := s.store.UpdateBalance(w.UserID, payout); err != nil {
					return nil, err
				}
				_ = achievement.IncrementStat(s.store.DB, w.UserID, "wagers_won", 1)
				res.WagerResults = append(res.WagerResults, WagerResult{UserID: w.UserID, Amount: payout, Won: true})
			} else {
				_ = achievement.IncrementStat(s.store.DB, w.UserID, "wagers_lost", 1)
				res.WagerResults = append(res.WagerResults, WagerResult{UserID: w.UserID, Amount: w.Amount, Won: false})
			}
		}
	}

	winningOptName := bet.Option1
	if winningOption == "b" {
		winningOptName = bet.Option2
	}

	if err := s.store.DB.Model(&model.Bet{}).
		Where("id = ?", betID).
		Updates(map[string]any{"status": "CLOSE", "winner": winningOptName}).Error; err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Service) FreezeBet(creatorID, betID int64) error {
	bet, err := s.getBet(betID)
	if err != nil {
		return err
	}
	if bet.CreatorID != creatorID {
		return ErrNotCreator
	}
	if bet.Status == "CLOSE" {
		return ErrClosed
	}
	if bet.Status == "FROZEN" {
		return ErrFrozen
	}
	return s.store.DB.Model(&model.Bet{}).
		Where("id = ?", betID).
		Update("status", "FROZEN").Error
}

func (s *Service) ShowOdds(betID int64) (*OddsResult, error) {
	bet, err := s.getBet(betID)
	if err != nil {
		return nil, err
	}

	var wagers []model.Wager
	if err := s.store.DB.Where("bet_id = ?", betID).Find(&wagers).Error; err != nil {
		return nil, err
	}

	total := 0
	pool1 := 0
	pool2 := 0
	for _, w := range wagers {
		total += w.Amount
		if w.Option == "a" {
			pool1 += w.Amount
		} else {
			pool2 += w.Amount
		}
	}

	res := &OddsResult{
		BetID:       bet.ID,
		Description: bet.Description,
		Status:      bet.Status,
		Option1:     bet.Option1,
		Option2:     bet.Option2,
		Pool1:       pool1,
		Pool2:       pool2,
		Total:       total,
		Winner:      bet.Winner,
	}

	if pool1 > 0 {
		res.Odds1 = formatOdds(float64(total) / float64(pool1))
	} else {
		res.Odds1 = "N/A"
	}
	if pool2 > 0 {
		res.Odds2 = formatOdds(float64(total) / float64(pool2))
	} else {
		res.Odds2 = "N/A"
	}

	return res, nil
}

func formatOdds(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	return s + "x"
}
