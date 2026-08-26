package model

import "time"

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

type UserFarming struct {
	ID         int64     `gorm:"primaryKey;column:id;autoIncrement"`
	UserID     int64     `gorm:"index:idx_user_farming_user;column:user_id"`
	ZoneKey    string    `gorm:"column:zone_key"`
	PlotIndex  int       `gorm:"column:plot_index"`
	ItemName   string    `gorm:"column:item_name"`
	PlantTime  time.Time `gorm:"column:plant_time"`
	GrowTime   int       `gorm:"column:grow_time"`
	Watered    bool      `gorm:"column:watered;default:false"`
	Mutated    bool      `gorm:"column:mutated;default:false"`
	Mysterious bool      `gorm:"column:mysterious;default:false"`
}

type UserAchievement struct {
	UserID        int64  `gorm:"primaryKey;column:user_id"`
	AchievementID string `gorm:"primaryKey;column:achievement_id"`
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

// QuestNotification is a queued quest-event message waiting to be surfaced to
// the user by the next command they run.
type QuestNotification struct {
	ID          uint      `gorm:"primaryKey;column:id;autoIncrement"`
	UserID      int64     `gorm:"index:idx_qn_user_created;column:user_id"`
	QuestID     string    `gorm:"column:quest_id"`
	Completed   bool      `gorm:"column:completed;default:false"`
	NextStepKey string    `gorm:"column:next_step_key"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
}

// JournalScene is a queued atmospheric scene (Chronicler intro, rank-up moment,
// recognition line). Key is an i18n key, Params its replacements. DM requests
// private-message delivery with a fallback to the activity result.
type JournalScene struct {
	ID        uint      `gorm:"primaryKey;column:id;autoIncrement"`
	UserID    int64     `gorm:"index:idx_js_user_created;column:user_id"`
	Key       string    `gorm:"column:key"`
	Params    string    `gorm:"column:params;type:text"`
	DM        bool      `gorm:"column:dm;default:false"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

// DataMigration records a one-time data migration that has been applied, so it
// runs exactly once across restarts. Migrations are registered in
// internal/db and executed right after AutoMigrate.
type DataMigration struct {
	ID        string    `gorm:"primaryKey;column:id"`
	AppliedAt time.Time `gorm:"column:applied_at;autoCreateTime"`
}

func (UserCropHarvest) TableName() string   { return "user_crop_harvests" }
func (UserFossilHarvest) TableName() string { return "user_fossil_harvests" }
func (UserFarming) TableName() string       { return "user_farming" }
func (UserAchievement) TableName() string   { return "user_achievements" }
func (UserQuest) TableName() string         { return "user_quests" }
func (UserQuestData) TableName() string     { return "user_quest_data" }
func (QuestNotification) TableName() string { return "quest_notifications" }
func (JournalScene) TableName() string      { return "journal_scenes" }
func (DataMigration) TableName() string     { return "data_migrations" }
