package model

import "time"

// User is the core player account.
type User struct {
	UserID          int64 `gorm:"primaryKey;column:user_id"`
	Balance         int   `gorm:"default:100"`
	Bank            int   `gorm:"default:0"`
	Crowns          int   `gorm:"default:0"`
	ExtraInvSlots   int   `gorm:"default:0"`
	ExtraPetSlots   int   `gorm:"default:0"`
	BossLeagueStage int   `gorm:"default:0"`
}

// Cooldown tracks per-activity cooldown timestamps.
type Cooldown struct {
	UserID       int64     `gorm:"primaryKey;column:user_id"`
	ActivityName string    `gorm:"primaryKey;column:activity_name"`
	LastUsed     time.Time `gorm:"column:last_used"`
}

// GameLimit enforces a daily usage cap per (user, game).
type GameLimit struct {
	UserID   int64  `gorm:"primaryKey;column:user_id"`
	GameName string `gorm:"primaryKey;column:game_name"`
	DateStr  string `gorm:"primaryKey;column:date_str"`
	Count    int    `gorm:"default:0"`
}

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

type Item struct {
	ID          int64  `gorm:"primaryKey;column:id;autoIncrement"`
	Name        string `gorm:"column:name;uniqueIndex"`
	Price       int    `gorm:"column:price"`
	Description string `gorm:"column:description"`
	EffectType  string `gorm:"column:effect_type"`
}

