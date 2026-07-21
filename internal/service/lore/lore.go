package lore

import (
	"time"

	"gorm.io/gorm/clause"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

// Discover records a lore fragment discovery. Returns true if it's a new discovery.
func (s *Service) Discover(userID int64, loreID string) (bool, error) {
	frag := Get(loreID)
	if frag == nil {
		return false, nil
	}
	var count int64
	s.store.DB.Model(&model.UserLoreEntry{}).
		Where("user_id = ? AND lore_id = ?", userID, loreID).Count(&count)
	if count > 0 {
		return false, nil
	}
	err := s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "lore_id"}},
		DoNothing: true,
	}).Create(&model.UserLoreEntry{
		UserID:       userID,
		LoreID:       loreID,
		DiscoveredAt: time.Now(),
	}).Error
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetDiscovered returns a set of all lore IDs discovered by the user.
func (s *Service) GetDiscovered(userID int64) (map[string]bool, error) {
	return loadDiscovered(s.store.DB, userID), nil
}

// CategoryProgress returns (discovered, total) for a category.
func (s *Service) CategoryProgress(userID int64, cat Category) (int, int, error) {
	total := Count(cat)
	discovered := loadDiscovered(s.store.DB, userID)
	count := 0
	for _, f := range AllInCategory(cat) {
		if discovered[f.ID] {
			count++
		}
	}
	return count, total, nil
}

// AllProgress returns progress for every category plus grand totals.
func (s *Service) AllProgress(userID int64) (map[Category]struct{ D, T int }, int, int, error) {
	out := make(map[Category]struct{ D, T int })
	totalD, totalT := 0, 0
	for _, cat := range []Category{CatAether, CatTide, CatRoot, CatField, CatRust, CatEcho, CatBonus} {
		d, t, err := s.CategoryProgress(userID, cat)
		if err != nil {
			return nil, 0, 0, err
		}
		out[cat] = struct{ D, T int }{d, t}
		totalD += d
		totalT += t
	}
	return out, totalD, totalT, nil
}

// TotalDiscovered returns total count of fragments discovered by the user.
func (s *Service) TotalDiscovered(userID int64) (int, error) {
	var count int64
	if err := s.store.DB.Model(&model.UserLoreEntry{}).
		Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}


