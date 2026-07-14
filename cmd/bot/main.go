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

	economycog "guacagamblebot/internal/cogs/economy"
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
	router := interaction.NewRouter(bot)

	str := store.New(database, cfg)
	economycog.Register(router, str, cfg)

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
