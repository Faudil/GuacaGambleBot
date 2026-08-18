package interaction

import (
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/logger"
	"guacagamblebot/internal/store"
)

const rateLimitCooldown = 500 * time.Millisecond

// ownerGatedDomains lists the personal, single-user menus whose interactive
// components may only be operated by the user who created the embed. Their
// custom_ids carry the owner id as the final element (see components.EncodeOwner).
var ownerGatedDomains = map[string]struct{}{
	"character":    {},
	"farm":         {},
	"pets":         {},
	"sanctuary":    {},
	"housing":      {},
	"economy":      {},
	"bank":         {},
	"casino":       {},
	"mine":         {},
	"fish":         {},
	"arch":         {},
	"hunt":         {},
	"expedition":   {},
	"jobs":         {},
	"skills":       {},
	"quest":        {},
	"achievements": {},
	"journal":      {},
	"loan":         {},
	"start":        {},
	"npc":          {},
	"boss":         {},
	"use":          {},
	"inventory":    {},
}

// ownerGatedExemptions lists (domain, action) pairs inside owner-gated domains
// that are intentionally shared and must remain open to everyone.
var ownerGatedExemptions = map[string]struct{}{
	"pets::battle_accept":  {},
	"pets::battle_decline": {},
}

func isOwnerGated(domain, action string) bool {
	if _, ok := ownerGatedDomains[domain]; !ok {
		return false
	}
	if _, ok := ownerGatedExemptions[domain+"::"+action]; ok {
		return false
	}
	return true
}

// Bot bundles the discordgo session, database and shared config for handlers.
// Session is wrapped in a DeferringSession by NewRouter so handlers never have
// to manage Discord's 3s interaction window themselves.
type Bot struct {
	Session Session
	DB      *gorm.DB
	Prefix  string
}

// SlashHandler handles an application (slash) command interaction.
type SlashHandler func(b *Bot, i *discordgo.InteractionCreate)

// PrefixHandler handles a classic prefixed message command.
type PrefixHandler func(b *Bot, s *discordgo.Session, m *discordgo.Message)

// ComponentHandler handles a button/select interaction.
type ComponentHandler func(b *Bot, i *discordgo.InteractionCreate)

// ModalHandler handles a modal submit interaction.
type ModalHandler func(b *Bot, i *discordgo.InteractionCreate)

// Router dispatches incoming interactions and prefix messages to handlers,
// giving every cog both a slash command and a `!prefix` entry point that open
// the same embed interface.
type Router struct {
	bot        *Bot
	store      *store.Store
	rawSession *discordgo.Session
	slash      map[string]SlashHandler
	slashDefs  []*discordgo.ApplicationCommand
	prefix     map[string]PrefixHandler
	component  map[string]ComponentHandler
	modal      map[string]ModalHandler

	rateLimitMu    sync.Mutex
	rateLimitTimes map[string]time.Time
}

// Session returns the underlying discordgo session.
func (r *Router) Session() *discordgo.Session { return r.rawSession }

func NewRouter(bot *Bot, st *store.Store) *Router {
	raw, _ := bot.Session.(*discordgo.Session)
	if raw != nil {
		// Swap in the deferred-response translation layer: the router
		// acknowledges every interaction immediately and the handler's reply is
		// rewritten into an edit/follow-up, so no cog can ever miss Discord's
		// 3s response window.
		bot.Session = NewDeferringSession(raw)
	}
	return &Router{
		bot:            bot,
		store:          st,
		rawSession:     raw,
		slash:          map[string]SlashHandler{},
		slashDefs:      []*discordgo.ApplicationCommand{},
		prefix:         map[string]PrefixHandler{},
		component:      map[string]ComponentHandler{},
		modal:          map[string]ModalHandler{},
		rateLimitTimes: map[string]time.Time{},
	}
}

// Slash registers a slash command and its handler.
func (r *Router) Slash(name, description string, h SlashHandler) {
	r.slash[name] = h
	r.slashDefs = append(r.slashDefs, &discordgo.ApplicationCommand{Name: name, Description: description})
}

