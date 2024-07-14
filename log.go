package main

import (
	"log/slog"
	"os"
)

var logger *slog.Logger 

func init() {
	logOptions := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	logger = slog.New(slog.NewTextHandler(os.Stderr, logOptions))
}

