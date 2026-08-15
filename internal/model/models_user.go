package model

import "time"

type User struct {
	UserID          int64 `gorm:"primaryKey;column:user_id"`
	Balance         int   `gorm:"default:100"`
	Bank            int   `gorm:"default:0"`
	Crowns          int   `gorm:"default:0"`
	ExtraInvSlots   int   `gorm:"default:0"`
	ExtraPetSlots   int   `gorm:"default:0"`
	BossLeagueStage int   `gorm:"default:0"`
}

type Cooldown struct {
	UserID       int64     `gorm:"primaryKey;column:user_id"`
	ActivityName string    `gorm:"primaryKey;column:activity_name"`
	LastUsed     time.Time `gorm:"column:last_used"`
}

type GameLimit struct {
	UserID   int64  `gorm:"primaryKey;column:user_id"`
	GameName string `gorm:"primaryKey;column:game_name"`
	DateStr  string `gorm:"primaryKey;column:date_str"`
	Count    int    `gorm:"default:0"`
}

type UserStat struct {
	UserID              int64 `gorm:"primaryKey;column:user_id"`
	PvpWins             int   `gorm:"column:pvp_wins;default:0"`
	PvpLosses           int   `gorm:"column:pvp_losses;default:0"`
	PveWins             int   `gorm:"column:pve_wins;default:0"`
	ItemsMined          int   `gorm:"column:items_mined;default:0"`
	ItemsFished         int   `gorm:"column:items_fished;default:0"`
	ItemsFarmed         int   `gorm:"column:items_farmed;default:0"`
	MoneyEarned         int   `gorm:"column:money_earned;default:0"`
	PetsFed             int   `gorm:"column:pets_fed;default:0"`
	CoinflipLost        int   `gorm:"column:coinflip_lost;default:0"`
	CoinflipWon         int   `gorm:"column:coinflip_won;default:0"`
	CasinoLost          int   `gorm:"column:casino_lost;default:0"`
	CasinoWon           int   `gorm:"column:casino_won;default:0"`
	SlotsWon            int   `gorm:"column:slots_won;default:0"`
	SlotsLost           int   `gorm:"column:slots_lost;default:0"`
	BlackjackWon        int   `gorm:"column:blackjack_won;default:0"`
	BlackjackLost       int   `gorm:"column:blackjack_lost;default:0"`
	RouletteWon         int   `gorm:"column:roulette_won;default:0"`
	RouletteLost        int   `gorm:"column:roulette_lost;default:0"`
	LottoParticipations int   `gorm:"column:lotto_participations;default:0"`
	LottoWon            int   `gorm:"column:lotto_won;default:0"`
	BetsWon             int   `gorm:"column:bets_won;default:0"`
	BetsLost            int   `gorm:"column:bets_lost;default:0"`
	WagersWon           int   `gorm:"column:wagers_won;default:0"`
	WagersLost          int   `gorm:"column:wagers_lost;default:0"`
	CasinoSpent         int   `gorm:"column:casino_spent;default:0"`
	SlotsSpent          int   `gorm:"column:slots_spent;default:0"`
	SlotsMoneyWon       int   `gorm:"column:slots_money_won;default:0"`
	SlotsMoneyLost      int   `gorm:"column:slots_money_lost;default:0"`
	CoinflipSpent       int   `gorm:"column:coinflip_spent;default:0"`
	CoinflipMoneyWon    int   `gorm:"column:coinflip_money_won;default:0"`
	CoinflipMoneyLost   int   `gorm:"column:coinflip_money_lost;default:0"`
	BlackjackSpent      int   `gorm:"column:blackjack_spent;default:0"`
	BlackjackMoneyWon   int   `gorm:"column:blackjack_money_won;default:0"`
	BlackjackMoneyLost  int   `gorm:"column:blackjack_money_lost;default:0"`
	RouletteSpent       int   `gorm:"column:roulette_spent;default:0"`
	RouletteMoneyWon    int   `gorm:"column:roulette_money_won;default:0"`
	RouletteMoneyLost   int   `gorm:"column:roulette_money_lost;default:0"`
	DailyUses           int   `gorm:"column:daily_uses;default:0"`
	ItemsSoldMarket     int   `gorm:"column:items_sold_market;default:0"`
	ItemsBoughtMarket   int   `gorm:"column:items_bought_market;default:0"`
	ToolDynamiteUses    int   `gorm:"column:tool_dynamite_uses;default:0"`
	ToolHammerUses      int   `gorm:"column:tool_hammer_uses;default:0"`
	ToolBrushUses       int   `gorm:"column:tool_brush_uses;default:0"`
}

func (User) TableName() string      { return "users" }
func (Cooldown) TableName() string  { return "cooldowns" }
func (GameLimit) TableName() string { return "game_limits" }
func (UserStat) TableName() string  { return "user_stats" }
