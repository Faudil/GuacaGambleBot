package loan

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	loansvc "guacagamblebot/internal/service/loan"
	"guacagamblebot/internal/store"
)

// Cog implements the Loan "embed interface": borrow, repay and list debts.
type Cog struct {
	store *store.Store
	cfg   *config.Config
	svc   *loansvc.Service
}

// Register wires the cog into the router under both slash and prefix triggers.
func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg, svc: loansvc.New(s)}
	r.Slash("loan", "cmd.loan.desc", c.onSlashMenu)
	r.Prefix("loan", c.onPrefixMenu)
	r.Prefix("ln", c.onPrefixMenu)
	r.Component("loan", "borrow", c.onBorrowOpen)
	r.Component("loan", "repay", c.onRepayOpen)
	r.Component("loan", "list", c.onList)
	r.Modal("loan", "borrow_submit", c.onBorrowSubmit)
	r.Modal("loan", "repay_submit", c.onRepaySubmit)
}

func (c *Cog) onSlashMenu(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	embed, comps := c.menu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, embed, comps))
}

func (c *Cog) onPrefixMenu(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	lang := c.store.GetLanguage(interaction.ToInt64(m.GuildID))
	userID := interaction.ToInt64(m.Author.ID)
	embed, comps := c.menu(lang, userID)
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: comps,
	})
}

func (c *Cog) menu(lang string, userID int64) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	embed := components.Embed(
		i18n.T("loan.menu_title", lang),
		i18n.T("loan.menu_desc", lang),
		components.ColorInfo,
	)
	comps := []discordgo.MessageComponent{
		components.ActionRow(
			components.Button(i18n.T("loan.btn_borrow", lang), components.EncodeOwner(userID, "loan", "borrow"), discordgo.PrimaryButton),
			components.Button(i18n.T("loan.btn_repay", lang), components.EncodeOwner(userID, "loan", "repay"), discordgo.DangerButton),
			components.Button(i18n.T("loan.btn_list", lang), components.EncodeOwner(userID, "loan", "list"), discordgo.SecondaryButton),
		),
	}
	return embed, comps
}

func (c *Cog) onBorrowOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("loan", "borrow_submit"),
		i18n.T("loan.borrow_modal_title", lang),
		components.TextInput("amount", i18n.T("loan.amount_label", lang), true, "100", discordgo.TextInputShort, 1, 12),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onRepayOpen(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	modal := components.ModalResponse(
		components.Encode("loan", "repay_submit"),
		i18n.T("loan.repay_title", lang),
		components.TextInput("amount", i18n.T("loan.amount_label", lang), true, "100", discordgo.TextInputShort, 1, 12),
	)
	_ = b.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: modal,
	})
}

func (c *Cog) onBorrowSubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	values := interaction.ModalValues(i)
	amount, err := strconv.Atoi(strings.TrimSpace(values["amount"]))
	if err != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "loan.invalid")
		return
	}
	if berr := c.svc.Borrow(int(userID), amount); berr != nil {
		switch berr {
		case loansvc.ErrMaxLoan:
			interaction.RespondError(b, i, lang, "loan.max_loan")
		default:
			interaction.RespondError(b, i, lang, "loan.invalid")
		}
		return
	}
	embed := components.Embed(i18n.T("loan.borrow_done", lang, map[string]any{"amount": amount}), "", components.ColorSuccess)
	_, comps := c.menu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onRepaySubmit(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	values := interaction.ModalValues(i)
	amount, err := strconv.Atoi(strings.TrimSpace(values["amount"]))
	if err != nil || amount <= 0 {
		interaction.RespondError(b, i, lang, "loan.invalid")
		return
	}
	paid, rerr := c.svc.Repay(int(userID), amount)
	if rerr != nil {
		switch rerr {
		case loansvc.ErrNoDebt:
			interaction.RespondError(b, i, lang, "loan.no_debt")
		case loansvc.ErrExceedsDebt:
			interaction.RespondError(b, i, lang, "loan.exceeds")
		default:
			interaction.RespondError(b, i, lang, "loan.invalid")
		}
		return
	}
	embed := components.Embed(i18n.T("loan.repay_done", lang, map[string]any{"amount": amount}), "", components.ColorSuccess)
	_ = paid
	_, comps := c.menu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}

func (c *Cog) onList(b *interaction.Bot, i *discordgo.InteractionCreate) {
	lang := c.store.GetLanguage(interaction.ToInt64(i.GuildID))
	userID := interaction.ToInt64(interaction.UserID(i))
	loans, err := c.svc.List(userID)
	if err != nil {
		interaction.RespondError(b, i, lang, "loan.invalid")
		return
	}
	embed := components.Embed(i18n.T("loan.list_title", lang), "", components.ColorInfo)
	if len(loans) == 0 {
		embed.Description = i18n.T("loan.list_empty", lang)
	} else {
		desc := ""
		for _, l := range loans {
			lender := i18n.T("loan.bank_name", lang)
			if l.LenderID != 0 {
				lender = interaction.Mention(l.LenderID)
			}
			desc += i18n.T("loan.list_entry", lang, map[string]any{
				"lender": lender,
				"amount": strconv.Itoa(l.AmountDue),
			}) + "\n"
		}
		embed.Description = desc
	}
	_, comps := c.menu(lang, userID)
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseUpdateMessage, embed, comps))
}
