package main

import (
	"fmt"
	"log/slog"
	"os"

	workerapp "github.com/gocourier/internal/app/worker"
	"github.com/gocourier/internal/bootstrap"
	"github.com/gocourier/internal/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := bootstrap.SignalContext()
	defer stop()

	infra, cleanup, err := bootstrap.New(ctx, cfg, bootstrap.Options{StartMetricsServer: true})
	if err != nil {
		return err
	}
	defer cleanup()

	app := workerapp.New(cfg, infra)
	return app.Run(ctx)
}
