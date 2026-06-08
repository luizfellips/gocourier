package telemetry

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type queryStartKey struct{}

type queryTracer struct{}

func NewPGXTracer() pgx.QueryTracer {
	return queryTracer{}
}

func (queryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx, span := StartSpan(ctx, "db.query",
		attribute.String("db.statement", truncateSQL(data.SQL)),
	)
	ctx = context.WithValue(ctx, queryStartKey{}, time.Now())
	_ = span
	return ctx
}

func (queryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	span := spanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	defer span.End()

	op := "query"
	if data.CommandTag.String() != "" {
		op = data.CommandTag.String()
	}
	start, _ := ctx.Value(queryStartKey{}).(time.Time)
	if !start.IsZero() && MetricsGlobal() != nil {
		MetricsGlobal().DBQueryDuration.WithLabelValues(op).Observe(time.Since(start).Seconds())
	}
	if data.Err != nil {
		span.RecordError(data.Err)
		span.SetStatus(codes.Error, data.Err.Error())
	}
}

func truncateSQL(sql string) string {
	if len(sql) > 120 {
		return sql[:120] + "..."
	}
	return sql
}
