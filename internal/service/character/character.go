package character

import (
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

// Service holds the Character cog business logic.
type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

// ProfileResult holds the data shown by the profile view.
type ProfileResult struct {
	Wallet   int
	Bank     int
	Crowns   int
	AchCount int
}

// Profile returns the wallet, bank, crown and achievement counts for a user.
func (s *Service) Profile(userID int64) (*ProfileResult, error) {
	wallet, err := s.store.GetBalance(userID)
	if err != nil {
		return nil, err
	}
	_, bank, err := s.store.GetBankData(userID)
	if err != nil {
		return nil, err
	}
	var u model.User
	if err := s.store.DB.Where("user_id = ?", userID).First(&u).Error; err != nil {
		return nil, err
	}
	var achCount int64
	if err := s.store.DB.Model(&model.UserAchievement{}).
		Where("user_id = ?", userID).Count(&achCount).Error; err != nil {
		return nil, err
	}
	return &ProfileResult{
		Wallet:   wallet,
		Bank:     bank,
		Crowns:   u.Crowns,
		AchCount: int(achCount),
	}, nil
}
