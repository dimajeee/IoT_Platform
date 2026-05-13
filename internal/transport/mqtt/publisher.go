package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/dmitrijsterligov/iot-platform/internal/domain"
)

type Publisher struct {
	client pahomqtt.Client
	logger *slog.Logger
}

func NewPublisher(broker, clientID string, logger *slog.Logger) (*Publisher, error) {
	options := pahomqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetAutoReconnect(true).
		SetConnectRetry(true)

	client := pahomqtt.NewClient(options)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("connect to mqtt broker: %w", token.Error())
	}

	return &Publisher{
		client: client,
		logger: logger,
	}, nil
}

func (p *Publisher) PublishTelemetry(ctx context.Context, telemetry domain.Telemetry) error {
	payload, err := json.Marshal(telemetry)
	if err != nil {
		return fmt.Errorf("marshal telemetry payload: %w", err)
	}

	topic := fmt.Sprintf("iot/sensors/%s/telemetry", telemetry.SensorID)
	token := p.client.Publish(topic, 1, false, payload)
	token.Wait()

	select {
	case <-ctx.Done():
		return fmt.Errorf("publish telemetry canceled: %w", ctx.Err())
	default:
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("publish telemetry to topic %s: %w", topic, err)
	}

	p.logger.Info("telemetry published", slog.String("sensor_id", telemetry.SensorID), slog.String("topic", topic))

	return nil
}

func (p *Publisher) Close() {
	p.client.Disconnect(250)
}
