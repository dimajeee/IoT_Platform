package mqtt

import "testing"

func TestDecodeESP32TopicValue(t *testing.T) {
	tests := []struct {
		name       string
		topic      string
		payload    string
		sensorID   string
		sensorType string
		value      float64
		unit       string
	}{
		{
			name:       "ds18b20 temperature",
			topic:      "/esp32/DS18B20/temperature",
			payload:    "25.25",
			sensorID:   "temperature-sensor-1",
			sensorType: "temperature",
			value:      25.25,
			unit:       "C",
		},
		{
			name:       "dht22 humidity",
			topic:      "/esp32/DHT22/humidity",
			payload:    "46.50",
			sensorID:   "humidity-sensor-1",
			sensorType: "humidity",
			value:      46.50,
			unit:       "%",
		},
		{
			name:       "scd30 co2",
			topic:      "/esp32/SCD30/ppm",
			payload:    "722.9",
			sensorID:   "co2-sensor-1",
			sensorType: "co2",
			value:      722.9,
			unit:       "ppm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := decodeTelemetry(tt.topic, []byte(tt.payload), "esp32-topic-value", -1)
			if err != nil {
				t.Fatalf("decodeTelemetry returned error: %v", err)
			}

			if !ok {
				t.Fatal("expected message to be decoded")
			}

			if got.SensorID != tt.sensorID {
				t.Fatalf("expected sensor id %q, got %q", tt.sensorID, got.SensorID)
			}

			if got.SensorType != tt.sensorType {
				t.Fatalf("expected sensor type %q, got %q", tt.sensorType, got.SensorType)
			}

			if got.Value != tt.value {
				t.Fatalf("expected value %v, got %v", tt.value, got.Value)
			}

			if got.Unit != tt.unit {
				t.Fatalf("expected unit %q, got %q", tt.unit, got.Unit)
			}
		})
	}
}

func TestDecodeESP32TopicValueSkipsStatus(t *testing.T) {
	_, ok, err := decodeTelemetry("/esp32/SCD30/status", []byte("0"), "esp32-topic-value", -1)
	if err != nil {
		t.Fatalf("decodeTelemetry returned error: %v", err)
	}

	if ok {
		t.Fatal("expected status topic to be skipped")
	}
}
