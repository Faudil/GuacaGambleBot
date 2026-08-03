package archeology

import (
	"gorm.io/gorm"

	"guacagamblebot/internal/model"
)

type ToolMasteryTier struct {
	Uses   int
	TitleID string
	Bonus  float64
}

var ToolMasteryTiers = []ToolMasteryTier{
	{Uses: 50, TitleID: "arch.mastery_specialist", Bonus: 1.10},
	{Uses: 100, TitleID: "arch.mastery_master", Bonus: 1.20},
	{Uses: 200, TitleID: "arch.mastery_legend", Bonus: 1.30},
}

func (s *Service) GetToolMastery(userID int64) map[string]struct {
	Uses int
	TitleID string
}{
	result := make(map[string]struct {
		Uses int
		TitleID string
	})
	for _, toolID := range []string{"dynamite", "hammer", "brush"} {
		col := "tool_" + toolID + "_uses"
		var uses int
		s.store.DB.Model(&model.UserStat{}).Where("user_id = ?", userID).Pluck(col, &uses)
		titleID := ""
		for _, tier := range ToolMasteryTiers {
			if uses >= tier.Uses {
				titleID = tier.TitleID
			}
		}
		result[toolID] = struct {
			Uses int
			TitleID string
		}{Uses: uses, TitleID: titleID}
	}
	return result
}

func (s *Service) incrementToolUses(userID int64, toolID string) {
	col := "tool_" + toolID + "_uses"
	if err := s.store.DB.Where("user_id = ?", userID).FirstOrCreate(&model.UserStat{UserID: userID}).Error; err != nil {
		return
	}
	s.store.DB.Model(&model.UserStat{}).Where("user_id = ?", userID).
		UpdateColumn(col, gorm.Expr(col+" + ?", 1))
}

func (s *Service) GetToolMasteryBonus(userID int64, toolID string) float64 {
	col := "tool_" + toolID + "_uses"
	var uses int
	s.store.DB.Model(&model.UserStat{}).Where("user_id = ?", userID).Pluck(col, &uses)
	bonus := 1.0
	for _, tier := range ToolMasteryTiers {
		if uses >= tier.Uses {
			bonus = tier.Bonus
		}
	}
	return bonus
}

func (s *Service) HasJournalPages(userID int64) []int {
	var pages []int
	for i := 1; i <= JournalPageCount; i++ {
		pageID := itoa(i)
		var inv model.Inventory
		if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, "journal_page_"+pageID).First(&inv).Error; err == nil && inv.Quantity > 0 {
			pages = append(pages, i)
		}
	}
	return pages
}

func (s *Service) HasAllJournalPages(userID int64) bool {
	for i := 1; i <= JournalPageCount; i++ {
		pageID := itoa(i)
		var inv model.Inventory
		if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, "journal_page_"+pageID).First(&inv).Error; err != nil {
			return false
		}
	}
	return true
}

func (s *Service) GetJournalProgress(userID int64) (int, int) {
	count := 0
	for i := 1; i <= JournalPageCount; i++ {
		pageID := itoa(i)
		var inv model.Inventory
		if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, "journal_page_"+pageID).First(&inv).Error; err == nil && inv.Quantity > 0 {
			count++
		}
	}
	return count, JournalPageCount
}

func (s *Service) GetFossilHarvests(userID int64) []model.UserFossilHarvest {
	var harvests []model.UserFossilHarvest
	s.store.DB.Where("user_id = ?", userID).Find(&harvests)
	return harvests
}

func (s *Service) GetTotalFossilDigs(userID int64) int {
	var harvests []model.UserFossilHarvest
	s.store.DB.Where("user_id = ?", userID).Find(&harvests)
	total := 0
	for _, h := range harvests {
		total += h.Count
	}
	return total
}
