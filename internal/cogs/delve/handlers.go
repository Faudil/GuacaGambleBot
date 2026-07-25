package delve

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	delvesvc "guacagamblebot/internal/service/delve"
	petsvc "guacagamblebot/internal/service/pets"
)

func (c *Cog) onNavigate(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve. Start one with `/delve`!")
		return
	}
	s.RoomsCleared++
	s.Mana += 5
	if s.Mana > s.MaxMana {
		s.Mana = s.MaxMana
	}
	room := c.nextRoom(s, "en")
	embed, comps := c.renderRoomWithFallen(s, room, "en")
	c.respond(b, i, embed, comps)
}

func (c *Cog) onFight(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	zone := s.Zone
	seed := s.Seed + int64(s.RoomsCleared)
	rng := rand.New(rand.NewSource(seed))

	// Check for Gravewarden Morvain special boss
	name, hp, maxHP, atk, def, spawn := c.crimsvc.CheckGravewardenSpawn(userID, s.Floor, seed)
	if spawn {
		enemy := &delvesvc.Enemy{
			Name:  name,
			HP:    hp,
			MaxHP: maxHP,
			Atk:   atk,
			Def:   def,
			Zone:  s.Zone,
			Emoji: "💀",
		}
		c.svc.StartCombat(s, enemy)
		cs := c.svc.GetCombat(userID)
		embed := delvesvc.RenderCombatEmbed(s, cs, c.svc)
		embed.Title = "💀 Boss: Gravewarden Morvain"
		embed.Description = "The air grows cold. A figure of bone and shadow rises before you, ancient armor creaking with each movement. The Gravewarden has been waiting."
		comps := delvesvc.CombatRoomButtons()
		c.respond(b, i, embed, comps)
		return
	}

	enemy := delvesvc.GenerateEnemy(zone, s.Floor, rng)
	c.svc.StartCombat(s, enemy)
	cs := c.svc.GetCombat(userID)
	embed := delvesvc.RenderCombatEmbed(s, cs, c.svc)
	comps := delvesvc.CombatRoomButtons()
	c.respond(b, i, embed, comps)
}

func (c *Cog) onDefendStart(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.onFight(b, i)
}

