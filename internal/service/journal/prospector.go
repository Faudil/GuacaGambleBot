package journal

// Prospector — the gathering path: mining, fishing, farming, hunting, digging.
// New steps are just entries in the Steps slice; checks are plain closures.
func init() {
	paths["prospector"] = &Path{
		ID: "prospector", Emoji: "⛏️",
		TitleKey: "journal.paths.prospector.title",
		DescKey:  "journal.paths.prospector.desc",
		Steps: []Step{
			{
				TextKey:  "journal.paths.prospector.step1",
				Check:    statCheck("items_mined", 25),
				Discover: statCheck("items_mined", 5),
				Reward:   Reward{Money: 50},
			},
			{
				TextKey: "journal.paths.prospector.step2",
				Check:   statCheck("items_fished", 25),
				Reward:  Reward{Money: 50},
			},
			{
				TextKey: "journal.paths.prospector.step3",
				Check:   statCheck("items_farmed", 25),
				Reward:  Reward{Money: 50},
			},
			{
				TextKey: "journal.paths.prospector.step4",
				Check:   statCheck("pve_wins", 10),
				Reward:  Reward{Money: 150, ItemIDs: []string{"cave_egg"}},
			},
			{
				TextKey: "journal.paths.prospector.step5",
				// Choose your favorite gathering: any of the three suffices.
				Check: anyOf(
					statCheck("items_mined", 150),
					statCheck("items_fished", 100),
					statCheck("items_farmed", 100),
				),
				Reward: Reward{Money: 300, Crowns: 5},
			},
			{
				TextKey: "journal.paths.prospector.step6",
				Check:   countRowsCheck("user_fossil_harvests", "user_id = ?", 15),
				Reward:  Reward{Money: 300},
			},
			{
				TextKey: "journal.paths.prospector.step7",
				Check: and(
					statCheck("items_mined", 300),
					statCheck("items_fished", 200),
					statCheck("items_farmed", 200),
				),
				Reward: Reward{Money: 1000, Crowns: 15, ItemIDs: []string{"tundra_egg"}},
			},
		},
	}
}
