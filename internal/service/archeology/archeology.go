package archeology

import (
	"errors"
	"math/rand"

	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

var ErrDigLimit = errors.New("dig daily limit reached")

type GameState struct {
	PermitType string
	Depth      int
	Integrity  int
	Actions    int
	Finished   bool
}

type DigResult struct {
	ItemName string
	Value    int
}

var (
	ErrNoMoney  = errors.New("not enough money")
	ErrFinished = errors.New("game already finished")
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) NewGame(userID int64, permitType string) (*GameState, error) {
	ok, _, err := s.store.CheckGameLimit(userID, "dig", 10)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrDigLimit
	}

	if permitType == "faille" {
		bal, err := s.store.GetBalance(userID)
		if err != nil {
			return nil, err
		}
		if bal < 200 {
			return nil, ErrNoMoney
		}
		if _, err := s.store.UpdateBalance(userID, -200); err != nil {
			return nil, err
		}
	}

	_ = s.store.IncrementGameLimit(userID, "dig")

	return &GameState{
		PermitType: permitType,
		Depth:      50,
		Integrity:  100,
		Actions:    5,
	}, nil
}

type ActionType string

const (
	ActionDynamite ActionType = "dynamite"
	ActionHammer   ActionType = "hammer"
	ActionBrush    ActionType = "brush"
)

type ActionOutcome struct {
	State      GameState
	DepthRem   int
	RiskChance int
	IntLoss    int
	Damaged    bool
	Finished   bool
}

func (s *Service) ApplyAction(state *GameState, action ActionType) *ActionOutcome {
	var depthRem, riskChance, intLoss int
	switch action {
	case ActionDynamite:
		depthRem, riskChance, intLoss = 20, 50, 30
	case ActionHammer:
		depthRem, riskChance, intLoss = 10, 15, 10
	case ActionBrush:
		depthRem, riskChance, intLoss = 2, 0, 0
	}

	state.Actions--
	state.Depth -= depthRem
	if state.Depth < 0 {
		state.Depth = 0
	}

	finalRisk := riskChance
	if state.PermitType == "safe" && riskChance > 0 {
		finalRisk = riskChance / 2
	}


	damaged := false
	if finalRisk > 0 && rand.Intn(100) < finalRisk {
		state.Integrity -= intLoss
		if state.Integrity < 0 {
			state.Integrity = 0
		}
		damaged = true
	}

	finished := state.Depth <= 0 || state.Integrity <= 0 || state.Actions <= 0
	state.Finished = finished

	return &ActionOutcome{
		State:      *state,
		DepthRem:   depthRem,
		RiskChance: riskChance,
		IntLoss:    intLoss,
		Damaged:    damaged,
		Finished:   finished,
	}
}

func (s *Service) Resolve(state *GameState) *DigResult {
	if state.Integrity <= 0 {
		return &DigResult{ItemName: "bone_dust", Value: 1}
	}
	if state.Depth > 0 && state.Actions <= 0 {
		return &DigResult{ItemName: "bone_dust", Value: 1}
	}

	if state.Integrity < 50 {
		return &DigResult{ItemName: "damaged_fossil", Value: 50}
	}

	if state.Integrity == 100 {
		return &DigResult{ItemName: "pure_dna", Value: 3000}
	}

	if state.PermitType == "safe" {
		roll := rand.Float64()
		if roll < 0.60 {
			return &DigResult{ItemName: "common_fossil", Value: 150}
		} else if roll < 0.90 {
			return &DigResult{ItemName: "rare_fossil", Value: 300}
		} else {
			return &DigResult{ItemName: "epic_fossil", Value: 500}
		}
	}

	return &DigResult{ItemName: "legendary_fragment", Value: 1000}
}

var ReanimatePools = map[string]struct {
	ItemName string
	Pets     []string
}{
	"commun":    {ItemName: "common_fossil", Pets: []string{"Escargot", "Souris", "Cochon", "Grenouille", "Mouton"}},
	"rare":      {ItemName: "rare_fossil", Pets: []string{"Chien", "Chat", "Cheval", "Renard", "Singe", "Ours"}},
	"epic":      {ItemName: "epic_fossil", Pets: []string{"Chameau", "Panda", "Tigre", "Pieuvre"}},
	"legendary": {ItemName: "legendary_fragment", Pets: []string{"Dragon", "Tyrannosaure", "Diplodocus", "Mamouth"}},
	"pure_dna":  {ItemName: "pure_dna", Pets: []string{"Mégalodon", "Kraken", "Licorne", "Phoenix", "Cerbère"}},
}

func (s *Service) Reanimate(userID int64, rarity string) (string, error) {
	pool, ok := ReanimatePools[rarity]
	if !ok {
		return "", errors.New("invalid rarity")
	}

	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ? AND quantity >= ?", userID, pool.ItemName, 5).First(&inv).Error; err != nil {
		return "", errors.New("not enough parts")
	}

	if inv.Quantity <= 5 {
		s.store.DB.Delete(&inv)
	} else {
		s.store.DB.Model(&inv).UpdateColumn("quantity", gorm.Expr("quantity - 5"))
	}

	petName := pool.Pets[rand.Intn(len(pool.Pets))]

	pet := model.UserPet{
		UserID:   userID,
		PetType:  petName,
		Nickname: petName,
		Level:    1,
		XP:       0,
		MaxHP:    50,
		HP:       50,
		Atk:      10,
		Defense:  5,
		Speed:    10,
		DGE:      5,
		ACC:      0,
		CritC:    5,
		CritD:    1.5,
		IsActive: false,
	}
	if err := s.store.DB.Create(&pet).Error; err != nil {
		return "", err
	}

	return petName, nil
}
