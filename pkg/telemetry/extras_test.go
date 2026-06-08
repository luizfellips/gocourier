package telemetry

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestWrapHandlerAddsTraceFields(t *testing.T) {
	ctx := context.Background()
	p, err := Init(ctx, Config{ServiceName: "test", EnableOTLP: false})
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Shutdown(ctx) })

	var buf bytes.Buffer
	h := WrapHandler(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, span := StartSpan(ctx, "log.test")
	defer span.End()
	require.NoError(t, h.Handle(ctx, slog.NewRecord(time.Now(), slog.LevelInfo, "hello", 0)))
	require.Contains(t, buf.String(), "trace_id")
}

func TestTruncateSQL(t *testing.T) {
	short := truncateSQL("SELECT 1")
	require.Equal(t, "SELECT 1", short)
	long := truncateSQL(string(make([]byte, 150)))
	require.True(t, len(long) <= 123)
	require.Contains(t, long, "...")
}

func TestPGXTracer(t *testing.T) {
	ctx := context.Background()
	p, err := Init(ctx, Config{ServiceName: "test", EnableOTLP: false})
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Shutdown(ctx) })

	tracer := NewPGXTracer()
	ctx = tracer.TraceQueryStart(ctx, nil, pgx.TraceQueryStartData{SQL: "SELECT 1"})
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})
}

func TestRunPoolStatsReporterNilPool(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	RunPoolStatsReporter(ctx, nil)
}

func TestNewMetricsServerStarts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, NewMetricsServer(ctx, "127.0.0.1:0"))
}

func TestInjectTraceNilHeaders(t *testing.T) {
	InjectTrace(context.Background(), nil)
}

func TestExtractTraceEmptyHeaders(t *testing.T) {
	ctx := ExtractTrace(context.Background(), map[string]string{})
	require.NotNil(t, ctx)
}
