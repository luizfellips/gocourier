package worker

import (
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
		WorkerConcurrency:       2,
		WorkerChannels:          []string{"email"},
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
		TelCfg: telemetry.Config{ServiceName: "worker", MetricsAddr: ":9091"},
	}
	app := New(cfg, infra)
	if app.dispatch == nil || app.concurrency != 2 {
		t.Fatal("expected wired worker app")
	}
}
