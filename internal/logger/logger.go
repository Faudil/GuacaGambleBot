package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"guacagamblebot/internal/config"
)

var log *slog.Logger

func Init(cfg *config.Config) *slog.Logger {
	var w io.Writer = os.Stderr
	if cfg.LogFile != "" {
		f, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			slog.Error("failed to open log file, falling back to stderr", "path", cfg.LogFile, "error", err)
		} else {
			w = f
		}
	}

	level := parseLevel(cfg.LogLevel)
	addSource := cfg.LogAddSource

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: level, AddSource: addSource}

	switch strings.ToLower(cfg.LogFormat) {
	case "json":
		handler = slog.NewJSONHandler(w, opts)
	default:
		handler = slog.NewTextHandler(w, opts)
	}

	log = slog.New(handler)
	slog.SetDefault(log)
	return log
}

func Log() *slog.Logger {
	if log != nil {
		return log
	}
	return slog.Default()
}

func With(attrs ...any) *slog.Logger {
	return Log().With(attrs...)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
