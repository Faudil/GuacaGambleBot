package store

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
)

// RepaidLender describes a partial debt repayment to a single lender.
type RepaidLender struct {
	LenderID int64
	Amount   int
}

// Store is the data-access layer over GORM.
type Store struct {
	DB            *gorm.DB
	StartingBalance int
	DefaultPrefix   string
}

func New(db *gorm.DB, cfg *config.Config) *Store {
	return &Store{DB: db, StartingBalance: cfg.StartingBalance, DefaultPrefix: cfg.Prefix}
}

// ensureUser creates the user row with the starting balance if missing.
func (s *Store) ensureUser(userID int64) error {
	var u model.User
	return s.DB.Where(model.User{UserID: userID}).
		Attrs(map[string]any{"balance": s.StartingBalance}).
		FirstOrCreate(&u).Error
}

// GetBalance returns the user's wallet balance, creating the account if needed.
func (s *Store) GetBalance(userID int64) (int, error) {
	if err := s.ensureUser(userID); err != nil {
		return 0, err
	}
	var bal int
	if err := s.DB.Model(&model.User{}).
		Where("user_id = ?", userID).Pluck("balance", &bal).Error; err != nil {
		return 0, err
	}
	return bal, nil
}

// UpdateBalance atomically adjusts the wallet balance by delta and returns the
// new balance. Unlike the original Python helper, the user row is guaranteed to
// exist before the adjustment, so a negative first adjustment can never push the
// balance below the starting amount.
func (s *Store) UpdateBalance(userID int64, delta int) (int, error) {
	if err := s.ensureUser(userID); err != nil {
		return 0, err
	}
	if err := s.DB.Model(&model.User{}).
		Where("user_id = ?", userID).
		UpdateColumn("balance", gorm.Expr("balance + ?", delta)).Error; err != nil {
		return 0, err
	}
	var bal int
	if err := s.DB.Model(&model.User{}).
		Where("user_id = ?", userID).Pluck("balance", &bal).Error; err != nil {
		return 0, err
	}
	return bal, nil
}

// GetBankData returns the wallet and bank balances for a user.
func (s *Store) GetBankData(userID int64) (wallet, bank int, err error) {
	if err = s.ensureUser(userID); err != nil {
		return 0, 0, err
	}
	var u model.User
	if err = s.DB.Where("user_id = ?", userID).First(&u).Error; err != nil {
		return 0, 0, err
	}
	return u.Balance, u.Bank, nil
}

// AdjustColumn atomically adjusts a numeric user column (e.g. "bank") by delta
// and returns the new value. It guarantees the user row exists first.
func (s *Store) AdjustColumn(userID int64, column string, delta int) (int, error) {
	if err := s.ensureUser(userID); err != nil {
		return 0, err
	}
	if err := s.DB.Model(&model.User{}).
		Where("user_id = ?", userID).
		UpdateColumn(column, gorm.Expr(column+" + ?", delta)).Error; err != nil {
		return 0, err
	}
	var v int
	if err := s.DB.Model(&model.User{}).
		Where("user_id = ?", userID).Pluck(column, &v).Error; err != nil {
		return 0, err
	}
	return v, nil
}

// GetTotalDebt returns the sum of all open loans for a borrower.
func (s *Store) GetTotalDebt(userID int64) (int, error) {
	var total *int
	if err := s.DB.Model(&model.Loan{}).
		Where("borrower_id = ?", userID).
		Select("COALESCE(SUM(amount_due), 0)").Scan(&total).Error; err != nil {
		return 0, err
	}
	if total == nil {
		return 0, nil
	}
	return *total, nil
}

// RepayDebt distributes payment across the borrower's loans (oldest first),
// crediting each lender's wallet. It returns the total actually paid and the
// per-lender breakdown.
func (s *Store) RepayDebt(borrowerID int64, payment int) (int, []RepaidLender, error) {
	var lenders []RepaidLender
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var loans []model.Loan
		if err := tx.Where("borrower_id = ?", borrowerID).
			Order("id asc").Find(&loans).Error; err != nil {
			return err
		}
		remaining := payment
		for _, loan := range loans {
			if remaining <= 0 {
				break
			}
			toPay := min(remaining, loan.AmountDue)
			newAmt := loan.AmountDue - toPay
			if newAmt <= 0 {
				if err := tx.Delete(&model.Loan{}, loan.ID).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Model(&model.Loan{}).
					Where("id = ?", loan.ID).
					Update("amount_due", newAmt).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&model.User{}).
				Where("user_id = ?", loan.LenderID).
				UpdateColumn("balance", gorm.Expr("balance + ?", toPay)).Error; err != nil {
				return err
			}
			remaining -= toPay
			lenders = append(lenders, RepaidLender{LenderID: loan.LenderID, Amount: toPay})
		}
		if paid := payment - remaining; paid > 0 {
			if err := tx.Model(&model.User{}).
				Where("user_id = ?", borrowerID).
				UpdateColumn("balance", gorm.Expr("balance - ?", paid)).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, nil, err
	}
	paid := 0
	for _, l := range lenders {
		paid += l.Amount
	}
	return paid, lenders, nil
}

