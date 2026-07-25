package store

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"guacagamblebot/internal/model"
)

// -- UserCriminality --

func (s *Store) EnsureCriminality(userID int64) (*model.UserCriminality, error) {
	var c model.UserCriminality
	err := s.DB.Where("user_id = ?", userID).
		Attrs(model.UserCriminality{UserID: userID}).
		FirstOrCreate(&c).Error
	return &c, err
}

func (s *Store) GetCriminality(userID int64) (*model.UserCriminality, error) {
	return s.EnsureCriminality(userID)
}

func (s *Store) UpdateCriminality(userID int64, updates map[string]any) error {
	return s.DB.Model(&model.UserCriminality{}).
		Where("user_id = ?", userID).Updates(updates).Error
}

func (s *Store) AddNotoriety(userID int64, amount int) (int, error) {
	var c model.UserCriminality
	err := s.DB.Where("user_id = ?", userID).First(&c).Error
	if err == gorm.ErrRecordNotFound {
		if _, err := s.EnsureCriminality(userID); err != nil {
			return 0, err
		}
	}
	err = s.DB.Model(&model.UserCriminality{}).
		Where("user_id = ?", userID).
		UpdateColumn("notoriety", gorm.Expr("LEAST(notoriety + ?, 100)", amount)).Error
	if err != nil {
		return 0, err
	}
	var updated model.UserCriminality
	s.DB.Where("user_id = ?", userID).First(&updated)
	return updated.Notoriety, nil
}

func (s *Store) DecayNotoriety(userID int64, amount int) (int, error) {
	var c model.UserCriminality
	err := s.DB.Where("user_id = ?", userID).First(&c).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	err = s.DB.Model(&model.UserCriminality{}).
		Where("user_id = ?", userID).
		UpdateColumn("notoriety", gorm.Expr("GREATEST(notoriety - ?, 0)", amount)).Error
	if err != nil {
		return 0, err
	}
	var updated model.UserCriminality
	s.DB.Where("user_id = ?", userID).First(&updated)
	return updated.Notoriety, nil
}

// -- WorldCriminalityState --

func (s *Store) GetWorldState(serverID int64) (*model.WorldCriminalityState, error) {
	var ws model.WorldCriminalityState
	err := s.DB.Where("server_id = ?", serverID).
		Attrs(model.WorldCriminalityState{ServerID: serverID}).
		FirstOrCreate(&ws).Error
	return &ws, err
}

func (s *Store) SaveWorldState(ws *model.WorldCriminalityState) error {
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "server_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"awakened", "first_thief_id", "first_victim_id",
			"awakened_at", "mask_claimed_by", "mask_claimed_at",
		}),
	}).Create(ws).Error
}

func (s *Store) IsAwakened(serverID int64) (bool, error) {
	ws, err := s.GetWorldState(serverID)
	if err != nil {
		return false, err
	}
	return ws.Awakened, nil
}

// -- Bounties --

func (s *Store) CreateBounty(b *model.Bounty) error {
	return s.DB.Create(b).Error
}

func (s *Store) GetActiveBounties() ([]model.Bounty, error) {
	var bounties []model.Bounty
	err := s.DB.Where("claimed_by IS NULL").Order("placed_at desc").Find(&bounties).Error
	return bounties, err
}

func (s *Store) GetActiveBountiesForTarget(targetID int64) ([]model.Bounty, error) {
	var bounties []model.Bounty
	err := s.DB.Where("target_id = ? AND claimed_by IS NULL", targetID).Find(&bounties).Error
	return bounties, err
}

func (s *Store) ClaimBounty(bountyID int64, hunterID int64) error {
	return s.DB.Model(&model.Bounty{}).
		Where("id = ? AND claimed_by IS NULL", bountyID).
		Updates(map[string]any{
			"claimed_by": hunterID,
			"claimed_at": time.Now(),
		}).Error
}

func (s *Store) GetBounty(id int64) (*model.Bounty, error) {
	var b model.Bounty
	err := s.DB.First(&b, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &b, err
}

// -- TheftRecords --

func (s *Store) CreateTheftRecord(r *model.TheftRecord) error {
	return s.DB.Create(r).Error
}

func (s *Store) GetTheftRecordsForVictim(victimID int64) ([]model.TheftRecord, error) {
	var recs []model.TheftRecord
	err := s.DB.Where("victim_id = ?", victimID).Order("created_at desc").Find(&recs).Error
	return recs, err
}

func (s *Store) GetTheftRecordsByThief(thiefID int64) ([]model.TheftRecord, error) {
	var recs []model.TheftRecord
	err := s.DB.Where("thief_id = ?", thiefID).Order("created_at desc").Find(&recs).Error
	return recs, err
}

func (s *Store) ForgiveTheft(id int64) error {
	return s.DB.Model(&model.TheftRecord{}).
		Where("id = ?", id).Update("forgiven", true).Error
}

// -- CrimeRecords --

func (s *Store) AddCrimeRecord(userID int64, event string, detail string) error {
	return s.DB.Create(&model.CrimeRecord{
		UserID:    userID,
		Event:     event,
		Detail:    detail,
		CreatedAt: time.Now(),
	}).Error
}

func (s *Store) GetCrimeRecords(userID int64) ([]model.CrimeRecord, error) {
	var recs []model.CrimeRecord
	err := s.DB.Where("user_id = ?", userID).
		Order("created_at desc").Limit(50).Find(&recs).Error
	return recs, err
}

// -- HunterContracts --

func (s *Store) CreateContract(c *model.HunterContract) error {
	return s.DB.Create(c).Error
}

func (s *Store) GetAvailableContracts() ([]model.HunterContract, error) {
	var contracts []model.HunterContract
	err := s.DB.Where("available = true AND claimed_by IS NULL").
		Order("target_level asc").Find(&contracts).Error
	return contracts, err
}

func (s *Store) ClaimContract(contractID string, hunterID int64) error {
	return s.DB.Model(&model.HunterContract{}).
		Where("contract_id = ? AND available = true AND claimed_by IS NULL", contractID).
		Updates(map[string]any{
			"claimed_by": hunterID,
			"available":  false,
		}).Error
}

func (s *Store) GetContractByID(contractID string) (*model.HunterContract, error) {
	var c model.HunterContract
	err := s.DB.Where("contract_id = ?", contractID).First(&c).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &c, err
}

// GetUserQuest checks if a user has a specific quest record.
func (s *Store) GetUserQuest(userID int64, questID string) (*model.UserQuest, *model.UserQuestData, error) {
	var uq model.UserQuest
	if err := s.DB.Where("user_id = ? AND quest_id = ?", userID, questID).First(&uq).Error; err != nil {
		return nil, nil, err
	}
	var uqd model.UserQuestData
	if err := s.DB.Where("user_id = ? AND quest_id = ?", userID, questID).First(&uqd).Error; err != nil {
		return &uq, nil, nil
	}
	return &uq, &uqd, nil
}

// -- Helpers --

func (s *Store) CharacterLevel(userID int64) (int, error) {
	c, err := s.EnsureCharacter(userID)
	if err != nil {
		return 0, err
	}
	return c.Level, nil
}
