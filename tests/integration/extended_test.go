//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/pkg/apperrors"
	"github.com/gocourier/tests/testkit"
	"github.com/stretchr/testify/require"
)

func TestTransientRetryThenSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())

	req := notification.IngestRequest{
		SchemaVersion:  notification.SchemaVersion,
		IdempotencyKey: fmt.Sprintf("transient-retry-%s", "1"),
		Channel:        "email",
		Recipient:      json.RawMessage(`{"address":"fail-transient@example.com"}`),
		Template:       json.RawMessage(`{"id":"x"}`),
	}
	resp, err := stack.Ingest.Ingest(ctx, req)
	require.NoError(t, err)

	require.NoError(t, stack.Outbox.FlushOnce(ctx))
	err = stack.Dispatch.HandleMessage(ctx, "", mustPayload(resp.DeliveryID), nil)
	require.True(t, apperrors.IsTransient(err))

	d, err := stack.DeliveryRepo.FindByID(ctx, resp.DeliveryID)
	require.NoError(t, err)
	require.Equal(t, notification.StatusRetrying, d.Status)

	_, err = stack.Postgres.Pool.Exec(ctx, `
		UPDATE deliveries SET recipient = $2 WHERE id = $1
	`, resp.DeliveryID, json.RawMessage(`{"address":"ok@example.com"}`))
	require.NoError(t, err)

	require.NoError(t, stack.Dispatch.Dispatch(ctx, resp.DeliveryID))
	d, err = stack.DeliveryRepo.FindByID(ctx, resp.DeliveryID)
	require.NoError(t, err)
	require.Equal(t, notification.StatusSucceeded, d.Status)
}

func TestSchedulerEnqueuesDueNotification(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	cfg := testkit.DefaultStackConfig()
	stack := testkit.StartStack(ctx, t, cfg)
	cancel := stack.RunBackground(ctx, 1, true)
	defer cancel()

	scheduledAt := stack.Clock.Now().Add(200 * time.Millisecond)
	req := testkit.SampleIngestRequest("scheduled")
	req.ScheduledAt = &scheduledAt

	resp, err := stack.Ingest.Ingest(ctx, req)
	require.NoError(t, err)

	d, err := stack.DeliveryRepo.FindByID(ctx, resp.DeliveryID)
	require.NoError(t, err)
	require.Equal(t, notification.StatusPending, d.Status)

	stack.Clock.Advance(300 * time.Millisecond)
	time.Sleep(500 * time.Millisecond)

	require.NoError(t, stack.Outbox.FlushOnce(ctx))

	require.Eventually(t, func() bool {
		d, err := stack.DeliveryRepo.FindByID(ctx, resp.DeliveryID)
		if err != nil {
			return false
		}
		return d.Status == notification.StatusQueued || d.Status == notification.StatusSucceeded
	}, 5*time.Second, 100*time.Millisecond)
}

func TestReplayFromDLQ(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())

	req := notification.IngestRequest{
		SchemaVersion:  notification.SchemaVersion,
		IdempotencyKey: fmt.Sprintf("replay-%s", "1"),
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

	_, err = stack.Postgres.Pool.Exec(ctx, `
		UPDATE deliveries SET recipient = $2 WHERE id = $1
	`, resp.DeliveryID, json.RawMessage(`{"address":"ok@example.com"}`))
	require.NoError(t, err)

	require.NoError(t, stack.Replay.Replay(ctx, resp.DeliveryID))
	require.NoError(t, stack.Dispatch.HandleMessage(ctx, "", mustPayload(resp.DeliveryID), nil))

	d, err = stack.DeliveryRepo.FindByID(ctx, resp.DeliveryID)
	require.NoError(t, err)
	require.Equal(t, notification.StatusSucceeded, d.Status)
}

func TestOutboxRecoveryAfterRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())

	req := testkit.SampleIngestRequest("outbox-recovery")
	resp, err := stack.Ingest.Ingest(ctx, req)
	require.NoError(t, err)

	require.NoError(t, stack.Outbox.FlushOnce(ctx))
	require.NoError(t, stack.Dispatch.HandleMessage(ctx, "", mustPayload(resp.DeliveryID), nil))

	d, err := stack.DeliveryRepo.FindByID(ctx, resp.DeliveryID)
	require.NoError(t, err)
	require.Equal(t, notification.StatusSucceeded, d.Status)
}

func mustPayload(deliveryID string) []byte {
	b, _ := json.Marshal(map[string]string{"delivery_id": deliveryID})
	return b
}
