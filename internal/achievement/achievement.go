package achievement

import (
	"gorm.io/gorm"

	"guacagamblebot/internal/model"
)

// Achievement defines an unlockable with a predicate over player stats.
type Achievement struct {
	ID     string
	Emoji  string
	Glory  int
	Hidden bool
	Check  func(stats map[string]any) bool
}

var registry = map[string]*Achievement{}

func register(id, emoji string, glory int, check func(map[string]any) bool) {
	registry[id] = &Achievement{ID: id, Emoji: emoji, Glory: glory, Check: check}
}

func init() {
	num := func(s map[string]any, k string) int {
		if v, ok := s[k].(int); ok {
			return v
		}
		return 0
	}
	has := func(s map[string]any, k string) bool { return num(s, k) >= 1 }

	register("pvp_rookie", "⚔️", 10, func(s map[string]any) bool { return num(s, "pvp_wins") >= 1 })
	register("pvp_gladiator", "🏟️", 50, func(s map[string]any) bool { return num(s, "pvp_wins") >= 50 })
	register("pvp_punching_bag", "🩹", 5, func(s map[string]any) bool { return num(s, "pvp_losses") >= 10 })
	register("pve_hunter", "🌲", 20, func(s map[string]any) bool { return num(s, "pve_wins") >= 25 })

	// Hunt zone discoveries: one per progressive zone.
	register("hunt_unlock_mountain", "🏔️", 50, func(s map[string]any) bool { return num(s, "hunt_unlocked_mountain") >= 1 })
	register("hunt_unlock_ocean", "🌊", 100, func(s map[string]any) bool { return num(s, "hunt_unlocked_ocean") >= 1 })
	register("hunt_unlock_tundra", "❄️", 200, func(s map[string]any) bool { return num(s, "hunt_unlocked_tundra") >= 1 })
	register("hunt_unlock_volcano", "🌋", 400, func(s map[string]any) bool { return num(s, "hunt_unlocked_volcano") >= 1 })

	// Zone boss kills: first boss of each zone.
	register("hunt_boss_forest", "🌳", 20, func(s map[string]any) bool { return num(s, "hunt_boss_forest") >= 1 })
	register("hunt_boss_cave", "👑", 30, func(s map[string]any) bool { return num(s, "hunt_boss_cave") >= 1 })
	register("hunt_boss_desert", "🦂", 40, func(s map[string]any) bool { return num(s, "hunt_boss_desert") >= 1 })
	register("hunt_boss_mountain", "🗿", 60, func(s map[string]any) bool { return num(s, "hunt_boss_mountain") >= 1 })
	register("hunt_boss_ocean", "🐙", 80, func(s map[string]any) bool { return num(s, "hunt_boss_ocean") >= 1 })
	register("hunt_boss_tundra", "🐺", 120, func(s map[string]any) bool { return num(s, "hunt_boss_tundra") >= 1 })
	register("hunt_boss_volcano", "🐉", 200, func(s map[string]any) bool { return num(s, "hunt_boss_volcano") >= 1 })

	// Global boss kill milestones.
	register("hunt_boss_10", "🎯", 50, func(s map[string]any) bool { return num(s, "hunt_boss_total") >= 10 })
	register("hunt_boss_50", "⚔️", 200, func(s map[string]any) bool { return num(s, "hunt_boss_total") >= 50 })
	register("hunt_boss_100", "🏆", 500, func(s map[string]any) bool { return num(s, "hunt_boss_total") >= 100 })

	register("eco_1k", "💵", 20, func(s map[string]any) bool { return num(s, "balance") >= 1000 })
	register("eco_10k", "💸", 50, func(s map[string]any) bool { return num(s, "balance") >= 10000 })
	register("eco_50k", "📈", 100, func(s map[string]any) bool { return num(s, "balance") >= 50000 })
	register("eco_100k", "🤑", 200, func(s map[string]any) bool { return num(s, "balance") >= 100000 })
	register("eco_1m", "👑", 500, func(s map[string]any) bool { return num(s, "balance") >= 1000000 })
	register("eco_rich", "💰", 100, func(s map[string]any) bool { return num(s, "money_earned") >= 10000 })

	register("job_miner", "⛏️", 30, func(s map[string]any) bool { return num(s, "items_mined") >= 100 })
	register("pet_feeder", "🍖", 20, func(s map[string]any) bool { return num(s, "pets_fed") >= 50 })
	register("pet_level_10", "🥚", 20, func(s map[string]any) bool { return num(s, "max_pet_level") >= 10 })
	register("pet_level_20", "🐾", 50, func(s map[string]any) bool { return num(s, "max_pet_level") >= 20 })
	register("pet_level_50", "🐉", 100, func(s map[string]any) bool { return num(s, "max_pet_level") >= 35 })
	register("pet_level_100", "✨", 300, func(s map[string]any) bool { return num(s, "max_pet_level") >= 50 })

	register("coinflip_won_1k", "🪙", 10, func(s map[string]any) bool { return num(s, "coinflip_money_won") >= 1000 })
	register("coinflip_won_5k", "🪙", 20, func(s map[string]any) bool { return num(s, "coinflip_money_won") >= 5000 })
	register("coinflip_won_100k", "🪙", 100, func(s map[string]any) bool { return num(s, "coinflip_money_won") >= 100000 })
	register("coinflip_won_1m", "💰", 500, func(s map[string]any) bool { return num(s, "coinflip_money_won") >= 1000000 })
	register("coinflip_lost_1k", "🌧️", 10, func(s map[string]any) bool { return num(s, "coinflip_money_lost") >= 1000 })
	register("coinflip_lost_5k", "🌧️", 20, func(s map[string]any) bool { return num(s, "coinflip_money_lost") >= 5000 })
	register("coinflip_lost_100k", "💸", 100, func(s map[string]any) bool { return num(s, "coinflip_money_lost") >= 100000 })
	register("coinflip_lost_1m", "🤡", 500, func(s map[string]any) bool { return num(s, "coinflip_money_lost") >= 1000000 })

	register("slots_won_1k", "🎰", 10, func(s map[string]any) bool { return num(s, "slots_money_won") >= 1000 })
	register("slots_won_5k", "🎰", 20, func(s map[string]any) bool { return num(s, "slots_money_won") >= 5000 })
	register("slots_won_100k", "🎰", 100, func(s map[string]any) bool { return num(s, "slots_money_won") >= 100000 })
	register("slots_won_1m", "🎰", 500, func(s map[string]any) bool { return num(s, "slots_money_won") >= 1000000 })
	register("slots_lost_1k", "😠", 10, func(s map[string]any) bool { return num(s, "slots_money_lost") >= 1000 })
	register("slots_lost_5k", "😠", 20, func(s map[string]any) bool { return num(s, "slots_money_lost") >= 5000 })
	register("slots_lost_100k", "📉", 100, func(s map[string]any) bool { return num(s, "slots_money_lost") >= 100000 })
	register("slots_lost_1m", "💸", 500, func(s map[string]any) bool { return num(s, "slots_money_lost") >= 1000000 })

	register("blackjack_won_1k", "🃏", 10, func(s map[string]any) bool { return num(s, "blackjack_money_won") >= 1000 })
	register("blackjack_won_5k", "🃏", 20, func(s map[string]any) bool { return num(s, "blackjack_money_won") >= 5000 })
	register("blackjack_won_100k", "🃏", 100, func(s map[string]any) bool { return num(s, "blackjack_money_won") >= 100000 })
	register("blackjack_won_1m", "🃏", 500, func(s map[string]any) bool { return num(s, "blackjack_money_won") >= 1000000 })
	register("blackjack_lost_1k", "🤦", 10, func(s map[string]any) bool { return num(s, "blackjack_money_lost") >= 1000 })
	register("blackjack_lost_5k", "🤦", 20, func(s map[string]any) bool { return num(s, "blackjack_money_lost") >= 5000 })
	register("blackjack_lost_100k", "😔", 100, func(s map[string]any) bool { return num(s, "blackjack_money_lost") >= 100000 })
	register("blackjack_lost_1m", "🏚️", 500, func(s map[string]any) bool { return num(s, "blackjack_money_lost") >= 1000000 })

	register("roulette_won_1k", "🔫", 10, func(s map[string]any) bool { return num(s, "roulette_money_won") >= 1000 })
	register("roulette_won_5k", "🔫", 20, func(s map[string]any) bool { return num(s, "roulette_money_won") >= 5000 })
	register("roulette_won_100k", "🔫", 100, func(s map[string]any) bool { return num(s, "roulette_money_won") >= 100000 })
	register("roulette_won_1m", "🔫", 500, func(s map[string]any) bool { return num(s, "roulette_money_won") >= 1000000 })
	register("roulette_lost_1k", "🩸", 10, func(s map[string]any) bool { return num(s, "roulette_money_lost") >= 1000 })
	register("roulette_lost_5k", "🩸", 20, func(s map[string]any) bool { return num(s, "roulette_money_lost") >= 5000 })
	register("roulette_lost_100k", "💀", 100, func(s map[string]any) bool { return num(s, "roulette_money_lost") >= 100000 })
	register("roulette_lost_1m", "🪦", 500, func(s map[string]any) bool { return num(s, "roulette_money_lost") >= 1000000 })

	register("bet_rookie", "🎲", 10, func(s map[string]any) bool { return num(s, "wagers_won") >= 1 })
	register("bet_pro", "🔮", 50, func(s map[string]any) bool { return num(s, "wagers_won") >= 25 })

	register("lotto_rookie", "🎫", 10, func(s map[string]any) bool { return num(s, "lotto_participations") >= 10 })
	register("lotto_winner", "🎉", 200, func(s map[string]any) bool { return num(s, "lotto_won") >= 1 })

	register("daily_1", "📅", 10, func(s map[string]any) bool { return num(s, "daily_uses") >= 1 })
	register("daily_10", "📅", 20, func(s map[string]any) bool { return num(s, "daily_uses") >= 10 })
	register("daily_50", "📅", 50, func(s map[string]any) bool { return num(s, "daily_uses") >= 50 })
	register("daily_100", "📅", 100, func(s map[string]any) bool { return num(s, "daily_uses") >= 100 })
	register("daily_365", "📅", 500, func(s map[string]any) bool { return num(s, "daily_uses") >= 365 })

	register("pet_collector_common", "🦋", 50, func(s map[string]any) bool { return num(s, "collected_common_pets") >= 8 })
	register("pet_collector_rare", "🦁", 150, func(s map[string]any) bool { return num(s, "collected_rare_pets") >= 6 })
	register("pet_collector_epic", "🦄", 300, func(s map[string]any) bool { return num(s, "collected_epic_pets") >= 4 })
	register("pet_collector_legendary", "🐉", 500, func(s map[string]any) bool { return num(s, "collected_legendary_pets") >= 1 })
	register("pet_collector_all", "🌍", 1000, func(s map[string]any) bool { return num(s, "collected_all_pets") >= 19 })

	register("rank_bronze", "🥉", 50, func(s map[string]any) bool { return anyRank(s, "Bronze") })
	register("rank_silver", "🥈", 100, func(s map[string]any) bool { return anyRank(s, "Argent") })
	register("rank_gold", "🥇", 500, func(s map[string]any) bool { return anyRank(s, "Or") })
	register("rank_diamond", "💎", 1000, func(s map[string]any) bool { return anyRank(s, "Diamant") })
	register("rank_top5", "🌟", 5000, func(s map[string]any) bool { return anyRank(s, "Top 5") })

	register("community_initiate", "🧱", 10, func(s map[string]any) bool { return num(s, "community_money") >= 10000 || num(s, "community_items") >= 200 })
	register("community_supporter", "🏛️", 50, func(s map[string]any) bool { return num(s, "community_money") >= 500000 || num(s, "community_items") >= 5000 })
	register("community_pillar", "🏛️", 150, func(s map[string]any) bool { return num(s, "community_money") >= 5000000 || num(s, "community_items") >= 50000 })

	register("boss_league_1", "⚔️", 20, func(s map[string]any) bool { return num(s, "boss_league_stage") >= 1 })
	register("boss_league_2", "🏹", 50, func(s map[string]any) bool { return num(s, "boss_league_stage") >= 2 })
	register("boss_league_3", "🛡️", 100, func(s map[string]any) bool { return num(s, "boss_league_stage") >= 3 })
	register("boss_league_4", "⚡", 200, func(s map[string]any) bool { return num(s, "boss_league_stage") >= 4 })
	register("boss_league_5", "🏆", 500,
		func(s map[string]any) bool { return num(s, "boss_league_stage") >= 5 })

	// --- LORE COLLECTION ---
	register("lore_5", "📖", 20,
		func(s map[string]any) bool { return num(s, "lore_count") >= 5 })
	register("lore_10", "📚", 40,
		func(s map[string]any) bool { return num(s, "lore_count") >= 10 })
	register("lore_25", "📜", 80,
		func(s map[string]any) bool { return num(s, "lore_count") >= 25 })
	register("lore_all", "👁️", 200,
		func(s map[string]any) bool { return num(s, "lore_count") >= 48 })

	_ = has
}

