package mock

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/pkg/apperrors"
)

func TestSendSuccess(t *testing.T) {
	p := New(notification.ChannelEmail)
	d := &notification.Delivery{
		Channel:   notification.ChannelEmail,
		Recipient: json.RawMessage(`{"address":"ok@example.com"}`),
	}
	result, err := p.Send(context.Background(), d)
	if err != nil {
		t.Fatal(err)
	}
	if result.ProviderMessageID == "" {
		t.Fatal("expected provider message id")
	}
	if p.SentCount() != 1 {
		t.Fatal("expected sent count 1")
	}
}

func TestSendPermanentFailure(t *testing.T) {
	p := New(notification.ChannelEmail)
	d := &notification.Delivery{
		Recipient: json.RawMessage(`{"address":"fail-permanent@example.com"}`),
	}
	_, err := p.Send(context.Background(), d)
	if err == nil || !apperrors.IsPermanent(err) {
		t.Fatalf("expected permanent error, got %v", err)
	}
}

func TestSendTransientFailure(t *testing.T) {
	p := New(notification.ChannelSMS)
	d := &notification.Delivery{
		Recipient: json.RawMessage(`{"address":"fail-transient@example.com"}`),
	}
	_, err := p.Send(context.Background(), d)
	if err == nil || !apperrors.IsTransient(err) {
		t.Fatalf("expected transient error, got %v", err)
	}
}

func TestAllProviders(t *testing.T) {
	providers := All()
	if len(providers) != 4 {
		t.Fatalf("expected 4 providers, got %d", len(providers))
	}
}
