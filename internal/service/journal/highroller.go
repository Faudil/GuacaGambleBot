package journal

// High Roller — the casino path: coinflip, slots, blackjack, roulette, lotto,
// betting and wagers.
func init() {
	paths["highroller"] = &Path{
		ID: "highroller", Emoji: "🎰",
		TitleKey: "journal.paths.highroller.title",
		DescKey:  "journal.paths.highroller.desc",
		Steps: []Step{
			{
				TextKey:  "journal.paths.highroller.step1",
				Check:    sumStatsCheck([]string{"coinflip_won", "coinflip_lost"}, 15),
				Discover: sumStatsCheck([]string{"coinflip_won", "coinflip_lost"}, 3),
				Reward:   Reward{Money: 50},
			},
			{
				TextKey: "journal.paths.highroller.step2",
				Check:   statCheck("slots_won", 5),
				Reward:  Reward{Money: 100},
			},
			{
				TextKey: "journal.paths.highroller.step3",
				Check:   statCheck("blackjack_won", 3),
				Reward:  Reward{Money: 100},
			},
			{
				TextKey: "journal.paths.highroller.step4",
				Check:   statCheck("lotto_participations", 3),
				Reward:  Reward{Money: 150, Crowns: 5},
			},
			{
				TextKey: "journal.paths.highroller.step5",
				Check:   sumStatsCheck([]string{"roulette_won", "roulette_lost"}, 10),
				Reward:  Reward{Money: 200},
			},
			{
				TextKey: "journal.paths.highroller.step6",
				Check:   statCheck("bets_won", 1),
				Reward:  Reward{Money: 150},
			},
			{
				TextKey: "journal.paths.highroller.step7",
				Check:   statCheck("wagers_won", 2),
				Reward:  Reward{Money: 200},
			},
			{
				TextKey: "journal.paths.highroller.step8",
				Check: sumStatsCheck([]string{
					"slots_money_won", "coinflip_money_won",
					"blackjack_money_won", "roulette_money_won",
				}, 10000),
				Reward: Reward{Money: 1000, Crowns: 15},
			},
		},
	}
}
