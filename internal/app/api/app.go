package api

import (
	"context"
	"fmt"
	"log/slog"

	httphandler "github.com/gocourier/internal/adapters/http"
	chproviders "github.com/gocourier/internal/adapters/providers"
	"github.com/gocourier/internal/application/outbox"
	"github.com/gocourier/internal/bootstrap"
	"github.com/gocourier/internal/config"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/internal/wiring"
	"github.com/gocourier/pkg/telemetry"
)

// App runs the API binary: HTTP server and outbox publisher.
type App struct {
	server    *httphandler.Server
	publisher *outbox.Publisher
	log       *slog.Logger
	telCfg    telemetry.Config
	httpAddr  string
}

// New wires API-specific services from shared infrastructure.
func New(cfg *config.Config, infra *bootstrap.Infra) *App {
	repos := wiring.NewRepositories(infra.Pool)
	svc := wiring.NewServices(wiring.ParamsFromConfig(
		cfg, repos, infra.Broker, ports.SystemClock{}, infra.Log, chproviders.Default(),
	))

	server := httphandler.NewServer(
		svc.Ingest, svc.Replay, repos.Dashboard, cfg.APIKeys, infra.Log, cfg.ServiceName,
	)

	return &App{
		server:    server,
		publisher: svc.Outbox,
		log:       infra.Log,
		telCfg:    infra.TelCfg,
		httpAddr:  cfg.HTTPAddr,
	}
}

// Run starts background workers and blocks until the HTTP server stops.
func (a *App) Run(ctx context.Context) error {
	go func() {
		if err := a.publisher.Run(ctx); err != nil && ctx.Err() == nil {
			a.log.Error("outbox publisher stopped", "error", err)
		}
	}()

	a.log.Info("starting api server", "addr", a.httpAddr, "otel", a.telCfg.EnableOTLP)
	if err := a.server.ListenAndServe(ctx, a.httpAddr); err != nil {
		return fmt.Errorf("api server: %w", err)
	}
	return nil
}
