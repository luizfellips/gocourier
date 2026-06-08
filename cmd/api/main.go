package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	httphandler "github.com/gocourier/internal/adapters/http"
	"github.com/gocourier/internal/adapters/nats"
	"github.com/gocourier/internal/adapters/postgres"
	"github.com/gocourier/internal/adapters/providers/mock"
	"github.com/gocourier/internal/application/dispatch"
	"github.com/gocourier/internal/application/ingest"
	"github.com/gocourier/internal/application/outbox"
	"github.com/gocourier/internal/application/replay"
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

	store := postgres.NewStore(pool)
	deliveryRepo := postgres.NewDeliveryRepo(pool)
	dashboardRepo := postgres.NewDashboardRepo(pool)
	outboxRepo := postgres.NewOutboxRepo(pool)
	auditRepo := postgres.NewAuditRepo(pool)
	clock := ports.SystemClock{}

	ingestSvc := ingest.NewService(store, deliveryRepo, auditRepo, clock, cfg.NATSStreamPrefix, cfg.IdempotencyTTL)

	dispatchSvc := dispatch.NewService(
		deliveryRepo, auditRepo, broker, mock.All(), clock, log,
		dispatch.Config{
			MaxAttempts:  cfg.MaxRetryAttempts,
			RetryBase:    cfg.RetryBaseDelay,
			RetryMax:     cfg.RetryMaxDelay,
			StreamPrefix: cfg.NATSStreamPrefix,
			CBThreshold:  cfg.CircuitBreakerThreshold,
			CBWindow:     cfg.CircuitBreakerWindow,
			CBCooldown:   cfg.CircuitBreakerCooldown,
		},
	)
	replaySvc := replay.NewService(dispatchSvc)

	publisher := outbox.NewPublisher(outboxRepo, broker, log, cfg.OutboxPollInterval, cfg.OutboxBatchSize)
	go func() {
		if err := publisher.Run(ctx); err != nil && ctx.Err() == nil {
			log.Error("outbox publisher stopped", "error", err)
		}
	}()

	srv := httphandler.NewServer(ingestSvc, replaySvc, dashboardRepo, cfg.APIKeys, log, cfg.ServiceName)
	log.Info("starting api server", "addr", cfg.HTTPAddr, "otel", telCfg.EnableOTLP)
	if err := srv.ListenAndServe(ctx, cfg.HTTPAddr); err != nil {
		log.Error("api server failed", "error", err)
		os.Exit(1)
	}
}
