package casino

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/achievement"
	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	casinosvc "guacagamblebot/internal/service/casino"
	invsvc "guacagamblebot/internal/service/inventory"
	jsvc "guacagamblebot/internal/service/journal"
	npcsvc "guacagamblebot/internal/service/npcs"
	questssvc "guacagamblebot/internal/service/quests"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe"
)

var one = float64(1)

type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *casinosvc.Service
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	def := universe.Get(cfg.Universe)
	if def == nil {
		def = universe.Get("hoakhaven")
	}
	inv := invsvc.New(s, cfg)
	npcSvc := npcsvc.New(s, cfg, def, inv)
	c := &Cog{store: s, cfg: cfg, svc: casinosvc.New(s, cfg, npcSvc)}
	r.SlashWithOptions("casino", "Casino : machines à sous et pile ou face.",
		[]*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "amount",
				Description: "Montant à miser (optionnel)",
				Required:    false,
				MinValue:    &one,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "choice",
				Description: "Heads or Tails (optionnel - pour pile ou face direct)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Heads", Value: "heads"},
					{Name: "Tails", Value: "tails"},
				},
			},
		},
		c.onSlashMenu)
	r.Slash("cas", "Casino : machines à sous et pile ou face.", c.onSlashMenu)
	r.Prefix("casino", c.onPrefixMenu)
	r.Prefix("cas", c.onPrefixMenu)
	r.Component("casino", "slots", c.onSlotsOpen)
	r.Component("casino", "coinflip", c.onCoinflipOpen)
	r.Component("casino", "coinflip_choice", c.onCoinflipChoice)
	r.Component("casino", "slots_retry", c.onSlotsRetry)
	r.Component("casino", "coinflip_retry", c.onCoinflipRetry)
	r.Modal("casino", "slots_submit", c.onSlotsSubmit)
	r.Modal("casino", "coinflip_submit", c.onCoinflipSubmit)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	userID := interaction.ToInt64(interaction.UserID(i))
	opts := i.ApplicationCommandData().Options
	if len(opts) > 0 {
		amount := 0
		choice := ""
		for _, opt := range opts {
			switch opt.Name {
			case "amount":
				amount = int(opt.IntValue())
			case "choice":
				choice = opt.StringValue()
			}
		}
		if choice != "" && amount > 0 {
			c.playCoinflip(b, i, choice, amount, discordgo.InteractionResponseChannelMessageWithSource)
			return
		}
		if amount > 0 {
			c.playSlots(b, i, amount, discordgo.InteractionResponseChannelMessageWithSource)
			return
		}
	}
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	embed, comps := c.menu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	userID := interaction.ToInt64(m.Author.ID)
	parts := strings.Fields(m.Content)
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))

	if len(parts) > 2 {
		choice := strings.ToLower(parts[1])
		if choice == "heads" || choice == "tails" || choice == "face" || choice == "pile" {
			amount, err := strconv.Atoi(parts[2])
			if err == nil && amount > 0 {
				c.playCoinflipFromPrefix(b, s, m, choice, amount)
				return
			}
		}
	}
	if len(parts) > 1 {
		amount, err := strconv.Atoi(parts[1])
		if err == nil && amount > 0 {
			c.playSlotsFromPrefix(b, s, m, amount)
			return
		}
	}

	embed, comps := c.menu(lang, userID)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) menu(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := components.Embed(
		i18n.T("slots.title", lang),
		i18n.T("slots.state_start", lang),
		0xf1c40f,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("🎰 "+i18n.T("slots.title", lang), components.EncodeOwner(userID, "casino", "slots"), discordgo.PrimaryButton),
			components.Button("🪙 "+i18n.T("coinflip.legit_label", lang), components.EncodeOwner(userID, "casino", "coinflip"), discordgo.SuccessButton),
		),
	}
	return embed, comps
}

func (c *Cog) slotsEmbed(s1, s2, s3, stateText string, amount int, lang string, color int) *discordgo.MessageEmbed {
	machineDisplay := fmt.Sprintf("**»** %s   |   %s   |   %s  ****", s1, s2, s3)
	infoDisplay := i18n.T("slots.bet_info", lang, map[string]any{"amount": amount}) + "\n" + stateText
	return &discordgo.MessageEmbed{
		Title: "🎰 CASINO",
		Color: color,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Machine", Value: "# " + machineDisplay, Inline: false},
			{Name: "Infos", Value: infoDisplay, Inline: false},
		},
	}
}

