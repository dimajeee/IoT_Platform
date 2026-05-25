package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	rediscache "github.com/dmitrijsterligov/iot-platform/internal/cache/redis"
	"github.com/dmitrijsterligov/iot-platform/internal/config"
	postgresrepo "github.com/dmitrijsterligov/iot-platform/internal/repository/postgres"
	postgresstorage "github.com/dmitrijsterligov/iot-platform/internal/storage/postgres"
	redisstorage "github.com/dmitrijsterligov/iot-platform/internal/storage/redis"
	httptransport "github.com/dmitrijsterligov/iot-platform/internal/transport/http"
	mqtttransport "github.com/dmitrijsterligov/iot-platform/internal/transport/mqtt"
	deviceusecase "github.com/dmitrijsterligov/iot-platform/internal/usecase/device"
	telemetryusecase "github.com/dmitrijsterligov/iot-platform/internal/usecase/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

type Backend struct {
	cfg    config.Backend
	logger *slog.Logger
}

func NewBackend(cfg config.Backend, logger *slog.Logger) (*Backend, error) {
	return &Backend{
		cfg:    cfg,
		logger: logger,
	}, nil
}

func (a *Backend) Run(ctx context.Context) error {
	pgPool, err := connectPostgres(ctx, a.cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pgPool.Close()

	if err := postgresstorage.EnsureSchema(ctx, pgPool); err != nil {
		return fmt.Errorf("ensure postgres schema: %w", err)
	}

	redisClient, err := connectRedis(ctx, a.cfg.RedisAddr, a.cfg.RedisPass, a.cfg.RedisDB)
	if err != nil {
		return fmt.Errorf("connect redis: %w", err)
	}
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			a.logger.Error("failed to close redis client", slog.Any("error", closeErr))
		}
	}()

	repository := postgresrepo.NewTelemetryRepository(pgPool)
	cache := rediscache.NewTelemetryCache(redisClient, 24*time.Hour)
	service := telemetryusecase.NewService(repository, cache)
	queryService := telemetryusecase.NewQueryService(repository, cache)

	subscriber, err := mqtttransport.NewSubscriber(mqtttransport.SubscriberConfig{
		BrokerURL:          a.cfg.MQTTBrokerURL(),
		ClientID:           a.cfg.RuntimeMQTTClientID(),
		Username:           a.cfg.MQTTUsername,
		Password:           a.cfg.MQTTPassword,
		Topic:              a.cfg.MQTTTopic,
		Decoder:            a.cfg.MQTTDecoder,
		SensorIDTopicLevel: a.cfg.MQTTSensorIDTopicPos,
	}, a.logger, service)
	if err != nil {
		return fmt.Errorf("create mqtt subscriber: %w", err)
	}

	publisher, err := mqtttransport.NewPublisher(mqtttransport.PublisherConfig{
		BrokerURL: a.cfg.MQTTBrokerURL(),
		ClientID:  a.cfg.RuntimeMQTTClientID() + "-commands",
		Username:  a.cfg.MQTTUsername,
		Password:  a.cfg.MQTTPassword,
	}, a.logger)
	if err != nil {
		return fmt.Errorf("create mqtt command publisher: %w", err)
	}
	defer publisher.Close()

	commandService := deviceusecase.NewCommandService(publisher, a.cfg.MQTTCommandTopic)
	readiness := backendReadiness{
		pgPool:      pgPool,
		redisClient: redisClient,
		subscriber:  subscriber,
		publisher:   publisher,
	}
	httpServer := httptransport.NewServer(a.cfg.HTTPAddr, a.logger, queryService, commandService, readiness)

	a.logger.Info("backend started")

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)

	go func() {
		errCh <- subscriber.Start(runCtx)
	}()

	go func() {
		errCh <- httpServer.Start(runCtx)
	}()

	var firstErr error
	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil && firstErr == nil {
			firstErr = err
		}

		cancel()
	}

	if firstErr != nil {
		return fmt.Errorf("run backend services: %w", firstErr)
	}

	return nil
}

type backendReadiness struct {
	pgPool      *pgxpool.Pool
	redisClient *goredis.Client
	subscriber  *mqtttransport.Subscriber
	publisher   *mqtttransport.Publisher
}

func (h backendReadiness) Ready(ctx context.Context) error {
	if err := h.pgPool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres unavailable: %w", err)
	}

	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis unavailable: %w", err)
	}

	if h.subscriber == nil || !h.subscriber.IsConnected() {
		return fmt.Errorf("mqtt subscriber disconnected")
	}

	if h.publisher == nil || !h.publisher.IsConnected() {
		return fmt.Errorf("mqtt publisher disconnected")
	}

	return nil
}

func connectPostgres(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	var lastErr error
	for attempt := 1; attempt <= 10; attempt++ {
		pool, err := postgresstorage.NewPool(ctx, dsn)
		if err == nil {
			return pool, nil
		}

		lastErr = err
		if attempt < 10 {
			if err := waitForRetry(ctx, 2*time.Second); err != nil {
				return nil, fmt.Errorf("postgres connection canceled: %w", err)
			}
		}
	}

	return nil, fmt.Errorf("postgres unavailable after retries: %w", lastErr)
}

func connectRedis(ctx context.Context, addr, password string, db int) (*goredis.Client, error) {
	var lastErr error
	for attempt := 1; attempt <= 10; attempt++ {
		client, err := redisstorage.NewClient(ctx, addr, password, db)
		if err == nil {
			return client, nil
		}

		lastErr = err
		if attempt < 10 {
			if err := waitForRetry(ctx, 2*time.Second); err != nil {
				return nil, fmt.Errorf("redis connection canceled: %w", err)
			}
		}
	}

	return nil, fmt.Errorf("redis unavailable after retries: %w", lastErr)
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
