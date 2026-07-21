package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/store"

	achievementscog "guacagamblebot/internal/cogs/achievements"
	admincog "guacagamblebot/internal/cogs/admin"
	archeologycog "guacagamblebot/internal/cogs/archeology"
	bankcog "guacagamblebot/internal/cogs/bank"
	bettingcog "guacagamblebot/internal/cogs/betting"
	blackjackcog "guacagamblebot/internal/cogs/blackjack"
	bosscog "guacagamblebot/internal/cogs/boss"
	casinocog "guacagamblebot/internal/cogs/casino"
	charactercog "guacagamblebot/internal/cogs/character"
	communitycog "guacagamblebot/internal/cogs/community"
	craftingcog "guacagamblebot/internal/cogs/crafting"
	duelcog "guacagamblebot/internal/cogs/duel"
	economycog "guacagamblebot/internal/cogs/economy"
	expeditioncog "guacagamblebot/internal/cogs/expedition"
	farmcog "guacagamblebot/internal/cogs/farm"
	fishingcog "guacagamblebot/internal/cogs/fishing"
	housingcog "guacagamblebot/internal/cogs/housing"
	huntcog "guacagamblebot/internal/cogs/hunt"
	inventorycog "guacagamblebot/internal/cogs/inventory"
	itemmanagercog "guacagamblebot/internal/cogs/item_manager"
	jobscog "guacagamblebot/internal/cogs/jobs"
	leadercog "guacagamblebot/internal/cogs/leaderboard"
	loancog "guacagamblebot/internal/cogs/loan"
	lottocog "guacagamblebot/internal/cogs/lotto"
	marketcog "guacagamblebot/internal/cogs/market"
	miningcog "guacagamblebot/internal/cogs/mining"
	npcscog "guacagamblebot/internal/cogs/npcs"
	petscog "guacagamblebot/internal/cogs/pets"
	questscog "guacagamblebot/internal/cogs/quests"
	roulettecog "guacagamblebot/internal/cogs/roulette"
	shopcog "guacagamblebot/internal/cogs/shop"
	tournamentcog "guacagamblebot/internal/cogs/tournament"
	"guacagamblebot/internal/onboarding"
)

func main() {
	cfg := config.Load()

	if cfg.TZ != "" {
		_ = os.Setenv("TZ", cfg.TZ)
	}
	if cfg.DiscordToken == "" {
		log.Fatal("DISCORD_TOKEN is not set. Add it to your .env file.")
	}

	if err := i18n.Load("locales"); err != nil {
		log.Printf("warning: could not load locales: %v", err)
	}

	database, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatalf("failed to create discord session: %v", err)
	}
	dg.Identify.Intents = discordgo.IntentGuilds |
		discordgo.IntentGuildMessages |
		discordgo.IntentMessageContent

	bot := &interaction.Bot{Session: dg, DB: database, Prefix: cfg.Prefix}
	str := store.New(database, cfg)
	router := interaction.NewRouter(bot, str)

	admincog.Register(router, str, cfg)
	achievementscog.Register(router, str, cfg)
	archeologycog.Register(router, str, cfg)
	bankcog.Register(router, str, cfg)
	bettingcog.Register(router, str, cfg)
	blackjackcog.Register(router, str, cfg)
	bosscog.Register(router, str, cfg)
	casinocog.Register(router, str, cfg)
	charactercog.Register(router, str, cfg)
	communitycog.Register(router, str, cfg)
	craftingcog.Register(router, str, cfg)
	duelcog.Register(router, str, cfg)
	economycog.Register(router, str, cfg)
	expeditioncog.Register(router, str, cfg)
	farmcog.Register(router, str, cfg)
	fishingcog.Register(router, str, cfg)
	housingcog.Register(router, str, cfg)
	huntcog.Register(router, str, cfg)
	inventorycog.Register(router, str, cfg)
	itemmanagercog.Register(router, str, cfg)
	jobscog.Register(router, str, cfg)
	leadercog.Register(router, str, cfg)
	loancog.Register(router, str, cfg)
	lottocog.Register(router, str, cfg)
	marketcog.Register(router, str, cfg)
	miningcog.Register(router, str, cfg)
	npcscog.Register(router, str, cfg)
	petscog.Register(router, str, cfg)
	questscog.Register(router, str, cfg)
	roulettecog.Register(router, str, cfg)
	shopcog.Register(router, str, cfg)
	tournamentcog.Register(router, str, cfg)
	onboarding.Register(router, str, cfg)

	router.Register()

	guildID := ""
	if cfg.GuildID != 0 {
		guildID = strconv.FormatInt(cfg.GuildID, 10)
	}
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		if err := router.RegisterCommands(guildID); err != nil {
			log.Printf("warning: could not register slash commands: %v", err)
		}
		log.Printf("Logged in as %s", s.State.User.Username)
	})

	if err := dg.Open(); err != nil {
		log.Fatalf("failed to open discord connection: %v", err)
	}
	defer dg.Close()

	log.Println("GuacaGambleBot (Go) is online. Press CTRL-C to exit.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
	log.Println("Shutting down...")
}
