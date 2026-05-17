package config

import (
	"fmt"
	"log/slog"
	"time"
)

type Gateway struct {
	MQTTHost        string
	MQTTPort        int
	MQTTUsername    string
	MQTTPassword    string
	MQTTClientID    string
	PublishInterval time.Duration
	LogLevel        slog.Level
}

type HardwareGateway struct {
	MQTTHost     string
	MQTTPort     int
	MQTTUsername string
	MQTTPassword string
	MQTTClientID string
	SerialPort   string
	SerialBaud   int
	LogLevel     slog.Level
}

func MustLoadGateway() Gateway {
	return Gateway{
		MQTTHost:        getEnv("GATEWAY_MQTT_HOST", "localhost"),
		MQTTPort:        getEnvAsInt("GATEWAY_MQTT_PORT", 1883),
		MQTTUsername:    getEnv("GATEWAY_MQTT_USERNAME", ""),
		MQTTPassword:    getEnv("GATEWAY_MQTT_PASSWORD", ""),
		MQTTClientID:    getEnv("GATEWAY_MQTT_CLIENT_ID", "gateway-service"),
		PublishInterval: getEnvAsDuration("GATEWAY_PUBLISH_INTERVAL", 5*time.Second),
		LogLevel:        mustParseLevel(getEnv("GATEWAY_LOG_LEVEL", "info")),
	}
}

func MustLoadHardwareGateway() HardwareGateway {
	return HardwareGateway{
		MQTTHost:     getEnv("HARDWARE_GATEWAY_MQTT_HOST", "localhost"),
		MQTTPort:     getEnvAsInt("HARDWARE_GATEWAY_MQTT_PORT", 1883),
		MQTTUsername: getEnv("HARDWARE_GATEWAY_MQTT_USERNAME", ""),
		MQTTPassword: getEnv("HARDWARE_GATEWAY_MQTT_PASSWORD", ""),
		MQTTClientID: getEnv("HARDWARE_GATEWAY_MQTT_CLIENT_ID", "hardware-gateway-service"),
		SerialPort:   getEnv("HARDWARE_GATEWAY_SERIAL_PORT", "/dev/ttyUSB0"),
		SerialBaud:   getEnvAsInt("HARDWARE_GATEWAY_SERIAL_BAUD", 115200),
		LogLevel:     mustParseLevel(getEnv("HARDWARE_GATEWAY_LOG_LEVEL", "info")),
	}
}

func (c Gateway) MQTTBrokerURL() string {
	return fmt.Sprintf("tcp://%s:%d", c.MQTTHost, c.MQTTPort)
}

func (c HardwareGateway) MQTTBrokerURL() string {
	return fmt.Sprintf("tcp://%s:%d", c.MQTTHost, c.MQTTPort)
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}

	return value
}
