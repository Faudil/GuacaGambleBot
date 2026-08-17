package store

import (
	"guacagamblebot/internal/model"
)

// AddWinRecord stores a single winning payout (net profit) for a casino game so
// it can power the "biggest single wins" leaderboard.
func (s *Store) AddWinRecord(userID int64, game string, amount int) error {
	return s.DB.Create(&model.WinRecord{UserID: userID, Game: game, Amount: amount}).Error
}

// TopWinRecords returns the n biggest single wins for a game, newest first on
// ties.
func (s *Store) TopWinRecords(game string, n int) ([]model.WinRecord, error) {
	var records []model.WinRecord
	if err := s.DB.
		Where("game = ?", game).
		Order("amount desc, created_at asc").
		Limit(n).
		Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}
