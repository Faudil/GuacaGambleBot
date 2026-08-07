package npcs

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/model"
	invsvc "guacagamblebot/internal/service/inventory"
	npcsvc "guacagamblebot/internal/service/npcs"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *npcsvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	def := universe.Get(cfg.Universe)
	if def == nil {
		def = universe.Get("hoakhaven")
	}
	inv := invsvc.New(s, cfg)
	c := &Cog{store: s, cfg: cfg, svc: npcsvc.New(s, cfg, def, inv)}
	r.Slash("npc", "Interagis avec les personnages du village.", c.onSlashMenu)
	r.Prefix("npc", c.onPrefixMenu)
	r.Prefix("np", c.onPrefixMenu)
	r.Component("npc", "select", c.onNPCSelect)
	r.Component("npc", "back", c.onBack)
	for _, n := range def.NPCs {
		id := n.ID
		r.Component("npc", id, c.makeNPCSelect(id))
		r.Component("npc", "chat_"+id, c.onChat(id))
		r.Component("npc", "gift_"+id, c.onGiftRequest(id))
		r.Component("npc", "bio_"+id, c.onBio(id))
		r.Component("npc", "advice_"+id, c.onAdvice(id))
		r.Component("npc", "rankup_"+id, c.onRankUp(id))
		r.Component("npc", "shop_"+id, c.onShop(id))
		for _, item := range n.ShopItems {
			r.Component("npc", "buy_"+id+"_"+item.ItemID, c.onShopBuy(id, item.ItemID))
		}
	}
	r.Modal("npc", "gift_submit", c.onGiftSubmit)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(lang, b, i)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	embed := components.Embed(i18n.T("npcs.list_title", lang), "", 0x3498db)
	allNPCs := c.svc.GetAllNPCMeta()
	var desc string
	for _, npc := range allNPCs {
		desc += fmt.Sprintf("%s **%s**\n", npc.Emoji, npc.Name)
	}
	embed.Description = desc
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
}

func (c *Cog) menu(lang string, b *interaction.Bot, i *discordgo.InteractionCreate) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	userID := interaction.ToInt64(interaction.UserID(i))
	embed := components.Embed(i18n.T("npcs.list_title", lang), "", 0x3498db)
	var desc string
	allReps, _ := c.svc.GetAllReputations(userID)
	repMap := map[string]*model.UserNPCReputation{}
	for _, r := range allReps {
		repMap[r.NPCID] = &r
	}
	allNPCs := c.svc.GetAllNPCMeta()
	for _, npc := range allNPCs {
		rep := repMap[npc.ID]
		lvl := 1
		points := 0
		if rep != nil {
			lvl = rep.Level
			points = rep.Reputation
		}
		nextLvl := 100 * lvl
		rankName := npcsvc.RankName(lvl)
		filled := points * 10 / nextLvl
		if filled > 10 {
			filled = 10
		}
		bar := strings.Repeat("🟩", filled) + strings.Repeat("⬛", 10-filled)
		desc += fmt.Sprintf("%s **%s** — *%s*\n%s Lvl %d (%d/%d)\n\n", npc.Emoji, npc.Name, rankName, bar, lvl, points, nextLvl)
	}
	embed.Description = desc
	var options []discordgo.SelectMenuOption
	for _, npc := range allNPCs {
		options = append(options, discordgo.SelectMenuOption{
			Label: npc.Name,
			Value: npc.ID,
			Emoji: &discordgo.ComponentEmoji{Name: npc.Emoji},
		})
	}
	menu := discordgo.SelectMenu{
		CustomID:    components.Encode("npc", "select"),
		Placeholder: i18n.T("npcs.select_placeholder", lang),
		Options:     options,
	}
	comps := []discordgo.MessageComponent{
		components.ActionRow(menu),
	}
	return embed, comps
}

func timeGreeting(base string, last time.Time) string {
	if last.IsZero() {
		return base
	}
	hoursAgo := time.Since(last).Hours()
	switch {
	case hoursAgo < 1:
		return "Welcome back!"
	case hoursAgo < 4:
		return "Back already?"
	case hoursAgo < 12:
		return "Good to see you again."
	default:
		return base
	}
}

