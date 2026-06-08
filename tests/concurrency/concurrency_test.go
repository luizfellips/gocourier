//go:build concurrency

package concurrency

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/tests/testkit"
	"github.com/stretchr/testify/require"
)

type metricsResult struct {
	Count      int     `json:"count"`
	Succeeded  int     `json:"succeeded"`
	DurationMs int64   `json:"duration_ms"`
	Throughput float64 `json:"throughput_per_sec"`
}

func runConcurrentIngest(t *testing.T, count int) metricsResult {
	t.Helper()
	ctx := context.Background()
	cfg := testkit.DefaultStackConfig()
	cfg.WorkerConcurrency = 5
	stack := testkit.StartStack(ctx, t, cfg)
	cancel := stack.RunBackground(ctx, cfg.WorkerConcurrency, false)
	defer cancel()

	ids := make([]string, count)
	var wg sync.WaitGroup
	var errors atomic.Int32
	start := time.Now()

	wg.Add(count)
	for i := 0; i < count; i++ {
		i := i
		go func() {
			defer wg.Done()
			req := testkit.SampleIngestRequest(fmt.Sprintf("conc-%d-%d", count, i))
			resp, err := stack.Ingest.Ingest(ctx, req)
			if err != nil {
				errors.Add(1)
				return
			}
			ids[i] = resp.DeliveryID
		}()
	}
	wg.Wait()
	require.NoError(t, stack.Outbox.FlushOnce(ctx))

	succeeded := 0
	require.Eventually(t, func() bool {
		succeeded = 0
		for _, id := range ids {
			if id == "" {
				continue
			}
			d, err := stack.DeliveryRepo.FindByID(ctx, id)
			if err != nil {
				continue
			}
			if d.Status == notification.StatusSucceeded {
				succeeded++
			}
		}
		return succeeded >= count-int(errors.Load())
	}, 2*time.Minute, 200*time.Millisecond)

	elapsed := time.Since(start)
	return metricsResult{
		Count:      count,
		Succeeded:  succeeded,
		DurationMs: elapsed.Milliseconds(),
		Throughput: float64(count) / elapsed.Seconds(),
	}
}

func writeMetrics(t *testing.T, name string, result metricsResult) {
	t.Helper()
	path := os.Getenv("CONCURRENCY_METRICS_DIR")
	if path == "" {
		path = "."
	}
	b, _ := json.Marshal(result)
	_ = os.WriteFile(fmt.Sprintf("%s/%s-metrics.json", path, name), b, 0o644)
}

func TestConcurrent100(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency test in short mode")
	}
	result := runConcurrentIngest(t, 100)
	writeMetrics(t, "concurrent-100", result)
	require.GreaterOrEqual(t, result.Succeeded, 95, "at least 95% should succeed")
}

func TestConcurrent1000(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency test in short mode")
	}
	if os.Getenv("RUN_LARGE_CONCURRENCY") != "1" {
		t.Skip("set RUN_LARGE_CONCURRENCY=1 to run")
	}
	result := runConcurrentIngest(t, 1000)
	writeMetrics(t, "concurrent-1000", result)
	require.GreaterOrEqual(t, result.Succeeded, 950)
}

func TestConcurrent10000(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency test in short mode")
	}
	if os.Getenv("RUN_LARGE_CONCURRENCY") != "1" {
		t.Skip("set RUN_LARGE_CONCURRENCY=1 to run")
	}
	result := runConcurrentIngest(t, 10000)
	writeMetrics(t, "concurrent-10000", result)
	require.GreaterOrEqual(t, result.Succeeded, 9500)
}
