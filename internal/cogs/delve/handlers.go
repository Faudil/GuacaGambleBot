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
	jsvc "guacagamblebot/internal/service/journal"
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
			embed.Title = i18n.T("delve.handler.boss_gate_title", lang, map[string]any{"emoji": bossData.Emoji, "name": delvesvc.BossName(bossData.Name, lang)})
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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

	if text, dm := jsvc.SceneLine(c.store, userID, "delve", lang); text != "" {
		interaction.SendJournalScene(b, i, text, dm)
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
		embed.Title = i18n.T("delve.handler.gravewarden_title", lang)
		embed.Description = i18n.T("delve.handler.gravewarden_desc", lang)
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
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, c.noSessionMsg(i))
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	var effects []string
	json.Unmarshal([]byte(s.StatusEffects), &effects)
	effects = append(effects, "guarded")
	jb, _ := json.Marshal(effects)
	s.StatusEffects = string(jb)
	c.svc.AddFlag(s, "defended_room")
	c.saveSession(s)

	desc := i18n.T("delve.handler.defend_room", lang)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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

		desc := i18n.T("delve.handler.victory", lang, map[string]any{"enemy": delvesvc.EnemyName(res.EnemyName, lang)}) + "\n" + i18n.T("delve.handler.gold_gain", lang, map[string]any{"gold": fmt.Sprintf("%d", gold)}) + "\n"
		desc += xpLine
		if veilKey != nil {
			desc += "\n" + i18n.T("veil.delve_veil_key", lang) + "\n"
		}
		if artLeveled {
			desc += "\n" + i18n.T("delve.handler.artifact_leveled", lang) + "\n"
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
			desc += "\n\n" + i18n.T("delve.handler.boss_victory", lang, map[string]any{"name": delvesvc.BossName(bossData.Name, lang)})
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
		c.errorMsg(b, i, c.noSessionMsg(i))
		return
	}
	if s.Potions <= 0 {
		lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
		c.errorMsg(b, i, i18n.T("delve.handler.no_potions", lang))
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
		msg := i18n.T("delve.handler.flee_combat_success", lang, map[string]any{"gold": fmt.Sprintf("%d", lostGold)})
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
		msg := i18n.T("delve.handler.flee_combat_fail", lang, map[string]any{
			"enemy":  delvesvc.EnemyName(c.svc.GetCombat(userID).Enemy.Name, lang),
			"damage": fmt.Sprintf("%d", freeDmg),
		})
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
		msg := i18n.T("delve.handler.flee_success", lang)
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
		c.errorMsg(b, i, c.noSessionMsg(i))
		return
	}

	dex := delvesvc.EffectiveDEX(c.store, userID)
	dc := delvesvc.DisarmDC(s.Floor)
	if s.Torches == 0 {
		dc += 3
	}
	roll := rand.Intn(20) + dex
	success := roll >= dc

	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	var desc string
	if success {
		char, _ := c.store.EnsureCharacter(userID)
		luk := 0
		if char != nil {
			luk = char.LUK
		}
		desc = i18n.T("delve.handler.disarm_success", lang)
		loot := delvesvc.GenerateLoot(s.Zone, s.Floor, float64(luk)*0.01)
		if loot != nil {
			c.svc.AddItem(s, loot.Item)
			desc += "\n\n" + delvesvc.LootRewardText(loot.Item, lang)
			c.svc.AddFlag(s, "disarmed_treasure")
		}
	} else {
		dmg := delvesvc.TrapDamage(s.Floor)
		desc = i18n.T("delve.handler.disarm_fail", lang, map[string]any{"damage": fmt.Sprintf("%d", dmg)})
		s.HP -= dmg
		if s.HP < 0 {
			s.HP = 0
		}
	}

	c.saveSession(s)
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
		c.errorMsg(b, i, c.noSessionMsg(i))
		return
	}

	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	mimicChance := delvesvc.MimicChance(s.Floor)
	roll := rand.Intn(100)
	var desc string
	if roll < mimicChance {
		dmg := delvesvc.MimicDamage(s.Floor)
		desc = i18n.T("delve.handler.mimic", lang, map[string]any{"damage": fmt.Sprintf("%d", dmg)})
		s.HP -= dmg
		if s.HP < 0 {
			s.HP = 0
		}
		c.svc.AddFlag(s, "spared_mimic")
	} else if roll < mimicChance+30 {
		loot := delvesvc.GenerateLoot(s.Zone, s.Floor, 0)
		if loot != nil {
			c.svc.AddItem(s, loot.Item)
			desc = i18n.T("delve.handler.chest_open", lang) + "\n\n" + delvesvc.LootRewardText(loot.Item, lang)
			c.svc.AddFlag(s, "opened_treasure_trap")
		}
	} else {
		gold := delvesvc.GoldReward(s.Zone, s.Floor) * 3
		s.Gold += gold
		desc = i18n.T("delve.handler.chest_gold", lang, map[string]any{"gold": fmt.Sprintf("%d", gold)})
	}

	c.saveSession(s)
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
		c.errorMsg(b, i, c.noSessionMsg(i))
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	c.saveSession(s)
	embed, comps := c.buildFloorTransition(s, i18n.T("delve.handler.leave_room", lang), lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onSacrifice(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, c.noSessionMsg(i))
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	hpCost := 15 + s.Floor*5
	if s.MaxHP <= hpCost+10 {
		c.errorMsg(b, i, i18n.T("delve.too_frail", lang))
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

	desc := i18n.T("delve.handler.sacrifice_desc", lang, map[string]any{"cost": fmt.Sprintf("%d", hpCost)}) + "\n"
	desc += i18n.T("delve.handler.sacrifice_reward", lang, map[string]any{
		"rarity": delvesvc.RarityName(loot.Item.Rarity, lang),
		"item":   delvesvc.DelveItemName(loot.Item, lang),
	}) + "\n\n"
	desc += delvesvc.LootRewardText(loot.Item, lang)

	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onDesecrate(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, c.noSessionMsg(i))
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
	desc := i18n.T("delve.handler.desecrate_desc", lang, map[string]any{"gold": fmt.Sprintf("%d", gold)})
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onMerchantBrowse(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, c.noSessionMsg(i))
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
	desc := i18n.T("delve.handler.merchant_welcome", lang) + "\n\n"
	var comps []discordgo.MessageComponent

	for idx, item := range items {
		p := (int(item.Rarity) + 1) * basePrice
		desc += i18n.T("delve.handler.merchant_item_line", lang, map[string]any{
			"n":     fmt.Sprintf("%d", idx+1),
			"emoji": delvesvc.RarityEmoji[item.Rarity],
			"item":  delvesvc.DelveItemName(item, lang),
			"price": fmt.Sprintf("%d", p),
		}) + "\n"
	}
	desc += i18n.T("delve.handler.merchant_shard_line", lang, map[string]any{
		"item":  i18n.T("delve.loot.depth_shard", lang),
		"price": fmt.Sprintf("%d", basePrice*4),
	}) + "\n"
	desc += i18n.T("delve.handler.merchant_potion_line", lang, map[string]any{"price": fmt.Sprintf("%d", potionPrice)}) + "\n"
	desc += i18n.T("delve.handler.merchant_torch_line", lang, map[string]any{"price": fmt.Sprintf("%d", torchPrice)}) + "\n"
	desc += i18n.T("delve.handler.merchant_cache_line", lang, map[string]any{"price": fmt.Sprintf("%d", cachePrice)})

	// Store extra items for buy handler
	c.mu.Lock()
	c.merchantExtra[userID] = map[string]int{
		"potion_price": potionPrice,
		"torch_price":  torchPrice,
		"cache_price":  cachePrice,
	}
	c.mu.Unlock()

	comps = append(comps, components.ActionRow(
		components.Button(i18n.T("delve.merchant.buy1", lang), components.Encode("delve", "merchant_buy", "0"), discordgo.PrimaryButton),
		components.Button(i18n.T("delve.merchant.buy2", lang), components.Encode("delve", "merchant_buy", "1"), discordgo.SuccessButton),
		components.Button(i18n.T("delve.merchant.buy3", lang), components.Encode("delve", "merchant_buy", "2"), discordgo.DangerButton),
	))
	comps = append(comps, components.ActionRow(
		components.Button(i18n.T("delve.merchant.shard", lang), components.Encode("delve", "merchant_buy", "3"), discordgo.PrimaryButton),
		components.Button(i18n.T("delve.merchant.potion", lang), components.Encode("delve", "merchant_buy", "4"), discordgo.SuccessButton),
		components.Button(i18n.T("delve.merchant.torch", lang), components.Encode("delve", "merchant_buy", "5"), discordgo.SecondaryButton),
	))
	comps = append(comps, components.ActionRow(
		components.Button(i18n.T("delve.merchant.mystery", lang), components.Encode("delve", "merchant_buy", "6"), discordgo.DangerButton),
		components.Button("🚪 "+i18n.T("delve.buttons.leave", lang), components.Encode("delve", "leave"), discordgo.SecondaryButton),
	))

	embed := &discordgo.MessageEmbed{
		Title:       i18n.T("delve.handler.merchant_title", lang),
		Description: desc,
		Color:       0xf1c40f,
		Footer:      &discordgo.MessageEmbedFooter{Text: i18n.T("delve.handler.merchant_gold", lang, map[string]any{"gold": fmt.Sprintf("%d", s.Gold)})},
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
		c.errorMsg(b, i, c.noSessionMsg(i))
		return
	}

	c.mu.RLock()
	offers := c.merchantOffers[userID]
	c.mu.RUnlock()

	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	if idx < 0 || idx >= len(offers) {
		c.errorMsg(b, i, i18n.T("delve.handler.invalid_selection", lang))
		return
	}

	basePrice := delvesvc.MerchantPriceBase(s.Floor)

	var price int
	var desc string

	switch idx {
	case 0, 1, 2:
		item := offers[idx]
		price = (int(item.Rarity) + 1) * basePrice
		if s.Gold < price {
			c.errorMsg(b, i, i18n.T("delve.not_enough_gold", lang))
			return
		}
		s.Gold -= price
		c.svc.AddItem(s, item)
		desc = i18n.T("delve.handler.purchase", lang, map[string]any{"item": delvesvc.DelveItemName(item, lang), "gold": fmt.Sprintf("%d", price)})

	case 3: // Depth Shard
		price = basePrice * 4
		if s.Gold < price {
			c.errorMsg(b, i, i18n.T("delve.not_enough_gold", lang))
			return
		}
		s.Gold -= price
		shard := delvesvc.DelveItem{
			ID: "depth_shard", Name: "Depth Shard", Emoji: "💎", Rarity: delvesvc.Rare, Quantity: 1,
		}
		c.svc.AddItem(s, shard)
		desc = i18n.T("delve.handler.purchase", lang, map[string]any{"item": i18n.T("delve.loot.depth_shard", lang), "gold": fmt.Sprintf("%d", price)})

	case 4: // Potion
		price = delvesvc.PotionPrice(s.Floor)
		if s.Gold < price {
			c.errorMsg(b, i, i18n.T("delve.not_enough_gold", lang))
			return
		}
		s.Gold -= price
		s.Potions++
		if s.Potions > 3 {
			s.Potions = 3
		}
		desc = i18n.T("delve.handler.purchase_potion", lang, map[string]any{
			"gold": fmt.Sprintf("%d", price), "p": fmt.Sprintf("%d", s.Potions), "m": "3",
		})

	case 5: // Torch
		price = delvesvc.TorchPrice(s.Floor)
		if s.Gold < price {
			c.errorMsg(b, i, i18n.T("delve.not_enough_gold", lang))
			return
		}
		s.Gold -= price
		s.Torches++
		desc = i18n.T("delve.handler.purchase_torch", lang, map[string]any{
			"gold": fmt.Sprintf("%d", price), "t": fmt.Sprintf("%d", s.Torches),
		})

	case 6: // Mystery Cache
		price = delvesvc.MysteryCachePrice(s.Floor)
		if s.Gold < price {
			c.errorMsg(b, i, i18n.T("delve.not_enough_gold", lang))
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
			desc = i18n.T("delve.handler.cache_open", lang, map[string]any{
				"item": delvesvc.DelveItemName(loot.Item, lang),
				"text": delvesvc.LootRewardText(loot.Item, lang),
			})
		} else {
			desc = i18n.T("delve.handler.cache_empty", lang)
		}

	default:
		c.errorMsg(b, i, i18n.T("delve.handler.invalid_selection", lang))
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
		c.errorMsg(b, i, i18n.T("delve.not_enough_gold", lang))
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
		c.errorMsg(b, i, c.noSessionMsg(i))
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	riddle := riddleEntry{ID: riddlePool[rand.Intn(len(riddlePool))]}
	c.mu.Lock()
	c.riddles[userID] = riddle
	c.mu.Unlock()

	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: components.ModalResponse(
			components.Encode("delve", "puzzle_answer"),
			i18n.T("delve.riddle.modal_title", lang),
			components.TextInput("answer", i18n.T("delve.riddle."+riddle.ID+".question", lang), true, i18n.T("delve.riddle.placeholder", lang), discordgo.TextInputShort, 1, 100),
		),
	})
}

