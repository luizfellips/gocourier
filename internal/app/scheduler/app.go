package scheduler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gocourier/internal/application/scheduler"
	"github.com/gocourier/internal/bootstrap"
	"github.com/gocourier/internal/config"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/internal/wiring"
	"github.com/gocourier/pkg/telemetry"
)

// App runs the scheduler binary: due notification promotion via outbox.
type App struct {
	scheduler *scheduler.Service
	log       *slog.Logger
	telCfg    telemetry.Config
}

// New wires scheduler-specific services from shared infrastructure.
func New(cfg *config.Config, infra *bootstrap.Infra) *App {
	repos := wiring.NewRepositories(infra.Pool)
	svc := wiring.NewServices(wiring.ParamsFromConfig(
		cfg, repos, infra.Broker, ports.SystemClock{}, infra.Log, nil,
	))

	return &App{
		scheduler: svc.Scheduler,
		log:       infra.Log,
		telCfg:    infra.TelCfg,
	}
}

// Run blocks until the scheduler stops or the context is cancelled.
func (a *App) Run(ctx context.Context) error {
	a.log.Info("starting scheduler", "metrics", a.telCfg.MetricsAddr)
	if err := a.scheduler.Run(ctx); err != nil && ctx.Err() == nil {
		return fmt.Errorf("scheduler: %w", err)
	}
	return nil
}
