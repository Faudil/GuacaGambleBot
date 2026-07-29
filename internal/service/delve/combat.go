package delve

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/model"
)

type Enemy struct {
	Name     string `json:"name"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"max_hp"`
	Atk      int    `json:"atk"`
	Def      int    `json:"def"`
	MinFloor int    `json:"min_floor"`
	Zone     string `json:"zone"`
	Emoji    string `json:"emoji"`
}

var enemyTemplates = []Enemy{
	{Name: "Shambling Corpse", Atk: 8, Def: 2, MinFloor: 1, Zone: "crypt", Emoji: "🧟"},
	{Name: "Skeletal Archer", Atk: 10, Def: 1, MinFloor: 1, Zone: "crypt", Emoji: "💀"},
	{Name: "Crypt Rat Swarm", Atk: 6, Def: 1, MinFloor: 1, Zone: "crypt", Emoji: "🐀"},
	{Name: "Grave Warden", Atk: 12, Def: 4, MinFloor: 2, Zone: "crypt", Emoji: "⚰️"},
	{Name: "Bone Golem", Atk: 14, Def: 6, MinFloor: 3, Zone: "crypt", Emoji: "🦴"},
	{Name: "Wailing Banshee", Atk: 16, Def: 3, MinFloor: 3, Zone: "crypt", Emoji: "👻"},
	{Name: "Spore Shambler", Atk: 10, Def: 3, MinFloor: 4, Zone: "fungal_wilds", Emoji: "🍄"},
	{Name: "Fungal Crawler", Atk: 12, Def: 4, MinFloor: 4, Zone: "fungal_wilds", Emoji: "🐛"},
	{Name: "Acid Slime", Atk: 14, Def: 2, MinFloor: 5, Zone: "fungal_wilds", Emoji: "🟢"},
	{Name: "Myconid Warrior", Atk: 16, Def: 5, MinFloor: 5, Zone: "fungal_wilds", Emoji: "🌿"},
	{Name: "Fungal Brute", Atk: 20, Def: 8, MinFloor: 6, Zone: "fungal_wilds", Emoji: "🦧"},
	{Name: "Rusted Construct", Atk: 14, Def: 6, MinFloor: 7, Zone: "forge_district", Emoji: "⚙️"},
	{Name: "Forge Wraith", Atk: 18, Def: 4, MinFloor: 7, Zone: "forge_district", Emoji: "👤"},
	{Name: "Steam Guardian", Atk: 16, Def: 8, MinFloor: 8, Zone: "forge_district", Emoji: "🗿"},
	{Name: "Magma Hound", Atk: 22, Def: 6, MinFloor: 8, Zone: "forge_district", Emoji: "🐕"},
	{Name: "Iron Titan", Atk: 25, Def: 12, MinFloor: 9, Zone: "forge_district", Emoji: "🤖"},
	{Name: "Void Stalker", Atk: 22, Def: 8, MinFloor: 10, Zone: "abyss", Emoji: "👁️"},
	{Name: "Reality Weaver", Atk: 24, Def: 10, MinFloor: 10, Zone: "abyss", Emoji: "🌀"},
	{Name: "Abyssal Hulk", Atk: 28, Def: 14, MinFloor: 11, Zone: "abyss", Emoji: "🐙"},
	{Name: "Soul Reaper", Atk: 30, Def: 10, MinFloor: 12, Zone: "abyss", Emoji: "💜"},
	{Name: "Crypt Stalker", Atk: 12, Def: 3, MinFloor: 1, Zone: "crypt", Emoji: "🕯️"},
	{Name: "Bone Wraith", Atk: 18, Def: 1, MinFloor: 3, Zone: "crypt", Emoji: "💨"},
	{Name: "Spore Host", Atk: 8, Def: 6, MinFloor: 4, Zone: "fungal_wilds", Emoji: "🧫"},
	{Name: "Vine Horror", Atk: 16, Def: 3, MinFloor: 5, Zone: "fungal_wilds", Emoji: "🌱"},
	{Name: "Plague Bearer", Atk: 22, Def: 4, MinFloor: 6, Zone: "fungal_wilds", Emoji: "🦠"},
	{Name: "Clockwork Spider", Atk: 12, Def: 8, MinFloor: 7, Zone: "forge_district", Emoji: "🕷️"},
	{Name: "Molten Core", Atk: 26, Def: 2, MinFloor: 8, Zone: "forge_district", Emoji: "🌋"},
	{Name: "Arcane Sentinel", Atk: 20, Def: 10, MinFloor: 9, Zone: "forge_district", Emoji: "🔮"},
	{Name: "Echo Wraith", Atk: 24, Def: 6, MinFloor: 10, Zone: "abyss", Emoji: "👤"},
	{Name: "Void Devourer", Atk: 34, Def: 14, MinFloor: 13, Zone: "abyss", Emoji: "🕳️"},
}