func (c *Cog) makeNPCSelect(npcID string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
		userID := interaction.ToInt64(interaction.UserID(i))
		npcData := c.svc.GetNPCData(npcID)
		if npcData == nil {
			return
		}
		rep, _ := c.svc.GetReputation(userID, npcID)
		lvl := rep.Level
		points := rep.Reputation
		nextLvl := 100 * lvl
		rankName := npcsvc.RankName(lvl)
		filled := points * 10 / nextLvl
		if filled > 10 {
			filled = 10
		}
		bar := strings.Repeat("🟩", filled) + strings.Repeat("⬛", 10-filled)

		greeting := npcData.Greetings(lang)[0]
		if lvl >= 2 && len(npcData.Greetings(lang)) > 1 {
			greeting = npcData.Greetings(lang)[1]
		}
		if lvl >= 3 && len(npcData.Greetings(lang)) > 2 {
			greeting = npcData.Greetings(lang)[2]
		}
		greeting = timeGreeting(greeting, rep.LastInteraction)

		desc := fmt.Sprintf("*\"%s\"*\n\n", greeting)
		desc += fmt.Sprintf("**%s** · Lvl %d · %s\n", rankName, lvl, bar)
		desc += fmt.Sprintf("%s (%d/%d)\n\n", i18n.T("npcs.affinity_label_desc", lang, map[string]any{"points": points, "next": nextLvl}), points, nextLvl)
		desc += fmt.Sprintf("━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		desc += fmt.Sprintf("%s %s", npcData.Emoji, npcData.Role(lang))

		embed := components.Embed(
			fmt.Sprintf("%s %s", npcData.Emoji, npcData.Name),
			desc,
			npcData.Color,
		)

		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button("💬 "+i18n.T("npcs.chat_btn", lang), components.Encode("npc", "chat_"+npcID), discordgo.PrimaryButton),
				components.Button("🎁 "+i18n.T("npcs.gift_button", lang), components.Encode("npc", "gift_"+npcID), discordgo.SuccessButton),
				components.Button("📜 "+i18n.T("npcs.topic_bio", lang), components.Encode("npc", "bio_"+npcID), discordgo.SecondaryButton),
			),
			components.ActionRow(
				components.Button("💡 "+i18n.T("npcs.topic_advice", lang), components.Encode("npc", "advice_"+npcID), discordgo.SecondaryButton),
				components.Button("👑 "+i18n.T("npcs.rankup_button", lang), components.Encode("npc", "rankup_"+npcID), discordgo.SecondaryButton),
				components.Button("🏪 "+i18n.T("npcs.shop_button", lang), components.Encode("npc", "shop_"+npcID), discordgo.SecondaryButton),
			),
			components.ActionRow(
				components.Button("↩️", components.Encode("npc", "back"), discordgo.DangerButton),
			),
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
	}
}

func (c *Cog) onNPCSelect(b *interaction.Bot, i *discordgo.InteractionCreate) {
	cid := i.MessageComponentData().CustomID
	_, _, _ = components.Decode(cid)
	vals := i.MessageComponentData().Values
	if len(vals) < 1 {
		return
	}
	npcID := vals[0]
	makeNPCSelect := c.makeNPCSelect(npcID)
	makeNPCSelect(b, i)
}

func (c *Cog) onChat(npcID string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
		userID := interaction.ToInt64(interaction.UserID(i))
		npcData := c.svc.GetNPCData(npcID)
		if npcData == nil {
			return
		}

		event, err := c.svc.Chat(userID, npcID, lang)
		if err != nil {
			var cde *npcsvc.ChatCooldownError
			if errors.As(err, &cde) {
				mins := int(time.Until(cde.Until).Minutes())
				if mins < 1 {
					mins = 1
				}
				desc := i18n.T("npcs.chat_cooldown", lang, map[string]any{
					"name":    npcData.Name,
					"minutes": mins,
				})
				embed := components.Embed(
					fmt.Sprintf("%s %s — %s", npcData.Emoji, npcData.Name, i18n.T("npcs.chat_btn", lang)),
					desc,
					npcData.Color,
				)
				comps := []discordgo.MessageComponent{
					components.ActionRow(
						components.Button("↩️", components.Encode("npc", npcID), discordgo.SecondaryButton),
					),
				}
				_ = b.Session.InteractionRespond(i.Interaction,
					components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
			}
			return
		}

		desc := fmt.Sprintf("*\"%s\"*\n", event.Text)
		if event.RepBonus > 0 {
			desc += fmt.Sprintf("\n➕ +%d %s", event.RepBonus, i18n.T("npcs.rep_points", lang))
		}
		if event.ID == "secret" || strings.HasPrefix(event.ID, "secret_") {
			desc += "\n\n*✨ A rare moment of trust...*"
		}

		embed := components.Embed(
			fmt.Sprintf("%s %s — %s", npcData.Emoji, npcData.Name, i18n.T("npcs.chat_btn", lang)),
			desc,
			npcData.Color,
		)
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button("💬 "+i18n.T("npcs.chat_again_btn", lang), components.Encode("npc", "chat_"+npcID), discordgo.PrimaryButton),
				components.Button("↩️", components.Encode("npc", npcID), discordgo.SecondaryButton),
			),
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
	}
}