func (c *Cog) resolveCombatAndRender(b *interaction.Bot, i *discordgo.InteractionCreate, action string) {
	userID := interaction.ToInt64(i.Member.User.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	res := delvesvc.ResolveCombatRound(c.svc, s, action)

	if res.PlayerDefeated {
		c.svc.AddFlag(s, "fell_in_battle")
		s.Status = "fallen"
		now := time.Now()
		s.DiedAt = &now

		petIDs := c.svc.DeployedPets(s)
		for _, pid := range petIDs {
			var pet model.UserPet
			if err := c.store.DB.Where("id = ?", pid).First(&pet).Error; err == nil {
				if pet.BondLevel >= 75 && rand.Intn(100) < 50 {
					s.AutoRescued = true
					s.AutoRescuePet = pet.Nickname
					break
				}
				if pet.BondLevel >= 50 && rand.Intn(100) < 25 {
					s.AutoRescued = true
					s.AutoRescuePet = pet.Nickname
					break
				}
			}
		}

		midnight := time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour)
		c.store.SetCooldown(userID, "delve_death")
		_ = midnight
		c.saveSession(s)

		petSvc := petsvc.New(c.store, c.cfg)
		petSvc.AddArtifactXP(userID, petsvc.ArtifactDelveCompleteXP)

		rescueMsg := ""
		if s.AutoRescued {
			rescueMsg = fmt.Sprintf("\n\n🐾 **%s** refuses to leave your side! They will drag you to safety soon. Check back with `/delve` in a few minutes.", s.AutoRescuePet)
		} else {
			rescueMsg = "\n\n🆘 Your pets couldn't reach you. Another adventurer on this floor may find you and pull you back."
		}

		embed := &discordgo.MessageEmbed{
			Title:       "💀 You Have Fallen",
			Description: "The Undercroft has claimed you this day. Your journey ends... for now." + rescueMsg + "\n\n" + strings.Join(res.Log, "\n"),
			Color:       0xe74c3c,
		}
		c.respond(b, i, embed, nil)
	} else if res.EnemyDefeated {
		gold := delvesvc.GoldReward(s.Zone, s.Floor)
		s.Gold += gold
		if len(res.Loot) > 0 {
			c.svc.AddItem(s, res.Loot[0])
		}
		veilKey := delvesvc.MaybeDropVeilKey(s.Zone, s.Floor)
		if veilKey != nil {
			c.svc.AddItem(s, *veilKey)
		}
		c.svc.AddFlag(s, "first_descent")
		c.saveSession(s)

		petSvc := petsvc.New(c.store, c.cfg)
		_, artLeveled, _ := petSvc.AddArtifactXP(userID, petsvc.ArtifactDelveCombatXP)

		desc := fmt.Sprintf("Victory! The %s has been defeated.\n💰 +%d gold\n", res.EnemyName, gold)
		if veilKey != nil {
			desc += i18n.T("veil.delve_veil_key", lang) + "\n"
		}
		if artLeveled {
			desc += "\n💠 **Artifact leveled up!** Use `/artifact` to assign your new stat point.\n"
		}
		for _, log := range res.Log {
			desc += "\n" + log
		}

		// Check for Gravewarden Morvain victory → grant Mask of Malveillance
		if res.EnemyName == "Gravewarden Morvain" {
			lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
			announcement, err := c.crimsvc.GrantMaskToPlayer(userID, interaction.ToInt64(i.GuildID), lang)
			if err == nil && announcement != nil {
				desc += "\n\n🎭 **The Mask of Malveillance pulses with dark energy!**"
				go func() {
					ss, _ := c.store.GetServerSetting(interaction.ToInt64(i.GuildID))
					if ss != nil && ss.AnnouncementChannelID != 0 && announcement != nil {
						_, _ = b.Session.ChannelMessageSendEmbed(strconv.FormatInt(ss.AnnouncementChannelID, 10), announcement)
					}
				}()
			}
		}

		embed := &discordgo.MessageEmbed{
			Title:       "⚔️ Victory!",
			Description: desc,
			Color:       0x2ecc71,
		}
		roomDelve := delvesvc.GenerateRoom(s, "en")
		_, comps := c.renderRoomWithFallen(s, &roomDelve, "en")
		c.respond(b, i, embed, comps)
	} else {
		cs := c.svc.GetCombat(userID)
		embed := delvesvc.RenderCombatEmbed(s, cs, c.svc)
		embed.Description = strings.Join(res.Log, "\n")
		comps := delvesvc.CombatRoomButtons()
		c.respond(b, i, embed, comps)
	}
}

func (c *Cog) onCombatSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.resolveCombatAndRender(b, i, "slash")
}

func (c *Cog) onCombatFireball(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.resolveCombatAndRender(b, i, "fireball")
}

func (c *Cog) onCombatDefend(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.resolveCombatAndRender(b, i, "defend")
}

func (c *Cog) onCombatPotion(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	heal := 30
	s.HP += heal
	if s.HP > s.MaxHP {
		s.HP = s.MaxHP
	}
	c.saveSession(s)
	c.resolveCombatAndRender(b, i, "potion")
}

func (c *Cog) onCombatFlee(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	c.svc.EndCombat(userID)
	msg, _ := delvesvc.ResolveFlee(s)
	c.svc.AddFlag(s, "fled_from_depths")
	c.saveSession(s)

	roomDelve := delvesvc.GenerateRoom(s, "en")
	embed, comps := c.renderRoomWithFallen(s, &roomDelve, "en")
	embed.Description = msg + "\n\n" + embed.Description
	c.respond(b, i, embed, comps)
}

