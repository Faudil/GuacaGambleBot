package store

import (
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
)

// ErrInsufficientFunds is returned by money-moving operations that validate the
// user's balance inside a transaction (Debit, Transfer, BankDeposit).
var ErrInsufficientFunds = errors.New("insufficient funds")

// ErrBankFull is returned by BankDeposit when the bank already holds its
// maximum amount (maxBank).
var ErrBankFull = errors.New("bank full")

// RepaidLender describes a partial debt repayment to a single lender.
type RepaidLender struct {
	LenderID int64
	Amount   int
}

// QuestAdvanceFn is called by RecordActivity when an activity step reaches its
// target. Implementations should advance the quest step with proper next-step
// custom_data. Returns whether the quest fully completed, plus the i18n text
// key of the next step (empty when the quest is complete or has no next step).
type QuestAdvanceFn func(userID int64, questID string) (completed bool, nextStepKey string, err error)

// QuestNotification reports a quest event produced by RecordActivity so cogs
// can surface it to the user. Completed is true when the whole quest finished,
// false when only a step advanced (NextStepKey then points at the new objective).
type QuestNotification struct {
	QuestID     string
	Completed   bool
	NextStepKey string
}

// JournalAdvanceFn is called by RecordActivity after quest logic so the journal
// can re-check its milestone checks on any recorded activity.
type JournalAdvanceFn func(userID int64, stat string, amount int)

// WorldSpawnFn is called by RecordActivity after quest/journal logic to give the
// world-quest system a chance to spawn a new side quest. It should be cheap and
// return quickly; implementations decide internally whether to roll and start.
type WorldSpawnFn func(userID int64, stat string, amount int)

// Store is the data-access layer over GORM.
type Store struct {
	DB              *gorm.DB
	StartingBalance int
	DefaultPrefix   string
	questAdvanceFn  QuestAdvanceFn
	journalFn       JournalAdvanceFn
	worldSpawnFn    WorldSpawnFn

	settingsMu    sync.Mutex
	settingsCache map[int64]serverSettingsEntry
}

func (s *Store) SetQuestAdvanceFn(fn QuestAdvanceFn) { s.questAdvanceFn = fn }

// SetJournalFn registers the journal advance hook (journal.New wires this).
func (s *Store) SetJournalFn(fn JournalAdvanceFn) { s.journalFn = fn }

func (s *Store) SetWorldSpawnFn(fn WorldSpawnFn) { s.worldSpawnFn = fn }

func New(db *gorm.DB, cfg *config.Config) *Store {
	return &Store{DB: db, StartingBalance: cfg.StartingBalance, DefaultPrefix: cfg.Prefix}
}

// ensureUserTx creates the user row with the starting balance if missing.
func (s *Store) ensureUserTx(tx *gorm.DB, userID int64) error {
	var u model.User
	return tx.Where(model.User{UserID: userID}).
		Attrs(map[string]any{"balance": s.StartingBalance}).
		FirstOrCreate(&u).Error
}

// ensureUser creates the user row with the starting balance if missing.
func (s *Store) ensureUser(userID int64) error {
	return s.ensureUserTx(s.DB, userID)
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
	s.creditMoneyEarned(s.DB, userID, delta)
	var bal int
	if err := s.DB.Model(&model.User{}).
		Where("user_id = ?", userID).Pluck("balance", &bal).Error; err != nil {
		return 0, err
	}
	return bal, nil
}

// creditMoneyEarned accumulates the money_earned stat on every positive balance
// credit, backing the eco_rich achievement. It is a coarse proxy: refunds and
// loan disbursements count as earned money too. db may be the pool or an outer
// transaction handle.
func (s *Store) creditMoneyEarned(db *gorm.DB, userID int64, amount int) {
	if amount <= 0 {
		return
	}
	if err := db.Where("user_id = ?", userID).FirstOrCreate(&model.UserStat{UserID: userID}).Error; err != nil {
		return
	}
	_ = db.Model(&model.UserStat{}).
		Where("user_id = ?", userID).
		UpdateColumn("money_earned", gorm.Expr("money_earned + ?", amount)).Error
}