func (c *Cog) onGiftRequest(npcID string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
		npcData := c.svc.GetNPCData(npcID)
		if npcData == nil {
			return
		}
		placeholder := i18n.T("npcs.gift_placeholder", lang)
		_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID: components.Encode("npc", "gift_submit", npcID),
				Title:    i18n.T("npcs.gift_title", lang, map[string]any{"name": npcData.Name}),
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "gift_item",
							Label:       i18n.T("npcs.gift_item_label", lang),
							Style:       discordgo.TextInputShort,
							Required:    true,
							Placeholder: placeholder,
							MaxLength:   100,
						},
					}},
					discordgo.ActionsRow{Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "gift_qty",
							Label:     i18n.T("npcs.gift_qty_label", lang),
							Style:     discordgo.TextInputShort,
							Required:  false,
							MaxLength: 3,
						},
					}},
				},
			},
		})
	}
}

func (c *Cog) onGiftSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.ModalSubmitData().CustomID)
	if len(rest) < 1 {
		return
	}
	npcID := rest[0]

	npcData := c.svc.GetNPCData(npcID)
	if npcData == nil {
		return
	}

	itemID := i.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	qtyStr := i.ModalSubmitData().Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	qty := 1
	if qtyStr != "" {
		fmt.Sscanf(qtyStr, "%d", &qty)
	}
	if qty < 1 {
		qty = 1
	}

	repGained, err := c.svc.GiftItem(userID, npcID, itemID, qty)
	desc := ""
	if err != nil {
		desc = i18n.T("npcs.gift_fail", lang, map[string]any{"error": err.Error()})
	} else {
		desc = i18n.T("npcs.gift_success", lang, map[string]any{
			"name":   npcData.Name,
			"qty":    qty,
			"item":   itemID,
			"points": repGained,
		})
	}

	embed := components.Embed(
		fmt.Sprintf("🎁 %s — %s", npcData.Name, i18n.T("npcs.gift_button", lang)),
		desc,
		npcData.Color,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("↩️", components.Encode("npc", npcID), discordgo.SecondaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onBio(npcID string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
		npcData := c.svc.GetNPCData(npcID)
		if npcData == nil {
			return
		}
		embed := components.Embed(
			fmt.Sprintf("%s %s — %s", npcData.Emoji, npcData.Name, i18n.T("npcs.topic_bio", lang)),
			npcData.Description(lang),
			npcData.Color,
		)
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button("↩️", components.Encode("npc", npcID), discordgo.SecondaryButton),
			),
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
	}
}

func (c *Cog) onAdvice(npcID string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
		npcData := c.svc.GetNPCData(npcID)
		if npcData == nil {
			return
		}
		embed := components.Embed(
			fmt.Sprintf("%s %s — %s", npcData.Emoji, npcData.Name, i18n.T("npcs.topic_advice", lang)),
			"*"+npcData.Advice(lang)+"*",
			npcData.Color,
		)
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button("↩️", components.Encode("npc", npcID), discordgo.SecondaryButton),
			),
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
	}
}

