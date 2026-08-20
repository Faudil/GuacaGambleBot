package help

import (
	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/components"
	"guacagamblebot/internal/config"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/store"
)

type Cog struct {
	store *store.Store
	cfg   *config.Config
}

func Register(r *interaction.Router, s *store.Store, cfg *config.Config) {
	c := &Cog{store: s, cfg: cfg}
	r.Slash("help", "Show a list of all available commands.", c.onSlash)
	r.Prefix("help", c.onPrefix)
	r.Prefix("h", c.onPrefix)
}

func (c *Cog) onSlash(b *interaction.Bot, i *discordgo.InteractionCreate) {
	_ = b.Session.InteractionRespond(i.Interaction,
		components.InteractionResponse(discordgo.InteractionResponseChannelMessageWithSource, c.embed(), nil))
}

func (c *Cog) onPrefix(b *interaction.Bot, s *discordgo.Session, m *discordgo.Message) {
	_, _ = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{c.embed()},
	})
}

func (c *Cog) embed() *discordgo.MessageEmbed {
	prefix := c.cfg.Prefix

	e := &discordgo.MessageEmbed{
		Title:       "\U0001f9e9 GuacaGambleBot — Commands",
		Description: "Start with \"/setup\" to configure the bot for your server.\nPrefix commands also work (e.g. \"" + prefix + "daily\").",
		Color:       0x5865f2,
		Fields: []*discordgo.MessageEmbedField{
			components.Field("\U0001f3b0 Casino",
				"`"+prefix+"coinflip` / `/coinflip` — Heads or tails\n"+
					"`"+prefix+"slots` / `/slots` — Slot machine\n"+
					"`"+prefix+"blackjack` / `/blackjack` — Blackjack duel\n"+
					"`"+prefix+"roulette` / `/roulette` — Russian roulette\n"+
					"`"+prefix+"lotto` / `/lotto` — Server lottery\n"+
					"`"+prefix+"betting` / `/betting` — Custom bets\n"+
					"`"+prefix+"casino` / `/casino` — All casino games",
				true),
			components.Field("\U0001f4b0 Economy",
				"`"+prefix+"economy` / `/economy` — Balance, daily, give\n"+
					"`"+prefix+"daily` / `/daily` — Claim your daily reward\n"+
					"`"+prefix+"bal` / `/bal` — Check your balance\n"+
					"`"+prefix+"give @user 100` / `/give` — Send money\n"+
					"`"+prefix+"bank` / `/bank` — Deposit / withdraw\n"+
					"`"+prefix+"loan` / `/loan` — Borrow & repay\n"+
					"`"+prefix+"jobs` / `/jobs` — Job levels & XP\n"+
					"`"+prefix+"shop` / `/shop` — Buy items\n"+
					"`"+prefix+"market` / `/market` — Sell items",
				true),
			components.Field("\U0001f43e Pets",
				"`"+prefix+"pets` / `/pets` — Manage pets\n"+
					"`"+prefix+"play` / `/play` — Play with your pet\n"+
					"`"+prefix+"hatch` / `/hatch` — Hatch a mystery egg\n"+
					"`"+prefix+"hunt` / `/hunt` — Pet expedition\n"+
					"`"+prefix+"expedition` / `/expedition` — Send pet exploring\n"+
					"`"+prefix+"boss` / `/boss` — Boss league\n"+
					"`"+prefix+"duel` / `/duel` — Pet PvP\n"+
					"`"+prefix+"tournament` / `/tournament` — Pet tournaments",
				true),
			components.Field("\u26cf\uFE0F Activities",
				"`"+prefix+"mine` / `/mine` — Mining expedition\n"+
					"`"+prefix+"fish` / `/fish` — Fishing minigame\n"+
					"`"+prefix+"farm` / `/farm` — Farming\n"+
					"`"+prefix+"dig` / `/dig` — Fossil excavation\n"+
					"`"+prefix+"craft` / `/craft` — Crafting recipes",
				true),
			components.Field("\U0001f3e0 Social & RPG",
				"`"+prefix+"house` / `/house` — Buy & upgrade home\n"+
					"`"+prefix+"forge` / `/forge` — Fuse or scrap equipment\n"+
					"`"+prefix+"character` / `/character` — Player profile\n"+
					"`"+prefix+"skills` / `/skills` — Activate skills\n"+
					"`"+prefix+"inventory` / `/inventory` — Your bag\n"+
					"`"+prefix+"achievements` / `/achievements` — Trophies\n"+
					"`"+prefix+"lore` / `/lore` — Lore codex\n"+
					"`"+prefix+"quest` / `/quest` — Active quests\n"+
					"`"+prefix+"npc` / `/npc` — Talk to villagers\n"+
					"`"+prefix+"community` / `/community` — Community projects\n"+
					"`"+prefix+"leaderboard` / `/leaderboard` — Rankings",
				true),
			components.Field("\U0001f6e0\uFE0F Admin",
				"`"+prefix+"setup` / `/setup` — Configure the bot\n"+
					"`"+prefix+"admin` / `/admin` — Admin panel\n"+
					"`"+prefix+"help` / `/help` — This menu",
				true),
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Pro tip: you can change the prefix with /setup!",
		},
	}
	return e
}
