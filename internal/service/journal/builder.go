package journal

// Builder — the social/home path: housing, furniture, sanctuary, pets care,
// research and community projects.
func init() {
	paths["builder"] = &Path{
		ID: "builder", Emoji: "🏗️",
		TitleKey: "journal.paths.builder.title",
		DescKey:  "journal.paths.builder.desc",
		Steps: []Step{
			{
				TextKey:  "journal.paths.builder.step1",
				Check:    houseLevelCheck(1),
				Discover: statCheck("pets_fed", 1),
				Reward:   Reward{Money: 100},
			},
			{
				TextKey: "journal.paths.builder.step2",
				Check:   houseLevelCheck(2),
				Reward:  Reward{Money: 200},
			},
			{
				TextKey: "journal.paths.builder.step3",
				Check:   countRowsCheck("user_furniture", "user_id = ?", 3),
				Reward:  Reward{Money: 200, Crowns: 5},
			},
			{
				TextKey: "journal.paths.builder.step4",
				Check:   columnValueCheck("user_sanctuaries", "tier", 1),
				Reward:  Reward{Money: 250},
			},
			{
				TextKey: "journal.paths.builder.step5",
				Check:   statCheck("pets_fed", 20),
				Reward:  Reward{Money: 250},
			},
			{
				TextKey: "journal.paths.builder.step6",
				Check:   communityCheck(100),
				Reward:  Reward{Money: 300, Crowns: 10},
			},
			{
				TextKey: "journal.paths.builder.step7",
				Check:   countRowsCheck("user_research", "user_id = ? AND completed = 1", 1),
				Reward:  Reward{Money: 400},
			},
			{
				TextKey: "journal.paths.builder.step8",
				Check:   communityCheck(1000),
				Reward:  Reward{Money: 800, Crowns: 15},
			},
			{
				TextKey: "journal.paths.builder.step9",
				Check:   columnValueCheck("user_sanctuaries", "tier", 3),
				Reward:  Reward{Money: 500, Crowns: 5},
			},
			{
				TextKey: "journal.paths.builder.step10",
				Check:   statCheck("fusions_done", 1),
				Reward:  Reward{Money: 600},
			},
			{
				TextKey: "journal.paths.builder.step11",
				Check:   columnValueCheck("user_sanctuaries", "tier", 5),
				Reward:  Reward{Money: 1200, Crowns: 20},
			},
		},
	}
}
