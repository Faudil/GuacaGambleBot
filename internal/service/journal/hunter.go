package journal

import (
	"guacagamblebot/internal/store"
)

// Hunter — the hunting path: hunt wins, zone unlocks and zone boss kills.
// Zones: forest, cave, desert, mountain, ocean, tundra, volcano.
func init() {
	paths["hunter"] = &Path{
		ID: "hunter", Emoji: "🏹",
		TitleKey: "journal.paths.hunter.title",
		DescKey:  "journal.paths.hunter.desc",
		Steps: []Step{
			{
				TextKey: "journal.paths.hunter.step1",
				Check:   statCheck("pve_wins", 5),
				Reward:  Reward{Money: 50},
			},
			{
				TextKey: "journal.paths.hunter.step2",
				Check:   countRowsCheck("user_hunt_unlocks", "user_id = ?", 3),
				Reward:  Reward{Money: 100},
			},
			{
				TextKey: "journal.paths.hunter.step3",
				Check:   sumColumnCheck("user_hunt_zone_stats", "boss_kills", "user_id = ?", 3),
				Reward:  Reward{Money: 150},
			},
			{
				TextKey: "journal.paths.hunter.step4",
				Check:   statCheck("pve_wins", 25),
				Reward:  Reward{Money: 200},
			},
			{
				TextKey: "journal.paths.hunter.step5",
				Check:   sumColumnCheck("user_hunt_zone_stats", "boss_kills", "user_id = ?", 10),
				Reward:  Reward{Money: 300, Crowns: 5},
			},
			{
				TextKey: "journal.paths.hunter.step6",
				Check:   countRowsCheck("user_hunt_zone_stats", "user_id = ? AND zone_key = ? AND boss_kills >= 1", 1, "volcano"),
				Reward:  Reward{Money: 400},
			},
			{
				TextKey: "journal.paths.hunter.step7",
				// Slay the boss of every hunt zone at least once.
				Check: func(s *store.Store, userID int64) (int, int, bool) {
					var zones int
					s.DB.Raw("SELECT COUNT(DISTINCT zone_key) FROM user_hunt_zone_stats WHERE user_id = ? AND boss_kills >= 1", userID).Scan(&zones)
					return zones, 7, zones >= 7
				},
				Reward: Reward{Money: 700, Crowns: 10},
			},
			{
				TextKey: "journal.paths.hunter.step8",
				Check:   sumColumnCheck("user_hunt_zone_stats", "boss_kills", "user_id = ?", 100),
				Reward:  Reward{Money: 1000, Crowns: 15, ItemIDs: []string{"volcano_egg"}},
			},
		},
	}
}