func GenerateEnemy(zone string, floor int, rng *rand.Rand) *Enemy {
	var candidates []Enemy
	for _, e := range enemyTemplates {
		if e.Zone == zone && floor >= e.MinFloor {
			candidates = append(candidates, e)
		}
	}
	if len(candidates) == 0 {
		candidates = []Enemy{{Name: "Shadow", Atk: 10, Def: 2, MinFloor: 1, Zone: "", Emoji: "👾"}}
	}
	enemy := candidates[rng.Intn(len(candidates))]
	scale := 1.0 + float64(floor-1)*0.15
	enemy.MaxHP = int(float64(50+floor*10) * scale)
	enemy.HP = enemy.MaxHP
	enemy.Atk = int(float64(enemy.Atk) * scale)
	enemy.Def = int(float64(enemy.Def) * scale)
	return &enemy
}

func ApplyEnemyLevelScaling(enemy *Enemy, floor int, playerLevel int, rng *rand.Rand) {
	mult := LevelScalingMul(floor, playerLevel)
	variance := 1.0 + (rng.Float64()-0.5)*0.30
	final := mult * variance
	enemy.MaxHP = int(float64(enemy.MaxHP) * final)
	enemy.HP = enemy.MaxHP
	enemy.Atk = int(float64(enemy.Atk) * final)
	enemy.Def = int(float64(enemy.Def) * final)
	if enemy.Atk < 1 {
		enemy.Atk = 1
	}
	if enemy.Def < 0 {
		enemy.Def = 0
	}
	if enemy.MaxHP < 1 {
		enemy.MaxHP = 1
		enemy.HP = 1
	}
}

type CombatResult struct {
	EnemyName      string
	PlayerDamage   int
	EnemyDamage    int
	PetDamage      int
	EnemyDefeated  bool
	PlayerDefeated bool
	Log            []string
	Loot           []DelveItem
	WasCrit          bool
	PowerBlowBack    bool
	AppliedPoison    bool
	AppliedEntangled bool
	PartingBlast     int
	IgnoreDefendHit  bool
}

