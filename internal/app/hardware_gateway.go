package app

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"

	"github.com/dmitrijsterligov/iot-platform/internal/config"
	"github.com/dmitrijsterligov/iot-platform/internal/device"
	mqtttransport "github.com/dmitrijsterligov/iot-platform/internal/transport/mqtt"
	"github.com/dmitrijsterligov/iot-platform/internal/usecase"
)

type HardwareGateway struct {
	cfg    config.HardwareGateway
	logger *slog.Logger
}

func NewHardwareGateway(cfg config.HardwareGateway, logger *slog.Logger) (*HardwareGateway, error) {
	return &HardwareGateway{
		cfg:    cfg,
		logger: logger,
	}, nil
}

func (a *HardwareGateway) Run(ctx context.Context) error {
	if err := configureSerialPort(a.cfg.SerialPort, a.cfg.SerialBaud); err != nil {
		return fmt.Errorf("configure serial port: %w", err)
	}

	port, err := os.OpenFile(a.cfg.SerialPort, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open serial port %s: %w", a.cfg.SerialPort, err)
	}
	defer func() {
		if closeErr := port.Close(); closeErr != nil {
			a.logger.Error("failed to close serial port", slog.Any("error", closeErr))
		}
	}()

	publisher, err := mqtttransport.NewPublisher(mqtttransport.PublisherConfig{
		BrokerURL: a.cfg.MQTTBrokerURL(),
		ClientID:  a.cfg.MQTTClientID,
		Username:  a.cfg.MQTTUsername,
		Password:  a.cfg.MQTTPassword,
	}, a.logger)
	if err != nil {
		return fmt.Errorf("create mqtt publisher: %w", err)
	}
	defer publisher.Close()

	parser := device.NewSerialParser()
	publishUsecase := usecase.NewTelemetryPublisher(publisher)
	scanner := bufio.NewScanner(port)

	a.logger.Info("hardware gateway started", slog.String("serial_port", a.cfg.SerialPort), slog.Int("baud", a.cfg.SerialBaud))

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line := scanner.Text()
		items, err := parser.ParseLine(line)
		if err != nil {
			a.logger.Error("failed to parse serial line", slog.Any("error", err), slog.String("line", line))
			continue
		}

		if len(items) == 0 {
			a.logger.Debug("serial line skipped", slog.String("line", line))
			continue
		}

		for _, telemetry := range items {
			if err := publishUsecase.Publish(ctx, telemetry); err != nil {
				return fmt.Errorf("publish hardware telemetry: %w", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read serial port %s: %w", a.cfg.SerialPort, err)
	}

	return nil
}

func configureSerialPort(port string, baud int) error {
	command := exec.Command(
		"stty",
		"-F",
		port,
		strconv.Itoa(baud),
		"cs8",
		"-cstopb",
		"-parenb",
		"-ixon",
		"-ixoff",
		"raw",
	)

	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("run stty: %w: %s", err, string(output))
	}

	return nil
}