// UpdateBalanceTx adjusts the wallet balance by delta inside an existing
// transaction, ensuring the user row exists first. Callers running within an
// outer s.DB.Transaction must use this instead of UpdateBalance: the transaction
// holds the pool's single SQLite connection, so a nested pool query would
// deadlock (maxOpenConns is 1 in production).
func (s *Store) UpdateBalanceTx(tx *gorm.DB, userID int64, delta int) error {
	if err := s.ensureUserTx(tx, userID); err != nil {
		return err
	}
	if err := tx.Model(&model.User{}).
		Where("user_id = ?", userID).
		UpdateColumn("balance", gorm.Expr("balance + ?", delta)).Error; err != nil {
		return err
	}
	s.creditMoneyEarned(tx, userID, delta)
	return nil
}

// Debit checks that the user can afford amount and deducts it from the wallet
// in a single transaction, so concurrent callers cannot overdraw the balance.
// It returns the new balance.
func (s *Store) Debit(userID int64, amount int) (int, error) {
	var newBal int
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.ensureUserTx(tx, userID); err != nil {
			return err
		}
		var bal int
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", userID).Pluck("balance", &bal).Error; err != nil {
			return err
		}
		if bal < amount {
			return ErrInsufficientFunds
		}
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", userID).
			UpdateColumn("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
			return err
		}
		newBal = bal - amount
		return nil
	})
	if err != nil {
		return 0, err
	}
	return newBal, nil
}

// Transfer moves amount from sender to recipient atomically, refusing the
// transfer when the sender cannot afford it. It returns both new balances.
func (s *Store) Transfer(sender, recipient int64, amount int) (senderBal, recipientBal int, err error) {
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.ensureUserTx(tx, sender); err != nil {
			return err
		}
		if err := s.ensureUserTx(tx, recipient); err != nil {
			return err
		}
		var bal int
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", sender).Pluck("balance", &bal).Error; err != nil {
			return err
		}
		if bal < amount {
			return ErrInsufficientFunds
		}
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", sender).
			UpdateColumn("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", recipient).
			UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", sender).Pluck("balance", &senderBal).Error; err != nil {
			return err
		}
		return tx.Model(&model.User{}).
			Where("user_id = ?", recipient).Pluck("balance", &recipientBal).Error
	})
	return senderBal, recipientBal, err
}

// BankDeposit moves amount from the wallet into the bank in a single
// transaction, never letting the bank exceed maxBank: when amount would
// overflow the bank, only the remaining space is deposited. It returns the
// amount actually deposited and the new wallet and bank balances.
func (s *Store) BankDeposit(userID int64, amount, maxBank int) (deposited, wallet, bank int, err error) {
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.ensureUserTx(tx, userID); err != nil {
			return err
		}
		var bal int
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", userID).Pluck("balance", &bal).Error; err != nil {
			return err
		}
		var curBank int
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", userID).Pluck("bank", &curBank).Error; err != nil {
			return err
		}
		if curBank >= maxBank {
			return ErrBankFull
		}
		actual := min(amount, maxBank-curBank)
		if bal < actual {
			return ErrInsufficientFunds
		}
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", userID).
			UpdateColumn("balance", gorm.Expr("balance - ?", actual)).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", userID).
			UpdateColumn("bank", gorm.Expr("bank + ?", actual)).Error; err != nil {
			return err
		}
		var u model.User
		if err := tx.Where("user_id = ?", userID).First(&u).Error; err != nil {
			return err
		}
		deposited, wallet, bank = actual, u.Balance, u.Bank
		return nil
	})
	return deposited, wallet, bank, err
}

// BankWithdraw moves amount from the bank into the wallet in a single
// transaction. It returns the new wallet and bank balances.
func (s *Store) BankWithdraw(userID int64, amount int) (wallet, bank int, err error) {
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.ensureUserTx(tx, userID); err != nil {
			return err
		}
		var bal int
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", userID).Pluck("bank", &bal).Error; err != nil {
			return err
		}
		if bal < amount {
			return ErrInsufficientFunds
		}
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", userID).
			UpdateColumn("bank", gorm.Expr("bank - ?", amount)).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", userID).
			UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
			return err
		}
		var u model.User
		if err := tx.Where("user_id = ?", userID).First(&u).Error; err != nil {
			return err
		}
		wallet, bank = u.Balance, u.Bank
		return nil
	})
	return wallet, bank, err
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
	res := s.DB.Where("user_id = ? AND game_name = ? AND date_str = ?", userID, gameName, today).First(&gl)
	if res.Error == gorm.ErrRecordNotFound {
		return true, maxUsage, nil
	} else if res.Error != nil {
		return false, 0, res.Error
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
		Columns: []clause.Column{{Name: "user_id"}, {Name: "game_name"}, {Name: "date_str"}},
		DoUpdates: clause.Assignments(map[string]any{
			"count":    gorm.Expr("count + 1"),
			"date_str": today,
		}),
	}).Create(&model.GameLimit{
		UserID: userID, GameName: gameName, DateStr: today, Count: 1,
	}).Error
}

