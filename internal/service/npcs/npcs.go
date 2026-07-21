package npcs

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type NPCData struct {
	ID          string
	Name        string
	Emoji       string
	Color       int
	Description string
	Role        string
	Advice      string
	Chat        string
	Hint        string
	Greetings   []string
}

var NPCs = map[string]*NPCData{
	"elara": {
		ID: "elara", Name: "Elara", Emoji: "🌿", Color: 0x2ecc71,
		Description: "Gardienne des jardins du village et des écuries.",
		Role:        "Elle vous aide avec le farming et les pets. Une haute affinité réduit le temps de pousse des cultures.",
		Advice:      "Associez vos cultures à vos terres ! Les serres sont parfaites pour les graines tropicales.",
		Chat:        "Les pousses sont si vertes aujourd'hui. N'oubliez pas de nourrir vos petits compagnons !",
		Hint:        "(Aime les œufs mystère, fruits étoiles, pommes dorées, graines et baies)",
		Greetings:   []string{"Bonjour. Prends soin de la terre et des animaux.", "Ravie de te voir. Mes plantes poussent à merveille.", "Bonjour cher ami ! La nature elle-même chante en ta présence."},
	},
	"thorek": {
		ID: "thorek", Name: "Thorek", Emoji: "⛏️", Color: 0xe67e22,
		Description: "Forgeron et mineur du village.",
		Role:        "Il raffine les métaux et aide les mineurs. Une haute affinité réduit le risque d'effondrement.",
		Advice:      "Miner est une question de gestion de risque. Ne creusez pas trop profond si votre sac est plein !",
		Chat:        "Clang ! Clang ! Je travaille sur un nouveau prototype de pioche. Le travail acharné forge le caractère.",
		Hint:        "(Aime les pépites d'or, diamants, platine et minerais)",
		Greetings:   []string{"Qu'est-ce que tu veux ? Si t'as pas de pioche, tu perds mon temps.", "Ah, te voilà ! Trouvé du bon minerai récemment ?", "Bonjour mon ami ! Ma forge est toujours ouverte pour un travailleur comme toi."},
	},
	"irian": {
		ID: "irian", Name: "Irian", Emoji: "🎣", Color: 0x3498db,
		Description: "Pêcheur vétéran et gardien des quais.",
		Role:        "Il aide les pêcheurs à attraper des créatures rares. Une haute affinité étend la fenêtre de réaction.",
		Advice:      "Soyez rapide ! La pêche en océan a une fenêtre très serrée, mais c'est là que vivent les bêtes légendaires.",
		Chat:        "Je regarde l'horizon... On dit qu'un Kraken géant rôde dans les eaux profondes quand le ciel s'assombrit.",
		Hint:        "(Aime les tentacules de kraken, baleines, requins et poissons)",
		Greetings:   []string{"Chut... Tu vas effrayer les poissons.", "Hé marin. Senti la brise de mer récemment ?", "Ah, mon capitaine ! Les marées sont favorables aujourd'hui."},
	},
	"gamblebot": {
		ID: "gamblebot", Name: "GambleBot", Emoji: "🤖", Color: 0xf1c40f,
		Description: "Un croupier robot de pointe.",
		Role:        "Il gère le Casino du village. Améliorer votre réputation débloque jusqu'à 20% de réduction.",
		Advice:      "Toujours doubler sur un 11 au Blackjack, mais ne misez jamais plus que ce que vous pouvez perdre !",
		Chat:        "Bip boop ! Calcul de probabilité de gain... 99% de chance que vous devriez jouer aux machines à sous.",
		Hint:        "(Aime les objets brillants, pièces truquées et tickets VIP)",
		Greetings:   []string{"Bonjour humain. As-tu tenté ta chance aujourd'hui ?", "Content de te voir. On dirait que tu as misé sagement.", "Hé partenaire ! Qui allons-nous plumer aujourd'hui ?"},
	},
}

type DailyRepCap struct {
	Flat      int
	PerLevel  int
}

func GetDailyRepCap() DailyRepCap {
	return DailyRepCap{Flat: 500, PerLevel: 0}
}

type NPCInfo struct {
	ID          string
	Name        string
	Emoji       string
	Color       int
	Level       int
	Reputation  int
	NextLevel   int
	RankName    string
}

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) GetNPCData(id string) *NPCData {
	return NPCs[id]
}

