package util

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger 

func init() {
	logOptions := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	Logger = slog.New(slog.NewTextHandler(os.Stderr, logOptions))
}

