package battle

import (
	"fmt"
	"math"
	"math/rand"
)

type DamageType int

const (
	DamageImpact DamageType = iota
	DamageBite
	DamageScratch
	DamagePoison
	DamageFire
)

type BattlePet struct {
	ID       int64
	Nickname string
	Emoji    string
	PetType  string
	Level    int
	HP       int
	MaxHP    int
	Atk      int
	Defense  int
	Speed    int
	DGE      int
	ACC      int
	CritC    int
	CritD    float64
	SpcC     int

	Skills []string

	defenseMalus  int
	accMalus      int
	dgeMalus      int
	atkMalus      int
	speedMalus    int
	stunnedTurns  int
	poisonedTurns int
	burningTurns  int
	bleedingTurns int
	thornMult     int

	// PerkInt is a scratch map for skill-proc tracking during a battle.
	PerkInt map[string]int
}

func (p *BattlePet) IsAlive() bool { return p.HP > 0 }

func (p *BattlePet) realDefense() int {
	v := max(0, p.Defense-p.defenseMalus)
	if p.PerkInt["mod_rampage"] > 0 && p.HP < p.MaxHP*40/100 {
		v = v * 8 / 10
	}
	if cm := p.PerkInt["chaos_def_mult"]; cm > 0 {
		v = int(float64(v) * float64(cm) / 100.0)
	}
	return v
}
func (p *BattlePet) realACC() int {
	v := max(0, p.ACC-p.accMalus)
	return v
}
func (p *BattlePet) realAtk() int {
	v := max(0, p.Atk-p.atkMalus)
	if cm := p.PerkInt["chaos_atk_mult"]; cm > 0 {
		v = int(float64(v) * float64(cm) / 100.0)
	}
	return v
}
func (p *BattlePet) realDGE() int {
	v := max(0, p.DGE-p.dgeMalus)
	if p.PerkInt["mod_heavy_rain"] > 0 {
		v = v / 2
	}
	return v
}
func (p *BattlePet) realSpeed() int {
	v := max(0, p.Speed-p.speedMalus)
	if p.PerkInt["mod_heavy_rain"] > 0 {
		v = v * 9 / 10
	}
	if cm := p.PerkInt["chaos_spd_mult"]; cm > 0 {
		v = int(float64(v) * float64(cm) / 100.0)
	}
	return max(1, v)
}
func (p *BattlePet) thornsDmg() float64 {
	return float64(p.realDefense())*0.1 + float64(p.realDefense())*0.05*float64(p.thornMult)
}

func (p *BattlePet) healFull() {
	p.HP = p.MaxHP
	p.resetBattleState()
}

// resetBattleState clears all transient battle state (status effects, maluses
// and perk procs) without touching HP.
func (p *BattlePet) resetBattleState() {
	p.defenseMalus = 0
	p.accMalus = 0
	p.dgeMalus = 0
	p.atkMalus = 0
	p.speedMalus = 0
	p.stunnedTurns = 0
	p.poisonedTurns = 0
	p.burningTurns = 0
	p.bleedingTurns = 0
	p.thornMult = 0
	p.PerkInt = make(map[string]int)
}

type BattleResult struct {
	WinnerID int64
	Log      []string
	Pet1HP   int
	Pet2HP   int
	Pet1     *BattlePet
	Pet2     *BattlePet
	// Turns records every action with the HP snapshots right after it, so the
	// fight can be replayed with live HP bars.
	Turns []BattleTurn
}

// BattleTurn is a single battle action with the HP of both pets right after it.
type BattleTurn struct {
	Pet1HP int
	Pet2HP int
	Msg    string
}

func Simulate(p1, p2 *BattlePet, modID ...string) *BattleResult {
	p1.healFull()
	p2.healFull()
	return simulate(p1, p2, modID...)
}

// SimulatePreserveHP runs a battle without restoring the pets to full HP first:
// each pet starts with its current HP (clamped to MaxHP). Transient battle
// state (status effects, maluses, perk procs) is still reset between battles.
func SimulatePreserveHP(p1, p2 *BattlePet, modID ...string) *BattleResult {
	p1.resetBattleState()
	p2.resetBattleState()
	p1.HP = max(0, min(p1.MaxHP, p1.HP))
	p2.HP = max(0, min(p2.MaxHP, p2.HP))
	return simulate(p1, p2, modID...)
}

