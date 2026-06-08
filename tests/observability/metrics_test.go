//go:build integration

package observability

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httphandler "github.com/gocourier/internal/adapters/http"
	"github.com/gocourier/internal/application/replay"
	"github.com/gocourier/pkg/telemetry"
	"github.com/gocourier/tests/testkit"
	"github.com/stretchr/testify/require"
)

func TestMetricsEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping observability test in short mode")
	}

	ctx := context.Background()
	_, err := telemetry.Init(ctx, telemetry.Config{ServiceName: "test-api", EnableOTLP: false})
	require.NoError(t, err)

	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())
	srv := httphandler.NewServer(stack.Ingest, replay.NewService(stack.Dispatch), stack.DashboardRepo, []string{"test-key"}, stack.Log, "api")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	_, err = stack.Ingest.Ingest(ctx, testkit.SampleIngestRequest("metrics-endpoint"))
	require.NoError(t, err)

	resp, err := http.Get(ts.URL + "/metrics")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "notifications_received_total")
}

func TestPrometheusMetricAfterIngest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping observability test in short mode")
	}

	ctx := context.Background()
	p, err := telemetry.Init(ctx, telemetry.Config{ServiceName: "test-api", EnableOTLP: false})
	require.NoError(t, err)

	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())
	srv := httphandler.NewServer(stack.Ingest, replay.NewService(stack.Dispatch), stack.DashboardRepo, []string{"test-key"}, stack.Log, "api")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	beforeResp, err := http.Get(ts.URL + "/metrics")
	require.NoError(t, err)
	beforeBody, _ := io.ReadAll(beforeResp.Body)
	beforeResp.Body.Close()
	before := string(beforeBody)

	_, err = stack.Ingest.Ingest(ctx, testkit.SampleIngestRequest("metrics"))
	require.NoError(t, err)

	afterResp, err := http.Get(ts.URL + "/metrics")
	require.NoError(t, err)
	afterBody, _ := io.ReadAll(afterResp.Body)
	afterResp.Body.Close()

	require.NotEqual(t, before, string(afterBody))
	require.Contains(t, string(afterBody), `notifications_received_total{channel="email",duplicate="false"}`)
	_ = p
}

func TestTraceContextInLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping observability test in short mode")
	}

	ctx := context.Background()
	_, err := telemetry.Init(ctx, telemetry.Config{ServiceName: "test-api", EnableOTLP: false})
	require.NoError(t, err)

	stack := testkit.StartStack(ctx, t, testkit.DefaultStackConfig())
	ctx, span := telemetry.StartSpan(ctx, "notification.ingest")
	_, err = stack.Ingest.Ingest(ctx, testkit.SampleIngestRequest("trace"))
	span.End()
	require.NoError(t, err)

	headers := map[string]string{}
	telemetry.InjectTrace(ctx, headers)
	require.True(t, strings.HasPrefix(headers["traceparent"], "00-"))
}