func (c *Cog) onFlee(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	msg, _ := delvesvc.ResolveFlee(s)
	c.svc.AddFlag(s, "fled_from_depths")
	c.svc.EndSession(s, "fled")
	c.deleteSession(userID)

	petSvc := petsvc.New(c.store, c.cfg)
	petSvc.AddArtifactXP(userID, petsvc.ArtifactDelveCompleteXP)

	embed := &discordgo.MessageEmbed{
		Title:       "🏃 Escape",
		Description: msg + "\n\nYou emerge into the light, gasping. The Undercroft will wait for your return.",
		Color:       0xf39c12,
	}
	c.respond(b, i, embed, nil)
}

func (c *Cog) onDisarm(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	char, _ := c.store.EnsureCharacter(userID)
	roll := rand.Intn(20) + char.DEX
	success := roll >= 12

	var desc string
	if success {
		desc = "With steady hands, you disarm the trap. The treasure is yours!"
		loot := delvesvc.GenerateLoot(s.Zone, s.Floor, float64(char.LUK)*0.01)
		if loot != nil {
			c.svc.AddItem(s, loot.Item)
			desc += "\n\n" + delvesvc.LootRewardText(loot.Item)
			c.svc.AddFlag(s, "disarmed_treasure")
		}
	} else {
		desc = "You slip! A dart grazes your arm. (-15 HP)"
		s.HP -= 15
		if s.HP < 0 {
			s.HP = 0
		}
	}

	c.saveSession(s)
	roomDelve := delvesvc.GenerateRoom(s, "en")
	embed, comps := c.renderRoomWithFallen(s, &roomDelve, "en")
	embed.Description = desc + "\n\n" + embed.Description
	c.respond(b, i, embed, comps)
}

func (c *Cog) onOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	roll := rand.Intn(100)
	var desc string
	if roll < 30 {
		loot := delvesvc.GenerateLoot(s.Zone, s.Floor, 0)
		if loot != nil {
			c.svc.AddItem(s, loot.Item)
			desc = "You throw the chest open! Inside, a treasure awaits.\n\n" + delvesvc.LootRewardText(loot.Item)
			c.svc.AddFlag(s, "opened_treasure_trap")
		}
	} else if roll < 60 {
		gold := delvesvc.GoldReward(s.Zone, s.Floor) * 3
		s.Gold += gold
		desc = fmt.Sprintf("The chest is packed with coins! +%d gold!", gold)
	} else {
		desc = "The chest is a mimic! It bites you for 20 HP before scuttling away."
		s.HP -= 20
		if s.HP < 0 {
			s.HP = 0
		}
		c.svc.AddFlag(s, "spared_mimic")
	}

	c.saveSession(s)
	roomDelve := delvesvc.GenerateRoom(s, "en")
	embed, comps := c.renderRoomWithFallen(s,
		&roomDelve, "en")
	embed.Description = desc + "\n\n" + embed.Description
	c.respond(b, i, embed, comps)
}

func (c *Cog) onLeave(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	s.RoomsCleared++
	s.Mana += 5
	if s.Mana > s.MaxMana {
		s.Mana = s.MaxMana
	}
	room := c.nextRoom(s, "en")
	embed, comps := c.renderRoomWithFallen(s,
		room, "en")
	embed.Description = "You step away from the opportunity and press onward.\n\n" + embed.Description
	c.respond(b, i, embed, comps)
}

func (c *Cog) onSacrifice(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	hpCost := 30
	if s.MaxHP <= hpCost+10 {
		c.errorMsg(b, i, "You are too frail to make such a sacrifice.")
		return
	}

	s.MaxHP -= hpCost
	if s.HP > s.MaxHP {
		s.HP = s.MaxHP
	}

	loot := delvesvc.GenerateLoot(s.Zone, s.Floor, 0.15)
	c.svc.AddItem(s, loot.Item)
	c.svc.AddFlag(s, "sacrificed_hp")

	c.saveSession(s)

	desc := fmt.Sprintf("You offer a piece of your vitality to the altar. Max HP reduced by %d.\n", hpCost)
	desc += fmt.Sprintf("In return, a %s item appears: **%s**\n\n", loot.Item.Rarity.String(), loot.Item.Name)
	desc += delvesvc.LootRewardText(loot.Item)

	roomDelve := delvesvc.GenerateRoom(s, "en")
	embed, comps := c.renderRoomWithFallen(s,
		&roomDelve, "en")
	embed.Description = desc + "\n\n" + embed.Description
	c.respond(b, i, embed, comps)
}