func (c *Cog) onPuzzleAnswer(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, c.noSessionMsg(i))
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	data := i.ModalSubmitData()
	answer := normalizeAnswer(data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)

	c.mu.RLock()
	riddle, ok := c.riddles[userID]
	c.mu.RUnlock()
	if !ok {
		c.errorMsg(b, i, i18n.T("delve.no_riddle", lang))
		return
	}
	c.mu.Lock()
	delete(c.riddles, userID)
	c.mu.Unlock()

	var desc string
	if answer == normalizeAnswer(i18n.T("delve.riddle."+riddle.ID+".answer", lang)) {
		loot := delvesvc.GenerateLoot(s.Zone, s.Floor, 0.1)
		c.svc.AddItem(s, loot.Item)
		c.svc.AddFlag(s, "solved_riddle")
		desc = i18n.T("delve.handler.riddle_correct", lang) + "\n\n" + delvesvc.LootRewardText(loot.Item, lang)
	} else {
		s.HP -= 10
		if s.HP < 0 {
			s.HP = 0
		}
		desc = i18n.T("delve.handler.riddle_wrong", lang)
	}

	c.saveSession(s)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

type riddleEntry struct {
	ID string
}

var riddlePool = []string{"echo", "footsteps", "map", "river", "fire", "piano", "joke", "ton", "shirt", "cold", "clock"}

