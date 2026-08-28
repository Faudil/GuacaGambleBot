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
	petsvc "guacagamblebot/internal/service/pets"
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
	forgecog "guacagamblebot/internal/cogs/forge"
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

// cogRegistration is one help category and the cogs filed under it.
type cogRegistration struct {
	category string
	cogs     []func(*interaction.Router, *store.Store, *config.Config)
}

// cogGroups is the bot's whole command taxonomy in one table. /help is
// generated from it, so a cog added here appears in /help automatically, and a
// cog left out of it is reported by the startup check and by the tests in this
// package. Tests build their router from this same table, so what they inspect
// is exactly what the running bot registers.
func cogGroups() []cogRegistration {
	return []cogRegistration{
		{helpcog.CatStart, []func(*interaction.Router, *store.Store, *config.Config){
			startcog.Register,
			onboarding.Register,
			helpcog.Register,
		}},
		{helpcog.CatEconomy, []func(*interaction.Router, *store.Store, *config.Config){
			economycog.Register,
			bankcog.Register,
			loancog.Register,
			jobscog.Register,
			shopcog.Register,
			marketcog.Register,
			itemmanagercog.Register,
			inventorycog.Register,
			usecog.Register,
			craftingcog.Register,
			forgecog.Register,
		}},
		{helpcog.CatCasino, []func(*interaction.Router, *store.Store, *config.Config){
			casinocog.Register,
			blackjackcog.Register,
			roulettecog.Register,
			lottocog.Register,
			bettingcog.Register,
			duelcog.Register,
		}},
		{helpcog.CatRPG, []func(*interaction.Router, *store.Store, *config.Config){
			charactercog.Register,
			skillscog.Register,
			questscog.Register,
			journalcog.Register,
			achievementscog.Register,
			bosscog.Register,
			delvecog.Register,
			veilcog.Register,
			tournamentcog.Register,
		}},
		{helpcog.CatActivities, []func(*interaction.Router, *store.Store, *config.Config){
			fishingcog.Register,
			miningcog.Register,
			farmcog.Register,
			huntcog.Register,
			archeologycog.Register,
			expeditioncog.Register,
			housingcog.Register,
		}},
		{helpcog.CatPets, []func(*interaction.Router, *store.Store, *config.Config){
			petscog.Register,
			sanctuarycog.Register,
		}},
		{helpcog.CatWorld, []func(*interaction.Router, *store.Store, *config.Config){
			leadercog.Register,
			communitycog.Register,
			npcscog.Register,
			lorecog.Register,
			criminalitycog.Register,
		}},
		{helpcog.CatAdmin, []func(*interaction.Router, *store.Store, *config.Config){
			admincog.Register,
		}},
	}
}

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
	// Auto-migrate overflow roster >10 to sanctuary (allows overflow even if sanctuary max exceeded)
	func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Warn("overflow migration panic", "error", r)
			}
		}()
		if m, err := petsvc.New(str, cfg).MigrateOverflowToSanctuary(); err == nil && m > 0 {
			slog.Info("overflow migration: moved pets to sanctuary", "moved", m)
		}
	}()
	go elosimulation.Run(ctx, str)
	go elosimulation.RunWeeklyReset(ctx, str, dg)
	router := interaction.NewRouter(bot, str)

	for _, group := range cogGroups() {
		router.Categorize(group.category, func() {
			for _, register := range group.cogs {
				register(router, str, cfg)
			}
		})
	}

	for _, cmd := range router.CommandList() {
		if cmd.Category == "" {
			slog.Warn("command missing help category", "command", cmd.Name)
		}
	}

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
