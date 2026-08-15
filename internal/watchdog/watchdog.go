package watchdog

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
)

// pinger is the subset of *sql.DB the watchdog needs, so tests can inject a
// fake without a real SQLite driver.
type pinger interface {
	PingContext(ctx context.Context) error
}

// Options tunes the watchdog's liveness checks.
type Options struct {
	Interval      time.Duration // how often a health check runs
	DBPingTimeout time.Duration // max time the DB ping may take
	ProbeTimeout  time.Duration // max time the Discord REST probe may take
	MaxFailures   int           // consecutive failed checks before exiting
	HeartbeatPath string        // file touched on every healthy check
}

// Watcher monitors the bot's database and Discord connectivity and terminates
// the process after MaxFailures consecutive unhealthy checks. The container's
// restart policy (restart: always) brings the bot back, turning an indefinite
// hang into a bounded, automatic recovery.
type Watcher struct {
	sqlDB pinger
	dg    *discordgo.Session
	opts  Options

	// fatal is invoked instead of os.Exit so tests can observe the decision.
	fatal func(reason string)
}

// New returns a Watcher. Zero-valued options fall back to sensible defaults.
// Pass nil for sqlDB or dg to skip that check entirely.
func New(sqlDB pinger, dg *discordgo.Session, opts Options) *Watcher {
	w := &Watcher{
		sqlDB: sqlDB,
		dg:    dg,
		opts:  opts,
		fatal: func(reason string) {
			slog.Error("watchdog: bot unhealthy, exiting for restart", "reason", reason)
			os.Exit(1)
		},
	}
	if w.opts.Interval <= 0 {
		w.opts.Interval = 15 * time.Second
	}
	if w.opts.DBPingTimeout <= 0 {
		w.opts.DBPingTimeout = 3 * time.Second
	}
	if w.opts.ProbeTimeout <= 0 {
		w.opts.ProbeTimeout = 5 * time.Second
	}
	if w.opts.MaxFailures <= 0 {
		w.opts.MaxFailures = 2
	}
	return w
}

// Run starts the monitoring loop. It returns when ctx is cancelled.
func (w *Watcher) Run(ctx context.Context) {
	failures := 0
	ticker := time.NewTicker(w.opts.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("watchdog: shutting down")
			return
		case <-ticker.C:
			if reason := w.check(ctx); reason == "" {
				failures = 0
				w.touch()
			} else {
				failures++
				slog.Warn("watchdog: unhealthy check", "reason", reason, "failures", failures)
				if failures >= w.opts.MaxFailures {
					w.fatal(reason)
					return
				}
			}
		}
	}
}

// check verifies the DB and the Discord API are reachable. It returns "" when
// healthy, otherwise a human-readable reason.
func (w *Watcher) check(ctx context.Context) string {
	if w.sqlDB != nil {
		dctx, cancel := context.WithTimeout(ctx, w.opts.DBPingTimeout)
		err := w.sqlDB.PingContext(dctx)
		cancel()
		if err != nil {
			return "database ping failed: " + err.Error()
		}
	}
	if w.dg != nil && w.dg.State != nil && w.dg.State.User != nil {
		pctx, cancel := context.WithTimeout(ctx, w.opts.ProbeTimeout)
		_, err := w.dg.User(w.dg.State.User.ID, discordgo.WithContext(pctx))
		cancel()
		if !gatewayHealthy(err) {
			return "discord api unreachable: " + err.Error()
		}
	}
	return ""
}

// gatewayHealthy reports whether a Discord REST probe result indicates a live
// API connection. Any HTTP response — success, a REST error, or even a 429 rate
// limit — proves the network path works; only network-level failures (timeouts,
// refused connections) count as unhealthy. This keeps the probe from ever being
// hurt by Discord's rate limiter.
func gatewayHealthy(err error) bool {
	if err == nil {
		return true
	}
	var rle *discordgo.RateLimitError
	var re *discordgo.RESTError
	if errors.As(err, &rle) || errors.As(err, &re) {
		return true
	}
	return false
}

// touch records the current time in the heartbeat file so the container's
// healthcheck can distinguish a healthy process from a wedged one.
func (w *Watcher) touch() {
	if w.opts.HeartbeatPath == "" {
		return
	}
	if err := os.WriteFile(w.opts.HeartbeatPath, []byte(time.Now().UTC().Format(time.RFC3339)), 0o644); err != nil {
		slog.Warn("watchdog: failed to write heartbeat", "path", w.opts.HeartbeatPath, "error", err)
	}
}