var diacritics = strings.NewReplacer(
	"à", "a", "â", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"î", "i", "ï", "i",
	"ô", "o", "ö", "o",
	"ù", "u", "û", "u", "ü", "u",
	"ç", "c",
	"œ", "oe",
)

func normalizeAnswer(s string) string {
	return strings.TrimSpace(strings.ToLower(diacritics.Replace(s)))
}

func (c *Cog) onRestTorch(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, c.noSessionMsg(i))
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))

	if s.Torches <= 0 {
		c.errorMsg(b, i, i18n.T("delve.no_torches", lang))
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
	desc := i18n.T("delve.handler.rest_torch", lang, map[string]any{
		"hp": fmt.Sprintf("%d", heal), "mana": fmt.Sprintf("%d", manaRestore),
	})
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onRestSleep(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, c.noSessionMsg(i))
		return
	}

	roll := rand.Intn(100)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if roll < 25 {
		s.HP = s.MaxHP
		s.Mana = s.MaxMana
		c.svc.AddFlag(s, "slept_unprotected")
		c.saveSession(s)
		embed, comps := c.buildFloorTransition(s, i18n.T("delve.handler.sleep_restored", lang), lang)
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
		embed, comps := c.buildFloorTransition(s, i18n.T("delve.handler.sleep_ambush", lang, map[string]any{"damage": fmt.Sprintf("%d", dmg)}), lang)
		c.respond(b, i, embed, comps)
	}
}

