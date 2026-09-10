package logger

import (
	"log/slog"
	"os"
	"strings"
)

func Setup(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	l := slog.New(h)
	slog.SetDefault(l)
	return l
}

func Done(msg string, args ...any) {
	slog.Info(msg, append([]any{slog.Bool("done", true)}, args...)...)
}
