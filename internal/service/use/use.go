package use

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/service/magnet"
	"guacagamblebot/internal/store"
)

var (
	ErrNotUsable = errors.New("this item cannot be used")
	ErrNotOwned  = errors.New("you don't own this item")
)

// usableItems lists the consumable items that the /use command can apply.
var usableItems = map[string]struct{}{
	"beer":            {},
	"hook":            {},
	"coffee":          {},
	"bow":             {},
	"rigged_coin":     {},
	"vip_ticket":      {},
	"casino_token":    {},
	"scratch_ticket":  {},
	"fortune_cookie":  {},
	"rusty_magnet":    {},
	"magnet":          {},
	"electric_magnet": {},
}

// IsUsable reports whether the item can be consumed with the use command.
func IsUsable(itemID string) bool {
	_, ok := usableItems[itemID]
	return ok
}

// apply applies the effect of a usable item and returns a description of what
// happened. The caller is responsible for checking ownership and consuming the
// item from inventory.
func apply(st *store.Store, userID int64, itemID string) (string, error) {
	switch itemID {
	case "beer":
		if err := st.GrantGameLimitCredit(userID, "mine_descend", 2); err != nil {
			return "", err
		}
		return "🍺 **Beer!** The miner's spirits lift — +2 mining descends granted.", nil

	case "hook":
		if err := st.ClearCooldown(userID, "fish"); err != nil {
			return "", err
		}
		return "🪝 **Hook!** The fishing cooldown is reset.", nil

	case "coffee":
		if err := st.ClearCooldown(userID, "daily"); err != nil {
			return "", err
		}
		return "☕ **Coffee!** The daily claim cooldown is reset.", nil

	case "bow":
		if err := st.ClearCooldown(userID, "hunt"); err != nil {
			return "", err
		}
		return "🏹 **Bow!** The hunting cooldown is reset.", nil

	case "rigged_coin":
		if err := st.SetActiveBuff(userID, "rigged_coin"); err != nil {
			return "", err
		}
		return "🪙 **Rigged Coin!** Your next coinflip has 75% odds.", nil

	case "vip_ticket":
		if err := st.ResetGameLimit(userID, "slots"); err != nil {
			return "", err
		}
		if err := st.ResetGameLimit(userID, "coinflip"); err != nil {
			return "", err
		}
		if err := st.ResetGameLimit(userID, "mega_slots"); err != nil {
			return "", err
		}
		return "🎟️ **VIP Ticket!** Your casino, coinflip and mega slots limits are refreshed.", nil

	case "casino_token":
		if err := st.ResetGameLimit(userID, "slots"); err != nil {
			return "", err
		}
		if err := st.ResetGameLimit(userID, "mega_slots"); err != nil {
			return "", err
		}
		return "🎰 **Casino Token!** Your slots and mega slots limits are refreshed.", nil

	case "scratch_ticket":
		win := rand.Intn(1001)
		if _, err := st.UpdateBalance(userID, win); err != nil {
			return "", err
		}
		if win == 0 {
			return "🎰 **Scratch Ticket!** Nothing. Better luck next time!", nil
		}
		return "🎉 **Scratch Ticket!** You win **$" + itoa(win) + "**!", nil

	case "fortune_cookie":
		gold := 5 + rand.Intn(46)
		if _, err := st.UpdateBalance(userID, gold); err != nil {
			return "", err
		}
		return "🥠 **Fortune Cookie!** \"" + fortune() + "\" (+$" + itoa(gold) + ")", nil

	case "rusty_magnet", "magnet", "electric_magnet":
		picks := magnet.Pull(itemID)
		if len(picks) == 0 {
			return "", ErrNotUsable
		}
		counts := make(map[string]int, len(picks))
		for _, id := range picks {
			counts[id]++
		}
		for id, qty := range counts {
			if err := st.AddItemRaw(st.DB, userID, id, qty); err != nil {
				return "", err
			}
		}
		haul := magnetHaul(counts)
		switch itemID {
		case "rusty_magnet":
			return "🧲 **Rusty Magnet!** You pull some scrap ore from the ground: " + haul + ".", nil
		case "magnet":
			return "🧲 **Magnet!** The magnet sweeps up precious ore: " + haul + ".", nil
		default:
			return "⚡ **Electric Magnet!** A powerful field drags valuable ore to your pack: " + haul + ".", nil
		}

	default:
		return "", ErrNotUsable
	}
}

// magnetHaul renders the pulled ore list of a magnet use, e.g.
// "⛏️ Iron Ore ×2, 🪨 Coal".
func magnetHaul(counts map[string]int) string {
	parts := make([]string, 0, len(counts))
	for id, qty := range counts {
		it := items.Get(id)
		if it == nil {
			continue
		}
		if qty > 1 {
			parts = append(parts, fmt.Sprintf("%s %s ×%d", it.Emoji, it.Name, qty))
		} else {
			parts = append(parts, fmt.Sprintf("%s %s", it.Emoji, it.Name))
		}
	}
	return strings.Join(parts, ", ")
}

var fortunes = []string{
	"A gambler's luck is made, not found.",
	"The house always wins... eventually.",
	"Dig deep and the mountain will reward you.",
	"A full belly makes a happy pet.",
	"Fortune favors the bold, but not the reckless.",
	"The tide turns in your favor tomorrow.",
	"Your patience will be repaid in ore.",
}

func fortune() string {
	return fortunes[rand.Intn(len(fortunes))]
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	if neg {
		out = "-" + out
	}
	return out
}

// Service exposes the item-use logic.
type Service struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Service {
	return &Service{store: s, cfg: cfg}
}

// Apply checks ownership, applies the effect and removes the item from
// inventory. It returns a human-readable description of what happened.
func (s *Service) Apply(userID int64, itemID string) (string, error) {
	ok, err := s.store.HasItem(userID, itemID, 1)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrNotOwned
	}
	desc, err := apply(s.store, userID, itemID)
	if err != nil {
		return "", err
	}
	if err := s.store.RemoveInventoryItem(userID, itemID, 1); err != nil {
		return "", err
	}
	return desc, nil
}
