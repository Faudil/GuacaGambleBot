package pets

import (
	"math"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/achievement"
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

func (s *Service) GetPets(userID int64) ([]model.UserPet, error) {
	var pets []model.UserPet
	err := s.store.DB.Where("user_id = ?", userID).Find(&pets).Error
	return pets, err
}

func (s *Service) GetPetByID(petID int64) (*model.UserPet, error) {
	var pet model.UserPet
	err := s.store.DB.First(&pet, petID).Error
	if err != nil {
		return nil, err
	}
	return &pet, nil
}

func (s *Service) GetActivePet(userID int64) (*model.UserPet, error) {
	var pet model.UserPet
	err := s.store.DB.Where("user_id = ? AND is_active = ?", userID, true).First(&pet).Error
	if err != nil {
		return nil, err
	}
	return &pet, nil
}

func (s *Service) SetActivePet(userID int64, petID int64) error {
	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.UserPet{}).
			Where("user_id = ? AND is_active = ?", userID, true).
			Update("is_active", false).Error; err != nil {
			return err
		}
		return tx.Model(&model.UserPet{}).
			Where("id = ? AND user_id = ?", petID, userID).
			Update("is_active", true).Error
	})
}

func (s *Service) CreatePet(userID int64, petType string) (*model.UserPet, error) {
	pt, ok := PetTypes[petType]
	if !ok {
		return nil, nil
	}
	pet := model.UserPet{
		UserID:   userID,
		PetType:  petType,
		Nickname: petType,
		MaxHP:    pt.MaxHP,
		HP:       pt.MaxHP,
		Atk:      pt.Atk,
		Defense:  pt.Defense,
		Speed:    pt.Speed,
		DGE:      pt.DGE,
		ACC:      pt.ACC,
		CritC:    pt.CritC,
		CritD:    pt.CritD,
		Bonus:    pt.Bonus,
		Elo:      1000,
	}
	err := s.store.DB.Create(&pet).Error
	if err != nil {
		return nil, err
	}
	return &pet, nil
}

func (s *Service) UpdatePet(pet *model.UserPet) error {
	return s.store.DB.Save(pet).Error
}

func (s *Service) DeletePet(petID int64) error {
	return s.store.DB.Delete(&model.UserPet{}, petID).Error
}

func (s *Service) TransferPet(petID int64, newOwnerID int64) error {
	return s.store.DB.Model(&model.UserPet{}).
		Where("id = ?", petID).
		Update("user_id", newOwnerID).Error
}

func (s *Service) HealPet(pet *model.UserPet, cost int) error {
	if _, err := s.store.UpdateBalance(pet.UserID, -cost); err != nil {
		return err
	}
	pet.HP = pet.MaxHP
	return s.UpdatePet(pet)
}

func (s *Service) AddXP(pet *model.UserPet, amount int) bool {
	rMult := RarityXP[petTypeRarity(pet.PetType)]
	if (pet.Level >= 20 && pet.TrsLvl == 0) || pet.Level >= 30 {
		pet.XP = 0
		return false
	}
	pet.XP += amount
	leveled := false
	for pet.XP >= int(float64(pet.Level)*rMult*100) && pet.Level < 20 {
		pet.XP -= int(float64(pet.Level) * rMult * 100)
		pet.Level++
		pet.MaxHP += 5
		pet.HP += 5
		pet.Atk += 2
		pet.Defense += 1
		if pet.Level == 5 {
			pet.Elo = 1000
			pet.SpcC = 5
		} else if pet.Level == 10 {
			pet.SpcC = 10
		} else if pet.Level == 20 {
			pet.SpcC = 15
		}
		leveled = true
	}
	return leveled
}

func (s *Service) FeedPet(pet *model.UserPet, stat string, amount int) error {
	rCap := RarityFoodCapacity[petTypeRarity(pet.PetType)]
	if pet.FoodEaten >= rCap*pet.Level {
		return nil
	}
	switch stat {
	case "max_hp":
		pet.MaxHP += amount
		pet.HP += amount
	case "atk":
		pet.Atk += amount
	case "defense":
		pet.Defense += amount
	case "speed":
		pet.Speed += amount
	case "dge":
		pet.DGE += amount
	case "acc":
		pet.ACC += amount
	case "crit_c":
		pet.CritC += amount
	case "crit_d":
		pet.CritD += float64(amount)
	}
	pet.FoodEaten++
	return s.UpdatePet(pet)
}

func (s *Service) ForgetXP(pet *model.UserPet) bool {
	if pet.Level < 20 {
		return false
	}
	pt, ok := PetTypes[pet.PetType]
	if !ok {
		return false
	}
	pet.XP = 0
	pet.Level = 10
	pet.MaxHP = pt.MaxHP + 50
	pet.HP = pet.MaxHP
	pet.Atk = pt.Atk + 20
	pet.Defense = pt.Defense + 10
	pet.CritC = pt.CritC
	pet.CritD = pt.CritD
	pet.ACC = pt.ACC
	pet.DGE = pt.DGE
	pet.Speed = pt.Speed
	pet.FoodEaten = 0
	return true
}

func (s *Service) UpdateElo(p1, p2 *model.UserPet, result float64) (int, int) {
	if p1.Level < 5 || p2.Level < 5 {
		return 0, 0
	}
	K := 32.0
	eS := 1.0 / (1.0 + math.Pow(10, float64(p2.Elo-p1.Elo)/400))
	eO := 1.0 / (1.0 + math.Pow(10, float64(p1.Elo-p2.Elo)/400))
	scoreO := 1.0 - result
	dS := int(K * (result - eS))
	dO := int(K * (scoreO - eO))
	p1.Elo += dS
	p2.Elo += dO
	if p1.Elo < 0 {
		p1.Elo = 0
	}
	if p2.Elo < 0 {
		p2.Elo = 0
	}
	return dS, dO
}

func (s *Service) GetServerElo(petID, serverID int64) (int, error) {
	var elo model.ServerPetElo
	err := s.store.DB.Where("pet_id = ? AND server_id = ?", petID, serverID).First(&elo).Error
	if err != nil {
		return 1000, nil
	}
	return elo.Elo, nil
}

func (s *Service) UpdateServerElo(petID, serverID int64, elo int) error {
	return s.store.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "pet_id"}, {Name: "server_id"}},
		DoUpdates: clause.Assignments(map[string]any{"elo": elo}),
	}).Create(&model.ServerPetElo{PetID: petID, ServerID: serverID, Elo: elo}).Error
}

func (s *Service) CheckAndUnlock(userID int64) ([]*achievement.Achievement, error) {
	return achievement.CheckAndUnlock(s.store.DB, userID)
}

func (s *Service) IncrementStat(userID int64, stat string, amount int) error {
	return achievement.IncrementStat(s.store.DB, userID, stat, amount)
}

func petTypeRarity(petType string) string {
	if pt, ok := PetTypes[petType]; ok {
		return pt.Rarity
	}
	return RarityCommon
}
