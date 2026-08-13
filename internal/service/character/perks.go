package character

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

// Perk defines a level-up choice offered Skyrim-style at each level-up.
type Perk struct {
	ID          string
	Name        string
	Emoji       string
	Description string
	Passive     bool
}

var allPerks = []Perk{
	{ID: "perk_str", Name: "Strength", Emoji: "💪", Description: "Gain +2 permanent STR."},
	{ID: "perk_dex", Name: "Dexterity", Emoji: "🏹", Description: "Gain +2 permanent DEX."},
	{ID: "perk_int", Name: "Intelligence", Emoji: "🔮", Description: "Gain +2 permanent INT."},
	{ID: "perk_vit", Name: "Vitality", Emoji: "❤️", Description: "Gain +2 permanent VIT."},
	{ID: "perk_luk", Name: "Luck", Emoji: "🍀", Description: "Gain +2 permanent LUK."},
	{ID: "perk_gold", Name: "Gold Stash", Emoji: "💰", Description: "Receive gold equal to 20x your level."},
	{ID: "perk_crown", Name: "Crown Gift", Emoji: "👑", Description: "Receive 1 crown."},
	{ID: "perk_egg", Name: "Mystery Egg", Emoji: "🥚", Description: "Receive a random pet egg."},

	{ID: "perk_mine_yield", Name: "Master Miner", Emoji: "⛏️", Description: "+5% mining yield.", Passive: true},
	{ID: "perk_collapse_resist", Name: "Deep Roots", Emoji: "🪨", Description: "-5% mining collapse risk.", Passive: true},
	{ID: "perk_xp_boost", Name: "Quick Mind", Emoji: "📚", Description: "+5% character XP.", Passive: true},
	{ID: "perk_casino_edge", Name: "Card Sharp", Emoji: "🎰", Description: "+1% slots/coinflip win chance.", Passive: true},
	{ID: "perk_rare_find", Name: "Treasure Hunter", Emoji: "💎", Description: "+2% rare drops.", Passive: true},
	{ID: "perk_green_thumb", Name: "Green Thumb", Emoji: "🌱", Description: "-10% crop grow time.", Passive: true},
	{ID: "perk_trader", Name: "Silver Tongue", Emoji: "💵", Description: "+5% market sale price.", Passive: true},
	{ID: "perk_pet_whisperer", Name: "Pet Whisperer", Emoji: "🐾", Description: "+3 bond per pet feed.", Passive: true},
}

var perkByID = func() map[string]Perk {
	m := make(map[string]Perk, len(allPerks))
	for _, p := range allPerks {
		m[p.ID] = p
	}
	return m
}()

// GetPerk returns a perk definition by ID.
func GetPerk(id string) (Perk, bool) {
	p, ok := perkByID[id]
	return p, ok
}

// AllPerks returns a copy of the perk pool.
func AllPerks() []Perk {
	out := make([]Perk, len(allPerks))
	copy(out, allPerks)
	return out
}

// RollPerkChoices returns up to 3 random eligible perks for a character.
// Already-owned passives are excluded so each passive can only be taken once.
func RollPerkChoices(c *model.UserCharacter) []Perk {
	var owned []string
	_ = json.Unmarshal([]byte(c.Passives), &owned)
	ownedSet := make(map[string]bool, len(owned))
	for _, o := range owned {
		ownedSet[o] = true
	}
	var pool []Perk
	for _, p := range allPerks {
		if p.Passive && ownedSet[p.ID] {
			continue
		}
		pool = append(pool, p)
	}
	if len(pool) == 0 {
		return nil
	}
	perm := rand.Perm(len(pool))
	choices := make([]Perk, 0, 3)
	for _, idx := range perm {
		choices = append(choices, pool[idx])
		if len(choices) >= 3 {
			break
		}
	}
	return choices
}

// ApplyPerk consumes one perk point and applies the chosen perk, returning a
// short confirmation message.
func ApplyPerk(s *store.Store, userID int64, perkID string) (string, error) {
	p, ok := GetPerk(perkID)
	if !ok {
		return "", fmt.Errorf("unknown perk %q", perkID)
	}
	if err := s.DecrementPerkPoints(userID); err != nil {
		return "", err
	}
	if p.Passive {
		if err := s.AddPassive(userID, perkID); err != nil {
			return "", err
		}
		return p.Emoji + " **" + p.Name + "** unlocked! " + p.Description, nil
	}

	switch p.ID {
	case "perk_str", "perk_dex", "perk_int", "perk_vit", "perk_luk":
		stat := p.ID[len("perk_"):]
		if err := s.AddStatPoints(userID, stat, 2); err != nil {
			return "", err
		}
		return p.Emoji + " **+2 " + statLabel(stat) + "** granted!", nil
	case "perk_gold":
		c, err := s.GetCharacter(userID)
		if err != nil {
			return "", err
		}
		gold := 20 * c.Level
		if _, err := s.UpdateBalance(userID, gold); err != nil {
			return "", err
		}
		return p.Emoji + " **$" + itoa(gold) + "** added to your wallet!", nil
	case "perk_crown":
		if _, err := s.AdjustColumn(userID, "crowns", 1); err != nil {
			return "", err
		}
		return p.Emoji + " **+1 crown** granted!", nil
	case "perk_egg":
		egg := randomEgg()
		if err := s.AddItemRaw(s.DB, userID, egg, 1); err != nil {
			return "", err
		}
		return p.Emoji + " **" + eggName(egg) + "** added to your inventory!", nil
	}
	return "", fmt.Errorf("unknown perk %q", perkID)
}

func statLabel(stat string) string {
	switch stat {
	case "str":
		return "STR"
	case "dex":
		return "DEX"
	case "int":
		return "INT"
	case "vit":
		return "VIT"
	case "luk":
		return "LUK"
	}
	return stat
}

var eggs = []string{"forest_egg", "cave_egg", "desert_egg", "mountain_egg", "ocean_egg", "tundra_egg", "volcano_egg"}

func randomEgg() string {
	return eggs[rand.Intn(len(eggs))]
}

func eggName(id string) string {
	names := map[string]string{
		"forest_egg": "Forest Egg", "cave_egg": "Cave Egg", "desert_egg": "Desert Egg",
		"mountain_egg": "Mountain Egg", "ocean_egg": "Ocean Egg", "tundra_egg": "Tundra Egg",
		"volcano_egg": "Volcano Egg",
	}
	if n, ok := names[id]; ok {
		return n
	}
	return id
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