func simulate(p1, p2 *BattlePet, modID ...string) *BattleResult {
	applyBattleStartSkills(p1.Skills, p1, p2)
	applyBattleStartSkills(p2.Skills, p2, p1)

	if len(modID) > 0 && modID[0] != "" {
		ApplyModifierBeforeBattle(p1, p2, modID[0])
	}

	log := make([]string, 0, 10)
	turns := make([]BattleTurn, 0, 10)
	actions := 0
	atb1 := 0.0
	atb2 := 0.0
	maxActions := 70

	for p1.IsAlive() && p2.IsAlive() && actions <= maxActions {
		atb1 += float64(p1.realSpeed())
		atb2 += float64(p2.realSpeed())

		for (atb1 >= 100 || atb2 >= 100) && p1.IsAlive() && p2.IsAlive() {
			var attacker, defender *BattlePet
			aEmoji, dEmoji := "", ""
			an, dn := "", ""

			if atb1 >= 100 && atb2 >= 100 {
				if atb1 > atb2 {
					attacker, defender = p1, p2
					aEmoji, dEmoji = p1.Emoji, p2.Emoji
					an, dn = p1.Nickname, p2.Nickname
					atb1 -= 100
					atb2 += float64(p1.realSpeed()) / 2
				} else if atb2 > atb1 {
					attacker, defender = p2, p1
					aEmoji, dEmoji = p2.Emoji, p1.Emoji
					an, dn = p2.Nickname, p1.Nickname
					atb2 -= 100
					atb1 += float64(p2.realSpeed()) / 2
				} else {
					if p1.realSpeed() >= p2.realSpeed() {
						attacker, defender = p1, p2
						aEmoji, dEmoji = p1.Emoji, p2.Emoji
						an, dn = p1.Nickname, p2.Nickname
						atb1 -= 100
					} else {
						attacker, defender = p2, p1
						aEmoji, dEmoji = p2.Emoji, p1.Emoji
						an, dn = p2.Nickname, p1.Nickname
						atb2 -= 100
					}
				}
			} else if atb1 >= 100 {
				attacker, defender = p1, p2
				aEmoji, dEmoji = p1.Emoji, p2.Emoji
				an, dn = p1.Nickname, p2.Nickname
				atb1 -= 100
				atb2 += float64(p1.realSpeed()) / 2
			} else {
				attacker, defender = p2, p1
				aEmoji, dEmoji = p2.Emoji, p1.Emoji
				an, dn = p2.Nickname, p1.Nickname
				atb2 -= 100
				atb1 += float64(p2.realSpeed()) / 2
			}

			fatigueMult := 1.0
			if actions > 25 {
				fatigueMult = max(0.2, 1.0-float64(actions-25)*0.05)
			}

			msg := resolveAttack(attacker, defender, aEmoji, dEmoji, an, dn, fatigueMult)
			log = append(log, msg)
			if len(log) > 10 {
				log = log[1:]
			}
			turns = append(turns, BattleTurn{Pet1HP: p1.HP, Pet2HP: p2.HP, Msg: msg})
			actions++
		}
	}

	result := &BattleResult{
		Pet1HP: p1.HP,
		Pet2HP: p2.HP,
		Pet1:   p1,
		Pet2:   p2,
		Log:    log,
		Turns:  turns,
	}
	// Phoenix Rebirth check
	if !p1.IsAlive() && p1.PerkInt["rebirth"] > 0 {
		p1.PerkInt["rebirth"] = 0
		p1.HP = p1.MaxHP * 3 / 10
		result.Pet1HP = p1.HP
	}
	if !p2.IsAlive() && p2.PerkInt["rebirth"] > 0 {
		p2.PerkInt["rebirth"] = 0
		p2.HP = p2.MaxHP * 3 / 10
		result.Pet2HP = p2.HP
	}
	if !p1.IsAlive() && p2.IsAlive() {
		result.WinnerID = p2.ID
	} else if p1.IsAlive() && !p2.IsAlive() {
		result.WinnerID = p1.ID
	}
	return result
}