func (c *Cog) onSlotsOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	modal := components.ModalResponse(
		components.EncodeOwner(userID, "casino", "slots_submit"),
		i18n.T("slots.title", lang),
		components.TextInput("amount", i18n.T("economy.quantity", lang), true, "50", discordgo.TextInputShort, 1, 12),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onCoinflipOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed := components.Embed(i18n.T("slots.title", lang), i18n.T("coinflip.choose_prompt", lang), 0xf1c40f)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("coinflip.heads_label", lang), components.EncodeOwner(userID, "casino", "coinflip_choice", "heads"), discordgo.PrimaryButton),
			components.Button(i18n.T("coinflip.tails_label", lang), components.EncodeOwner(userID, "casino", "coinflip_choice", "tails"), discordgo.PrimaryButton),
		),
	}
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: comps,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *Cog) onCoinflipChoice(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	if len(rest) == 0 {
		interaction.RespondError(b, i, lang, "coinflip.choice_error")
		return
	}
	choice := rest[0]
	if choice != "heads" && choice != "tails" && choice != "face" && choice != "pile" {
		interaction.RespondError(b, i, lang, "coinflip.choice_error")
		return
	}
	modal := components.ModalResponse(
		components.EncodeOwner(userID, "casino", "coinflip_submit", choice),
		i18n.T("coinflip.legit_label", lang),
		components.TextInput("amount", i18n.T("economy.quantity", lang), true, "100", discordgo.TextInputShort, 1, 12),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onSlotsSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	values := interaction.ModalValues(i)
	amount, err := strconv.Atoi(strings.TrimSpace(values["amount"]))
	if err != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "coinflip.invalid_bet")
		return
	}
	c.playSlots(b, i, amount, discordgo.InteractionResponseUpdateMessage)
}

func (c *Cog) onCoinflipSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	_, _, rest := components.Decode(i.ModalSubmitData().CustomID)
	if len(rest) == 0 {
		interaction.RespondError(b, i, lang, "coinflip.choice_error")
		return
	}
	choice := rest[0]
	values := interaction.ModalValues(i)
	amount, err := strconv.Atoi(strings.TrimSpace(values["amount"]))
	if err != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "coinflip.invalid_bet")
		return
	}
	c.playCoinflip(b, i, choice, amount, discordgo.InteractionResponseChannelMessageWithSource)
}

func (c *Cog) onSlotsRetry(b *interaction.Bot, i *discordgo.InteractionCreate) {
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if len(rest) == 0 {
		interaction.RespondError(b, i, lang, "coinflip.invalid_bet")
		return
	}
	amount, err := strconv.Atoi(rest[0])
	if err != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "coinflip.invalid_bet")
		return
	}
	c.playSlots(b, i, amount, discordgo.InteractionResponseUpdateMessage)
}

func (c *Cog) onCoinflipRetry(b *interaction.Bot, i *discordgo.InteractionCreate) {
	_, _, rest := components.Decode(i.MessageComponentData().CustomID)
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	if len(rest) < 2 {
		interaction.RespondError(b, i, lang, "coinflip.invalid_bet")
		return
	}
	choice := rest[0]
	amount, err := strconv.Atoi(rest[1])
	if err != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "coinflip.invalid_bet")
		return
	}
	c.playCoinflip(b, i, choice, amount, discordgo.InteractionResponseUpdateMessage)
}