func (c *Cog) onDesecrate(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	gold := delvesvc.GoldReward(s.Zone, s.Floor) * 5
	s.Gold += gold
	c.svc.AddFlag(s, "desecrated_altar")
	c.saveSession(s)

	roomDelve := delvesvc.GenerateRoom(s, "en")
	embed, comps := c.renderRoomWithFallen(s,
		&roomDelve, "en")
	desc := fmt.Sprintf("You defile the altar and take its gold. (+%d gold)\nA dark mark settles over your soul...", gold)
	embed.Description = desc + "\n\n" + embed.Description
	c.respond(b, i, embed, comps)
}

func (c *Cog) onMerchantBrowse(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	var items []delvesvc.DelveItem
	rarities := []delvesvc.Rarity{delvesvc.Common, delvesvc.Uncommon, delvesvc.Rare}
	for _, r := range rarities {
		loot := delvesvc.GenerateLoot(s.Zone, s.Floor, 0)
		loot.Item.Rarity = r
		items = append(items, loot.Item)
	}
	items = append(items, delvesvc.DelveItem{
		ID: "depth_shard", Name: "Depth Shard", Emoji: "💎", Rarity: delvesvc.Rare, Quantity: 1,
	})

	c.mu.Lock()
	c.merchantOffers[userID] = items
	c.mu.Unlock()

	desc := "\"Take a look, take a look! Fine wares from the deep!\"\n\n"
	var comps []discordgo.MessageComponent

	for idx, item := range items {
		price := (int(item.Rarity) + 1) * 30
		desc += fmt.Sprintf("**%d.** %s %s — 💰 %d gold\n", idx+1, delvesvc.RarityEmoji[item.Rarity], item.Name, price)
	}

	comps = append(comps, components.ActionRow(
		components.Button("1️⃣ Buy 1", components.Encode("delve", "merchant_buy", "0"), discordgo.PrimaryButton),
		components.Button("2️⃣ Buy 2", components.Encode("delve", "merchant_buy", "1"), discordgo.SuccessButton),
		components.Button("3️⃣ Buy 3", components.Encode("delve", "merchant_buy", "2"), discordgo.DangerButton),
	))
	comps = append(comps, components.ActionRow(
		components.Button("💎 Buy Shard", components.Encode("delve", "merchant_buy", "3"), discordgo.PrimaryButton),
		components.Button("🚪 Leave", components.Encode("delve", "leave"), discordgo.SecondaryButton),
	))

	embed := &discordgo.MessageEmbed{
		Title:       "🛒 Wandering Merchant",
		Description: desc,
		Color:       0xf1c40f,
		Footer:      &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Your gold: %d 🪙", s.Gold)},
	}
	c.respond(b, i, embed, comps)
}

func (c *Cog) onMerchantBuy(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	idx := 0
	fmt.Sscanf(rest[0], "%d", &idx)

	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	c.mu.RLock()
	offers := c.merchantOffers[userID]
	c.mu.RUnlock()

	if idx < 0 || idx >= len(offers) {
		c.errorMsg(b, i, "Invalid selection.")
		return
	}

	item := offers[idx]
	price := (int(item.Rarity) + 1) * 30

	if s.Gold < price {
		c.errorMsg(b, i, "Not enough gold!")
		return
	}

	s.Gold -= price
	c.svc.AddItem(s, item)
	c.svc.AddFlag(s, "merchant_purchase")
	c.saveSession(s)

	c.mu.Lock()
	delete(c.merchantOffers, userID)
	c.mu.Unlock()

	roomDelve := delvesvc.GenerateRoom(s, "en")
	embed, comps := c.renderRoomWithFallen(s,
		&roomDelve, "en")
	embed.Description = fmt.Sprintf("You purchased **%s** for %d gold!\n\n", item.Name, price) + embed.Description
	c.respond(b, i, embed, comps)
}

