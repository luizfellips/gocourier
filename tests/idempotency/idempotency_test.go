//go:build integration

package idempotency

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/tests/testkit"
	"github.com/stretchr/testify/require"
)

func TestConcurrentSameIdempotencyKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())

	key := "concurrent-idem-key"
	req := testkit.SampleIngestRequestWithKey(key)

	const clients = 10
	var wg sync.WaitGroup
	var successCount atomic.Int32
	ids := make(chan string, clients)

	wg.Add(clients)
	for i := 0; i < clients; i++ {
		go func() {
			defer wg.Done()
			resp, err := stack.Ingest.Ingest(ctx, req)
			if err != nil {
				return
			}
			successCount.Add(1)
			ids <- resp.DeliveryID
		}()
	}
	wg.Wait()
	close(ids)

	require.Equal(t, int32(clients), successCount.Load())

	unique := map[string]struct{}{}
	for id := range ids {
		unique[id] = struct{}{}
	}
	require.Len(t, unique, 1, "expected exactly one delivery ID")
}

func TestSequentialDuplicateSubmits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())
	req := testkit.SampleIngestRequestWithKey("seq-dup-key")

	first, err := stack.Ingest.Ingest(ctx, req)
	require.NoError(t, err)

	for i := 0; i < 99; i++ {
		dup, err := stack.Ingest.Ingest(ctx, req)
		require.NoError(t, err)
		require.True(t, dup.Duplicate)
		require.Equal(t, first.DeliveryID, dup.DeliveryID)
	}
}

func TestSameKeyDifferentChannel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())
	key := "multi-channel-key"

	emailReq := testkit.SampleIngestRequestWithKey(key)
	smsReq := testkit.SampleIngestRequestWithKey(key)
	smsReq.Channel = "sms"
	smsReq.Recipient = json.RawMessage(`{"phone":"+15551234567"}`)

	emailResp, err := stack.Ingest.Ingest(ctx, emailReq)
	require.NoError(t, err)
	smsResp, err := stack.Ingest.Ingest(ctx, smsReq)
	require.NoError(t, err)
	require.NotEqual(t, emailResp.DeliveryID, smsResp.DeliveryID)
}

func TestDispatchDedupOnRedelivery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())

	resp, err := stack.Ingest.Ingest(ctx, testkit.SampleIngestRequest("dispatch-dedup"))
	require.NoError(t, err)

	stack.FlushAndDispatch(ctx, t, resp.DeliveryID)
	payload := mustPayload(resp.DeliveryID)

	require.NoError(t, stack.Dispatch.HandleMessage(ctx, "", payload, nil))
	require.NoError(t, stack.Dispatch.HandleMessage(ctx, "", payload, nil))

	d, err := stack.DeliveryRepo.FindByID(ctx, resp.DeliveryID)
	require.NoError(t, err)
	require.Equal(t, notification.StatusSucceeded, d.Status)

	provider := stack.MockEmailProvider()
	require.NotNil(t, provider)
	require.Equal(t, 1, provider.SentCount())
}

func mustPayload(deliveryID string) []byte {
	b, _ := json.Marshal(map[string]string{"delivery_id": deliveryID})
	return b
}
