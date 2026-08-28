package delve

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	crimsvc "guacagamblebot/internal/service/criminality"
	delvesvc "guacagamblebot/internal/service/delve"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store          *store.Store
	cfg            *config.Config
	svc            *delvesvc.Service
	crimsvc        *crimsvc.Service
	qsvc           *questssvc.Service
	sessions       map[int64]*model.DelveSession
	mu             sync.RWMutex
	merchantOffers map[int64][]delvesvc.DelveItem
	merchantExtra  map[int64]map[string]int
	riddles        map[int64]riddleEntry
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{
		store:          s,
		cfg:            cfg,
		svc:            delvesvc.New(s, cfg),
		crimsvc:        crimsvc.New(s, cfg),
		qsvc:           questssvc.New(s, cfg),
		sessions:       make(map[int64]*model.DelveSession),
		merchantOffers: make(map[int64][]delvesvc.DelveItem),
		merchantExtra:  make(map[int64]map[string]int),
		riddles:        make(map[int64]riddleEntry),
	}

	r.Slash("delve", "cmd.delve.desc", c.onSlashDelve)
	r.Slash("myjourney", "cmd.myjourney.desc", c.onSlashJourney)

	r.Prefix("delve", c.onPrefixDelve)
	r.Prefix("myjourney", c.onPrefixJourney)
	r.Prefix("gauntlet", c.onPrefixGauntlet)

	handlers := map[string]interaction.ComponentHandler{
		"nav":               c.onNavigate,
		"fight":             c.onFight,
		"defend_start":      c.onDefendStart,
		"flee":              c.onFlee,
		"disarm":            c.onDisarm,
		"open":              c.onOpen,
		"leave":             c.onLeave,
		"sacrifice":         c.onSacrifice,
		"desecrate":         c.onDesecrate,
		"merchant_browse":   c.onMerchantBrowse,
		"merchant_buy":      c.onMerchantBuy,
		"puzzle_solve":      c.onPuzzleSolve,
		"rest_torch":        c.onRestTorch,
		"rest_sleep":        c.onRestSleep,
		"npc_help":          c.onNpcHelp,
		"npc_betray":        c.onNpcBetray,
		"combat_slash":      c.onCombatSlash,
		"combat_fireball":   c.onCombatFireball,
		"combat_power_blow": c.onCombatPowerBlow,
		"combat_mend":       c.onCombatMend,
		"combat_defend":     c.onCombatDefend,
		"combat_potion":     c.onCombatPotion,
		"combat_flee":       c.onCombatFlee,
		"rescue":            c.onRescue,
		"ignore_fallen":     c.onIgnoreFallen,
		"floor_deeper":      c.onFloorDeeper,
		"floor_leave":       c.onFloorLeave,
		"shrine_pray":       c.onShrinePray,
		"shrine_donate":     c.onShrineDonate,
		"shrine_defile":     c.onShrineDefile,
		"tomb_open":         c.onTombOpen,
		"tomb_respect":      c.onTombRespect,
		"garden_harvest":    c.onGardenHarvest,
		"garden_burn":       c.onGardenBurn,
		"forge_temper":      c.onForgeTemper,
		"forge_scavenge":    c.onForgeScavenge,
		"forge_magnet":      c.onForgeMagnet,
		"rift_gaze":         c.onRiftGaze,
		"rift_disturb":      c.onRiftDisturb,
		"locked_key":        c.onLockedKey,
		"locked_force":      c.onLockedForce,
		"key_take":          c.onKeyTake,
		"npc_intimidate":    c.onNpcIntimidate,
		"merchant_haggle":   c.onMerchantHaggle,
		"rest_bandage":      c.onRestBandage,
		"archive_read":      c.onArchiveRead,
		"archive_search":    c.onArchiveSearch,
		"fountain_coin":     c.onFountainCoin,
		"fountain_drink":    c.onFountainDrink,
		"ossuary_search":    c.onOssuarySearch,
		"ossuary_rest":      c.onOssuaryRest,
		"warden_help":       c.onWardenHelp,
		"warden_listen":     c.onWardenListen,
	}
	for action, h := range handlers {
		r.Component("delve", action, h)
	}

	r.Modal("delve", "puzzle_answer", c.onPuzzleAnswer)
}

