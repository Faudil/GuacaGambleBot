package criminality

import (
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"

	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
)

type StealResult struct {
	Success       bool
	GoldStolen    int
	NotorietyGain int
	Message       string
	Exposed       bool
}

type BurgleResult struct {
	Success       bool
	ItemName      string
	NotorietyGain int
	Message       string
	IsMajor       bool
}

// AttemptPickpocket tries to steal gold from a target.
func (svc *Service) AttemptPickpocket(thiefID, targetID, serverID int64, lang string) *StealResult {
	result := &StealResult{}

	// Check level
	thiefLvl, err := svc.store.CharacterLevel(thiefID)
	if err != nil || thiefLvl < svc.cfg.MinLevelToTarget {
		result.Message = svc.T(lang, "steal.level_too_low", map[string]any{"level": svc.cfg.MinLevelToTarget})
		return result
	}
	targetLvl, err := svc.store.CharacterLevel(targetID)
	if err != nil || targetLvl < svc.cfg.MinLevelToTarget {
		result.Message = svc.T(lang, "steal.target_too_low", map[string]any{"level": svc.cfg.MinLevelToTarget})
		return result
	}

	// Check cooldowns
	ok, err := svc.store.CheckCooldown(thiefID, "steal_"+fmt.Sprint(targetID), time.Duration(svc.cfg.StealCooldownHours)*time.Hour)
	if err != nil || !ok {
		result.Message = svc.T(lang, "steal.cooldown")
		return result
	}

	ok, _, err = svc.store.CheckGameLimit(thiefID, "steal_total", svc.cfg.StealMaxPerDay)
	if err != nil || !ok {
		result.Message = svc.T(lang, "steal.limit_reached", map[string]any{"limit": svc.cfg.StealMaxPerDay})
		return result
	}

	// Check Pacifist's Blessing on target
	targetCrim, _ := svc.store.GetCriminality(targetID)
	if targetCrim != nil && targetCrim.PacifistUntil != nil && targetCrim.PacifistUntil.After(time.Now()) {
		result.Message = svc.T(lang, "steal.pacifist_protected")
		return result
	}

	// Get stats for skill check
	thiefChar, _ := svc.store.EnsureCharacter(thiefID)
	targetChar, _ := svc.store.EnsureCharacter(targetID)

	thiefSkill := thiefChar.DEX + thiefChar.LUK/2
	targetVigilance := targetChar.INT + targetChar.LUK/2

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	roll := rng.Intn(100)
	successChance := 40 + thiefSkill - targetVigilance
	if successChance > 90 {
		successChance = 90
	}
	if successChance < 5 {
		successChance = 5
	}

	// Apply steal cooldown and increment limit
	svc.store.SetCooldown(thiefID, "steal_"+fmt.Sprint(targetID))
	svc.store.IncrementGameLimit(thiefID, "steal_total")

	if roll < successChance {
		// Success
		targetBal, _ := svc.store.GetBalance(targetID)
		stolen := int(float64(targetBal) * svc.cfg.StealMaxGoldPercent)
		if stolen < 1 {
			stolen = 1
		}
		if stolen > 2000 {
			stolen = 2000
		}
		if stolen > targetBal {
			stolen = targetBal
		}

		if stolen > 0 {
			svc.store.UpdateBalance(thiefID, stolen)
			svc.store.UpdateBalance(targetID, -stolen)
		}

		notoriety := 5 + stolen/100
		if notoriety > 20 {
			notoriety = 20
		}

		svc.store.AddNotoriety(thiefID, notoriety)
		svc.store.CreateTheftRecord(&model.TheftRecord{
			ThiefID:    thiefID,
			VictimID:   targetID,
			StolenGold: stolen,
			WasBurgle:  false,
			Success:    true,
			CreatedAt:  time.Now(),
		})
		svc.store.AddCrimeRecord(thiefID, "stole",
			fmt.Sprintf(`{"victim_id":%d,"gold":%d,"notoriety":%d}`, targetID, stolen, notoriety))
		svc.store.AddCrimeRecord(targetID, "was_stolen_from",
			fmt.Sprintf(`{"thief_id":%d,"gold":%d}`, thiefID, stolen))

		// Check for awakening
		ws, _ := svc.store.GetWorldState(serverID)
		if ws != nil && !ws.Awakened {
			announcement := svc.OnFirstTheft(thiefID, targetID, serverID, lang)
			if announcement != nil {
				_ = announcement // caller needs to send this
			}
		}

		result.Success = true
		result.GoldStolen = stolen
		result.NotorietyGain = notoriety
		result.Message = svc.T(lang, "steal.success", map[string]any{"gold": stolen, "target": targetID})
		return result
	}

	// Failure
	notoriety := 5
	svc.store.AddNotoriety(thiefID, notoriety)
	svc.store.CreateTheftRecord(&model.TheftRecord{
		ThiefID:   thiefID,
		VictimID:  targetID,
		WasBurgle: false,
		Success:   false,
		CreatedAt: time.Now(),
	})

	result.Success = false
	result.NotorietyGain = notoriety
	result.Exposed = true
	result.Message = svc.T(lang, "steal.fail", map[string]any{"target": targetID, "notoriety": notoriety})
	return result
}

