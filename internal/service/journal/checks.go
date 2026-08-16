package journal

import (
	"guacagamblebot/internal/store"
)

// This file holds reusable Check builders. Column and table names passed to
// these helpers come from the path files (trusted literals, never user input).

// statCheck reads a numeric column from the user_stats table.
func statCheck(column string, target int) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		v := statValue(s, userID, column)
		return v, target, v >= target
	}
}

// sumStatsCheck sums several user_stats columns (e.g. total casino plays).
func sumStatsCheck(columns []string, target int) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		total := 0
		for _, col := range columns {
			total += statValue(s, userID, col)
		}
		return total, target, total >= target
	}
}

// countRowsCheck counts matching rows in an arbitrary table. The first "?" in
// where is always bound to the user ID; args are bound after it.
func countRowsCheck(table, where string, target int, args ...any) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		n := countRows(s, table, where, append([]any{userID}, args...)...)
		return n, target, n >= target
	}
}

// itemQtyCheck sums the quantity of an item in the player's inventory.
func itemQtyCheck(itemID string, target int) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		q := itemQty(s, userID, itemID)
		return q, target, q >= target
	}
}

// charLevelCheck requires the character to reach a level.
func charLevelCheck(level int) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		v := charValue(s, userID, "level")
		return v, level, v >= level
	}
}

// maxPetLevelCheck requires the player's highest-level pet to reach a level.
func maxPetLevelCheck(level int) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		var v int
		s.DB.Raw("SELECT COALESCE(MAX(level), 0) FROM user_pets WHERE user_id = ?", userID).Scan(&v)
		return v, level, v >= level
	}
}

// houseLevelCheck requires owning a house at or above a level.
func houseLevelCheck(level int) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		v := 0
		var h int
		if err := s.DB.Raw("SELECT COALESCE(MAX(level), 0) FROM user_housing WHERE user_id = ?", userID).Scan(&h).Error; err == nil {
			v = h
		}
		return v, level, v >= level
	}
}

// bankCheck requires the bank balance to reach an amount.
func bankCheck(target int) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		v := 0
		s.DB.Raw("SELECT COALESCE(bank, 0) FROM users WHERE user_id = ?", userID).Scan(&v)
		return v, target, v >= target
	}
}

// balanceCheck requires the wallet balance to reach an amount.
func balanceCheck(target int) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		v := 0
		s.DB.Raw("SELECT COALESCE(balance, 0) FROM users WHERE user_id = ?", userID).Scan(&v)
		return v, target, v >= target
	}
}

// bossLeagueCheck requires beating a number of Boss League stages.
func bossLeagueCheck(stage int) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		v := 0
		s.DB.Raw("SELECT COALESCE(boss_league_stage, 0) FROM users WHERE user_id = ?", userID).Scan(&v)
		return v, stage, v >= stage
	}
}

// questDoneCheck requires a quest to be completed.
func questDoneCheck(questID string) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		done := false
		s.DB.Raw("SELECT status = 'COMPLETED' FROM user_quests WHERE user_id = ? AND quest_id = ?", userID, questID).Scan(&done)
		if done {
			return 1, 1, true
		}
		return 0, 1, false
	}
}

// sumColumnCheck sums a column over matching rows in a table.
func sumColumnCheck(table, column, where string, target int, args ...any) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		var v int
		stmt := "SELECT COALESCE(SUM(" + column + "), 0) FROM " + table + " WHERE " + where
		stmtArgs := append([]any{userID}, args...)
		s.DB.Raw(stmt, stmtArgs...).Scan(&v)
		return v, target, v >= target
	}
}

// columnValueCheck reads a single column from the player's row in a table.
func columnValueCheck(table, column string, target int) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		var v int
		s.DB.Raw("SELECT COALESCE("+column+", 0) FROM "+table+" WHERE user_id = ?", userID).Scan(&v)
		return v, target, v >= target
	}
}

// flagCheck requires a delve flag to be earned.
func flagCheck(flagID string) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		n := countRows(s, "user_delve_flags", "user_id = ? AND flag_id = ?", userID, flagID)
		return n, 1, n >= 1
	}
}

// theftCheck requires a number of successful thefts.
func theftCheck(target int) Check {
	return countRowsCheck("theft_records", "thief_id = ? AND success = 1", target)
}

// communityCheck requires a total contribution (items + money) to community
// projects across servers.
func communityCheck(target int) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		var v int
		s.DB.Raw("SELECT COALESCE(SUM(total_items_invested + total_money_invested), 0) FROM user_community_stats WHERE user_id = ?", userID).Scan(&v)
		return v, target, v >= target
	}
}

// and combines several checks: all must pass, progress is summed.
func and(checks ...Check) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		progress, target, allDone := 0, 0, true
		for _, c := range checks {
			p, t, done := c(s, userID)
			progress += p
			target += t
			if !done {
				allDone = false
			}
		}
		return progress, target, allDone
	}
}

// anyOf passes when at least one check passes, showing the furthest progress.
func anyOf(checks ...Check) Check {
	return func(s *store.Store, userID int64) (int, int, bool) {
		bestP, bestT := 0, 1
		for _, c := range checks {
			p, t, done := c(s, userID)
			if done {
				return p, t, true
			}
			if t > 0 && float64(p)/float64(t) > float64(bestP)/float64(bestT) {
				bestP, bestT = p, t
			}
		}
		return bestP, bestT, false
	}
}

func statValue(s *store.Store, userID int64, column string) int {
	var v int
	s.DB.Raw("SELECT COALESCE("+column+", 0) FROM user_stats WHERE user_id = ?", userID).Scan(&v)
	return v
}

func charValue(s *store.Store, userID int64, column string) int {
	var v int
	s.DB.Raw("SELECT COALESCE("+column+", 0) FROM user_characters WHERE user_id = ?", userID).Scan(&v)
	return v
}

func countRows(s *store.Store, table, where string, args ...any) int {
	var n int64
	s.DB.Table(table).Where(where, args...).Count(&n)
	return int(n)
}

func itemQty(s *store.Store, userID int64, itemID string) int {
	var q int64
	s.DB.Table("inventory").Where("user_id = ? AND item_id = ?", userID, itemID).Select("COALESCE(SUM(quantity), 0)").Scan(&q)
	return int(q)
}
