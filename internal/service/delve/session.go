package delve

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	charsvc "guacagamblebot/internal/service/character"
	"guacagamblebot/internal/store"
)

// Rarity is the delve package's view of the canonical rarity enum, so delve
// loot tables can live next to the items they generate (items/catalog.go).
type Rarity = items.Rarity

const (
	Common    = items.RarityCommon
	Uncommon  = items.RarityUncommon
	Rare      = items.RarityRare
	Epic      = items.RarityEpic
	Legendary = items.RarityLegendary
)

var RarityEmoji = map[Rarity]string{
	Common:    "⬜",
	Uncommon:  "🟩",
	Rare:      "🔵",
	Epic:      "🟣",
	Legendary: "🟠",
}

type DelveItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Emoji       string `json:"emoji"`
	Rarity      Rarity `json:"rarity"`
	EquipSlot   string `json:"slot"`
	StatSTR     int    `json:"str"`
	StatDEX     int    `json:"dex"`
	StatINT     int    `json:"int"`
	StatVIT     int    `json:"vit"`
	StatLUK     int    `json:"luk"`
	Description string `json:"desc"`
	SetName     string `json:"set,omitempty"`
	IsCursed    bool   `json:"cursed"`
	IsSoulbound bool   `json:"soulbound"`
	Quantity    int    `json:"qty"`
	PrefixID    string `json:"prefix_id,omitempty"`
	BaseID      string `json:"base_id,omitempty"`
	SuffixID    string `json:"suffix_id,omitempty"`
}

type CombatState struct {
	Enemy            *Enemy
	Turn             int
	Active           bool
	EnemyFirstStrike bool
	PetBonded        bool
}

type Service struct {
	store   *store.Store
	cfg     *config.Config
	combats map[int64]*CombatState
	mu      sync.RWMutex
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{
		store:   s,
		cfg:     cfg,
		combats: make(map[int64]*CombatState),
	}
}

func (svc *Service) Store() *store.Store { return svc.store }
func (svc *Service) Cfg() *config.Config { return svc.cfg }