// RemoveInventoryItem removes quantity units of itemID from a user's inventory.
// The key is normalized to the canonical item ID so display names work too.
func (s *Store) RemoveInventoryItem(userID int64, itemID string, quantity int) error {
	canonical := items.Canonical(itemID)
	if canonical == "" {
		return nil
	}
	return s.DB.Model(&model.Inventory{}).
		Where("user_id = ? AND item_id = ? AND quantity > 0", userID, canonical).
		UpdateColumn("quantity", gorm.Expr("quantity - ?", quantity)).Error
}

// GrantGameLimitCredit refunds credits from today's usage count for gameName,
// never going below zero. Missing rows (no usage today) are left untouched.
func (s *Store) GrantGameLimitCredit(userID int64, gameName string, credits int) error {
	if credits <= 0 {
		return nil
	}
	today := time.Now().Format("2006-01-02")
	res := s.DB.Model(&model.GameLimit{}).
		Where("user_id = ? AND game_name = ? AND date_str = ? AND count > 0", userID, gameName, today).
		UpdateColumn("count", gorm.Expr("MAX(count - ?, 0)", credits))
	return res.Error
}

// ResetGameLimit clears today's usage for gameName so the full daily limit is
// available again. Rows from previous days are irrelevant to CheckGameLimit.
func (s *Store) ResetGameLimit(userID int64, gameName string) error {
	return s.DB.Where("user_id = ? AND game_name = ?", userID, gameName).
		Delete(&model.GameLimit{}).Error
}

// GetLanguage returns the configured language for a server (defaults to "fr").
func (s *Store) GetLanguage(serverID int64) string {
	if serverID == 0 {
		return "fr"
	}
	e := s.cachedServerSettings(serverID)
	if !e.hasRow || e.language == "" {
		return "fr"
	}
	return e.language
}

// JournalScene is a queued atmospheric scene (Chronicler intro, rank-up moment,
// recognition line). Key is an i18n key, Params its replacements. DM requests
// private-message delivery with a fallback to the activity result.
type JournalScene struct {
	Key    string
	Params map[string]any
	DM     bool
}

// PushJournalScene queues a scene for the user, surfaced after their next
// activity.
func (s *Store) PushJournalScene(userID int64, sc JournalScene) {
	params := "{}"
	if b, err := json.Marshal(sc.Params); err == nil {
		params = string(b)
	}
	if err := s.DB.Create(&model.JournalScene{
		UserID: userID, Key: sc.Key, Params: params, DM: sc.DM,
	}).Error; err != nil {
		slog.Error("failed to store journal scene", "user_id", userID, "error", err)
	}
}

// PopJournalScene returns the oldest pending scene for the user, consuming it.
// Scenes older than 24 hours are purged.
func (s *Store) PopJournalScene(userID int64) (JournalScene, bool) {
	var rows []model.JournalScene
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND created_at < ?", userID, time.Now().Add(-24*time.Hour)).
			Delete(&model.JournalScene{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).
			Order("id asc").Limit(1).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Delete(&model.JournalScene{}, rows[0].ID).Error
	})
	if err != nil || len(rows) == 0 {
		return JournalScene{}, false
	}
	var params map[string]any
	_ = json.Unmarshal([]byte(rows[0].Params), &params)
	return JournalScene{Key: rows[0].Key, Params: params, DM: rows[0].DM}, true
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
	s.settingsMu.Lock()
	delete(s.settingsCache, ss.ServerID)
	s.settingsMu.Unlock()
	return s.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "server_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"announcement_channel_id", "channel_id", "language", "prefix", "enabled", "onboarded_at", "universe",
		}),
	}).Create(ss).Error
}

