package lore

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/interaction"
	loresvc "guacagamblebot/internal/service/lore"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *loresvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: loresvc.New(s, cfg)}
	r.Slash("lore", "Browse discovered lore fragments", c.onSlash)
	r.Prefix("lore", c.onPrefix)
	r.Component("lore", "category", c.onCategorySelect)
	r.Component("lore", "back", c.onBack)
}

func (c *Cog) onSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.showMenu(b, i, false)
}

func (c *Cog) onPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	embed, comps := c.buildOverview(userID, lang)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) showMenu(b *interaction.Bot, i *discordgo.InteractionCreate, edit bool) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.buildOverview(userID, lang)
	if edit {
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
	} else {
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
	}
}

var catMeta = map[loresvc.Category]struct {
	Name string
	Emoji string
	Color int
	Hint string
}{
	loresvc.CatAether:   {"Aether-Logs", "🟦", 0x3498db, "Found while mining deep strata"},
	loresvc.CatTide:     {"Tide-Scrolls", "🟩", 0x2ecc71, "Found while fishing"},
	loresvc.CatRoot:     {"Root-Whispers", "🟫", 0x8b4513, "Found while farming"},
	loresvc.CatField:    {"Field Observations", "🟧", 0xe67e22, "Found while hunting"},
	loresvc.CatRust:     {"Rust-Memories", "🟥", 0xe74c3c, "Found while excavating fossils"},
	loresvc.CatEcho:     {"Echo-Shards", "🟨", 0xf1c40f, "Found during pet expeditions"},
	loresvc.CatBonus:    {"Secret Fragments", "💠", 0x9b59b6, "Special discoveries"},
}

func (c *Cog) buildOverview(userID int64, lang string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	progress, totalD, totalT, err := c.svc.AllProgress(userID)
	if err != nil {
		return components.Embed("Error", "Could not load lore progress.", 0xff0000), nil
	}

	desc := fmt.Sprintf("**Codex of the Strata** — %d / %d fragments discovered\n\n", totalD, totalT)
	catOrder := []loresvc.Category{loresvc.CatAether, loresvc.CatTide, loresvc.CatRoot, loresvc.CatField, loresvc.CatRust, loresvc.CatEcho, loresvc.CatBonus}

	var selectOpts []discordgo.SelectMenuOption

	for _, cat := range catOrder {
		p := progress[cat]
		meta := catMeta[cat]

		bar := progressBar(p.D, p.T, 10)
		desc += fmt.Sprintf("%s **%s** — %s (%d/%d)\n", meta.Emoji, meta.Name, bar, p.D, p.T)

		selectOpts = append(selectOpts, discordgo.SelectMenuOption{
			Label:       meta.Name,
			Value:       string(cat),
			Description: fmt.Sprintf("%d/%d discovered — %s", p.D, p.T, meta.Hint),
			Emoji:       &discordgo.ComponentEmoji{Name: meta.Emoji},
		})
	}

	embed := components.Embed("📖 Lore Codex", desc, 0x2c3e50)
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("Use !lore to browse your collection | %d%% complete", percent(totalD, totalT)),
	}

	sel := discordgo.SelectMenu{
		CustomID:    components.Encode("lore", "category"),
		Placeholder: "Choose a category to explore...",
		Options:     selectOpts,
	}

	comps := []discordgo.MessageComponent{
		components.ActionRow(&sel),
	}

	return embed, comps
}

func (c *Cog) onCategorySelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	cat := loresvc.Category(i.MessageComponentData().Values[0])

	fragments := loresvc.AllInCategory(cat)
	discovered, _ := c.svc.GetDiscovered(userID) // need to expose this
	// Temporary: build from service
	progress, _, _, _ := c.svc.AllProgress(userID)
	p := progress[cat]
	meta := catMeta[cat]

	desc := fmt.Sprintf("%s **%s** — %d/%d\n\n", meta.Emoji, meta.Name, p.D, p.T)

	for _, f := range fragments {
		if discovered[f.ID] {
			desc += fmt.Sprintf("✅ **%s** — %s\n", f.Title(lang), f.Emoji)
		} else {
			desc += fmt.Sprintf("❓ **???** — *%s*\n", meta.Hint)
		}
	}

	embed := components.Embed("📖 Lore Codex — "+meta.Name, desc, meta.Color)
	backBtn := components.Button("← Back", components.Encode("lore", "back"), discordgo.SecondaryButton)

	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed,
			[]discordgo.MessageComponent{components.ActionRow(backBtn)}))
}

func (c *Cog) onBack(b *interaction.Bot, i *discordgo.InteractionCreate) {
	c.showMenu(b, i, true)
}

func progressBar(current, total, width int) string {
	if total == 0 {
		return strings.Repeat("░", width)
	}
	filled := current * width / total
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func percent(a, b int) int {
	if b == 0 {
		return 0
	}
	return a * 100 / b
}