// SlashWithOptions registers a slash command with Discord option arguments
// (e.g. user picks, number inputs) and its handler.
func (r *Router) SlashWithOptions(name, description string, options []*discordgo.ApplicationCommandOption, h SlashHandler) {
	r.slash[name] = h
	r.slashDefs = append(r.slashDefs, &discordgo.ApplicationCommand{Name: name, Description: description, Options: options})
}

// Prefix registers a `!name` message handler.
func (r *Router) Prefix(name string, h PrefixHandler) {
	r.prefix[name] = h
}

// Component registers a button/select handler keyed by (domain, action).
func (r *Router) Component(domain, action string, h ComponentHandler) {
	r.component[domain+"::"+action] = h
}

// Modal registers a modal-submit handler keyed by (domain, action).
func (r *Router) Modal(domain, action string, h ModalHandler) {
	r.modal[domain+"::"+action] = h
}

// Register wires the global discordgo event handlers.
func (r *Router) Register() {
	r.rawSession.AddHandler(r.onInteraction)
	r.rawSession.AddHandler(r.onMessage)
}

// DispatchInteraction routes an interaction to the registered handler. It is the
// entry point used by the gateway and is exported so tests can drive handlers
// without a live Discord connection.
func (r *Router) DispatchInteraction(i *discordgo.InteractionCreate) {
	r.onInteraction(r.rawSession, i)
}

// modalOpenerActions lists every component (domain, action) that answers with a
// modal instead of a message update. These must NOT be deferred: the modal
// response is the first and only reply Discord accepts, so the router lets them
// respond directly. Keep this in sync with every InteractionResponseModal site
// in the cogs; a missed opener is caught loudly in the logs (the DeferringSession
// drops the modal reply with an error).
var modalOpenerActions = map[string]struct{}{
	"betting::create":         {},
	"betting::place":          {},
	"betting::close":          {},
	"betting::odds":           {},
	"betting::freeze":         {},
	"admin::airdrop":          {},
	"admin::airdrop_item":     {},
	"admin::givecrowns":       {},
	"admin::setlang":          {},
	"duel::challenge":         {},
	"onboarding::advanced":    {},
	"veil::whisper_answer":    {},
	"blackjack::challenge":    {},
	"roulette::new":           {},
	"loan::borrow":            {},
	"loan::repay":             {},
	"community::contribute":   {},
	"pets::rename_btn":        {},
	"bank::deposit":           {},
	"bank::withdraw":          {},
	"inventory::pick":         {},
	"casino::slots":           {},
	"casino::coinflip_choice": {},
	"casino::mega_slots":      {},
	"market::action":          {},
	"economy::give":           {},
	"lotto::buy":              {},
	"delve::puzzle_solve":     {},
}

// modalOpenerPrefixes covers dynamically registered modal openers, e.g. one
// gift button per NPC ("npc::gift_<id>").
var modalOpenerPrefixes = []string{
	"npc::gift_",
}

func isModalOpener(domain, action string) bool {
	if _, ok := modalOpenerActions[domain+"::"+action]; ok {
		return true
	}
	for _, p := range modalOpenerPrefixes {
		if strings.HasPrefix(domain+"::"+action, p) {
			return true
		}
	}
	return false
}

// deferInteraction acknowledges the interaction with a deferred response so the
// handler has 15 minutes instead of 3 seconds to produce its real reply. The
// handler's response is then translated into an edit/follow-up by the
// DeferringSession. Returns whether the deferred acknowledgment was sent.
func (r *Router) deferInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	respType := discordgo.InteractionResponseDeferredChannelMessageWithSource
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		// Deferred channel message: slash replies arrive as follow-ups.
	case discordgo.InteractionMessageComponent:
		domain, action, _ := components.Decode(i.MessageComponentData().CustomID)
		if isModalOpener(domain, action) {
			return false // modal openers must respond directly with the modal
		}
		respType = discordgo.InteractionResponseDeferredMessageUpdate
	case discordgo.InteractionModalSubmit:
		respType = discordgo.InteractionResponseDeferredMessageUpdate
	default:
		return false
	}
	if s == nil || s.Client == nil {
		return false // no HTTP client (unit tests); handlers respond directly
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: respType}); err != nil {
		logger.Log().Warn("failed to defer interaction; falling back to direct response",
			"error", err,
			"user", UserID(i),
			"guild", i.GuildID,
		)
		return false
	}
	if ds, ok := r.bot.Session.(*DeferringSession); ok {
		ds.deferInteraction(i.Interaction)
	}
	return true
}

