package leaderboard

import (
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

// Service holds the Leaderboard cog business logic.
type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

// Top returns the n users with the highest wallet balance.
func (s *Service) Top(n int) ([]model.User, error) {
	var users []model.User
	if err := s.store.DB.Model(&model.User{}).
		Order("balance desc").
		Limit(n).
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