func resolveAttack(attacker, defender *BattlePet, aEmoji, dEmoji, an, dn string, fatigueMult float64) string {
	tickMsg := tickEffects(attacker)
	parts := make([]string, 0, 4)
	if tickMsg != "" {
		parts = append(parts, tickMsg)
	}
	if !attacker.IsAlive() {
		return joinParts(parts)
	}

	if attacker.stunnedTurns > 0 {
		attacker.stunnedTurns--
		parts = append(parts, fmt.Sprintf("💫 %s **%s** is stunned (%d turns left)", aEmoji, an, attacker.stunnedTurns+1))
		if attacker.stunnedTurns <= 0 {
			attacker.defenseMalus = 0
		}
		return joinParts(parts)
	}

	if attacker.PerkInt["mod_chaos"] > 0 {
		stat := []string{"atk", "def", "spd"}[rand.Intn(3)]
		mult := 1.0
		if rand.Intn(2) == 0 {
			mult = 1.20
		} else {
			mult = 0.80
		}
		switch stat {
		case "atk":
			attacker.PerkInt["chaos_atk_mult"] = int(mult * 100)
		case "def":
			attacker.PerkInt["chaos_def_mult"] = int(mult * 100)
		case "spd":
			attacker.PerkInt["chaos_spd_mult"] = int(mult * 100)
		}
	}
	hitChance := max(20, min(100, int(100+float64(attacker.realACC())-float64(defender.realDGE())*fatigueMult)))
	if attacker.PerkInt["mod_shadow_realm"] > 0 {
		hitChance = max(20, hitChance-25)
	}
	if defender.PerkInt["mod_heavy_rain"] > 0 {
		hitChance = max(20, hitChance-50)
	}
	if rand.Intn(100)+1 > hitChance {
		if defender.PerkInt["mod_shadow_realm"] > 0 {
			heal := max(1, defender.MaxHP*8/100)
			defender.HP = min(defender.MaxHP, defender.HP+heal)
			parts = append(parts, fmt.Sprintf("🌑 %s **%s** dodges and heals **%d** HP!", dEmoji, dn, heal))
		} else {
			parts = append(parts, fmt.Sprintf("💨 %s **%s** dodges %s's attack!", dEmoji, dn, an))
		}
		return joinParts(parts)
	}

	defForDmg := float64(defender.realDefense())
	if attacker.PerkInt["piercing"] > 0 {
		defForDmg *= 0.60
	}
	if ap := attacker.PerkInt["artifact_piercing"]; ap > 0 {
		pct := 1.0 - float64(ap)*3.0/100.0
		if pct < 0.7 {
			pct = 0.7
		}
		defForDmg *= pct
	}
	if defender.PerkInt["artifact_warding"] > 0 {
		defForDmg = math.Min(defForDmg*1.2, defForDmg+10)
	}
	baseDmg := max(float64(attacker.realAtk())*0.2, float64(attacker.realAtk())-defForDmg*fatigueMult)
	critChance := attacker.CritC
	if attacker.PerkInt["mod_starlight"] > 0 {
		critChance += 15
	}
	isCrit := rand.Intn(100)+1 <= critChance
	critMult := 1.0
	if isCrit {
		critMult = 1 + (attacker.CritD-1)/2 + rand.Float64()*(attacker.CritD-(1+(attacker.CritD-1)/2))
	}
	baseDmg = baseDmg * critMult * (0.9 + rand.Float64()*0.2)

	// Last Stand: 2x damage when below 25% HP
	if attacker.PerkInt["last_stand"] > 0 && attacker.HP < attacker.MaxHP/4 {
		baseDmg *= 2.0
	}
	// Dragon's Fury: first attack deals 2x
	if attacker.PerkInt["dragon_fury"] > 0 {
		baseDmg *= 2.0
		attacker.PerkInt["dragon_fury"] = 0 // consumed
	}
	// Berserker: +3% crit per 10% HP lost
	if attacker.PerkInt["berserker"] > 0 {
		hpPct := 100 - (attacker.HP*100)/attacker.MaxHP
		critBonus := (hpPct / 10) * 3
		if rand.Intn(100)+1 <= critBonus {
			baseDmg *= attacker.CritD
		}
	}

	finalDmg := int(math.Round(baseDmg))
	if finalDmg < 1 {
		finalDmg = 1
	}

	if attacker.PerkInt["mod_burning_sun"] > 0 && getDamageType(attacker.PetType) != nil && *getDamageType(attacker.PetType) == DamageFire {
		finalDmg = int(math.Round(float64(finalDmg) * 1.40))
	}
	if defender.PerkInt["mod_thunderstorm"] > 0 && defender.realSpeed() < 15 {
		finalDmg = int(math.Round(float64(finalDmg) * 1.20))
	}
	if attacker.PerkInt["mod_rampage"] > 0 && attacker.HP < attacker.MaxHP*40/100 {
		finalDmg = int(math.Round(float64(finalDmg) * 1.30))
	}

	defender.HP = max(0, defender.HP-finalDmg)

	dmgType := getDamageType(attacker.PetType)
	spcC := attacker.SpcC
	if attacker.PerkInt["mod_starlight"] > 0 && dmgType != nil {
		spcC = 100
	}
	effectTrigger := dmgType != nil && rand.Intn(100)+1 <= spcC

	effectMsg := ""
	if effectTrigger && dmgType != nil {
		effectMsg = applySpecialEffect(defender, *dmgType)
	}

	if isCrit {
		parts = append(parts, fmt.Sprintf("💥 **CRITICAL HIT!** ⚔️ %s **%s** deals **%d** damage%s", aEmoji, an, finalDmg, effectMsg))
	} else {
		parts = append(parts, fmt.Sprintf("⚔️ %s **%s** deals **%d** damage%s", aEmoji, an, finalDmg, effectMsg))
	}

	if effectTrigger && dmgType != nil && *dmgType == DamageScratch {
		heal := int(float64(attacker.realAtk()) * 0.3)
		if isCrit {
			heal *= 2
		}
		attacker.HP = min(attacker.MaxHP, attacker.HP+heal)
		parts[len(parts)-1] += fmt.Sprintf(" and heals for **%d** HP 🩸", heal)
	}
	if av := attacker.PerkInt["artifact_vampirism"]; av > 0 {
		lifesteal := max(1, finalDmg*av/100)
		attacker.HP = min(attacker.MaxHP, attacker.HP+lifesteal)
		parts[len(parts)-1] += fmt.Sprintf(" 🩸 drains **%d** HP", lifesteal)
	}
	if attacker.PerkInt["mod_blood_moon"] > 0 {
		lifesteal := max(1, finalDmg*50/100)
		attacker.HP = min(attacker.MaxHP, attacker.HP+lifesteal)
		parts[len(parts)-1] += fmt.Sprintf(" 🌕 leeches **%d** HP", lifesteal)
	}

	// Counter: 30% chance to reflect 50% damage back to attacker
	if defender.PerkInt["counter"] > 0 && rand.Float64() < 0.30 {
		counterDmg := max(1, finalDmg/2)
		attacker.HP = max(0, attacker.HP-counterDmg)
		parts = append(parts, fmt.Sprintf(" 🛡️ **%s** counters for **%d** damage!", dn, counterDmg))
	}

	// Thornmail: always procs, no probability check
	if defender.PerkInt["thornmail"] > 0 {
		defender.thornMult++
		td := int(math.Round(defender.thornsDmg()))
		if td > 0 {
			attacker.HP = max(0, attacker.HP-td)
			parts = append(parts, fmt.Sprintf(" but takes **%d** thorn damage 🌵", td))
		}
	} else {
		thornsProb := min(0.70, float64(defender.realDefense())/max(float64(1), float64(defender.realAtk())))
		if rand.Float64() < thornsProb {
			defender.thornMult++
			td := int(math.Round(defender.thornsDmg()))
			if td > 0 {
				attacker.HP = max(0, attacker.HP-td)
				parts = append(parts, fmt.Sprintf(" but takes **%d** thorn damage 🌵", td))
			}
		}
	}

	return joinParts(parts)
}