func (c *Cog) onPuzzleSolve(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	riddle := riddlePool[rand.Intn(len(riddlePool))]
	c.mu.Lock()
	c.riddles[userID] = riddle
	c.mu.Unlock()

	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: components.ModalResponse(
			components.Encode("delve", "puzzle_answer"),
			"🧩 Solve the Riddle",
			components.TextInput("answer", riddle.Question, true, "Type your answer...", discordgo.TextInputShort, 1, 100),
		),
	})
}

func (c *Cog) onPuzzleAnswer(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	data := i.ModalSubmitData()
	answer := strings.TrimSpace(strings.ToLower(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value))

	c.mu.RLock()
	riddle, ok := c.riddles[userID]
	c.mu.RUnlock()
	if !ok {
		c.errorMsg(b, i, "No active riddle.")
		return
	}
	c.mu.Lock()
	delete(c.riddles, userID)
	c.mu.Unlock()

	var desc string
	if answer == riddle.Answer {
		loot := delvesvc.GenerateLoot(s.Zone, s.Floor, 0.1)
		c.svc.AddItem(s, loot.Item)
		c.svc.AddFlag(s, "solved_riddle")
		desc = "Correct! The door swings open, revealing a hidden chamber.\n\n" + delvesvc.LootRewardText(loot.Item)
	} else {
		s.HP -= 10
		if s.HP < 0 {
			s.HP = 0
		}
		desc = `"Wrong," groans the statue. A dart hits you from the darkness. (-10 HP)`
	}

	c.saveSession(s)
	roomDelve := delvesvc.GenerateRoom(s, "en")
	embed, comps := c.renderRoomWithFallen(s,
		&roomDelve, "en")
	embed.Description = desc + "\n\n" + embed.Description
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

type riddleEntry struct {
	Question string
	Answer   string
}

var riddlePool = []riddleEntry{
	{Question: "I speak without a mouth and hear without ears. I have no body, but I come alive with wind. What am I?", Answer: "echo"},
	{Question: "The more you take, the more you leave behind. What am I?", Answer: "footsteps"},
	{Question: "I have cities, but no houses. I have mountains, but no trees. I have water, but no fish. What am I?", Answer: "map"},
	{Question: "What can run but never walks, has a mouth but never talks, has a head but never weeps, has a bed but never sleeps?", Answer: "river"},
	{Question: "I am not alive, but I grow; I don't have lungs, but I need air. What am I?", Answer: "fire"},
}

func (c *Cog) onRestTorch(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	if s.Torches <= 0 {
		c.errorMsg(b, i, "You have no torches left!")
		return
	}
	s.Torches--
	heal := s.MaxHP / 2
	s.HP += heal
	if s.HP > s.MaxHP {
		s.HP = s.MaxHP
	}
	s.Mana = s.MaxMana
	c.svc.AddFlag(s, "used_torch")
	c.saveSession(s)

	roomDelve := delvesvc.GenerateRoom(s, "en")
	embed, comps := c.renderRoomWithFallen(s,
		&roomDelve, "en")
	embed.Description = fmt.Sprintf("You light a torch and rest. Recovered %d HP and full Mana.\n\n", heal) + embed.Description
	c.respond(b, i, embed, comps)
}

func (c *Cog) onRestSleep(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	roll := rand.Intn(100)
	if roll < 40 {
		s.HP = s.MaxHP
		s.Mana = s.MaxMana
		c.svc.AddFlag(s, "slept_unprotected")
		c.saveSession(s)
		roomDelve := delvesvc.GenerateRoom(s, "en")
		embed, comps := c.renderRoomWithFallen(s,
			&roomDelve, "en")
		embed.Description = "You sleep deeply and wake fully restored. HP and Mana at maximum!\n\n" + embed.Description
		c.respond(b, i, embed, comps)
	} else {
		s.HP -= 20
		s.Mana = s.MaxMana
		if s.HP < 0 {
			s.HP = 0
		}
		c.svc.AddFlag(s, "ambushed_while_sleeping")
		c.saveSession(s)
		roomDelve := delvesvc.GenerateRoom(s, "en")
		embed, comps := c.renderRoomWithFallen(s,
			&roomDelve, "en")
		embed.Description = "You are ambushed in your sleep! Take 20 damage and scramble to your feet.\n\n" + embed.Description
		c.respond(b, i, embed, comps)
	}
}

func (c *Cog) onNpcHelp(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	loot := delvesvc.GenerateLoot(s.Zone, s.Floor, 0.05)
	c.svc.AddItem(s, loot.Item)
	c.svc.AddFlag(s, "freed_prisoner")
	gold := delvesvc.GoldReward(s.Zone, s.Floor)
	s.Gold += gold
	c.saveSession(s)

	roomDelve := delvesvc.GenerateRoom(s, "en")
	embed, comps := c.renderRoomWithFallen(s,
		&roomDelve, "en")
	desc := fmt.Sprintf("You free the captive. They press a gift into your hands before disappearing into the shadows.\n+%d gold\n", gold)
	desc += "\n" + delvesvc.LootRewardText(loot.Item)
	embed.Description = desc + "\n\n" + embed.Description
	c.respond(b, i, embed, comps)
}

func (c *Cog) onNpcBetray(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	gold := delvesvc.GoldReward(s.Zone, s.Floor) * 4
	s.Gold += gold
	c.svc.AddFlag(s, "betrayed_npc")
	c.saveSession(s)

	roomDelve := delvesvc.GenerateRoom(s, "en")
	embed, comps := c.renderRoomWithFallen(s, &roomDelve, "en")
	desc := fmt.Sprintf("You sell them out for %d gold. Their betrayed eyes follow you as you walk away.", gold)
	embed.Description = desc + "\n\n" + embed.Description
	c.respond(b, i, embed, comps)
}

func (c *Cog) onRescue(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		return
	}
	victimID := interaction.ToInt64(rest[0])
	if userID == victimID {
		c.errorMsg(b, i, "You cannot rescue yourself!")
		return
	}

	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	if s.Torches < 1 {
		s.HP -= 10
		if s.HP <= 0 {
			c.errorMsg(b, i, "You're too weak to rescue anyone!")
			return
		}
		c.saveSession(s)
	} else {
		s.Torches--
		c.saveSession(s)
	}

	victimSession, _ := c.svc.GetSession(victimID)
	if victimSession == nil || victimSession.Status != "fallen" {
		c.errorMsg(b, i, "That player is no longer fallen.")
		return
	}

	c.svc.AddFlag(victimSession, "fell_in_battle")
	c.svc.EndSession(victimSession, "rescued")
	c.store.ClearCooldown(victimID, "delve_death")
	c.deleteSession(victimID)

	c.svc.AddFlag(s, "rescued_another")
	c.saveSession(s)

	dmChannel, err := b.Session.UserChannelCreate(fmt.Sprintf("%d", victimID))
	if err == nil {
		mention := fmt.Sprintf("<@%d>", userID)
		b.Session.ChannelMessageSend(dmChannel.ID, fmt.Sprintf("🆘 **You've been rescued!**\n%s found you in the depths of The Undercroft and pulled you to safety! You may now delve again.", mention))
	}

	roomDelve := delvesvc.GenerateRoom(s, "en")
	embed, comps := c.renderRoomWithFallen(s, &roomDelve, "en")
	embed.Description = fmt.Sprintf("🤝 You rescued <@%d> from the darkness!\n\n", victimID) + embed.Description
	c.respond(b, i, embed, comps)
}

func (c *Cog) onIgnoreFallen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	roomDelve := delvesvc.GenerateRoom(s, "en")
	embed, comps := c.renderRoomWithFallen(s, &roomDelve, "en")
	embed.Description = "You turn away from the cries in the darkness and press on.\n\n" + embed.Description
	c.respond(b, i, embed, comps)
}