func (c *Cog) onNpcHelp(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, c.noSessionMsg(i))
		return
	}

	loot := delvesvc.GenerateLoot(s.Zone, s.Floor, 0.05)
	c.svc.AddItem(s, loot.Item)
	c.svc.AddFlag(s, "freed_prisoner")
	gold := delvesvc.GoldReward(s.Zone, s.Floor)
	s.Gold += gold
	c.saveSession(s)

	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	desc := i18n.T("delve.handler.npc_help", lang, map[string]any{"gold": fmt.Sprintf("%d", gold)}) + "\n"
	desc += "\n" + delvesvc.LootRewardText(loot.Item, lang)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onNpcBetray(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, c.noSessionMsg(i))
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

	desc := i18n.T("delve.handler.npc_betray", lang, map[string]any{"gold": fmt.Sprintf("%d", gold)})
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
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if userID == victimID {
		c.errorMsg(b, i, i18n.T("delve.rescue_self", lang))
		return
	}

	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, c.noSessionMsg(i))
		return
	}

	if s.Torches < 1 {
		s.HP -= 10
		if s.HP <= 0 {
			c.errorMsg(b, i, i18n.T("delve.rescue_no_torches_hp", lang))
			return
		}
		c.saveSession(s)
	} else {
		s.Torches--
		c.saveSession(s)
	}

	victimSession, _ := c.svc.GetSession(victimID)
	if victimSession == nil || victimSession.Status != "fallen" {
		c.errorMsg(b, i, i18n.T("delve.rescue_already_gone", lang))
		return
	}
	if victimSession.GuildID != s.GuildID || victimSession.Floor != s.Floor {
		c.errorMsg(b, i, i18n.T("delve.handler.rescue_wrong_floor", lang))
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
		b.Session.ChannelMessageSend(dmChannel.ID, i18n.T("delve.handler.rescue_dm", lang, map[string]any{"rescuer": mention}))
	}

	desc := i18n.T("delve.handler.rescue_done", lang, map[string]any{"user": fmt.Sprintf("<@%d>", victimID)})
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onIgnoreFallen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, c.noSessionMsg(i))
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.buildFloorTransition(s, i18n.T("delve.ignore_fallen", lang), lang)
	c.respond(b, i, embed, comps)
}

