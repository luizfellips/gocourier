package ports

import (
	"context"
	"encoding/json"

	"github.com/gocourier/internal/domain/notification"
)

type ProviderResult struct {
	ProviderMessageID string          `json:"provider_message_id,omitempty"`
	Response          json.RawMessage `json:"response,omitempty"`
}

type ChannelProvider interface {
	Channel() notification.Channel
	Send(ctx context.Context, d *notification.Delivery) (ProviderResult, error)
}
