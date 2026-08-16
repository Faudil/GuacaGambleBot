package admin

import (
	"errors"

	"gorm.io/gorm"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

var ErrItemNotFound = errors.New("item not found")

type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) GiveMoney(userID int64, amount int) (int, error) {
	return s.store.UpdateBalance(userID, amount)
}

func (s *Service) GiveCrowns(userID int64, amount int) error {
	return s.store.DB.Model(&model.User{}).Where("user_id = ?", userID).
		UpdateColumn("crowns", gorm.Expr("crowns + ?", amount)).Error
}

func (s *Service) AirdropAll(amount int) (int, error) {
	res := s.store.DB.Model(&model.User{}).
		Where("1 = 1").
		UpdateColumn("balance", gorm.Expr("balance + ?", amount))
	return int(res.RowsAffected), res.Error
}

func (s *Service) GiveItem(userID int64, itemID string, quantity int) error {
	if quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	it := items.Get(itemID)
	if it == nil {
		return ErrItemNotFound
	}
	return s.store.AddItemRaw(s.store.DB, userID, it.ID, quantity)
}

func (s *Service) AirdropItemAll(itemID string, quantity int) (int, error) {
	if quantity <= 0 {
		return 0, errors.New("quantity must be positive")
	}
	it := items.Get(itemID)
	if it == nil {
		return 0, ErrItemNotFound
	}
	var ids []int64
	if err := s.store.DB.Model(&model.User{}).Pluck("user_id", &ids).Error; err != nil {
		return 0, err
	}
	count := 0
	err := s.store.DB.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			if err := s.store.AddItemRaw(tx, id, it.ID, quantity); err != nil {
				return err
			}
			count++
		}
		return nil
	})
	return count, err
}

func (s *Service) ResetEconomy() error {
	return s.store.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("1 = 1").Update("balance", s.cfg.StartingBalance).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("1 = 1").Update("bank", 0).Error; err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Delete(&model.Loan{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *Service) SetLanguage(serverID int64, lang string) error {
	ss := &model.ServerSetting{ServerID: serverID, Language: lang}
	return s.store.SaveServerSetting(ss)
}
