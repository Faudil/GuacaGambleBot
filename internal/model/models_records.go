package model

import "time"

// WinRecord tracks an individual winning payout (net profit) for the casino
// games, powering the "biggest single wins" leaderboard. Game is one of
// "slots" or "coinflip".
type WinRecord struct {
	ID        uint      `gorm:"primaryKey;column:id;autoIncrement"`
	UserID    int64     `gorm:"index:idx_winrecord_user;column:user_id"`
	Game      string    `gorm:"index:idx_winrecord_game_amount;column:game"`
	Amount    int       `gorm:"index:idx_winrecord_game_amount;column:amount"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (WinRecord) TableName() string { return "win_records" }