func (svc *Service) StartSession(userID, guildID, channelID int64) (*model.DelveSession, error) {
	session := &model.DelveSession{
		UserID:        userID,
		GuildID:       guildID,
		ChannelID:     channelID,
		Floor:         1,
		Zone:          "crypt",
		HP:            100,
		MaxHP:         100,
		Mana:          50,
		MaxMana:       50,
		Torches:       3,
		Keys:          0,
		Potions:       1,
		Gold:          0,
		Inventory:     "[]",
		DeployedPets:  "[]",
		Flags:         "[]",
		StatusEffects: "[]",
		RoomsCleared:  0,
		Seed:          rand.Int63(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	char, err := svc.store.EnsureCharacter(userID)
	if err != nil || char == nil {
		return nil, err
	}
	if err == nil && char != nil {
		session.MaxHP = 100 + char.VIT*10
		session.MaxMana = 50 + char.INT*5
	}

	equipped, _ := svc.store.GetEquipped(userID)
	for _, eq := range equipped {
		session.MaxHP += eq.StatVIT * 10
		session.MaxMana += eq.StatINT * 5
	}
	session.HP = session.MaxHP
	session.Mana = session.MaxMana

	if err := svc.store.SaveDelveSession(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (svc *Service) GetSession(userID int64) (*model.DelveSession, error) {
	return svc.store.GetDelveSession(userID)
}

func (svc *Service) SaveSession(session *model.DelveSession) error {
	session.UpdatedAt = time.Now()
	return svc.store.SaveDelveSession(session)
}

func (svc *Service) EndSession(session *model.DelveSession, outcome string) error {
	svc.mu.Lock()
	delete(svc.combats, session.UserID)
	svc.mu.Unlock()

	var flags []string
	json.Unmarshal([]byte(session.Flags), &flags)

	history := &model.DelveRunHistory{
		UserID:      session.UserID,
		RunDate:     time.Now(),
		Floors:      session.RoomsCleared,
		Outcome:     outcome,
		FlagsEarned: session.Flags,
		LootSummary: session.Inventory,
	}
	svc.store.SaveDelveRunHistory(history)
	_ = svc.store.RecordActivity(session.UserID, "delve_completions", 1)
	_ = svc.store.RecordActivity(session.UserID, "delve_floors_cleared", session.RoomsCleared)

	for _, fid := range flags {
		svc.store.AddDelveFlag(session.UserID, fid, fmt.Sprintf(`{"run":%d}`, session.RoomsCleared))
	}

	var inv []DelveItem
	json.Unmarshal([]byte(session.Inventory), &inv)
	for _, di := range inv {
		if di.EquipSlot != "" {
			// Equipment item → create UserEquipment instance with affixes
			rarStr := strings.ToLower(di.Rarity.String())
			slot := di.EquipSlot

			affixes := items.RollAffixes(items.Rarity(rarStr), slot)
			var applied []items.AppliedAffix
			for _, a := range affixes {
				applied = append(applied, items.AppliedAffix{
					ID:    a.ID,
					Name:  a.Name,
					Stat:  a.Stat,
					Value: items.RollAffixValue(a),
				})
			}

			totalSTR, totalDEX, totalINT, totalVIT, totalLUK := di.StatSTR, di.StatDEX, di.StatINT, di.StatVIT, di.StatLUK
			for _, a := range applied {
				switch a.Stat {
				case "str":
					totalSTR += a.Value
				case "dex":
					totalDEX += a.Value
				case "int":
					totalINT += a.Value
				case "vit":
					totalVIT += a.Value
				case "luk":
					totalLUK += a.Value
				}
			}

			affixData, _ := json.Marshal(applied)
			svc.store.CreateEquipment(session.UserID, di.ID, di.Name, di.Emoji,
				rarStr, slot, minLevelForRarity(di.Rarity),
				totalSTR, totalDEX, totalINT, totalVIT, totalLUK,
				affixData, "")
		} else {
			// Non-equipment item (veil key, etc.) → add to regular inventory
			it := items.GetDynamicByID(di.ID)
			if it == nil {
				item := items.Item{
					ID:          di.ID,
					Name:        di.Name,
					Emoji:       di.Emoji,
					Price:       di.Rarity.Rank() * 50,
					Description: di.Description,
					EffectType:  "equipment",
					Category:    items.Delve,
				}
				items.RegisterDynamic(item)
				it = items.GetDynamicByID(di.ID)
			}
			if it != nil {
				svc.store.AddItemRaw(svc.store.DB, session.UserID, it.ID, di.Quantity)
			}
		}
	}

	return svc.store.DeleteDelveSession(session.UserID)
}

func (svc *Service) AddFlag(session *model.DelveSession, flagID string) {
	var flags []string
	json.Unmarshal([]byte(session.Flags), &flags)
	for _, f := range flags {
		if f == flagID {
			return
		}
	}
	flags = append(flags, flagID)
	b, _ := json.Marshal(flags)
	session.Flags = string(b)
}

func (svc *Service) HasFlag(session *model.DelveSession, flagID string) bool {
	var flags []string
	json.Unmarshal([]byte(session.Flags), &flags)
	for _, f := range flags {
		if f == flagID {
			return true
		}
	}
	return false
}

func (svc *Service) AddItem(session *model.DelveSession, item DelveItem) {
	var inv []DelveItem
	json.Unmarshal([]byte(session.Inventory), &inv)
	var b []byte
	for i := range inv {
		if inv[i].ID == item.ID {
			inv[i].Quantity += item.Quantity
			b, _ = json.Marshal(inv)
			session.Inventory = string(b)
			return
		}
	}
	item.Quantity = 1
	if item.SetName != "" {
		svc.AddFlag(session, "set_item_collected")
	}
	inv = append(inv, item)
	b, _ = json.Marshal(inv)
	session.Inventory = string(b)
}

func (svc *Service) GetInventory(session *model.DelveSession) []DelveItem {
	var inv []DelveItem
	json.Unmarshal([]byte(session.Inventory), &inv)
	return inv
}

func (svc *Service) StartCombat(session *model.DelveSession, enemy *Enemy) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	svc.combats[session.UserID] = &CombatState{
		Enemy:     enemy,
		Turn:      0,
		Active:    true,
		PetBonded: charsvc.ConsumeBuff(svc.store, session.UserID, "pet_bond"),
	}
}

func (svc *Service) GetCombat(userID int64) *CombatState {
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	return svc.combats[userID]
}

func (svc *Service) EndCombat(userID int64) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	delete(svc.combats, userID)
}

func (svc *Service) DeployedPets(session *model.DelveSession) []int64 {
	var ids []int64
	json.Unmarshal([]byte(session.DeployedPets), &ids)
	return ids
}

func (svc *Service) StatusEffects(session *model.DelveSession) []string {
	var effects []string
	json.Unmarshal([]byte(session.StatusEffects), &effects)
	return effects
}
