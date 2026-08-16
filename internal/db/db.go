package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
)

const (
	busyTimeout  = 5000
	maxOpenConns = 4
	maxIdleConns = 2
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
	// synchronous=NORMAL keeps WAL durability (recovery-safe) while avoiding an
	// fsync on every commit, drastically reducing write stalls. A hang in one
	// commit can no longer freeze the whole bot on the single connection.
	db.Exec("PRAGMA synchronous=NORMAL")
	// Keep the WAL small so reads never have to scan a large log.
	db.Exec("PRAGMA wal_autocheckpoint=512")
	db.Exec(fmt.Sprintf("PRAGMA busy_timeout=%d", busyTimeout))

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	// Recycle pooled connections so a wedged or poisoned connection (e.g. a
	// stuck file descriptor after an interrupted write) is eventually replaced
	// instead of blocking the pool forever.
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	if err := Migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

// Migrate creates/updates all tables to match the model definitions, then runs
// any pending one-time data migrations.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
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
		&model.JournalScene{},
		&model.DataMigration{},
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
		&model.MiningSession{},
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
	); err != nil {
		return err
	}
	return runDataMigrations(db)
}

// DataMigration is a one-time, idempotent data fix applied after schema
// migrations. Each entry is guarded by a marker row in the data_migrations
// table so it runs exactly once, even across restarts.
type DataMigration struct {
	ID  string
	Run func(tx *gorm.DB) error
}

// dataMigrations lists all registered data migrations in application order.
var dataMigrations = []DataMigration{
	{ID: "tutorial_step_reorder", Run: migrateTutorialSteps},
	{ID: "tutorial_rewind_skipped_hunt", Run: migrateTutorialRewindSkippedHunt},
}

// runDataMigrations applies any data migrations not yet recorded.
func runDataMigrations(db *gorm.DB) error {
	for _, m := range dataMigrations {
		var count int64
		if err := db.Model(&model.DataMigration{}).Where("id = ?", m.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := m.Run(tx); err != nil {
				return err
			}
			return tx.Create(&model.DataMigration{ID: m.ID}).Error
		})
		if err != nil {
			return fmt.Errorf("data migration %s: %w", m.ID, err)
		}
	}
	return nil
}

// tutorialStepRemap maps the old tutorial step_index to the new order introduced
// by the tutorial reorder (dig after hunting/pet care, housing after selling at
// the market). Identity entries are omitted.
var tutorialStepRemap = map[int]int{
	7:  12, // dig
	8:  13, // capsule
	9:  16, // house requirement
	10: 17, // deposit intro
	11: 18, // deposit
	12: 7,  // old egg grant step → new egg grant step (still grants the egg)
	13: 8,  // hunt
	14: 9,  // feed intro
	15: 10, // feed
	16: 11, // dig intro
	17: 20, // casino
	18: 14, // market intro → sell activity
	19: 14, // sell activity
	20: 15, // market done
	21: 22, // community intro
	22: 23, // community requirement
	23: 24, // community done
	24: 25, // delve intro
	25: 26, // delve
	26: 27, // vault key
	27: 28, // guardian event
	28: 29, // boss
	29: 30, // boss transition
	30: 31, // finale
}

// migrateTutorialSteps remaps the step_index of ACTIVE tutorial quests to the
// reordered step layout. Completed quests are left untouched.
func migrateTutorialSteps(tx *gorm.DB) error {
	var stmt strings.Builder
	stmt.WriteString("UPDATE user_quest_data SET step_index = CASE step_index")
	for old, new := range tutorialStepRemap {
		fmt.Fprintf(&stmt, " WHEN %d THEN %d", old, new)
	}
	stmt.WriteString(" ELSE step_index END WHERE quest_id = 'tutorial' AND EXISTS (SELECT 1 FROM user_quests uq WHERE uq.user_id = user_quest_data.user_id AND uq.quest_id = 'tutorial' AND uq.status = 'ACTIVE')")
	return tx.Exec(stmt.String()).Error
}

// migrateTutorialRewindSkippedHunt rewinds ACTIVE tutorial players who sit past
// the reordered hunting/pet-care block (step index >= 11) without ever having
// hunted (no pve_wins recorded) back to the egg-grant step, so nobody skips the
// new content. Hunting always records a win, so legitimate fresh players who
// progressed through the block are never touched.
func migrateTutorialRewindSkippedHunt(tx *gorm.DB) error {
	return tx.Exec(`
		UPDATE user_quest_data
		SET step_index = 7, progress_value = 0, custom_data = '{}'
		WHERE quest_id = 'tutorial'
		  AND step_index >= 11
		  AND EXISTS (SELECT 1 FROM user_quests uq
		              WHERE uq.user_id = user_quest_data.user_id
		                AND uq.quest_id = 'tutorial' AND uq.status = 'ACTIVE')
		  AND NOT EXISTS (SELECT 1 FROM user_stats us
		                  WHERE us.user_id = user_quest_data.user_id AND us.pve_wins > 0)`).Error
}
