package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/dmitrijsterligov/iot-platform/internal/app"
	"github.com/dmitrijsterligov/iot-platform/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.MustLoadBackend()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	application, err := app.NewBackend(cfg, logger)
	if err != nil {
		logger.Error("failed to initialize backend", slog.Any("error", err))
		os.Exit(1)
	}

	if err := application.Run(ctx); err != nil {
		logger.Error("backend stopped with error", slog.Any("error", err))
		os.Exit(1)
	}
}
