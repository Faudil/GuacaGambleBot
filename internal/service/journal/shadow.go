package journal

// Shadow — the crime path: thefts, notoriety and infamy.
func init() {
	paths["shadow"] = &Path{
		ID: "shadow", Emoji: "🌑",
		TitleKey: "journal.paths.shadow.title",
		DescKey:  "journal.paths.shadow.desc",
		Steps: []Step{
			{
				TextKey:  "journal.paths.shadow.step1",
				Check:    theftCheck(1),
				Discover: countRowsCheck("theft_records", "thief_id = ?", 1),
				Reward:   Reward{Money: 100},
			},
			{
				TextKey: "journal.paths.shadow.step2",
				Check:   theftCheck(5),
				Reward:  Reward{Money: 200},
			},
			{
				TextKey: "journal.paths.shadow.step3",
				Check:   columnValueCheck("user_criminality", "notoriety", 50),
				Reward:  Reward{Money: 200, Crowns: 5},
			},
			{
				TextKey: "journal.paths.shadow.step4",
				Check:   theftCheck(15),
				Reward:  Reward{Money: 400},
			},
			{
				TextKey: "journal.paths.shadow.step5",
				Check:   columnValueCheck("user_criminality", "thief_infamy", 50),
				Reward:  Reward{Money: 400, Crowns: 10},
			},
			{
				TextKey: "journal.paths.shadow.step6",
				Check:   theftCheck(30),
				Reward:  Reward{Money: 800},
			},
			{
				TextKey: "journal.paths.shadow.step7",
				Check:   columnValueCheck("user_criminality", "has_mask", 1),
				Reward:  Reward{Money: 1000, Crowns: 15},
			},
		},
	}
}
