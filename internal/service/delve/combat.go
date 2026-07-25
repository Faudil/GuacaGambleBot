package delve

import (
	"fmt"
	"math/rand"

	"github.com/bwmarrin/discordgo"

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
	// Crypt
	{Name: "Shambling Corpse", Atk: 8, Def: 2, MinFloor: 1, Zone: "crypt", Emoji: "🧟"},
	{Name: "Skeletal Archer", Atk: 10, Def: 1, MinFloor: 1, Zone: "crypt", Emoji: "💀"},
	{Name: "Crypt Rat Swarm", Atk: 6, Def: 1, MinFloor: 1, Zone: "crypt", Emoji: "🐀"},
	{Name: "Grave Warden", Atk: 12, Def: 4, MinFloor: 2, Zone: "crypt", Emoji: "⚰️"},
	{Name: "Bone Golem", Atk: 14, Def: 6, MinFloor: 3, Zone: "crypt", Emoji: "🦴"},
	{Name: "Wailing Banshee", Atk: 16, Def: 3, MinFloor: 3, Zone: "crypt", Emoji: "👻"},
	// Fungal Wilds
	{Name: "Spore Shambler", Atk: 10, Def: 3, MinFloor: 4, Zone: "fungal_wilds", Emoji: "🍄"},
	{Name: "Fungal Crawler", Atk: 12, Def: 4, MinFloor: 4, Zone: "fungal_wilds", Emoji: "🐛"},
	{Name: "Acid Slime", Atk: 14, Def: 2, MinFloor: 5, Zone: "fungal_wilds", Emoji: "🟢"},
	{Name: "Myconid Warrior", Atk: 16, Def: 5, MinFloor: 5, Zone: "fungal_wilds", Emoji: "🌿"},
	{Name: "Fungal Brute", Atk: 20, Def: 8, MinFloor: 6, Zone: "fungal_wilds", Emoji: "🦧"},
	// Forge District
	{Name: "Rusted Construct", Atk: 14, Def: 6, MinFloor: 7, Zone: "forge_district", Emoji: "⚙️"},
	{Name: "Forge Wraith", Atk: 18, Def: 4, MinFloor: 7, Zone: "forge_district", Emoji: "👤"},
	{Name: "Steam Guardian", Atk: 16, Def: 8, MinFloor: 8, Zone: "forge_district", Emoji: "🗿"},
	{Name: "Magma Hound", Atk: 22, Def: 6, MinFloor: 8, Zone: "forge_district", Emoji: "🐕"},
	{Name: "Iron Titan", Atk: 25, Def: 12, MinFloor: 9, Zone: "forge_district", Emoji: "🤖"},
	// Abyss
	{Name: "Void Stalker", Atk: 22, Def: 8, MinFloor: 10, Zone: "abyss", Emoji: "👁️"},
	{Name: "Reality Weaver", Atk: 24, Def: 10, MinFloor: 10, Zone: "abyss", Emoji: "🌀"},
	{Name: "Abyssal Hulk", Atk: 28, Def: 14, MinFloor: 11, Zone: "abyss", Emoji: "🐙"},
	{Name: "Soul Reaper", Atk: 30, Def: 10, MinFloor: 12, Zone: "abyss", Emoji: "💜"},
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

type CombatResult struct {
	EnemyName     string
	PlayerDamage  int
	EnemyDamage   int
	PetDamage     int
	EnemyDefeated bool
	PlayerDefeated bool
	Log           []string
	Loot          []DelveItem
}

func ResolveCombatRound(svc *Service, session *model.DelveSession, action string) *CombatResult {
	res := &CombatResult{}
	cs := svc.GetCombat(session.UserID)
	if cs == nil {
		res.Log = append(res.Log, "No active combat.")
		return res
	}

	enemy := cs.Enemy
	res.EnemyName = enemy.Name
	char, _ := svc.store.EnsureCharacter(session.UserID)
	charAtk := 10 + char.STR*2
	manaCost := 0

	switch action {
	case "slash":
		dmg := charAtk + rand.Intn(5)
		armorReduction := enemy.Def / 2
		netDmg := dmg - armorReduction
		if netDmg < 1 {
			netDmg = 1
		}
		enemy.HP -= netDmg
		res.PlayerDamage = netDmg
		res.Log = append(res.Log, fmt.Sprintf("You slash the %s for %d damage!", enemy.Name, netDmg))
	case "fireball":
		manaCost = 15
		if session.Mana >= manaCost {
			session.Mana -= manaCost
			dmg := charAtk + 10 + rand.Intn(8)
			enemy.HP -= dmg
			res.PlayerDamage = dmg
			res.Log = append(res.Log, fmt.Sprintf("Fireball engulfs the %s for %d damage!", enemy.Name, dmg))
		} else {
			res.Log = append(res.Log, "Not enough mana! You fumble.")
		}
	case "defend":
		res.Log = append(res.Log, "You brace for impact.")
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
		res.Log = append(res.Log, fmt.Sprintf("Your pets deal %d damage.", petDmg))
	}

	if enemy.HP <= 0 {
		res.EnemyDefeated = true
		res.Log = append(res.Log, fmt.Sprintf("The %s crumbles to dust!", enemy.Name))
		svc.EndCombat(session.UserID)
		return res
	}

	enemyDmg := enemy.Atk + rand.Intn(4)
	if action == "defend" {
		enemyDmg = enemyDmg / 2
	}
	session.HP -= enemyDmg
	res.EnemyDamage = enemyDmg
	if enemyDmg > 0 {
		res.Log = append(res.Log, fmt.Sprintf("The %s strikes back for %d damage!", enemy.Name, enemyDmg))
	}
	if session.HP <= 0 {
		session.HP = 0
		res.PlayerDefeated = true
		res.Log = append(res.Log, "You have fallen in battle...")
		svc.EndCombat(session.UserID)
		return res
	}

	cs.Turn++
	res.Log = append(res.Log, fmt.Sprintf("Turn %d: %s HP: %d/%d | Your HP: %d/%d",
		cs.Turn, enemy.Name, enemy.HP, enemy.MaxHP, session.HP, session.MaxHP))
	return res
}

func RenderCombatEmbed(session *model.DelveSession, cs *CombatState, svc *Service) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("⚔️ Encounter: %s (HP %d/%d)", cs.Enemy.Name, cs.Enemy.HP, cs.Enemy.MaxHP),
		Color: 0xe74c3c,
	}
	fields := []*discordgo.MessageEmbedField{
		{Name: fmt.Sprintf("🛡️ You — HP %d/%d  🔥 Mana %d/%d", session.HP, session.MaxHP, session.Mana, session.MaxMana), Value: "\u200b", Inline: false},
	}

	petIDs := svc.DeployedPets(session)
	if len(petIDs) > 0 {
		petLines := ""
		for _, pid := range petIDs {
			var pet model.UserPet
			if err := svc.store.DB.Where("id = ?", pid).First(&pet).Error; err == nil {
				petLines += fmt.Sprintf("🐾 %s — Auto-attacking: ~%d dmg/turn\n", pet.Nickname, pet.Atk/5)
			}
		}
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Pets", Value: petLines, Inline: false})
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
	return fmt.Sprintf("You flee the depths, escaping with your life. You lose %d gold.", lostGold), lostGold
}
