package inventory

import (
	"errors"

	"gorm.io/gorm"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
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
	ItemName  string
	Quantity  int
	Item      *items.Item
	EquipInfo *EquipInfo // non-nil for equipment items
}

type EquipInfo struct {
	EquipID    uint
	Rarity     string
	Emoji      string
	StatSTR    int
	StatDEX    int
	StatINT    int
	StatVIT    int
	StatLUK    int
	SetID      string
	IsEquipped bool
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
	limit, err := s.store.InventoryLimit(s.store.DB, userID)
	if err != nil {
		return nil, err
	}

	// Stackable items from Inventory table
	var inv []model.Inventory
	if err := s.store.DB.Where("user_id = ? AND quantity > 0", userID).Find(&inv).Error; err != nil {
		return nil, err
	}
	entries := make([]InvEntry, 0, len(inv)+10)
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

	// Equipment instances from UserEquipment table
	allEquip, err := s.store.GetAllUserEquipment(userID)
	if err == nil {
		for _, eq := range allEquip {
			base := items.Get(eq.BaseID)
			name := eq.Name
			if name == "" && base != nil {
				name = base.Name
			}
			entries = append(entries, InvEntry{
				ItemName: name,
				Quantity: 1,
				Item:     base,
				EquipInfo: &EquipInfo{
					EquipID:    eq.ID,
					Rarity:     eq.Rarity,
					Emoji:      eq.Emoji,
					StatSTR:    eq.StatSTR,
					StatDEX:    eq.StatDEX,
					StatINT:    eq.StatINT,
					StatVIT:    eq.StatVIT,
					StatLUK:    eq.StatLUK,
					SetID:      eq.SetID,
					IsEquipped: eq.IsEquipped,
				},
			})
			current++
		}
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
	return s.store.AddItemRaw(db, userID, itemID, quantity)
}
