package pets

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/model"
)

type TierInfo struct {
	Name     string
	Emoji    string
	MinScore int
	Coins    int
	Crowns   int
	ItemID   string
}

var Tiers = []TierInfo{
	{Name: "Grandmaster", Emoji: "👑", MinScore: -1, Coins: 25000, Crowns: 50, ItemID: "personality_mirror"},
	{Name: "Master", Emoji: "⭐", MinScore: -1, Coins: 10000, Crowns: 20, ItemID: "skill_scroll"},
	{Name: "Diamond", Emoji: "💎", MinScore: 2000, Coins: 5000, Crowns: 10, ItemID: "skill_scroll"},
	{Name: "Platinum", Emoji: "🏆", MinScore: 1000, Coins: 2500, Crowns: 5, ItemID: "volcano_egg"},
	{Name: "Gold", Emoji: "🥇", MinScore: 500, Coins: 1000, Crowns: 2, ItemID: "gold_nugget"},
	{Name: "Silver", Emoji: "🥈", MinScore: 250, Coins: 500, Crowns: 1, ItemID: "iron_ore"},
	{Name: "Bronze", Emoji: "🥉", MinScore: 100, Coins: 200, Crowns: 0, ItemID: ""},
}

func TierForScore(score int, rank int) *TierInfo {
	if rank == 1 {
		return &Tiers[0]
	}
	if rank <= 5 {
		return &Tiers[1]
	}
	for _, t := range Tiers[2:] {
		if score >= t.MinScore {
			return &t
		}
	}
	return nil
}

func CurrentWeekID() string {
	y, w := time.Now().ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

func CurrentWeekStart() time.Time {
	now := time.Now().UTC()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

func NextSundayMidnight() time.Time {
	now := time.Now().UTC()
	daysUntilSunday := (7 - int(now.Weekday())) % 7
	if daysUntilSunday == 0 {
		daysUntilSunday = 7
	}
	next := now.AddDate(0, 0, daysUntilSunday)
	return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, time.UTC)
}

func (s *Service) GetWeeklyRank(userID, serverID int64) (*model.WeeklyRank, error) {
	var wr model.WeeklyRank
	weekID := CurrentWeekID()
	err := s.store.DB.Where("user_id = ? AND server_id = ? AND week_id = ?", userID, serverID, weekID).First(&wr).Error
	if err != nil {
		return nil, err
	}
	return &wr, nil
}

func (s *Service) GetWeeklyRankHistory(userID, serverID int64, limit int) ([]model.WeeklyRank, error) {
	var ranks []model.WeeklyRank
	err := s.store.DB.Where("user_id = ? AND server_id = ?", userID, serverID).
		Order("week_id desc").
		Limit(limit).
		Find(&ranks).Error
	return ranks, err
}

func (s *Service) GetWeeklyLeaderboardForWeek(serverID string, weekID string, limit int) ([]model.WeeklyRank, error) {
	var ranks []model.WeeklyRank
	sid := toInt64(serverID)
	err := s.store.DB.Where("server_id = ? AND week_id = ?", sid, weekID).
		Order("score desc").
		Limit(limit).
		Find(&ranks).Error
	return ranks, err
}

func (s *Service) GetRankPosition(userID, serverID int64) (int, error) {
	weekID := CurrentWeekID()
	var count int64
	err := s.store.DB.Model(&model.WeeklyRank{}).
		Where("server_id = ? AND week_id = ? AND score > (SELECT COALESCE(score,0) FROM weekly_ranks WHERE user_id = ? AND server_id = ? AND week_id = ?)", serverID, weekID, userID, serverID, weekID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count) + 1, nil
}

func (s *Service) AddWeeklyScore(userID, serverID int64, scoreDelta, isWin int) {
	weekID := CurrentWeekID()
	_ = s.store.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "server_id"}, {Name: "week_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"score":  gorm.Expr("score + ?", scoreDelta),
			"wins":   gorm.Expr("wins + ?", isWin),
			"losses": gorm.Expr("losses + ?", 1-isWin),
		}),
	}).Create(&model.WeeklyRank{
		UserID: userID, ServerID: serverID, WeekID: weekID,
		Score: scoreDelta, Wins: isWin, Losses: 1 - isWin,
	}).Error
}

