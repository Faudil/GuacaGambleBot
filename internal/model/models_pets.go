package model

import "time"

type PetHistoryEntry struct {
	Time   time.Time `json:"time"`
	Event  string    `json:"event"`
	Detail string    `json:"detail"`
}

type UserPet struct {
	ID           int64   `gorm:"primaryKey;column:id;autoIncrement"`
	UserID       int64   `gorm:"column:user_id"`
	ServerID     int64   `gorm:"column:server_id;default:0"`
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
	BondLevel    int     `gorm:"column:bond_level;default:0"`
	History      string  `gorm:"column:history;default:'[]'"`
	Title        string  `gorm:"column:title;default:''"`
	SkillPoints  int     `gorm:"column:skill_points;default:0"`
	Personality  string  `gorm:"column:personality;default:brave"`
	InSanctuary  bool    `gorm:"column:in_sanctuary;default:false"`
	ShowcaseSlot int     `gorm:"column:showcase_slot;default:0"`
}

type UserPetSkill struct {
	PetID   int64  `gorm:"primaryKey;column:pet_id"`
	Slot    int    `gorm:"primaryKey;column:slot"`
	SkillID string `gorm:"column:skill_id"`
}

type ServerPetElo struct {
	PetID    int64 `gorm:"primaryKey;column:pet_id"`
	ServerID int64 `gorm:"primaryKey;column:server_id"`
	Elo      int   `gorm:"column:elo;default:1000"`
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

func (UserPetSkill) TableName() string    { return "user_pet_skills" }
func (ServerPetElo) TableName() string    { return "server_pet_elo" }
func (PetExpedition) TableName() string   { return "pet_expeditions" }
func (UserPetArtifact) TableName() string { return "user_pet_artifacts" }
