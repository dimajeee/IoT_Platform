package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/dmitrijsterligov/iot-platform/internal/domain"
	goredis "github.com/redis/go-redis/v9"
)

type TelemetryCache struct {
	client *goredis.Client
	ttl    time.Duration
}

const latestSensorIndexKey = "sensors:latest:index"

func NewTelemetryCache(client *goredis.Client, ttl time.Duration) *TelemetryCache {
	return &TelemetryCache{
		client: client,
		ttl:    ttl,
	}
}

func (c *TelemetryCache) SetLatest(ctx context.Context, telemetry domain.Telemetry) error {
	payload, err := json.Marshal(telemetry)
	if err != nil {
		return fmt.Errorf("marshal telemetry for cache: %w", err)
	}

	key := latestStateKey(telemetry.SensorID)
	if err := c.client.Set(ctx, key, payload, c.ttl).Err(); err != nil {
		return fmt.Errorf("set redis key %s: %w", key, err)
	}

	if err := c.client.SAdd(ctx, latestSensorIndexKey, telemetry.SensorID).Err(); err != nil {
		return fmt.Errorf("add sensor id %s to latest index: %w", telemetry.SensorID, err)
	}

	return nil
}

func (c *TelemetryCache) GetLatest(ctx context.Context, sensorID string) (domain.Telemetry, error) {
	payload, err := c.client.Get(ctx, latestStateKey(sensorID)).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return domain.Telemetry{}, fmt.Errorf("latest telemetry is missing for sensor %s: %w", sensorID, domain.ErrNotFound)
		}

		return domain.Telemetry{}, fmt.Errorf("get latest telemetry for sensor %s: %w", sensorID, err)
	}

	var item domain.Telemetry
	if err := json.Unmarshal(payload, &item); err != nil {
		return domain.Telemetry{}, fmt.Errorf("unmarshal latest telemetry for sensor %s: %w", sensorID, err)
	}

	return item, nil
}

func (c *TelemetryCache) ListLatest(ctx context.Context) ([]domain.Telemetry, error) {
	sensorIDs, err := c.client.SMembers(ctx, latestSensorIndexKey).Result()
	if err != nil {
		return nil, fmt.Errorf("list latest telemetry sensor ids: %w", err)
	}

	slices.Sort(sensorIDs)

	items := make([]domain.Telemetry, 0, len(sensorIDs))
	for _, sensorID := range sensorIDs {
		item, err := c.GetLatest(ctx, sensorID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}

			return nil, fmt.Errorf("get latest telemetry for sensor %s while listing: %w", sensorID, err)
		}

		items = append(items, item)
	}

	return items, nil
}

func latestStateKey(sensorID string) string {
	return fmt.Sprintf("sensor:%s:latest_state", sensorID)
}
