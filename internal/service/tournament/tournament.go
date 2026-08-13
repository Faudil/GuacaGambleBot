package tournament

import (
	"math/rand"

	"guacagamblebot/internal/battle"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	petsvc "guacagamblebot/internal/service/pets"
	"guacagamblebot/internal/store"
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) GetBalance(userID int64) (int, error) {
	return s.store.GetBalance(userID)
}

func (s *Service) UpdateBalance(userID int64, delta int) (int, error) {
	return s.store.UpdateBalance(userID, delta)
}

func (s *Service) GetActivePet(userID int64) (*model.UserPet, error) {
	var pet model.UserPet
	err := s.store.DB.Where("user_id = ? AND is_active = ?", userID, true).First(&pet).Error
	if err != nil {
		return nil, err
	}
	return &pet, nil
}

type TournamentPlayer struct {
	UserID int64
	Pet    *model.UserPet
}

type TournamentState struct {
	ServerID   int64
	CreatorID  int64
	Fee        int
	Players    []TournamentPlayer
	Started    bool
}

type MatchResult struct {
	WinnerID int64
	LoserID  int64
	Draw     bool
	Log      []string
}

func (s *Service) SimulateMatch(p1, p2 *TournamentPlayer) *MatchResult {
	p1.Pet.HP = p1.Pet.MaxHP
	p2.Pet.HP = p2.Pet.MaxHP

	bp1 := s.toBattlePet(p1.Pet)
	bp2 := s.toBattlePet(p2.Pet)

	result := battle.Simulate(bp1, bp2)

	p1.Pet.HP = bp1.HP
	p2.Pet.HP = bp2.HP

	if result.WinnerID == p1.Pet.ID {
		return &MatchResult{WinnerID: p1.UserID, LoserID: p2.UserID, Log: result.Log}
	} else if result.WinnerID == p2.Pet.ID {
		return &MatchResult{WinnerID: p2.UserID, LoserID: p1.UserID, Log: result.Log}
	}
	return &MatchResult{WinnerID: 0, LoserID: 0, Draw: true, Log: result.Log}
}

func (s *Service) toBattlePet(pet *model.UserPet) *battle.BattlePet {
	emoji := "🐾"
	if pt := petsvc.PetTypes[pet.PetType]; pt != nil {
		emoji = pt.Emoji
	}
	var skills []model.UserPetSkill
	s.store.DB.Where("pet_id = ?", pet.ID).Find(&skills)
	skillIDs := make([]string, 0, len(skills))
	for _, sk := range skills {
		skillIDs = append(skillIDs, sk.SkillID)
	}
	return &battle.BattlePet{
		ID: pet.ID, Nickname: pet.Nickname, Emoji: emoji, PetType: pet.PetType,
		Level: pet.Level, HP: pet.MaxHP, MaxHP: pet.MaxHP,
		Atk: pet.Atk, Defense: pet.Defense, Speed: pet.Speed,
		DGE: pet.DGE, ACC: pet.ACC, CritC: pet.CritC, CritD: pet.CritD, SpcC: pet.SpcC,
		Skills: skillIDs,
	}
}

func ShufflePlayers(players []TournamentPlayer) {
	rand.Shuffle(len(players), func(i, j int) {
		players[i], players[j] = players[j], players[i]
	})
}
