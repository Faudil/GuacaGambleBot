package inventory

import (
	"errors"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
	"gorm.io/gorm"
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
		it := items.Get("")
		if iv.ItemID > 0 {
			var dbItem model.Item
			if err := s.store.DB.First(&dbItem, iv.ItemID).Error; err == nil {
				it = items.Get(dbItem.Name)
			}
		}
		name := ""
		if it != nil {
			name = it.Name
		} else {
			var dbItem model.Item
			if err := s.store.DB.First(&dbItem, iv.ItemID).Error; err == nil {
				name = dbItem.Name
			}
		}
		it2 := items.Get(name)
		entries = append(entries, InvEntry{ItemName: name, Quantity: iv.Quantity, Item: it2})
		current += iv.Quantity
	}
	return &InvResult{Entries: entries, Current: current, Limit: limit, UserID: userID}, nil
}

func (s *Service) HasItem(userID int64, itemName string, quantity int) bool {
	var dbItem model.Item
	if err := s.store.DB.Where("name = ?", itemName).First(&dbItem).Error; err != nil {
		return false
	}
	var inv model.Inventory
	if err := s.store.DB.Where("user_id = ? AND item_id = ?", userID, dbItem.ID).First(&inv).Error; err != nil {
		return false
	}
	return inv.Quantity >= quantity
}

func (s *Service) RemoveItem(db *gorm.DB, userID int64, itemName string, quantity int) error {
	var dbItem model.Item
	if err := db.Where("name = ?", itemName).First(&dbItem).Error; err != nil {
		return err
	}
	return db.Where("user_id = ? AND item_id = ?", userID, dbItem.ID).
		UpdateColumn("quantity", gorm.Expr("quantity - ?", quantity)).Error
}

func (s *Service) AddItem(db *gorm.DB, userID int64, itemName string, quantity int) error {
	var dbItem model.Item
	if err := db.Where("name = ?", itemName).FirstOrCreate(&dbItem, model.Item{
		Name: itemName, Price: 0, Description: "", EffectType: "",
	}).Error; err != nil {
		return err
	}
	return db.Where("user_id = ? AND item_id = ?", userID, dbItem.ID).
		FirstOrCreate(&model.Inventory{UserID: userID, ItemID: dbItem.ID, Quantity: 0}).
		UpdateColumn("quantity", gorm.Expr("quantity + ?", quantity)).Error
}
