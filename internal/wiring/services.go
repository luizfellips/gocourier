package wiring

import (
	"log/slog"
	"time"

	"github.com/gocourier/internal/adapters/nats"
	"github.com/gocourier/internal/application/dispatch"
	"github.com/gocourier/internal/application/ingest"
	"github.com/gocourier/internal/application/outbox"
	"github.com/gocourier/internal/application/replay"
	"github.com/gocourier/internal/application/scheduler"
	"github.com/gocourier/internal/config"
	"github.com/gocourier/internal/ports"
)

// Services groups application-layer services wired from shared dependencies.
type Services struct {
	Ingest    *ingest.Service
	Dispatch  *dispatch.Service
	Replay    *replay.Service
	Scheduler *scheduler.Service
	Outbox    *outbox.Publisher
}

// Params holds everything needed to construct application services.
type Params struct {
	Repos                 Repositories
	Broker                *nats.Broker
	Clock                 ports.Clock
	Log                   *slog.Logger
	Providers             []ports.ChannelProvider
	StreamPrefix          string
	IdempotencyTTL        time.Duration
	Dispatch              dispatch.Config
	OutboxPollInterval    time.Duration
	OutboxBatchSize       int
	SchedulerPollInterval time.Duration
}

// ParamsFromConfig maps application config into wiring parameters.
func ParamsFromConfig(
	cfg *config.Config,
	repos Repositories,
	broker *nats.Broker,
	clock ports.Clock,
	log *slog.Logger,
	providers []ports.ChannelProvider,
) Params {
	return Params{
		Repos:          repos,
		Broker:         broker,
		Clock:          clock,
		Log:            log,
		Providers:      providers,
		StreamPrefix:   cfg.NATSStreamPrefix,
		IdempotencyTTL: cfg.IdempotencyTTL,
		Dispatch: dispatch.Config{
			MaxAttempts:  cfg.MaxRetryAttempts,
			RetryBase:    cfg.RetryBaseDelay,
			RetryMax:     cfg.RetryMaxDelay,
			StreamPrefix: cfg.NATSStreamPrefix,
			CBThreshold:  cfg.CircuitBreakerThreshold,
			CBWindow:     cfg.CircuitBreakerWindow,
			CBCooldown:   cfg.CircuitBreakerCooldown,
		},
		OutboxPollInterval:    cfg.OutboxPollInterval,
		OutboxBatchSize:       cfg.OutboxBatchSize,
		SchedulerPollInterval: cfg.SchedulerPollInterval,
	}
}

// NewServices constructs all application services from shared wiring parameters.
func NewServices(p Params) Services {
	ingestSvc := ingest.NewService(
		p.Repos.Store, p.Repos.Delivery, p.Repos.Audit, p.Clock,
		p.StreamPrefix, p.IdempotencyTTL,
	)
	dispatchSvc := dispatch.NewService(
		p.Repos.Delivery, p.Repos.Audit, p.Broker, p.Providers, p.Clock, p.Log, p.Dispatch,
	)
	replaySvc := replay.NewService(dispatchSvc)
	publisher := outbox.NewPublisher(
		p.Repos.Outbox, p.Broker, p.Log, p.OutboxPollInterval, p.OutboxBatchSize,
	)
	schedSvc := scheduler.NewService(
		p.Repos.Scheduled, p.Repos.Delivery, p.Repos.Outbox, p.Repos.Audit,
		p.Clock, p.Log, p.StreamPrefix, p.SchedulerPollInterval, p.OutboxBatchSize,
	)

	return Services{
		Ingest:    ingestSvc,
		Dispatch:  dispatchSvc,
		Replay:    replaySvc,
		Scheduler: schedSvc,
		Outbox:    publisher,
	}
}
