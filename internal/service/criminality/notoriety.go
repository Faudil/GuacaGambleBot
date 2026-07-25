package criminality

import (
	"fmt"
	"math/rand"
	"time"
)

// DecayNotoriety checks and applies daily notoriety decay. Returns the new notoriety.
func (svc *Service) DecayNotoriety(userID int64) (int, error) {
	ok, err := svc.store.CheckCooldown(userID, "notoriety_decay", 24*time.Hour)
	if err != nil || !ok {
		// Already decayed today — just return current
		c, err := svc.store.GetCriminality(userID)
		if err != nil {
			return 0, err
		}
		return c.Notoriety, nil
	}

	svc.store.SetCooldown(userID, "notoriety_decay")
	return svc.store.DecayNotoriety(userID, svc.cfg.NotorietyDecayDaily)
}

// CheckGuardAttack simulates a guard encounter for high-notoriety players.
// Returns (attacked bool, message).
func (svc *Service) CheckGuardAttack(userID int64, lang string) (bool, string, error) {
	crim, err := svc.store.GetCriminality(userID)
	if err != nil {
		return false, "", err
	}

	if crim.Notoriety < 51 {
		return false, "", nil
	}

	// Guard detection roll — higher notoriety = more likely
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	chance := crim.Notoriety - 50 // 1-50%
	if chance > 50 {
		chance = 50
	}
	if rng.Intn(100) < chance {
		// Try to dodge (DEX check)
		char, _ := svc.store.EnsureCharacter(userID)
		dodgeRoll := rng.Intn(20) + char.DEX
		if dodgeRoll >= 15 {
			return true, svc.T(lang, "guard.dodge"), nil
		}

		// Caught — fine gold
		bal, _ := svc.store.GetBalance(userID)
		fine := bal / 20
		if fine < 10 {
			fine = 10
		}
		if fine > bal {
			fine = bal
		}
		if fine > 0 {
			svc.store.UpdateBalance(userID, -fine)
		}
		return true, svc.T(lang, "guard.caught", map[string]any{"gold": fine}), nil
	}

	return false, "", nil
}

// ApplyPacifistBlessing grants the Pacifist's Blessing for 7 days.
func (svc *Service) ApplyPacifistBlessing(userID int64, lang string) (string, error) {
	bal, _ := svc.store.GetBalance(userID)
	if bal < svc.cfg.PacifistGoldPrice {
		return "", fmt.Errorf(svc.T(lang, "blessing.need_gold", map[string]any{"gold": svc.cfg.PacifistGoldPrice}))
	}

	svc.store.UpdateBalance(userID, -svc.cfg.PacifistGoldPrice)
	pacifistUntil := time.Now().Add(7 * 24 * time.Hour)
	svc.store.UpdateCriminality(userID, map[string]any{"pacifist_until": pacifistUntil})

	svc.store.AddCrimeRecord(userID, "pacifist_blessing", fmt.Sprintf(`{"duration":"7d","cost":%d}`, svc.cfg.PacifistGoldPrice))

	return svc.T(lang, "blessing.success", map[string]any{"gold": svc.cfg.PacifistGoldPrice}), nil
}

// ApplyCleanSlate hides the player from hunter tracking for 24h.
func (svc *Service) ApplyCleanSlate(userID int64, lang string) (string, error) {
	bal, _ := svc.store.GetBalance(userID)
	if bal < svc.cfg.CleanSlateGoldPrice {
		return "", fmt.Errorf(svc.T(lang, "cleanse.need_gold", map[string]any{"gold": svc.cfg.CleanSlateGoldPrice}))
	}

	svc.store.UpdateBalance(userID, -svc.cfg.CleanSlateGoldPrice)
	svc.store.UpdateCriminality(userID, map[string]any{"notoriety": 0})

	svc.store.AddCrimeRecord(userID, "clean_slate", fmt.Sprintf(`{"cost":%d}`, svc.cfg.CleanSlateGoldPrice))

	return svc.T(lang, "cleanse.success", map[string]any{"gold": svc.cfg.CleanSlateGoldPrice}), nil
}
