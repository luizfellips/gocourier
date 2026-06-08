package telemetry

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UpdatePoolStats reads pgxpool stats into Prometheus gauges.
func UpdatePoolStats(pool *pgxpool.Pool) {
	m := MetricsGlobal()
	if m == nil || pool == nil {
		return
	}
	stat := pool.Stat()
	m.DBPoolAcquired.Set(float64(stat.AcquiredConns()))
	m.DBPoolIdle.Set(float64(stat.IdleConns()))
	m.DBPoolMax.Set(float64(stat.MaxConns()))
}

// RunPoolStatsReporter periodically updates pool gauges.
func RunPoolStatsReporter(ctx context.Context, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				UpdatePoolStats(pool)
			}
		}
	}()
}