func (c *Cog) onRankUp(npcID string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
		userID := interaction.ToInt64(interaction.UserID(i))
		npcData := c.svc.GetNPCData(npcID)
		if npcData == nil {
			return
		}

		rep, _ := c.svc.GetReputation(userID, npcID)
		currentLevel := rep.Level

		rankName := npcsvc.RankName(currentLevel)
		nextRankName := npcsvc.RankName(currentLevel + 1)

		var desc string
		var comps []discordgo.MessageComponent

		if currentLevel >= 5 {
			desc = i18n.T("npcs.rankup_max", lang, map[string]any{"name": npcData.Name})
		} else if rep.Reputation >= 100*currentLevel {
			err := c.svc.RankUp(userID, npcID)
			if err != nil {
				desc = i18n.T("npcs.rankup_fail", lang, map[string]any{"error": err.Error()})
			} else {
				desc = i18n.T("npcs.rankup_success", lang, map[string]any{
					"rank": nextRankName,
					"name": npcData.Name,
				})
			}
		} else {
			needed := 100*currentLevel - rep.Reputation
			desc = i18n.T("npcs.rankup_no_rep", lang, map[string]any{
				"name":   npcData.Name,
				"rank":   rankName,
				"needed": needed,
				"level":  currentLevel,
			})
		}

		embed := components.Embed(
			fmt.Sprintf("👑 %s — %s", npcData.Name, i18n.T("npcs.rankup_button", lang)),
			desc,
			npcData.Color,
		)
		comps = []discordgo.MessageComponent{
			components.ActionRow(
				components.Button("↩️", components.Encode("npc", npcID), discordgo.SecondaryButton),
			),
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
	}
}

func (c *Cog) onShop(npcID string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
		userID := interaction.ToInt64(interaction.UserID(i))
		npcData := c.svc.GetNPCData(npcID)
		if npcData == nil {
			return
		}

		items := c.svc.GetAvailableShopItems(userID, npcID)
		rep, _ := c.svc.GetReputation(userID, npcID)

		var desc string
		if len(items) == 0 {
			desc = i18n.T("npcs.shop_empty", lang, map[string]any{"name": npcData.Name})
		} else {
			desc = i18n.T("npcs.shop_title", lang, map[string]any{"name": npcData.Name}) + "\n\n"
			desc += fmt.Sprintf("💰 %s: %d\n", i18n.T("npcs.rep_points", lang), rep.Reputation)
			desc += fmt.Sprintf("🪙 %s: ?\n\n", i18n.T("npcs.coins_label", lang))
			for _, item := range items {
				label := item.LabelEN
				if lang == "fr" {
					label = item.LabelFR
				}
				locked := ""
				if rep.Level < item.MinLevel {
					locked = " 🔒"
				}
				desc += fmt.Sprintf("%s **%s** — %d rep + %d coins%s\n", item.Emoji, label, item.RepCost, item.CoinCost, locked)
			}
		}

		embed := components.Embed(
			fmt.Sprintf("🏪 %s — %s", npcData.Name, i18n.T("npcs.shop_button", lang)),
			desc,
			npcData.Color,
		)

		var comps []discordgo.MessageComponent
		var row []discordgo.MessageComponent
		for _, item := range items {
			if rep.Level >= item.MinLevel {
				label := item.LabelEN
				if lang == "fr" {
					label = item.LabelFR
				}
				row = append(row, components.Button(
					item.Emoji+" "+label,
					components.Encode("npc", "buy_"+npcID+"_"+item.ItemID),
					discordgo.SuccessButton,
				))
			}
		}
		if len(row) > 0 {
			comps = append(comps, components.ActionRow(row...))
		}
		comps = append(comps, components.ActionRow(
			components.Button("↩️", components.Encode("npc", npcID), discordgo.SecondaryButton),
		))

		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
	}
}

func (c *Cog) onShopBuy(npcID string, itemID string) func(b *interaction.Bot, i *discordgo.InteractionCreate) {
	return func(b *interaction.Bot, i *discordgo.InteractionCreate) {
		lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
		userID := interaction.ToInt64(interaction.UserID(i))
		npcData := c.svc.GetNPCData(npcID)
		if npcData == nil {
			return
		}

		err := c.svc.ShopBuy(userID, npcID, itemID)
		var desc string
		if err != nil {
			desc = i18n.T("npcs.shop_buy_fail", lang, map[string]any{"error": err.Error()})
		} else {
			desc = i18n.T("npcs.shop_buy_success", lang, map[string]any{"item": itemID, "name": npcData.Name})
		}

		embed := components.Embed(
			fmt.Sprintf("🏪 %s", npcData.Name),
			desc,
			npcData.Color,
		)
		comps := []discordgo.MessageComponent{
			components.ActionRow(
				components.Button("↩️ "+i18n.T("npcs.shop_button", lang), components.Encode("npc", "shop_"+npcID), discordgo.SecondaryButton),
				components.Button("↩️", components.Encode("npc", npcID), discordgo.SecondaryButton),
			),
		}
		_ = b.Session.InteractionRespond(i.Interaction,
			components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
	}
}

func (c *Cog) onBack(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(lang, b, i)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}
