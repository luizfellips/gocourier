package wiring

import (
	"testing"
	"time"

	"github.com/gocourier/internal/adapters/nats"
	"github.com/gocourier/internal/application/dispatch"
	"github.com/gocourier/internal/config"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/logger"
)

func TestNewRepositoriesNilPool(t *testing.T) {
	repos := NewRepositories(nil)
	if repos.Store == nil || repos.Delivery == nil || repos.Dashboard == nil {
		t.Fatal("expected repositories")
	}
}

func TestParamsFromConfig(t *testing.T) {
	cfg := &config.Config{
		NATSStreamPrefix:        "notifications",
		IdempotencyTTL:          time.Hour,
		MaxRetryAttempts:        5,
		RetryBaseDelay:          time.Second,
		RetryMaxDelay:           time.Minute,
		CircuitBreakerThreshold: 3,
		CircuitBreakerWindow:    time.Minute,
		CircuitBreakerCooldown:  time.Minute,
		OutboxPollInterval:      time.Second,
		OutboxBatchSize:         25,
		SchedulerPollInterval:   time.Second,
	}
	repos := NewRepositories(nil)
	p := ParamsFromConfig(cfg, repos, &nats.Broker{}, ports.SystemClock{}, logger.New("error"), nil)
	if p.StreamPrefix != "notifications" {
		t.Fatalf("stream prefix: %s", p.StreamPrefix)
	}
	if p.Dispatch.MaxAttempts != 5 {
		t.Fatalf("max attempts: %d", p.Dispatch.MaxAttempts)
	}
	if p.OutboxBatchSize != 25 {
		t.Fatalf("batch size: %d", p.OutboxBatchSize)
	}
}

func TestNewServices(t *testing.T) {
	repos := NewRepositories(nil)
	p := Params{
		Repos:                 repos,
		Broker:                &nats.Broker{},
		Clock:                 ports.SystemClock{},
		Log:                   logger.New("error"),
		StreamPrefix:          "notifications",
		IdempotencyTTL:        time.Hour,
		OutboxPollInterval:    time.Second,
		OutboxBatchSize:       10,
		SchedulerPollInterval: time.Second,
		Dispatch: dispatch.Config{
			MaxAttempts:  3,
			RetryBase:    time.Millisecond,
			RetryMax:     time.Second,
			StreamPrefix: "notifications",
			CBThreshold:  5,
			CBWindow:     time.Minute,
			CBCooldown:   time.Minute,
		},
	}
	svc := NewServices(p)
	if svc.Ingest == nil || svc.Dispatch == nil || svc.Replay == nil || svc.Scheduler == nil || svc.Outbox == nil {
		t.Fatal("expected all services")
	}
}
