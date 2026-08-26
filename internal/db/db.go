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
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
)

const (
	busyTimeout  = 5000
	maxOpenConns = 8
	maxIdleConns = 4
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
	// The SQLite migrator rebuilds any table whose DDL differs from the
	// model's expectations. Its rebuild copy step skips columns with leading
	// whitespace (a parsing quirk), so a rebuild of a populated table with a
	// NOT NULL column that ends up omitted fails with a constraint error.
	// Tables created by the hand-written housing migrations historically
	// differed from the AutoMigrate output, so every restart crashed here.
	// Bring those tables into the exact AutoMigrate shape before AutoMigrate
	// gets a chance to rebuild them.
	if err := repairHousingSchema(db); err != nil {
		return err
	}
	if err := repairFurnitureSchema(db); err != nil {
		return err
	}
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
		&model.UserDailyLog{},
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
	{ID: "inventory_canonical_ids", Run: migrateInventoryCanonicalIDs},
	{ID: "inventory_cleanup_zero_quantity", Run: migrateInventoryCleanupZeroQuantity},
	{ID: "equip_slot_jewelry", Run: migrateEquipSlotJewelry},
	{ID: "job_xp_formula_rebalance", Run: migrateJobXPFormulaRebalance},
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

// repairHousingSchema rewrites user_housing into the exact DDL AutoMigrate
// would create, so AutoMigrate never attempts (and crashes on) a rebuild.
// Handled shapes: AutoMigrate-fresh (no-op), custom composite-PK shape created
// by migrateHousingCompositePK, and legacy single-PK shapes (with or without
// house_type / is_active / stored_items).
func repairHousingSchema(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.UserHousing{}) {
		return nil
	}
	pks, err := pkColumns(db, "user_housing")
	if err != nil {
		return err
	}
	if containsAll(pks, "user_id", "house_type") {
		cols, err := tableColumns(db, "user_housing")
		if err != nil {
			return err
		}
		for _, c := range cols {
			if c.Name == "is_active" && strings.EqualFold(c.Type, "numeric") {
				return nil
			}
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		legacy := map[string]bool{}
		existing, err := tableColumns(tx, "user_housing")
		if err != nil {
			return err
		}
		for _, c := range existing {
			legacy[c.Name] = true
		}

		if err := tx.Exec("ALTER TABLE user_housing RENAME TO user_housing_legacy").Error; err != nil {
			return err
		}
		if err := tx.Migrator().CreateTable(&model.UserHousing{}); err != nil {
			return err
		}

		// Build the copy expression column by column so it works regardless of
		// which columns the legacy table happened to have.
		expr := func(name, fallback string) string {
			if legacy[name] {
				return name
			}
			return fallback
		}
		sel := "user_id"
		if legacy["house_type"] {
			sel += ", COALESCE(NULLIF(house_type, ''), 'cardboard_box')"
		} else {
			sel += ", 'cardboard_box'"
		}
		sel += ", " + expr("level", "1")
		sel += ", " + expr("last_collected", "NULL")
		sel += ", " + expr("is_active", "1")
		sel += ", " + expr("custom_name", "NULL")
		sel += ", " + expr("custom_color", "NULL")
		sel += ", " + expr("under_construction", "NULL")
		sel += ", " + expr("finish_time", "NULL")
		sel += ", " + expr("stored_items", "'{}'")

		ins := "INSERT INTO user_housing (user_id, house_type, level, last_collected, is_active, custom_name, custom_color, under_construction, finish_time, stored_items) SELECT " + sel + " FROM user_housing_legacy"
		if err := tx.Exec(ins).Error; err != nil {
			return err
		}
		return tx.Exec("DROP TABLE user_housing_legacy").Error
	})
}

// repairFurnitureSchema rewrites user_furniture into the exact AutoMigrate
// shape. The legacy shape predates house-scoped furniture (no house_type
// column); those items are attached to the user's active house, mirroring
// migrateFurnitureHouseScope.
func repairFurnitureSchema(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.UserFurniture{}) {
		return nil
	}
	pks, err := pkColumns(db, "user_furniture")
	if err != nil {
		return err
	}
	if containsAll(pks, "user_id", "house_type", "furniture_id") {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("ALTER TABLE user_furniture RENAME TO user_furniture_legacy").Error; err != nil {
			return err
		}
		if err := tx.Migrator().CreateTable(&model.UserFurniture{}); err != nil {
			return err
		}

		houseExpr := "'cardboard_box'"
		if db.Migrator().HasTable(&model.UserHousing{}) {
			houseExpr = "COALESCE(uh.house_type, 'cardboard_box')"
		}
		ins := `INSERT INTO user_furniture (user_id, house_type, furniture_id, placed_at)
			SELECT lf.user_id, ` + houseExpr + `, lf.furniture_id, lf.placed_at
			FROM user_furniture_legacy lf`
		if db.Migrator().HasTable(&model.UserHousing{}) {
			ins += ` LEFT JOIN user_housing uh ON uh.user_id = lf.user_id AND uh.is_active = 1`
		}
		if err := tx.Exec(ins).Error; err != nil {
			return err
		}
		return tx.Exec("DROP TABLE user_furniture_legacy").Error
	})
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

// migrateInventoryCleanupZeroQuantity removes inventory rows that hold no
// units. They are invisible to every consumer (all lookups filter quantity >
// 0), so they only clutter the table and could confuse name-based scans.
func migrateInventoryCleanupZeroQuantity(tx *gorm.DB) error {
	if !tx.Migrator().HasTable(&model.Inventory{}) {
		return nil
	}
	return tx.Where("quantity <= 0").Delete(&model.Inventory{}).Error
}

