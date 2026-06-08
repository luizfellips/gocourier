//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gocourier/internal/adapters/providers/mock"
	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/pkg/apperrors"
	"github.com/stretchr/testify/require"
)

func TestMockProviderFailureClassification(t *testing.T) {
	p := mock.New(notification.ChannelEmail)

	permanentDelivery := &notification.Delivery{
		Recipient: json.RawMessage(`{"address":"fail-permanent@example.com"}`),
		Channel:   notification.ChannelEmail,
	}
	_, err := p.Send(context.Background(), permanentDelivery)
	require.True(t, apperrors.IsPermanent(err))

	transientDelivery := &notification.Delivery{
		Recipient: json.RawMessage(`{"address":"fail-transient@example.com"}`),
		Channel:   notification.ChannelEmail,
	}
	_, err = p.Send(context.Background(), transientDelivery)
	require.True(t, apperrors.IsTransient(err))
}
