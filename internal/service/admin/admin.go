package admin

import (
	"gorm.io/gorm"

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