// migrateInventoryCanonicalIDs folds inventory rows keyed by a display name
// (e.g. "Fertilizer", "Steel Pickaxe") into their canonical item IDs. Such rows
// were created by the daily shop before it stored items by ID; they are
// invisible to every lookup that uses the canonical id, so the player's items
// appear missing.
func migrateInventoryCanonicalIDs(tx *gorm.DB) error {
	if !tx.Migrator().HasTable(&model.Inventory{}) {
		return nil
	}
	var rows []model.Inventory
	if err := tx.Find(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		it := items.Get(r.ItemID)
		if it == nil || it.ID == r.ItemID {
			continue
		}
		var existing model.Inventory
		err := tx.Where("user_id = ? AND item_id = ?", r.UserID, it.ID).First(&existing).Error
		if err == nil {
			if err := tx.Model(&model.Inventory{}).
				Where("user_id = ? AND item_id = ?", r.UserID, it.ID).
				UpdateColumn("quantity", gorm.Expr("quantity + ?", r.Quantity)).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&model.Inventory{}).
				Where("user_id = ? AND item_id = ?", r.UserID, r.ItemID).
				UpdateColumn("item_id", it.ID).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("user_id = ? AND item_id = ?", r.UserID, r.ItemID).Delete(&model.Inventory{}).Error; err != nil {
			return err
		}
	}
	return nil
}

// migrateEquipSlotJewelry renames the "accessory" equipment slot to "jewelry"
// and reclassifies the few pieces that do not fit the physical-type taxonomy.
// Slots are derived from what the object is: jewelry = precious ornaments worn
// on the body (rings, amulets, pendants, brooches, talismans, crowns); trinket =
// every other non-weapon/non-armor piece (charms, masks, badges, orbs, cores,
// keys, ...). Delve drops store their generated base id (e.g.
// "delve_flaming_arcane_orb_of_the_..."), so they are matched by pattern.
func migrateEquipSlotJewelry(tx *gorm.DB) error {
	if !tx.Migrator().HasTable(&model.UserEquipment{}) {
		return nil
	}
	// accessory → jewelry, except non-precious pieces that move to trinket.
	if err := tx.Exec(`
		UPDATE user_equipment SET equip_slot = 'jewelry'
		WHERE equip_slot = 'accessory'
		  AND base_id NOT IN ('lucky_charm', 'silent_steps', 'reinforced_badge')
		  AND base_id NOT LIKE 'delve_%arcane_orb%'`).Error; err != nil {
		return err
	}
	if err := tx.Exec(`
		UPDATE user_equipment SET equip_slot = 'trinket'
		WHERE equip_slot = 'accessory'
		  AND (base_id IN ('lucky_charm', 'silent_steps', 'reinforced_badge')
		       OR base_id LIKE 'delve_%arcane_orb%')`).Error; err != nil {
		return err
	}
	// Talismans are necklaces → jewelry.
	if err := tx.Exec(`
		UPDATE user_equipment SET equip_slot = 'jewelry'
		WHERE equip_slot = 'trinket' AND base_id = 'dragon_slayer_talisman'`).Error; err != nil {
		return err
	}
	// The remap can leave a user with two equipped items in the same slot
	// (e.g. lucky_charm next to another trinket, or ring next to talisman).
	// Keep the oldest equipped piece and unequip the rest.
	return tx.Exec(`
		UPDATE user_equipment SET is_equipped = 0
		WHERE is_equipped = 1
		  AND id NOT IN (
		    SELECT MIN(id) FROM user_equipment
		    WHERE is_equipped = 1
		    GROUP BY user_id, equip_slot
		  )`).Error
}

// oldJobXPForLevel and rebalancedJobXPForLevel are frozen copies of the job
// leveling formula from before/after the 2026-08 rebalance (50+level*25 to
// 50+level*50 — see internal/service/jobs.XPForLevel). They are duplicated
// here rather than imported so this migration keeps converting between
// exactly these two formulas even if the live formula changes again later,
// and so internal/db does not import internal/service/jobs (that would form
// an import cycle through the service/store test packages that import
// internal/db to set up a real database).
func oldJobXPForLevel(level int) int {
	return 50 + level*25
}

func rebalancedJobXPForLevel(level int) int {
	return 50 + level*50
}

// migrateJobXPFormulaRebalance recomputes every job's level and XP-into-level
// after the leveling formula changed from oldJobXPForLevel to
// rebalancedJobXPForLevel. A stored (level, xp) pair encodes progress measured
// against whichever formula was active when it accrued, so it is converted
// back to total XP earned under the old formula and then re-leveled forward
// under the new one — this is the same forward-progression logic each job
// service uses when granting XP, just replayed from zero.
func migrateJobXPFormulaRebalance(tx *gorm.DB) error {
	if !tx.Migrator().HasTable(&model.Job{}) {
		return nil
	}
	var jobs []model.Job
	if err := tx.Find(&jobs).Error; err != nil {
		return err
	}
	for _, j := range jobs {
		totalXP := j.XP
		for lvl := 1; lvl < j.Level; lvl++ {
			totalXP += oldJobXPForLevel(lvl)
		}

		newLevel, newXP := 1, totalXP
		next := rebalancedJobXPForLevel(newLevel)
		for newXP >= next {
			newXP -= next
			newLevel++
			next = rebalancedJobXPForLevel(newLevel)
		}

		if newLevel == j.Level && newXP == j.XP {
			continue
		}
		if err := tx.Model(&model.Job{}).
			Where("user_id = ? AND job_name = ?", j.UserID, j.JobName).
			Updates(map[string]any{"level": newLevel, "xp": newXP}).Error; err != nil {
			return err
		}
	}
	return nil
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
