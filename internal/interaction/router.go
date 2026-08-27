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

const (
	rateLimitCooldown = 500 * time.Millisecond
	// directRespondWindow is how long a handler may run before the router
	// acknowledges the interaction with a deferred response. Handlers that
	// reply within the window keep Discord's 3s window untouched and their
	// answer is sent directly (one HTTP call, no "thinking" flash); slower
	// handlers fall back to the deferred acknowledgment and the DeferringSession
	// rewrites their reply into a follow-up.
	directRespondWindow = 2 * time.Second
)

// ownerGatedDomains lists the personal, single-user menus whose interactive
// components may only be operated by the user who created the embed. Their
// custom_ids carry the owner id as the final element (see components.EncodeOwner).
var ownerGatedDomains = map[string]struct{}{
	"character":    {},
	"farm":         {},
	"pets":         {},
	"sanctuary":    {},
	"house":        {},
	"economy":      {},
	"bank":         {},
	"casino":       {},
	"mine":         {},
	"fish":         {},
	"arch":         {},
	"hunt":         {},
	"forge":        {},
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
	"crafting":     {},
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

	directRespondWindow time.Duration
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

		directRespondWindow: directRespondWindow,
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
	"betting::create":            {},
	"betting::place":             {},
	"betting::close":             {},
	"betting::odds":              {},
	"betting::freeze":            {},
	"admin::airdrop":             {},
	"admin::airdrop_item":        {},
	"admin::givecrowns":          {},
	"admin::setlang":             {},
	"duel::challenge":            {},
	"onboarding::advanced":       {},
	"veil::whisper_answer":       {},
	"blackjack::challenge":       {},
	"roulette::new":              {},
	"loan::borrow":               {},
	"loan::repay":                {},
	"community::contribute":      {},
	"pets::rename_btn":           {},
	"sanctuary::pet_search_open": {},
	"bank::deposit":              {},
	"bank::withdraw":             {},
	"inventory::pick":            {},
	"casino::slots":              {},
	"casino::coinflip_choice":    {},
	"casino::mega_slots":         {},
	"market::action":             {},
	"market::sellitem":           {},
	"economy::give":              {},
	"lotto::buy":                 {},
	"delve::puzzle_solve":        {},
	"arch::dustpick":             {},
}

// isModalOpener reports whether a component answers with a modal instead of a
// message update.
func isModalOpener(domain, action string) bool {
	_, ok := modalOpenerActions[domain+"::"+action]
	return ok
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
	case discordgo.InteractionApplicationCommandAutocomplete:
		return false // autocomplete must respond directly within 3s
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

// dispatchInteraction runs the handler and only sends the deferred
// acknowledgment when the handler does not reply within directRespondWindow.
// A handler that replies in time sends its answer directly (one HTTP call, no
// "thinking" flash); a slower handler gets the deferred ack and its reply is
// rewritten into a follow-up. The DeferringSession's isDeferred check is
// mutex-guarded, so a reply racing the acknowledgment is always classified
// consistently: whichever of the two calls happens first wins.
func (r *Router) dispatchInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, h func()) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer r.recoverPanic(i)
		h()
	}()
	select {
	case <-done:
	case <-time.After(r.directRespondWindow):
		r.deferInteraction(s, i)
		<-done
	}
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
	case discordgo.InteractionApplicationCommand, discordgo.InteractionApplicationCommandAutocomplete:
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
		if h, ok := r.slash[data.Name]; ok {
			// Autocomplete must respond synchronously without defer window.
			if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
				h(r.bot, i)
				return
			}
			r.dispatchInteraction(s, i, func() { h(r.bot, i) })
		} else {
			log.Warn("no handler for slash command", "cmd", data.Name)
			r.deferInteraction(s, i)
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
		key := domain + "::" + action
		if h, ok := r.component[key]; ok {
			r.dispatchInteraction(s, i, func() { h(r.bot, i) })
		} else {
			log.Warn("no handler for component", "custom_id", cid, "key", key)
			r.deferInteraction(s, i)
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
		key := domain + "::" + action
		if h, ok := r.modal[key]; ok {
			r.dispatchInteraction(s, i, func() { h(r.bot, i) })
		} else {
			log.Warn("no handler for modal", "custom_id", cid, "key", key)
			r.deferInteraction(s, i)
		}
	}

	total := time.Since(start)
	fields := []any{
		"user", uid,
		"guild", gid,
		"duration", total.String(),
	}
	// Attribute the elapsed time to Discord API round-trips vs in-process handler
	// work, so slow interactions can be diagnosed without a profiler.
	if ds, ok := r.bot.Session.(*DeferringSession); ok && i.Interaction != nil {
		discord := ds.TakeDiscordTime(i.Interaction.ID)
		compute := total - discord
		if compute < 0 {
			compute = 0
		}
		fields = append(fields, "discord_ms", millis(discord), "compute_ms", millis(compute))
	}
	log.Info("interaction completed", fields...)
}

// millis renders a duration as fractional milliseconds for structured logs.
func millis(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000
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
