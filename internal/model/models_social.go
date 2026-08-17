package model

import "time"

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
	ServerID      int64  `gorm:"primaryKey;column:server_id"`
	WinningNumber int    `gorm:"column:winning_number"`
	Jackpot       int    `gorm:"column:jackpot"`
	LastBonusDate string `gorm:"column:last_bonus_date;default:"`
}

type UserHousing struct {
	UserID            int64      `gorm:"primaryKey;column:user_id"`
	HouseType         string     `gorm:"primaryKey;column:house_type"`
	Level             int        `gorm:"column:level;default:1"`
	LastCollected     *time.Time `gorm:"column:last_collected"`
	IsActive          bool       `gorm:"column:is_active;default:false"`
	CustomName        *string    `gorm:"column:custom_name"`
	CustomColor       *string    `gorm:"column:custom_color"`
	UnderConstruction *string    `gorm:"column:under_construction"`
	FinishTime        *time.Time `gorm:"column:finish_time"`
	StoredItems       string     `gorm:"column:stored_items;default:'{}'"`
}

type UserHousingUpgrade struct {
	UserID    int64  `gorm:"primaryKey;column:user_id"`
	UpgradeID string `gorm:"primaryKey;column:upgrade_id"`
}

type UserFurniture struct {
	UserID      int64     `gorm:"primaryKey;column:user_id"`
	HouseType   string    `gorm:"primaryKey;column:house_type"`
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
	Chats   int    `gorm:"default:0"`
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
	ServerID          int64  `gorm:"primaryKey;column:server_id"`
	ProjectID         string `gorm:"primaryKey;column:project_id"`
	ResourceType      string `gorm:"primaryKey;column:resource_type"`
	AmountContributed int    `gorm:"default:0"`
}

type UserCommunityStat struct {
	UserID             int64 `gorm:"primaryKey;column:user_id"`
	ServerID           int64 `gorm:"primaryKey;column:server_id"`
	TotalMoneyInvested int   `gorm:"default:0"`
	TotalItemsInvested int   `gorm:"default:0"`
}

func (ServerSetting) TableName() string             { return "server_settings" }
func (ServerLottoState) TableName() string          { return "server_lotto_state" }
func (UserHousing) TableName() string               { return "user_housing" }
func (UserHousingUpgrade) TableName() string        { return "user_housing_upgrades" }
func (UserFurniture) TableName() string             { return "user_furniture" }
func (UserResearch) TableName() string              { return "user_research" }
func (UserSanctuary) TableName() string             { return "user_sanctuaries" }
func (UserNPCReputation) TableName() string         { return "user_npc_reputation" }
func (UserNPCDailyRep) TableName() string           { return "user_npc_daily_rep" }
func (ServerProject) TableName() string             { return "server_projects" }
func (ServerProjectContribution) TableName() string { return "server_project_contributions" }
func (UserCommunityStat) TableName() string         { return "user_community_stats" }
