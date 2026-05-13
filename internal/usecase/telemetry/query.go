package telemetry

import (
	"context"
	"fmt"

	"github.com/dmitrijsterligov/iot-platform/internal/domain"
)

type QueryRepository interface {
	List(ctx context.Context, filter domain.TelemetryFilter) ([]domain.Telemetry, error)
}

type StateReader interface {
	GetLatest(ctx context.Context, sensorID string) (domain.Telemetry, error)
	ListLatest(ctx context.Context) ([]domain.Telemetry, error)
}

type QueryService struct {
	repository  QueryRepository
	stateReader StateReader
}

func NewQueryService(repository QueryRepository, stateReader StateReader) *QueryService {
	return &QueryService{
		repository:  repository,
		stateReader: stateReader,
	}
}

func (s *QueryService) List(ctx context.Context, filter domain.TelemetryFilter) ([]domain.Telemetry, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}

	if filter.Limit > 500 {
		filter.Limit = 500
	}

	items, err := s.repository.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list telemetry: %w", err)
	}

	return items, nil
}

func (s *QueryService) GetLatest(ctx context.Context, sensorID string) (domain.Telemetry, error) {
	if sensorID == "" {
		return domain.Telemetry{}, fmt.Errorf("get latest telemetry: sensor id is empty")
	}

	item, err := s.stateReader.GetLatest(ctx, sensorID)
	if err != nil {
		return domain.Telemetry{}, fmt.Errorf("get latest telemetry for sensor %s: %w", sensorID, err)
	}

	return item, nil
}

func (s *QueryService) ListLatest(ctx context.Context) ([]domain.Telemetry, error) {
	items, err := s.stateReader.ListLatest(ctx)
	if err != nil {
		return nil, fmt.Errorf("list latest telemetry: %w", err)
	}

	return items, nil
}
