package model

import "time"

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
	Potions       int        `gorm:"column:potions;default:1"`
	Gold          int        `gorm:"column:gold;default:0"`
	Inventory     string     `gorm:"column:inventory;type:text"`
	DeployedPets  string     `gorm:"column:deployed_pets;type:text"`
	Flags         string     `gorm:"column:flags;type:text"`
	StatusEffects string     `gorm:"column:status_effects;type:text"`
	RoomsCleared  int        `gorm:"column:rooms_cleared;default:0"`
	Seed          int64      `gorm:"column:seed"`
	MessageID     string     `gorm:"column:message_id"`
	Status        string     `gorm:"column:status;default:active"`
	AutoRescued   bool       `gorm:"column:auto_rescued;default:false"`
	AutoRescuePet string     `gorm:"column:auto_rescue_pet;default:''"`
	DiedAt        *time.Time `gorm:"column:died_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

type UserDelveFlag struct {
	ID       int64     `gorm:"primaryKey;column:id;autoIncrement"`
	UserID   int64     `gorm:"index:idx_delve_flags_user;column:user_id"`
	FlagID   string    `gorm:"column:flag_id"`
	Metadata string    `gorm:"column:metadata;type:text"`
	EarnedAt time.Time `gorm:"column:earned_at;autoCreateTime"`
}

type DelveRunHistory struct {
	ID          int64     `gorm:"primaryKey;column:id;autoIncrement"`
	UserID      int64     `gorm:"index:idx_delve_history_user;column:user_id"`
	RunDate     time.Time `gorm:"column:run_date"`
	Floors      int       `gorm:"column:floors"`
	Outcome     string    `gorm:"column:outcome"`
	FlagsEarned string    `gorm:"column:flags_earned;type:text"`
	LootSummary string    `gorm:"column:loot_summary;type:text"`
}

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

func (DelveSession) TableName() string        { return "delve_sessions" }
func (UserDelveFlag) TableName() string        { return "user_delve_flags" }
func (DelveRunHistory) TableName() string      { return "delve_run_history" }
func (DelveGauntletScore) TableName() string   { return "delve_gauntlet_scores" }
