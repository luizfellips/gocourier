package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gocourier/internal/adapters/nats"
	"github.com/gocourier/internal/adapters/postgres"
	"github.com/gocourier/internal/config"
	"github.com/gocourier/pkg/logger"
	"github.com/gocourier/pkg/telemetry"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Infra holds shared process infrastructure: logging, telemetry, database, and messaging.
type Infra struct {
	Log    *slog.Logger
	Pool   *pgxpool.Pool
	Broker *nats.Broker
	TelCfg telemetry.Config

	telemetry *telemetry.Provider
}

// Options tune bootstrap behavior per binary.
type Options struct {
	StartMetricsServer bool
}

// New wires telemetry, PostgreSQL, and NATS for a service process.
func New(ctx context.Context, cfg *config.Config, opts Options) (*Infra, func(), error) {
	log := logger.New(cfg.LogLevel)

	telCfg := telemetry.ConfigFromEnv(cfg.ServiceName, cfg.MetricsAddr)
	provider, err := telemetry.Init(ctx, telCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("telemetry init: %w", err)
	}

	cleanup := func() {
		_ = provider.Shutdown(context.Background())
	}

	if opts.StartMetricsServer {
		if err := telemetry.NewMetricsServer(ctx, telCfg.MetricsAddr); err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("metrics server: %w", err)
		}
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("database connection: %w", err)
	}
	telemetry.RunPoolStatsReporter(ctx, pool)

	broker, err := nats.NewBroker(nats.Config{
		URL:          cfg.NATSURL,
		StreamPrefix: cfg.NATSStreamPrefix,
		MaxDeliver:   cfg.NATSMaxDeliver,
		AckWait:      cfg.NATSAckWait,
	})
	if err != nil {
		pool.Close()
		cleanup()
		return nil, nil, fmt.Errorf("nats connection: %w", err)
	}

	if err := broker.EnsureStreams(ctx); err != nil {
		_ = broker.Close()
		pool.Close()
		cleanup()
		return nil, nil, fmt.Errorf("ensure streams: %w", err)
	}

	fullCleanup := func() {
		_ = broker.Close()
		pool.Close()
		cleanup()
	}

	return &Infra{
		Log:       log,
		Pool:      pool,
		Broker:    broker,
		TelCfg:    telCfg,
		telemetry: provider,
	}, fullCleanup, nil
}
