package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gocourier/internal/adapters/postgres"
	"github.com/gocourier/internal/application/scheduler"
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

	scheduledRepo := postgres.NewScheduledRepo(pool)
	deliveryRepo := postgres.NewDeliveryRepo(pool)
	outboxRepo := postgres.NewOutboxRepo(pool)
	auditRepo := postgres.NewAuditRepo(pool)
	clock := ports.SystemClock{}

	svc := scheduler.NewService(
		scheduledRepo, deliveryRepo, outboxRepo, auditRepo, clock, log,
		cfg.NATSStreamPrefix, cfg.SchedulerPollInterval, cfg.OutboxBatchSize,
	)

	log.Info("starting scheduler", "metrics", telCfg.MetricsAddr)
	if err := svc.Run(ctx); err != nil && ctx.Err() == nil {
		log.Error("scheduler stopped", "error", err)
		os.Exit(1)
	}
}
