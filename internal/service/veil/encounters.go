package veil

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type EncounterResult struct {
	Complete    bool
	NextPhase   string
	Description string
	PublicEmbed *discordgo.MessageEmbed
	Comps       []discordgo.MessageComponent
}

func StartEncounter(raid *model.VeilRaid, lang string) EncounterResult {
	switch raid.Phase {
	case "whispering":
		return startWhisperingGallery(raid, lang)
	case "shifting_flames":
		return startShiftingFlames(raid, lang)
	case "guardian":
		return startGuardian(raid, lang)
	case "breach":
		return startBreachCore(raid, lang)
	default:
		return EncounterResult{Complete: true, NextPhase: "boss_p1"}
	}
}

func startWhisperingGallery(raid *model.VeilRaid, lang string) EncounterResult {
	return EncounterResult{
		Complete: false,
		PublicEmbed: &discordgo.MessageEmbed{
			Title:       i18n.T("veil.encounter.whisper_title", lang),
			Description: i18n.T("veil.encounter.whisper_desc", lang),
			Color:       0x9b59b6,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Hint", Value: i18n.T("veil.encounter.whisper_hint", lang)},
			},
		},
		Comps: []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("veil.encounter.whisper_btn", lang), components.Encode("veil", "whisper_answer"), discordgo.PrimaryButton),
			),
		},
	}
}

var whisperPool = []struct {
	Fragments []string
	Answer    string
}{
	{
		Fragments: []string{"the forgotten", "king awakens", "at dawn's", "first light"},
		Answer:    "theforgottenkingawakensatdawnsfirstlight",
	},
	{
		Fragments: []string{"speak now", "the name", "of the", "veil warden"},
		Answer:    "speaknowthenameoftheveilwarden",
	},
	{
		Fragments: []string{"darkness", "recedes when", "the flame", "is kindled within"},
		Answer:    "darknessrecedeswhentheflameiskindledwithin",
	},
}

func GetWhisperFragments(raid *model.VeilRaid) (map[int64]string, string) {
	idx := int(raid.ID) % len(whisperPool)
	pool := whisperPool[idx]
	participants := (&Service{}).GetParticipants(raid)
	assignments := map[int64]string{}
	for i, pid := range participants {
		if i < len(pool.Fragments) {
			assignments[pid] = pool.Fragments[i]
		}
	}
	return assignments, pool.Answer
}

func CheckWhisperAnswer(raid *model.VeilRaid, answer string, lang string) (bool, string) {
	_, correct := GetWhisperFragments(raid)
	if strings.ToLower(strings.TrimSpace(answer)) == correct {
		return true, i18n.T("veil.encounter.whisper_success", lang)
	}
	return false, i18n.T("veil.encounter.whisper_fail", lang)
}

func startShiftingFlames(raid *model.VeilRaid, lang string) EncounterResult {
	return EncounterResult{
		Complete: false,
		PublicEmbed: &discordgo.MessageEmbed{
			Title:       i18n.T("veil.encounter.flames_title", lang),
			Description: i18n.T("veil.encounter.flames_desc", lang),
			Color:       0xe67e22,
		},
		Comps: []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("veil.encounter.flames_btn_protect", lang), components.Encode("veil", "flame_protect"), discordgo.PrimaryButton),
				components.Button(i18n.T("veil.encounter.flames_btn_extinguish", lang), components.Encode("veil", "flame_extinguish"), discordgo.SuccessButton),
				components.Button(i18n.T("veil.encounter.flames_btn_scout", lang), components.Encode("veil", "flame_scout"), discordgo.SecondaryButton),
			),
		},
	}
}

type FlameState struct {
	AP          int
	Intensity   int
	ExitReveal  int
	EngulfedBy  int64
}

func ProcessFlameTurn(mech string, actions map[int64]string, lang string) (bool, string, string) {
	var fs FlameState
	store.UnmarshalJSON(mech, &fs)
	if fs.AP == 0 {
		fs.AP = 5
	}

	desc := ""
	for userID, action := range actions {
		switch action {
		case "flame_protect":
			if fs.AP > 0 {
				fs.AP--
				fs.EngulfedBy = 0
				desc += i18n.T("veil.flame.protect", lang, map[string]any{"user": strconv.FormatInt(userID, 10)})
			}
		case "flame_extinguish":
			if fs.AP > 0 {
				fs.AP--
				fs.Intensity--
				if fs.Intensity < 0 {
					fs.Intensity = 0
				}
				desc += i18n.T("veil.flame.extinguish", lang, map[string]any{"user": strconv.FormatInt(userID, 10), "intensity": fs.Intensity})
			}
		case "flame_scout":
			if fs.AP >= 2 {
				fs.AP -= 2
				fs.ExitReveal++
				desc += i18n.T("veil.flame.scout", lang, map[string]any{"user": strconv.FormatInt(userID, 10), "reveal": fs.ExitReveal})
			}
		}
	}

	fs.Intensity++
	if fs.EngulfedBy == 0 {
		fs.EngulfedBy = int64(rand.Intn(100))
	}

	desc += i18n.T("veil.flame.status", lang, map[string]any{"intensity": fs.Intensity, "reveal": fs.ExitReveal, "ap": fs.AP})

	complete := fs.ExitReveal >= 3

	b, _ := json.Marshal(fs)
	return complete, desc, string(b)
}

