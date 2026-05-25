package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

type Backend struct {
	HTTPAddr             string
	MQTTHost             string
	MQTTPort             int
	MQTTUsername         string
	MQTTPassword         string
	MQTTClientID         string
	InstanceID           string
	MQTTTopic            string
	MQTTDecoder          string
	MQTTSensorIDTopicPos int
	MQTTCommandTopic     string
	PostgresDSN          string
	RedisAddr            string
	RedisPass            string
	RedisDB              int
	LogLevel             slog.Level
}

func MustLoadBackend() Backend {
	return Backend{
		HTTPAddr:             getEnv("BACKEND_HTTP_ADDR", ":8080"),
		MQTTHost:             getEnv("BACKEND_MQTT_HOST", "localhost"),
		MQTTPort:             getEnvAsInt("BACKEND_MQTT_PORT", 1883),
		MQTTUsername:         getEnv("BACKEND_MQTT_USERNAME", ""),
		MQTTPassword:         getEnv("BACKEND_MQTT_PASSWORD", ""),
		MQTTClientID:         getEnv("BACKEND_MQTT_CLIENT_ID", "backend-service"),
		InstanceID:           getEnv("BACKEND_INSTANCE_ID", ""),
		MQTTTopic:            getEnv("BACKEND_MQTT_TOPIC", "/esp32/+/+"),
		MQTTDecoder:          getEnv("BACKEND_MQTT_DECODER", "esp32-topic-value"),
		MQTTSensorIDTopicPos: getEnvAsInt("BACKEND_MQTT_SENSOR_ID_TOPIC_POS", -1),
		MQTTCommandTopic:     getEnv("BACKEND_MQTT_COMMAND_TOPIC", "/esp32/interval"),
		PostgresDSN:          getEnv("BACKEND_POSTGRES_DSN", "postgres://iot:iot@localhost:5432/iot?sslmode=disable"),
		RedisAddr:            getEnv("BACKEND_REDIS_ADDR", "localhost:6379"),
		RedisPass:            getEnv("BACKEND_REDIS_PASSWORD", ""),
		RedisDB:              getEnvAsInt("BACKEND_REDIS_DB", 0),
		LogLevel:             mustParseLevel(getEnv("BACKEND_LOG_LEVEL", "info")),
	}
}

func (c Backend) MQTTBrokerURL() string {
	return fmt.Sprintf("tcp://%s:%d", c.MQTTHost, c.MQTTPort)
}

func (c Backend) RuntimeMQTTClientID() string {
	if c.InstanceID == "" {
		return c.MQTTClientID
	}

	return fmt.Sprintf("%s-%s", c.MQTTClientID, c.InstanceID)
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
