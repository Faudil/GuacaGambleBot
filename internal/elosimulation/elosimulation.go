package elosimulation

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/service/pets"
	"guacagamblebot/internal/store"
)

func Run(st *store.Store) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		simulateRound(st)
	}
}

func RunWeeklyReset(st *store.Store, s *discordgo.Session) {
	for {
		time.Sleep(time.Until(nextSundayMidnight()))
		performWeeklyResetForAll(st, s)
	}
}

func nextSundayMidnight() time.Time {
	now := time.Now().UTC()
	daysUntilSunday := (7 - int(now.Weekday())) % 7
	if daysUntilSunday == 0 {
		daysUntilSunday = 7
	}
	next := now.AddDate(0, 0, daysUntilSunday)
	return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, time.UTC)
}

func currentWeekID() string {
	y, week := time.Now().ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, week)
}

func performWeeklyResetForAll(st *store.Store, s *discordgo.Session) {
	var servers []model.ServerSetting
	st.DB.Find(&servers)
	for _, ss := range servers {
		performWeeklyReset(st, s, ss)
	}
}

func performWeeklyReset(st *store.Store, s *discordgo.Session, ss model.ServerSetting) {
	weekID := currentWeekID()
	var ranks []model.WeeklyRank
	st.DB.Where("server_id = ? AND week_id = ?", ss.ServerID, weekID).
		Order("score desc").Find(&ranks)

	topFive := make([]model.WeeklyRank, 0, 5)
	for i, r := range ranks {
		tier := tierForScore(r.Score, i+1)
		if tier == nil {
			continue
		}
		if tier.Coins > 0 {
			_, _ = st.UpdateBalance(r.UserID, tier.Coins)
		}
		if tier.Crowns > 0 {
			_, _ = st.AdjustColumn(r.UserID, "crowns", tier.Crowns)
		}
		if tier.ItemID != "" {
			_ = st.DB.Exec(
				`INSERT INTO inventory (user_id, item_id, quantity) VALUES (?, ?, 1)
				 ON CONFLICT(user_id, item_id) DO UPDATE SET quantity = quantity + 1`,
				r.UserID, tier.ItemID,
			)
		}
		if i < 5 {
			topFive = append(topFive, r)
		}
	}

	modID, boostedStr, nerfedStr := rollWeeklyModifier(st, ss.ServerID)

	if len(topFive) > 0 {
		sendWeeklyAnnouncement(s, ss, topFive, modID, boostedStr, nerfedStr)
	}
}

func sendWeeklyAnnouncement(s *discordgo.Session, ss model.ServerSetting, topFive []model.WeeklyRank, modID, boostedStr, nerfedStr string) {
	if ss.AnnouncementChannelID == 0 {
		return
	}

	lang := ss.Language
	if lang == "" {
		lang = "fr"
	}

	desc := ""
	medals := []string{"👑", "🥈", "🥉", "4️⃣", "5️⃣"}
	tierNames := []string{"Grandmaster", "Master", "Diamond", "Platinum", "Gold"}
	rewardLines := []string{
		"25,000 🪙 + 50 👑 + Personality Mirror",
		"10,000 🪙 + 20 👑 + Skill Scroll",
		"5,000 🪙 + 10 👑 + Skill Scroll",
		"2,500 🪙 + 5 👑 + Mystery Egg",
		"1,000 🪙 + 2 👑 + Gold Nugget",
	}

	for i, r := range topFive {
		desc += fmt.Sprintf("%s <@%d> — **%d pts**\n", medals[i], r.UserID, r.Score)
		desc += fmt.Sprintf("   🏆 %s — %s\n", tierNames[i], rewardLines[i])
	}

	modName := modID
	switch modID {
	case "burning_sun":
		modName = "☀️ Burning Sun"
	case "heavy_rain":
		modName = "🌧️ Heavy Rain"
	case "starlight":
		modName = "✨ Starlight"
	case "iron_will":
		modName = "🛡️ Iron Will"
	case "blood_moon":
		modName = "🌕 Blood Moon"
	case "thunderstorm":
		modName = "⛈️ Thunderstorm"
	case "shadow_realm":
		modName = "🌑 Shadow Realm"
	case "rampage":
		modName = "💢 Rampage"
	case "frost_aura":
		modName = "❄️ Frost Aura"
	case "chaos":
		modName = "🌀 Chaos"
	}

	boostedEmojis := ""
	nerfedEmojis := ""
	for _, s := range pets.SplitModStats(boostedStr) {
		if def := pets.GetArtifactStat(s); def != nil {
			boostedEmojis += def.Emoji + " "
		}
	}
	for _, s := range pets.SplitModStats(nerfedStr) {
		if def := pets.GetArtifactStat(s); def != nil {
			nerfedEmojis += def.Emoji + " "
		}
	}

	modLine := fmt.Sprintf("\n**New Weekly Modifier:** %s\nBoosted: %s | Nerfed: %s", modName, boostedEmojis, nerfedEmojis)

	embed := &discordgo.MessageEmbed{
		Title:       i18n.T("weekly.announcement_title", lang),
		Description: desc + modLine,
		Color:       0xf1c40f,
		Footer:      &discordgo.MessageEmbedFooter{Text: i18n.T("weekly.announcement_footer", lang)},
	}

	_, err := s.ChannelMessageSendEmbed(strconv.FormatInt(ss.AnnouncementChannelID, 10), embed)
	if err != nil {
		log.Printf("elosimulation: failed to send weekly announcement: %v", err)
	}
}

