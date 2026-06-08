package testkit

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/gocourier/internal/adapters/nats"
	"github.com/gocourier/internal/adapters/postgres"
	chproviders "github.com/gocourier/internal/adapters/providers"
	"github.com/gocourier/internal/adapters/providers/mock"
	"github.com/gocourier/internal/application/dispatch"
	"github.com/gocourier/internal/application/ingest"
	"github.com/gocourier/internal/application/outbox"
	"github.com/gocourier/internal/application/replay"
	"github.com/gocourier/internal/application/scheduler"
	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/internal/wiring"
	"github.com/gocourier/pkg/logger"
	"github.com/stretchr/testify/require"
)

// StackConfig tunes the test stack.
type StackConfig struct {
	NATSMaxDeliver    int
	NATSAckWait       time.Duration
	MaxAttempts       int
	WorkerConcurrency int
	IdempotencyTTL    time.Duration
	CBThreshold       int
	CBWindow          time.Duration
	CBCooldown        time.Duration
}

// DefaultStackConfig returns sensible defaults for integration tests.
func DefaultStackConfig() StackConfig {
	return StackConfig{
		NATSMaxDeliver:    8,
		NATSAckWait:       10 * time.Second,
		MaxAttempts:       8,
		WorkerConcurrency: 2,
		IdempotencyTTL:    24 * time.Hour,
		CBThreshold:       100,
		CBWindow:          60 * time.Second,
		CBCooldown:        30 * time.Second,
	}
}

// Stack wires Postgres, NATS, and application services for integration tests.
type Stack struct {
	Postgres *PostgresContainer
	Broker   *nats.Broker
	Clock    *FixedClock

	Store         *postgres.Store
	DeliveryRepo  *postgres.DeliveryRepo
	OutboxRepo    *postgres.OutboxRepo
	ScheduledRepo *postgres.ScheduledRepo
	AuditRepo     *postgres.AuditRepo
	DashboardRepo *postgres.DashboardRepo

	Ingest    *ingest.Service
	Dispatch  *dispatch.Service
	Replay    *replay.Service
	Scheduler *scheduler.Service
	Outbox    *outbox.Publisher

	Providers []ports.ChannelProvider
	Log       *slog.Logger
}

// StartStack creates containers and wires services.
func StartStack(ctx context.Context, t *testing.T, cfg StackConfig) *Stack {
	t.Helper()
	pg := StartPostgres(ctx, t)
	natsURL := StartNATS(ctx, t)

	broker, err := nats.NewBroker(nats.Config{
		URL:          natsURL,
		StreamPrefix: "notifications",
		MaxDeliver:   cfg.NATSMaxDeliver,
		AckWait:      cfg.NATSAckWait,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = broker.Close() })
	require.NoError(t, broker.EnsureStreams(ctx))

	log := logger.New("error")
	clock := NewFixedClock(time.Now().UTC())
	providers := chproviders.Default()

	repos := wiring.NewRepositories(pg.Pool)
	svc := wiring.NewServices(wiring.Params{
		Repos:          repos,
		Broker:         broker,
		Clock:          clock,
		Log:            log,
		Providers:      providers,
		StreamPrefix:   "notifications",
		IdempotencyTTL: cfg.IdempotencyTTL,
		Dispatch: dispatch.Config{
			MaxAttempts:  cfg.MaxAttempts,
			RetryBase:    time.Second,
			RetryMax:     30 * time.Minute,
			StreamPrefix: "notifications",
			CBThreshold:  cfg.CBThreshold,
			CBWindow:     cfg.CBWindow,
			CBCooldown:   cfg.CBCooldown,
		},
		OutboxPollInterval:    100 * time.Millisecond,
		OutboxBatchSize:       50,
		SchedulerPollInterval: 100 * time.Millisecond,
	})

	return &Stack{
		Postgres:      pg,
		Broker:        broker,
		Clock:         clock,
		Store:         repos.Store,
		DeliveryRepo:  repos.Delivery,
		OutboxRepo:    repos.Outbox,
		ScheduledRepo: repos.Scheduled,
		AuditRepo:     repos.Audit,
		DashboardRepo: repos.Dashboard,
		Ingest:        svc.Ingest,
		Dispatch:      svc.Dispatch,
		Replay:        svc.Replay,
		Scheduler:     svc.Scheduler,
		Outbox:        svc.Outbox,
		Providers:     providers,
		Log:           log,
	}
}

// RunBackground starts outbox publisher, worker, and optionally scheduler.
func (s *Stack) RunBackground(ctx context.Context, workerConcurrency int, withScheduler bool) context.CancelFunc {
	bgCtx, cancel := context.WithCancel(ctx)
	go func() { _ = s.Outbox.Run(bgCtx) }()
	go func() { _ = s.Dispatch.Run(bgCtx, workerConcurrency, nil) }()
	if withScheduler {
		go func() { _ = s.Scheduler.Run(bgCtx) }()
	}
	return cancel
}

// FlushAndDispatch synchronously flushes outbox and dispatches a delivery.
func (s *Stack) FlushAndDispatch(ctx context.Context, t *testing.T, deliveryID string) {
	t.Helper()
	require.NoError(t, s.Outbox.FlushOnce(ctx))
	payload, _ := json.Marshal(map[string]string{"delivery_id": deliveryID})
	require.NoError(t, s.Dispatch.HandleMessage(ctx, "", payload, nil))
}

// MockEmailProvider returns the email mock provider from the stack.
func (s *Stack) MockEmailProvider() *mock.Provider {
	for _, p := range s.Providers {
		if mp, ok := p.(*mock.Provider); ok && mp.Channel() == notification.ChannelEmail {
			return mp
		}
	}
	return nil
}