func (c *Cog) playSlots(b *interaction.Bot, i *discordgo.InteractionCreate, amount int, responseType discordgo.InteractionResponseType) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	res, serr := c.svc.SpinSlots(userID, amount)
	if serr != nil {
		switch serr {
		case casinosvc.ErrNoMoney:
			interaction.RespondError(b, i, lang, "slots.no_money")
		default:
			interaction.RespondError(b, i, lang, "coinflip.invalid_bet")
		}
		return
	}
	_ = c.store.RecordActivity(userID, "casino_games_played", 1)
	questMsg, _ := c.store.PopQuestNotification(userID)
	if text, dm := jsvc.SceneLine(c.store, userID, "casino", lang); text != "" {
		interaction.SendJournalScene(b, i, text, dm)
	}

	blurple := 0x7289da
	_, menuComps := c.menu(lang, userID)
	embed := c.slotsEmbed("🌀", "🌀", "🌀", i18n.T("slots.state_start", lang), amount, lang, blurple)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(responseType, embed, menuComps))

	go func() {
		time.Sleep(1 * time.Second)

		embed := c.slotsEmbed(res.Symbol1, "🌀", "🌀", i18n.T("slots.state_rolling", lang), amount, lang, blurple)
		_, _ = b.Session.InteractionResponseEdit(i.Interaction, components.WebhookEditResponse(embed, menuComps))

		time.Sleep(500 * time.Millisecond)

		embed = c.slotsEmbed(res.Symbol1, res.Symbol2, "🌀", i18n.T("slots.state_suspense", lang), amount, lang, blurple)
		_, _ = b.Session.InteractionResponseEdit(i.Interaction, components.WebhookEditResponse(embed, menuComps))

		if res.Symbol1 == res.Symbol2 {
			time.Sleep(1500 * time.Millisecond)
		} else {
			time.Sleep(500 * time.Millisecond)
		}

		color := 0xe74c3c
		if res.WinType == "JACKPOT" {
			color = 0xf1c40f
		} else if res.IsWin {
			color = 0x2ecc71
		}
		flavor := c.getSlotsFlavor(res.WinType, res.WinSym, lang)
		var status string
		if res.IsWin {
			status = flavor + "\n" + i18n.T("slots.gain", lang, map[string]any{"amount": res.Payout})
		} else {
			status = flavor + "\n" + i18n.T("slots.loss", lang, map[string]any{"amount": amount})
		}
		if res.LeveledUp {
			status += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": res.NewLevel})
		}

		embed = c.slotsEmbed(res.Symbol1, res.Symbol2, res.Symbol3, status, amount, lang, color)
		resultComps := c.slotsResultComps(amount, lang, userID)
		_, _ = b.Session.InteractionResponseEdit(i.Interaction, components.WebhookEditResponse(embed, resultComps))

		if questMsg.QuestID != "" {
			interaction.SendQuestNotification(b, i, questMsg, lang)
		}

		unlocks, _ := achievement.CheckAndUnlock(b.DB, userID)
		if len(unlocks) > 0 {
			interaction.SendAchievements(b, i, lang, unlocks)
		}
	}()
}

func (c *Cog) playCoinflip(b *interaction.Bot, i *discordgo.InteractionCreate, choice string, amount int, responseType discordgo.InteractionResponseType) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))

	res, cerr := c.svc.Coinflip(userID, choice, amount, false)
	if cerr != nil {
		switch cerr {
		case casinosvc.ErrNoMoney:
			interaction.RespondError(b, i, lang, "coinflip.no_money")
		case casinosvc.ErrChoice:
			interaction.RespondError(b, i, lang, "coinflip.choice_error")
		case casinosvc.ErrMaxBet:
			interaction.RespondError(b, i, lang, "coinflip.max_bet")
		default:
			interaction.RespondError(b, i, lang, "coinflip.invalid_bet")
		}
		return
	}
	_ = c.store.RecordActivity(userID, "casino_games_played", 1)
	questMsg, _ := c.store.PopQuestNotification(userID)
	if text, dm := jsvc.SceneLine(c.store, userID, "casino", lang); text != "" {
		interaction.SendJournalScene(b, i, text, dm)
	}

	_, menuComps := c.menu(lang, userID)
	blurple := 0x7289da
	embed := c.slotsEmbed("🌀", "🌀", "🌀", i18n.T("slots.state_start", lang), amount, lang, blurple)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(responseType, embed, menuComps))

	go func() {
		time.Sleep(600 * time.Millisecond)

		embed := components.Embed(i18n.T("slots.title", lang), i18n.T("coinflip.spinning", lang), blurple)
		_, _ = b.Session.InteractionResponseEdit(i.Interaction, components.WebhookEditResponse(embed, menuComps))

		time.Sleep(600 * time.Millisecond)

		var text string
		color := 0x2ecc71
		if res.Win {
			text = i18n.T("coinflip.win_msg", lang, map[string]any{"result": strings.ToUpper(res.Result)})
		} else {
			text = i18n.T("coinflip.lose_msg", lang, map[string]any{"result": strings.ToUpper(res.Result)})
			color = 0xe74c3c
		}
		if res.LeveledUp {
			text += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": res.NewLevel})
		}

		embed = components.Embed(i18n.T("slots.title", lang), text, color)
		resultComps := c.coinflipResultComps(choice, amount, lang, userID)
		_, _ = b.Session.InteractionResponseEdit(i.Interaction, components.WebhookEditResponse(embed, resultComps))

		if questMsg.QuestID != "" {
			interaction.SendQuestNotification(b, i, questMsg, lang)
		}

		unlocks, _ := achievement.CheckAndUnlock(b.DB, userID)
		if len(unlocks) > 0 {
			interaction.SendAchievements(b, i, lang, unlocks)
		}
	}()
}

