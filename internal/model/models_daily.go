package model

import "time"

// UserDailyLog records one daily quest per user per day. The row is upserted
// when the quest is generated (so anti-repeat sees it immediately) and marked
// completed when the turn-in is delivered.
type UserDailyLog struct {
	ID          uint       `gorm:"primaryKey;column:id;autoIncrement"`
	UserID      int64      `gorm:"uniqueIndex:idx_udl_user_date;column:user_id"`
	DateStr     string     `gorm:"uniqueIndex:idx_udl_user_date;column:date_str"` // YYYY-MM-DD
	Requestor   string     `gorm:"column:requestor"`
	TurnInItem  string     `gorm:"column:turnin_item"`
	Completed   bool       `gorm:"column:completed;default:false"`
	CompletedAt *time.Time `gorm:"column:completed_at"`
}

func (UserDailyLog) TableName() string { return "user_daily_log" }