func (s *Service) GetWeeklyLeaderboard(serverID string, limit int) ([]model.WeeklyRank, error) {
	var ranks []model.WeeklyRank
	weekID := CurrentWeekID()
	sid := toInt64(serverID)
	err := s.store.DB.Where("server_id = ? AND week_id = ?", sid, weekID).
		Order("score desc").
		Limit(limit).
		Find(&ranks).Error
	return ranks, err
}

func (s *Service) GetActiveModifier(serverID int64) (*model.WeeklyModifier, error) {
	weekID := CurrentWeekID()
	var m model.WeeklyModifier
	err := s.store.DB.Where("server_id = ? AND week_id = ?", serverID, weekID).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Service) EnsureWeeklyModifier(serverID int64) (*model.WeeklyModifier, error) {
	m, err := s.GetActiveModifier(serverID)
	if err == nil {
		return m, nil
	}
	return s.RollWeeklyModifier(serverID)
}

func (s *Service) RollWeeklyModifier(serverID int64) (*model.WeeklyModifier, error) {
	weekID := CurrentWeekID()
	mod := RandomModifier()
	wm := &model.WeeklyModifier{
		ServerID:  serverID,
		WeekID:    weekID,
		Modifier:  mod.ID,
		Boosted:   joinStrings(mod.Boosted),
		Nerfed:    joinStrings(mod.Nerfed),
		CreatedAt: time.Now().UTC(),
	}
	err := s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "server_id"}, {Name: "week_id"}},
		DoUpdates: clause.Assignments(map[string]any{"modifier": mod.ID, "boosted": joinStrings(mod.Boosted), "nerfed": joinStrings(mod.Nerfed), "created_at": time.Now().UTC()}),
	}).Create(wm).Error
	return wm, err
}

func (s *Service) PerformWeeklyReset(serverID int64) (int, error) {
	weekID := CurrentWeekID()
	var ranks []model.WeeklyRank
	if err := s.store.DB.Where("server_id = ? AND week_id = ?", serverID, weekID).
		Order("score desc").Find(&ranks).Error; err != nil {
		return 0, err
	}

	rewarded := 0
	for i, r := range ranks {
		tier := TierForScore(r.Score, i+1)
		if tier == nil {
			continue
		}
		if tier.Coins > 0 {
			if _, err := s.store.UpdateBalance(r.UserID, tier.Coins); err != nil {
				slog.Error("weekly: failed to award coins", "user", r.UserID, "coins", tier.Coins, "error", err)
			}
		}
		if tier.Crowns > 0 {
			if _, err := s.store.AdjustColumn(r.UserID, "crowns", tier.Crowns); err != nil {
				slog.Error("weekly: failed to award crowns", "user", r.UserID, "crowns", tier.Crowns, "error", err)
			}
		}
		if tier.ItemID != "" {
			if err := s.store.AddItemRaw(s.store.DB, r.UserID, tier.ItemID, 1); err != nil {
				slog.Error("weekly: failed to award item", "user", r.UserID, "item", tier.ItemID, "error", err)
			}
		}
		rewarded++
	}

	if _, err := s.RollWeeklyModifier(serverID); err != nil {
		slog.Error("weekly: failed to roll modifier", "server", serverID, "error", err)
	}
	return rewarded, nil
}

func (s *Service) CalculateScoreDelta(winnerElo, loserElo int) int {
	expected := 1.0 / (1.0 + math.Pow(10, float64(loserElo-winnerElo)/400))
	delta := 32.0 * (1.0 - expected)
	return int(math.Round(delta))
}

func joinStrings(strs []string) string {
	out := ""
	for i, s := range strs {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}

func SplitModStats(s string) []string {
	if s == "" {
		return nil
	}
	parts := make([]string, 0)
	current := ""
	for _, c := range s {
		if c == ',' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func toInt64(s string) int64 {
	var v int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			v = v*10 + int64(c-'0')
		}
	}
	return v
}
