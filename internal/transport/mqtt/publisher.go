package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/dmitrijsterligov/iot-platform/internal/domain"
)

type Publisher struct {
	client pahomqtt.Client
	logger *slog.Logger
}

type PublisherConfig struct {
	BrokerURL string
	ClientID  string
	Username  string
	Password  string
}

func NewPublisher(cfg PublisherConfig, logger *slog.Logger) (*Publisher, error) {
	logger.Info(
		"connecting to mqtt broker",
		slog.String("broker", cfg.BrokerURL),
		slog.String("client_id", cfg.ClientID),
		slog.Bool("username_set", cfg.Username != ""),
	)

	options := pahomqtt.NewClientOptions().
		AddBroker(cfg.BrokerURL).
		SetClientID(cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(false).
		SetConnectTimeout(10 * time.Second)

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

func (p *Publisher) PublishRaw(ctx context.Context, topic string, payload []byte) error {
	token := p.client.Publish(topic, 1, false, payload)
	token.Wait()

	select {
	case <-ctx.Done():
		return fmt.Errorf("publish mqtt message canceled: %w", ctx.Err())
	default:
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("publish mqtt message to topic %s: %w", topic, err)
	}

	p.logger.Info("mqtt message published", slog.String("topic", topic))

	return nil
}

func (p *Publisher) Close() {
	p.client.Disconnect(250)
}

func (p *Publisher) IsConnected() bool {
	return p.client.IsConnected()
}
