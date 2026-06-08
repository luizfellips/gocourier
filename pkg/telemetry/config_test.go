package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_SDK_DISABLED", "")
	cfg := ConfigFromEnv("api", ":9090")
	require.Equal(t, "api", cfg.ServiceName)
	require.Equal(t, ":9090", cfg.MetricsAddr)
	require.Equal(t, "parentbased_always_on", cfg.TracesSampler)
	require.False(t, cfg.EnableOTLP)
}

func TestConfigFromEnvOTLP(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	t.Setenv("OTEL_SERVICE_NAME", "custom")
	t.Setenv("METRICS_ADDR", ":9191")
	t.Setenv("OTEL_TRACES_SAMPLER", "always_on")
	cfg := ConfigFromEnv("api", ":9090")
	require.Equal(t, "custom", cfg.ServiceName)
	require.Equal(t, "localhost:4317", cfg.OTLPEndpoint)
	require.Equal(t, ":9191", cfg.MetricsAddr)
	require.Equal(t, "always_on", cfg.TracesSampler)
	require.True(t, cfg.EnableOTLP)
}

func TestConfigFromEnvDisabled(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	t.Setenv("OTEL_SDK_DISABLED", "true")
	cfg := ConfigFromEnv("api", ":9090")
	require.False(t, cfg.EnableOTLP)
}

func TestWrapHTTP(t *testing.T) {
	called := false
	handler := WrapHTTP("api", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/notifications", nil))
	require.True(t, called)
	require.Equal(t, http.StatusAccepted, rec.Code)
}

func TestNewMetricsServerEmptyAddr(t *testing.T) {
	require.NoError(t, NewMetricsServer(context.Background(), ""))
}
