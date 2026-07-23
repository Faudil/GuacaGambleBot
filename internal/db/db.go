package db

import (
	"context"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
)

// noNotFoundLogger wraps the default GORM logger so that ErrRecordNotFound is
// silently ignored — these are expected in many normal code paths (empty tables,
// pre-setup guilds, etc.) and are handled gracefully by callers.
type noNotFoundLogger struct {
	logger.Interface
}

func (l noNotFoundLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if err == gorm.ErrRecordNotFound {
		return
	}
	l.Interface.Trace(ctx, begin, fc, err)
}

// Open connects to the SQLite database, runs migrations and returns the handle.
func Open(cfg *config.Config) (*gorm.DB, error) {
	dialector := sqlite.Open(cfg.DBPath)
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: noNotFoundLogger{logger.Default.LogMode(logger.Warn)},
	})
	if err != nil {
		return nil, err
	}
	if err := Migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

// Migrate creates/updates all tables to match the model definitions.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.Cooldown{},
		&model.GameLimit{},
		&model.Bet{},
		&model.Wager{},
		&model.Inventory{},
		&model.LottoState{},
		&model.Job{},
		&model.UserPet{},
		&model.UserPetSkill{},
		&model.UserStat{},
		&model.UserAchievement{},
		&model.ServerPetElo{},
		&model.ServerSetting{},
		&model.ServerLottoState{},
		&model.UserHousing{},
		&model.UserHousingUpgrade{},
		&model.PetExpedition{},
		&model.UserFarming{},
		&model.UserQuest{},
		&model.UserQuestData{},
		&model.UserNPCReputation{},
		&model.UserNPCDailyRep{},
		&model.ServerProject{},
		&model.ServerProjectContribution{},
		&model.UserCommunityStat{},
		&model.Loan{},
		&model.UserCharacter{},
		&model.CharacterEquipment{},
		&model.ActiveBuff{},
		&model.UserLoreEntry{},
	)
}