func (c *Cog) loadSessionRaw(userID int64) *model.DelveSession {
	c.mu.RLock()
	s, ok := c.sessions[userID]
	c.mu.RUnlock()
	if ok {
		return s
	}
	s, err := c.svc.GetSession(userID)
	if err != nil || s == nil {
		return nil
	}
	c.mu.Lock()
	c.sessions[userID] = s
	c.mu.Unlock()
	return s
}

func (c *Cog) loadSession(userID int64) *model.DelveSession {
	s := c.loadSessionRaw(userID)
	if s != nil && s.Status == "active" {
		return s
	}
	return nil
}

func (c *Cog) getFallenSession(userID int64) *model.DelveSession {
	s := c.loadSessionRaw(userID)
	if s != nil && s.Status == "fallen" {
		return s
	}
	return nil
}

func (c *Cog) saveSession(s *model.DelveSession) {
	c.mu.Lock()
	c.sessions[s.UserID] = s
	c.mu.Unlock()
	c.svc.SaveSession(s)
}

func (c *Cog) deleteSession(userID int64) {
	c.mu.Lock()
	delete(c.sessions, userID)
	delete(c.merchantOffers, userID)
	delete(c.riddles, userID)
	c.mu.Unlock()
}

func (c *Cog) nextRoom(session *model.DelveSession, lang string) *delvesvc.Room {
	session.Floor++
	session.Zone = delvesvc.ZoneForFloor(session.Floor)

	var effects []string
	json.Unmarshal([]byte(session.StatusEffects), &effects)
	var newEffects []string
	poisonTicked := false
	for _, e := range effects {
		if strings.HasPrefix(e, "poisoned") {
			parts := strings.SplitN(e, ":", 2)
			turns := 3
			if len(parts) == 2 {
				v, _ := strconv.Atoi(parts[1])
				turns = v
			}
			turns--
			if turns > 0 {
				newEffects = append(newEffects, fmt.Sprintf("poisoned:%d", turns))
			}
			poisonTicked = true
		} else {
			newEffects = append(newEffects, e)
		}
	}
	if poisonTicked {
		dmg := delvesvc.PoisonPerRoom(session.Floor)
		session.HP -= dmg
		if session.HP < 0 {
			session.HP = 0
		}
	}
	b, _ := json.Marshal(newEffects)
	session.StatusEffects = string(b)

	gen := delvesvc.GenerateRoom(session, lang)
	return &gen
}

func (c *Cog) renderRoom(session *model.DelveSession, room *delvesvc.Room, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := delvesvc.BuildRoomEmbed(session, room, lang, c.svc)
	comps := delvesvc.BuildRoomComponents(room, lang)
	return embed, comps
}

func (c *Cog) renderRoomWithFallen(session *model.DelveSession, room *delvesvc.Room, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed, comps := c.renderRoom(session, room, lang)
	fallen, err := c.store.GetFallenPlayersOnFloor(session.GuildID, int64(session.Floor), 2)
	if err == nil && len(fallen) > 0 {
		delvesvc.MaybeAddRescueOverlay(room, fallen, session.UserID, session.GuildID, c.store, lang)
		embed.Description = room.Description
		comps = delvesvc.BuildRoomComponents(room, lang)
		_ = fallen
	}
	return embed, comps
}

