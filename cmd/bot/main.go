package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"guacagamblebot/internal/config"
	"guacagamblebot/internal/db"
	"guacagamblebot/internal/elosimulation"
	"guacagamblebot/internal/i18n"
	"guacagamblebot/internal/interaction"
	"guacagamblebot/internal/logger"
	"guacagamblebot/internal/store"
	"guacagamblebot/internal/universe/hoakhaven"
	"guacagamblebot/internal/universe/scifi"
	"guacagamblebot/internal/universe/scorch"
	"guacagamblebot/internal/watchdog"

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
	criminalitycog "guacagamblebot/internal/cogs/criminality"
	delvecog "guacagamblebot/internal/cogs/delve"
	duelcog "guacagamblebot/internal/cogs/duel"
	economycog "guacagamblebot/internal/cogs/economy"
	expeditioncog "guacagamblebot/internal/cogs/expedition"
	farmcog "guacagamblebot/internal/cogs/farm"
	fishingcog "guacagamblebot/internal/cogs/fishing"
	helpcog "guacagamblebot/internal/cogs/help"
	housingcog "guacagamblebot/internal/cogs/housing"
	huntcog "guacagamblebot/internal/cogs/hunt"
	inventorycog "guacagamblebot/internal/cogs/inventory"
	itemmanagercog "guacagamblebot/internal/cogs/item_manager"
	jobscog "guacagamblebot/internal/cogs/jobs"
	journalcog "guacagamblebot/internal/cogs/journal"
	leadercog "guacagamblebot/internal/cogs/leaderboard"
	loancog "guacagamblebot/internal/cogs/loan"
	lorecog "guacagamblebot/internal/cogs/lore"
	lottocog "guacagamblebot/internal/cogs/lotto"
	marketcog "guacagamblebot/internal/cogs/market"
	miningcog "guacagamblebot/internal/cogs/mining"
	npcscog "guacagamblebot/internal/cogs/npcs"
	petscog "guacagamblebot/internal/cogs/pets"
	questscog "guacagamblebot/internal/cogs/quests"
	roulettecog "guacagamblebot/internal/cogs/roulette"
	sanctuarycog "guacagamblebot/internal/cogs/sanctuary"
	shopcog "guacagamblebot/internal/cogs/shop"
	skillscog "guacagamblebot/internal/cogs/skills"
	startcog "guacagamblebot/internal/cogs/start"
	tournamentcog "guacagamblebot/internal/cogs/tournament"
	usecog "guacagamblebot/internal/cogs/use"
	veilcog "guacagamblebot/internal/cogs/veil"
	"guacagamblebot/internal/onboarding"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg)
	hoakhaven.Register()
	scifi.Register()
	scorch.Register()

	if cfg.TZ != "" {
		_ = os.Setenv("TZ", cfg.TZ)
	}
	if cfg.DiscordToken == "" {
		slog.Error("DISCORD_TOKEN is not set. Add it to your .env file.")
		os.Exit(1)
	}

	if err := i18n.Load("locales"); err != nil {
		slog.Warn("could not load locales", "error", err)
	}

	database, err := db.Open(cfg)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}

	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		slog.Error("failed to create discord session", "error", err)
		os.Exit(1)
	}
	dg.Identify.Intents = discordgo.IntentGuilds |
		discordgo.IntentGuildMessages |
		discordgo.IntentMessageContent |
		discordgo.IntentGuildMembers

	ctx, stopELO := context.WithCancel(context.Background())

	bot := &interaction.Bot{Session: dg, DB: database, Prefix: cfg.Prefix}
	str := store.New(database, cfg)
	go elosimulation.Run(ctx, str)
	go elosimulation.RunWeeklyReset(ctx, str, dg)
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
	criminalitycog.Register(router, str, cfg)
	delvecog.Register(router, str, cfg)
	duelcog.Register(router, str, cfg)
	economycog.Register(router, str, cfg)
	expeditioncog.Register(router, str, cfg)
	farmcog.Register(router, str, cfg)
	fishingcog.Register(router, str, cfg)
	housingcog.Register(router, str, cfg)
	huntcog.Register(router, str, cfg)
	helpcog.Register(router, str, cfg)
	inventorycog.Register(router, str, cfg)
	itemmanagercog.Register(router, str, cfg)
	journalcog.Register(router, str, cfg)
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
	sanctuarycog.Register(router, str, cfg)
	shopcog.Register(router, str, cfg)
	lorecog.Register(router, str, cfg)
	skillscog.Register(router, str, cfg)
	startcog.Register(router, str, cfg)
	tournamentcog.Register(router, str, cfg)
	usecog.Register(router, str, cfg)
	veilcog.Register(router, str, cfg)
	onboarding.Register(router, str, cfg)

	router.Register()

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Connect) {
		slog.Info("gateway connected")
	})
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Disconnect) {
		slog.Warn("gateway disconnected")
	})
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.RateLimit) {
		slog.Warn("rate limited by discord",
			"bucket", r.Bucket,
			"retry_after", r.RetryAfter.String(),
			"url", r.URL,
		)
	})

	guildID := ""
	if cfg.GuildID != 0 {
		guildID = strconv.FormatInt(cfg.GuildID, 10)
	}
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		scope := "global"
		if guildID != "" {
			scope = "guild:" + guildID
		}
		if err := router.RegisterCommands(guildID); err != nil {
			slog.Warn("could not register slash commands", "error", err, "scope", scope)
		} else {
			slog.Info("slash commands registered", "scope", scope, "count", len(router.Commands()))
		}
		slog.Info("logged in", "username", s.State.User.Username)
	})

	if err := dg.Open(); err != nil {
		slog.Error("failed to open discord connection", "error", err)
		os.Exit(1)
	}
	defer dg.Close()

	// Watchdog: monitors DB + Discord liveness and exits the process after
	// MaxFailures consecutive unhealthy checks. restart: always in the compose
	// file brings the bot back, so a hang becomes a bounded, automatic restart
	// instead of an indefinite freeze.
	if sqlDB, sqlErr := database.DB(); sqlErr != nil {
		slog.Warn("watchdog: could not access sql.DB, DB liveness check disabled", "error", sqlErr)
	} else {
		go watchdog.New(sqlDB, dg, watchdog.Options{
			Interval:      15 * time.Second,
			DBPingTimeout: 3 * time.Second,
			ProbeTimeout:  5 * time.Second,
			MaxFailures:   2,
			HeartbeatPath: "/tmp/bot.heartbeat",
		}).Run(ctx)
	}

	slog.Info("GuacaGambleBot (Go) is online. Press CTRL-C to exit.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop
	slog.Info("shutting down")
	stopELO()
}
