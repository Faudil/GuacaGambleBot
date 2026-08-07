package journal

import (
	"guacagamblebot/internal/store"
)

// Historian — the knowledge path: lore fragments, fossils, research,
// achievements, NPC secrets and reputation.
func init() {
	paths["historian"] = &Path{
		ID: "historian", Emoji: "📜",
		TitleKey: "journal.paths.historian.title",
		DescKey:  "journal.paths.historian.desc",
		Steps: []Step{
			{
				TextKey: "journal.paths.historian.step1",
				Check:   countRowsCheck("user_lore", "user_id = ?", 5),
				Reward:  Reward{Money: 50},
			},
			{
				TextKey: "journal.paths.historian.step2",
				Check:   sumColumnCheck("user_fossil_harvests", "count", "user_id = ?", 10),
				Reward:  Reward{Money: 100},
			},
			{
				TextKey: "journal.paths.historian.step3",
				Check:   countRowsCheck("user_research", "user_id = ? AND completed = 1", 1),
				Reward:  Reward{Money: 150},
			},
			{
				TextKey: "journal.paths.historian.step4",
				Check:   countRowsCheck("user_lore", "user_id = ?", 20),
				Reward:  Reward{Money: 200, Crowns: 5},
			},
			{
				TextKey: "journal.paths.historian.step5",
				Check:   countRowsCheck("user_achievements", "user_id = ?", 25),
				Reward:  Reward{Money: 300},
			},
			{
				TextKey: "journal.paths.historian.step6",
				Check:   countRowsCheck("user_npc_secrets", "user_id = ?", 3),
				Reward:  Reward{Money: 300, Crowns: 5},
			},
			{
				TextKey: "journal.paths.historian.step7",
				// Reach reputation level 5 with any NPC.
				Check: func(s *store.Store, userID int64) (int, int, bool) {
					var level int
					s.DB.Raw("SELECT COALESCE(MAX(level), 0) FROM user_npc_reputation WHERE user_id = ?", userID).Scan(&level)
					return level, 5, level >= 5
				},
				Reward: Reward{Money: 500},
			},
			{
				TextKey: "journal.paths.historian.step8",
				Check:   countRowsCheck("user_lore", "user_id = ?", 45),
				Reward:  Reward{Money: 1000, Crowns: 15},
			},
		},
	}
}
