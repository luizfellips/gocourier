//go:build integration

package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/tests/testkit"
	"github.com/stretchr/testify/require"
)

func ingestPermanentFailure(t *testing.T, stack *testkit.Stack, key string) string {
	t.Helper()
	ctx := context.Background()
	req := notification.IngestRequest{
		SchemaVersion:  notification.SchemaVersion,
		IdempotencyKey: key,
		Channel:        "email",
		Recipient:      json.RawMessage(`{"address":"fail-permanent@example.com"}`),
		Template:       json.RawMessage(`{"id":"x"}`),
	}
	resp, err := stack.Ingest.Ingest(ctx, req)
	require.NoError(t, err)
	require.NoError(t, stack.Outbox.FlushOnce(ctx))
	require.NoError(t, stack.Dispatch.HandleMessage(ctx, "", mustPayload(resp.DeliveryID), nil))

	d, err := stack.DeliveryRepo.FindByID(ctx, resp.DeliveryID)
	require.NoError(t, err)
	require.Equal(t, notification.StatusDLQ, d.Status)
	return resp.DeliveryID
}

func TestReplayBatch10(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping replay test in short mode")
	}

	ctx := context.Background()
	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())

	ids := make([]string, 0, 10)
	for i := 0; i < 10; i++ {
		ids = append(ids, ingestPermanentFailure(t, stack, fmt.Sprintf("replay-key-%d", i)))
	}

	for _, id := range ids {
		_, err := stack.Postgres.Pool.Exec(ctx, `
			UPDATE deliveries SET recipient = $2 WHERE id = $1
		`, id, json.RawMessage(`{"address":"ok@example.com"}`))
		require.NoError(t, err)
		require.NoError(t, stack.Replay.Replay(ctx, id))
	}

	for _, id := range ids {
		require.NoError(t, stack.Dispatch.HandleMessage(ctx, "", mustPayload(id), nil))
		d, err := stack.DeliveryRepo.FindByID(ctx, id)
		require.NoError(t, err)
		require.Equal(t, notification.StatusSucceeded, d.Status)
	}
}

func mustPayload(deliveryID string) []byte {
	b, _ := json.Marshal(map[string]string{"delivery_id": deliveryID})
	return b
}
