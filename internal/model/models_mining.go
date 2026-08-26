package model

import "time"

// MiningSession is a persisted in-progress mining expedition, so a bot restart
// (patch, watchdog, redeploy) does not lose the player's depth, effects or
// loot. The bag is stored as a JSON-encoded list of service/mining.BagEntry.
type MiningSession struct {
	UserID         int64     `gorm:"primaryKey;column:user_id"`
	Depth          int       `gorm:"column:depth;default:1"`
	ToolID         string    `gorm:"column:tool_id;default:''"`
	GhostVeilTurns int       `gorm:"column:ghost_veil_turns;default:0"`
	RiskMod        int       `gorm:"column:risk_mod;default:0"`
	RiskTurns      int       `gorm:"column:risk_turns;default:0"`
	Bag            string    `gorm:"column:bag;type:text"`
	Contract       string    `gorm:"column:contract;type:text"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (MiningSession) TableName() string { return "mining_sessions" }
