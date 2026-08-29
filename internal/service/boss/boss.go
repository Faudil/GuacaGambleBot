package boss

import (
	"gorm.io/gorm"

	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
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
	XP          int
	Achievement string
	// Skills are battle skill IDs (see service/pets/skills.go) applied to the
	// boss at battle start, e.g. "phoenix_rebirth".
	Skills []string
	// Image is the asset file (e.g. "bosses/vezir.png") shown on the boss
	// fight embeds. Empty means no picture.
	Image string
}

var BossLeague = []BossStage{
	{
		Stage: 1, NameFR: "Vezir, le Moissonneur d'Étincelles", NameEN: "Vezir, the Spark Collector",
		Species: "Souris", Level: 20, HP: 250, Atk: 35, Defense: 15, Speed: 25,
		DGE: 10, ACC: 10, CritC: 8, CritD: 1.5, SpcC: 0,
		DescFR:      "Un éclaireur vif comme l'éclair. Ses attaques surviennent sans prévenir. Ne sous-estime pas sa petite taille.",
		DescEN:      "A scout quick as lightning. His attacks come without warning. Don't underestimate his size.",
		RewardMoney: 200, RewardItems: map[string]int{"coffee": 1},
		XP: 50, Achievement: "boss_league_1", Image: "bosses/vezir.png",
	},
	{
		Stage: 2, NameFR: "Tal'Rok, le Gardien de Pierre", NameEN: "Tal'Rok, the Stone Sentinel",
		Species: "Ours", Level: 25, HP: 350, Atk: 45, Defense: 30, Speed: 15,
		DGE: 5, ACC: 15, CritC: 10, CritD: 2.0, SpcC: 0,
		DescFR:      "Une muraille vivante. Perce sa défense ou il t'épuisera. Pas de pitié.",
		DescEN:      "A living wall. Break his defense or he'll wear you down. No mercy.",
		RewardMoney: 500, RewardItems: map[string]int{"vip_ticket": 1},
		XP: 100, Achievement: "boss_league_2", Image: "bosses/talrok.png",
	},
	{
		Stage: 3, NameFR: "Kael, le Foudroyeur", NameEN: "Kael, the Storm Striker",
		Species: "Aigle", Level: 30, HP: 420, Atk: 50, Defense: 20, Speed: 40,
		DGE: 22, ACC: 25, CritC: 20, CritD: 2.0, SpcC: 5,
		DescFR:      "Il frappe comme la tempête. Un coup critique et c'est fini. Vitesse et précision — riposte ou meurs.",
		DescEN:      "He strikes like the storm. One crit and it's over. Speed and precision — counter or fall.",
		RewardMoney: 1000, RewardItems: map[string]int{"fortune_cookie": 2},
		XP: 150, Achievement: "boss_league_3", Image: "bosses/kael.png",
	},
	{
		Stage: 4, NameFR: "Vorgath, l'Abyssal", NameEN: "Vorgath, the Abyssal",
		Species: "Kraken", Level: 38, HP: 550, Atk: 65, Defense: 35, Speed: 25,
		DGE: 20, ACC: 10, CritC: 15, CritD: 1.5, SpcC: 10,
		DescFR:      "Des profondeurs il t'observe. Son poison te ronge à chaque tour. Un combat d'endurance.",
		DescEN:      "From the deep he watches. His poison eats at you each turn. A battle of endurance.",
		RewardMoney: 2500, RewardItems: map[string]int{"forget_potion": 1},
		XP: 200, Achievement: "boss_league_4", Image: "bosses/vorgath.png",
	},
	{
		Stage: 5, NameFR: "Solaris, le Phénix Éternel", NameEN: "Solaris, the Eternal Phoenix",
		Species: "Phoenix", Level: 45, HP: 750, Atk: 85, Defense: 45, Speed: 50,
		DGE: 25, ACC: 15, CritC: 15, CritD: 1.5, SpcC: 15,
		DescFR:      "L'ultime gardien de la Ligue. Il renaît de ses cendres. Pour le vaincre, il faut tout donner.",
		DescEN:      "The League's final guardian. He rises from his ashes. To win, you must give everything.",
		RewardMoney: 5000, RewardItems: map[string]int{},
		XP: 250, Achievement: "boss_league_5", Image: "bosses/solaris.png",
		Skills: []string{"phoenix_rebirth"},
	},
	{
		Stage: 6, NameFR: "Le Gardien du Coffre", NameEN: "The Vault Guardian",
		Species: "Robot", Level: 15, HP: 150, Atk: 25, Defense: 15, Speed: 15,
		DGE: 10, ACC: 10, CritC: 8, CritD: 1.5, SpcC: 5,
		DescFR:      "Un mécha de combat ancestral émerge du Coffre. Ses plaques d'acier noir brillent d'une lueur bleue. Il ne reconnaît plus ami ou ennemi — seulement les ordres gravés dans son noyau.",
		DescEN:      "An ancient combat mech rises from the Vault. Its black steel plates glow with blue light. It no longer knows friend from foe — only the orders etched into its core.",
		RewardMoney: 0, RewardItems: map[string]int{},
		Achievement: "", Image: "bosses/vault_guardian.png",
	},
	{
		Stage: 7, NameFR: "Krag, le Champion de l'Arène", NameEN: "Krag, the Arena Champion",
		Species: "Lion", Level: 20, HP: 200, Atk: 30, Defense: 14, Speed: 22,
		DGE: 10, ACC: 12, CritC: 10, CritD: 1.5, SpcC: 0,
		DescFR:      "Le roi de l'arène. Aucun challenger n'a survécu à son rugissement. Son regard perçant jauge chaque adversaire — montre-lui de quoi ton familier est fait.",
		DescEN:      "The king of the arena. No challenger has survived his roar. His piercing gaze sizes up every opponent — show him what your pet is made of.",
		RewardMoney: 0, RewardItems: map[string]int{},
		XP: 0, Achievement: "", Image: "bosses/krag.png",
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
	charsvc.AddXP(s.store, userID, stage*50)
	return s.store.DB.Model(&model.User{}).
		Where("user_id = ?", userID).
		Update("boss_league_stage", stage).Error
}

func (s *Service) CreateBossPet(stage BossStage) *battle.BattlePet {
	pt := petEmoji(stage.Species)
	return &battle.BattlePet{
		ID: -1, Nickname: stage.NameFR, Emoji: pt, PetType: stage.Species,
		Level: stage.Level, HP: stage.HP, MaxHP: stage.HP,
		Atk: stage.Atk, Defense: stage.Defense, Speed: stage.Speed,
		DGE: stage.DGE, ACC: stage.ACC, CritC: stage.CritC, CritD: stage.CritD, SpcC: stage.SpcC,
		Skills: stage.Skills,
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
	"Souris": "🐀", "Ours": "🐻", "Aigle": "🦅", "Kraken": "🦑", "Phoenix": "🐦‍🔥", "Robot": "🤖", "Lion": "🦁",
}

func petEmoji(species string) string {
	if e, ok := typeEmojis[species]; ok {
		return e
	}
	return "🐾"
}