func (c *Cog) playSlotsFromPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message, amount int) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)

	res, serr := c.svc.SpinSlots(userID, amount)
	if serr != nil {
		var msg string
		switch serr {
		case casinosvc.ErrNoMoney:
			msg = i18n.T("slots.no_money", lang)
		default:
			msg = i18n.T("coinflip.invalid_bet", lang)
		}
		_, _ = s.ChannelMessageSend(m.ChannelID, msg)
		return
	}
	_ = c.store.RecordActivity(userID, "casino_games_played", 1)
	questMsg, _ := c.store.PopQuestNotification(userID)
	if text, dm := jsvc.SceneLine(c.store, userID, "casino", lang); text != "" {
		interaction.SendJournalSceneMsg(s, m.ChannelID, m.Author.ID, text, dm)
	}

	blurple := 0x7289da
	_, menuComps := c.menu(lang, userID)
	embed := c.slotsEmbed("🌀", "🌀", "🌀", i18n.T("slots.state_start", lang), amount, lang, blurple)
	msg, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: menuComps,
	})
	if err != nil || msg == nil {
		return
	}

	msgID := msg.ID
	go func() {
		time.Sleep(1 * time.Second)

		embed := c.slotsEmbed(res.Symbol1, "🌀", "🌀", i18n.T("slots.state_rolling", lang), amount, lang, blurple)
		_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    m.ChannelID,
			ID:         msgID,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &menuComps,
		})

		time.Sleep(500 * time.Millisecond)

		embed = c.slotsEmbed(res.Symbol1, res.Symbol2, "🌀", i18n.T("slots.state_suspense", lang), amount, lang, blurple)
		_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    m.ChannelID,
			ID:         msgID,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &menuComps,
		})

		if res.Symbol1 == res.Symbol2 {
			time.Sleep(1500 * time.Millisecond)
		} else {
			time.Sleep(500 * time.Millisecond)
		}

		color := 0xe74c3c
		if res.WinType == "JACKPOT" {
			color = 0xf1c40f
		} else if res.IsWin {
			color = 0x2ecc71
		}
		flavor := c.getSlotsFlavor(res.WinType, res.WinSym, lang)
		var status string
		if res.IsWin {
			status = flavor + "\n" + i18n.T("slots.gain", lang, map[string]any{"amount": res.Payout})
		} else {
			status = flavor + "\n" + i18n.T("slots.loss", lang, map[string]any{"amount": amount})
		}
		if res.LeveledUp {
			status += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": res.NewLevel})
		}
		if questMsg.QuestID != "" {
			status += "\n\n" + questssvc.QuestNotificationMsg(questMsg, lang)
		}

		embed = c.slotsEmbed(res.Symbol1, res.Symbol2, res.Symbol3, status, amount, lang, color)
		resultComps := c.slotsResultComps(amount, lang, userID)
		_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    m.ChannelID,
			ID:         msgID,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &resultComps,
		})
	}()
}

