package boss

import (
	"gorm.io/gorm"

	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) DB() *gorm.DB { return s.store.DB }

type BossStage struct {
	Stage       int
	NameFR      string
	NameEN      string
	Species     string
	Level       int
	HP          int
	Atk         int
	Defense     int
	Speed       int
	DGE         int
	ACC         int
	CritC       int
	CritD       float64
	SpcC        int
	DescFR      string
	DescEN      string
	RewardMoney int
	RewardItems map[string]int
	Achievement string
}

var BossLeague = []BossStage{
	{
		Stage: 1, NameFR: "Dresseur Novice", NameEN: "Rookie Collector",
		Species: "Souris", Level: 5, HP: 80, Atk: 15, Defense: 8, Speed: 15,
		DGE: 5, ACC: 5, CritC: 5, CritD: 1.5, SpcC: 0,
		DescFR: "Un débutant enthousiaste avec une Souris rapide. Un bon test de départ !",
		DescEN: "An enthusiastic beginner with a quick Mouse. A good starting test!",
		RewardMoney: 200, RewardItems: map[string]int{"café": 1, "œuf mystère": 1},
		Achievement: "boss_league_1",
	},
	{
		Stage: 2, NameFR: "Gardien de Pierre", NameEN: "Stone Sentinel",
		Species: "Ours", Level: 10, HP: 150, Atk: 25, Defense: 20, Speed: 8,
		DGE: 2, ACC: 10, CritC: 5, CritD: 2.0, SpcC: 0,
		DescFR: "Un Ours robuste doté d'une défense impressionnante. Vous devrez percer sa carapace.",
		DescEN: "A sturdy Bear with impressive defense. You'll need to break through its guard.",
		RewardMoney: 500, RewardItems: map[string]int{"ticket vip": 2, "terrain : potager": 1},
		Achievement: "boss_league_2",
	},
	{
		Stage: 3, NameFR: "Foudre Céleste", NameEN: "Storm Striker",
		Species: "Aigle", Level: 15, HP: 200, Atk: 35, Defense: 15, Speed: 35,
		DGE: 22, ACC: 25, CritC: 20, CritD: 2.0, SpcC: 0,
		DescFR: "Un Aigle féroce qui attaque à une vitesse fulgurante et inflige de lourds dégâts critiques.",
		DescEN: "A fierce Eagle attacking at lightning speed and inflicting heavy critical damage.",
		RewardMoney: 1000, RewardItems: map[string]int{"terrain : verger enchanté": 1, "fortune cookie": 2},
		Achievement: "boss_league_3",
	},
	{
		Stage: 4, NameFR: "Léviathan des Abysses", NameEN: "Abyssal Leviathan",
		Species: "Kraken", Level: 20, HP: 300, Atk: 40, Defense: 30, Speed: 22,
		DGE: 20, ACC: 10, CritC: 15, CritD: 1.5, SpcC: 5,
		DescFR: "Le Kraken mythique des profondeurs. Ses attaques de type POISON affaibliront votre familier sur la durée.",
		DescEN: "The mythical Kraken of the deep. Its POISON-type attacks will wear down your pet over time.",
		RewardMoney: 2500, RewardItems: map[string]int{"terrain : serre tropicale": 1, "potion d'oubli": 1},
		Achievement: "boss_league_4",
	},
	{
		Stage: 5, NameFR: "Le Phénix Éternel", NameEN: "The Eternal Phoenix",
		Species: "Phoenix", Level: 30, HP: 500, Atk: 60, Defense: 40, Speed: 40,
		DGE: 25, ACC: 15, CritC: 15, CritD: 1.5, SpcC: 10,
		DescFR: "L'ultime boss de la ligue. Le Phénix renaît de ses cendres avec des stats colossales et des attaques de FEU.",
		DescEN: "The final league boss. The Phoenix rises with colossal stats and FIRE attacks.",
		RewardMoney: 5000, RewardItems: map[string]int{"trophée de boss": 1},
		Achievement: "boss_league_5",
	},
}

func (s *Service) GetStage(userID int64) (int, error) {
	var u model.User
	if err := s.store.DB.Where("user_id = ?", userID).First(&u).Error; err != nil {
		return 0, err
	}
	return u.BossLeagueStage, nil
}

func (s *Service) SetStage(userID int64, stage int) error {
	return s.store.DB.Model(&model.User{}).
		Where("user_id = ?", userID).
		Update("boss_league_stage", stage).Error
}

func (s *Service) CreateBossPet(stage BossStage) *battle.BattlePet {
	pt := petEmoji(stage.Species)
	return &battle.BattlePet{
		ID: -1, Nickname: stage.NameFR, Emoji: pt,
		Level: stage.Level, HP: stage.HP, MaxHP: stage.HP,
		Atk: stage.Atk, Defense: stage.Defense, Speed: stage.Speed,
		DGE: stage.DGE, ACC: stage.ACC, CritC: stage.CritC, CritD: stage.CritD, SpcC: stage.SpcC,
	}
}

func (s *Service) GetBalance(userID int64) (int, error) {
	return s.store.GetBalance(userID)
}

func (s *Service) UpdateBalance(userID int64, delta int) (int, error) {
	return s.store.UpdateBalance(userID, delta)
}

func (s *Service) IncrementStat(userID int64, stat string, amount int) error {
	return s.store.DB.Where("user_id = ?", userID).
		FirstOrCreate(&model.UserStat{UserID: userID}).Error
}

var typeEmojis = map[string]string{
	"Souris": "🐀", "Ours": "🐻", "Aigle": "🦅", "Kraken": "🦑", "Phoenix": "🐦‍🔥",
}

func petEmoji(species string) string {
	if e, ok := typeEmojis[species]; ok {
		return e
	}
	return "🐾"
}
