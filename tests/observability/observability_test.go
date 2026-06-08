//go:build integration

package observability

import (
	"context"
	"testing"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/tests/testkit"
	"github.com/stretchr/testify/require"
)

func TestAuditTrailCompleteness(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping observability test in short mode")
	}

	ctx := context.Background()
	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())

	resp, err := stack.Ingest.Ingest(ctx, testkit.SampleIngestRequest("audit"))
	require.NoError(t, err)
	stack.FlushAndDispatch(ctx, t, resp.DeliveryID)

	var count int
	require.NoError(t, stack.Postgres.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_events WHERE delivery_id = $1
	`, resp.DeliveryID).Scan(&count))
	require.GreaterOrEqual(t, count, 1)
}

func TestDeliveryTerminalStateRecorded(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping observability test in short mode")
	}

	ctx := context.Background()
	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())

	resp, err := stack.Ingest.Ingest(ctx, testkit.SampleIngestRequest("terminal"))
	require.NoError(t, err)
	stack.FlushAndDispatch(ctx, t, resp.DeliveryID)

	d, err := stack.DeliveryRepo.FindByID(ctx, resp.DeliveryID)
	require.NoError(t, err)
	require.Equal(t, notification.StatusSucceeded, d.Status)
}
