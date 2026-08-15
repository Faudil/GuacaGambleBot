package delve

import (
	"fmt"
	"strings"

	"guacagamblebot/internal/i18n"
	charsvc "guacagamblebot/internal/service/character"
	"guacagamblebot/internal/store"
)

type CombatAbility struct {
	ID          string
	Name        string
	Emoji       string
	Description string
	UnlockLevel int
	ManaCost    int
}

type AbilityStatus struct {
	Ability  CombatAbility
	Unlocked bool
}

var combatAbilities = []CombatAbility{
	{
		ID: "slash", Name: "Slash", Emoji: "⚔️",
		Description: "A basic attack with your weapon or fists.",
		UnlockLevel: 1, ManaCost: 0,
	},
	{
		ID: "defend", Name: "Defend", Emoji: "🛡️",
		Description: "Brace yourself and halve the enemy's next strike.",
		UnlockLevel: 1, ManaCost: 0,
	},
	{
		ID: "fireball", Name: "Fireball", Emoji: "🔥",
		Description: "Unleash a fiery blast that ignores armor.",
		UnlockLevel: 3, ManaCost: 15,
	},
	{
		ID: "power_blow", Name: "Power Blow", Emoji: "💥",
		Description: "A devastating strike dealing double damage. If the enemy survives, their counter hits harder.",
		UnlockLevel: 6, ManaCost: 10,
	},
	{
		ID: "mend", Name: "Mend", Emoji: "💚",
		Description: "Restore 25% of your max HP. Does not deal damage.",
		UnlockLevel: 9, ManaCost: 20,
	},
}

func GetCombatAbilities(playerLevel int) []AbilityStatus {
	out := make([]AbilityStatus, len(combatAbilities))
	for i, a := range combatAbilities {
		out[i] = AbilityStatus{
			Ability:  a,
			Unlocked: playerLevel >= a.UnlockLevel,
		}
	}
	return out
}

func GetWeaponDisplay(s *store.Store, userID int64) (emoji, name string) {
	equipped, err := s.GetEquipped(userID)
	if err != nil {
		return "👊", "Punch"
	}
	for _, eq := range equipped {
		if eq.EquipSlot == "weapon" {
			if eq.Emoji != "" {
				return eq.Emoji, "Swing"
			}
			return "🗡️", "Swing"
		}
	}
	return "👊", "Punch"
}

func TranslateWeaponName(name, lang string) string {
	key := "delve.weapon." + strings.ToLower(name)
	tr := i18n.T(key, lang)
	if tr == key {
		return name
	}
	return tr
}

func EffectiveAtk(s *store.Store, userID int64) int {
	stats, err := charsvc.GetEffectiveStats(s, userID)
	if err != nil {
		return 10 + 5*2
	}
	return 10 + stats.TotalSTR()*2
}

func EffectiveDEX(s *store.Store, userID int64) int {
	stats, err := charsvc.GetEffectiveStats(s, userID)
	if err != nil {
		return 5
	}
	return stats.TotalDEX()
}

func EffectiveINT(s *store.Store, userID int64) int {
	stats, err := charsvc.GetEffectiveStats(s, userID)
	if err != nil {
		return 5
	}
	return stats.TotalINT()
}

func WeaponLabel(s *store.Store, userID int64, lang string) string {
	emoji, name := GetWeaponDisplay(s, userID)
	return fmt.Sprintf("%s %s", emoji, TranslateWeaponName(name, lang))
}
