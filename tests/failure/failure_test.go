//go:build failure

package failure

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/tests/testkit"
	"github.com/stretchr/testify/require"
)

// TestWorkerCrashMidDispatch verifies redelivery after simulated partial processing.
func TestWorkerCrashMidDispatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping failure test in short mode")
	}

	ctx := context.Background()
	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())

	resp, err := stack.Ingest.Ingest(ctx, testkit.SampleIngestRequest("crash-mid"))
	require.NoError(t, err)
	require.NoError(t, stack.Outbox.FlushOnce(ctx))

	d, err := stack.DeliveryRepo.FindByID(ctx, resp.DeliveryID)
	require.NoError(t, err)
	require.NoError(t, d.StartProcessing(stack.Clock.Now()))
	require.NoError(t, stack.DeliveryRepo.Update(ctx, d))

	// Simulated crash — redispatch completes delivery
	require.NoError(t, stack.Dispatch.Dispatch(ctx, resp.DeliveryID))
	d, err = stack.DeliveryRepo.FindByID(ctx, resp.DeliveryID)
	require.NoError(t, err)
	require.Equal(t, notification.StatusSucceeded, d.Status)
}

// TestBrokerOutageOutboxPending verifies outbox retains messages when publish fails.
func TestBrokerOutageOutboxPending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping failure test in short mode")
	}

	ctx := context.Background()
	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())

	resp, err := stack.Ingest.Ingest(ctx, testkit.SampleIngestRequest("broker-outage"))
	require.NoError(t, err)

	// Close broker to simulate outage
	require.NoError(t, stack.Broker.Close())

	err = stack.Outbox.FlushOnce(ctx)
	// flush continues on per-message failures; broker closed means publish fails silently in loop
	_ = err

	var pending int
	require.NoError(t, stack.Postgres.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM outbox WHERE status = 'pending'
	`).Scan(&pending))
	require.GreaterOrEqual(t, pending, 1)

	_ = resp
}

// TestSlowProviderRetry verifies transient failures enter retrying state.
func TestSlowProviderRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping failure test in short mode")
	}

	ctx := context.Background()
	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())

	req := testkit.SampleIngestRequest("slow-provider")
	req.Recipient = json.RawMessage(`{"address":"fail-transient@example.com"}`)

	resp, err := stack.Ingest.Ingest(ctx, req)
	require.NoError(t, err)
	stack.FlushAndDispatch(ctx, t, resp.DeliveryID)

	require.Eventually(t, func() bool {
		d, err := stack.DeliveryRepo.FindByID(ctx, resp.DeliveryID)
		if err != nil {
			return false
		}
		return d.Status == notification.StatusRetrying
	}, 5*time.Second, 100*time.Millisecond)
}