func (svc *Service) ResolveCombatRound(session *model.DelveSession, action string, lang string) *CombatResult {
	res := &CombatResult{}
	cs := svc.GetCombat(session.UserID)
	if cs == nil {
		res.Log = append(res.Log, i18n.T("delve.combat.no_combat", lang))
		return res
	}

	enemy := cs.Enemy
	res.EnemyName = enemy.Name

	if cs.Turn == 0 && cs.EnemyFirstStrike {
		firstDmg := enemy.Atk + rand.Intn(4)
		session.HP -= firstDmg
		res.EnemyDamage = firstDmg
		res.Log = append(res.Log, i18n.T("delve.combat.first_strike", lang, map[string]any{"enemy": enemy.Name, "damage": firstDmg}))
		cs.EnemyFirstStrike = false
		cs.Turn++
		if session.HP <= 0 {
			session.HP = 0
			res.PlayerDefeated = true
			res.Log = append(res.Log, i18n.T("delve.combat.fallen_battle", lang))
			svc.EndCombat(session.UserID)
			return res
		}
	}

	atk := EffectiveAtk(svc.store, session.UserID)
	intVal := EffectiveINT(svc.store, session.UserID)
	manaCost := 0

	switch action {
	case "slash":
		dmg := atk + rand.Intn(5)
		armorReduction := enemy.Def / 2
		netDmg := dmg - armorReduction
		if netDmg < 1 {
			netDmg = 1
		}
		enemy.HP -= netDmg
		res.PlayerDamage = netDmg
		res.Log = append(res.Log, i18n.T("delve.combat.slash", lang, map[string]any{"enemy": enemy.Name, "damage": netDmg}))
	case "fireball":
		manaCost = 15
		if session.Mana >= manaCost {
			session.Mana -= manaCost
			dmg := atk + 10 + intVal*2 + rand.Intn(8)
			enemy.HP -= dmg
			res.PlayerDamage = dmg
			res.Log = append(res.Log, i18n.T("delve.combat.fireball", lang, map[string]any{"enemy": enemy.Name, "damage": dmg}))
		} else {
			res.Log = append(res.Log, i18n.T("delve.combat.no_mana", lang))
		}
	case "power_blow":
		manaCost = 10
		if session.Mana >= manaCost {
			session.Mana -= manaCost
			dmg := atk + rand.Intn(5)
			armorReduction := enemy.Def / 2
			netDmg := dmg - armorReduction
			if netDmg < 1 {
				netDmg = 1
			}
			netDmg *= 2
			enemy.HP -= netDmg
			res.PlayerDamage = netDmg
			res.Log = append(res.Log, i18n.T("delve.combat.power_blow", lang, map[string]any{"damage": netDmg}))
			res.PowerBlowBack = true
		} else {
			res.Log = append(res.Log, i18n.T("delve.combat.no_mana", lang))
		}
	case "mend":
		manaCost = 20
		if session.Mana >= manaCost {
			session.Mana -= manaCost
			heal := session.MaxHP / 4
			session.HP += heal
			if session.HP > session.MaxHP {
				session.HP = session.MaxHP
			}
			res.Log = append(res.Log, i18n.T("delve.combat.mend", lang, map[string]any{"heal": heal}))
		} else {
			res.Log = append(res.Log, i18n.T("delve.combat.no_mana", lang))
		}
	case "defend":
		res.Log = append(res.Log, i18n.T("delve.combat.defend", lang))
	case "potion":
		res.Log = append(res.Log, i18n.T("delve.combat.potion", lang))
	}

	if res.PlayerDamage > 0 {
		var effects []string
		json.Unmarshal([]byte(session.StatusEffects), &effects)
		var newEffects []string
		for _, e := range effects {
			switch e {
			case "fortified":
				res.PlayerDamage = int(float64(res.PlayerDamage) * 1.25)
				res.Log = append(res.Log, i18n.T("delve.combat.fortified", lang))
			case "entangled":
				res.PlayerDamage = res.PlayerDamage / 2
				if res.PlayerDamage < 1 {
					res.PlayerDamage = 1
				}
				res.Log = append(res.Log, i18n.T("delve.combat.entangled", lang))
			case "blessed":
				res.PlayerDamage = int(float64(res.PlayerDamage) * 1.10)
				newEffects = append(newEffects, e)
				continue
			default:
				newEffects = append(newEffects, e)
				continue
			}
		}
		if len(effects) != len(newEffects) {
			jb, _ := json.Marshal(newEffects)
			session.StatusEffects = string(jb)
		}
	}

	if res.AppliedPoison {
		var effects []string
		json.Unmarshal([]byte(session.StatusEffects), &effects)
		effects = append(effects, "poisoned:3")
		jb, _ := json.Marshal(effects)
		session.StatusEffects = string(jb)
	}

	petDmg := 0
	petIDs := svc.DeployedPets(session)
	for _, pid := range petIDs {
		var pet model.UserPet
		if err := svc.store.DB.Where("id = ?", pid).First(&pet).Error; err == nil {
			dmg := pet.Atk / 5
			if dmg < 1 {
				dmg = 1
			}
			petDmg += dmg
		}
	}
	if petDmg > 0 {
		enemy.HP -= petDmg
		res.PetDamage = petDmg
		res.Log = append(res.Log, i18n.T("delve.combat.pets_damage", lang, map[string]any{"damage": petDmg}))
	}

	if enemy.HP <= 0 {
		res.EnemyDefeated = true
		res.Log = append(res.Log, i18n.T("delve.combat.enemy_defeated", lang, map[string]any{"enemy": enemy.Name}))

		if enemy.Name == "Spore Host" {
			res.AppliedPoison = true
			res.Log = append(res.Log, i18n.T("delve.combat.spore_host_poison", lang))
		}

		if enemy.Name == "Molten Core" {
			res.PartingBlast = enemy.MaxHP * 20 / 100
			res.Log = append(res.Log, i18n.T("delve.combat.molten_core_explode", lang, map[string]any{"damage": res.PartingBlast}))
			session.HP -= res.PartingBlast
			if session.HP <= 0 {
				session.HP = 0
				res.PlayerDefeated = true
				res.Log = append(res.Log, i18n.T("delve.combat.fallen_battle", lang))
				svc.EndCombat(session.UserID)
				return res
			}
		}

		svc.EndCombat(session.UserID)
		return res
	}

	enemyDmg := enemy.Atk + rand.Intn(4)

	critChance := EnemyCritChance(cs.Turn)
	if enemy.Name == "Crypt Stalker" {
		critChance *= 2
	}
	if rng := rand.Intn(100); rng < critChance {
		enemyDmg = int(float64(enemyDmg) * 1.5)
		res.WasCrit = true
	}

	if enemy.Name == "Bone Wraith" && res.WasCrit {
		enemyDmg = int(float64(enemyDmg) * 1.3)
		res.Log = append(res.Log, i18n.T("delve.combat.bone_wraith_pierce", lang))
	}

	if enemy.Name == "Arcane Sentinel" && action == "defend" {
		enemyDmg = enemyDmg * 2
		res.IgnoreDefendHit = true
	}

	if action == "defend" && !res.IgnoreDefendHit {
		enemyDmg = enemyDmg / 2
	}

	var effects []string
	json.Unmarshal([]byte(session.StatusEffects), &effects)
	for _, e := range effects {
		if e == "marked" {
			enemyDmg = int(float64(enemyDmg) * 1.10)
			break
		}
	}

	for _, e := range effects {
		if strings.HasPrefix(e, "entangled") {
			res.AppliedEntangled = true
			break
		}
	}

	if enemy.Name == "Plague Bearer" && res.WasCrit {
		res.AppliedPoison = true
		res.Log = append(res.Log, i18n.T("delve.combat.plague_bearer_poison", lang))
	}

	if enemy.Name == "Echo Wraith" {
		res.AppliedEntangled = true
	}

	if action == "power_blow" {
		enemyDmg = int(float64(enemyDmg) * 1.5)
	}

	session.HP -= enemyDmg
	res.EnemyDamage = enemyDmg
	critTag := ""
	if res.WasCrit {
		critTag = i18n.T("delve.combat.crit_tag", lang)
	}
	res.Log = append(res.Log, i18n.T("delve.combat.strikes_back", lang, map[string]any{"enemy": enemy.Name, "damage": enemyDmg, "crit": critTag}))

	if session.HP <= 0 {
		session.HP = 0
		res.PlayerDefeated = true
		res.Log = append(res.Log, i18n.T("delve.combat.fallen_battle", lang))
		svc.EndCombat(session.UserID)
		return res
	}

	cs.Turn++
	res.Log = append(res.Log, i18n.T("delve.combat.turn_status", lang, map[string]any{
		"turn": cs.Turn, "enemy": enemy.Name, "ehp": enemy.HP, "emhp": enemy.MaxHP,
		"php": session.HP, "pmhp": session.MaxHP,
	}))
	return res
}

