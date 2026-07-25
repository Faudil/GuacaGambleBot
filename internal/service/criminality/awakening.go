package criminality

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/bwmarrin/discordgo"
)

// RequiredShadowFlags lists the delve flags that together qualify a player
// as "touched by shadow". Having N of these grants the permanent flag.
var shadowQualifyingFlags = []string{
	"betrayed_npc",
	"desecrated_altar",
	"slept_unprotected",
	"sacrificed_hp",
	"freed_prisoner",
	"ambushed_while_sleeping",
}

const (
	shadowRequiredCount = 2
	maskBossFloorMin    = 8
	maskBossFloorMax    = 10
)

// GravewardenMorvain is the special boss that drops the Mask.
var GravewardenMorvain = &EnemyModel{
	Name:     "Gravewarden Morvain",
	Emoji:    "💀",
	HP:       150,
	MaxHP:    150,
	Atk:      25,
	Def:      12,
	Zone:     "forge_district",
	IsBoss:   true,
	IsMaskBearer: true,
}

// EnemyModel is a lightweight definition for special enemies.
type EnemyModel struct {
	Name          string
	Emoji         string
	HP            int
	MaxHP         int
	Atk           int
	Def           int
	Zone          string
	IsBoss        bool
	IsMaskBearer  bool
}

