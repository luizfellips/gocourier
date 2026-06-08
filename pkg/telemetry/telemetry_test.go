package telemetry

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestMetricsHandlerExposesBusinessMetrics(t *testing.T) {
	ctx := context.Background()
	p, err := Init(ctx, Config{ServiceName: "test", EnableOTLP: false})
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Shutdown(ctx) })

	p.Metrics.NotificationsReceived.WithLabelValues("email", "false").Inc()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	MetricsHandler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body, err := io.ReadAll(rec.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "notifications_received_total")
}

func TestInitWithoutOTLP(t *testing.T) {
	ctx := context.Background()
	p, err := Init(ctx, Config{ServiceName: "test", EnableOTLP: false})
	require.NoError(t, err)
	require.NotNil(t, p.Metrics)
	require.NotNil(t, p.Tracer)
	require.NoError(t, p.Shutdown(ctx))
}

func TestRecordAPIRequest(t *testing.T) {
	ctx := context.Background()
	p, err := Init(ctx, Config{ServiceName: "test", EnableOTLP: false})
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Shutdown(ctx) })

	RecordAPIRequest("POST", "/v1/notifications", "202", 0)

	rec := httptest.NewRecorder()
	MetricsHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	require.Contains(t, rec.Body.String(), "api_requests_total")
}

func TestInjectExtractTrace(t *testing.T) {
	ctx := context.Background()
	p, err := Init(ctx, Config{ServiceName: "test", EnableOTLP: false})
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Shutdown(ctx) })

	ctx, span := StartSpan(ctx, "parent")
	defer span.End()

	headers := map[string]string{}
	InjectTrace(ctx, headers)
	require.NotEmpty(t, headers["traceparent"])

	childCtx := ExtractTrace(context.Background(), headers)
	childSpan := spanFromContext(childCtx)
	require.True(t, childSpan.SpanContext().IsValid())
	require.Equal(t, span.SpanContext().TraceID(), childSpan.SpanContext().TraceID())
}

func TestTraceContextCreatesSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	ctx, span := tp.Tracer("test").Start(context.Background(), "notification.ingest")
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	require.Equal(t, "notification.ingest", spans[0].Name)
	_ = ctx
}