func startGuardian(raid *model.VeilRaid, lang string) EncounterResult {
	return EncounterResult{
		Complete: false,
		PublicEmbed: &discordgo.MessageEmbed{
			Title:       i18n.T("veil.encounter.guardian_title", lang),
			Description: i18n.T("veil.encounter.guardian_desc", lang),
			Color:       0x3498db,
			Fields: []*discordgo.MessageEmbedField{
				{Name: i18n.T("veil.encounter.guardian_hp", lang), Value: i18n.T("veil.encounter.guardian_hp_val", lang)},
			},
		},
		Comps: []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("veil.encounter.guardian_btn_atk", lang), components.Encode("veil", "guard_atk"), discordgo.DangerButton),
				components.Button(i18n.T("veil.encounter.guardian_btn_prot", lang), components.Encode("veil", "guard_prot"), discordgo.PrimaryButton),
				components.Button(i18n.T("veil.encounter.guardian_btn_skip", lang), components.Encode("veil", "guard_skip"), discordgo.SecondaryButton),
			),
		},
	}
}

func ProcessGuardianTurn(mech string, actions map[int64]string, lang string) (bool, string, string) {
	type GuardState struct {
		HP      int
		Turn    int
		TopDmg  int64
		TopAmt  int
		Protect int
	}
	var gs GuardState
	store.UnmarshalJSON(mech, &gs)
	if gs.HP == 0 {
		gs.HP = 800
	}

	desc := ""
	gs.Turn++

	for userID, action := range actions {
		switch action {
		case "guard_atk":
			dmg := 40 + rand.Intn(20)
			gs.HP -= dmg
			desc += i18n.T("veil.guardian.atk", lang, map[string]any{"user": strconv.FormatInt(userID, 10), "dmg": dmg})
			if dmg > gs.TopAmt {
				gs.TopAmt = dmg
				gs.TopDmg = userID
			}
		case "guard_prot":
			gs.Protect++
			desc += i18n.T("veil.guardian.prot", lang, map[string]any{"user": strconv.FormatInt(userID, 10), "protectors": gs.Protect})
		}
	}

	if gs.Turn%3 == 0 && gs.TopDmg > 0 {
		if gs.Protect < 2 {
			desc += i18n.T("veil.guardian.chrono_fail", lang, map[string]any{"target": strconv.FormatInt(gs.TopDmg, 10)})
		} else {
			desc += i18n.T("veil.guardian.chrono_block", lang, map[string]any{"target": strconv.FormatInt(gs.TopDmg, 10)})
		}
		gs.Protect = 0
		gs.TopDmg = 0
		gs.TopAmt = 0
	}

	desc += i18n.T("veil.guardian.status", lang, map[string]any{"hp": gs.HP, "turn": gs.Turn})
	complete := gs.HP <= 0

	b, _ := json.Marshal(gs)
	return complete, desc, string(b)
}

func startBreachCore(raid *model.VeilRaid, lang string) EncounterResult {
	return EncounterResult{
		Complete: false,
		PublicEmbed: &discordgo.MessageEmbed{
			Title:       i18n.T("veil.encounter.breach_title", lang),
			Description: i18n.T("veil.encounter.breach_desc", lang),
			Color:       0x2ecc71,
			Fields: []*discordgo.MessageEmbedField{
				{Name: i18n.T("veil.encounter.breach_field_awe_name", lang), Value: i18n.T("veil.encounter.breach_field_awe_val", lang)},
				{Name: i18n.T("veil.encounter.breach_field_defy_name", lang), Value: i18n.T("veil.encounter.breach_field_defy_val", lang)},
				{Name: i18n.T("veil.encounter.breach_field_fear_name", lang), Value: i18n.T("veil.encounter.breach_field_fear_val", lang)},
			},
		},
		Comps: []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("veil.encounter.breach_btn_awe", lang), components.Encode("veil", "breach_awe"), discordgo.PrimaryButton),
				components.Button(i18n.T("veil.encounter.breach_btn_defy", lang), components.Encode("veil", "breach_defy"), discordgo.DangerButton),
				components.Button(i18n.T("veil.encounter.breach_btn_fear", lang), components.Encode("veil", "breach_fear"), discordgo.SecondaryButton),
			),
		},
	}
}

func GetBreachBoon(votes map[string]int, lang string) string {
	maxVotes := 0
	boon := ""
	for k, v := range votes {
		if v > maxVotes {
			maxVotes = v
			boon = k
		}
	}
	switch boon {
	case "awe":
		return i18n.T("veil.encounter.breach_boon_awe", lang)
	case "defiance":
		return i18n.T("veil.encounter.breach_boon_defiance", lang)
	case "fear":
		return i18n.T("veil.encounter.breach_boon_fear", lang)
	default:
		return i18n.T("veil.encounter.breach_boon_default", lang)
	}
}
