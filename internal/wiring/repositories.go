package wiring

import (
	"github.com/gocourier/internal/adapters/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repositories groups PostgreSQL adapters constructed from a single pool.
type Repositories struct {
	Store     *postgres.Store
	Delivery  *postgres.DeliveryRepo
	Dashboard *postgres.DashboardRepo
	Outbox    *postgres.OutboxRepo
	Scheduled *postgres.ScheduledRepo
	Audit     *postgres.AuditRepo
}

// NewRepositories constructs all postgres repositories from a pool.
func NewRepositories(pool *pgxpool.Pool) Repositories {
	return Repositories{
		Store:     postgres.NewStore(pool),
		Delivery:  postgres.NewDeliveryRepo(pool),
		Dashboard: postgres.NewDashboardRepo(pool),
		Outbox:    postgres.NewOutboxRepo(pool),
		Scheduled: postgres.NewScheduledRepo(pool),
		Audit:     postgres.NewAuditRepo(pool),
	}
}
