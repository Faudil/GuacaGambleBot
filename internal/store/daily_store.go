package store

import (
	"time"

	"gorm.io/gorm/clause"

	"guacagamblebot/internal/model"
)

// DailyHistoryEntry is one past daily quest row, used by the generator for
// anti-repeat selection and streak tracking.
type DailyHistoryEntry struct {
	DateStr   string
	Requestor string
	TurnIn    string
	Completed bool
}

// LogDailyQuest upserts today's daily quest row. It runs at generation so the
// anti-repeat window sees the requestor and turn-in immediately, before the
// quest is completed.
func (s *Store) LogDailyQuest(userID int64, requestor, turnIn string) error {
	row := &model.UserDailyLog{
		UserID: userID, DateStr: time.Now().Format("2006-01-02"),
		Requestor: requestor, TurnInItem: turnIn,
	}
	return s.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "date_str"}},
		DoUpdates: clause.Assignments(map[string]any{
			"requestor": requestor, "turnin_item": turnIn, "completed": false,
		}),
	}).Create(row).Error
}

// CompleteDailyQuest marks today's daily quest row as completed.
func (s *Store) CompleteDailyQuest(userID int64, requestor, turnIn string) error {
	return s.DB.Model(&model.UserDailyLog{}).
		Where("user_id = ? AND date_str = ?", userID, time.Now().Format("2006-01-02")).
		Updates(map[string]any{
			"requestor": requestor, "turnin_item": turnIn,
			"completed": true, "completed_at": time.Now(),
		}).Error
}

// RecentDailyQuests returns the user's most recent daily quest rows, newest
// first, capped at limit.
func (s *Store) RecentDailyQuests(userID int64, limit int) ([]DailyHistoryEntry, error) {
	var rows []model.UserDailyLog
	if err := s.DB.Where("user_id = ?", userID).
		Order("date_str desc").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]DailyHistoryEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, DailyHistoryEntry{
			DateStr: r.DateStr, Requestor: r.Requestor, TurnIn: r.TurnInItem, Completed: r.Completed,
		})
	}
	return out, nil
}

// DailyStreak counts consecutive completed daily quests. An in-progress quest
// today does not break the streak; a missed full day resets it.
func (s *Store) DailyStreak(userID int64) int {
	var rows []model.UserDailyLog
	if err := s.DB.Where("user_id = ? AND completed = ?", userID, true).
		Order("date_str desc").Limit(30).Find(&rows).Error; err != nil {
		return 0
	}
	// Walk backwards from today; if today is not completed yet, start from
	// yesterday so the streak survives until the day actually ends.
	day := time.Now()
	if len(rows) == 0 || rows[0].DateStr != day.Format("2006-01-02") {
		day = day.AddDate(0, 0, -1)
	}
	streak := 0
	for _, r := range rows {
		if r.DateStr != day.Format("2006-01-02") {
			break
		}
		streak++
		day = day.AddDate(0, 0, -1)
	}
	return streak
}
