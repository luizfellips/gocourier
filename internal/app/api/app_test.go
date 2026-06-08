package api

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

func testConfig() *config.Config {
	return &config.Config{
		ServiceName:           "api",
		HTTPAddr:              ":0",
		APIKeys:               []string{"test-key"},
		NATSStreamPrefix:      "notifications",
		IdempotencyTTL:        time.Hour,
		OutboxPollInterval:    time.Second,
		OutboxBatchSize:       10,
		SchedulerPollInterval: time.Second,
		MaxRetryAttempts:      3,
		RetryBaseDelay:        time.Millisecond,
		RetryMaxDelay:         time.Second,
		CircuitBreakerThreshold: 5,
		CircuitBreakerWindow:  time.Minute,
		CircuitBreakerCooldown: time.Minute,
	}
}

func testInfra() *bootstrap.Infra {
	return &bootstrap.Infra{
		Log:    logger.New("error"),
		Broker: &nats.Broker{},
		TelCfg: telemetry.Config{ServiceName: "api"},
	}
}

func TestNewApp(t *testing.T) {
	app := New(testConfig(), testInfra())
	if app.server == nil || app.publisher == nil {
		t.Fatal("expected wired app")
	}
}

func TestRunWithCancelledContext(t *testing.T) {
	app := New(testConfig(), testInfra())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = app.Run(ctx)
}
