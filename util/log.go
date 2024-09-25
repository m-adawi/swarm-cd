package util

import (
	"log/slog"
	"os"
	"strings"
)

var Logger *slog.Logger

func init() {
	level := getLogLevelFromEnv()
	logOptions := &slog.HandlerOptions{Level: level}
	Logger = slog.New(slog.NewTextHandler(os.Stderr, logOptions))
}

func getLogLevelFromEnv() slog.Level {
	envLogLevel := strings.ToLower(os.Getenv("LOG_LEVEL"))
	switch envLogLevel {
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
