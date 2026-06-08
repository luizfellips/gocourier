package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	global   *Provider
	globalMu sync.RWMutex
)

// Provider holds telemetry state for a process.
type Provider struct {
	Config   Config
	Metrics  *Metrics
	Tracer   trace.Tracer
	shutdown func(context.Context) error
}

// Global returns the process telemetry provider (may be nil before Init).
func Global() *Provider {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global
}

// MetricsGlobal returns metrics or a no-op safe nil check pattern.
func MetricsGlobal() *Metrics {
	if p := Global(); p != nil {
		return p.Metrics
	}
	return nil
}

// Init configures tracing, metrics, and global propagation.
func Init(ctx context.Context, cfg Config) (*Provider, error) {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	metrics := newMetrics(reg)

	res, err := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry resource: %w", err)
	}

	var tp *sdktrace.TracerProvider
	var shutdownFns []func(context.Context) error

	if cfg.EnableOTLP {
		exporter, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("otlp trace exporter: %w", err)
		}
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
		shutdownFns = append(shutdownFns, tp.Shutdown)
	} else {
		tp = sdktrace.NewTracerProvider(sdktrace.WithResource(res))
		shutdownFns = append(shutdownFns, tp.Shutdown)
	}

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := tp.Tracer(cfg.ServiceName)
	p := &Provider{
		Config:  cfg,
		Metrics: metrics,
		Tracer:  tracer,
		shutdown: func(ctx context.Context) error {
			var first error
			for _, fn := range shutdownFns {
				if err := fn(ctx); err != nil && first == nil {
					first = err
				}
			}
			return first
		},
	}

	globalMu.Lock()
	global = p
	globalMu.Unlock()

	return p, nil
}

// Shutdown flushes telemetry exporters.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil || p.shutdown == nil {
		return nil
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return p.shutdown(cctx)
}

// StartSpan starts a traced span with optional attributes.
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := otel.Tracer("github.com/gocourier")
	if p := Global(); p != nil {
		tracer = p.Tracer
	}
	return tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

func spanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// Logger returns slog with trace_id/span_id from context.
func Logger(base *slog.Logger) *slog.Logger {
	return base.With("telemetry", true)
}
