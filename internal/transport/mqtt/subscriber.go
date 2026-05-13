package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/dmitrijsterligov/iot-platform/internal/domain"
)

type TelemetryHandler interface {
	Handle(ctx context.Context, telemetry domain.Telemetry) error
}

type Subscriber struct {
	client  pahomqtt.Client
	logger  *slog.Logger
	topic   string
	handler TelemetryHandler
}

func NewSubscriber(broker, clientID, topic string, logger *slog.Logger, handler TelemetryHandler) (*Subscriber, error) {
	options := pahomqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetCleanSession(false)

	client := pahomqtt.NewClient(options)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("connect to mqtt broker: %w", token.Error())
	}

	return &Subscriber{
		client:  client,
		logger:  logger,
		topic:   topic,
		handler: handler,
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

func (s *Subscriber) onMessage(ctx context.Context) pahomqtt.MessageHandler {
	return func(_ pahomqtt.Client, message pahomqtt.Message) {
		telemetry, err := decodeTelemetry(message.Topic(), message.Payload())
		if err != nil {
			s.logger.Error("failed to decode telemetry", slog.Any("error", err), slog.String("topic", message.Topic()))
			return
		}

		if err := s.handler.Handle(ctx, telemetry); err != nil {
			s.logger.Error("failed to handle telemetry", slog.Any("error", err), slog.String("sensor_id", telemetry.SensorID))
			return
		}

		s.logger.Info("telemetry processed", slog.String("sensor_id", telemetry.SensorID))
	}
}

func decodeTelemetry(topic string, payload []byte) (domain.Telemetry, error) {
	sensorID, err := sensorIDFromTopic(topic)
	if err != nil {
		return domain.Telemetry{}, fmt.Errorf("extract sensor id from topic: %w", err)
	}

	var telemetry domain.Telemetry
	if err := json.Unmarshal(payload, &telemetry); err != nil {
		return domain.Telemetry{}, fmt.Errorf("unmarshal telemetry payload: %w", err)
	}

	telemetry.SensorID = sensorID
	return telemetry, nil
}

func sensorIDFromTopic(topic string) (string, error) {
	parts := strings.Split(topic, "/")
	if len(parts) != 4 {
		return "", fmt.Errorf("unexpected topic format: %s", topic)
	}

	if parts[0] != "iot" || parts[1] != "sensors" || parts[3] != "telemetry" {
		return "", fmt.Errorf("unexpected topic structure: %s", topic)
	}

	if parts[2] == "" {
		return "", fmt.Errorf("sensor id is empty in topic: %s", topic)
	}

	return parts[2], nil
}
