package telemetry

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// MetricsHandler returns Prometheus scrape handler.
func MetricsHandler() http.Handler {
	if p := Global(); p != nil && p.Metrics != nil {
		return promhttp.HandlerFor(p.Metrics.Registry, promhttp.HandlerOpts{})
	}
	return promhttp.Handler()
}

// NewMetricsServer starts a standalone /metrics and /health server.
func NewMetricsServer(ctx context.Context, addr string) error {
	if addr == "" {
		return nil
	}
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", MetricsHandler())
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shCtx)
	}()
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// best-effort sidecar server
		}
	}()
	return nil
}

// WrapHTTP wraps an handler with OpenTelemetry HTTP instrumentation.
func WrapHTTP(serviceName string, handler http.Handler) http.Handler {
	return otelhttp.NewHandler(handler, serviceName,
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			if r.Pattern != "" {
				return r.Method + " " + r.Pattern
			}
			return r.Method + " " + r.URL.Path
		}),
	)
}

// RecordAPIRequest records HTTP metrics (call from middleware).
func RecordAPIRequest(method, route, status string, duration time.Duration) {
	m := MetricsGlobal()
	if m == nil {
		return
	}
	m.APIRequestsTotal.WithLabelValues(method, route, status).Inc()
	m.APIRequestDuration.WithLabelValues(method, route, status).Observe(duration.Seconds())
}
