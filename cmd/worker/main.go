package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gocourier/internal/adapters/nats"
	"github.com/gocourier/internal/adapters/postgres"
	"github.com/gocourier/internal/adapters/providers/mock"
	"github.com/gocourier/internal/application/dispatch"
	"github.com/gocourier/internal/config"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/logger"
	"github.com/gocourier/pkg/telemetry"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log := logger.New(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	telCfg := telemetry.ConfigFromEnv(cfg.ServiceName, cfg.MetricsAddr)
	provider, err := telemetry.Init(ctx, telCfg)
	if err != nil {
		log.Error("telemetry init failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = provider.Shutdown(context.Background())
	}()
	if err := telemetry.NewMetricsServer(ctx, telCfg.MetricsAddr); err != nil {
		log.Error("metrics server failed", "error", err)
		os.Exit(1)
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	telemetry.RunPoolStatsReporter(ctx, pool)

	broker, err := nats.NewBroker(nats.Config{
		URL:          cfg.NATSURL,
		StreamPrefix: cfg.NATSStreamPrefix,
		MaxDeliver:   cfg.NATSMaxDeliver,
		AckWait:      cfg.NATSAckWait,
	})
	if err != nil {
		log.Error("nats connection failed", "error", err)
		os.Exit(1)
	}
	defer broker.Close()

	if err := broker.EnsureStreams(ctx); err != nil {
		log.Error("ensure streams failed", "error", err)
		os.Exit(1)
	}

	deliveryRepo := postgres.NewDeliveryRepo(pool)
	auditRepo := postgres.NewAuditRepo(pool)
	clock := ports.SystemClock{}

	svc := dispatch.NewService(
		deliveryRepo, auditRepo, broker, mock.All(), clock, log,
		cfg.MaxRetryAttempts, cfg.RetryBaseDelay, cfg.RetryMaxDelay,
		cfg.NATSStreamPrefix,
		cfg.CircuitBreakerThreshold, cfg.CircuitBreakerWindow, cfg.CircuitBreakerCooldown,
	)

	log.Info("starting worker", "channels", cfg.WorkerChannels, "concurrency", cfg.WorkerConcurrency, "metrics", telCfg.MetricsAddr)
	if err := svc.Run(ctx, cfg.WorkerConcurrency, cfg.WorkerChannels); err != nil && ctx.Err() == nil {
		log.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
