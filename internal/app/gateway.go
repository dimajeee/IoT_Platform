package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dmitrijsterligov/iot-platform/internal/config"
	"github.com/dmitrijsterligov/iot-platform/internal/device"
	mqtttransport "github.com/dmitrijsterligov/iot-platform/internal/transport/mqtt"
	"github.com/dmitrijsterligov/iot-platform/internal/usecase"
)

type Gateway struct {
	cfg    config.Gateway
	logger *slog.Logger
}

func NewGateway(cfg config.Gateway, logger *slog.Logger) (*Gateway, error) {
	return &Gateway{
		cfg:    cfg,
		logger: logger,
	}, nil
}

func (a *Gateway) Run(ctx context.Context) error {
	publisher, err := mqtttransport.NewPublisher(mqtttransport.PublisherConfig{
		BrokerURL: a.cfg.MQTTBrokerURL(),
		ClientID:  a.cfg.MQTTClientID,
		Username:  a.cfg.MQTTUsername,
		Password:  a.cfg.MQTTPassword,
	}, a.logger)
	if err != nil {
		return fmt.Errorf("create mqtt publisher: %w", err)
	}
	defer publisher.Close()

	simulator := device.NewSimulator()
	publishUsecase := usecase.NewTelemetryPublisher(publisher)

	ticker := time.NewTicker(a.cfg.PublishInterval)
	defer ticker.Stop()

	a.logger.Info("gateway started", slog.Duration("interval", a.cfg.PublishInterval))

	for {
		for _, telemetry := range simulator.Generate() {
			if err := publishUsecase.Publish(ctx, telemetry); err != nil {
				return fmt.Errorf("publish generated telemetry: %w", err)
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