// ServerPrefix returns the command prefix configured for a guild, falling back
// to the global default when no guild-specific prefix is set.
func (s *Store) ServerPrefix(serverID int64) string {
	if serverID == 0 {
		return s.DefaultPrefix
	}
	e := s.cachedServerSettings(serverID)
	if !e.hasRow || e.prefix == "" {
		return s.DefaultPrefix
	}
	return e.prefix
}

// IsEnabled reports whether the bot is active in a guild. Guilds without a
// settings row (or with server_id 0, i.e. DMs) are enabled by default.
func (s *Store) IsEnabled(serverID int64) bool {
	if serverID == 0 {
		return true
	}
	e := s.cachedServerSettings(serverID)
	return !e.hasRow || e.enabled
}

// serverSettingsEntry is a cached copy of a guild's settings row.
type serverSettingsEntry struct {
	hasRow   bool
	language string
	prefix   string
	enabled  bool
	fetched  time.Time
}

// serverSettingsCacheTTL bounds how stale a cached settings row may be. Reads
// happen on every interaction and every message, so serving them from memory
// keeps per-event DB traffic off the hot SQLite path.
const serverSettingsCacheTTL = 10 * time.Second

// cachedServerSettings returns the guild settings from memory, fetching the
// row on a miss. Writes invalidate the entry via SaveServerSetting.
func (s *Store) cachedServerSettings(serverID int64) serverSettingsEntry {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	if e, ok := s.settingsCache[serverID]; ok && time.Since(e.fetched) < serverSettingsCacheTTL {
		return e
	}
	e := serverSettingsEntry{fetched: time.Now()}
	if ss, err := s.GetServerSetting(serverID); err == nil && ss != nil {
		e.hasRow = true
		e.language = ss.Language
		e.prefix = ss.Prefix
		e.enabled = ss.Enabled
	}
	if s.settingsCache == nil {
		s.settingsCache = make(map[int64]serverSettingsEntry)
	}
	s.settingsCache[serverID] = e
	return e
}

// ─── Procedural daily quests ────────────────────────────────────
// The daily quest's generated recipe is stored as JSON in the daily_quest
// custom_data column. The recipe is data-driven: a requestor NPC, 2-3 steps
// (activity steps then a final turn-in) and a small reward.

// DailyStepKind discriminates the two daily quest step shapes.
type DailyStepKind string

const (
	DailyStepActivity DailyStepKind = "activity"
	DailyStepTurnIn   DailyStepKind = "turnin"
)

// DailyStep is one step of a generated daily quest recipe.
type DailyStep struct {
	Kind    DailyStepKind  `json:"kind"`
	Stat    string         `json:"stat,omitempty"`  // activity steps
	Count   int            `json:"count,omitempty"` // activity steps
	Zone    string         `json:"zone,omitempty"`  // hunt_<zone> steps
	Items   map[string]int `json:"items,omitempty"` // turn-in steps: item id -> qty
	TextKey string         `json:"text_key"`        // i18n key for this step's line
}

// DailyReward is the reward attached to a generated daily quest.
type DailyReward struct {
	Money     int    `json:"money"`
	ItemID    string `json:"item_id,omitempty"` // small item, or the jackpot egg
	Crowns    int    `json:"crowns"`
	RepNPC    string `json:"rep_npc,omitempty"` // NPC gaining reputation
	RepPoints int    `json:"rep_points"`
}

// DailyRecipe is the full generated daily quest.
type DailyRecipe struct {
	Requestor string      `json:"requestor"`
	TitleKey  string      `json:"title_key"`
	IntroKey  string      `json:"intro_key"`
	MoodKey   string      `json:"mood_key,omitempty"`  // atmospheric intro prefix
	ThankKey  string      `json:"thank_key,omitempty"` // requestor's completion line
	Steps     []DailyStep `json:"steps"`               // last step is always a turn-in
	Reward    DailyReward `json:"reward"`
}

// TurnInItem returns the item id requested by the final turn-in step, or ""
// when the recipe has no turn-in.
func (r *DailyRecipe) TurnInItem() string {
	if len(r.Steps) == 0 {
		return ""
	}
	last := r.Steps[len(r.Steps)-1]
	for id := range last.Items {
		return id
	}
	return ""
}

