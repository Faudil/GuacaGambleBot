package model

import "time"

type VeilRaid struct {
	ID             int64      `gorm:"primaryKey;column:id;autoIncrement"`
	GuildID        int64      `gorm:"column:guild_id"`
	LeaderID       int64      `gorm:"column:leader_id"`
	Status         string     `gorm:"column:status;default:forming"`
	Phase          string     `gorm:"column:phase;default:whispering"`
	ParticipantIDs string     `gorm:"column:participant_ids;type:text"`
	PlayerStates   string     `gorm:"column:player_states;type:text"`
	Turn           int        `gorm:"column:turn;default:0"`
	TurnEndsAt     *time.Time `gorm:"column:turn_ends_at"`
	BossHP         int        `gorm:"column:boss_hp;default:1500"`
	BossMaxHP      int        `gorm:"column:boss_max_hp;default:1500"`
	BossPhase      int        `gorm:"column:boss_phase;default:1"`
	Mechanics      string     `gorm:"column:mechanics;type:text"`
	AddHP          string     `gorm:"column:add_hp;type:text"`
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

func (VeilRaid) TableName() string            { return "veil_raids" }
func (VeilRaidLockout) TableName() string     { return "veil_raid_lockouts" }
func (VeilRaidHallOfFame) TableName() string  { return "veil_raid_hall_of_fame" }
