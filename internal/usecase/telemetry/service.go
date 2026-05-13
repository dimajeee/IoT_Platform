package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/dmitrijsterligov/iot-platform/internal/domain"
)

type Repository interface {
	Save(ctx context.Context, telemetry domain.Telemetry) error
}

type Cache interface {
	SetLatest(ctx context.Context, telemetry domain.Telemetry) error
}

type Service struct {
	repository Repository
	cache      Cache
}

func NewService(repository Repository, cache Cache) *Service {
	return &Service{
		repository: repository,
		cache:      cache,
	}
}

func (s *Service) Handle(ctx context.Context, telemetry domain.Telemetry) error {
	if telemetry.SensorID == "" {
		return fmt.Errorf("validate telemetry: sensor id is empty")
	}

	if telemetry.SensorType == "" {
		return fmt.Errorf("validate telemetry: sensor type is empty")
	}

	if telemetry.RecordedAt.IsZero() {
		telemetry.RecordedAt = time.Now().UTC()
	}

	if telemetry.Unit == "" {
		telemetry.Unit = defaultUnit(telemetry.SensorType)
	}

	if err := s.repository.Save(ctx, telemetry); err != nil {
		return fmt.Errorf("save telemetry: %w", err)
	}

	if err := s.cache.SetLatest(ctx, telemetry); err != nil {
		return fmt.Errorf("cache latest telemetry: %w", err)
	}

	return nil
}

func defaultUnit(sensorType string) string {
	switch sensorType {
	case "temperature":
		return "C"
	case "humidity":
		return "%"
	case "co2":
		return "ppm"
	default:
		return "unknown"
	}
}