func tickEffects(p *BattlePet) string {
	parts := make([]string, 0, 6)

	if _, ok := p.PerkInt["mod_start_burn"]; ok {
		delete(p.PerkInt, "mod_start_burn")
		p.burningTurns = max(p.burningTurns, 1)
		if p.dgeMalus == 0 {
			p.dgeMalus = int(float64(p.DGE) * 0.8)
		}
		parts = append(parts, fmt.Sprintf("🔥 %s is set on fire by the Burning Sun!", p.Emoji+" **"+p.Nickname+"**"))
	}

	if rl := p.PerkInt["artifact_rejuvenation"]; rl > 0 {
		heal := max(1, p.MaxHP*rl/100)
		p.HP = min(p.MaxHP, p.HP+heal)
		parts = append(parts, fmt.Sprintf("💚 %s regenerates **%d** HP from Rejuvenation.", p.Emoji+" **"+p.Nickname+"**", heal))
	}

	if regen, ok := p.PerkInt["regeneration"]; ok {
		p.PerkInt["regeneration"] = (regen + 1) % 3
		if regen == 2 { // every 3rd tick
			heal := max(1, p.MaxHP/20)
			p.HP = min(p.MaxHP, p.HP+heal)
			parts = append(parts, fmt.Sprintf("💚 **%s** regenerates **%d** HP.", p.Nickname, heal))
		}
	}
	if p.poisonedTurns > 0 {
		dmg := max(1, int(float64(p.MaxHP)*0.05))
		p.HP = max(0, p.HP-dmg)
		parts = append(parts, fmt.Sprintf("🧪 **%s** suffers from poison and loses **%d** HP.", p.Nickname, dmg))
		p.poisonedTurns--
		if p.PerkInt["mod_frost_aura"] > 0 {
			p.poisonedTurns--
		}
		if p.poisonedTurns <= 0 {
			p.poisonedTurns = 0
			p.atkMalus = 0
			parts = append(parts, fmt.Sprintf("✨ **%s** is no longer poisoned!", p.Nickname))
		}
	}
	if p.burningTurns > 0 {
		dmg := max(5, int(float64(p.MaxHP)*0.08))
		p.HP = max(0, p.HP-dmg)
		parts = append(parts, fmt.Sprintf("🔥 **%s** burns and loses **%d** HP.", p.Nickname, dmg))
		p.burningTurns--
		if p.PerkInt["mod_frost_aura"] > 0 {
			p.burningTurns--
		}
		if p.burningTurns <= 0 {
			p.burningTurns = 0
			p.dgeMalus = 0
			parts = append(parts, fmt.Sprintf("💦 **%s** is no longer burning!", p.Nickname))
		}
	}
	if p.bleedingTurns > 0 {
		dmg := max(2, int(float64(p.MaxHP)*0.06))
		p.HP = max(0, p.HP-dmg)
		parts = append(parts, fmt.Sprintf("🩸 **%s** bleeds and loses **%d** HP.", p.Nickname, dmg))
		if p.PerkInt["mod_frost_aura"] > 0 {
			p.bleedingTurns--
		}
		p.bleedingTurns--
		if p.bleedingTurns <= 0 {
			p.bleedingTurns = 0
			p.speedMalus = 0
			parts = append(parts, fmt.Sprintf("🩹 **%s** is no longer bleeding!", p.Nickname))
		}
	}
	return joinParts(parts)
}