var (
	// ErrDailyNotActive is returned when no active daily quest exists.
	ErrDailyNotActive = errors.New("no active daily quest")
	// ErrDailyNotTurnIn is returned when the current daily step is not a turn-in.
	ErrDailyNotTurnIn = errors.New("current daily step is not a turn-in")
)

// DailyMissingItem describes one item the player lacks for a turn-in step.
type DailyMissingItem struct {
	ItemID string
	Needed int
	Have   int
}

// DailyMissingItemsError is returned by ClaimDailyTurnIn when the player does
// not hold the requested items yet.
type DailyMissingItemsError struct {
	Items []DailyMissingItem
}

func (e *DailyMissingItemsError) Error() string {
	return "daily turn-in items missing"
}

// StartDailyQuest initialises (or resets) the active daily quest for a user
// with a generated recipe (JSON).
func (s *Store) StartDailyQuest(userID int64, recipeJSON string) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "quest_id"}},
			DoUpdates: clause.Assignments(map[string]any{"status": "ACTIVE", "started_at": gorm.Expr("CURRENT_TIMESTAMP")}),
		}).Create(&model.UserQuest{UserID: userID, QuestID: "daily_quest", Status: "ACTIVE", StartedAt: time.Now()}).Error; err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "quest_id"}},
			DoUpdates: clause.Assignments(map[string]any{"step_index": 0, "progress_value": 0, "custom_data": recipeJSON}),
		}).Create(&model.UserQuestData{UserID: userID, QuestID: "daily_quest", StepIndex: 0, ProgressValue: 0, CustomData: recipeJSON}).Error
	})
}

// GetDailyRecipe loads and parses the user's daily quest recipe. Returns
// ErrDailyNotActive when no daily quest row exists.
func (s *Store) GetDailyRecipe(userID int64) (*DailyRecipe, error) {
	var d model.UserQuestData
	if err := s.DB.Where("user_id = ? AND quest_id = 'daily_quest'", userID).First(&d).Error; err != nil {
		return nil, ErrDailyNotActive
	}
	var recipe DailyRecipe
	if err := json.Unmarshal([]byte(d.CustomData), &recipe); err != nil {
		return nil, err
	}
	return &recipe, nil
}