func (r *Router) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer r.recoverPanic(i)

	log := logger.Log()
	uid := UserID(i)
	gid := i.GuildID
	start := time.Now()

	if !r.checkRateLimit(uid) {
		log.Warn("rate limited user", "user", uid, "guild", gid)
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⏳ Please slow down! You're sending commands too fast.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		data := i.ApplicationCommandData()
		if r.store != nil && data.Name != "setup" && !r.store.IsEnabled(ToInt64(gid)) {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ This bot is disabled on this server. Use `/setup` to enable it.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		log.Info("slash command",
			"cmd", data.Name,
			"user", uid,
			"guild", gid,
		)
		r.deferInteraction(s, i)
		if h, ok := r.slash[data.Name]; ok {
			h(r.bot, i)
		} else {
			log.Warn("no handler for slash command", "cmd", data.Name)
		}

	case discordgo.InteractionMessageComponent:
		cid := i.MessageComponentData().CustomID
		domain, action, _ := components.Decode(cid)
		if r.store != nil && domain != "onboarding" && !r.store.IsEnabled(ToInt64(gid)) {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ This bot is disabled on this server. Use `/setup` to enable it.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		if isOwnerGated(domain, action) {
			ownerID, ok := components.OwnerID(cid)
			if !ok || ownerID != ToInt64(uid) {
				lang := "en"
				if r.store != nil {
					lang = r.store.GetLanguage(ToInt64(gid))
				}
				NotYourMenu(r.bot, i, lang, ownerID)
				return
			}
		}
		log.Info("component interaction",
			"domain", domain,
			"action", action,
			"user", uid,
			"guild", gid,
		)
		r.deferInteraction(s, i)
		key := domain + "::" + action
		if h, ok := r.component[key]; ok {
			h(r.bot, i)
		} else {
			log.Warn("no handler for component", "custom_id", cid, "key", key)
		}

	case discordgo.InteractionModalSubmit:
		cid := i.ModalSubmitData().CustomID
		domain, action, _ := components.Decode(cid)
		if r.store != nil && domain != "onboarding" && !r.store.IsEnabled(ToInt64(gid)) {
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ This bot is disabled on this server. Use `/setup` to enable it.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		log.Info("modal submit",
			"domain", domain,
			"action", action,
			"user", uid,
			"guild", gid,
		)
		r.deferInteraction(s, i)
		key := domain + "::" + action
		if h, ok := r.modal[key]; ok {
			h(r.bot, i)
		} else {
			log.Warn("no handler for modal", "custom_id", cid, "key", key)
		}
	}

	log.Info("interaction completed",
		"user", uid,
		"guild", gid,
		"duration", time.Since(start).String(),
	)
}

func (r *Router) onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	defer r.recoverPanicMsg(m)

	log := logger.Log()
	start := time.Now()

	if m.Author != nil && m.Author.Bot {
		return
	}

	if !r.checkRateLimit(m.Author.ID) {
		log.Warn("rate limited user", "user", m.Author.ID, "guild", m.GuildID)
		return
	}
	gid := ToInt64(m.GuildID)
	prefix := r.bot.Prefix
	if r.store != nil {
		prefix = r.store.ServerPrefix(gid)
	}
	if !strings.HasPrefix(m.Content, prefix) {
		return
	}
	trimmed := strings.TrimPrefix(m.Content, prefix)
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return
	}
	if r.store != nil && parts[0] != "setup" && !r.store.IsEnabled(gid) {
		return
	}

	log.Info("prefix command",
		"cmd", parts[0],
		"user", m.Author.ID,
		"guild", fmt.Sprint(gid),
	)
	if h, ok := r.prefix[parts[0]]; ok {
		h(r.bot, s, m.Message)
	} else {
		log.Warn("no handler for prefix command", "cmd", parts[0])
	}

	log.Info("message completed",
		"user", m.Author.ID,
		"guild", fmt.Sprint(gid),
		"duration", time.Since(start).String(),
	)
}

