package inventory

import (
	"errors"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound = errors.New("item not found")
	ErrNoMoney  = errors.New("insufficient funds")
	ErrNoItem   = errors.New("item not owned")
)

type Service struct {
	store *store.Store
	cfg   *config.Config
}

type InvEntry struct {
	ItemName string
	Quantity int
	Item     *items.Item
}

type InvResult struct {
	Entries []InvEntry
	Current int
	Limit   int
	UserID  int64
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

func (s *Service) GetInventory(userID int64) (*InvResult, error) {
	if _, err := s.store.GetBalance(userID); err != nil {
		return nil, err
	}
	var u model.User
	if err := s.store.DB.Where("user_id = ?", userID).First(&u).Error; err != nil {
		return nil, err
	}
	limit := 50 + u.ExtraInvSlots

	var inv []model.Inventory
	if err := s.store.DB.Where("user_id = ? AND quantity > 0", userID).Find(&inv).Error; err != nil {
		return nil, err
	}
	entries := make([]InvEntry, 0, len(inv))
	current := 0
	for _, iv := range inv {
		it := items.Get(iv.ItemID)
		name := ""
		if it != nil {
			name = it.Name
		}
		entries = append(entries, InvEntry{ItemName: name, Quantity: iv.Quantity, Item: it})
		current += iv.Quantity
	}
	return &InvResult{Entries: entries, Current: current, Limit: limit, UserID: userID}, nil
}

func (s *Service) HasItem(userID int64, itemID string, quantity int) bool {
	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, itemID).First(&inv).Error; err != nil {
		return false
	}
	return inv.Quantity >= quantity
}

func (s *Service) RemoveItem(db *gorm.DB, userID int64, itemID string, quantity int) error {
	return db.Model(&model.Inventory{}).
		Where("user_id = ? AND item_id = ?", userID, itemID).
		UpdateColumn("quantity", gorm.Expr("quantity - ?", quantity)).Error
}

func (s *Service) AddItem(db *gorm.DB, userID int64, itemID string, quantity int) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "item_id"}},
		DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("quantity + ?", quantity)},
		)}).Create(&model.Inventory{UserID: userID, ItemID: itemID, Quantity: quantity}).Error
}
