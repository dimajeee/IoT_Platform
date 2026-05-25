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

	cfg := config.MustLoadHardwareGateway()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	application, err := app.NewHardwareGateway(cfg, logger)
	if err != nil {
		logger.Error("failed to initialize hardware gateway", slog.Any("error", err))
		os.Exit(1)
	}

	if err := application.Run(ctx); err != nil {
		logger.Error("hardware gateway stopped with error", slog.Any("error", err))
		os.Exit(1)
	}
}