func applySpecialEffect(target *BattlePet, dt DamageType) string {
	switch dt {
	case DamageBite:
		target.bleedingTurns = 3
		target.speedMalus = int(float64(target.Speed) * 0.2)
		return " and inflicts **Bleeding** 🩸"
	case DamageScratch:
		target.accMalus = int(float64(target.ACC) * 0.3)
		target.dgeMalus = int(float64(target.DGE) * 0.3)
		return " and **weakens** it (Accuracy/Dodge reduced) 📉"
	case DamagePoison:
		target.poisonedTurns = 3
		target.atkMalus = int(float64(target.Atk) * 0.3)
		return " and **poisons** it 🧪"
	case DamageImpact:
		target.stunnedTurns = 1 + rand.Intn(2)
		target.defenseMalus = int(float64(target.Defense) * 0.2)
		return " and **stuns** it 💫"
	case DamageFire:
		target.burningTurns = 2
		target.dgeMalus = int(float64(target.DGE) * 0.8)
		return " and **burns** it 🔥"
	}
	return ""
}

var petDamageTypes = map[string]DamageType{
	"Escargot": DamageImpact, "Souris": DamageBite, "Cochon": DamageImpact,
	"Grenouille": DamageImpact, "Taupe": DamageScratch, "Pélican": DamageBite,
	"Mouton": DamageImpact, "Abeille": DamagePoison, "Chien": DamageBite,
	"Chat": DamageScratch, "Cheval": DamageImpact, "Renard": DamageBite,
	"Singe": DamageBite, "Ours": DamageBite, "Chameau": DamageImpact,
	"Panda": DamageBite, "Tigre": DamageScratch, "Pieuvre": DamagePoison,
	"Dragon": DamageFire, "Hamster": DamageFire, "Fourmi": DamageBite,
	"Hérisson": DamageScratch, "Canard": DamageBite, "Chouette": DamageScratch,
	"Paresseux": DamageScratch, "Kangourou": DamageImpact, "Iguane": DamagePoison,
	"Gorille": DamageImpact, "Scorpion": DamagePoison, "Bison": DamageImpact,
	"Rhino": DamageImpact, "Aigle": DamageScratch, "Crocodile": DamageBite,
	"Putois": DamagePoison, "Dauphin": DamageImpact, "Léopard": DamageBite,
	"Lion": DamageScratch, "Ours polaire": DamageBite,
	"Tyrannosaure": DamageBite, "Diplodocus": DamageImpact,
	"Mamouth": DamageImpact, "Mégalodon": DamageBite,
	"Kraken": DamagePoison, "Licorne": DamageFire,
	"Phoenix": DamageFire, "Cerbère": DamageFire,
	"Fenrir": DamageScratch, "Ratatosk": DamageScratch,
	"Nidhögg": DamagePoison, "Bedawang": DamagePoison,
	"Trilobite": DamageImpact, "Ammonite": DamageImpact, "Anomalocaris": DamageBite,
	"Orthoceras": DamageImpact, "Méganeura": DamagePoison,
	"Archéoptéryx": DamageScratch, "Ptéranodon": DamageScratch,
	"Dimétrodon": DamageBite, "Smilodon": DamageBite, "Mégalocéros": DamageImpact,
	"Doedicurus": DamageImpact, "Mosasaurus": DamageBite, "Titanoboa": DamagePoison,
	"Phorusrhacos": DamageScratch, "Rhinocéros laineux": DamageImpact,
	"Entelodon": DamageImpact,
}

