package device

import "testing"

func TestSerialParserParseJSONLine(t *testing.T) {
	parser := NewSerialParser()

	items, err := parser.ParseLine(`{"sensor_type":"co2","value":658,"unit":"ppm"}`)
	if err != nil {
		t.Fatalf("ParseLine returned error: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if items[0].SensorID != "co2-sensor-1" {
		t.Fatalf("expected default co2 sensor id, got %q", items[0].SensorID)
	}

	if items[0].Value != 658 {
		t.Fatalf("expected co2 value 658, got %v", items[0].Value)
	}
}

func TestSerialParserParseTextLines(t *testing.T) {
	parser := NewSerialParser()

	tests := []struct {
		name       string
		line       string
		sensorID   string
		sensorType string
		value      float64
	}{
		{
			name:       "temperature",
			line:       "DS18B20 Temperature: 24.6 C",
			sensorID:   "temperature-sensor-1",
			sensorType: "temperature",
			value:      24.6,
		},
		{
			name:       "humidity",
			line:       "DHT22 Humidity: 41.2 %",
			sensorID:   "humidity-sensor-1",
			sensorType: "humidity",
			value:      41.2,
		},
		{
			name:       "co2",
			line:       "SCD30 CO2: 658 ppm",
			sensorID:   "co2-sensor-1",
			sensorType: "co2",
			value:      658,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := parser.ParseLine(tt.line)
			if err != nil {
				t.Fatalf("ParseLine returned error: %v", err)
			}

			if len(items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(items))
			}

			if items[0].SensorID != tt.sensorID {
				t.Fatalf("expected sensor id %q, got %q", tt.sensorID, items[0].SensorID)
			}

			if items[0].SensorType != tt.sensorType {
				t.Fatalf("expected sensor type %q, got %q", tt.sensorType, items[0].SensorType)
			}

			if items[0].Value != tt.value {
				t.Fatalf("expected value %v, got %v", tt.value, items[0].Value)
			}
		})
	}
}
