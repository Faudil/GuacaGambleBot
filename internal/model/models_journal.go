package model

import "time"

// UserJournalEntry tracks a player's progress along one journal path.
// StepIndex points at the next step to complete; all steps before it are done.
type UserJournalEntry struct {
	UserID    int64     `gorm:"primaryKey;column:user_id"`
	PathID    string    `gorm:"primaryKey;column:path_id"`
	StepIndex int       `gorm:"column:step_index;default:0"`
	Progress  int       `gorm:"column:progress;default:0"`
	Tracked   bool      `gorm:"column:tracked;default:false"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (UserJournalEntry) TableName() string { return "user_journal_entries" }

// UserJournalMastery marks a player who completed every journal path. The
// existence of this row is a secret until earned: nothing surfaces it early.
type UserJournalMastery struct {
	UserID     int64     `gorm:"primaryKey;column:user_id"`
	UnlockedAt time.Time `gorm:"column:unlocked_at"`
}

func (UserJournalMastery) TableName() string { return "user_journal_mastery" }