type Inventory struct {
	UserID   int64 `gorm:"primaryKey;column:user_id"`
	ItemID   int64 `gorm:"primaryKey;column:item_id"`
	Quantity int   `gorm:"column:quantity;default:1"`
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

type UserPet struct {
	ID           int64   `gorm:"primaryKey;column:id;autoIncrement"`
	UserID       int64   `gorm:"column:user_id"`
	PetType      string  `gorm:"column:pet_type"`
	Nickname     string  `gorm:"column:nickname"`
	Level        int     `gorm:"column:level;default:1"`
	FoodEaten    int     `gorm:"column:food_eaten;default:0"`
	Bonus        int     `gorm:"column:bonus;default:0"`
	XP           int     `gorm:"column:xp;default:0"`
	MaxHP        int     `gorm:"column:max_hp;default:50"`
	HP           int     `gorm:"column:hp;default:50"`
	Atk          int     `gorm:"column:atk;default:10"`
	Defense      int     `gorm:"column:defense;default:5"`
	Speed        int     `gorm:"column:speed;default:10"`
	DGE          int     `gorm:"column:dge;default:5"`
	ACC          int     `gorm:"column:acc;default:0"`
	CritC        int     `gorm:"column:crit_c;default:5"`
	CritD        float64 `gorm:"column:crit_d;default:1.5"`
	SpcC         int     `gorm:"column:spc_c;default:0"`
	TrsLvl       int     `gorm:"column:trs_lvl;default:0"`
	Elo          int     `gorm:"column:elo;default:1000"`
	IsActive     bool    `gorm:"column:is_active;default:false"`
	OnExpedition bool    `gorm:"column:on_expedition;default:false"`
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
}

type UserAchievement struct {
	UserID        int64  `gorm:"primaryKey;column:user_id"`
	AchievementID string `gorm:"primaryKey;column:achievement_id"`
}

type ServerPetElo struct {
	PetID   int64 `gorm:"primaryKey;column:pet_id"`
	ServerID int64 `gorm:"primaryKey;column:server_id"`
	Elo     int   `gorm:"column:elo;default:1000"`
}

type ServerSetting struct {
	ServerID              int64      `gorm:"primaryKey;column:server_id"`
	AnnouncementChannelID int64      `gorm:"column:announcement_channel_id"`
	ChannelID             int64      `gorm:"column:channel_id"`
	Language              string     `gorm:"column:language;default:fr"`
	Prefix                string     `gorm:"column:prefix"`
	Enabled               bool       `gorm:"column:enabled"`
	OnboardedAt           *time.Time `gorm:"column:onboarded_at"`
}

type ServerLottoState struct {
	ServerID      int64 `gorm:"primaryKey;column:server_id"`
	WinningNumber int   `gorm:"column:winning_number"`
	Jackpot       int   `gorm:"column:jackpot"`
	LastBonusDate string `gorm:"column:last_bonus_date;default:"`
}

type UserHousing struct {
	UserID            int64     `gorm:"primaryKey;column:user_id"`
	HouseType         string    `gorm:"column:house_type"`
	Level             int       `gorm:"column:level;default:1"`
	LastCollected     *time.Time `gorm:"column:last_collected"`
	CustomName        *string   `gorm:"column:custom_name"`
	CustomColor       *string   `gorm:"column:custom_color"`
	UnderConstruction *string   `gorm:"column:under_construction"`
	FinishTime        *time.Time `gorm:"column:finish_time"`
	StoredItems       string    `gorm:"column:stored_items;default:'{}'"`
}

type UserHousingUpgrade struct {
	UserID    int64  `gorm:"primaryKey;column:user_id"`
	UpgradeID string `gorm:"primaryKey;column:upgrade_id"`
}

type PetExpedition struct {
	ID          int64     `gorm:"primaryKey;column:id;autoIncrement"`
	UserID      int64     `gorm:"column:user_id"`
	PetID       int64     `gorm:"column:pet_id"`
	StartTime   time.Time `gorm:"column:start_time"`
	EndTime     time.Time `gorm:"column:end_time"`
	RewardXP    int       `gorm:"column:reward_xp;default:0"`
	RewardItems string    `gorm:"column:reward_items"`
	Log         string    `gorm:"column:log"`
	IsClaimed   bool      `gorm:"column:is_claimed;default:false"`
}

type UserFarming struct {
	ID        int64     `gorm:"primaryKey;column:id;autoIncrement"`
	UserID    int64     `gorm:"column:user_id"`
	ZoneKey   string    `gorm:"column:zone_key"`
	PlotIndex int       `gorm:"column:plot_index"`
	ItemName  string    `gorm:"column:item_name"`
	PlantTime time.Time `gorm:"column:plant_time"`
	GrowTime  int       `gorm:"column:grow_time"`
}

type UserQuest struct {
	UserID      int64      `gorm:"primaryKey;column:user_id"`
	QuestID     string     `gorm:"primaryKey;column:quest_id"`
	Status      string     `gorm:"column:status;default:ACTIVE"`
	StartedAt   time.Time  `gorm:"column:started_at"`
	CompletedAt *time.Time `gorm:"column:completed_at"`
}

type UserQuestData struct {
	UserID        int64  `gorm:"primaryKey;column:user_id"`
	QuestID       string `gorm:"primaryKey;column:quest_id"`
	StepIndex     int    `gorm:"column:step_index;default:0"`
	ProgressValue int    `gorm:"column:progress_value;default:0"`
	CustomData    string `gorm:"column:custom_data;default:'{}'"`
}

type UserNPCReputation struct {
	UserID     int64  `gorm:"primaryKey;column:user_id"`
	NPCID      string `gorm:"primaryKey;column:npc_id"`
	Reputation int    `gorm:"default:0"`
	Level      int    `gorm:"default:1"`
}

type UserNPCDailyRep struct {
	UserID  int64  `gorm:"primaryKey;column:user_id"`
	NPCID   string `gorm:"primaryKey;column:npc_id"`
	DateStr string `gorm:"primaryKey;column:date_str"`
	Amount  int    `gorm:"default:0"`
}

type ServerProject struct {
	ServerID  int64  `gorm:"primaryKey;column:server_id"`
	ProjectID string `gorm:"primaryKey;column:project_id"`
	Level     int    `gorm:"default:1"`
}

type ServerProjectContribution struct {
	ServerID         int64  `gorm:"primaryKey;column:server_id"`
	ProjectID        string `gorm:"primaryKey;column:project_id"`
	ResourceType     string `gorm:"primaryKey;column:resource_type"`
	AmountContributed int   `gorm:"default:0"`
}

type UserCommunityStat struct {
	UserID             int64 `gorm:"primaryKey;column:user_id"`
	ServerID           int64 `gorm:"primaryKey;column:server_id"`
	TotalMoneyInvested int   `gorm:"default:0"`
	TotalItemsInvested int   `gorm:"default:0"`
}

type Loan struct {
	ID         int64  `gorm:"primaryKey;column:id;autoIncrement"`
	BorrowerID int64  `gorm:"column:borrower_id"`
	LenderID   int64  `gorm:"column:lender_id"`
	AmountDue  int    `gorm:"column:amount_due"`
	CreatedAt  string `gorm:"column:created_at"`
}

func (User) TableName() string                      { return "users" }
func (Cooldown) TableName() string                  { return "cooldowns" }
func (GameLimit) TableName() string                 { return "game_limits" }
func (Bet) TableName() string                       { return "bets" }
func (Wager) TableName() string                     { return "wagers" }
func (Item) TableName() string                      { return "items" }
func (Inventory) TableName() string                 { return "inventory" }
func (LottoState) TableName() string                { return "lotto_state" }
func (Job) TableName() string                       { return "jobs" }
func (UserPet) TableName() string                   { return "user_pets" }
func (UserStat) TableName() string                  { return "user_stats" }
func (UserAchievement) TableName() string           { return "user_achievements" }
func (ServerPetElo) TableName() string              { return "server_pet_elo" }
func (ServerSetting) TableName() string             { return "server_settings" }
func (ServerLottoState) TableName() string          { return "server_lotto_state" }
func (UserHousing) TableName() string               { return "user_housing" }
func (UserHousingUpgrade) TableName() string        { return "user_housing_upgrades" }
func (PetExpedition) TableName() string             { return "pet_expeditions" }
func (UserFarming) TableName() string               { return "user_farming" }
func (UserQuest) TableName() string                 { return "user_quests" }
func (UserQuestData) TableName() string             { return "user_quest_data" }
func (UserNPCReputation) TableName() string         { return "user_npc_reputation" }
func (UserNPCDailyRep) TableName() string           { return "user_npc_daily_rep" }
func (ServerProject) TableName() string             { return "server_projects" }
func (ServerProjectContribution) TableName() string { return "server_project_contributions" }
func (UserCommunityStat) TableName() string         { return "user_community_stats" }
func (Loan) TableName() string                      { return "loans" }