func (r *Router) recoverPanic(i *discordgo.InteractionCreate) {
	if v := recover(); v != nil {
		logger.Log().Error("panic recovered in interaction handler",
			"panic", v,
			"stack", string(debug.Stack()),
			"user", UserID(i),
			"guild", i.GuildID,
		)
		// Never leave the user waiting on the 3s interaction window: send a
		// generic ephemeral error so they get feedback instead of a timeout.
		if r.bot != nil && r.bot.Session != nil && i.Interaction != nil {
			lang := "en"
			if r.store != nil {
				lang = r.store.GetLanguage(ToInt64(i.GuildID))
			}
			_ = r.bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: i18n.T("common.internal_error", lang),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		}
	}
}

func (r *Router) recoverPanicMsg(m *discordgo.MessageCreate) {
	if v := recover(); v != nil {
		uid := ""
		if m.Author != nil {
			uid = m.Author.ID
		}
		logger.Log().Error("panic recovered in message handler",
			"panic", v,
			"stack", string(debug.Stack()),
			"user", uid,
			"guild", m.GuildID,
		)
	}
}

// RegisterCommands publishes all slash commands to Discord. Pass a guild ID to
// register them only there (instant), or "" for global registration.
func (r *Router) Commands() []*discordgo.ApplicationCommand {
	return r.slashDefs
}

func (r *Router) checkRateLimit(userID string) bool {
	r.rateLimitMu.Lock()
	defer r.rateLimitMu.Unlock()

	last, ok := r.rateLimitTimes[userID]
	now := time.Now()
	if ok && now.Sub(last) < rateLimitCooldown {
		return false
	}
	r.rateLimitTimes[userID] = now

	if len(r.rateLimitTimes) > 10000 {
		cleanup := make(map[string]time.Time, len(r.rateLimitTimes)/2)
		for k, v := range r.rateLimitTimes {
			if now.Sub(v) < rateLimitCooldown*10 {
				cleanup[k] = v
			}
		}
		r.rateLimitTimes = cleanup
	}
	return true
}

// RegisterCommands publishes all slash commands to Discord and purges the
// opposite scope so commands never show up twice. BulkOverwrite only replaces
// commands in the scope it targets; without the purge, switching between
// global and guild-scoped registration leaves stale duplicates behind.
func (r *Router) RegisterCommands(guildID string) error {
	appID := r.rawSession.State.User.ID
	_, err := r.rawSession.ApplicationCommandBulkOverwrite(appID, guildID, r.slashDefs)
	if err != nil {
		return err
	}
	if guildID != "" {
		// Guild-scoped registration: drop stale global copies.
		_, purgeErr := r.rawSession.ApplicationCommandBulkOverwrite(appID, "", []*discordgo.ApplicationCommand{})
		if purgeErr != nil {
			logger.Log().Warn("could not purge global slash commands", "error", purgeErr)
		} else {
			logger.Log().Info("purged stale global slash commands")
		}
		return nil
	}
	// Global registration: drop stale guild-scoped copies in every guild.
	for _, g := range r.rawSession.State.Guilds {
		_, purgeErr := r.rawSession.ApplicationCommandBulkOverwrite(appID, g.ID, []*discordgo.ApplicationCommand{})
		if purgeErr != nil {
			logger.Log().Warn("could not purge guild slash commands", "error", purgeErr, "guild", g.ID)
		} else {
			logger.Log().Info("purged stale guild slash commands", "guild", g.ID)
		}
	}
	return nil
}
