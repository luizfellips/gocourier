package nats

import (
	"context"
	"testing"
)

func TestSanitizeConsumerName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"worker-1-notifications.>", "worker-1-notifications"},
		{"***", "consumer"},
		{"valid_name-123", "valid_name-123"},
		{"dots.and>stars*", "dots-and-stars"},
	}
	for _, tt := range tests {
		if got := sanitizeConsumerName(tt.in); got != tt.want {
			t.Fatalf("sanitizeConsumerName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBrokerCloseNilConn(t *testing.T) {
	b := &Broker{}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSubscribeUnknownStream(t *testing.T) {
	b := &Broker{streamPrefix: "notifications"}
	err := b.Subscribe(context.Background(), "UNKNOWN", "worker-1", nil)
	if err == nil {
		t.Fatal("expected error for unknown stream")
	}
}
