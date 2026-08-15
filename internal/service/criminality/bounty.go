package criminality

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"guacagamblebot/internal/model"
)

type HuntResult struct {
	Success     bool
	MeritGained int
	GoldReward  int
	Message     string
	Captured    bool
	TrackClues  []string
}

// PlaceBounty places a bounty on a criminal player.
func (svc *Service) PlaceBounty(placerID, targetID int64, amount int, anonymous bool, lang string) (string, error) {
	// Lock the gold from placer's balance
	bal, err := svc.store.GetBalance(placerID)
	if err != nil {
		return "", err
	}
	if bal < amount {
		return "", fmt.Errorf("you only have %d gold", bal)
	}
	svc.store.UpdateBalance(placerID, -amount)

	bounty := &model.Bounty{
		TargetID:    targetID,
		PlacerID:    placerID,
		Amount:      amount,
		PlacedAt:    time.Now(),
		IsAnonymous: anonymous,
	}
	if err := svc.store.CreateBounty(bounty); err != nil {
		svc.store.UpdateBalance(placerID, amount) // refund
		return "", err
	}

	svc.store.AddCrimeRecord(placerID, "bounty_placed",
		fmt.Sprintf(`{"target_id":%d,"amount":%d,"anonymous":%v}`, targetID, amount, anonymous))
	svc.store.AddCrimeRecord(targetID, "bounty_received",
		fmt.Sprintf(`{"amount":%d}`, amount))

	placerDisplay := svc.T(lang, "bounty.anonymous")
	if !anonymous {
		placerDisplay = fmt.Sprintf("<@%d>", placerID)
	}

	return svc.T(lang, "bounty.placed", map[string]any{"placer": placerDisplay, "amount": amount, "target": targetID}), nil
}

// ListBounties returns all active bounties formatted as a string.
func (svc *Service) ListBounties(lang string) string {
	bounties, err := svc.store.GetActiveBounties()
	if err != nil || len(bounties) == 0 {
		return svc.T(lang, "bounty.board_empty")
	}

	var lines []string
	for _, b := range bounties {
		placer := svc.T(lang, "bounty.anonymous")
		if !b.IsAnonymous {
			placer = fmt.Sprintf("<@%d>", b.PlacerID)
		}
		lines = append(lines, svc.T(lang, "bounty.line", map[string]any{"target": b.TargetID, "amount": b.Amount, "placer": placer}))
	}
	return svc.T(lang, "bounty.board_list", map[string]any{"entries": strings.Join(lines, "\n")})
}

// StartHunt begins the tracking phase for a hunter against a target.
func (svc *Service) StartHunt(hunterID, targetID, serverID int64, lang string) (string, bool) {
	// Check notoriety threshold
	targetCrim, err := svc.store.GetCriminality(targetID)
	if err != nil || targetCrim.Notoriety < svc.cfg.NotorietyHuntThreshold {
		return svc.T(lang, "hunt.notoriety_too_low", map[string]any{
			"notoriety": safeNotoriety(targetCrim), "threshold": svc.cfg.NotorietyHuntThreshold,
		}), false
	}

	// Check cooldown
	ok, _ := svc.store.CheckCooldown(hunterID, "hunt_"+fmt.Sprint(targetID), time.Duration(svc.cfg.HuntCooldownHours)*time.Hour)
	if !ok {
		return svc.T(lang, "hunt.cooldown"), false
	}

	// Check pacifist
	if targetCrim.PacifistUntil != nil && targetCrim.PacifistUntil.After(time.Now()) {
		return svc.T(lang, "hunt.pacifist_protected"), false
	}

	svc.store.SetCooldown(hunterID, "hunt_"+fmt.Sprint(targetID))
	svc.store.AddCrimeRecord(hunterID, "hunt_started",
		fmt.Sprintf(`{"target_id":%d,"server_id":%d}`, targetID, serverID))

	return svc.T(lang, "hunt.started", map[string]any{"target": targetID}), true
}

func safeNotoriety(c *model.UserCriminality) int {
	if c == nil {
		return 0
	}
	return c.Notoriety
}

