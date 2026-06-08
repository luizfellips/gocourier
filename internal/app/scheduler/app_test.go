package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/gocourier/internal/adapters/nats"
	"github.com/gocourier/internal/bootstrap"
	"github.com/gocourier/internal/config"
	"github.com/gocourier/pkg/logger"
	"github.com/gocourier/pkg/telemetry"
)

func TestNewApp(t *testing.T) {
	cfg := &config.Config{
		NATSStreamPrefix:        "notifications",
		IdempotencyTTL:          time.Hour,
		OutboxPollInterval:      time.Second,
		OutboxBatchSize:         10,
		SchedulerPollInterval:   time.Second,
		MaxRetryAttempts:        3,
		RetryBaseDelay:          time.Millisecond,
		RetryMaxDelay:           time.Second,
		CircuitBreakerThreshold: 5,
		CircuitBreakerWindow:    time.Minute,
		CircuitBreakerCooldown:  time.Minute,
	}
	infra := &bootstrap.Infra{
		Log:    logger.New("error"),
		Broker: &nats.Broker{},
		TelCfg: telemetry.Config{ServiceName: "scheduler", MetricsAddr: ":9092"},
	}
	app := New(cfg, infra)
	if app.scheduler == nil {
		t.Fatal("expected wired scheduler app")
	}
}

func TestRunExitsOnCancel(t *testing.T) {
	cfg := &config.Config{
		NATSStreamPrefix:        "notifications",
		IdempotencyTTL:          time.Hour,
		OutboxPollInterval:      time.Second,
		OutboxBatchSize:         10,
		SchedulerPollInterval:   time.Millisecond,
		MaxRetryAttempts:        3,
		RetryBaseDelay:          time.Millisecond,
		RetryMaxDelay:           time.Second,
		CircuitBreakerThreshold: 5,
		CircuitBreakerWindow:    time.Minute,
		CircuitBreakerCooldown:  time.Minute,
	}
	infra := &bootstrap.Infra{
		Log:    logger.New("error"),
		Broker: &nats.Broker{},
		TelCfg: telemetry.Config{ServiceName: "scheduler"},
	}
	app := New(cfg, infra)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := app.Run(ctx); err != nil {
		t.Fatalf("expected nil on cancel, got %v", err)
	}
}