func (c *Cog) buildFloorTransition(session *model.DelveSession, summary string, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	char, _ := c.store.EnsureCharacter(session.UserID)
	playerLevel := 1
	if char != nil {
		playerLevel = char.Level
	}
	danger := delvesvc.CalcDanger(session.Floor, playerLevel)

	title := i18n.T("delve.floor_title", lang, map[string]any{"floor": fmt.Sprintf("%d", session.Floor)})
	dangerLine := delvesvc.DescribeDanger(danger, lang)
	hpLine := i18n.T("delve.floor_summary_hp", lang, map[string]any{
		"hp":       fmt.Sprintf("%d", session.HP),
		"max_hp":   fmt.Sprintf("%d", session.MaxHP),
		"mana":     fmt.Sprintf("%d", session.Mana),
		"max_mana": fmt.Sprintf("%d", session.MaxMana),
	})
	itemsLine := i18n.T("delve.floor_summary_items", lang, map[string]any{
		"torches": fmt.Sprintf("%d", session.Torches),
		"keys":    fmt.Sprintf("%d", session.Keys),
		"gold":    fmt.Sprintf("%d", session.Gold),
	})
	desc := dangerLine + "\n\n" + summary + "\n\n" + hpLine + "\n" + itemsLine
	potionLine := i18n.T("delve.room.potions_line", lang, map[string]any{"potions": fmt.Sprintf("%d", session.Potions)})
	desc += "\n" + potionLine

	var pets []int64
	json.Unmarshal([]byte(session.DeployedPets), &pets)
	if len(pets) > 0 {
		desc += "\n🐾 " + i18n.T("delve.room.pets_line", lang, map[string]any{"pets": fmt.Sprintf("%d", len(pets))})
	}

	var effects []string
	json.Unmarshal([]byte(session.StatusEffects), &effects)
	if len(effects) > 0 {
		var displayEffects []string
		for _, e := range effects {
			statusKey := e
			if i := strings.Index(e, ":"); i > 0 {
				statusKey = e[:i]
			}
			displayEffects = append(displayEffects, i18n.T("delve.status."+statusKey, lang))
		}
		desc += "\n⚠️ " + strings.Join(displayEffects, ", ")
	}

	if session.Torches == 0 {
		desc += "\n🌑 **" + i18n.T("delve.combat.darkness_warning", lang) + "**"
	}

	if danger.IsPunished {
		desc += "\n⚠️ " + i18n.T("delve.handler.weakness_warning", lang)
	}

	color := components.ColorWarning
	switch {
	case danger.Skulls >= 4:
		color = components.ColorDanger
	case danger.Skulls >= 2:
		color = components.ColorWarning
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: desc,
		Color:       color,
	}
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("⬇️ "+i18n.T("delve.floor_deeper", lang), components.Encode("delve", "floor_deeper"), discordgo.SuccessButton),
			components.Button("🚪 "+i18n.T("delve.floor_leave", lang), components.Encode("delve", "floor_leave"), discordgo.SecondaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) respond(b *interaction.Bot, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent) {
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) respondNew(b *interaction.Bot, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed, comps []discordgo.MessageComponent) {
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) errorMsg(b *interaction.Bot, i *discordgo.InteractionCreate, msg string) {
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) noSessionMsg(i *discordgo.InteractionCreate) string {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	return i18n.T("delve.no_session", lang, map[string]any{"prefix": "/"})
}

func (c *Cog) startDelve(b *interaction.Bot, userID, guildID, channelID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent, *model.DelveSession, error) {
	s, err := c.svc.StartSession(userID, guildID, channelID)
	if err != nil {
		return nil, nil, nil, err
	}
	lang := c.store.GetLanguage(guildID)
	c.saveSession(s)
	room := delvesvc.GenerateRoom(s, lang)
	// Tutorial: while the player is on the delve step, the first room is the
	// Vault Key chamber instead of a random (possibly lethal) room.
	if c.qsvc.IsTutorialOnDelveStep(userID) {
		room = delvesvc.VaultKeyRoom(lang)
	}
	embed, comps := c.renderRoom(s, &room, lang)
	return embed, comps, s, nil
}

const staleSessionTimeout = 2 * time.Hour

func (c *Cog) canStartDelve(userID int64, lang string) (bool, string) {
	raw := c.loadSessionRaw(userID)
	if raw != nil {
		if raw.Status == "active" {
			if time.Since(raw.UpdatedAt) >= staleSessionTimeout {
				c.svc.EndSession(raw, "abandoned")
				c.deleteSession(userID)
			} else {
				return false, i18n.T("delve.already_active", lang)
			}
		}
		if raw.Status == "fallen" {
			if raw.AutoRescued && raw.DiedAt != nil {
				elapsed := time.Since(*raw.DiedAt)
				if elapsed >= 5*time.Minute {
					c.svc.AddFlag(raw, "fell_in_battle")
					c.svc.EndSession(raw, "automatically rescued")
					c.store.ClearCooldown(userID, "delve_death")
					c.deleteSession(userID)
					return false, i18n.T("delve.auto_rescued", lang, map[string]any{"pet": raw.AutoRescuePet})
				}
				remaining := 5*time.Minute - elapsed
				return false, i18n.T("delve.auto_rescue_wait", lang, map[string]any{"pet": raw.AutoRescuePet, "minutes": fmt.Sprintf("%d", int(remaining.Minutes())+1)})
			}
			// A new day has dawned and the death cooldown has elapsed:
			// the fallen player is freed and can delve again.
			midnight := time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour)
			if ok, _ := c.store.CheckCooldown(userID, "delve_death", time.Until(midnight)); ok {
				c.svc.EndSession(raw, "rescued")
				c.store.ClearCooldown(userID, "delve_death")
				c.deleteSession(userID)
				return false, i18n.T("delve.revived", lang)
			}
			return false, i18n.T("delve.fallen_wait", lang, map[string]any{"floor": fmt.Sprintf("%d", raw.Floor)})
		}
	}
	midnight := time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour)
	ok, _ := c.store.CheckCooldown(userID, "delve_death", time.Until(midnight))
	if !ok {
		return false, i18n.T("delve.cooldown_death", lang)
	}
	free, err := c.store.FreeSlots(c.store.DB, userID)
	if err != nil {
		return false, i18n.T("delve.error", lang)
	}
	if free <= 0 {
		return false, i18n.T("inventory.full", lang)
	}
	return true, ""
}