// CheckGameLimit returns whether the user may still play gameName today, plus
// how many uses remain. Limits reset automatically at the start of a new day.
func (s *Store) CheckGameLimit(userID int64, gameName string, maxUsage int) (bool, int, error) {
	today := time.Now().Format("2006-01-02")
	var gl model.GameLimit
	res := s.DB.Where("user_id = ? AND game_name = ?", userID, gameName).First(&gl)
	if res.Error == gorm.ErrRecordNotFound {
		return true, maxUsage, nil
	} else if res.Error != nil {
		return false, 0, res.Error
	}
	if gl.DateStr != today {
		if err := s.DB.Model(&model.GameLimit{}).
			Where("user_id = ? AND game_name = ?", userID, gameName).
			Updates(map[string]any{"date_str": today, "count": 0}).Error; err != nil {
			return false, 0, err
		}
		return true, maxUsage, nil
	}
	if gl.Count >= maxUsage {
		return false, 0, nil
	}
	return true, maxUsage - gl.Count, nil
}

// IncrementGameLimit records one more play of gameName for today.
func (s *Store) IncrementGameLimit(userID int64, gameName string) error {
	today := time.Now().Format("2006-01-02")
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "game_name"}, {Name: "date_str"}},
		DoUpdates: clause.Assignments(map[string]any{
			"count":    gorm.Expr("count + 1"),
			"date_str": today,
		}),
	}).Create(&model.GameLimit{
		UserID: userID, GameName: gameName, DateStr: today, Count: 1,
	}).Error
}

// GetLanguage returns the configured language for a server (defaults to "fr").
func (s *Store) GetLanguage(serverID int64) string {
	if serverID == 0 {
		return "fr"
	}
	var ss model.ServerSetting
	if err := s.DB.Where("server_id = ?", serverID).First(&ss).Error; err != nil || ss.Language == "" {
		return "fr"
	}
	return ss.Language
}

// GetServerSetting returns the guild settings row, or nil when none exists.
func (s *Store) GetServerSetting(serverID int64) (*model.ServerSetting, error) {
	var ss model.ServerSetting
	err := s.DB.Where("server_id = ?", serverID).First(&ss).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ss, nil
}

// SaveServerSetting upserts a guild settings row keyed by server_id.
func (s *Store) SaveServerSetting(ss *model.ServerSetting) error {
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "server_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"announcement_channel_id", "channel_id", "language", "prefix", "enabled", "onboarded_at",
		}),
	}).Create(ss).Error
}

// ServerPrefix returns the command prefix configured for a guild, falling back
// to the global default when no guild-specific prefix is set.
func (s *Store) ServerPrefix(serverID int64) string {
	if serverID == 0 {
		return s.DefaultPrefix
	}
	ss, err := s.GetServerSetting(serverID)
	if err != nil || ss == nil || ss.Prefix == "" {
		return s.DefaultPrefix
	}
	return ss.Prefix
}

// IsEnabled reports whether the bot is active in a guild. Guilds without a
// settings row (or with server_id 0, i.e. DMs) are enabled by default.
func (s *Store) IsEnabled(serverID int64) bool {
	if serverID == 0 {
		return true
	}
	ss, err := s.GetServerSetting(serverID)
	if err != nil || ss == nil {
		return true
	}
	return ss.Enabled
}

// StartDailyQuest initialises (or resets) the active daily quest for a user.
func (s *Store) StartDailyQuest(userID int64, stat string, count int, textKey string) error {
	custom, _ := json.Marshal(map[string]any{
		"target_stat":  stat,
		"target_count": count,
		"text_key":     textKey,
	})
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "quest_id"}},
			DoUpdates: clause.Assignments(map[string]any{"status": "ACTIVE", "started_at": gorm.Expr("CURRENT_TIMESTAMP")}),
		}).Create(&model.UserQuest{UserID: userID, QuestID: "daily_quest", Status: "ACTIVE", StartedAt: time.Now()}).Error; err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "quest_id"}},
			DoUpdates: clause.Assignments(map[string]any{"step_index": 0, "progress_value": 0, "custom_data": string(custom)}),
		}).Create(&model.UserQuestData{UserID: userID, QuestID: "daily_quest", StepIndex: 0, ProgressValue: 0, CustomData: string(custom)}).Error
	})
}

// HasDailyQuestToday reports whether the user already has an active daily quest
// started today.
func (s *Store) HasDailyQuestToday(userID int64) (bool, error) {
	var count int64
	if err := s.DB.Model(&model.UserQuest{}).
		Where("user_id = ? AND quest_id = 'daily_quest' AND status = 'ACTIVE' AND date(started_at, 'localtime') = date('now', 'localtime')", userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// RecordActivity advances the active daily quest when its target stat matches.
func (s *Store) RecordActivity(userID int64, stat string, amount int) error {
	var q model.UserQuest
	if err := s.DB.Where("user_id = ? AND quest_id = 'daily_quest' AND status = 'ACTIVE'", userID).First(&q).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	var d model.UserQuestData
	if err := s.DB.Where("user_id = ? AND quest_id = 'daily_quest'", userID).First(&d).Error; err != nil {
		return err
	}
	var cd map[string]any
	if err := json.Unmarshal([]byte(d.CustomData), &cd); err != nil {
		return err
	}
	if cd["target_stat"] != stat {
		return nil
	}
	targetCount, _ := cd["target_count"].(float64)
	newVal := d.ProgressValue + amount
	completed := newVal >= int(targetCount)
	if err := s.DB.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = 'daily_quest'", userID).
		Update("progress_value", newVal).Error; err != nil {
		return err
	}
	if completed {
		if err := s.DB.Model(&model.UserQuest{}).
			Where("user_id = ? AND quest_id = 'daily_quest'", userID).
			Updates(map[string]any{"status": "COMPLETED", "completed_at": time.Now()}).Error; err != nil {
			return err
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