// GetDailyQuestData returns the user's daily quest progress row.
func (s *Store) GetDailyQuestData(userID int64) (*model.UserQuestData, error) {
	var d model.UserQuestData
	if err := s.DB.Where("user_id = ? AND quest_id = 'daily_quest'", userID).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// HasDailyQuestToday reports whether the user already received a daily quest
// today, whether it is still active or already completed — one quest per day.
func (s *Store) HasDailyQuestToday(userID int64) (bool, error) {
	var count int64
	if err := s.DB.Model(&model.UserQuest{}).
		Where("user_id = ? AND quest_id = 'daily_quest' AND date(started_at, 'localtime') = date('now', 'localtime')", userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// RecordActivity advances any active quest whose current step targets the given
// stat (e.g. "items_mined"). For daily_quest the whole quest is marked completed
// when the target is reached; for other quests (e.g. tutorial) the step index is
// advanced so the user can continue the narrative.
func (s *Store) RecordActivity(userID int64, stat string, amount int) error {
	var active []model.UserQuest
	if err := s.DB.Where("user_id = ? AND status = 'ACTIVE'", userID).Find(&active).Error; err != nil {
		return err
	}
	for _, q := range active {
		if err := s.recordActivityForQuest(userID, q, stat, amount); err != nil {
			slog.Warn("RecordActivity failed", "user_id", userID, "quest_id", q.QuestID, "err", err)
		}
	}
	if s.journalFn != nil {
		s.journalFn(userID, stat, amount)
	}
	if s.worldSpawnFn != nil {
		s.worldSpawnFn(userID, stat, amount)
	}
	return nil
}

// recordActivityForQuest atomically increments the quest progress inside a
// transaction. The daily quest advances step by step (activity steps move to
// the next step, the final turn-in is handled by ClaimDailyTurnIn); other
// quests (e.g. tutorial) keep their ACTIVE status and the step
// advancement/completion is delegated to the quest advancement hook, which
// runs outside the transaction because it uses the store's own connection,
// which would deadlock inside it.
func (s *Store) recordActivityForQuest(userID int64, q model.UserQuest, stat string, amount int) error {
	var reached, daily bool
	var nextKey string
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var d model.UserQuestData
		if err := tx.Where("user_id = ? AND quest_id = ?", userID, q.QuestID).First(&d).Error; err != nil {
			return err
		}
		if q.QuestID == "daily_quest" {
			return s.recordDailyActivity(tx, userID, d, stat, amount, &reached, &daily, &nextKey)
		}
		var cd map[string]any
		if err := json.Unmarshal([]byte(d.CustomData), &cd); err != nil {
			slog.Warn("RecordActivity: json unmarshal failed", "user_id", userID, "quest_id", q.QuestID, "custom_data", d.CustomData, "err", err)
			return nil
		}
		if cd["target_stat"] != stat {
			return nil
		}
		targetCount, _ := cd["target_count"].(float64)
		if err := tx.Model(&model.UserQuestData{}).
			Where("user_id = ? AND quest_id = ?", userID, q.QuestID).
			UpdateColumn("progress_value", gorm.Expr("progress_value + ?", amount)).Error; err != nil {
			return err
		}
		var newVal int
		if err := tx.Model(&model.UserQuestData{}).
			Where("user_id = ? AND quest_id = ?", userID, q.QuestID).
			Pluck("progress_value", &newVal).Error; err != nil {
			return err
		}
		if newVal < int(targetCount) {
			return nil
		}
		reached = true
		daily = false
		return nil
	})
	if err != nil {
		return err
	}
	if !reached {
		return nil
	}
	if daily {
		s.pushQuestNotification(userID, QuestNotification{QuestID: q.QuestID, Completed: true})
		return nil
	}
	if nextKey != "" {
		s.pushQuestNotification(userID, QuestNotification{QuestID: q.QuestID, Completed: false, NextStepKey: nextKey})
	}
	if s.questAdvanceFn == nil {
		return nil
	}
	completed, nextKey, err := s.questAdvanceFn(userID, q.QuestID)
	if err != nil {
		slog.Warn("RecordActivity: questAdvanceFn failed", "quest_id", q.QuestID, "err", err)
	} else {
		s.pushQuestNotification(userID, QuestNotification{QuestID: q.QuestID, Completed: completed, NextStepKey: nextKey})
	}
	return nil
}

// recordDailyActivity advances the procedural daily quest when the current
// step is an activity step matching the recorded stat. On reaching the step
// target the quest moves to the next step (out flags: reached=true when the
// final step completed, daily=true, nextKey the following step's text key).
func (s *Store) recordDailyActivity(tx *gorm.DB, userID int64, d model.UserQuestData, stat string, amount int, reached, daily *bool, nextKey *string) error {
	var recipe DailyRecipe
	if err := json.Unmarshal([]byte(d.CustomData), &recipe); err != nil {
		slog.Warn("recordDailyActivity: json unmarshal failed", "user_id", userID, "custom_data", d.CustomData, "err", err)
		return nil
	}
	if d.StepIndex >= len(recipe.Steps) {
		return nil
	}
	step := recipe.Steps[d.StepIndex]
	if step.Kind != DailyStepActivity || step.Stat != stat {
		return nil
	}
	if err := tx.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = 'daily_quest'", userID).
		UpdateColumn("progress_value", gorm.Expr("progress_value + ?", amount)).Error; err != nil {
		return err
	}
	var newVal int
	if err := tx.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = 'daily_quest'", userID).
		Pluck("progress_value", &newVal).Error; err != nil {
		return err
	}
	if newVal < step.Count {
		return nil
	}
	next := d.StepIndex + 1
	if next >= len(recipe.Steps) {
		res := tx.Model(&model.UserQuest{}).
			Where("user_id = ? AND quest_id = 'daily_quest' AND status = 'ACTIVE'", userID).
			Updates(map[string]any{"status": "COMPLETED", "completed_at": time.Now()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		*reached = true
		*daily = true
		return s.grantDailyQuestReward(tx, userID, &recipe.Reward)
	}
	if err := tx.Model(&model.UserQuestData{}).
		Where("user_id = ? AND quest_id = 'daily_quest'", userID).
		Updates(map[string]any{"step_index": next, "progress_value": 0}).Error; err != nil {
		return err
	}
	// Step texts carry requestor/item placeholders, so the notification uses a
	// generic "one objective done" line and the detail lives in /daily.
	*reached = true
	*nextKey = "quests.daily.step_done"
	return nil
}

// ClaimDailyTurnIn delivers the current turn-in step's items and advances the
// daily quest. When the turn-in was the final step the quest is completed and
// its reward granted inside the same transaction. Returns completed=true and a
// notification is queued in that case.
func (s *Store) ClaimDailyTurnIn(userID int64) (completed bool, err error) {
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var uq model.UserQuest
		if err := tx.Where("user_id = ? AND quest_id = 'daily_quest' AND status = 'ACTIVE'", userID).First(&uq).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDailyNotActive
			}
			return err
		}
		var d model.UserQuestData
		if err := tx.Where("user_id = ? AND quest_id = 'daily_quest'", userID).First(&d).Error; err != nil {
			return err
		}
		var recipe DailyRecipe
		if err := json.Unmarshal([]byte(d.CustomData), &recipe); err != nil {
			return err
		}
		if d.StepIndex >= len(recipe.Steps) {
			return ErrDailyNotActive
		}
		step := recipe.Steps[d.StepIndex]
		if step.Kind != DailyStepTurnIn {
			return ErrDailyNotTurnIn
		}
		var missing []DailyMissingItem
		for itemID, qty := range step.Items {
			var inv model.Inventory
			err := tx.Where("user_id = ? AND item_id = ?", userID, itemID).First(&inv).Error
			have := 0
			if err == nil {
				have = inv.Quantity
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if have < qty {
				missing = append(missing, DailyMissingItem{ItemID: itemID, Needed: qty, Have: have})
			}
		}
		if len(missing) > 0 {
			return &DailyMissingItemsError{Items: missing}
		}
		for itemID, qty := range step.Items {
			if err := tx.Model(&model.Inventory{}).
				Where("user_id = ? AND item_id = ?", userID, itemID).
				UpdateColumn("quantity", gorm.Expr("quantity - ?", qty)).Error; err != nil {
				return err
			}
		}
		next := d.StepIndex + 1
		if next >= len(recipe.Steps) {
			if err := tx.Model(&model.UserQuest{}).
				Where("user_id = ? AND quest_id = 'daily_quest'", userID).
				Updates(map[string]any{"status": "COMPLETED", "completed_at": time.Now()}).Error; err != nil {
				return err
			}
			if err := s.grantDailyQuestReward(tx, userID, &recipe.Reward); err != nil {
				return err
			}
			completed = true
			return nil
		}
		return tx.Model(&model.UserQuestData{}).
			Where("user_id = ? AND quest_id = 'daily_quest'", userID).
			Updates(map[string]any{"step_index": next, "progress_value": 0}).Error
	})
	if err != nil {
		return false, err
	}
	if completed {
		s.pushQuestNotification(userID, QuestNotification{QuestID: "daily_quest", Completed: true})
	}
	return completed, nil
}

