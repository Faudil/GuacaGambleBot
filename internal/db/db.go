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
		&model.WinRecord{},
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
	{ID: "housing_composite_pk", Run: migrateHousingCompositePK},
	{ID: "housing_activate_existing", Run: migrateHousingActivateExisting},
	{ID: "furniture_house_scope", Run: migrateFurnitureHouseScope},
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

// migrateHousingActivateExisting marks every existing housing row as the active
// house. Before multiple houses per user existed, each user had exactly one row,
// so promoting them all is correct. Fresh databases have no rows and are a no-op.
func migrateHousingActivateExisting(tx *gorm.DB) error {
	if !tx.Migrator().HasTable(&model.UserHousing{}) {
		return nil
	}
	return tx.Exec("UPDATE user_housing SET is_active = 1").Error
}

// tableColumn is a row from `PRAGMA table_info`.
type tableColumn struct {
	CID  int     `gorm:"column:cid"`
	Name string  `gorm:"column:name"`
	Type string  `gorm:"column:type"`
	NN   int     `gorm:"column:notnull"`
	Dflt *string `gorm:"column:dflt_value"`
	PK   int     `gorm:"column:pk"`
}

// tableColumns returns the column definitions for a table.
func tableColumns(tx *gorm.DB, table string) ([]tableColumn, error) {
	var cols []tableColumn
	if err := tx.Raw("PRAGMA table_info(" + table + ")").Scan(&cols).Error; err != nil {
		return nil, err
	}
	return cols, nil
}

// pkColumns returns the ordered primary-key column names of a table.
func pkColumns(tx *gorm.DB, table string) ([]string, error) {
	cols, err := tableColumns(tx, table)
	if err != nil {
		return nil, err
	}
	var pks []string
	for _, c := range cols {
		if c.PK > 0 {
			pks = append(pks, c.Name)
		}
	}
	return pks, nil
}

func containsAll(haystack []string, needles ...string) bool {
	for _, n := range needles {
		found := false
		for _, h := range haystack {
			if h == n {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// migrateHousingCompositePK rebuilds legacy user_housing tables whose primary
// key is only user_id (created before multiple houses per user existed). GORM's
// AutoMigrate cannot alter primary keys on an existing table, so the table is
// recreated explicitly with the composite (user_id, house_type) key and the
// is_active column. Existing rows are preserved; a missing is_active column is
// backfilled to 0 here and promoted to 1 by migrateHousingActivateExisting.
func migrateHousingCompositePK(tx *gorm.DB) error {
	if !tx.Migrator().HasTable(&model.UserHousing{}) {
		return nil
	}
	pks, err := pkColumns(tx, "user_housing")
	if err != nil {
		return err
	}
	if containsAll(pks, "user_id", "house_type") {
		return nil
	}
	cols, err := tableColumns(tx, "user_housing")
	if err != nil {
		return err
	}
	hasActive := false
	for _, c := range cols {
		if c.Name == "is_active" {
			hasActive = true
			break
		}
	}
	sel := `SELECT user_id, house_type, level, last_collected, custom_name, custom_color, under_construction, finish_time, stored_items`
	if hasActive {
		sel += `, is_active`
	} else {
		sel += `, 0`
	}
	ins := `user_id, house_type, level, last_collected, custom_name, custom_color, under_construction, finish_time, stored_items, is_active`
	return tx.Exec(`
CREATE TABLE user_housing_new (
	user_id INTEGER NOT NULL,
	house_type TEXT NOT NULL,
	level INTEGER DEFAULT 1,
	last_collected DATETIME,
	is_active INTEGER DEFAULT 0,
	custom_name TEXT,
	custom_color TEXT,
	under_construction TEXT,
	finish_time DATETIME,
	stored_items TEXT DEFAULT '{}',
	PRIMARY KEY (user_id, house_type)
);
INSERT INTO user_housing_new (` + ins + `) ` + sel + ` FROM user_housing;
DROP TABLE user_housing;
ALTER TABLE user_housing_new RENAME TO user_housing;`).Error
}

// migrateFurnitureHouseScope ties placed furniture to the user's active house by
// rebuilding user_furniture with house_type in its primary key. Legacy rows were
// global to the user; they are reassigned to the user's active house (or dropped
// when the user has no house). Runs after housing_activate_existing so is_active
// reflects the true active house.
func migrateFurnitureHouseScope(tx *gorm.DB) error {
	if !tx.Migrator().HasTable(&model.UserFurniture{}) {
		return nil
	}
	pks, err := pkColumns(tx, "user_furniture")
	if err != nil {
		return err
	}
	if containsAll(pks, "user_id", "house_type", "furniture_id") {
		return nil
	}
	return tx.Exec(`
CREATE TABLE user_furniture_new (
	user_id INTEGER NOT NULL,
	house_type TEXT NOT NULL,
	furniture_id TEXT NOT NULL,
	placed_at DATETIME,
	PRIMARY KEY (user_id, house_type, furniture_id)
);
INSERT INTO user_furniture_new (user_id, house_type, furniture_id, placed_at)
	SELECT uf.user_id, uh.house_type, uf.furniture_id, uf.placed_at
	FROM user_furniture uf
	JOIN user_housing uh ON uh.user_id = uf.user_id AND uh.is_active = 1;
DROP TABLE user_furniture;
ALTER TABLE user_furniture_new RENAME TO user_furniture;`).Error
}
