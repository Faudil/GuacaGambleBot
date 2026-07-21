package battle

import (
	"fmt"
	"math"
	"math/rand"
)

type DamageType int

const (
	DamageImpact  DamageType = iota
	DamageBite
	DamageScratch
	DamagePoison
	DamageFire
)

type BattlePet struct {
	ID       int64
	Nickname string
	Emoji    string
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
}

func (p *BattlePet) IsAlive() bool { return p.HP > 0 }

func (p *BattlePet) realDefense() int  { return max(0, p.Defense-p.defenseMalus) }
func (p *BattlePet) realACC() int      { return max(0, p.ACC-p.accMalus) }
func (p *BattlePet) realAtk() int      { return max(0, p.Atk-p.atkMalus) }
func (p *BattlePet) realDGE() int      { return max(0, p.DGE-p.dgeMalus) }
func (p *BattlePet) realSpeed() int    { return max(1, p.Speed-p.speedMalus) }
func (p *BattlePet) thornsDmg() float64 {
	return float64(p.realDefense())*0.1 + float64(p.realDefense())*0.05*float64(p.thornMult)
}

func (p *BattlePet) healFull() {
	p.HP = p.MaxHP
	p.defenseMalus = 0
	p.accMalus = 0
	p.dgeMalus = 0
	p.atkMalus = 0
	p.speedMalus = 0
	p.stunnedTurns = 0
	p.poisonedTurns = 0
	p.burningTurns = 0
	p.bleedingTurns = 0
	p.thornMult = 1
}

type BattleResult struct {
	WinnerID int64
	Log      []string
	Pet1HP   int
	Pet2HP   int
	Pet1     *BattlePet
	Pet2     *BattlePet
}

func Simulate(p1, p2 *BattlePet) *BattleResult {
	p1.healFull()
	p2.healFull()

	log := make([]string, 0, 10)
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
				fatigueMult = max(0.2, 1.0-float64(actions-50)*0.05)
			}

			msg := resolveAttack(attacker, defender, aEmoji, dEmoji, an, dn, fatigueMult)
			log = append(log, msg)
			if len(log) > 10 {
				log = log[1:]
			}
			actions++
		}
	}

	result := &BattleResult{
		Pet1HP: p1.HP,
		Pet2HP: p2.HP,
		Pet1:   p1,
		Pet2:   p2,
		Log:    log,
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

	hitChance := max(20, min(100, int(100+float64(attacker.realACC())-float64(defender.realDGE())*fatigueMult)))
	if rand.Intn(100)+1 > hitChance {
		parts = append(parts, fmt.Sprintf("💨 %s **%s** dodges %s's attack!", dEmoji, dn, an))
		return joinParts(parts)
	}

	baseDmg := max(float64(attacker.realAtk())*0.2, float64(attacker.realAtk())-float64(defender.realDefense())*fatigueMult)
	isCrit := rand.Intn(100)+1 <= attacker.CritC
	critMult := 1.0
	if isCrit {
		critMult = 1 + (attacker.CritD-1)/2 + rand.Float64()*(attacker.CritD-(1+(attacker.CritD-1)/2))
	}
	baseDmg = baseDmg * critMult * (0.9 + rand.Float64()*0.2)
	finalDmg := int(math.Round(baseDmg))
	if finalDmg < 1 {
		finalDmg = 1
	}

	defender.HP = max(0, defender.HP-finalDmg)

	dmgType := getDamageType(attacker.Nickname)
	effectTrigger := dmgType != nil && rand.Intn(100)+1 <= attacker.SpcC

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

	thornsProb := min(0.70, float64(defender.realDefense())/max(float64(1), float64(defender.realAtk())))
	if rand.Float64() < thornsProb {
		defender.thornMult++
		td := int(math.Round(defender.thornsDmg()))
		if td > 0 {
			attacker.HP = max(0, attacker.HP-td)
			parts = append(parts, fmt.Sprintf(" but takes **%d** thorn damage 🌵", td))
		}
	}

	return joinParts(parts)
}

func tickEffects(p *BattlePet) string {
	parts := make([]string, 0, 3)
	if p.poisonedTurns > 0 {
		dmg := max(1, int(float64(p.MaxHP)*0.05))
		p.HP = max(0, p.HP-dmg)
		parts = append(parts, fmt.Sprintf("🧪 **%s** suffers from poison and loses **%d** HP.", p.Nickname, dmg))
		p.poisonedTurns--
		if p.poisonedTurns == 0 {
			p.atkMalus = 0
			parts = append(parts, fmt.Sprintf("✨ **%s** is no longer poisoned!", p.Nickname))
		}
	}
	if p.burningTurns > 0 {
		dmg := max(5, int(float64(p.MaxHP)*0.08))
		p.HP = max(0, p.HP-dmg)
		parts = append(parts, fmt.Sprintf("🔥 **%s** burns and loses **%d** HP.", p.Nickname, dmg))
		p.burningTurns--
		if p.burningTurns == 0 {
			p.dgeMalus = 0
			parts = append(parts, fmt.Sprintf("💦 **%s** is no longer burning!", p.Nickname))
		}
	}
	if p.bleedingTurns > 0 {
		dmg := max(2, int(float64(p.MaxHP)*0.06))
		p.HP = max(0, p.HP-dmg)
		parts = append(parts, fmt.Sprintf("🩸 **%s** bleeds and loses **%d** HP.", p.Nickname, dmg))
		p.bleedingTurns--
		if p.bleedingTurns == 0 {
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
}

func getDamageType(name string) *DamageType {
	if dt, ok := petDamageTypes[name]; ok {
		return &dt
	}
	return nil
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
