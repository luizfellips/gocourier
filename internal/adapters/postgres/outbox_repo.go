package postgres

import (
	"context"
	"encoding/json"

	"github.com/gocourier/internal/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OutboxRepo struct {
	pool *pgxpool.Pool
}

func NewOutboxRepo(pool *pgxpool.Pool) *OutboxRepo {
	return &OutboxRepo{pool: pool}
}

func (r *OutboxRepo) Enqueue(ctx context.Context, deliveryID string, subject string, payload []byte, headers map[string]string) error {
	headersJSON, err := json.Marshal(headers)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO outbox (delivery_id, subject, payload, headers)
		VALUES ($1, $2, $3, $4)
	`, deliveryID, subject, payload, headersJSON)
	return err
}

func (r *OutboxRepo) FetchPending(ctx context.Context, limit int) ([]ports.OutboxMessage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, delivery_id, subject, payload, headers, attempts
		FROM outbox
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []ports.OutboxMessage
	for rows.Next() {
		var m ports.OutboxMessage
		var headersJSON []byte
		if err := rows.Scan(&m.ID, &m.DeliveryID, &m.Subject, &m.Payload, &headersJSON, &m.Attempts); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(headersJSON, &m.Headers)
		if m.Headers == nil {
			m.Headers = map[string]string{}
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (r *OutboxRepo) CountPending(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox WHERE status = 'pending'`).Scan(&count)
	return count, err
}

func (r *OutboxRepo) MarkPublished(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE outbox SET status = 'published', published_at = NOW() WHERE id = $1
	`, id)
	return err
}

func (r *OutboxRepo) MarkFailed(ctx context.Context, id int64, errMsg string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE outbox SET attempts = attempts + 1, last_error = $2 WHERE id = $1
	`, id, errMsg)
	return err
}

func (r *OutboxRepo) HasPublishedForDelivery(ctx context.Context, deliveryID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM outbox WHERE delivery_id = $1 AND status = 'published'
		)
	`, deliveryID).Scan(&exists)
	return exists, err
}
