// Package journal implements the /journal cog: a player-driven goal board
// made of progression paths (see internal/service/journal for the engine).
package journal

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/items"
	jsvc "guacagamblebot/internal/service/journal"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *jsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: jsvc.New(s)}
	r.Slash("journal", "Your personal journal: goals, paths and ranks.", c.onSlash)
	r.Prefix("journal", c.onPrefix)
	r.Prefix("j", c.onPrefix)
	r.Component("journal", "show", c.onShow)
	r.Component("journal", "track", c.onTrack)
	r.Component("journal", "leaderboard", c.onLeaderboard)
	r.Component("journal", "back", c.onBack)
}

func (c *Cog) onSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	guildID := interaction.ToInt64(i.GuildID)
	embed, comps := c.buildEmbed(b.Session, lang, guildID, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	guildID := interaction.ToInt64(m.GuildID)
	embed, comps := c.buildEmbed(s, lang, guildID, userID)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) onShow(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	guildID := interaction.ToInt64(i.GuildID)
	embed, comps := c.buildEmbed(b.Session, lang, guildID, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onTrack(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	guildID := interaction.ToInt64(i.GuildID)
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) > 0 {
		_ = c.svc.Track(userID, rest[0])
	}
	embed, comps := c.buildEmbed(b.Session, lang, guildID, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onLeaderboard(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.buildLeaderboard(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onBack(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	guildID := interaction.ToInt64(i.GuildID)
	embed, comps := c.buildEmbed(b.Session, lang, guildID, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

// announceMastery posts the Mastery legend reveal to the guild's announcement
// channel. Fires on every unlock (each player who masters all paths).
func (c *Cog) announceMastery(sess interaction.Session, lang string, guildID, userID int64) {
	if sess == nil || guildID == 0 {
		return
	}
	go func() {
		ss, _ := c.store.GetServerSetting(guildID)
		if ss == nil || ss.AnnouncementChannelID == 0 {
			return
		}
		embed := components.Embed(
			i18n.T("journal.mastery.announce_title", lang, map[string]any{"user": interaction.Mention(userID)}),
			i18n.T("journal.mastery.announce_desc", lang),
			0xf1c40f,
		)
		_, _ = sess.ChannelMessageSendEmbed(strconv.FormatInt(ss.AnnouncementChannelID, 10), embed)
	}()
}

func progressBar(current, target int) string {
	const totalBlocks = 10
	if target <= 0 {
		return ""
	}
	filled := current * totalBlocks / target
	if filled > totalBlocks {
		filled = totalBlocks
	}
	empty := totalBlocks - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

func rewardString(r jsvc.Reward, lang string) string {
	var parts []string
	if r.Money > 0 {
		parts = append(parts, i18n.T("journal.reward.money", lang, map[string]any{"amount": r.Money}))
	}
	if r.Crowns > 0 {
		parts = append(parts, i18n.T("journal.reward.crowns", lang, map[string]any{"amount": r.Crowns}))
	}
	for _, id := range r.ItemIDs {
		parts = append(parts, items.LocalizedName(id, lang))
	}
	return strings.Join(parts, ", ")
}

func rankName(rank int, lang string) string {
	return i18n.T("journal.ranks."+strconv.Itoa(rank), lang)
}

func (c *Cog) buildEmbed(sess interaction.Session, lang string, guildID, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	v, err := c.svc.View(userID)
	if err != nil {
		return components.Embed(i18n.T("journal.title", lang), i18n.T("journal.error", lang), 0xe74c3c), nil
	}

	// Fresh Mastery unlock: celebrate in the server announcement channel.
	if v.MasteryNew {
		c.announceMastery(sess, lang, guildID, userID)
	}

	var desc strings.Builder
	desc.WriteString(i18n.T("journal.desc", lang) + "\n")
	desc.WriteString("\n**" + i18n.T("journal.rumors_title", lang) + "**")

	// The secret Mastery legend, revealed only to those who earned it.
	if v.MasteryUnlocked {
		desc.WriteString("\n**" + i18n.T("journal.mastery.title", lang) + "**\n" +
			i18n.T("journal.mastery.desc", lang) + "\n")
	} else if jsvc.HighestRank(c.store, userID) >= 1 {
		// The locked door: a tease for players who earned their first rank.
		desc.WriteString("\n🔒 " + i18n.T("journal.hall_locked", lang) + "\n")
	}

	// Fresh completions banner.
	if len(v.Completions) > 0 {
		desc.WriteString("\n**" + i18n.T("journal.completed", lang) + "**\n")
		for _, comp := range v.Completions {
			line := comp.PathEmoji + " " + i18n.T(comp.StepTextKey, lang)
			if rs := rewardString(comp.Reward, lang); rs != "" {
				line += " — +" + rs
			}
			if comp.AllDone {
				line += " " + i18n.T("journal.path_complete", lang)
			}
			desc.WriteString("✅ " + line + "\n")
		}
	}

	// Tracked paths first, then the rest.
	var tracked, others []jsvc.PathView
	for _, p := range v.Paths {
		if p.Tracked {
			tracked = append(tracked, p)
		} else {
			others = append(others, p)
		}
	}

	writePath := func(p jsvc.PathView) {
		desc.WriteString("\n**" + p.Emoji + " " + i18n.T(p.TitleKey, lang) + "** — " +
			rankName(p.Rank, lang) + " (" + strconv.Itoa(p.Completed) + "/" + strconv.Itoa(p.Total) + ")\n")
		if p.AllDone {
			desc.WriteString("  ✅ " + i18n.T("journal.path_complete", lang) + "\n")
		} else if p.HasRumor {
			// The surfaced rumor: opener + objective + progress.
			desc.WriteString("  🗞️ " + i18n.T("journal.paths."+p.PathID+".rumor", lang) + "\n")
			for _, st := range p.Steps {
				line := i18n.T(st.TextKey, lang)
				switch {
				case st.Done:
					desc.WriteString("  ✅ " + line + "\n")
				case st.Discovered && st.Target > 0:
					bar := progressBar(st.Progress, st.Target)
					desc.WriteString("  🔸 " + line + " — " + strconv.Itoa(st.Progress) + "/" + strconv.Itoa(st.Target))
					if bar != "" {
						desc.WriteString(" `" + bar + "`")
					}
					desc.WriteString("\n")
				}
			}
		} else {
			// The current step has not surfaced yet: keep it a mystery.
			desc.WriteString("  🔒 " + i18n.T("journal.mystery", lang) + "\n")
			for _, st := range p.Steps {
				if st.Done {
					desc.WriteString("  ✅ " + i18n.T(st.TextKey, lang) + "\n")
				}
			}
		}
	}

	for _, p := range tracked {
		writePath(p)
	}
	for _, p := range others {
		writePath(p)
	}

	// Action rows: track buttons for every path (5 max per row), then nav.
	var comps []discordgo.MessageComponent
	var btns []discordgo.MessageComponent
	for _, p := range v.Paths {
		label := "📌"
		if p.Tracked {
			label = "📍"
		}
		btns = append(btns, components.Button(label+" "+i18n.T(p.TitleKey, lang),
			components.EncodeOwner(userID, "journal", "track", p.PathID), discordgo.SecondaryButton))
	}
	for len(btns) > 0 {
		n := 5
		if len(btns) < n {
			n = len(btns)
		}
		comps = append(comps, components.ActionRow(btns[:n]...))
		btns = btns[n:]
	}
	comps = append(comps, components.ActionRow(
		components.Button("🏆 "+i18n.T("journal.btn_leaderboard", lang), components.EncodeOwner(userID, "journal", "leaderboard"), discordgo.PrimaryButton),
		components.Button("🔄", components.EncodeOwner(userID, "journal", "show"), discordgo.SecondaryButton),
	))

	return components.Embed(i18n.T("journal.title", lang)+" — <@"+strconv.FormatInt(userID, 10)+">", desc.String(), 0x2ecc71), comps
}

func (c *Cog) buildLeaderboard(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	var desc strings.Builder
	desc.WriteString(i18n.T("journal.lb_desc", lang) + "\n")

	for _, p := range jsvc.GetPaths() {
		entries, err := c.svc.Leaderboard(p.ID, 5)
		if err != nil || len(entries) == 0 {
			continue
		}
		desc.WriteString("\n**" + p.Emoji + " " + i18n.T(p.TitleKey, lang) + "**\n")
		for i, e := range entries {
			desc.WriteString(i18n.T("journal.lb_entry", lang, map[string]any{
				"rank": i + 1,
				"user": interaction.Mention(e.UserID),
				"name": rankName(jsvc.RankFor(e.StepIndex, len(p.Steps)), lang),
				"step": e.StepIndex,
			}) + "\n")
		}
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("◀ "+i18n.T("journal.btn_back", lang), components.EncodeOwner(userID, "journal", "back"), discordgo.SecondaryButton),
		),
	}
	return components.Embed(i18n.T("journal.title", lang), desc.String(), 0xf1c40f), comps
}
