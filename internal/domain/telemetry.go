package domain

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type Telemetry struct {
	SensorID   string    `json:"sensor_id"`
	SensorType string    `json:"sensor_type"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	RecordedAt time.Time `json:"recorded_at"`
}

type TelemetryFilter struct {
	SensorID   string
	SensorType string
	Limit      int
}