func RenderCombatEmbed(session *model.DelveSession, cs *CombatState, svc *Service, lang string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: i18n.T("delve.combat.encounter_title", lang, map[string]any{
			"enemy": cs.Enemy.Name, "hp": cs.Enemy.HP, "max_hp": cs.Enemy.MaxHP,
		}),
		Color: 0xe74c3c,
	}
	fields := []*discordgo.MessageEmbedField{
		{Name: "\u200b", Value: "\u200b", Inline: false},
	}

	petIDs := svc.DeployedPets(session)
	if len(petIDs) > 0 {
		petLines := ""
		for _, pid := range petIDs {
			var pet model.UserPet
			if err := svc.store.DB.Where("id = ?", pid).First(&pet).Error; err == nil {
				petLines += i18n.T("delve.combat.pet_line", lang, map[string]any{
					"pet": pet.Nickname, "dmg": pet.Atk / 5,
				})
			}
		}
		fields = append(fields, &discordgo.MessageEmbedField{Name: i18n.T("delve.combat.pets_label", lang), Value: petLines, Inline: false})
	}

	warning := ""
	var effects []string
	json.Unmarshal([]byte(session.StatusEffects), &effects)
	for _, e := range effects {
		if e == "marked" {
			warning = i18n.T("delve.combat.marked_warning", lang)
			break
		}
	}

	if session.Torches == 0 {
		if warning != "" {
			warning += "\n" + i18n.T("delve.combat.darkness_warning", lang)
		} else {
			warning = i18n.T("delve.combat.darkness_warning", lang)
		}
	}

	if warning != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "\u200b", Value: warning, Inline: false})
	}

	embed.Fields = fields
	return embed
}

func ResolveFlee(session *model.DelveSession) (string, int) {
	lostGold := session.Gold / 2
	session.Gold -= lostGold
	session.HP -= 10
	if session.HP < 0 {
		session.HP = 0
	}
	return i18n.T("delve.combat.flee_description", "en", map[string]any{"gold": lostGold}), lostGold
}

func DescribeDanger(d DangerInfo) string {
	skulls := strings.Repeat("💀", d.Skulls)
	lvl := fmt.Sprintf("Rec. Lv %d", d.RecLevel)
	out := skulls + " " + lvl
	if d.IsPunished {
		out += " ⚠️"
	}
	return out
}