// AttemptBurgle tries to steal an unequipped item from a target.
func (svc *Service) AttemptBurgle(thiefID, targetID, serverID int64, lang string) *BurgleResult {
	result := &BurgleResult{}

	// Check level
	thiefLvl, err := svc.store.CharacterLevel(thiefID)
	if err != nil || thiefLvl < svc.cfg.MinLevelToTarget {
		result.Message = svc.T(lang, "burgle.level_too_low", map[string]any{"level": svc.cfg.MinLevelToTarget})
		return result
	}
	targetLvl, _ := svc.store.CharacterLevel(targetID)
	if targetLvl < svc.cfg.MinLevelToTarget {
		result.Message = svc.T(lang, "burgle.target_too_low")
		return result
	}

	// Check burgle rank (need Burglar = rank 1+)
	crim, _ := svc.store.GetCriminality(thiefID)
	if crim.ThiefRank < 1 {
		result.Message = svc.T(lang, "burgle.need_rank")
		return result
	}

	// Check cooldown
	ok, err := svc.store.CheckCooldown(thiefID, "burgle_"+fmt.Sprint(targetID), time.Duration(svc.cfg.BurgleCooldownDays)*24*time.Hour)
	if err != nil || !ok {
		result.Message = svc.T(lang, "burgle.cooldown")
		return result
	}

	// Check Pacifist
	targetCrim, _ := svc.store.GetCriminality(targetID)
	if targetCrim != nil && targetCrim.PacifistUntil != nil && targetCrim.PacifistUntil.After(time.Now()) {
		result.Message = svc.T(lang, "burgle.pacifist_protected")
		return result
	}

	// Check the thief has room to carry a stolen item
	free, err := svc.store.FreeSlots(svc.store.DB, thiefID)
	if err != nil {
		result.Message = svc.T(lang, "burgle.error")
		return result
	}
	if free <= 0 {
		result.Message = svc.T(lang, "burgle.bag_full")
		return result
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Get target's inventory to find a stealable item (equipment is in UserEquipment, not Inventory)
	var inv []model.Inventory
	svc.store.DB.Where("user_id = ?", targetID).Find(&inv)

	if len(inv) == 0 {
		result.Message = svc.T(lang, "burgle.nothing_worth")
		return result
	}

	// Filter to Common/Uncommon items
	var stealable []model.Inventory
	for _, entry := range inv {
		it := items.Get(entry.ItemID)
		if it != nil && (it.Rarity == items.RarityCommon || it.Rarity == items.RarityUncommon) {
			stealable = append(stealable, entry)
		}
	}

	// Major theft (Rare+) detection
	var majorTarget []model.Inventory
	for _, entry := range inv {
		it := items.Get(entry.ItemID)
		if it != nil && it.Rarity == items.RarityRare || it.Rarity == items.RarityEpic || it.Rarity == items.RarityLegendary {
			majorTarget = append(majorTarget, entry)
		}
	}

	if len(stealable) == 0 && len(majorTarget) == 0 {
		result.Message = svc.T(lang, "burgle.nothing_worth")
		return result
	}

	// Set cooldown
	svc.store.SetCooldown(thiefID, "burgle_"+fmt.Sprint(targetID))

	// Skill check — burgle is harder
	thiefChar, _ := svc.store.EnsureCharacter(thiefID)
	targetChar, _ := svc.store.EnsureCharacter(targetID)
	thiefSkill := thiefChar.DEX + thiefChar.LUK/3
	targetVigilance := targetChar.INT + targetChar.LUK/2
	successChance := 25 + thiefSkill - targetVigilance
	if successChance > 75 {
		successChance = 75
	}
	if successChance < 2 {
		successChance = 2
	}

	roll := rng.Intn(100)
	if roll >= successChance {
		notoriety := 10
		svc.store.AddNotoriety(thiefID, notoriety)
		result.Success = false
		result.NotorietyGain = notoriety
		result.Message = svc.T(lang, "burgle.fail", map[string]any{"target": targetID, "notoriety": notoriety})
		return result
	}

	// Success — pick a random stealable item, or a major one if we want risk
	var targetItem model.Inventory
	notorietyBase := 10

	if len(majorTarget) > 0 && rng.Intn(100) < 20 {
		// Major theft attempt
		result.IsMajor = true
		notorietyBase = 30
		targetItem = majorTarget[rng.Intn(len(majorTarget))]
	} else if len(stealable) > 0 {
		targetItem = stealable[rng.Intn(len(stealable))]
	} else {
		result.Message = svc.T(lang, "burgle.nothing_worth_alt")
		return result
	}

	it := items.Get(targetItem.ItemID)
	if it == nil {
		result.Message = svc.T(lang, "burgle.item_gone")
		return result
	}

	// Remove from target, add to thief
	svc.store.DB.Where("user_id = ? AND item_id = ?", targetID, targetItem.ItemID).
		UpdateColumn("quantity", gorm.Expr("quantity - 1"))
	svc.store.AddItemRaw(svc.store.DB, thiefID, targetItem.ItemID, 1)

	notoriety := notorietyBase
	svc.store.AddNotoriety(thiefID, notoriety)
	svc.store.CreateTheftRecord(&model.TheftRecord{
		ThiefID:    thiefID,
		VictimID:   targetID,
		StolenItem: targetItem.ItemID,
		WasBurgle:  true,
		Success:    true,
		CreatedAt:  time.Now(),
	})
	svc.store.AddCrimeRecord(thiefID, "burgled",
		fmt.Sprintf(`{"victim_id":%d,"item":"%s","notoriety":%d}`, targetID, it.ID, notoriety))

	result.Success = true
	result.ItemName = it.Name
	result.NotorietyGain = notoriety
	result.Message = svc.T(lang, "burgle.success", map[string]any{"item": it.Name, "target": targetID, "notoriety": notoriety})
	return result
}
