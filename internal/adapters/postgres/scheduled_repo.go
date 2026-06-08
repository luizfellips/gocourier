package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ScheduledRepo struct {
	pool *pgxpool.Pool
}

func NewScheduledRepo(pool *pgxpool.Pool) *ScheduledRepo {
	return &ScheduledRepo{pool: pool}
}

func (r *ScheduledRepo) Enqueue(ctx context.Context, deliveryID string, scheduledAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO scheduled_notifications (delivery_id, scheduled_at)
		VALUES ($1, $2)
		ON CONFLICT (delivery_id) DO NOTHING
	`, deliveryID, scheduledAt)
	return err
}

func (r *ScheduledRepo) FetchDue(ctx context.Context, before time.Time, limit int) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT delivery_id::text FROM scheduled_notifications
		WHERE processed = FALSE AND scheduled_at <= $1
		ORDER BY scheduled_at ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *ScheduledRepo) MarkProcessed(ctx context.Context, deliveryID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE scheduled_notifications SET processed = TRUE WHERE delivery_id = $1
	`, deliveryID)
	return err
}
