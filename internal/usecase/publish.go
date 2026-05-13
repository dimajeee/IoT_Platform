package usecase

import (
	"context"
	"fmt"

	"github.com/dmitrijsterligov/iot-platform/internal/domain"
)

type Publisher interface {
	PublishTelemetry(ctx context.Context, telemetry domain.Telemetry) error
}

type TelemetryPublisher struct {
	publisher Publisher
}

func NewTelemetryPublisher(publisher Publisher) *TelemetryPublisher {
	return &TelemetryPublisher{publisher: publisher}
}

func (u *TelemetryPublisher) Publish(ctx context.Context, telemetry domain.Telemetry) error {
	if telemetry.SensorID == "" {
		return fmt.Errorf("validate telemetry: sensor id is empty")
	}

	if telemetry.SensorType == "" {
		return fmt.Errorf("validate telemetry: sensor type is empty")
	}

	if err := u.publisher.PublishTelemetry(ctx, telemetry); err != nil {
		return fmt.Errorf("publish telemetry: %w", err)
	}

	return nil
}
