package lore

import (
	"time"

	"gorm.io/gorm/clause"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
)

type Service struct {
	store       *store.Store
	cfg         *config.Config
	universe    *universe.Definition
	cachedByID  map[string]universe.Fragment
	cachedByCat map[universe.Category][]universe.Fragment
}

func New(s *store.Store, cfg *config.Config, def *universe.Definition) *Service {
	return &Service{store: s, cfg: cfg, universe: def}
}

func (s *Service) Universe() *universe.Definition {
	return s.universe
}

func (s *Service) Discover(userID int64, loreID string) (bool, error) {
	frag := s.Get(loreID)
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

func (s *Service) GetDiscovered(userID int64) (map[string]bool, error) {
	return s.loadDiscovered(s.store.DB, userID), nil
}

func (s *Service) CategoryProgress(userID int64, cat universe.Category) (int, int, error) {
	total := s.Count(cat)
	discovered := s.loadDiscovered(s.store.DB, userID)
	count := 0
	for _, f := range s.AllInCategory(cat) {
		if discovered[f.ID] {
			count++
		}
	}
	return count, total, nil
}

func (s *Service) AllProgress(userID int64) (map[universe.Category]struct{ D, T int }, int, int, error) {
	out := make(map[universe.Category]struct{ D, T int })
	totalD, totalT := 0, 0
	for _, cat := range s.Categories() {
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

func (s *Service) TotalDiscovered(userID int64) (int, error) {
	var count int64
	if err := s.store.DB.Model(&model.UserLoreEntry{}).
		Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}
