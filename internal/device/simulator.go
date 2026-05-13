package device

import (
	"math/rand"
	"time"

	"github.com/dmitrijsterligov/iot-platform/internal/domain"
)

type Simulator struct {
	sensors []sensorProfile
	random  *rand.Rand
}

type sensorProfile struct {
	id         string
	sensorType string
	unit       string
	min        float64
	max        float64
}

func NewSimulator() *Simulator {
	return &Simulator{
		sensors: []sensorProfile{
			{id: "temperature-sensor-1", sensorType: "temperature", unit: "C", min: 18, max: 30},
			{id: "humidity-sensor-1", sensorType: "humidity", unit: "%", min: 35, max: 65},
			{id: "co2-sensor-1", sensorType: "co2", unit: "ppm", min: 420, max: 1200},
		},
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *Simulator) Generate() []domain.Telemetry {
	items := make([]domain.Telemetry, 0, len(s.sensors))

	for _, sensor := range s.sensors {
		items = append(items, domain.Telemetry{
			SensorID:   sensor.id,
			SensorType: sensor.sensorType,
			Value:      sensor.min + s.random.Float64()*(sensor.max-sensor.min),
			Unit:       sensor.unit,
			RecordedAt: time.Now().UTC(),
		})
	}

	return items
}
