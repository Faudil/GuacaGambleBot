package delve

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	"guacagamblebot/internal/model"
	delvesvc "guacagamblebot/internal/service/delve"
	petsvc "guacagamblebot/internal/service/pets"
)

func (c *Cog) onFloorDeeper(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	// Boss gate check: if the current floor has a boss and flag not set
	bossData := delvesvc.BossForFloor(s.Floor)
	if bossData != nil {
		flag := fmt.Sprintf("boss_f%d", bossData.Floor)
		if !c.svc.HasFlag(s, flag) {
			// Spawn boss combat
			seed := s.Seed + int64(s.RoomsCleared)
			rng := rand.New(rand.NewSource(seed))
			enemy := delvesvc.GenerateEnemy(s.Zone, s.Floor, rng)
			char, _ := c.store.EnsureCharacter(userID)
			playerLevel := 1
			if char != nil {
				playerLevel = char.Level
			}
			delvesvc.ApplyEnemyLevelScaling(enemy, s.Floor, playerLevel, rng)
			enemy.MaxHP = int(float64(enemy.MaxHP) * 1.75)
			enemy.HP = enemy.MaxHP
			enemy.Atk = int(float64(enemy.Atk) * 1.75)
			enemy.Def = int(float64(enemy.Def) * 1.75)
			enemy.Name = bossData.Name
			enemy.Emoji = bossData.Emoji

			c.svc.StartCombat(s, enemy)
			cs := c.svc.GetCombat(userID)
			embed := delvesvc.RenderCombatEmbed(s, cs, c.svc, lang)
			embed.Title = i18n.T("delve.handler.boss_gate_title", lang, map[string]any{"emoji": bossData.Emoji, "name": bossData.Name})
			embed.Description = i18n.T("delve.handler.boss_gate_desc", lang, map[string]any{"floor": s.Floor + 1})
			abilities := delvesvc.GetCombatAbilities(playerLevel)
			weaponEmoji, weaponName := delvesvc.GetWeaponDisplay(c.store, userID)
			comps := delvesvc.CombatRoomButtons(lang, abilities, weaponEmoji, weaponName)
			c.respond(b, i, embed, comps)
			return
		}
	}

	// Normal floor advance
	s.RoomsCleared++
	s.Mana += 2
	if s.Mana > s.MaxMana {
		s.Mana = s.MaxMana
	}

	// Floor clear XP
	fxp := delvesvc.FloorClearXP(s.Floor)
	leveledUp, newLevel, _ := c.store.AddCharacterXP(userID, fxp)

	c.saveSession(s)
	room := c.nextRoom(s, lang)
	embed, comps := c.renderRoomWithFallen(s, room, lang)
	if leveledUp {
		embed.Description += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": newLevel})
	}
	c.respond(b, i, embed, comps)
}

func (c *Cog) onFloorLeave(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	c.svc.AddFlag(s, "left_voluntarily")
	c.svc.EndSession(s, "left")
	c.deleteSession(userID)
	desc := i18n.T("delve.left_voluntarily", lang)
	if n, ok := c.store.PopQuestNotification(userID); ok {
		interaction.SendQuestNotification(b, i, n, lang)
	}
	embed := &discordgo.MessageEmbed{
		Title:       "🌅 " + i18n.T("delve.floor_leave", lang),
		Description: desc,
		Color:       0x2ecc71,
	}
	c.respond(b, i, embed, nil)
}

