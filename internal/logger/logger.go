package logger

import (
	"log/slog"
	"os"
	"strings"
)

type Options struct {
	Level     string
	Format    string // text|json
	AddSource bool
	Env       string
}

func New(opts Options) *slog.Logger {
	level := parseLevel(opts.Level)

	hopts := &slog.HandlerOptions{
		Level:     level,
		AddSource: opts.AddSource,
	}

	var h slog.Handler
	switch strings.ToLower(strings.TrimSpace(opts.Format)) {
	case "json":
		h = slog.NewJSONHandler(os.Stdout, hopts)
	default:
		h = slog.NewTextHandler(os.Stdout, hopts)
	}

	l := slog.New(h)

	env := strings.TrimSpace(opts.Env)
	if env != "" {
		l = l.With("env", env)
	}

	return l
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