func (c *Cog) onSlashDelve(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	allowed, msg := c.canStartDelve(userID, lang)
	if !allowed {
		if strings.HasPrefix(msg, "🆘") || strings.HasPrefix(msg, "⏳") {
			_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{Title: "🔔 Rescue Status", Description: msg, Color: components.ColorSuccess},
					},
				},
			})
		} else {
			c.errorMsg(b, i, msg)
		}
		return
	}
	embed, comps, _, err := c.startDelve(b, userID, interaction.ToInt64(i.GuildID), interaction.ToInt64(i.ChannelID))
	if err != nil {
		c.errorMsg(b, i, i18n.T("delve.failed_start", lang, map[string]any{"err": err.Error()}))
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixDelve(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	allowed, msg := c.canStartDelve(userID, lang)
	if !allowed {
		_, _ = s.ChannelMessageSend(m.ChannelID, msg)
		return
	}
	embed, comps, _, err := c.startDelve(b, userID, interaction.ToInt64(m.GuildID), interaction.ToInt64(m.ChannelID))
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("delve.failed_start", lang, map[string]any{"err": err.Error()}))
		return
	}
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) onSlashJourney(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	pages, err := delvesvc.BuildChronicle(userID, c.svc, lang)
	if err != nil {
		c.errorMsg(b, i, i18n.T("delve.failed_chronicle", lang, map[string]any{"err": err.Error()}))
		return
	}
	if len(pages) == 0 {
		pages = append(pages, &discordgo.MessageEmbed{
			Title:       i18n.T("delve.chronicle.title", lang),
			Description: i18n.T("delve.chronicle.empty", lang),
			Color:       components.ColorArcane,
		})
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, pages[0], nil))
}

func (c *Cog) onPrefixJourney(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	pages, err := delvesvc.BuildChronicle(userID, c.svc, lang)
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, i18n.T("delve.failed_chronicle", lang, map[string]any{"err": err.Error()}))
		return
	}
	if len(pages) == 0 {
		pages = append(pages, &discordgo.MessageEmbed{
			Title:       i18n.T("delve.chronicle.title", lang),
			Description: i18n.T("delve.chronicle.empty", lang),
			Color:       components.ColorArcane,
		})
	}
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{pages[0]},
	})
}
