package db

import (
	"context"
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
)

const (
	busyTimeout  = 5000
	maxOpenConns = 1
	maxIdleConns = 1
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

	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec(fmt.Sprintf("PRAGMA busy_timeout=%d", busyTimeout))

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)

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
		&model.UserCropHarvest{},
		&model.UserFossilHarvest{},
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
		&model.QuestNotification{},
		&model.UserNPCReputation{},
		&model.UserNPCDailyRep{},
		&model.UserNPCSecret{},
		&model.ServerProject{},
		&model.ServerProjectContribution{},
		&model.UserCommunityStat{},
		&model.Loan{},
		&model.UserCharacter{},
		&model.UserEquipment{},
		&model.ActiveBuff{},
		&model.MarketState{},
		&model.UserLoreEntry{},
		&model.UserHuntUnlock{},
		&model.UserHuntZoneStat{},
		&model.UserFurniture{},
		&model.UserResearch{},
		&model.WeeklyRank{},
		&model.WeeklyModifier{},
		&model.UserPetArtifact{},
		&model.DelveSession{},
		&model.UserDelveFlag{},
		&model.DelveRunHistory{},
		&model.DelveGauntletScore{},
		&model.UserSanctuary{},
		&model.UserCriminality{},
		&model.WorldCriminalityState{},
		&model.Bounty{},
		&model.TheftRecord{},
		&model.CrimeRecord{},
		&model.HunterContract{},
		&model.VeilRaid{},
		&model.VeilRaidLockout{},
		&model.VeilRaidHallOfFame{},
		&model.UserJournalEntry{},
		&model.UserJournalMastery{},
	)
}
