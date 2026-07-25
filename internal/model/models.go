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

type Inventory struct {
	UserID   int64  `gorm:"primaryKey;column:user_id"`
	ItemID   string `gorm:"primaryKey;column:item_id"`
	Quantity int    `gorm:"column:quantity;default:0"`
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

type PetHistoryEntry struct {
	Time   time.Time `json:"time"`
	Event  string    `json:"event"`
	Detail string    `json:"detail"`
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

	BondLevel     int    `gorm:"column:bond_level;default:0"`
	History       string `gorm:"column:history;default:'[]'"`
	Title         string `gorm:"column:title;default:''"`
	SkillPoints   int    `gorm:"column:skill_points;default:0"`
	Personality   string `gorm:"column:personality;default:brave"`
	InSanctuary   bool   `gorm:"column:in_sanctuary;default:false"`
	ShowcaseSlot  int    `gorm:"column:showcase_slot;default:0"`
}

type UserPetSkill struct {
	PetID   int64  `gorm:"primaryKey;column:pet_id"`
	Slot    int    `gorm:"primaryKey;column:slot"`
	SkillID string `gorm:"column:skill_id"`
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

type UserCropHarvest struct {
	UserID   int64  `gorm:"primaryKey;column:user_id"`
	CropName string `gorm:"primaryKey;column:crop_name"`
	Count    int    `gorm:"column:count;default:0"`
}

type UserFossilHarvest struct {
	UserID   int64  `gorm:"primaryKey;column:user_id"`
	FossilID string `gorm:"primaryKey;column:fossil_id"`
	Count    int    `gorm:"column:count;default:0"`
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
	Universe              string     `gorm:"column:universe;default:hoakhaven"`
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

type UserFurniture struct {
	UserID      int64     `gorm:"primaryKey;column:user_id"`
	FurnitureID string    `gorm:"primaryKey;column:furniture_id"`
	PlacedAt    time.Time `gorm:"column:placed_at"`
}

type UserResearch struct {
	UserID     int64      `gorm:"primaryKey;column:user_id"`
	ResearchID string     `gorm:"primaryKey;column:research_id"`
	StartTime  *time.Time `gorm:"column:start_time"`
	FinishTime *time.Time `gorm:"column:finish_time"`
	Completed  bool       `gorm:"column:completed;default:false"`
}

type UserSanctuary struct {
	UserID            int64      `gorm:"primaryKey;column:user_id"`
	Tier              int        `gorm:"column:tier;default:0"`
	LastCollect       *time.Time `gorm:"column:last_collect"`
	UnderConstruction *string    `gorm:"column:under_construction"`
	FinishTime        *time.Time `gorm:"column:finish_time"`
}

func (UserSanctuary) TableName() string { return "user_sanctuaries" }

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
	ID         int64     `gorm:"primaryKey;column:id;autoIncrement"`
	UserID     int64     `gorm:"column:user_id"`
	ZoneKey    string    `gorm:"column:zone_key"`
	PlotIndex  int       `gorm:"column:plot_index"`
	ItemName   string    `gorm:"column:item_name"`
	PlantTime  time.Time `gorm:"column:plant_time"`
	GrowTime   int       `gorm:"column:grow_time"`
	Watered    bool      `gorm:"column:watered;default:false"`
	Mutated    bool      `gorm:"column:mutated;default:false"`
	Mysterious bool      `gorm:"column:mysterious;default:false"`
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
	UserID          int64     `gorm:"primaryKey;column:user_id"`
	NPCID           string    `gorm:"primaryKey;column:npc_id"`
	Reputation      int       `gorm:"default:0"`
	Level           int       `gorm:"default:1"`
	LastInteraction time.Time `gorm:"column:last_interaction"`
}

type UserNPCDailyRep struct {
	UserID  int64  `gorm:"primaryKey;column:user_id"`
	NPCID   string `gorm:"primaryKey;column:npc_id"`
	DateStr string `gorm:"primaryKey;column:date_str"`
	Amount  int    `gorm:"default:0"`
}

type UserNPCSecret struct {
	UserID   int64  `gorm:"primaryKey;column:user_id"`
	NPCID    string `gorm:"primaryKey;column:npc_id"`
	SecretID string `gorm:"primaryKey;column:secret_id"`
	Seen     bool   `gorm:"default:true"`
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

// UserCharacter holds the RPG character progression data.
type UserCharacter struct {
	UserID      int64 `gorm:"primaryKey;column:user_id"`
	Level       int   `gorm:"column:level;default:1"`
	XP          int   `gorm:"column:xp;default:0"`
	SkillPoints int   `gorm:"column:skill_points;default:0"`
	STR         int   `gorm:"column:str;default:5"`
	DEX         int   `gorm:"column:dex;default:5"`
	INT         int   `gorm:"column:int;default:5"`
	VIT         int   `gorm:"column:vit;default:5"`
	LUK         int   `gorm:"column:luk;default:5"`
}

// UserEquipment represents a unique gear instance with base stats + rolled affixes.
type UserEquipment struct {
	ID         uint      `gorm:"primaryKey;column:id;autoIncrement"`
	UserID     int64     `gorm:"index:idx_ue_user;column:user_id"`
	BaseID     string    `gorm:"column:base_id"`                           // items.Item.ID reference
	Name       string    `gorm:"column:name"`                              // display name (prefix + base + suffix)
	Emoji      string    `gorm:"column:emoji"`
	Rarity     string    `gorm:"column:rarity"`                            // common → legendary
	EquipSlot  string    `gorm:"column:equip_slot"`                        // weapon, armor, accessory, trinket
	StatSTR    int       `gorm:"column:stat_str;default:0"`                // base + affix total
	StatDEX    int       `gorm:"column:stat_dex;default:0"`
	StatINT    int       `gorm:"column:stat_int;default:0"`
	StatVIT    int       `gorm:"column:stat_vit;default:0"`
	StatLUK    int       `gorm:"column:stat_luk;default:0"`
	Affixes    string    `gorm:"column:affixes;type:text;default:'[]'"`    // JSON array of affix objects
	SetID      string    `gorm:"column:set_id"`                            // set key, empty if none
	IsEquipped bool      `gorm:"column:is_equipped;default:false"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

// ActiveBuff records an active skill buff that will be consumed on next use.
type ActiveBuff struct {
	UserID    int64     `gorm:"primaryKey;column:user_id"`
	SkillID   string    `gorm:"primaryKey;column:skill_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

// MarketState tracks the dynamic price and volume for a marketable item.
type MarketState struct {
	ItemID       string `gorm:"primaryKey;column:item_id"`
	CurrentPrice int    `gorm:"column:current_price"`
	DailySold    int    `gorm:"column:daily_sold;default:0"`
	DailyBought  int    `gorm:"column:daily_bought;default:0"`
	LastReset    string `gorm:"column:last_reset"`   // "2006-01-02"
	WeekID       string `gorm:"column:week_id"`      // "2026-W30"
	IsActive     bool   `gorm:"column:is_active;default:false"`
}

func (MarketState) TableName() string { return "market_state" }

// UserLoreEntry records a lore fragment discovered by a player.
type UserLoreEntry struct {
	UserID       int64     `gorm:"primaryKey;column:user_id"`
	LoreID       string    `gorm:"primaryKey;column:lore_id"`
	DiscoveredAt time.Time `gorm:"column:discovered_at;autoCreateTime"`
}

func (UserLoreEntry) TableName() string { return "user_lore" }

type WeeklyRank struct {
	UserID   int64 `gorm:"primaryKey;column:user_id"`
	ServerID int64 `gorm:"primaryKey;column:server_id"`
	WeekID   string `gorm:"primaryKey;column:week_id"`
	Score    int   `gorm:"column:score;default:0"`
	Wins     int   `gorm:"column:wins;default:0"`
	Losses   int   `gorm:"column:losses;default:0"`
}

type WeeklyModifier struct {
	ServerID  int64     `gorm:"primaryKey;column:server_id"`
	WeekID    string    `gorm:"primaryKey;column:week_id"`
	Modifier  string    `gorm:"column:modifier"`
	Boosted   string    `gorm:"column:boosted"`
	Nerfed    string    `gorm:"column:nerfed"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

type UserPetArtifact struct {
	UserID        int64  `gorm:"primaryKey;column:user_id"`
	Level         int    `gorm:"column:level;default:1"`
	XP            int    `gorm:"column:xp;default:0"`
	UnspentPoints int    `gorm:"column:unspent_points;default:0"`
	Stat1         string `gorm:"column:stat1"`
	Stat1Lvl      int    `gorm:"column:stat1_lvl;default:1"`
	Stat2         string `gorm:"column:stat2"`
	Stat2Lvl      int    `gorm:"column:stat2_lvl;default:1"`
	Stat3         string `gorm:"column:stat3"`
	Stat3Lvl      int    `gorm:"column:stat3_lvl;default:1"`
}

func (WeeklyRank) TableName() string         { return "weekly_ranks" }
func (WeeklyModifier) TableName() string      { return "weekly_modifiers" }
func (UserPetArtifact) TableName() string     { return "user_pet_artifacts" }

// DelveSession stores an active dungeon run for a player.
type DelveSession struct {
	UserID        int64      `gorm:"primaryKey;column:user_id"`
	GuildID       int64      `gorm:"column:guild_id"`
	ChannelID     int64      `gorm:"column:channel_id"`
	Floor         int        `gorm:"column:floor;default:1"`
	Zone          string     `gorm:"column:zone;default:crypt"`
	HP            int        `gorm:"column:hp"`
	MaxHP         int        `gorm:"column:max_hp"`
	Mana          int        `gorm:"column:mana"`
	MaxMana       int        `gorm:"column:max_mana"`
	Torches       int        `gorm:"column:torches;default:3"`
	Keys          int        `gorm:"column:keys;default:0"`
	Gold          int        `gorm:"column:gold;default:0"`
	Inventory     string     `gorm:"column:inventory;type:text"`      // JSON
	DeployedPets  string     `gorm:"column:deployed_pets;type:text"`  // JSON []int64
	Flags         string     `gorm:"column:flags;type:text"`          // JSON []string
	StatusEffects string     `gorm:"column:status_effects;type:text"` // JSON []string
	RoomsCleared  int        `gorm:"column:rooms_cleared;default:0"`
	Seed          int64      `gorm:"column:seed"`
	MessageID     string     `gorm:"column:message_id"`  // discord message ID for the embed
	Status        string     `gorm:"column:status;default:active"`
	AutoRescued   bool       `gorm:"column:auto_rescued;default:false"`
	AutoRescuePet string     `gorm:"column:auto_rescue_pet;default:''"`
	DiedAt        *time.Time `gorm:"column:died_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

// UserDelveFlag records a permanent story flag earned by a player.
type UserDelveFlag struct {
	ID       int64     `gorm:"primaryKey;column:id;autoIncrement"`
	UserID   int64     `gorm:"index:idx_delve_flags_user;column:user_id"`
	FlagID   string    `gorm:"column:flag_id"`
	Metadata string    `gorm:"column:metadata;type:text"` // JSON
	EarnedAt time.Time `gorm:"column:earned_at;autoCreateTime"`
}

// DelveRunHistory archives a completed dungeon run for the chronicle.
type DelveRunHistory struct {
	ID          int64     `gorm:"primaryKey;column:id;autoIncrement"`
	UserID      int64     `gorm:"index:idx_delve_history_user;column:user_id"`
	RunDate     time.Time `gorm:"column:run_date"`
	Floors      int       `gorm:"column:floors"`
	Outcome     string    `gorm:"column:outcome"`      // "escaped", "fell", "fled"
	FlagsEarned string    `gorm:"column:flags_earned;type:text"` // JSON
	LootSummary string    `gorm:"column:loot_summary;type:text"` // JSON
}

// DelveGauntletScore records a player's weekly gauntlet performance.
type DelveGauntletScore struct {
	ID        int64     `gorm:"primaryKey;column:id;autoIncrement"`
	UserID    int64     `gorm:"index:idx_gauntlet_user;column:user_id"`
	GuildID   int64     `gorm:"index:idx_gauntlet_guild;column:guild_id"`
	WeekStart string    `gorm:"column:week_start"`
	Floor     int       `gorm:"column:floor"`
	Score     int       `gorm:"column:score"`
	Seed      string    `gorm:"column:seed"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (DelveSession) TableName() string         { return "delve_sessions" }
func (UserDelveFlag) TableName() string          { return "user_delve_flags" }
func (DelveRunHistory) TableName() string        { return "delve_run_history" }
func (DelveGauntletScore) TableName() string     { return "delve_gauntlet_scores" }

// VeilRaid stores an active raid group and its state.
type VeilRaid struct {
	ID             int64      `gorm:"primaryKey;column:id;autoIncrement"`
	GuildID        int64      `gorm:"column:guild_id"`
	LeaderID       int64      `gorm:"column:leader_id"`
	Status         string     `gorm:"column:status;default:forming"`
	Phase          string     `gorm:"column:phase;default:whispering"`
	ParticipantIDs string     `gorm:"column:participant_ids;type:text"` // JSON []int64
	PlayerStates   string     `gorm:"column:player_states;type:text"`   // JSON map[int64]PlayerState
	Turn           int        `gorm:"column:turn;default:0"`
	TurnEndsAt     *time.Time `gorm:"column:turn_ends_at"`
	BossHP         int        `gorm:"column:boss_hp;default:1500"`
	BossMaxHP      int        `gorm:"column:boss_max_hp;default:1500"`
	BossPhase      int        `gorm:"column:boss_phase;default:1"`
	Mechanics      string     `gorm:"column:mechanics;type:text"`     // JSON
	AddHP          string     `gorm:"column:add_hp;type:text"`        // JSON
	CrystalUsed    bool       `gorm:"column:crystal_used;default:false"`
	MessageID      string     `gorm:"column:message_id"`
	ThreadID       string     `gorm:"column:thread_id"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

type VeilPlayerState struct {
	UserID    int64    `json:"user_id"`
	HP        int      `json:"hp"`
	MaxHP     int      `json:"max_hp"`
	Action    string   `json:"action"`
	ActionVal int      `json:"action_val"`
	TargetID  int64    `json:"target_id"`
	Status    string   `json:"status"`
	Chronicle []string `json:"chronicle"`
}

type VeilRaidLockout struct {
	UserID    int64      `gorm:"primaryKey;column:user_id"`
	WeekStart string     `gorm:"primaryKey;column:week_start"`
	Cleared   bool       `gorm:"column:cleared;default:false"`
	HelpedAt  *time.Time `gorm:"column:helped_at"`
}

type VeilRaidHallOfFame struct {
	ID        int64     `gorm:"primaryKey;column:id;autoIncrement"`
	GuildID   int64     `gorm:"column:guild_id"`
	WeekStart string    `gorm:"column:week_start"`
	ClearTime int       `gorm:"column:clear_time"`
	Group     string    `gorm:"column:group_data;type:text"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (VeilRaid) TableName() string             { return "veil_raids" }
func (VeilRaidLockout) TableName() string        { return "veil_raid_lockouts" }
func (VeilRaidHallOfFame) TableName() string     { return "veil_raid_hall_of_fame" }

func (User) TableName() string                      { return "users" }
func (Cooldown) TableName() string                  { return "cooldowns" }
func (GameLimit) TableName() string                 { return "game_limits" }
func (Bet) TableName() string                       { return "bets" }
func (Wager) TableName() string                     { return "wagers" }
func (Inventory) TableName() string                 { return "inventory" }
func (LottoState) TableName() string                { return "lotto_state" }
func (Job) TableName() string                       { return "jobs" }
func (UserPetSkill) TableName() string              { return "user_pet_skills" }
func (UserStat) TableName() string                  { return "user_stats" }
func (UserCropHarvest) TableName() string           { return "user_crop_harvests" }
func (UserFossilHarvest) TableName() string          { return "user_fossil_harvests" }
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
func (UserCharacter) TableName() string             { return "user_characters" }
func (UserEquipment) TableName() string             { return "user_equipment" }
func (ActiveBuff) TableName() string                 { return "active_buffs" }
// -- Criminality System Models --

// UserCriminality stores the criminal career of a player.
type UserCriminality struct {
	UserID       int64      `gorm:"primaryKey;column:user_id"`
	Alignment    string     `gorm:"column:alignment;default:neutral"` // neutral, thief, hunter
	ThiefRank    int        `gorm:"column:thief_rank;default:0"`
	ThiefInfamy  int        `gorm:"column:thief_infamy;default:0"`
	HunterRank   int        `gorm:"column:hunter_rank;default:0"`
	HunterMerit  int        `gorm:"column:hunter_merit;default:0"`
	Notoriety    int        `gorm:"column:notoriety;default:0"`
	DisguiseUsed bool       `gorm:"column:disguise_used;default:false"`
	PrisonUntil  *time.Time `gorm:"column:prison_until"`
	RedeemedAt   *time.Time `gorm:"column:redeemed_at"`
	PacifistUntil *time.Time `gorm:"column:pacifist_until"`
	HasMask      bool       `gorm:"column:has_mask;default:false"`
}

// WorldCriminalityState stores the per-server awakening state.
type WorldCriminalityState struct {
	ServerID     int64      `gorm:"primaryKey;column:server_id"`
	Awakened     bool       `gorm:"column:awakened;default:false"`
	FirstThiefID *int64     `gorm:"column:first_thief_id"`
	FirstVictimID *int64    `gorm:"column:first_victim_id"`
	AwakenedAt   *time.Time `gorm:"column:awakened_at"`
	MaskClaimedBy *int64    `gorm:"column:mask_claimed_by"`
	MaskClaimedAt *time.Time `gorm:"column:mask_claimed_at"`
}

// Bounty placed on a criminal player.
type Bounty struct {
	ID          int64      `gorm:"primaryKey;column:id;autoIncrement"`
	TargetID    int64      `gorm:"index:idx_bounty_target;column:target_id"`
	PlacerID    int64      `gorm:"column:placer_id"`
	Amount      int        `gorm:"column:amount"`
	PlacedAt    time.Time  `gorm:"column:placed_at"`
	ClaimedBy   *int64     `gorm:"column:claimed_by"`
	ClaimedAt   *time.Time `gorm:"column:claimed_at"`
	IsAnonymous bool       `gorm:"column:is_anonymous;default:false"`
}

// TheftRecord logs every attempted theft.
type TheftRecord struct {
	ID         int64     `gorm:"primaryKey;column:id;autoIncrement"`
	ThiefID    int64     `gorm:"index:idx_theft_thief;column:thief_id"`
	VictimID   int64     `gorm:"index:idx_theft_victim;column:victim_id"`
	StolenGold int       `gorm:"column:stolen_gold;default:0"`
	StolenItem string    `gorm:"column:stolen_item;default:''"`
	WasBurgle  bool      `gorm:"column:was_burgle;default:false"`
	Success    bool      `gorm:"column:success;default:false"`
	Forgiven   bool      `gorm:"column:forgiven;default:false"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

// CrimeRecord stores chronicle entries for the criminality system.
type CrimeRecord struct {
	ID        int64     `gorm:"primaryKey;column:id;autoIncrement"`
	UserID    int64     `gorm:"index:idx_crime_user;column:user_id"`
	Event     string    `gorm:"column:event"`
	Detail    string    `gorm:"column:detail;type:text"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

// HunterContract represents a PvE bounty contract on the board.
type HunterContract struct {
	ID          int64  `gorm:"primaryKey;column:id;autoIncrement"`
	ContractID  string `gorm:"uniqueIndex;column:contract_id"`
	TargetName  string `gorm:"column:target_name"`
	TargetLevel int    `gorm:"column:target_level"`
	Zone        string `gorm:"column:zone"`
	RewardMerit int    `gorm:"column:reward_merit"`
	RewardGold  int    `gorm:"column:reward_gold"`
	IsLegendary bool   `gorm:"column:is_legendary;default:false"`
	Available   bool   `gorm:"column:available;default:true"`
	ClaimedBy   *int64 `gorm:"column:claimed_by"`
}

func (UserCriminality) TableName() string      { return "user_criminality" }
func (WorldCriminalityState) TableName() string { return "world_criminality_state" }
func (Bounty) TableName() string               { return "bounties" }
func (TheftRecord) TableName() string          { return "theft_records" }
func (CrimeRecord) TableName() string          { return "crime_records" }
func (HunterContract) TableName() string       { return "hunter_contracts" }

func (UserFurniture) TableName() string              { return "user_furniture" }
func (UserResearch) TableName() string               { return "user_research" }