func (s *Service) GetAllNPCMeta() []*NPCData {
	out := make([]*NPCData, 0, len(NPCs))
	for _, n := range NPCs {
		out = append(out, n)
	}
	return out
}

func (s *Service) GetReputation(userID int64, npcID string) (*model.UserNPCReputation, error) {
	var r model.UserNPCReputation
	err := s.store.DB.Where("user_id = ? AND npc_id = ?", userID, npcID).FirstOrCreate(&r, model.UserNPCReputation{
		UserID: userID, NPCID: npcID, Reputation: 0, Level: 1,
	}).Error
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Service) GetAllReputations(userID int64) ([]model.UserNPCReputation, error) {
	var reps []model.UserNPCReputation
	if err := s.store.DB.Where("user_id = ?", userID).Find(&reps).Error; err != nil {
		return nil, err
	}
	return reps, nil
}

func (s *Service) AddReputation(userID int64, npcID string, points int) (int, error) {
	cap := GetDailyRepCap()
	today := time.Now().Format("2006-01-02")
	var daily model.UserNPCDailyRep
	err := s.store.DB.Where("user_id = ? AND npc_id = ? AND date_str = ?", userID, npcID, today).First(&daily).Error
	if err != nil {
		daily = model.UserNPCDailyRep{UserID: userID, NPCID: npcID, DateStr: today, Amount: 0}
	}
	remaining := cap.Flat - daily.Amount
	if remaining <= 0 {
		return 0, nil
	}
	if points > remaining {
		points = remaining
	}
	if err := s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "npc_id"}, {Name: "date_str"}},
		DoUpdates: clause.Assignments(map[string]any{"amount": gorm.Expr("amount + ?", points)}),
	}).Create(&model.UserNPCDailyRep{
		UserID: userID, NPCID: npcID, DateStr: today, Amount: points,
	}).Error; err != nil {
		return 0, err
	}
	var rep model.UserNPCReputation
	if err := s.store.DB.Where("user_id = ? AND npc_id = ?", userID, npcID).First(&rep).Error; err != nil {
		return 0, err
	}
	newRep := rep.Reputation + points
	nextLevel := rep.Level
	for newRep >= 100*nextLevel {
		newRep -= 100 * nextLevel
		nextLevel++
	}
	if err := s.store.DB.Model(&model.UserNPCReputation{}).
		Where("user_id = ? AND npc_id = ?", userID, npcID).
		Updates(map[string]any{"reputation": newRep, "level": nextLevel}).Error; err != nil {
		return 0, err
	}
	return points, nil
}

func (s *Service) RankUp(userID int64, npcID string) error {
	rep, err := s.GetReputation(userID, npcID)
	if err != nil {
		return err
	}
	if rep.Reputation < 100*rep.Level {
		return fmt.Errorf("not enough reputation")
	}
	newLevel := rep.Level + 1
	return s.store.DB.Model(&model.UserNPCReputation{}).
		Where("user_id = ? AND npc_id = ?", userID, npcID).
		Updates(map[string]any{"level": newLevel, "reputation": 0}).Error
}

func RankName(level int) string {
	names := []string{"Inconnu", "Connaissance", "Associé", "Ami", "Partenaire"}
	idx := level - 1
	if idx >= len(names) {
		idx = len(names) - 1
	}
	if idx < 0 {
		idx = 0
	}
	return names[idx]
}

type NPCBonus struct {
	ShopDiscount          float64
	GamblePayout          float64
	XPBoost               float64
	MiningRiskReduction   int
	FarmingSpeedBoost     float64
	FishingTimeBonus      float64
}

func (s *Service) GetBonuses(userID int64) *NPCBonus {
	b := &NPCBonus{}
	for _, npc := range NPCs {
		rep, _ := s.GetReputation(userID, npc.ID)
		lvl := rep.Level
		switch npc.ID {
		case "gamblebot":
			b.ShopDiscount = maxFloat(b.ShopDiscount, float64(lvl)*2.0)
		case "thorek":
			b.MiningRiskReduction = maxInt(b.MiningRiskReduction, lvl*2)
		case "elara":
			b.FarmingSpeedBoost = maxFloat(b.FarmingSpeedBoost, float64(lvl)*2.0)
		case "irian":
			b.FishingTimeBonus = maxFloat(b.FishingTimeBonus, float64(lvl)*0.5)
		}
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
