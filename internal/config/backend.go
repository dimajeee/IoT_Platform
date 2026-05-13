package config

import (
	"log/slog"
	"os"
	"strconv"
)

type Backend struct {
	HTTPAddr     string
	MQTTBroker   string
	MQTTClientID string
	MQTTTopic    string
	PostgresDSN  string
	RedisAddr    string
	RedisPass    string
	RedisDB      int
	LogLevel     slog.Level
}

func MustLoadBackend() Backend {
	return Backend{
		HTTPAddr:     getEnv("BACKEND_HTTP_ADDR", ":8080"),
		MQTTBroker:   getEnv("BACKEND_MQTT_BROKER", "tcp://localhost:1883"),
		MQTTClientID: getEnv("BACKEND_MQTT_CLIENT_ID", "backend-service"),
		MQTTTopic:    getEnv("BACKEND_MQTT_TOPIC", "iot/sensors/+/telemetry"),
		PostgresDSN:  getEnv("BACKEND_POSTGRES_DSN", "postgres://iot:iot@localhost:5432/iot?sslmode=disable"),
		RedisAddr:    getEnv("BACKEND_REDIS_ADDR", "localhost:6379"),
		RedisPass:    getEnv("BACKEND_REDIS_PASSWORD", ""),
		RedisDB:      getEnvAsInt("BACKEND_REDIS_DB", 0),
		LogLevel:     mustParseLevel(getEnv("BACKEND_LOG_LEVEL", "info")),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}

func mustParseLevel(raw string) slog.Level {
	var level slog.Level
	if err := level.UnmarshalText([]byte(raw)); err != nil {
		return slog.LevelInfo
	}

	return level
}
