package testkit

import (
	"context"
	"testing"

	"github.com/gocourier/internal/adapters/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer wraps a testcontainers Postgres instance.
type PostgresContainer struct {
	Pool *pgxpool.Pool
	URL  string
}

// StartPostgres spins up Postgres with migrations applied.
func StartPostgres(ctx context.Context, t *testing.T) *PostgresContainer {
	t.Helper()
	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("gocourier"),
		tcpostgres.WithUsername("gocourier"),
		tcpostgres.WithPassword("gocourier"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })

	pgURL, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := postgres.NewPool(ctx, pgURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return &PostgresContainer{Pool: pool, URL: pgURL}
}