func getDamageType(name string) *DamageType {
	if dt, ok := PetDamageType(name); ok {
		return &dt
	}
	return nil
}

// PetDamageType returns the damage type of a pet species, if one is registered.
func PetDamageType(name string) (DamageType, bool) {
	dt, ok := petDamageTypes[name]
	return dt, ok
}

func applyBattleStartSkills(skills []string, owner, opponent *BattlePet) {
	for _, skillID := range skills {
		switch skillID {
		case "iron_shell":
			owner.Defense = int(float64(owner.Defense) * 1.20)
		case "keen_edge":
			owner.Atk = int(float64(owner.Atk) * 1.15)
		case "evasive":
			owner.DGE = int(float64(owner.DGE) * 1.20)
		case "last_stand":
			owner.PerkInt["last_stand"] = 1
		case "regeneration":
			owner.PerkInt["regeneration"] = 0
		case "counter":
			owner.PerkInt["counter"] = 1
		case "piercing_strike":
			owner.PerkInt["piercing"] = 1
		case "berserker":
			owner.PerkInt["berserker"] = 1
		case "thornmail":
			owner.PerkInt["thornmail"] = 1
		case "phoenix_rebirth":
			owner.PerkInt["rebirth"] = 1
		case "dragon_fury":
			owner.PerkInt["dragon_fury"] = 1
		}
	}
}

func ApplyModifierBeforeBattle(p1, p2 *BattlePet, modID string) {
	switch modID {
	case "burning_sun":
		p1.PerkInt["mod_burning_sun"] = 1
		p2.PerkInt["mod_burning_sun"] = 1
		p1.PerkInt["mod_start_burn"] = 1
		p2.PerkInt["mod_start_burn"] = 1
	case "heavy_rain":
		p1.PerkInt["mod_heavy_rain"] = 1
		p2.PerkInt["mod_heavy_rain"] = 1
	case "starlight":
		p1.PerkInt["mod_starlight"] = 1
		p2.PerkInt["mod_starlight"] = 1
	case "iron_will":
		p1.PerkInt["mod_iron_will"] = 1
		p2.PerkInt["mod_iron_will"] = 1
		p1.Defense = int(float64(p1.Defense) * 1.30)
		p2.Defense = int(float64(p2.Defense) * 1.30)
		p1.Atk = int(float64(p1.Atk) * 0.85)
		p2.Atk = int(float64(p2.Atk) * 0.85)
	case "blood_moon":
		p1.PerkInt["mod_blood_moon"] = 1
		p2.PerkInt["mod_blood_moon"] = 1
	case "thunderstorm":
		p1.PerkInt["mod_thunderstorm"] = 1
		p2.PerkInt["mod_thunderstorm"] = 1
	case "shadow_realm":
		p1.PerkInt["mod_shadow_realm"] = 1
		p2.PerkInt["mod_shadow_realm"] = 1
	case "rampage":
		p1.PerkInt["mod_rampage"] = 1
		p2.PerkInt["mod_rampage"] = 1
	case "frost_aura":
		p1.Defense += 15
		p2.Defense += 15
		p1.PerkInt["mod_frost_aura"] = 1
		p2.PerkInt["mod_frost_aura"] = 1
	case "chaos":
		p1.PerkInt["mod_chaos"] = 1
		p2.PerkInt["mod_chaos"] = 1
	}
}

func joinParts(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "\n"
		}
		out += p
	}
	return out
}
