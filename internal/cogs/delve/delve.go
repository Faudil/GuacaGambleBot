package delve

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	crimsvc "guacagamblebot/internal/service/criminality"
	delvesvc "guacagamblebot/internal/service/delve"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store          *store.Store
	cfg            *config.Config
	svc            *delvesvc.Service
	crimsvc        *crimsvc.Service
	sessions       map[int64]*model.DelveSession
	mu             sync.RWMutex
	merchantOffers map[int64][]delvesvc.DelveItem
	riddles        map[int64]riddleEntry
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{
		store:          s,
		cfg:            cfg,
		svc:            delvesvc.New(s, cfg),
		crimsvc:        crimsvc.New(s, cfg),
		sessions:       make(map[int64]*model.DelveSession),
		merchantOffers: make(map[int64][]delvesvc.DelveItem),
		riddles:        make(map[int64]riddleEntry),
	}

	r.Slash("delve", "Enter The Undercroft dungeon", c.onSlashDelve)
	r.Slash("myjourney", "View your personal chronicle", c.onSlashJourney)

	r.Prefix("delve", c.onPrefixDelve)
	r.Prefix("myjourney", c.onPrefixJourney)
	r.Prefix("gauntlet", c.onPrefixGauntlet)

	handlers := map[string]interaction.ComponentHandler{
		"nav":             c.onNavigate,
		"fight":           c.onFight,
		"defend_start":    c.onDefendStart,
		"flee":            c.onFlee,
		"disarm":          c.onDisarm,
		"open":            c.onOpen,
		"leave":           c.onLeave,
		"sacrifice":       c.onSacrifice,
		"desecrate":       c.onDesecrate,
		"merchant_browse": c.onMerchantBrowse,
		"merchant_buy":    c.onMerchantBuy,
		"puzzle_solve":    c.onPuzzleSolve,
		"rest_torch":      c.onRestTorch,
		"rest_sleep":      c.onRestSleep,
		"npc_help":        c.onNpcHelp,
		"npc_betray":      c.onNpcBetray,
		"combat_slash":    c.onCombatSlash,
		"combat_fireball": c.onCombatFireball,
		"combat_defend":   c.onCombatDefend,
		"combat_potion":   c.onCombatPotion,
		"combat_flee":     c.onCombatFlee,
		"rescue":          c.onRescue,
		"ignore_fallen":   c.onIgnoreFallen,
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
	gen := delvesvc.GenerateRoom(session, lang)
	return &gen
}

func (c *Cog) renderRoom(session *model.DelveSession, room *delvesvc.Room, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := delvesvc.BuildRoomEmbed(session, room, lang, c.svc)
	comps := delvesvc.BuildRoomComponents(room)
	return embed, comps
}

func (c *Cog) renderRoomWithFallen(session *model.DelveSession, room *delvesvc.Room, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed, comps := c.renderRoom(session, room, lang)
	fallen, err := c.store.GetFallenPlayersOnFloor(session.GuildID, int64(session.Floor), 2)
	if err == nil && len(fallen) > 0 {
		delvesvc.MaybeAddRescueOverlay(room, fallen, session.UserID, session.GuildID, c.store)
		embed.Description = room.Description
		comps = delvesvc.BuildRoomComponents(room)
		_ = fallen
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

func (c *Cog) startDelve(b *interaction.Bot, userID, guildID, channelID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent, *model.DelveSession, error) {
	s, err := c.svc.StartSession(userID, guildID, channelID)
	if err != nil {
		return nil, nil, nil, err
	}
	c.saveSession(s)
	room := delvesvc.GenerateRoom(s, "en")
	embed, comps := c.renderRoom(s, &room, "en")
	return embed, comps, s, nil
}

func (c *Cog) canStartDelve(userID int64) (bool, string) {
	raw := c.loadSessionRaw(userID)
	if raw != nil {
		if raw.Status == "active" {
			return false, "You already have an active delve! Use the flee button to escape."
		}
		if raw.Status == "fallen" {
			if raw.AutoRescued && raw.DiedAt != nil {
				elapsed := time.Since(*raw.DiedAt)
				if elapsed >= 5*time.Minute {
					c.svc.AddFlag(raw, "fell_in_battle")
					c.svc.EndSession(raw, "automatically rescued")
					c.store.ClearCooldown(userID, "delve_death")
					c.deleteSession(userID)
					return false, "🆘 **You've been rescued!**\nYour loyal **" + raw.AutoRescuePet + "** dragged you from the darkness! You're free to delve again."
				}
				remaining := 5*time.Minute - elapsed
				return false, fmt.Sprintf("⏳ Your **%s** is working to free you. Check back in %d minute(s).", raw.AutoRescuePet, int(remaining.Minutes())+1)
			}
			return false, fmt.Sprintf("💀 You fell on floor %d. Your pets couldn't reach you. Another adventurer may find you there, or try again tomorrow.", raw.Floor)
		}
	}
	midnight := time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour)
	ok, _ := c.store.CheckCooldown(userID, "delve_death", time.Until(midnight))
	if !ok {
		return false, "You cannot delve again today. Wait for rescue or until tomorrow."
	}
	return true, ""
}

func (c *Cog) onSlashDelve(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	allowed, msg := c.canStartDelve(userID)
	if !allowed {
		if strings.HasPrefix(msg, "🆘") || strings.HasPrefix(msg, "⏳") {
			_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{Title: "🔔 Rescue Status", Description: msg, Color: 0x2ecc71},
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
		c.errorMsg(b, i, fmt.Sprintf("Failed to start delve: %v", err))
		return
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixDelve(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	allowed, msg := c.canStartDelve(userID)
	if !allowed {
		_, _ = s.ChannelMessageSend(m.ChannelID, msg)
		return
	}
	embed, comps, _, err := c.startDelve(b, userID, interaction.ToInt64(m.GuildID), interaction.ToInt64(m.ChannelID))
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Failed to start delve: %v", err))
		return
	}
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) onSlashJourney(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(i.Member.User.ID)
	pages, err := delvesvc.BuildChronicle(userID, c.svc, "en")
	if err != nil {
		c.errorMsg(b, i, fmt.Sprintf("Failed to load chronicle: %v", err))
		return
	}
	if len(pages) == 0 {
		pages = append(pages, &discordgo.MessageEmbed{
			Title:       "📖 Personal Chronicle",
			Description: "No chronicle entries yet. Begin your journey with `/delve`!",
			Color:       0x9b59b6,
		})
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, pages[0], nil))
}

func (c *Cog) onPrefixJourney(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	pages, err := delvesvc.BuildChronicle(userID, c.svc, "en")
	if err != nil {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Failed to load chronicle.")
		return
	}
	if len(pages) == 0 {
		pages = append(pages, &discordgo.MessageEmbed{
			Title:       "📖 Personal Chronicle",
			Description: "No chronicle entries yet. Begin your journey with `!delve`!",
			Color:       0x9b59b6,
		})
	}
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{pages[0]},
	})
}
