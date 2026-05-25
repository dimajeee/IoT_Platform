package telemetry

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dmitrijsterligov/iot-platform/internal/domain"
)

type recordingRepository struct {
	calls *[]string
	err   error
	item  domain.Telemetry
}

func (r *recordingRepository) Save(_ context.Context, telemetry domain.Telemetry) error {
	*r.calls = append(*r.calls, "repository.Save")
	r.item = telemetry
	return r.err
}

type recordingCache struct {
	calls *[]string
	err   error
	item  domain.Telemetry
}

func (c *recordingCache) SetLatest(_ context.Context, telemetry domain.Telemetry) error {
	*c.calls = append(*c.calls, "cache.SetLatest")
	c.item = telemetry
	return c.err
}

func TestServiceHandleSavesRepositoryBeforeCache(t *testing.T) {
	calls := make([]string, 0, 2)
	repository := &recordingRepository{calls: &calls}
	cache := &recordingCache{calls: &calls}
	service := NewService(repository, cache)

	err := service.Handle(context.Background(), domain.Telemetry{
		SensorID:   "temperature-sensor-1",
		SensorType: "temperature",
		Value:      24.5,
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	if got, want := strings.Join(calls, ","), "repository.Save,cache.SetLatest"; got != want {
		t.Fatalf("call order = %s, want %s", got, want)
	}

	if repository.item.RecordedAt.IsZero() {
		t.Fatal("repository received zero RecordedAt")
	}

	if cache.item.RecordedAt.IsZero() {
		t.Fatal("cache received zero RecordedAt")
	}

	if cache.item.Unit != "C" {
		t.Fatalf("cache item unit = %q, want %q", cache.item.Unit, "C")
	}
}

func TestServiceHandleDoesNotCacheWhenRepositoryFails(t *testing.T) {
	calls := make([]string, 0, 2)
	repository := &recordingRepository{
		calls: &calls,
		err:   errors.New("postgres failed"),
	}
	cache := &recordingCache{calls: &calls}
	service := NewService(repository, cache)

	err := service.Handle(context.Background(), domain.Telemetry{
		SensorID:   "humidity-sensor-1",
		SensorType: "humidity",
		Value:      44.1,
	})
	if err == nil {
		t.Fatal("Handle() error = nil, want repository error")
	}

	if got, want := strings.Join(calls, ","), "repository.Save"; got != want {
		t.Fatalf("call order = %s, want %s", got, want)
	}
}