// PushQuestNotification queues a quest event so the user's next command can
// surface it. Exported for the world-quest spawn hook (quests service).
func (s *Store) PushQuestNotification(userID int64, n QuestNotification) {
	s.pushQuestNotification(userID, n)
}

// pushQuestNotification queues a quest event so the user's next command can
// surface it. Notifications older than 24 hours are purged on pop.
func (s *Store) pushQuestNotification(userID int64, n QuestNotification) {
	if err := s.DB.Create(&model.QuestNotification{
		UserID: userID, QuestID: n.QuestID, Completed: n.Completed, NextStepKey: n.NextStepKey,
	}).Error; err != nil {
		slog.Error("failed to store quest notification", "user_id", userID, "error", err)
	}
}

// PopQuestNotification returns the oldest pending quest notification for the
// user, consuming it. Returns ok=false when nothing is pending.
func (s *Store) PopQuestNotification(userID int64) (QuestNotification, bool) {
	var rows []model.QuestNotification
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND created_at < ?", userID, time.Now().Add(-24*time.Hour)).
			Delete(&model.QuestNotification{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).
			Order("id asc").Limit(1).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Delete(&model.QuestNotification{}, rows[0].ID).Error
	})
	if err != nil {
		slog.Error("failed to pop quest notification", "user_id", userID, "error", err)
		return QuestNotification{}, false
	}
	if len(rows) == 0 {
		return QuestNotification{}, false
	}
	return QuestNotification{QuestID: rows[0].QuestID, Completed: rows[0].Completed, NextStepKey: rows[0].NextStepKey}, true
}

