package journal

// Champion — the combat path: duels, hunts, pets, bosses, delve, leveling.
func init() {
	paths["champion"] = &Path{
		ID: "champion", Emoji: "⚔️",
		TitleKey: "journal.paths.champion.title",
		DescKey:  "journal.paths.champion.desc",
		Steps: []Step{
			{
				TextKey: "journal.paths.champion.step1",
				Check:   statCheck("pvp_wins", 5),
				Reward:  Reward{Money: 100},
			},
			{
				TextKey: "journal.paths.champion.step2",
				Check:   statCheck("pve_wins", 15),
				Reward:  Reward{Money: 150},
			},
			{
				TextKey: "journal.paths.champion.step3",
				Check:   maxPetLevelCheck(10),
				Reward:  Reward{Money: 150, Crowns: 5},
			},
			{
				TextKey: "journal.paths.champion.step4",
				Check:   bossLeagueCheck(2),
				Reward:  Reward{Money: 300},
			},
			{
				TextKey: "journal.paths.champion.step5",
				Check:   sumColumnCheck("user_hunt_zone_stats", "boss_kills", "user_id = ?", 1),
				Reward:  Reward{Money: 300},
			},
			{
				TextKey: "journal.paths.champion.step6",
				Check:   countRowsCheck("delve_run_history", "user_id = ?", 1),
				Reward:  Reward{Money: 300, Crowns: 10},
			},
			{
				TextKey: "journal.paths.champion.step7",
				Check:   charLevelCheck(25),
				Reward:  Reward{Money: 500},
			},
			{
				TextKey: "journal.paths.champion.step8",
				Check:   bossLeagueCheck(5),
				Reward:  Reward{Money: 1000, Crowns: 15, ItemIDs: []string{"boss_trophy"}},
			},
		},
	}
}
