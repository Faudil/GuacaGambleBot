package lore

import (
	"math/rand"

	"gorm.io/gorm"

	"guacagamblebot/internal/model"
	"guacagamblebot/internal/universe"
)

func (s *Service) fragments() []universe.Fragment {
	return s.universe.Fragments
}

func (s *Service) byID() map[string]universe.Fragment {
	if s.cachedByID != nil {
		return s.cachedByID
	}
	s.cachedByID = make(map[string]universe.Fragment, len(s.fragments()))
	for _, f := range s.fragments() {
		s.cachedByID[f.ID] = f
	}
	return s.cachedByID
}

func (s *Service) byCategory() map[universe.Category][]universe.Fragment {
	if s.cachedByCat != nil {
		return s.cachedByCat
	}
	s.cachedByCat = make(map[universe.Category][]universe.Fragment)
	for _, f := range s.fragments() {
		s.cachedByCat[f.Category] = append(s.cachedByCat[f.Category], f)
	}
	return s.cachedByCat
}

func (s *Service) Get(id string) *universe.Fragment {
	f, ok := s.byID()[id]
	if !ok {
		return nil
	}
	return &f
}

func (s *Service) AllInCategory(cat universe.Category) []universe.Fragment {
	return s.byCategory()[cat]
}

func (s *Service) Count(cat universe.Category) int {
	return len(s.byCategory()[cat])
}

func (s *Service) TotalCount() int {
	return len(s.fragments())
}

func (s *Service) Categories() []universe.Category {
	seen := map[universe.Category]bool{}
	var cats []universe.Category
	for _, f := range s.fragments() {
		if !seen[f.Category] {
			seen[f.Category] = true
			cats = append(cats, f.Category)
		}
	}
	return cats
}

func (s *Service) PickUndiscovered(db *gorm.DB, userID int64, cat universe.Category) *universe.Fragment {
	pool := s.byCategory()[cat]
	discovered := s.loadDiscovered(db, userID)
	var candidates []universe.Fragment
	for _, f := range pool {
		if !discovered[f.ID] {
			candidates = append(candidates, f)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	f := candidates[rand.Intn(len(candidates))]
	return &f
}

func (s *Service) loadDiscovered(db *gorm.DB, userID int64) map[string]bool {
	var entries []model.UserLoreEntry
	db.Where("user_id = ?", userID).Find(&entries)
	out := make(map[string]bool, len(entries))
	for _, e := range entries {
		out[e.LoreID] = true
	}
	return out
}
