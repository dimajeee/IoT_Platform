package device

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

var intervalPattern = regexp.MustCompile(`^[1-9][0-9]*(s|m|h)$`)

type CommandPublisher interface {
	PublishRaw(ctx context.Context, topic string, payload []byte) error
}

type CommandService struct {
	publisher     CommandPublisher
	intervalTopic string
}

func NewCommandService(publisher CommandPublisher, intervalTopic string) *CommandService {
	return &CommandService{
		publisher:     publisher,
		intervalTopic: intervalTopic,
	}
}

func (s *CommandService) SetInterval(ctx context.Context, interval string) error {
	interval = strings.TrimSpace(interval)
	if !intervalPattern.MatchString(interval) {
		return fmt.Errorf("validate interval: expected format like 2s, 3m or 4h")
	}

	if err := s.publisher.PublishRaw(ctx, s.intervalTopic, []byte(interval)); err != nil {
		return fmt.Errorf("publish interval command: %w", err)
	}

	return nil
}