func rollWeeklyModifier(st *store.Store, serverID int64) (modID, boosted, nerfed string) {
	weekID := currentWeekID()
	mods := []string{"burning_sun", "heavy_rain", "starlight", "iron_will", "blood_moon",
		"thunderstorm", "shadow_realm", "rampage", "frost_aura", "chaos"}
	modID = mods[rand.Intn(len(mods))]

	switch modID {
	case "burning_sun":
		boosted, nerfed = "impact,rejuvenation", "resilience,warding"
	case "heavy_rain":
		boosted, nerfed = "piercing,precision", "fortune,haste"
	case "starlight":
		boosted, nerfed = "fortune,might", "resilience,vampirism"
	case "iron_will":
		boosted, nerfed = "resilience,warding", "impact,might"
	case "blood_moon":
		boosted, nerfed = "vampirism,impact", "rejuvenation,resilience"
	case "thunderstorm":
		boosted, nerfed = "haste,piercing", "resilience,warding"
	case "shadow_realm":
		boosted, nerfed = "rejuvenation,precision", "fortune,might"
	case "rampage":
		boosted, nerfed = "might,impact", "warding,rejuvenation"
	case "frost_aura":
		boosted, nerfed = "resilience,warding", "vampirism,haste"
	case "chaos":
		boosted, nerfed = "impact,fortune", "resilience,warding"
	}

	_ = st.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "server_id"}, {Name: "week_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"modifier": modID, "boosted": boosted, "nerfed": nerfed, "created_at": time.Now().UTC(),
		}),
	}).Create(&model.WeeklyModifier{
		ServerID: serverID, WeekID: weekID, Modifier: modID,
		Boosted: boosted, Nerfed: nerfed, CreatedAt: time.Now().UTC(),
	}).Error
	return
}

type tierInfo struct {
	MinScore int
	Coins    int
	Crowns   int
	ItemID   string
}

var tierList = []tierInfo{
	{MinScore: 2000, Coins: 5000, Crowns: 10, ItemID: "skill_scroll"},
	{MinScore: 1000, Coins: 2500, Crowns: 5, ItemID: "volcano_egg"},
	{MinScore: 500, Coins: 1000, Crowns: 2, ItemID: "gold_nugget"},
	{MinScore: 250, Coins: 500, Crowns: 1, ItemID: "iron_ore"},
	{MinScore: 100, Coins: 200, Crowns: 0, ItemID: ""},
}

func tierForScore(score int, rank int) *tierInfo {
	if rank == 1 {
		return &tierInfo{Coins: 25000, Crowns: 50, ItemID: "personality_mirror"}
	}
	if rank <= 5 {
		return &tierInfo{Coins: 10000, Crowns: 20, ItemID: "skill_scroll"}
	}
	for _, t := range tierList {
		if score >= t.MinScore {
			return &t
		}
	}
	return nil
}

func simulateRound(st *store.Store) {
	p1, p2, err := st.GetRandomActivePetPair(5, 500)
	if err != nil {
		return
	}
	if p2 == nil {
		return
	}

	result := battle.Simulate(toBattlePet(p1), toBattlePet(p2), "")

	var score float64
	if result.WinnerID == p1.ID {
		score = 1.0
	} else if result.WinnerID == p2.ID {
		score = 0.0
	} else {
		score = 0.5
	}

	K := 32.0
	e1 := 1.0 / (1.0 + math.Pow(10, float64(p2.Elo-p1.Elo)/400))
	e2 := 1.0 / (1.0 + math.Pow(10, float64(p1.Elo-p2.Elo)/400))
	d1 := int(K * (score - e1))
	d2 := int(K * ((1.0 - score) - e2))

	newElo1 := p1.Elo + d1
	newElo2 := p2.Elo + d2
	if newElo1 < 0 {
		newElo1 = 0
	}
	if newElo2 < 0 {
		newElo2 = 0
	}

	txErr := st.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.UserPet{}).
			Where("id = ?", p1.ID).Update("elo", newElo1).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.UserPet{}).
			Where("id = ?", p2.ID).Update("elo", newElo2).Error; err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		log.Printf("elosimulation: failed to persist Elo update: %v", txErr)
	}

	if score == 1.0 {
		addWeeklyScore(st, p1.UserID, p1.ID, p2.UserID, p2.ID, 5)
	} else if score == 0.0 {
		addWeeklyScore(st, p2.UserID, p2.ID, p1.UserID, p1.ID, 5)
	}
}

func addWeeklyScore(st *store.Store, winnerUserID, winnerPetID, loserUserID, loserPetID int64, scoreDelta int) {
	weekID := currentWeekID()
	_ = st.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "server_id"}, {Name: "week_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"score":  gorm.Expr("score + ?", scoreDelta),
			"wins":   gorm.Expr("wins + 1"),
			"losses": gorm.Expr("losses + 0"),
		}),
	}).Create(&model.WeeklyRank{
		UserID: winnerUserID, ServerID: 0, WeekID: weekID,
		Score: scoreDelta, Wins: 1, Losses: 0,
	}).Error
	_ = st.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "server_id"}, {Name: "week_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"score":  gorm.Expr("score + 2"),
			"wins":   gorm.Expr("wins + 0"),
			"losses": gorm.Expr("losses + 1"),
		}),
	}).Create(&model.WeeklyRank{
		UserID: loserUserID, ServerID: 0, WeekID: weekID,
		Score: 2, Wins: 0, Losses: 1,
	}).Error
}

func toBattlePet(p *model.UserPet) *battle.BattlePet {
	bp := &battle.BattlePet{
		ID:       p.ID,
		Nickname: p.Nickname,
		Level:    p.Level,
		MaxHP:    p.MaxHP,
		HP:       p.HP,
		Atk:      p.Atk,
		Defense:  p.Defense,
		Speed:    p.Speed,
		DGE:      p.DGE,
		ACC:      p.ACC,
		CritC:    p.CritC,
		CritD:    p.CritD,
		SpcC:     p.SpcC,
	}
	return bp
}
