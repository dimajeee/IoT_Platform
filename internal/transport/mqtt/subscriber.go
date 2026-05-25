package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/dmitrijsterligov/iot-platform/internal/domain"
)

type TelemetryHandler interface {
	Handle(ctx context.Context, telemetry domain.Telemetry) error
}

type SubscriberConfig struct {
	BrokerURL          string
	ClientID           string
	Username           string
	Password           string
	Topic              string
	Decoder            string
	SensorIDTopicLevel int
}

type Subscriber struct {
	client             pahomqtt.Client
	logger             *slog.Logger
	topic              string
	decoder            string
	sensorIDTopicLevel int
	handler            TelemetryHandler
}

func NewSubscriber(cfg SubscriberConfig, logger *slog.Logger, handler TelemetryHandler) (*Subscriber, error) {
	logger.Info(
		"connecting to mqtt broker",
		slog.String("broker", cfg.BrokerURL),
		slog.String("client_id", cfg.ClientID),
		slog.String("topic", cfg.Topic),
		slog.String("decoder", cfg.Decoder),
		slog.Bool("username_set", cfg.Username != ""),
	)

	options := pahomqtt.NewClientOptions().
		AddBroker(cfg.BrokerURL).
		SetClientID(cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(false).
		SetConnectTimeout(10 * time.Second).
		SetCleanSession(false)

	if cfg.Username != "" {
		options.SetUsername(cfg.Username)
		options.SetPassword(cfg.Password)
	}

	client := pahomqtt.NewClient(options)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("connect to mqtt broker: %w", token.Error())
	}

	logger.Info(
		"connected to mqtt broker",
		slog.String("broker", cfg.BrokerURL),
		slog.String("client_id", cfg.ClientID),
	)

	return &Subscriber{
		client:             client,
		logger:             logger,
		topic:              cfg.Topic,
		decoder:            cfg.Decoder,
		sensorIDTopicLevel: cfg.SensorIDTopicLevel,
		handler:            handler,
	}, nil
}

func (s *Subscriber) Start(ctx context.Context) error {
	if token := s.client.Subscribe(s.topic, 1, s.onMessage(ctx)); token.Wait() && token.Error() != nil {
		return fmt.Errorf("subscribe to topic %s: %w", s.topic, token.Error())
	}

	s.logger.Info("mqtt subscriber started", slog.String("topic", s.topic))

	<-ctx.Done()

	s.client.Disconnect(250)
	return nil
}

func (s *Subscriber) IsConnected() bool {
	return s.client.IsConnected()
}

func (s *Subscriber) onMessage(ctx context.Context) pahomqtt.MessageHandler {
	return func(_ pahomqtt.Client, message pahomqtt.Message) {
		telemetry, ok, err := decodeTelemetry(message.Topic(), message.Payload(), s.decoder, s.sensorIDTopicLevel)
		if err != nil {
			s.logger.Error("failed to decode telemetry", slog.Any("error", err), slog.String("topic", message.Topic()))
			return
		}

		if !ok {
			s.logger.Debug("mqtt message skipped", slog.String("topic", message.Topic()))
			return
		}

		if err := s.handler.Handle(ctx, telemetry); err != nil {
			s.logger.Error("failed to handle telemetry", slog.Any("error", err), slog.String("sensor_id", telemetry.SensorID))
			return
		}

		s.logger.Info("telemetry processed", slog.String("sensor_id", telemetry.SensorID))
	}
}

func decodeTelemetry(topic string, payload []byte, decoder string, sensorIDTopicLevel int) (domain.Telemetry, bool, error) {
	switch decoder {
	case "", "json":
		telemetry, err := decodeJSONTelemetry(topic, payload, sensorIDTopicLevel)
		if err != nil {
			return domain.Telemetry{}, false, err
		}

		return telemetry, true, nil
	case "esp32-topic-value":
		return decodeESP32TopicValue(topic, payload)
	default:
		return domain.Telemetry{}, false, fmt.Errorf("unsupported mqtt decoder: %s", decoder)
	}
}

func decodeJSONTelemetry(topic string, payload []byte, sensorIDTopicLevel int) (domain.Telemetry, error) {
	var telemetry domain.Telemetry
	if err := json.Unmarshal(payload, &telemetry); err != nil {
		return domain.Telemetry{}, fmt.Errorf("unmarshal telemetry payload: %w", err)
	}

	if sensorIDTopicLevel >= 0 {
		sensorID, err := sensorIDFromTopic(topic, sensorIDTopicLevel)
		if err != nil {
			return domain.Telemetry{}, fmt.Errorf("extract sensor id from topic: %w", err)
		}

		telemetry.SensorID = sensorID
	}

	if telemetry.SensorID == "" {
		return domain.Telemetry{}, fmt.Errorf("sensor id is empty in payload and topic")
	}

	return telemetry, nil
}

func decodeESP32TopicValue(topic string, payload []byte) (domain.Telemetry, bool, error) {
	parts := strings.Split(strings.Trim(topic, "/"), "/")
	if len(parts) != 3 {
		return domain.Telemetry{}, false, fmt.Errorf("unexpected esp32 topic format: %s", topic)
	}

	if parts[0] != "esp32" {
		return domain.Telemetry{}, false, fmt.Errorf("unexpected esp32 topic root: %s", topic)
	}

	device := strings.ToUpper(parts[1])
	metric := strings.ToLower(parts[2])
	if metric == "status" {
		return domain.Telemetry{}, false, nil
	}

	sensorID, sensorType, unit, ok := esp32SensorMapping(device, metric)
	if !ok {
		return domain.Telemetry{}, false, fmt.Errorf("unsupported esp32 measurement topic: %s", topic)
	}

	rawValue := strings.TrimSpace(string(payload))
	value, err := strconv.ParseFloat(strings.ReplaceAll(rawValue, ",", "."), 64)
	if err != nil {
		return domain.Telemetry{}, false, fmt.Errorf("parse esp32 payload value %q: %w", rawValue, err)
	}

	return domain.Telemetry{
		SensorID:   sensorID,
		SensorType: sensorType,
		Value:      value,
		Unit:       unit,
		RecordedAt: time.Now().UTC(),
	}, true, nil
}

func esp32SensorMapping(device string, metric string) (string, string, string, bool) {
	switch {
	case device == "DS18B20" && metric == "temperature":
		return "temperature-sensor-1", "temperature", "C", true
	case device == "DHT22" && metric == "humidity":
		return "humidity-sensor-1", "humidity", "%", true
	case device == "SCD30" && metric == "ppm":
		return "co2-sensor-1", "co2", "ppm", true
	default:
		return "", "", "", false
	}
}

func sensorIDFromTopic(topic string, sensorIDTopicLevel int) (string, error) {
	parts := strings.Split(topic, "/")
	if sensorIDTopicLevel >= len(parts) {
		return "", fmt.Errorf("topic %s does not have segment index %d", topic, sensorIDTopicLevel)
	}

	if parts[sensorIDTopicLevel] == "" {
		return "", fmt.Errorf("sensor id is empty in topic: %s", topic)
	}

	return parts[sensorIDTopicLevel], nil
}