// CheckAndGrantTouchedByShadow checks if a player qualifies for the
// touched_by_shadow permanent flag. Returns true if the flag was just granted.
func (svc *Service) CheckAndGrantTouchedByShadow(userID int64) (bool, error) {
	has, err := svc.store.HasDelveFlag(userID, "touched_by_shadow")
	if err != nil {
		return false, err
	}
	if has {
		return false, nil
	}

	flags, err := svc.store.GetDelveFlags(userID)
	if err != nil {
		return false, err
	}

	owned := make(map[string]bool, len(flags))
	for _, f := range flags {
		owned[f.FlagID] = true
	}

	count := 0
	for _, qf := range shadowQualifyingFlags {
		if owned[qf] {
			count++
		}
	}

	if count >= shadowRequiredCount {
		if err := svc.store.AddDelveFlag(userID, "touched_by_shadow", `{"source":"criminality_awakening"}`); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// CanDropMask checks if the Gravewarden can appear for this player on this floor.
func (svc *Service) CanDropMask(userID int64, floor int) (bool, error) {
	if floor < maskBossFloorMin || floor > maskBossFloorMax {
		return false, nil
	}
	has, err := svc.store.HasDelveFlag(userID, "touched_by_shadow")
	if err != nil {
		return false, err
	}
	if !has {
		return false, nil
	}
	alreadyHasMask, err := svc.store.HasDelveFlag(userID, "mask_of_malveillance")
	if err != nil {
		return false, err
	}
	return !alreadyHasMask, nil
}

// IsAwakened returns whether the criminality system is active on this server.
func (svc *Service) IsAwakened(serverID int64) (bool, error) {
	return svc.store.IsAwakened(serverID)
}

// IsCommandAllowed checks if a specific criminal command may be used.
// Returns (allowed, reason).
func (svc *Service) IsCommandAllowed(userID int64, serverID int64, command string, lang string) (bool, string, error) {
	awake, err := svc.IsAwakened(serverID)
	if err != nil {
		return false, "", err
	}
	if !awake {
		return false, svc.T(lang, "shadow.not_awake"), nil
	}

	crim, err := svc.store.GetCriminality(userID)
	if err != nil {
		return false, "", err
	}

	switch command {
	case "steal", "burgle":
		if crim.Alignment == "thief" || crim.HasMask {
			return true, "", nil
		}
		return false, svc.T(lang, "shadow.not_thief"), nil
	case "bounty", "hunt", "track":
		if crim.Alignment == "hunter" {
			return true, "", nil
		}
		return false, svc.T(lang, "shadow.not_hunter"), nil
	default:
		return true, "", nil
	}
}

// OnFirstMaskEquip handles the first equipping of the Mask of Malveillance.
// Sends a global announcement. Returns the announcement message if any.
func (svc *Service) OnFirstMaskEquip(userID int64, serverID int64, lang string) *discordgo.MessageEmbed {
	ws, err := svc.store.GetWorldState(serverID)
	if err != nil {
		return nil
	}
	if ws.MaskClaimedBy != nil {
		return nil
	}

	ws.MaskClaimedBy = &userID
	now := time.Now()
	ws.MaskClaimedAt = &now
	svc.store.SaveWorldState(ws)

	svc.store.AddCrimeRecord(userID, "mask_claimed",
		`{"server_id":`+fmt.Sprint(serverID)+`}`)

	return &discordgo.MessageEmbed{
		Title:       svc.T(lang, "awakening.mask_title"),
		Description: svc.T(lang, "awakening.mask_desc"),
		Color:       0x2c3e50,
		Footer: &discordgo.MessageEmbedFooter{
			Text: svc.T(lang, "awakening.mask_footer"),
		},
	}
}

// OnFirstTheft handles the world's first theft after awakening.
// Returns the announcement embed and the victim quest info.
func (svc *Service) OnFirstTheft(thiefID int64, victimID int64, serverID int64, lang string) *discordgo.MessageEmbed {
	ws, err := svc.store.GetWorldState(serverID)
	if err != nil || ws.Awakened {
		return nil
	}

	ws.Awakened = true
	ws.FirstThiefID = &thiefID
	ws.FirstVictimID = &victimID
	now := time.Now()
	ws.AwakenedAt = &now
	svc.store.SaveWorldState(ws)

	svc.store.AddCrimeRecord(thiefID, "awakening_first_theft",
		fmt.Sprintf(`{"victim_id":%d,"server_id":%d}`, victimID, serverID))
	svc.store.AddCrimeRecord(victimID, "awakening_first_victim",
		fmt.Sprintf(`{"thief_id":%d,"server_id":%d}`, thiefID, serverID))

	return &discordgo.MessageEmbed{
		Title:       svc.T(lang, "awakening.first_theft_title"),
		Description: svc.T(lang, "awakening.first_theft_desc"),
		Color:       0x8e44ad,
		Footer: &discordgo.MessageEmbedFooter{
			Text: svc.T(lang, "awakening.first_theft_footer"),
		},
	}
}

// CheckGravewardenSpawn checks whether the Gravewarden should appear in
// this delve room. Returns a special enemy if the conditions are met, nil otherwise.
func (svc *Service) CheckGravewardenSpawn(userID int64, floor int, rngSeed int64) (enemyName string, enemyHP int, enemyMaxHP int, enemyAtk int, enemyDef int, shouldSpawn bool) {
	if floor < maskBossFloorMin || floor > maskBossFloorMax {
		return "", 0, 0, 0, 0, false
	}
	has, err := svc.store.HasDelveFlag(userID, "touched_by_shadow")
	if err != nil || !has {
		return "", 0, 0, 0, 0, false
	}
	alreadyHas, err := svc.store.HasDelveFlag(userID, "mask_of_malveillance")
	if err != nil || alreadyHas {
		return "", 0, 0, 0, 0, false
	}
	// 35% spawn chance when conditions are met
	rng := rand.New(rand.NewSource(rngSeed))
	if rng.Intn(100) >= 35 {
		return "", 0, 0, 0, 0, false
	}
	return GravewardenMorvain.Name, GravewardenMorvain.HP, GravewardenMorvain.MaxHP, GravewardenMorvain.Atk, GravewardenMorvain.Def, true
}

// GrantMaskToPlayer records that a player has obtained the Mask.
func (svc *Service) GrantMaskToPlayer(userID int64, serverID int64, lang string) (*discordgo.MessageEmbed, error) {
	if err := svc.store.AddDelveFlag(userID, "mask_of_malveillance", `{"source":"gravewarden_morvain"}`); err != nil {
		return nil, err
	}
	if err := svc.store.UpdateCriminality(userID, map[string]any{"has_mask": true}); err != nil {
		return nil, err
	}
	svc.store.AddCrimeRecord(userID, "mask_claimed",
		fmt.Sprintf(`{"server_id":%d,"source":"gravewarden_morvain"}`, serverID))
	announcement := svc.OnFirstMaskEquip(userID, serverID, lang)
	return announcement, nil
}