func (c *Cog) playCoinflipFromPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message, choice string, amount int) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)

	res, cerr := c.svc.Coinflip(userID, choice, amount, false)
	if cerr != nil {
		var msg string
		switch cerr {
		case casinosvc.ErrNoMoney:
			msg = i18n.T("coinflip.no_money", lang)
		case casinosvc.ErrChoice:
			msg = i18n.T("coinflip.choice_error", lang)
		case casinosvc.ErrMaxBet:
			msg = i18n.T("coinflip.max_bet", lang)
		default:
			msg = i18n.T("coinflip.invalid_bet", lang)
		}
		_, _ = s.ChannelMessageSend(m.ChannelID, msg)
		return
	}
	_ = c.store.RecordActivity(userID, "casino_games_played", 1)
	questMsg, _ := c.store.PopQuestNotification(userID)
	if text, dm := jsvc.SceneLine(c.store, userID, "casino", lang); text != "" {
		interaction.SendJournalSceneMsg(s, m.ChannelID, m.Author.ID, text, dm)
	}

	_, menuComps := c.menu(lang, userID)
	blurple := 0x7289da
	startMsg := i18n.T("coinflip.start_msg", lang, map[string]any{"choice": choice, "amount": amount})
	embed := components.Embed(i18n.T("slots.title", lang), startMsg, blurple)
	msg, err := s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: menuComps,
	})
	if err != nil || msg == nil {
		return
	}

	msgID := msg.ID
	go func() {
		time.Sleep(600 * time.Millisecond)

		embed := components.Embed(i18n.T("slots.title", lang), i18n.T("coinflip.spinning", lang), blurple)
		_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    m.ChannelID,
			ID:         msgID,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &menuComps,
		})

		time.Sleep(600 * time.Millisecond)

		var text string
		color := 0x2ecc71
		if res.Win {
			text = i18n.T("coinflip.win_msg", lang, map[string]any{"result": strings.ToUpper(res.Result)})
		} else {
			text = i18n.T("coinflip.lose_msg", lang, map[string]any{"result": strings.ToUpper(res.Result)})
			color = 0xe74c3c
		}
		if res.LeveledUp {
			text += "\n" + i18n.T("character.level_up", lang, map[string]any{"level": res.NewLevel})
		}
		if questMsg.QuestID != "" {
			text += "\n\n" + questssvc.QuestNotificationMsg(questMsg, lang)
		}

		embed = components.Embed(i18n.T("slots.title", lang), text, color)
		resultComps := c.coinflipResultComps(choice, amount, lang, userID)
		_, _ = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
			Channel:    m.ChannelID,
			ID:         msgID,
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &resultComps,
		})
	}()
}

func (c *Cog) slotsResultComps(amount int, lang string, userID int64) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("🔄 Retry", components.EncodeOwner(userID, "casino", "slots_retry", strconv.Itoa(amount)), discordgo.PrimaryButton),
			components.Button("🎰 "+i18n.T("slots.title", lang), components.EncodeOwner(userID, "casino", "slots"), discordgo.PrimaryButton),
			components.Button("🪙 "+i18n.T("coinflip.legit_label", lang), components.EncodeOwner(userID, "casino", "coinflip"), discordgo.SuccessButton),
		),
	}
}

func (c *Cog) coinflipResultComps(choice string, amount int, lang string, userID int64) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		components.ActionRow(
			components.Button("🔄 Retry", components.EncodeOwner(userID, "casino", "coinflip_retry", choice, strconv.Itoa(amount)), discordgo.SuccessButton),
			components.Button("🎰 "+i18n.T("slots.title", lang), components.EncodeOwner(userID, "casino", "slots"), discordgo.PrimaryButton),
			components.Button("🪙 "+i18n.T("coinflip.legit_label", lang), components.EncodeOwner(userID, "casino", "coinflip"), discordgo.SuccessButton),
		),
	}
}

func (c *Cog) getSlotsFlavor(winType, symbol, lang string) string {
	if winType == "JACKPOT" {
		switch symbol {
		case "💎":
			return i18n.T("slots.jackpot_diamond", lang)
		case "🔔":
			return i18n.T("slots.jackpot_bell", lang)
		default:
			return i18n.T("slots.jackpot_generic", lang, map[string]any{"symbol": symbol})
		}
	}
	if winType == "PAIRE" {
		switch symbol {
		case "💎":
			return i18n.T("slots.pair_diamond", lang)
		case "🔔":
			return i18n.T("slots.pair_bell", lang)
		case "🍋":
			return i18n.T("slots.pair_lemon", lang)
		case "🍇":
			return i18n.T("slots.pair_grape", lang)
		case "🍒":
			return i18n.T("slots.pair_cherry", lang)
		}
	}
	return i18n.T("slots.lose_generic", lang)
}
