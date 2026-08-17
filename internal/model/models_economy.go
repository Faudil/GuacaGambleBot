package model

type Bet struct {
	ID          int64  `gorm:"primaryKey;column:id;autoIncrement"`
	CreatorID   int64  `gorm:"column:creator_id"`
	Description string `gorm:"column:description"`
	Option1     string `gorm:"column:option1"`
	Option2     string `gorm:"column:option2"`
	Status      string `gorm:"column:status;default:OPEN"`
	Winner      string `gorm:"column:winner"`
}

type Wager struct {
	ID     int64  `gorm:"primaryKey;column:id;autoIncrement"`
	BetID  int64  `gorm:"column:bet_id"`
	UserID int64  `gorm:"column:user_id"`
	Option string `gorm:"column:option"`
	Amount int    `gorm:"column:amount"`
}

type Inventory struct {
	UserID   int64  `gorm:"primaryKey;column:user_id"`
	ItemID   string `gorm:"primaryKey;column:item_id"`
	Quantity int    `gorm:"column:quantity;default:0"`
	// Durability is the remaining uses of the active tool in a stack. Zero on a
	// legacy row is treated as a full tool and lazily initialized on first use.
	Durability int `gorm:"column:durability;default:0"`
}

type LottoState struct {
	ID            int64  `gorm:"primaryKey;column:id"`
	WinningNumber int    `gorm:"column:winning_number"`
	Jackpot       int    `gorm:"column:jackpot"`
	LastBonusDate string `gorm:"column:last_bonus_date;default:"`
}

type Job struct {
	UserID  int64  `gorm:"primaryKey;column:user_id"`
	JobName string `gorm:"primaryKey;column:job_name"`
	Level   int    `gorm:"column:level;default:1"`
	XP      int    `gorm:"column:xp;default:0"`
}

type Loan struct {
	ID         int64  `gorm:"primaryKey;column:id;autoIncrement"`
	BorrowerID int64  `gorm:"column:borrower_id"`
	LenderID   int64  `gorm:"column:lender_id"`
	AmountDue  int    `gorm:"column:amount_due"`
	CreatedAt  string `gorm:"column:created_at"`
}

type MarketState struct {
	ItemID       string `gorm:"primaryKey;column:item_id"`
	CurrentPrice int    `gorm:"column:current_price"`
	DailySold    int    `gorm:"column:daily_sold;default:0"`
	DailyBought  int    `gorm:"column:daily_bought;default:0"`
	LastReset    string `gorm:"column:last_reset"`
	WeekID       string `gorm:"column:week_id"`
	IsActive     bool   `gorm:"column:is_active;default:false"`
}

func (Bet) TableName() string         { return "bets" }
func (Wager) TableName() string       { return "wagers" }
func (Inventory) TableName() string   { return "inventory" }
func (LottoState) TableName() string  { return "lotto_state" }
func (Job) TableName() string         { return "jobs" }
func (Loan) TableName() string        { return "loans" }
func (MarketState) TableName() string { return "market_state" }