// registerHidden registers an achievement that is never auto-unlocked by
// CheckAndUnlock (its check always fails). Other systems grant it explicitly —
// the journal service inserts it when a player masters every path.
func init() {
	registerHidden("journal_mastery", "🏅", 10000)
}

func registerHidden(id, emoji string, glory int) {
	registry[id] = &Achievement{
		ID: id, Emoji: emoji, Glory: glory, Hidden: true,
		Check: func(map[string]any) bool { return false },
	}
}
func anyRank(s map[string]any, label string) bool {
	ranks, ok := s["pet_ranks"].([]string)
	if !ok {
		return false
	}
	for _, r := range ranks {
		if contains(r, label) {
			return true
		}
	}
	return false
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

// All returns every registered achievement.
func All() []*Achievement {
	out := make([]*Achievement, 0, len(registry))
	for _, a := range registry {
		out = append(out, a)
	}
	return out
}

// Get returns a single achievement by ID.
func Get(id string) (*Achievement, bool) {
	a, ok := registry[id]
	return a, ok
}

// localPetRarity returns the rarity of a pet type without importing the pets package.
func localPetRarity(petType string) string {
	m := map[string]string{
		"Escargot": "common", "Souris": "common", "Cochon": "common", "Grenouille": "common",
		"Taupe": "common", "Pélican": "common", "Mouton": "common", "Abeille": "common",
		"Hamster": "common", "Fourmi": "common", "Hérisson": "common", "Canard": "common",
		"Chouette": "common", "Paresseux": "common", "Putois": "common", "Bison": "common",
		"Chien": "rare", "Chat": "rare", "Cheval": "rare", "Renard": "rare",
		"Singe": "rare", "Ours": "rare", "Gorille": "rare", "Scorpion": "rare",
		"Ours polaire": "rare",
		"Chameau": "epic", "Panda": "epic", "Tigre": "epic", "Pieuvre": "epic",
		"Kangourou": "epic", "Iguane": "epic", "Aigle": "epic", "Rhino": "epic",
		"Crocodile": "epic", "Dauphin": "epic", "Léopard": "epic", "Lion": "epic",
		"Dragon": "legendary", "Tyrannosaure": "legendary", "Diplodocus": "legendary",
		"Mamouth": "legendary", "Mégalodon": "legendary", "Kraken": "legendary",
		"Licorne": "legendary", "Phoenix": "legendary", "Cerbère": "legendary",
		"Fenrir": "legendary", "Ratatosk": "legendary", "Nidhögg": "legendary", "Bedawang": "legendary",
	}
	if r, ok := m[petType]; ok {
		return r
	}
	return "common"
}

// BuildStats gathers the player's stats used for achievement evaluation.
// Derived fields (pet collection, community, ranks) default to zero and are
// filled in by their respective systems as they are ported.
func BuildStats(db *gorm.DB, userID int64) (map[string]any, error) {
	stats := map[string]any{
		"balance":              0,
		"boss_league_stage":    0,
		"lore_count":           0,
		"max_pet_level":        0,
		"pet_ranks":            []string{},
		"collected_common_pets": 0,
		"collected_rare_pets":   0,
		"collected_epic_pets":   0,
		"collected_legendary_pets": 0,
		"collected_all_pets":    0,
		"community_money":       0,
		"community_items":       0,
	}
	var u model.User
	if res := db.Where("user_id = ?", userID).First(&u); res.Error == nil {
		stats["balance"] = u.Balance
		stats["boss_league_stage"] = u.BossLeagueStage
	}
	var us model.UserStat
	if res := db.Where("user_id = ?", userID).First(&us); res.Error == nil {
		stats["pvp_wins"] = us.PvpWins
		stats["pvp_losses"] = us.PvpLosses
		stats["pve_wins"] = us.PveWins
		stats["items_mined"] = us.ItemsMined
		stats["items_fished"] = us.ItemsFished
		stats["items_farmed"] = us.ItemsFarmed
		stats["money_earned"] = us.MoneyEarned
		stats["pets_fed"] = us.PetsFed
		stats["coinflip_lost"] = us.CoinflipLost
		stats["coinflip_won"] = us.CoinflipWon
		stats["casino_lost"] = us.CasinoLost
		stats["casino_won"] = us.CasinoWon
		stats["slots_won"] = us.SlotsWon
		stats["slots_lost"] = us.SlotsLost
		stats["blackjack_won"] = us.BlackjackWon
		stats["blackjack_lost"] = us.BlackjackLost
		stats["roulette_won"] = us.RouletteWon
		stats["roulette_lost"] = us.RouletteLost
		stats["lotto_participations"] = us.LottoParticipations
		stats["lotto_won"] = us.LottoWon
		stats["bets_won"] = us.BetsWon
		stats["bets_lost"] = us.BetsLost
		stats["wagers_won"] = us.WagersWon
		stats["wagers_lost"] = us.WagersLost
		stats["casino_spent"] = us.CasinoSpent
		stats["slots_spent"] = us.SlotsSpent
		stats["slots_money_won"] = us.SlotsMoneyWon
		stats["slots_money_lost"] = us.SlotsMoneyLost
		stats["coinflip_spent"] = us.CoinflipSpent
		stats["coinflip_money_won"] = us.CoinflipMoneyWon
		stats["coinflip_money_lost"] = us.CoinflipMoneyLost
		stats["blackjack_spent"] = us.BlackjackSpent
		stats["blackjack_money_won"] = us.BlackjackMoneyWon
		stats["blackjack_money_lost"] = us.BlackjackMoneyLost
		stats["roulette_spent"] = us.RouletteSpent
		stats["roulette_money_won"] = us.RouletteMoneyWon
		stats["roulette_money_lost"] = us.RouletteMoneyLost
		stats["daily_uses"] = us.DailyUses
	}

	// Pet collection stats (computed from PetTypes rarity map)
	var userPets []model.UserPet
	db.Where("user_id = ?", userID).Find(&userPets)
	collectedTypes := make(map[string]bool)
	collectedRarity := map[string]int{"common": 0, "rare": 0, "epic": 0, "legendary": 0}
	maxPetLvl := 0
	for _, p := range userPets {
		collectedTypes[p.PetType] = true
		if p.Level > maxPetLvl {
			maxPetLvl = p.Level
		}
		r := localPetRarity(p.PetType)
		collectedRarity[r]++
	}
	stats["max_pet_level"] = maxPetLvl
	stats["collected_common_pets"] = collectedRarity["common"]
	stats["collected_rare_pets"] = collectedRarity["rare"]
	stats["collected_epic_pets"] = collectedRarity["epic"]
	stats["collected_legendary_pets"] = collectedRarity["legendary"]
	stats["collected_all_pets"] = len(collectedTypes)

	var loreCount int64
	db.Model(&model.UserLoreEntry{}).Where("user_id = ?", userID).Count(&loreCount)
	stats["lore_count"] = int(loreCount)

	// Hunt zone unlocks (progressive zones).
	var unlocks []model.UserHuntUnlock
	db.Where("user_id = ?", userID).Find(&unlocks)
	for _, u := range unlocks {
		stats["hunt_unlocked_"+u.ZoneKey] = 1
	}

	// Hunt zone boss kills.
	var zoneStats []model.UserHuntZoneStat
	db.Where("user_id = ?", userID).Find(&zoneStats)
	bossTotal := 0
	for _, zs := range zoneStats {
		stats["hunt_boss_"+zs.ZoneKey] = zs.BossKills
		bossTotal += zs.BossKills
	}
	stats["hunt_boss_total"] = bossTotal

	return stats, nil
}

// IncrementStat adds amount to the named user_stats column, ensuring the row
// exists first so the first increment is counted correctly.
func IncrementStat(db *gorm.DB, userID int64, stat string, amount int) error {
	if err := db.Where("user_id = ?", userID).FirstOrCreate(&model.UserStat{UserID: userID}).Error; err != nil {
		return err
	}
	return db.Model(&model.UserStat{}).
		Where("user_id = ?", userID).
		UpdateColumn(stat, gorm.Expr(stat+" + ?", amount)).Error
}

// CheckAndUnlock evaluates achievements for a user and persists any newly
// satisfied ones. It returns the list of newly unlocked achievements.
func CheckAndUnlock(db *gorm.DB, userID int64) ([]*Achievement, error) {
	stats, err := BuildStats(db, userID)
	if err != nil {
		return nil, err
	}
	var unlocked []model.UserAchievement
	if err := db.Where("user_id = ?", userID).Find(&unlocked).Error; err != nil {
		return nil, err
	}
	unlockedSet := make(map[string]bool, len(unlocked))
	for _, u := range unlocked {
		unlockedSet[u.AchievementID] = true
	}
	var fresh []*Achievement
	for _, a := range registry {
		if unlockedSet[a.ID] {
			continue
		}
		if a.Check(stats) {
			if err := db.Create(&model.UserAchievement{UserID: userID, AchievementID: a.ID}).Error; err != nil {
				return nil, err
			}
			fresh = append(fresh, a)
		}
	}
	return fresh, nil
}
