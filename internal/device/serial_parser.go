package device

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dmitrijsterligov/iot-platform/internal/domain"
)

var measurementPatterns = map[string][]*regexp.Regexp{
	"temperature": {
		regexp.MustCompile(`(?:ds18b20|temperature|temp)\D*([-+]?\d+(?:[.,]\d+)?)`),
		regexp.MustCompile(`([-+]?\d+(?:[.,]\d+)?)\s*(?:c|°c)`),
	},
	"humidity": {
		regexp.MustCompile(`(?:humidity|hum|dht22)\D*([-+]?\d+(?:[.,]\d+)?)`),
		regexp.MustCompile(`([-+]?\d+(?:[.,]\d+)?)\s*%`),
	},
	"co2": {
		regexp.MustCompile(`(?:co2|co₂)\D*([-+]?\d+(?:[.,]\d+)?)`),
		regexp.MustCompile(`([-+]?\d+(?:[.,]\d+)?)\s*ppm`),
	},
}

type SerialParser struct{}

func NewSerialParser() *SerialParser {
	return &SerialParser{}
}

func (p *SerialParser) ParseLine(line string) ([]domain.Telemetry, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	if items, ok, err := parseJSONTelemetry(line); ok || err != nil {
		return items, err
	}

	items := make([]domain.Telemetry, 0, 3)
	if item, ok, err := parseTextMeasurement(line, "temperature"); err != nil {
		return nil, err
	} else if ok {
		items = append(items, item)
	}

	if item, ok, err := parseTextMeasurement(line, "humidity"); err != nil {
		return nil, err
	} else if ok {
		items = append(items, item)
	}

	if item, ok, err := parseTextMeasurement(line, "co2"); err != nil {
		return nil, err
	} else if ok {
		items = append(items, item)
	}

	return items, nil
}

func parseJSONTelemetry(line string) ([]domain.Telemetry, bool, error) {
	if !strings.HasPrefix(line, "{") && !strings.HasPrefix(line, "[") {
		return nil, false, nil
	}

	var item domain.Telemetry
	if err := json.Unmarshal([]byte(line), &item); err == nil && (item.SensorID != "" || item.SensorType != "") {
		return []domain.Telemetry{normalizeTelemetry(item)}, true, nil
	}

	var items []domain.Telemetry
	if err := json.Unmarshal([]byte(line), &items); err != nil {
		return nil, true, fmt.Errorf("parse json telemetry line: %w", err)
	}

	for index := range items {
		items[index] = normalizeTelemetry(items[index])
	}

	return items, true, nil
}

func parseTextMeasurement(line string, sensorType string) (domain.Telemetry, bool, error) {
	lower := strings.ToLower(line)
	if !lineMentionsSensor(lower, sensorType) {
		return domain.Telemetry{}, false, nil
	}

	if shouldSkipAuxiliaryMeasurement(lower, sensorType) {
		return domain.Telemetry{}, false, nil
	}

	value, ok, err := firstNumberAfterKeyword(lower, sensorType)
	if err != nil {
		return domain.Telemetry{}, false, err
	}

	if !ok {
		return domain.Telemetry{}, false, nil
	}

	return domain.Telemetry{
		SensorID:   defaultSensorID(sensorType),
		SensorType: sensorType,
		Value:      value,
		Unit:       defaultUnit(sensorType),
		RecordedAt: time.Now().UTC(),
	}, true, nil
}

func firstNumberAfterKeyword(line string, sensorType string) (float64, bool, error) {
	for _, pattern := range measurementPatterns[sensorType] {
		matches := pattern.FindStringSubmatch(line)
		if len(matches) < 2 || matches[1] == "" {
			continue
		}

		value, err := strconv.ParseFloat(strings.ReplaceAll(matches[1], ",", "."), 64)
		if err != nil {
			return 0, false, fmt.Errorf("parse %s value %q: %w", sensorType, matches[1], err)
		}

		return value, true, nil
	}

	return 0, false, nil
}

func lineMentionsSensor(line string, sensorType string) bool {
	for _, keyword := range sensorKeywords(sensorType) {
		if strings.Contains(line, keyword) {
			return true
		}
	}

	return false
}

func shouldSkipAuxiliaryMeasurement(line string, sensorType string) bool {
	switch sensorType {
	case "temperature":
		return !strings.Contains(line, "ds18b20") && (strings.Contains(line, "dht22") || strings.Contains(line, "scd30"))
	case "humidity":
		return strings.Contains(line, "scd30") && !strings.Contains(line, "dht22")
	default:
		return false
	}
}

func sensorKeywords(sensorType string) []string {
	switch sensorType {
	case "temperature":
		return []string{"temperature", "temp", "ds18b20"}
	case "humidity":
		return []string{"humidity", "hum", "dht22"}
	case "co2":
		return []string{"co2", "co₂", "scd30", "ppm"}
	default:
		return nil
	}
}

func normalizeTelemetry(item domain.Telemetry) domain.Telemetry {
	if item.RecordedAt.IsZero() {
		item.RecordedAt = time.Now().UTC()
	}

	if item.SensorID == "" && item.SensorType != "" {
		item.SensorID = defaultSensorID(item.SensorType)
	}

	if item.Unit == "" && item.SensorType != "" {
		item.Unit = defaultUnit(item.SensorType)
	}

	return item
}

func defaultSensorID(sensorType string) string {
	switch sensorType {
	case "temperature":
		return "temperature-sensor-1"
	case "humidity":
		return "humidity-sensor-1"
	case "co2":
		return "co2-sensor-1"
	default:
		return sensorType + "-sensor-1"
	}
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
