package veil

import (
	"encoding/json"
	"math/rand"
	"strconv"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/model"
	"guacagamblebot/internal/store"
)

type BossTurnResult struct {
	EncounterDone bool
	BossDefeated  bool
	PartyWiped    bool
	Desc          string
	PhaseChanged  int
	Embed         *discordgo.MessageEmbed
	Comps         []discordgo.MessageComponent
	UpdateMech    string
}

func StartBossPhase(raid *model.VeilRaid, phase int, lang string) BossTurnResult {
	res := BossTurnResult{}
	switch phase {
	case 1:
		res = startBossPhase1(raid, lang)
	case 2:
		res = startBossPhase2(raid, lang)
	case 3:
		res = startBossPhase3(raid, lang)
	}
	return res
}

func startBossPhase1(raid *model.VeilRaid, lang string) BossTurnResult {
	raid.BossPhase = 1
	addCount := 2
	if len((&Service{}).GetParticipants(raid)) < 4 {
		addCount = 1
	}
	adds := make([]int, addCount)
	for i := range adds {
		adds[i] = 200
	}
	b, _ := json.Marshal(adds)
	raid.AddHP = string(b)

	return BossTurnResult{
		Embed: &discordgo.MessageEmbed{
			Title:       i18n.T("veil.boss.p1_title", lang),
			Description: i18n.T("veil.boss.p1_desc", lang),
			Color:       components.ColorArcane,
			Fields: []*discordgo.MessageEmbedField{
				{Name: i18n.T("veil.boss.p1_field_boss", lang), Value: i18n.T("veil.boss.p1_field_boss_val", lang, map[string]any{"hp": raid.BossHP, "max": raid.BossMaxHP})},
				{Name: i18n.T("veil.boss.p1_field_adds", lang), Value: i18n.T("veil.boss.p1_field_adds_val", lang, map[string]any{"a": adds[0], "b": adds[1]})},
			},
		},
		Comps: []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("veil.boss.p1_btn_boss", lang), components.Encode("veil", "boss_atk1"), discordgo.DangerButton),
				components.Button(i18n.T("veil.boss.p1_btn_add", lang), components.Encode("veil", "boss_add"), discordgo.SecondaryButton),
			),
		},
	}
}

func startBossPhase2(raid *model.VeilRaid, lang string) BossTurnResult {
	raid.BossPhase = 2
	return BossTurnResult{
		Embed: &discordgo.MessageEmbed{
			Title:       i18n.T("veil.boss.p2_title", lang),
			Description: i18n.T("veil.boss.p2_desc", lang),
			Color:       components.ColorDanger,
		},
		Comps: []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("veil.boss.btn_attack", lang), components.Encode("veil", "boss_atk2"), discordgo.DangerButton),
				components.Button(i18n.T("veil.boss.p2_btn_heal", lang), components.Encode("veil", "boss_heal2"), discordgo.SuccessButton),
				components.Button(i18n.T("veil.boss.p2_btn_prot", lang), components.Encode("veil", "boss_prot2"), discordgo.PrimaryButton),
			),
			components.ActionRow(
				components.Button(i18n.T("veil.boss.p2_btn_stabilize", lang), components.Encode("veil", "stabilize"), discordgo.PrimaryButton),
			),
		},
	}
}

func startBossPhase3(raid *model.VeilRaid, lang string) BossTurnResult {
	raid.BossPhase = 3
	return BossTurnResult{
		Embed: &discordgo.MessageEmbed{
			Title:       i18n.T("veil.boss.p3_title", lang),
			Description: i18n.T("veil.boss.p3_desc", lang),
			Color:       components.ColorDanger,
		},
		Comps: []discordgo.MessageComponent{
			components.ActionRow(
				components.Button(i18n.T("veil.boss.btn_attack", lang), components.Encode("veil", "boss_atk3"), discordgo.DangerButton),
				components.Button(i18n.T("veil.boss.p3_btn_heal", lang), components.Encode("veil", "boss_heal3"), discordgo.SuccessButton),
				components.Button(i18n.T("veil.boss.p3_btn_prot", lang), components.Encode("veil", "boss_prot3"), discordgo.PrimaryButton),
			),
		},
	}
}

