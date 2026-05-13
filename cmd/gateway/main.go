package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/dmitrijsterligov/iot-platform/internal/app"
	"github.com/dmitrijsterligov/iot-platform/internal/config"
)

func main() {
	cfg := config.MustLoadGateway()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	application, err := app.NewGateway(cfg, logger)
	if err != nil {
		logger.Error("failed to initialize gateway", slog.Any("error", err))
		os.Exit(1)
	}

	if err := application.Run(context.Background()); err != nil {
		logger.Error("gateway stopped with error", slog.Any("error", err))
		os.Exit(1)
	}
}
