package interaction

import (
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/logger"
)

// Session is the subset of *discordgo.Session that cogs use through
// interaction.Bot. The router swaps the concrete session for a
// DeferringSession, so every handler's response is transparently routed
// through Discord's deferred-response pipeline without modifying any cog.
type Session interface {
	InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error
	InteractionResponseEdit(i *discordgo.Interaction, r *discordgo.WebhookEdit, opts ...discordgo.RequestOption) (*discordgo.Message, error)
	FollowupMessageCreate(i *discordgo.Interaction, wait bool, r *discordgo.WebhookParams, opts ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageSend(channelID, content string, opts ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageSendEmbed(channelID string, embed *discordgo.MessageEmbed, opts ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, opts ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageEditComplex(m *discordgo.MessageEdit, opts ...discordgo.RequestOption) (*discordgo.Message, error)
	UserChannelCreate(userID string, opts ...discordgo.RequestOption) (*discordgo.Channel, error)
	GuildMember(guildID, userID string, opts ...discordgo.RequestOption) (*discordgo.Member, error)
	GuildMembers(guildID string, after string, limit int, opts ...discordgo.RequestOption) ([]*discordgo.Member, error)
}

// DeferringSession acknowledges interactions with a deferred response before
// the handler runs (so the first response never races Discord's 3s window) and
// translates the handler's eventual response into the corresponding
// post-deferral call:
//
//	UpdateMessage            -> InteractionResponseEdit
//	ChannelMessageWithSource -> FollowupMessageCreate (flags preserved)
//	Deferred*                -> no-op (the router already deferred)
//	Modal                    -> logged + dropped (modal openers must never be
//	                            deferred; a drop here means the opener is missing
//	                            from the router's modalOpenerActions registry)
//
// All other methods are forwarded unchanged to the underlying session.
type DeferringSession struct {
	Session

	mu       sync.Mutex
	deferred map[string]time.Time
}

// NewDeferringSession wraps base with the deferred-response translation layer.
func NewDeferringSession(base Session) *DeferringSession {
	return &DeferringSession{
		Session:  base,
		deferred: make(map[string]time.Time),
	}
}

// deferInteraction records that the router acknowledged this interaction with
// a deferred response, so the handler's reply must be translated instead of
// sent directly.
func (d *DeferringSession) deferInteraction(i *discordgo.Interaction) {
	if i == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.deferred) > 20000 {
		// Interactions complete within minutes; drop stale markers so the map
		// never grows without bound.
		cutoff := time.Now().Add(-10 * time.Minute)
		keep := make(map[string]time.Time, len(d.deferred)/2)
		for id, t := range d.deferred {
			if t.After(cutoff) {
				keep[id] = t
			}
		}
		d.deferred = keep
	}
	d.deferred[i.ID] = time.Now()
}

func (d *DeferringSession) isDeferred(i *discordgo.Interaction) bool {
	if i == nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	_, ok := d.deferred[i.ID]
	return ok
}

func (d *DeferringSession) InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, opts ...discordgo.RequestOption) error {
	if !d.isDeferred(i) {
		return d.Session.InteractionRespond(i, r, opts...)
	}
	switch r.Type {
	case discordgo.InteractionResponseDeferredMessageUpdate,
		discordgo.InteractionResponseDeferredChannelMessageWithSource:
		// Already acknowledged by the router.
		return nil
	case discordgo.InteractionResponseUpdateMessage:
		return d.translateEdit(i, r.Data, opts...)
	case discordgo.InteractionResponseChannelMessageWithSource:
		return d.translateFollowup(i, r.Data, opts...)
	case discordgo.InteractionResponseModal:
		logger.Log().Error("modal response attempted on a deferred interaction; add the opener to router.modalOpenerActions",
			"interaction_id", i.ID)
		return nil
	default:
		logger.Log().Warn("unsupported response type on a deferred interaction, sending as-is",
			"type", r.Type,
			"interaction_id", i.ID)
		return d.Session.InteractionRespond(i, r, opts...)
	}
}

// translateEdit rewrites an UpdateMessage response into an edit of the
// original message, which is the only legal reply after a deferred update.
func (d *DeferringSession) translateEdit(i *discordgo.Interaction, data *discordgo.InteractionResponseData, opts ...discordgo.RequestOption) error {
	edit := &discordgo.WebhookEdit{}
	if data.Content != "" {
		c := data.Content
		edit.Content = &c
	}
	if data.Embeds != nil {
		edit.Embeds = &data.Embeds
	}
	if data.Components != nil {
		edit.Components = &data.Components
	}
	if data.AllowedMentions != nil {
		edit.AllowedMentions = data.AllowedMentions
	}
	if len(data.Files) > 0 {
		edit.Files = data.Files
	}
	if data.Attachments != nil {
		edit.Attachments = data.Attachments
	}
	if _, err := d.Session.InteractionResponseEdit(i, edit, opts...); err != nil {
		logger.Log().Warn("deferred update translation failed", "interaction_id", i.ID, "error", err)
		return err
	}
	return nil
}

// translateFollowup rewrites a ChannelMessageWithSource response into a
// follow-up message, which is the only legal reply after a deferred channel
// message. Message flags (e.g. ephemeral) are preserved.
func (d *DeferringSession) translateFollowup(i *discordgo.Interaction, data *discordgo.InteractionResponseData, opts ...discordgo.RequestOption) error {
	params := &discordgo.WebhookParams{
		Content:         data.Content,
		TTS:             data.TTS,
		Embeds:          data.Embeds,
		Components:      data.Components,
		AllowedMentions: data.AllowedMentions,
		Flags:           data.Flags,
		Files:           data.Files,
	}
	if data.Attachments != nil {
		params.Attachments = *data.Attachments
	}
	if _, err := d.Session.FollowupMessageCreate(i, false, params, opts...); err != nil {
		logger.Log().Warn("deferred follow-up translation failed", "interaction_id", i.ID, "error", err)
		return err
	}
	return nil
}
