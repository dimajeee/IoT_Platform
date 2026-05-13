package config

import (
	"log/slog"
	"time"
)

type Gateway struct {
	MQTTBroker      string
	MQTTClientID    string
	PublishInterval time.Duration
	LogLevel        slog.Level
}

func MustLoadGateway() Gateway {
	return Gateway{
		MQTTBroker:      getEnv("GATEWAY_MQTT_BROKER", "tcp://localhost:1883"),
		MQTTClientID:    getEnv("GATEWAY_MQTT_CLIENT_ID", "gateway-service"),
		PublishInterval: getEnvAsDuration("GATEWAY_PUBLISH_INTERVAL", 5*time.Second),
		LogLevel:        mustParseLevel(getEnv("GATEWAY_LOG_LEVEL", "info")),
	}
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
