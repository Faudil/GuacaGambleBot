package interaction

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/store"
)

// Bot bundles the discordgo session, database and shared config for handlers.
type Bot struct {
	Session *discordgo.Session
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
	slash      map[string]SlashHandler
	slashDefs  []*discordgo.ApplicationCommand
	prefix     map[string]PrefixHandler
	component  map[string]ComponentHandler
	modal      map[string]ModalHandler
}

// Session returns the underlying discordgo session.
func (r *Router) Session() *discordgo.Session { return r.bot.Session }

func NewRouter(bot *Bot, st *store.Store) *Router {
	return &Router{
		bot:       bot,
		store:     st,
		slash:     map[string]SlashHandler{},
		slashDefs: []*discordgo.ApplicationCommand{},
		prefix:    map[string]PrefixHandler{},
		component: map[string]ComponentHandler{},
		modal:     map[string]ModalHandler{},
	}
}

// Slash registers a slash command and its handler.
func (r *Router) Slash(name, description string, h SlashHandler) {
	r.slash[name] = h
	r.slashDefs = append(r.slashDefs, &discordgo.ApplicationCommand{Name: name, Description: description})
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
	r.bot.Session.AddHandler(r.onInteraction)
	r.bot.Session.AddHandler(r.onMessage)
}

// DispatchInteraction routes an interaction to the registered handler. It is the
// entry point used by the gateway and is exported so tests can drive handlers
// without a live Discord connection.
func (r *Router) DispatchInteraction(i *discordgo.InteractionCreate) {
	r.onInteraction(r.bot.Session, i)
}

func (r *Router) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		data := i.ApplicationCommandData()
		// The setup command is always allowed so a disabled guild can be
		// re-enabled from within Discord.
		if r.store != nil && data.Name != "setup" && !r.store.IsEnabled(ToInt64(i.GuildID)) {
			return
		}
		if h, ok := r.slash[data.Name]; ok {
			h(r.bot, i)
		}
	case discordgo.InteractionMessageComponent:
		cid := i.MessageComponentData().CustomID
		domain, action, _ := components.Decode(cid)
		if r.store != nil && domain != "onboarding" && !r.store.IsEnabled(ToInt64(i.GuildID)) {
			return
		}
		if h, ok := r.component[domain+"::"+action]; ok {
			h(r.bot, i)
		}
	case discordgo.InteractionModalSubmit:
		cid := i.ModalSubmitData().CustomID
		domain, action, _ := components.Decode(cid)
		if r.store != nil && domain != "onboarding" && !r.store.IsEnabled(ToInt64(i.GuildID)) {
			return
		}
		if h, ok := r.modal[domain+"::"+action]; ok {
			h(r.bot, i)
		}
	}
}

func (r *Router) onMessage(s *discordgo.Session, m *discordgo.Message) {
	if m.Author != nil && m.Author.Bot {
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
	// The setup command is always allowed so a disabled guild can be
	// re-enabled from within Discord.
	if r.store != nil && parts[0] != "setup" && !r.store.IsEnabled(gid) {
		return
	}
	if h, ok := r.prefix[parts[0]]; ok {
		h(r.bot, s, m)
	}
}

// RegisterCommands publishes all slash commands to Discord. Pass a guild ID to
// register them only there (instant), or "" for global registration.
func (r *Router) RegisterCommands(guildID string) error {
	appID := r.bot.Session.State.User.ID
	for _, cmd := range r.slashDefs {
		if _, err := r.bot.Session.ApplicationCommandCreate(appID, guildID, cmd); err != nil {
			return err
		}
	}
	return nil
}
