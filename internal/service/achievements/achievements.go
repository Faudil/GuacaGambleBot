package achievements

import (
	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

// View is a single achievement as seen by the achievements menu.
type View struct {
	ID       string
	Emoji    string
	Glory    int
	Unlocked bool
}

// Service holds the Achievements cog business logic.
type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

// List returns every achievement with the invoking user's unlock state.
func (s *Service) List(userID int64) ([]View, error) {
	var rows []model.UserAchievement
	if err := s.store.DB.Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	unlocked := make(map[string]bool, len(rows))
	for _, r := range rows {
		unlocked[r.AchievementID] = true
	}

	out := make([]View, 0, len(achievement.All()))
	for _, a := range achievement.All() {
		out = append(out, View{
			ID:       a.ID,
			Emoji:    a.Emoji,
			Glory:    a.Glory,
			Unlocked: unlocked[a.ID],
		})
	}
	return out, nil
}