// grantDailyQuestReward hands out the reward promised by a daily quest recipe:
// credits, a small item and crowns, all idempotently inside the same
// transaction as the completion. Reputation is granted by the dailyquest
// service (it lives outside the store's transaction).
func (s *Store) grantDailyQuestReward(tx *gorm.DB, userID int64, r *DailyReward) error {
	if r == nil {
		return nil
	}
	if r.Money > 0 {
		if err := s.UpdateBalanceTx(tx, userID, r.Money); err != nil {
			return err
		}
	}
	if r.Crowns > 0 {
		if err := s.ensureUserTx(tx, userID); err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).
			Where("user_id = ?", userID).
			UpdateColumn("crowns", gorm.Expr("crowns + ?", r.Crowns)).Error; err != nil {
			return err
		}
	}
	if r.ItemID != "" {
		if err := s.AddItemRaw(tx, userID, r.ItemID, 1); err != nil {
			return err
		}
	}
	return nil
}

// CreateQuest creates a new quest entry and its step data for a user.
func (s *Store) CreateQuest(userID int64, questID string) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model.UserQuest{
			UserID: userID, QuestID: questID, Status: "ACTIVE", StartedAt: time.Now(),
		}).Error; err != nil {
			return err
		}
		return tx.Create(&model.UserQuestData{
			UserID: userID, QuestID: questID, StepIndex: 0, ProgressValue: 0, CustomData: "{}",
		}).Error
	})
}

// GetRandomActivePetPair returns two active (is_active=1) pets with level >=
// minLevel, preferring a pair whose Elo difference is within eloRange. If the
// range-constrained query fails it falls back to the closest Elo match, exactly
// as the Python get_random_pet_and_opponent does.
func (s *Store) GetRandomActivePetPair(minLevel, eloRange int) (pet1, pet2 *model.UserPet, err error) {
	var p1, p2 model.UserPet
	if err := s.DB.
		Where("level >= ? AND is_active = ?", minLevel, true).
		Order("RANDOM()").
		First(&p1).Error; err != nil {
		return nil, nil, err
	}

	if err := s.DB.
		Where("level >= ? AND id != ? AND ABS(elo - ?) <= ? AND is_active = ?", minLevel, p1.ID, p1.Elo, eloRange, true).
		Order("RANDOM()").
		First(&p2).Error; err == nil {
		return &p1, &p2, nil
	}

	// Fallback: closest Elo match.
	if err := s.DB.
		Where("level >= ? AND id != ? AND is_active = ?", minLevel, p1.ID, true).
		Order(gorm.Expr("ABS(elo - ?) ASC, RANDOM()", p1.Elo)).
		First(&p2).Error; err != nil {
		return &p1, nil, nil
	}
	return &p1, &p2, nil
}

// CheckCooldown returns true if the cooldown for the given activity has elapsed.
func (s *Store) CheckCooldown(userID int64, activity string, duration time.Duration) (bool, error) {
	var cd model.Cooldown
	err := s.DB.Where("user_id = ? AND activity_name = ?", userID, activity).First(&cd).Error
	if err == gorm.ErrRecordNotFound {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return time.Now().UTC().Sub(cd.LastUsed.UTC()) >= duration, nil
}

// ClearCooldown removes a cooldown record for the given activity.
func (s *Store) ClearCooldown(userID int64, activity string) error {
	return s.DB.Where("user_id = ? AND activity_name = ?", userID, activity).
		Delete(&model.Cooldown{}).Error
}

// SetCooldown records the current time as the last use of an activity.
func (s *Store) SetCooldown(userID int64, activity string) error {
	now := time.Now().UTC()
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "activity_name"}},
		DoUpdates: clause.Assignments(map[string]any{"last_used": now}),
	}).Create(&model.Cooldown{
		UserID: userID, ActivityName: activity, LastUsed: now,
	}).Error
}
