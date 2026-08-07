package model

import "time"

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

type UserEquipment struct {
	ID         uint      `gorm:"primaryKey;column:id;autoIncrement"`
	UserID     int64     `gorm:"index:idx_ue_user;column:user_id"`
	BaseID     string    `gorm:"column:base_id"`
	Name       string    `gorm:"column:name"`
	Emoji      string    `gorm:"column:emoji"`
	Rarity     string    `gorm:"column:rarity"`
	EquipSlot  string    `gorm:"column:equip_slot"`
	StatSTR    int       `gorm:"column:stat_str;default:0"`
	StatDEX    int       `gorm:"column:stat_dex;default:0"`
	StatINT    int       `gorm:"column:stat_int;default:0"`
	StatVIT    int       `gorm:"column:stat_vit;default:0"`
	StatLUK    int       `gorm:"column:stat_luk;default:0"`
	Affixes    string    `gorm:"column:affixes;type:text;default:'[]'"`
	SetID      string    `gorm:"column:set_id"`
	IsEquipped bool      `gorm:"column:is_equipped;default:false"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

type ActiveBuff struct {
	UserID    int64     `gorm:"primaryKey;column:user_id"`
	SkillID   string    `gorm:"primaryKey;column:skill_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

type UserLoreEntry struct {
	UserID       int64     `gorm:"primaryKey;column:user_id"`
	LoreID       string    `gorm:"primaryKey;column:lore_id"`
	DiscoveredAt time.Time `gorm:"column:discovered_at;autoCreateTime"`
}

type UserHuntUnlock struct {
	UserID     int64     `gorm:"primaryKey;column:user_id"`
	ZoneKey    string    `gorm:"primaryKey;column:zone_key"`
	UnlockedAt time.Time `gorm:"column:unlocked_at;autoCreateTime"`
}

type UserHuntZoneStat struct {
	UserID    int64 `gorm:"primaryKey;column:user_id"`
	ZoneKey   string `gorm:"primaryKey;column:zone_key"`
	Wins      int   `gorm:"column:wins;default:0"`
	BossKills int   `gorm:"column:boss_kills;default:0"`
}

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

func (UserCharacter) TableName() string     { return "user_characters" }
func (UserEquipment) TableName() string     { return "user_equipment" }
func (ActiveBuff) TableName() string        { return "active_buffs" }
func (UserLoreEntry) TableName() string     { return "user_lore" }
func (UserHuntUnlock) TableName() string    { return "user_hunt_unlocks" }
func (UserHuntZoneStat) TableName() string  { return "user_hunt_zone_stats" }
func (WeeklyRank) TableName() string        { return "weekly_ranks" }
func (WeeklyModifier) TableName() string     { return "weekly_modifiers" }