// === New Room Handlers ===

func (c *Cog) onShrinePray(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, c.noSessionMsg(i))
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
		c.errorMsg(b, i, c.noSessionMsg(i))
		return
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	cost := delvesvc.ShrineDonateCost(s.Floor)
	if s.Gold < cost {
		c.errorMsg(b, i, i18n.T("delve.not_enough_gold", lang))
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
			desc = i18n.T("delve.handler.tomb_open_success", lang) + "\n\n" + delvesvc.LootRewardText(loot.Item, lang)
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
		desc += "\n\n" + i18n.T("delve.handler.garden_seed_found", lang, map[string]any{"emoji": seedEmoji, "item": seedName})
	}
	c.saveSession(s)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onGardenBurn(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, c.noSessionMsg(i))
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
		desc += "\n\n" + delvesvc.LootRewardText(loot.Item, lang)
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
		desc += "\n\n" + i18n.T("delve.handler.garden_seed_found_also", lang, map[string]any{"emoji": seedEmoji, "item": seedName})
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
		desc += "\n\n" + delvesvc.LootRewardText(loot.Item, lang)
		desc += "\n" + i18n.T("delve.handler.gold_gain", lang, map[string]any{"gold": fmt.Sprintf("%d", gold)})
	}
	c.saveSession(s)
	embed, comps := c.buildFloorTransition(s, desc, lang)
	c.respond(b, i, embed, comps)
}

func (c *Cog) onLockedForce(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	s := c.loadSession(userID)
	if s == nil {
		c.errorMsg(b, i, c.noSessionMsg(i))
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
			desc += "\n\n" + delvesvc.LootRewardText(loot.Item, lang)
			desc += "\n" + i18n.T("delve.handler.gold_gain", lang, map[string]any{"gold": fmt.Sprintf("%d", gold)})
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
			desc += "\n\n" + delvesvc.LootRewardText(loot.Item, lang)
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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
		c.errorMsg(b, i, c.noSessionMsg(i))
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




