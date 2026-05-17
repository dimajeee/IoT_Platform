package device

import (
	"context"
	"testing"
)

type fakeCommandPublisher struct {
	topic   string
	payload string
}

func (p *fakeCommandPublisher) PublishRaw(_ context.Context, topic string, payload []byte) error {
	p.topic = topic
	p.payload = string(payload)
	return nil
}

func TestCommandServiceSetInterval(t *testing.T) {
	publisher := &fakeCommandPublisher{}
	service := NewCommandService(publisher, "/esp32/interval")

	if err := service.SetInterval(context.Background(), " 2s "); err != nil {
		t.Fatalf("SetInterval returned error: %v", err)
	}

	if publisher.topic != "/esp32/interval" {
		t.Fatalf("expected topic /esp32/interval, got %q", publisher.topic)
	}

	if publisher.payload != "2s" {
		t.Fatalf("expected payload 2s, got %q", publisher.payload)
	}
}

func TestCommandServiceSetIntervalRejectsInvalidFormat(t *testing.T) {
	publisher := &fakeCommandPublisher{}
	service := NewCommandService(publisher, "/esp32/interval")

	if err := service.SetInterval(context.Background(), "2seconds"); err == nil {
		t.Fatal("expected validation error")
	}

	if publisher.payload != "" {
		t.Fatalf("expected no publish, got payload %q", publisher.payload)
	}
}
