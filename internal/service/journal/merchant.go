package journal

// Merchant Prince — the economy path: earning, market, bank, housing,
// community projects and wealth.
func init() {
	paths["merchant"] = &Path{
		ID: "merchant", Emoji: "💰",
		TitleKey: "journal.paths.merchant.title",
		DescKey:  "journal.paths.merchant.desc",
		Steps: []Step{
			{
				TextKey:  "journal.paths.merchant.step1",
				Check:    statCheck("money_earned", 1000),
				Discover: statCheck("money_earned", 200),
				Reward:   Reward{Money: 50},
			},
			{
				TextKey: "journal.paths.merchant.step2",
				Check:   statCheck("items_sold_market", 20),
				Reward:  Reward{Money: 100},
			},
			{
				TextKey: "journal.paths.merchant.step3",
				Check:   statCheck("items_bought_market", 20),
				Reward:  Reward{Money: 100},
			},
			{
				TextKey: "journal.paths.merchant.step4",
				Check:   bankCheck(2000),
				Reward:  Reward{Money: 150},
			},
			{
				TextKey: "journal.paths.merchant.step5",
				Check:   houseLevelCheck(1),
				Reward:  Reward{Money: 200, Crowns: 5},
			},
			{
				TextKey: "journal.paths.merchant.step6",
				Check:   communityCheck(500),
				Reward:  Reward{Money: 300},
			},
			{
				TextKey: "journal.paths.merchant.step7",
				Check:   statCheck("money_earned", 10000),
				Reward:  Reward{Money: 500, Crowns: 10},
			},
			{
				TextKey: "journal.paths.merchant.step8",
				Check:   balanceCheck(100000),
				Reward:  Reward{Money: 1000, Crowns: 15},
			},
		},
	}
}
