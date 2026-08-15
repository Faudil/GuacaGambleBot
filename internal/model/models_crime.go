package model

import "time"

type UserCriminality struct {
	UserID        int64      `gorm:"primaryKey;column:user_id"`
	Alignment     string     `gorm:"column:alignment;default:neutral"`
	ThiefRank     int        `gorm:"column:thief_rank;default:0"`
	ThiefInfamy   int        `gorm:"column:thief_infamy;default:0"`
	HunterRank    int        `gorm:"column:hunter_rank;default:0"`
	HunterMerit   int        `gorm:"column:hunter_merit;default:0"`
	Notoriety     int        `gorm:"column:notoriety;default:0"`
	DisguiseUsed  bool       `gorm:"column:disguise_used;default:false"`
	PrisonUntil   *time.Time `gorm:"column:prison_until"`
	RedeemedAt    *time.Time `gorm:"column:redeemed_at"`
	PacifistUntil *time.Time `gorm:"column:pacifist_until"`
	HasMask       bool       `gorm:"column:has_mask;default:false"`
}

type WorldCriminalityState struct {
	ServerID      int64      `gorm:"primaryKey;column:server_id"`
	Awakened      bool       `gorm:"column:awakened;default:false"`
	FirstThiefID  *int64     `gorm:"column:first_thief_id"`
	FirstVictimID *int64     `gorm:"column:first_victim_id"`
	AwakenedAt    *time.Time `gorm:"column:awakened_at"`
	MaskClaimedBy *int64     `gorm:"column:mask_claimed_by"`
	MaskClaimedAt *time.Time `gorm:"column:mask_claimed_at"`
}

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

type CrimeRecord struct {
	ID        int64     `gorm:"primaryKey;column:id;autoIncrement"`
	UserID    int64     `gorm:"index:idx_crime_user;column:user_id"`
	Event     string    `gorm:"column:event"`
	Detail    string    `gorm:"column:detail;type:text"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

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

func (UserCriminality) TableName() string       { return "user_criminality" }
func (WorldCriminalityState) TableName() string { return "world_criminality_state" }
func (Bounty) TableName() string                { return "bounties" }
func (TheftRecord) TableName() string           { return "theft_records" }
func (CrimeRecord) TableName() string           { return "crime_records" }
func (HunterContract) TableName() string        { return "hunter_contracts" }
