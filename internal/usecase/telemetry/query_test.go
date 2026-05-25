package telemetry

import (
	"context"
	"testing"

	"github.com/dmitrijsterligov/iot-platform/internal/domain"
)

type recordingQueryRepository struct {
	listCalled bool
	filter     domain.TelemetryFilter
	items      []domain.Telemetry
}

func (r *recordingQueryRepository) List(_ context.Context, filter domain.TelemetryFilter) ([]domain.Telemetry, error) {
	r.listCalled = true
	r.filter = filter
	return r.items, nil
}

type recordingStateReader struct {
	getCalled  bool
	listCalled bool
	sensorID   string
	item       domain.Telemetry
	items      []domain.Telemetry
}

func (r *recordingStateReader) GetLatest(_ context.Context, sensorID string) (domain.Telemetry, error) {
	r.getCalled = true
	r.sensorID = sensorID
	return r.item, nil
}

func (r *recordingStateReader) ListLatest(_ context.Context) ([]domain.Telemetry, error) {
	r.listCalled = true
	return r.items, nil
}

func TestQueryServiceGetLatestReadsStateReaderOnly(t *testing.T) {
	repository := &recordingQueryRepository{}
	stateReader := &recordingStateReader{
		item: domain.Telemetry{SensorID: "co2-sensor-1", SensorType: "co2", Value: 700},
	}
	service := NewQueryService(repository, stateReader)

	item, err := service.GetLatest(context.Background(), "co2-sensor-1")
	if err != nil {
		t.Fatalf("GetLatest() error = %v", err)
	}

	if !stateReader.getCalled {
		t.Fatal("GetLatest() did not call state reader")
	}

	if repository.listCalled {
		t.Fatal("GetLatest() called repository, want cache/state reader only")
	}

	if item.SensorID != "co2-sensor-1" {
		t.Fatalf("item.SensorID = %q, want %q", item.SensorID, "co2-sensor-1")
	}
}

func TestQueryServiceListLatestReadsStateReaderOnly(t *testing.T) {
	repository := &recordingQueryRepository{}
	stateReader := &recordingStateReader{
		items: []domain.Telemetry{{SensorID: "temperature-sensor-1", SensorType: "temperature", Value: 24.5}},
	}
	service := NewQueryService(repository, stateReader)

	items, err := service.ListLatest(context.Background())
	if err != nil {
		t.Fatalf("ListLatest() error = %v", err)
	}

	if !stateReader.listCalled {
		t.Fatal("ListLatest() did not call state reader")
	}

	if repository.listCalled {
		t.Fatal("ListLatest() called repository, want cache/state reader only")
	}

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
}

func TestQueryServiceListUsesRepository(t *testing.T) {
	repository := &recordingQueryRepository{}
	stateReader := &recordingStateReader{}
	service := NewQueryService(repository, stateReader)

	_, err := service.List(context.Background(), domain.TelemetryFilter{Limit: -1})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if !repository.listCalled {
		t.Fatal("List() did not call repository")
	}

	if repository.filter.Limit != 50 {
		t.Fatalf("repository filter limit = %d, want 50", repository.filter.Limit)
	}

	if stateReader.getCalled || stateReader.listCalled {
		t.Fatal("List() called state reader, want repository only")
	}
}
