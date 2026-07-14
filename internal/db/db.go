package db

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
)

// Open connects to the SQLite database, runs migrations and returns the handle.
func Open(cfg *config.Config) (*gorm.DB, error) {
	dialector := sqlite.Open(cfg.DBPath)
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
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
		&model.Item{},
		&model.Inventory{},
		&model.LottoState{},
		&model.Job{},
		&model.UserPet{},
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
	)
}