// TrackProgress runs a clue-gathering step for the hunt.
func (svc *Service) TrackProgress(hunterID, targetID int64, lang string) (string, bool) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	clueKeys := []string{
		"bootprint",
		"hooded_figure",
		"discarded",
		"coins",
		"sewers",
		"tavern",
		"cloth",
		"dog",
	}

	// Random chance to find the target (30% base)
	found := rng.Intn(100) < 30
	if found {
		return svc.T(lang, "hunt.track_found", map[string]any{"target": targetID}), true
	}

	clueKey := clueKeys[rng.Intn(len(clueKeys))]
	clue := svc.T(lang, "clue."+clueKey)
	return svc.T(lang, "hunt.track_clue", map[string]any{"clue": clue}), false
}

// EngageHunt resolves a PvP hunt confrontation using dice-roll combat.
func (svc *Service) EngageHunt(hunterID, targetID int64, capture bool, lang string) *HuntResult {
	result := &HuntResult{}

	hunterChar, _ := svc.store.EnsureCharacter(hunterID)
	targetChar, _ := svc.store.EnsureCharacter(targetID)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	hunterPower := hunterChar.STR + hunterChar.DEX + hunterChar.VIT + rng.Intn(20)
	targetPower := targetChar.STR + targetChar.DEX + targetChar.VIT + rng.Intn(20)

	// Pet bonus — each side gets pet power
	hunterPets := svc.getTotalPetPower(hunterID)
	targetPets := svc.getTotalPetPower(targetID)
	hunterPower += hunterPets
	targetPower += targetPets

	if hunterPower > targetPower {
		// Hunter wins
		merit := 5
		gold := 100

		// Bonus merit for capture
		if capture {
			merit += 5
			// Send target to prison for 24h
			prisonUntil := time.Now().Add(24 * time.Hour)
			svc.store.UpdateCriminality(targetID, map[string]any{"prison_until": prisonUntil})
			// Clear stolen gold from recent thefts (find and remove)
			recentThefts, _ := svc.store.GetTheftRecordsByThief(targetID)
			for _, t := range recentThefts {
				if t.Success && t.StolenGold > 0 {
					svc.store.UpdateBalance(t.VictimID, t.StolenGold)
					svc.store.UpdateBalance(targetID, -t.StolenGold)
					break
				}
			}
		}

		// Apply merit
		crim, _ := svc.store.GetCriminality(hunterID)
		svc.store.UpdateCriminality(hunterID, map[string]any{"hunter_merit": crim.HunterMerit + merit})
		svc.store.UpdateBalance(hunterID, gold)

		// Collect bounties
		bounties, _ := svc.store.GetActiveBountiesForTarget(targetID)
		totalBounty := 0
		for _, b := range bounties {
			totalBounty += b.Amount
			svc.store.UpdateBalance(hunterID, b.Amount)
			svc.store.ClaimBounty(b.ID, hunterID)
		}

		svc.store.AddNotoriety(targetID, -10)
		svc.store.AddCrimeRecord(hunterID, "hunt_won",
			fmt.Sprintf(`{"target_id":%d,"merit":%d,"gold":%d,"bounty":%d,"captured":%v}`, targetID, merit, gold, totalBounty, capture))
		svc.store.AddCrimeRecord(targetID, "hunt_lost",
			fmt.Sprintf(`{"hunter_id":%d,"captured":%v}`, hunterID, capture))

		result.Success = true
		result.MeritGained = merit
		result.GoldReward = gold + totalBounty
		result.Captured = capture

		if capture {
			result.Message = svc.T(lang, "hunt.win_capture", map[string]any{"target": targetID, "merit": merit, "gold": gold, "bounty": totalBounty})
		} else {
			result.Message = svc.T(lang, "hunt.win_defeat", map[string]any{"target": targetID, "merit": merit, "gold": gold, "bounty": totalBounty})
		}
		return result
	}

	// Criminal wins
	svc.store.AddCrimeRecord(hunterID, "hunt_lost",
		fmt.Sprintf(`{"target_id":%d}`, targetID))
	svc.store.AddCrimeRecord(targetID, "hunt_won",
		fmt.Sprintf(`{"hunter_id":%d}`, hunterID))

	result.Success = false
	result.Message = svc.T(lang, "hunt.lose", map[string]any{"target": targetID})
	return result
}

func (svc *Service) getTotalPetPower(userID int64) int {
	var pets []model.UserPet
	svc.store.DB.Where("user_id = ? AND is_active = ?", userID, true).Find(&pets)
	power := 0
	for _, p := range pets {
		power += (p.Atk + p.Defense + p.HP/10) / 3
	}
	return power
}