func (c *Cog) onNavigate(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve. Start one with `/delve`!")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	s.RoomsCleared++
	s.Mana += 2
	if s.Mana > s.MaxMana {
		s.Mana = s.MaxMana
	}

	// Torch burn
	if s.Torches > 0 && rand.Intn(100) < delvesvc.TorchBurnChance(s.Floor) {
		s.Torches--
	}

	// Corridor event (empty room only)
	eventRng := rand.New(rand.NewSource(int64(userID) + int64(s.RoomsCleared)*9999))
	event := delvesvc.RollCorridorEvent(s.Zone, s.Floor, s.Torches == 0, eventRng)
	switch event {
	case delvesvc.CorridorTrap:
		dex := delvesvc.EffectiveDEX(c.store, userID)
		dc := delvesvc.DisarmDC(s.Floor)
		if s.Torches == 0 {
			dc += 3
		}
		if rand.Intn(20)+dex >= dc {
			c.saveSession(s)
			// Show room after narrowly avoiding trap
			room := c.nextRoom(s, lang)
			room.Description = i18n.T("delve.combat.navigate_trap_avoid", lang) + "\n\n" + room.Description
			embed, comps := c.renderRoomWithFallen(s, room, lang)
			c.respond(b, i, embed, comps)
			return
		}
		dmg := delvesvc.TrapDamage(s.Floor)
		s.HP -= dmg
		c.saveSession(s)
		if s.HP <= 0 {
			s.HP = 0
			c.saveSession(s)
			c.applyFallenPenalties(b, i, s, userID, lang)
			return
		}
		room := c.nextRoom(s, lang)
		room.Description = i18n.T("delve.combat.navigate_trap_hit", lang, map[string]any{"damage": dmg}) + "\n\n" + room.Description
		embed, comps := c.renderRoomWithFallen(s, room, lang)
		c.respond(b, i, embed, comps)
		return

	case delvesvc.CorridorAmbush:
		char, _ := c.store.EnsureCharacter(userID)
		playerLevel := 1
		if char != nil {
			playerLevel = char.Level
		}
		seed := s.Seed + int64(s.RoomsCleared)
		rng := rand.New(rand.NewSource(seed))
		enemy := delvesvc.GenerateEnemy(s.Zone, s.Floor, rng)
		delvesvc.ApplyEnemyLevelScaling(enemy, s.Floor, playerLevel, rng)
		c.svc.StartCombat(s, enemy)
		cs := c.svc.GetCombat(userID)
		cs.EnemyFirstStrike = true
		c.saveSession(s)
		embed := delvesvc.RenderCombatEmbed(s, cs, c.svc, lang)
		embed.Description = i18n.T("delve.combat.ambush_desc", lang)
		abilities := delvesvc.GetCombatAbilities(playerLevel)
		weaponEmoji, weaponName := delvesvc.GetWeaponDisplay(c.store, userID)
		c.respond(b, i, embed, delvesvc.CombatRoomButtons(lang, abilities, weaponEmoji, weaponName))
		return

	case delvesvc.CorridorCollapse:
		if s.Torches > 0 {
			s.Torches--
			c.saveSession(s)
		} else {
			dmg := 10 + 3*s.Floor
			s.HP -= dmg
			c.saveSession(s)
			if s.HP <= 0 {
				s.HP = 0
				c.saveSession(s)
				c.applyFallenPenalties(b, i, s, userID, lang)
				return
			}
		}
		room := c.nextRoom(s, lang)
		room.Description = i18n.T("delve.combat.navigate_collapse", lang) + "\n\n" + room.Description
		embed, comps := c.renderRoomWithFallen(s, room, lang)
		c.respond(b, i, embed, comps)
		return

	case delvesvc.CorridorSpectral:
		var effects []string
		json.Unmarshal([]byte(s.StatusEffects), &effects)
		hasCurse := false
		for _, e := range effects {
			if e == "cursed" {
				hasCurse = true
				break
			}
		}
		if !hasCurse {
			effects = append(effects, "cursed")
			jb, _ := json.Marshal(effects)
			s.StatusEffects = string(jb)
			c.saveSession(s)
		}
		room := c.nextRoom(s, lang)
		room.Description = i18n.T("delve.handler.graveyard_spook", lang) + "\n\n" + room.Description
		embed, comps := c.renderRoomWithFallen(s, room, lang)
		c.respond(b, i, embed, comps)
		return

	case delvesvc.CorridorSporeCloud:
		var effects []string
		json.Unmarshal([]byte(s.StatusEffects), &effects)
		effects = append(effects, "poisoned:3")
		jb, _ := json.Marshal(effects)
		s.StatusEffects = string(jb)
		c.saveSession(s)
		room := c.nextRoom(s, lang)
		room.Description = i18n.T("delve.handler.spore_cloud", lang) + "\n\n" + room.Description
		embed, comps := c.renderRoomWithFallen(s, room, lang)
		c.respond(b, i, embed, comps)
		return

	case delvesvc.CorridorSteamVent:
		if s.Torches > 0 {
			s.Torches--
			xp := delvesvc.FloorClearXP(s.Floor)
			leveledUp, newLevel, _ := c.store.AddCharacterXP(userID, xp)
			c.saveSession(s)
			room := c.nextRoom(s, lang)
			room.Description = i18n.T("delve.handler.steam_vent_seal", lang, map[string]any{"xp": xp}) + "\n\n" + room.Description
			if leveledUp {
				room.Description += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": newLevel})
			}
			embed, comps := c.renderRoomWithFallen(s, room, lang)
			c.respond(b, i, embed, comps)
		} else {
			dmg := delvesvc.SteamVentDamage(s.Floor)
			s.HP -= dmg
			c.saveSession(s)
			if s.HP <= 0 {
				s.HP = 0
				c.saveSession(s)
				c.applyFallenPenalties(b, i, s, userID, lang)
				return
			}
			room := c.nextRoom(s, lang)
			room.Description = i18n.T("delve.handler.steam_vent_hit", lang, map[string]any{"damage": dmg}) + "\n\n" + room.Description
			embed, comps := c.renderRoomWithFallen(s, room, lang)
			c.respond(b, i, embed, comps)
		}
		return

	case delvesvc.CorridorWhispers:
		var effects []string
		json.Unmarshal([]byte(s.StatusEffects), &effects)
		effects = append(effects, "enlightened", "cursed")
		jb, _ := json.Marshal(effects)
		s.StatusEffects = string(jb)
		c.saveSession(s)
		room := c.nextRoom(s, lang)
		room.Description = i18n.T("delve.handler.whispers", lang) + "\n\n" + room.Description
		embed, comps := c.renderRoomWithFallen(s, room, lang)
		c.respond(b, i, embed, comps)
		return
	}

	room := c.nextRoom(s, lang)
	embed, comps := c.renderRoomWithFallen(s, room, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onFight(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	zone := s.Zone
	seed := s.Seed + int64(s.RoomsCleared)
	rng := rand.New(rand.NewSource(seed))

	char, _ := c.store.EnsureCharacter(userID)
	playerLevel := 1
	if char != nil {
		playerLevel = char.Level
	}
	abilities := delvesvc.GetCombatAbilities(playerLevel)
	weaponEmoji, weaponName := delvesvc.GetWeaponDisplay(c.store, userID)

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
		embed := delvesvc.RenderCombatEmbed(s, cs, c.svc, lang)
		embed.Title = "💀 Boss: Gravewarden Morvain"
		embed.Description = "The air grows cold. A figure of bone and shadow rises before you, ancient armor creaking with each movement. The Gravewarden has been waiting."
		comps := delvesvc.CombatRoomButtons(lang, abilities, weaponEmoji, weaponName)
		c.respond(b, i, embed, comps)
		return
	}

	enemy := delvesvc.GenerateEnemy(zone, s.Floor, rng)
	delvesvc.ApplyEnemyLevelScaling(enemy, s.Floor, playerLevel, rng)
	c.svc.StartCombat(s, enemy)
	cs := c.svc.GetCombat(userID)
	embed := delvesvc.RenderCombatEmbed(s, cs, c.svc, lang)
	comps := delvesvc.CombatRoomButtons(lang, abilities, weaponEmoji, weaponName)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onDefendStart(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.onFight(b, i)
}

func (c *Cog) applyFallenPenalties(b *interaction.Bot, i *discordgo.InteractionCreate, s *model.DelveSession, userID int64, lang string) {
	if len(s.Flags) > 0 {
		var flags []string
		if err := json.Unmarshal([]byte(s.Flags), &flags); err == nil {
			for _, f := range flags {
				if f == "fell_in_battle" {
					return
				}
			}
		}
	}

	lootDesc := ""
	inv := c.svc.GetInventory(s)
	var kept []delvesvc.DelveItem
	var lost int
	droppedGold := s.Gold * 30 / 100
	s.Gold -= droppedGold
	if s.Gold < 0 {
		s.Gold = 0
	}
	if droppedGold > 0 {
		lootDesc = i18n.T("delve.handler.death_gold_dropped", lang, map[string]any{"gold": droppedGold}) + "\n"
	}
	for _, it := range inv {
		if it.IsSoulbound || lost >= 1 {
			kept = append(kept, it)
			continue
		}
		lost++
		if lootDesc != "" {
			lootDesc += "\n"
		}
		lootDesc += i18n.T("delve.handler.death_item_lost", lang, map[string]any{"item": it.Name})
	}
	bInv, _ := json.Marshal(kept)
	s.Inventory = string(bInv)

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

	c.store.SetCooldown(userID, "delve_death")
	c.saveSession(s)

	petSvc := petsvc.New(c.store, c.cfg)
	petSvc.AddArtifactXP(userID, petsvc.ArtifactDelveCompleteXP)

	rescueMsg := ""
	if s.AutoRescued {
		rescueMsg = "\n\n🐾 " + i18n.T("delve.handler.death_auto_rescue_msg", lang, map[string]any{"pet": s.AutoRescuePet})
	} else {
		rescueMsg = "\n\n🆘 " + i18n.T("delve.handler.death_no_rescue_msg", lang)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "💀 " + i18n.T("delve.handler.death_fallen_title", lang),
		Description: i18n.T("delve.handler.death_fallen_desc", lang) + "\n" + lootDesc + rescueMsg,
		Color:       0xe74c3c,
	}
	c.respond(b, i, embed, nil)
}

func (c *Cog) resolveCombatAndRender(b *interaction.Bot, i *discordgo.InteractionCreate, action string) {
	userID := interaction.ToInt64(i.Member.User.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	res := c.svc.ResolveCombatRound(s, action, lang)

	if res.PlayerDefeated {
		c.applyFallenPenalties(b, i, s, userID, lang)
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

		char, _ := c.store.EnsureCharacter(userID)
		playerLevel := 1
		if char != nil {
			playerLevel = char.Level
		}

		xpEarned := delvesvc.CombatXP(s.Floor, playerLevel)
		if delvesvc.BossForFloor(s.Floor) != nil || res.EnemyName == "Gravewarden Morvain" {
			xpEarned = delvesvc.BossXP(s.Floor)
		}
		// Consume enlightened status
		var effects []string
		json.Unmarshal([]byte(s.StatusEffects), &effects)
		var newEffects []string
		for _, e := range effects {
			if e == "enlightened" {
				xpEarned = xpEarned * 125 / 100
			} else {
				newEffects = append(newEffects, e)
			}
		}
		s.StatusEffects = "[]"
		if len(newEffects) > 0 {
			jb, _ := json.Marshal(newEffects)
			s.StatusEffects = string(jb)
		}
		xpLine := ""
		leveledUp, newLevel, _ := c.store.AddCharacterXP(userID, xpEarned)
		if leveledUp {
			xpLine = "\n🎉 " + i18n.T("delve.handler.level_up", lang, map[string]any{"level": newLevel})
		} else {
			xpLine = "\n✨ " + i18n.T("delve.handler.xp_gain", lang, map[string]any{"xp": xpEarned})
		}

		desc := i18n.T("delve.handler.victory", lang, map[string]any{"enemy": res.EnemyName}) + "\n💰 +" + fmt.Sprintf("%d", gold) + " gold\n"
		desc += xpLine
		if veilKey != nil {
			desc += "\n" + i18n.T("veil.delve_veil_key", lang) + "\n"
		}
		if artLeveled {
			desc += "\n💠 **Artifact leveled up!** Use `/artifact` to assign your new stat point.\n"
		}
		for _, log := range res.Log {
			desc += "\n" + log
		}

		// Check for Gravewarden Morvain victory → grant Mask of Malveillance
		if res.EnemyName == "Gravewarden Morvain" {
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

		// Boss gate flag
		bossData := delvesvc.BossForFloor(s.Floor)
		if bossData != nil {
			flag := fmt.Sprintf("boss_f%d", bossData.Floor)
			c.svc.AddFlag(s, flag)
			desc += "\n\n" + i18n.T("delve.handler.boss_victory", lang, map[string]any{"name": bossData.Name})
		}

		embed, comps := c.buildFloorTransition(s, desc, lang)
		c.respond(b, i, embed, comps)
	} else {
		cs := c.svc.GetCombat(userID)
		embed := delvesvc.RenderCombatEmbed(s, cs, c.svc, lang)
		embed.Description = strings.Join(res.Log, "\n")
		char, _ := c.store.EnsureCharacter(userID)
		playerLevel := 1
		if char != nil {
			playerLevel = char.Level
		}
		abilities := delvesvc.GetCombatAbilities(playerLevel)
		weaponEmoji, weaponName := delvesvc.GetWeaponDisplay(c.store, userID)
		comps := delvesvc.CombatRoomButtons(lang, abilities, weaponEmoji, weaponName)
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

func (c *Cog) onCombatPowerBlow(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.resolveCombatAndRender(b, i, "power_blow")
}

func (c *Cog) onCombatMend(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.resolveCombatAndRender(b, i, "mend")
}

func (c *Cog) onCombatPotion(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	if s.Potions <= 0 {
		c.errorMsg(b, i, "You have no potions left!")
		return
	}
	s.Potions--
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
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	dex := delvesvc.EffectiveDEX(c.store, userID)
	dc := delvesvc.CombatFleeDC(s.Floor)
	if s.Torches == 0 {
		dc += 2
	}
	roll := rand.Intn(20) + dex
	if roll >= dc {
		c.svc.EndCombat(userID)
		lostGold := s.Gold / 4
		s.Gold -= lostGold
		if s.Gold < 0 {
			s.Gold = 0
		}
		msg := fmt.Sprintf("You slip away from combat, leaving %d gold behind.", lostGold)
		c.svc.AddFlag(s, "fled_from_depths")
		c.saveSession(s)

		embed, comps := c.buildFloorTransition(s, msg, lang)
		c.respond(b, i, embed, comps)
	} else {
		freeDmg := c.svc.GetCombat(userID).Enemy.Atk + rand.Intn(4)
		s.HP -= freeDmg
		c.saveSession(s)
		if s.HP <= 0 {
			s.HP = 0
			c.applyFallenPenalties(b, i, s, userID, lang)
			return
		}
		msg := fmt.Sprintf("You stumble! The %s lands a free hit for %d damage!", c.svc.GetCombat(userID).Enemy.Name, freeDmg)
		cs := c.svc.GetCombat(userID)
		embed := delvesvc.RenderCombatEmbed(s, cs, c.svc, lang)
		embed.Description = msg
		char2, _ := c.store.EnsureCharacter(userID)
		pl := 1
		if char2 != nil {
			pl = char2.Level
		}
		abilities := delvesvc.GetCombatAbilities(pl)
		weaponEmoji, weaponName := delvesvc.GetWeaponDisplay(c.store, userID)
		c.respond(b, i, embed, delvesvc.CombatRoomButtons(lang, abilities, weaponEmoji, weaponName))
	}
}

func (c *Cog) onFlee(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	dex := delvesvc.EffectiveDEX(c.store, userID)
	dc := delvesvc.FleeDC(s.Floor)
	if s.Torches == 0 {
		dc += 3
	}
	roll := rand.Intn(20) + dex
	if roll >= dc {
		msg := "You slip away before the enemy can react, retreating to the safety of the passage."
		c.svc.AddFlag(s, "fled_room")
		c.saveSession(s)
		c.svc.EndCombat(userID)
		embed, comps := c.buildFloorTransition(s, msg, lang)
		c.respond(b, i, embed, comps)
		return
	}

	// Flee failed — forced into combat with enemy first strike
	seed := s.Seed + int64(s.RoomsCleared)
	rng := rand.New(rand.NewSource(seed))
	enemy := delvesvc.GenerateEnemy(s.Zone, s.Floor, rng)
	char, _ := c.store.EnsureCharacter(userID)
	playerLevel := 1
	if char != nil {
		playerLevel = char.Level
	}
	delvesvc.ApplyEnemyLevelScaling(enemy, s.Floor, playerLevel, rng)
	c.svc.StartCombat(s, enemy)
	cs := c.svc.GetCombat(userID)
	cs.EnemyFirstStrike = true

	embed := delvesvc.RenderCombatEmbed(s, cs, c.svc, lang)
	embed.Description = i18n.T("delve.handler.flee_fail", lang)
	abilities := delvesvc.GetCombatAbilities(playerLevel)
	weaponEmoji, weaponName := delvesvc.GetWeaponDisplay(c.store, userID)
	comps := delvesvc.CombatRoomButtons(lang, abilities, weaponEmoji, weaponName)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onDisarm(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	dex := delvesvc.EffectiveDEX(c.store, userID)
	dc := delvesvc.DisarmDC(s.Floor)
	if s.Torches == 0 {
		dc += 3
	}
	roll := rand.Intn(20) + dex
	success := roll >= dc

	var desc string
	if success {
		char, _ := c.store.EnsureCharacter(userID)
		luk := 0
		if char != nil {
			luk = char.LUK
		}
		desc = "With steady hands, you disarm the trap. The treasure is yours!"
		loot := delvesvc.GenerateLoot(s.Zone, s.Floor, float64(luk)*0.01)
		if loot != nil {
			c.svc.AddItem(s, loot.Item)
			desc += "\n\n" + delvesvc.LootRewardText(loot.Item)
			c.svc.AddFlag(s, "disarmed_treasure")
		}
	} else {
		dmg := delvesvc.TrapDamage(s.Floor)
		desc = fmt.Sprintf("You slip! A trap triggers for %d damage.", dmg)
		s.HP -= dmg
		if s.HP < 0 {
			s.HP = 0
		}
	}

	c.saveSession(s)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if s.HP <= 0 {
		c.applyFallenPenalties(b, i, s, userID, lang)
		return
	}
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	mimicChance := delvesvc.MimicChance(s.Floor)
	roll := rand.Intn(100)
	var desc string
	if roll < mimicChance {
		dmg := delvesvc.MimicDamage(s.Floor)
		desc = fmt.Sprintf("The chest is a mimic! It bites you for %d damage before scuttling away.", dmg)
		s.HP -= dmg
		if s.HP < 0 {
			s.HP = 0
		}
		c.svc.AddFlag(s, "spared_mimic")
	} else if roll < mimicChance+30 {
		loot := delvesvc.GenerateLoot(s.Zone, s.Floor, 0)
		if loot != nil {
			c.svc.AddItem(s, loot.Item)
			desc = "You throw the chest open! Inside, a treasure awaits.\n\n" + delvesvc.LootRewardText(loot.Item)
			c.svc.AddFlag(s, "opened_treasure_trap")
		}
	} else {
		gold := delvesvc.GoldReward(s.Zone, s.Floor) * 3
		s.Gold += gold
		desc = fmt.Sprintf("The chest is packed with coins! +%d gold!", gold)
	}

	c.saveSession(s)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if s.HP <= 0 {
		c.applyFallenPenalties(b, i, s, userID, lang)
		return
	}
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onLeave(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	c.saveSession(s)
	embed, comps := c.buildFloorTransition(s, "You step away from the opportunity.", lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onSacrifice(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}

	hpCost := 15 + s.Floor*5
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

	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.buildFloorTransition(s, desc, lang)
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
	var effects []string
	json.Unmarshal([]byte(s.StatusEffects), &effects)
	effects = append(effects, "marked")
	jb, _ := json.Marshal(effects)
	s.StatusEffects = string(jb)
	c.svc.AddFlag(s, "desecrated_altar")
	c.saveSession(s)

	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	desc := fmt.Sprintf("You defile the altar and take its gold. (+%d gold)\nA dark mark settles over your soul... Enemies will hit harder.", gold)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onMerchantBrowse(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

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
	potionPrice := delvesvc.PotionPrice(s.Floor)
	torchPrice := delvesvc.TorchPrice(s.Floor)
	cachePrice := delvesvc.MysteryCachePrice(s.Floor)

	c.mu.Lock()
	c.merchantOffers[userID] = items
	c.mu.Unlock()

	basePrice := delvesvc.MerchantPriceBase(s.Floor)
	desc := "\"Take a look, take a look! Fine wares from the deep!\"\n\n"
	var comps []discordgo.MessageComponent

	for idx, item := range items {
		p := (int(item.Rarity) + 1) * basePrice
		desc += fmt.Sprintf("**%d.** %s %s — 💰 %d gold\n", idx+1, delvesvc.RarityEmoji[item.Rarity], item.Name, p)
	}
	desc += fmt.Sprintf("**4.** 💎 Depth Shard — 💰 %d gold\n", basePrice*4)
	desc += fmt.Sprintf("**5.** 🧪 Potion (heals 30 HP) — 💰 %d gold\n", potionPrice)
	desc += fmt.Sprintf("**6.** 🔦 Torch — 💰 %d gold\n", torchPrice)
	desc += fmt.Sprintf("**7.** 🎁 Mystery Cache (random item!) — 💰 %d gold\n", cachePrice)

	// Store extra items for buy handler
	c.mu.Lock()
	c.merchantExtra[userID] = map[string]int{
		"potion_price": potionPrice,
		"torch_price":  torchPrice,
		"cache_price":  cachePrice,
	}
	c.mu.Unlock()

	comps = append(comps, components.ActionRow(
		components.Button("1️⃣ Buy 1", components.Encode("delve", "merchant_buy", "0"), discordgo.PrimaryButton),
		components.Button("2️⃣ Buy 2", components.Encode("delve", "merchant_buy", "1"), discordgo.SuccessButton),
		components.Button("3️⃣ Buy 3", components.Encode("delve", "merchant_buy", "2"), discordgo.DangerButton),
	))
	comps = append(comps, components.ActionRow(
		components.Button("💎 Shard", components.Encode("delve", "merchant_buy", "3"), discordgo.PrimaryButton),
		components.Button("🧪 Potion", components.Encode("delve", "merchant_buy", "4"), discordgo.SuccessButton),
		components.Button("🔦 Torch", components.Encode("delve", "merchant_buy", "5"), discordgo.SecondaryButton),
	))
	comps = append(comps, components.ActionRow(
		components.Button("🎁 Mystery", components.Encode("delve", "merchant_buy", "6"), discordgo.DangerButton),
		components.Button("🚪 "+i18n.T("delve.buttons.leave", lang), components.Encode("delve", "leave"), discordgo.SecondaryButton),
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

	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	basePrice := delvesvc.MerchantPriceBase(s.Floor)

	var price int
	var desc string

	switch idx {
	case 0, 1, 2:
		item := offers[idx]
		price = (int(item.Rarity) + 1) * basePrice
		if s.Gold < price {
			c.errorMsg(b, i, "Not enough gold!")
			return
		}
		s.Gold -= price
		c.svc.AddItem(s, item)
		desc = fmt.Sprintf("You purchased **%s** for %d gold!", item.Name, price)

	case 3: // Depth Shard
		price = basePrice * 4
		if s.Gold < price {
			c.errorMsg(b, i, "Not enough gold!")
			return
		}
		s.Gold -= price
		shard := delvesvc.DelveItem{
			ID: "depth_shard", Name: "Depth Shard", Emoji: "💎", Rarity: delvesvc.Rare, Quantity: 1,
		}
		c.svc.AddItem(s, shard)
		desc = fmt.Sprintf("You purchased **Depth Shard** for %d gold!", price)

	case 4: // Potion
		price = delvesvc.PotionPrice(s.Floor)
		if s.Gold < price {
			c.errorMsg(b, i, "Not enough gold!")
			return
		}
		s.Gold -= price
		s.Potions++
		if s.Potions > 3 {
			s.Potions = 3
		}
		desc = fmt.Sprintf("You purchased a **Potion** for %d gold! (%d/%d)", price, s.Potions, 3)

	case 5: // Torch
		price = delvesvc.TorchPrice(s.Floor)
		if s.Gold < price {
			c.errorMsg(b, i, "Not enough gold!")
			return
		}
		s.Gold -= price
		s.Torches++
		desc = fmt.Sprintf("You purchased a **Torch** for %d gold! (%d)", price, s.Torches)

	case 6: // Mystery Cache
		price = delvesvc.MysteryCachePrice(s.Floor)
		if s.Gold < price {
			c.errorMsg(b, i, "Not enough gold!")
			return
		}
		s.Gold -= price
		char, _ := c.store.EnsureCharacter(userID)
		luk := 0.0
		if char != nil {
			luk = float64(char.LUK) * 0.02
		}
		loot := delvesvc.GenerateLoot(s.Zone, s.Floor, luk)
		if loot != nil {
			c.svc.AddItem(s, loot.Item)
			desc = fmt.Sprintf("You crack open the Mystery Cache! Inside: %s\n%s", loot.Item.Name, delvesvc.LootRewardText(loot.Item))
		} else {
			desc = "The cache is empty... Bad luck."
		}

	default:
		c.errorMsg(b, i, "Invalid selection.")
		return
	}

	// Apply haggle discount/markup
	c.mu.RLock()
	if extra, ok := c.merchantExtra[userID]; ok {
		if extra["haggle_discount"] > 0 {
			price = price * 70 / 100
		} else if extra["haggle_markup"] > 0 {
			price = price * 120 / 100
		}
	}
	c.mu.RUnlock()

	if s.Gold < price {
		c.errorMsg(b, i, "Not enough gold!")
		return
	}
	s.Gold -= price

	c.svc.AddFlag(s, "merchant_purchase")
	c.saveSession(s)

	c.mu.Lock()
	delete(c.merchantOffers, userID)
	delete(c.merchantExtra, userID)
	c.mu.Unlock()

	embed, comps := c.buildFloorTransition(s, desc, lang)
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
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.buildFloorTransition(s, desc, lang)
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
	{Question: "What has keys but can't open locks?", Answer: "piano"},
	{Question: "I can be cracked, made, told, and played. What am I?", Answer: "joke"},
	{Question: "Forward I am heavy, backward I am not. What am I?", Answer: "ton"},
	{Question: "I have a neck but no head, two arms but no hands. What am I?", Answer: "shirt"},
	{Question: "What can you catch but not throw?", Answer: "cold"},
	{Question: "I have a face but no eyes, hands but no arms. What am I?", Answer: "clock"},
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
	heal := s.MaxHP * 35 / 100
	s.HP += heal
	if s.HP > s.MaxHP {
		s.HP = s.MaxHP
	}
	manaRestore := s.MaxMana * 50 / 100
	s.Mana += manaRestore
	if s.Mana > s.MaxMana {
		s.Mana = s.MaxMana
	}
	c.svc.AddFlag(s, "used_torch")
	c.saveSession(s)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	desc := fmt.Sprintf("You light a torch and rest. Recovered %d HP and %d Mana.", heal, manaRestore)
	embed, comps := c.buildFloorTransition(s, desc, lang)
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
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if roll < 25 {
		s.HP = s.MaxHP
		s.Mana = s.MaxMana
		c.svc.AddFlag(s, "slept_unprotected")
		c.saveSession(s)
		embed, comps := c.buildFloorTransition(s, "You sleep deeply and wake fully restored. HP and Mana at maximum!", lang)
		c.respond(b, i, embed, comps)
	} else {
		dmg := delvesvc.AmbushDamage(s.Floor)
		s.HP -= dmg
		s.Mana = s.MaxMana/2 + s.MaxMana*50/100
		if s.Mana > s.MaxMana {
			s.Mana = s.MaxMana
		}
		if s.HP < 0 {
			s.HP = 0
		}
		c.svc.AddFlag(s, "ambushed_while_sleeping")
		c.saveSession(s)
		if s.HP <= 0 {
			c.applyFallenPenalties(b, i, s, userID, lang)
			return
		}
		embed, comps := c.buildFloorTransition(s, fmt.Sprintf("You are ambushed in your sleep! Take %d damage and scramble to your feet.", dmg), lang)
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

	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	desc := fmt.Sprintf("You free the captive. They press a gift into your hands before disappearing into the shadows.\n+%d gold\n", gold)
	desc += "\n" + delvesvc.LootRewardText(loot.Item)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onNpcBetray(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	if rand.Intn(100) < delvesvc.BackfireChance(s.Floor) {
		char, _ := c.store.EnsureCharacter(userID)
		playerLevel := 1
		if char != nil {
			playerLevel = char.Level
		}
		seed := s.Seed + int64(s.RoomsCleared)
		rng := rand.New(rand.NewSource(seed))
		enemy := delvesvc.GenerateEnemy(s.Zone, s.Floor, rng)
		delvesvc.ApplyEnemyLevelScaling(enemy, s.Floor, playerLevel, rng)
		c.svc.StartCombat(s, enemy)
		cs := c.svc.GetCombat(userID)
		cs.EnemyFirstStrike = true
		c.saveSession(s)
		embed := delvesvc.RenderCombatEmbed(s, cs, c.svc, lang)
		embed.Description = i18n.T("delve.handler.doppelganger", lang)
		abilities := delvesvc.GetCombatAbilities(playerLevel)
		weaponEmoji, weaponName := delvesvc.GetWeaponDisplay(c.store, userID)
		c.respond(b, i, embed, delvesvc.CombatRoomButtons(lang, abilities, weaponEmoji, weaponName))
		return
	}

	gold := delvesvc.GoldReward(s.Zone, s.Floor) * 3
	s.Gold += gold
	c.svc.AddFlag(s, "betrayed_npc")
	c.saveSession(s)

	desc := fmt.Sprintf("You sell them out for %d gold. Their betrayed eyes follow you as you walk away.", gold)
	embed, comps := c.buildFloorTransition(s, desc, lang)
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

	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	desc := fmt.Sprintf("🤝 You rescued <@%d> from the darkness!", victimID)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onIgnoreFallen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.buildFloorTransition(s, "You turn away from the cries in the darkness and press on.", lang)
	c.respond(b, i, embed, comps)
}

// === New Room Handlers ===

func (c *Cog) onShrinePray(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	char, _ := c.store.EnsureCharacter(userID)
	luk := 5
	if char != nil {
		luk = char.LUK
	}
	dc := delvesvc.ShrinePrayDC(s.Floor)
	roll := rand.Intn(20) + luk
	var desc string
	if roll >= dc {
		var effects []string
		json.Unmarshal([]byte(s.StatusEffects), &effects)
		effects = append(effects, "blessed")
		jb, _ := json.Marshal(effects)
		s.StatusEffects = string(jb)
		desc = i18n.T("delve.handler.shrine_pray_success", lang)
	} else {
		dmg := delvesvc.ShrineBacklash(s.Floor)
		s.HP -= dmg
		if s.HP < 0 {
			s.HP = 0
		}
		desc = i18n.T("delve.handler.shrine_pray_fail", lang, map[string]any{"damage": dmg})
	}
	c.saveSession(s)
	if s.HP <= 0 {
		c.applyFallenPenalties(b, i, s, userID, lang)
		return
	}
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onShrineDonate(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	cost := delvesvc.ShrineDonateCost(s.Floor)
	if s.Gold < cost {
		c.errorMsg(b, i, "Not enough gold!")
		return
	}
	s.Gold -= cost
	var effects []string
	json.Unmarshal([]byte(s.StatusEffects), &effects)
	effects = append(effects, "blessed")
	jb, _ := json.Marshal(effects)
	s.StatusEffects = string(jb)
	c.svc.AddFlag(s, "shrine_blessed")
	c.saveSession(s)
	desc := i18n.T("delve.handler.shrine_donate", lang, map[string]any{"gold": cost})
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onShrineDefile(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	gold := delvesvc.GoldReward(s.Zone, s.Floor) * 3
	s.Gold += gold
	var effects []string
	json.Unmarshal([]byte(s.StatusEffects), &effects)
	effects = append(effects, "cursed")
	jb, _ := json.Marshal(effects)
	s.StatusEffects = string(jb)
	c.svc.AddFlag(s, "shrine_defiled")
	c.saveSession(s)
	desc := i18n.T("delve.handler.shrine_defile", lang, map[string]any{"gold": gold})
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onTombOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	var desc string
	if rand.Intn(100) < 40 {
		loot := delvesvc.GenerateLoot(s.Zone, s.Floor, 0)
		if loot != nil {
			delvesvc.AssignSetName(&loot.Item, s.Zone)
			c.svc.AddItem(s, loot.Item)
			c.svc.AddFlag(s, "tomb_raider")
			desc = i18n.T("delve.handler.tomb_open_success", lang) + "\n\n" + delvesvc.LootRewardText(loot.Item)
		} else {
			desc = i18n.T("delve.handler.tomb_open_empty", lang)
		}
	} else {
		seed := s.Seed + int64(s.RoomsCleared)
		rng := rand.New(rand.NewSource(seed))
		enemy := delvesvc.GenerateEnemy(s.Zone, s.Floor, rng)
		char, _ := c.store.EnsureCharacter(userID)
		pl := 1
		if char != nil {
			pl = char.Level
		}
		delvesvc.ApplyEnemyLevelScaling(enemy, s.Floor, pl, rng)
		c.svc.StartCombat(s, enemy)
		cs := c.svc.GetCombat(userID)
		embed := delvesvc.RenderCombatEmbed(s, cs, c.svc, lang)
		embed.Description = i18n.T("delve.handler.tomb_open_guardian", lang)
		abilities := delvesvc.GetCombatAbilities(pl)
		weaponEmoji, weaponName := delvesvc.GetWeaponDisplay(c.store, userID)
		c.respond(b, i, embed, delvesvc.CombatRoomButtons(lang, abilities, weaponEmoji, weaponName))
		return
	}
	c.saveSession(s)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onTombRespect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	var effects []string
	json.Unmarshal([]byte(s.StatusEffects), &effects)
	effects = append(effects, "blessed")
	jb, _ := json.Marshal(effects)
	s.StatusEffects = string(jb)
	c.saveSession(s)
	desc := i18n.T("delve.handler.tomb_respect", lang)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onGardenHarvest(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	var desc string
	if rand.Intn(100) < 50 {
		heal := 15 + 5*s.Floor
		s.HP += heal
		if s.HP > s.MaxHP {
			s.HP = s.MaxHP
		}
		desc = i18n.T("delve.handler.garden_harvest_heal", lang, map[string]any{"heal": heal})
	} else {
		var effects []string
		json.Unmarshal([]byte(s.StatusEffects), &effects)
		effects = append(effects, "poisoned:3")
		jb, _ := json.Marshal(effects)
		s.StatusEffects = string(jb)
		desc = i18n.T("delve.handler.garden_harvest_poison", lang)
	}
	if rand.Float64() < 0.25 {
		seed := randomGardenSeed()
		c.store.AddItemRaw(c.store.DB, userID, seed, 1)
		it := items.Get(seed)
		seedName := seed
		seedEmoji := "🌱"
		if it != nil {
			seedName = it.Name
			seedEmoji = it.Emoji
		}
		desc += "\n\n" + fmt.Sprintf("%s You found a **%s**!", seedEmoji, seedName)
	}
	c.saveSession(s)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onGardenBurn(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	if s.Torches <= 0 {
		c.errorMsg(b, i, i18n.T("delve.handler.no_torches_burn", lang))
		return
	}
	s.Torches--
	heal := s.MaxHP * 20 / 100
	s.HP += heal
	if s.HP > s.MaxHP {
		s.HP = s.MaxHP
	}
	loot := delvesvc.GenerateLoot(s.Zone, s.Floor, 0.05)
	desc := i18n.T("delve.handler.garden_burn", lang, map[string]any{"heal": heal})
	if loot != nil {
		delvesvc.AssignSetName(&loot.Item, s.Zone)
		c.svc.AddItem(s, loot.Item)
		desc += "\n\n" + delvesvc.LootRewardText(loot.Item)
	}
	if rand.Float64() < 0.30 {
		seed := randomGardenSeed()
		c.store.AddItemRaw(c.store.DB, userID, seed, 1)
		it := items.Get(seed)
		seedName := seed
		seedEmoji := "🌱"
		if it != nil {
			seedName = it.Name
			seedEmoji = it.Emoji
		}
		desc += "\n\n" + fmt.Sprintf("%s You also found a **%s**!", seedEmoji, seedName)
	}
	c.saveSession(s)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func randomGardenSeed() string {
	seeds := []string{
		"wheat_seed", "corn_seed", "carrot_seed", "potato_seed",
		"tomato_seed", "pumpkin_seed", "coffee_seed", "cocoa_seed",
		"strawberry_seed",
	}
	return seeds[rand.Intn(len(seeds))]
}

func (c *Cog) onForgeTemper(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	if s.Torches <= 0 {
		c.errorMsg(b, i, i18n.T("delve.handler.no_torches_forge", lang))
		return
	}
	s.Torches--
	var effects []string
	json.Unmarshal([]byte(s.StatusEffects), &effects)
	effects = append(effects, "fortified")
	jb, _ := json.Marshal(effects)
	s.StatusEffects = string(jb)
	c.saveSession(s)
	desc := i18n.T("delve.handler.forge_temper", lang)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onForgeScavenge(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	gold := 30 + 10*s.Floor
	s.Gold += gold
	key := delvesvc.MaybeDropKey(s.Zone, s.Floor)
	desc := i18n.T("delve.handler.forge_scavenge", lang, map[string]any{"gold": gold})
	if key != nil {
		s.Keys++
		desc += "\n\n" + i18n.T("delve.handler.forge_scavenge_key", lang)
	}
	c.saveSession(s)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onRiftGaze(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	intVal := delvesvc.EffectiveINT(c.store, userID)
	dc := delvesvc.DisarmDC(s.Floor)
	var desc string
	if rand.Intn(20)+intVal >= dc {
		var effects []string
		json.Unmarshal([]byte(s.StatusEffects), &effects)
		effects = append(effects, "enlightened")
		jb, _ := json.Marshal(effects)
		s.StatusEffects = string(jb)
		desc = i18n.T("delve.handler.rift_gaze_success", lang)
	} else {
		dmg := 10 + 3*s.Floor
		s.HP -= dmg
		if s.HP < 0 {
			s.HP = 0
		}
		desc = i18n.T("delve.handler.rift_gaze_fail", lang, map[string]any{"damage": dmg})
	}
	c.saveSession(s)
	if s.HP <= 0 {
		c.applyFallenPenalties(b, i, s, userID, lang)
		return
	}
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onRiftDisturb(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	seed := s.Seed + int64(s.RoomsCleared)
	rng := rand.New(rand.NewSource(seed))
	enemy := delvesvc.GenerateEnemy(s.Zone, s.Floor, rng)
	char, _ := c.store.EnsureCharacter(userID)
	pl := 1
	if char != nil {
		pl = char.Level
	}
	delvesvc.ApplyEnemyLevelScaling(enemy, s.Floor, pl, rng)
	enemy.MaxHP = int(float64(enemy.MaxHP) * 1.5)
	enemy.HP = enemy.MaxHP
	enemy.Atk = int(float64(enemy.Atk) * 1.5)

	c.svc.StartCombat(s, enemy)
	cs := c.svc.GetCombat(userID)
	embed := delvesvc.RenderCombatEmbed(s, cs, c.svc, lang)
	embed.Description = i18n.T("delve.handler.rift_disturb", lang)

	abilities := delvesvc.GetCombatAbilities(pl)
	weaponEmoji, weaponName := delvesvc.GetWeaponDisplay(c.store, userID)
	c.respond(b, i, embed, delvesvc.CombatRoomButtons(lang, abilities, weaponEmoji, weaponName))
}

func (c *Cog) onLockedKey(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	if s.Keys <= 0 {
		c.errorMsg(b, i, i18n.T("delve.handler.no_keys", lang))
		return
	}
	s.Keys--
	loot := delvesvc.GenerateLoot(s.Zone, s.Floor, 0.05)
	desc := i18n.T("delve.handler.locked_key_success", lang)
	if loot != nil {
		gold := delvesvc.GoldReward(s.Zone, s.Floor)
		s.Gold += gold
		c.svc.AddItem(s, loot.Item)
		c.svc.AddFlag(s, "key_master")
		desc += "\n\n" + delvesvc.LootRewardText(loot.Item)
		desc += fmt.Sprintf("\n💰 +%d gold", gold)
	}
	c.saveSession(s)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onLockedForce(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	char, _ := c.store.EnsureCharacter(userID)
	str := 5
	if char != nil {
		str = char.STR
	}
	dc := delvesvc.ForceDoorDC(s.Floor)
	var desc string
	if rand.Intn(20)+str >= dc {
		loot := delvesvc.GenerateLoot(s.Zone, s.Floor, 0)
		desc = i18n.T("delve.handler.locked_force_success", lang)
		if loot != nil {
			gold := delvesvc.GoldReward(s.Zone, s.Floor)
			s.Gold += gold
			c.svc.AddItem(s, loot.Item)
			desc += "\n\n" + delvesvc.LootRewardText(loot.Item)
			desc += fmt.Sprintf("\n💰 +%d gold", gold)
		}
	} else {
		dmg := delvesvc.TrapDamage(s.Floor)
		s.HP -= dmg
		if s.HP < 0 {
			s.HP = 0
		}
		desc = i18n.T("delve.handler.locked_force_fail", lang, map[string]any{"damage": dmg})
	}
	c.saveSession(s)
	if s.HP <= 0 {
		c.applyFallenPenalties(b, i, s, userID, lang)
		return
	}
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onNpcIntimidate(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	char, _ := c.store.EnsureCharacter(userID)
	str := 5
	if char != nil {
		str = char.STR
	}
	dc := delvesvc.IntimidateDC(s.Floor)
	if rand.Intn(20)+str >= dc {
		loot := delvesvc.GenerateLoot(s.Zone, s.Floor, 0)
		desc := i18n.T("delve.handler.npc_intimidate_success", lang)
		if loot != nil {
			c.svc.AddItem(s, loot.Item)
			desc += "\n\n" + delvesvc.LootRewardText(loot.Item)
		}
		c.saveSession(s)
		embed, comps := c.buildFloorTransition(s, desc, lang)
		c.respond(b, i, embed, comps)
	} else {
		c.onNpcBetray(b, i)
	}
}

func (c *Cog) onMerchantHaggle(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	char, _ := c.store.EnsureCharacter(userID)
	luk := 0
	if char != nil {
		luk = char.LUK
	}
	dc := delvesvc.ShrinePrayDC(s.Floor)
	if rand.Intn(20)+luk >= dc {
		c.mu.Lock()
		if c.merchantExtra[userID] == nil {
			c.merchantExtra[userID] = make(map[string]int)
		}
		c.merchantExtra[userID]["haggle_discount"] = 1
		c.mu.Unlock()
		desc := i18n.T("delve.handler.merchant_haggle_success", lang)
		embed, comps := c.buildFloorTransition(s, desc, lang)
		c.respond(b, i, embed, comps)
	} else {
		c.mu.Lock()
		if c.merchantExtra[userID] == nil {
			c.merchantExtra[userID] = make(map[string]int)
		}
		c.merchantExtra[userID]["haggle_markup"] = 1
		c.mu.Unlock()
		desc := i18n.T("delve.handler.merchant_haggle_fail", lang)
		embed, comps := c.buildFloorTransition(s, desc, lang)
		c.respond(b, i, embed, comps)
	}
}

func (c *Cog) onRestBandage(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, "No active delve.")
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	if s.Potions <= 0 {
		c.errorMsg(b, i, i18n.T("delve.handler.no_potions_bandage", lang))
		return
	}
	s.Potions--
	s.HP += 15
	if s.HP > s.MaxHP {
		s.HP = s.MaxHP
	}
	c.saveSession(s)
	desc := i18n.T("delve.handler.rest_bandage", lang)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}
