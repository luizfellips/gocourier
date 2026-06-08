package worker

import (
	"context"
	"fmt"
	"log/slog"

	chproviders "github.com/gocourier/internal/adapters/providers"
	"github.com/gocourier/internal/application/dispatch"
	"github.com/gocourier/internal/bootstrap"
	"github.com/gocourier/internal/config"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/internal/wiring"
	"github.com/gocourier/pkg/telemetry"
)

// App runs the worker binary: NATS consumers and channel dispatch.
type App struct {
	dispatch    *dispatch.Service
	log         *slog.Logger
	telCfg      telemetry.Config
	concurrency int
	channels    []string
}

// New wires worker-specific services from shared infrastructure.
func New(cfg *config.Config, infra *bootstrap.Infra) *App {
	repos := wiring.NewRepositories(infra.Pool)
	svc := wiring.NewServices(wiring.ParamsFromConfig(
		cfg, repos, infra.Broker, ports.SystemClock{}, infra.Log, chproviders.Default(),
	))

	return &App{
		dispatch:    svc.Dispatch,
		log:         infra.Log,
		telCfg:      infra.TelCfg,
		concurrency: cfg.WorkerConcurrency,
		channels:    cfg.WorkerChannels,
	}
}

// Run blocks until the worker stops or the context is cancelled.
func (a *App) Run(ctx context.Context) error {
	a.log.Info("starting worker",
		"channels", a.channels,
		"concurrency", a.concurrency,
		"metrics", a.telCfg.MetricsAddr,
	)
	if err := a.dispatch.Run(ctx, a.concurrency, a.channels); err != nil && ctx.Err() == nil {
		return fmt.Errorf("worker: %w", err)
	}
	return nil
}