func ResolveBossTurn(raid *model.VeilRaid, actions map[int64]string, lang string) BossTurnResult {
	res := BossTurnResult{}
	participants := (&Service{}).GetParticipants(raid)
	switch raid.BossPhase {
	case 1:
		var adds []int
		store.UnmarshalJSON(raid.AddHP, &adds)
		if len(adds) < 2 {
			adds = []int{200, 200}
		}

		addsAlive := 0
		for _, hp := range adds {
			if hp > 0 {
				addsAlive++
			}
		}

		for userID, action := range actions {
			switch action {
			case "attack_add":
				for i := range adds {
					if adds[i] > 0 {
						dmg := 60 + rand.Intn(30)
						adds[i] -= dmg
						res.Desc += i18n.T("veil.boss.p1_action_add", lang, map[string]any{"user": strconv.FormatInt(userID, 10), "add": i + 1, "dmg": dmg})
						break
					}
				}
			case "attack":
				dmg := 40 + rand.Intn(20)
				if addsAlive > 0 {
					dmg /= 2
				}
				raid.BossHP -= dmg
				res.Desc += i18n.T("veil.boss.p1_action_boss", lang, map[string]any{"user": strconv.FormatInt(userID, 10), "dmg": dmg})
			}
		}

		addsAlive = 0
		for _, hp := range adds {
			if hp > 0 {
				addsAlive++
			}
		}
		res.Desc += i18n.T("veil.boss.p1_status", lang, map[string]any{"alive": addsAlive, "hp": raid.BossHP, "max": raid.BossMaxHP})

		b, _ := json.Marshal(adds)
		raid.AddHP = string(b)

		if raid.BossHP <= raid.BossMaxHP*70/100 {
			res.PhaseChanged = 2
			res.Desc += i18n.T("veil.boss.transition_p2", lang)
		}

	case 2:
		raid.Turn++
		stabilized := 0
		for userID, action := range actions {
			switch action {
			case "stabilize":
				stabilized++
			case "attack":
				dmg := 50 + rand.Intn(30)
				raid.BossHP -= dmg
				res.Desc += i18n.T("veil.boss.p2_action_atk", lang, map[string]any{"user": strconv.FormatInt(userID, 10), "dmg": dmg})
			case "heal":
				states := (&Service{}).GetPlayerStates(raid)
				lowestHP := int64(0)
				lowestVal := 9999
				for _, ps := range states {
					if ps.HP < lowestVal && ps.HP > 0 {
						lowestVal = ps.HP
						lowestHP = ps.UserID
					}
				}
				heal := 50 + rand.Intn(30)
				if ps, ok := states[lowestHP]; ok {
					ps.HP += heal
					if ps.HP > ps.MaxHP {
						ps.HP = ps.MaxHP
					}
					states[lowestHP] = ps
				}
				res.Desc += i18n.T("veil.boss.p2_action_heal", lang, map[string]any{"user": strconv.FormatInt(userID, 10), "target": strconv.FormatInt(lowestHP, 10), "heal": heal})
			case "protect":
				res.Desc += i18n.T("veil.boss.p2_action_prot", lang, map[string]any{"user": strconv.FormatInt(userID, 10)})
			}
		}

		required := len(participants) * 60 / 100
		if raid.Turn%3 == 0 && stabilized < required {
			res.Desc += i18n.T("veil.boss.p2_tear_fail", lang, map[string]any{"actual": stabilized, "required": required})
		} else if raid.Turn%3 == 0 {
			res.Desc += i18n.T("veil.boss.p2_tear_ok", lang, map[string]any{"actual": stabilized, "required": required})
		}

		if raid.BossHP <= raid.BossMaxHP*30/100 {
			res.PhaseChanged = 3
			res.Desc += i18n.T("veil.boss.transition_p3", lang)
		}

	case 3:
		marked := participants[rand.Intn(len(participants))]
		protected := false
		vulnerable := false

		for userID, action := range actions {
			switch action {
			case "attack":
				dmg := 50 + rand.Intn(30)
				if vulnerable {
					dmg *= 2
				}
				raid.BossHP -= dmg
				res.Desc += i18n.T("veil.boss.p3_action_atk", lang, map[string]any{"user": strconv.FormatInt(userID, 10), "dmg": dmg})
			case "heal":
				if userID == marked {
					protected = true
				}
				res.Desc += i18n.T("veil.boss.p3_action_heal", lang, map[string]any{"user": strconv.FormatInt(userID, 10)})
			case "protect":
				protected = true
				res.Desc += i18n.T("veil.boss.p3_action_prot", lang, map[string]any{"user": strconv.FormatInt(userID, 10)})
			}
		}

		if !protected {
			res.Desc += i18n.T("veil.boss.p3_sacrifice_fail", lang, map[string]any{"marked": strconv.FormatInt(marked, 10)})
			raid.BossHP += 150
		} else {
			res.Desc += i18n.T("veil.boss.p3_sacrifice_ok", lang, map[string]any{"marked": strconv.FormatInt(marked, 10)})
			vulnerable = true
		}
		res.Desc += i18n.T("veil.boss.p3_boss_hp", lang, map[string]any{"hp": raid.BossHP, "max": raid.BossMaxHP})
	}

	if raid.BossHP <= 0 {
		res.BossDefeated = true
		res.Desc += i18n.T("veil.boss.victory", lang)
	}

	return res
}

func RenderBossEmbed(raid *model.VeilRaid, resultDesc string, lang string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       i18n.T("veil.boss.render_title", lang, map[string]any{"phase": raid.BossPhase, "turn": raid.Turn}),
		Description: resultDesc,
		Color:       components.ColorDanger,
	}
	states := (&Service{}).GetPlayerStates(raid)
	playerLine := ""
	for _, ps := range states {
		playerLine += i18n.T("veil.boss.render_player_hp", lang, map[string]any{"user": strconv.FormatInt(ps.UserID, 10), "hp": ps.HP, "max": ps.MaxHP}) + "\n"
	}
	embed.Fields = []*discordgo.MessageEmbedField{
		{Name: i18n.T("veil.boss.render_boss_hp", lang, map[string]any{"hp": raid.BossHP, "max": raid.BossMaxHP}), Value: playerLine, Inline: false},
	}
	return embed
}

func RenderAnchorEmbed(lang string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       i18n.T("veil.boss.anchor_title", lang),
		Description: i18n.T("veil.boss.anchor_desc", lang),
		Color:       components.ColorWarning,
	}
}

func AnchorComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("⚓ Anchor!", components.Encode("veil", "anchor"), discordgo.DangerButton),
		),
	}
}
