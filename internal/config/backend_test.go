package config

import "testing"

func TestRuntimeMQTTClientIDUsesBaseWhenInstanceIDEmpty(t *testing.T) {
	cfg := Backend{MQTTClientID: "backend-service"}

	if got, want := cfg.RuntimeMQTTClientID(), "backend-service"; got != want {
		t.Fatalf("RuntimeMQTTClientID() = %q, want %q", got, want)
	}
}

func TestRuntimeMQTTClientIDAppendsInstanceID(t *testing.T) {
	cfg := Backend{
		MQTTClientID: "backend-service",
		InstanceID:   "iot-platform-backend-7fdbcc7f5c-k9zq8",
	}

	if got, want := cfg.RuntimeMQTTClientID(), "backend-service-iot-platform-backend-7fdbcc7f5c-k9zq8"; got != want {
		t.Fatalf("RuntimeMQTTClientID() = %q, want %q", got, want)
	}
}
