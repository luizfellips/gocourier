package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// InjectTrace writes W3C trace context into a string map (e.g. NATS headers).
func InjectTrace(ctx context.Context, headers map[string]string) {
	if headers == nil {
		return
	}
	carrier := propagation.MapCarrier(headers)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

// ExtractTrace restores trace context from message headers.
func ExtractTrace(ctx context.Context, headers map[string]string) context.Context {
	if len(headers) == 0 {
		return ctx
	}
	carrier := propagation.MapCarrier(headers)
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}
