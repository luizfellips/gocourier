package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/pkg/apperrors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeliveryRepo struct {
	pool *pgxpool.Pool
}

func NewDeliveryRepo(pool *pgxpool.Pool) *DeliveryRepo {
	return &DeliveryRepo{pool: pool}
}

func (r *DeliveryRepo) Save(ctx context.Context, d *notification.Delivery) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO deliveries (
			id, idempotency_key, tenant_id, channel, priority,
			recipient, template, payload, status, scheduled_at,
			correlation_id, causation_id, retry_count, last_error,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
	`,
		d.ID, d.IdempotencyKey, d.TenantID, d.Channel, d.Priority,
		d.Recipient, nullableJSON(d.Template), nullableJSON(d.Payload),
		d.Status, d.ScheduledAt, d.CorrelationID, d.CausationID,
		d.RetryCount, nullString(d.LastError), d.CreatedAt, d.UpdatedAt,
	)
	return err
}

func (r *DeliveryRepo) Update(ctx context.Context, d *notification.Delivery) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE deliveries SET
			status = $2, retry_count = $3, last_error = $4, updated_at = $5
		WHERE id = $1
	`, d.ID, d.Status, d.RetryCount, nullString(d.LastError), d.UpdatedAt)
	return err
}

func (r *DeliveryRepo) UpdateIfStatus(ctx context.Context, d *notification.Delivery, expected notification.Status) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE deliveries SET
			status = $2, retry_count = $3, last_error = $4, updated_at = $5
		WHERE id = $1 AND status = $6
	`, d.ID, d.Status, d.RetryCount, nullString(d.LastError), d.UpdatedAt, expected)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *DeliveryRepo) FindByID(ctx context.Context, id string) (*notification.Delivery, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, idempotency_key, tenant_id, channel, priority,
			recipient, template, payload, status, scheduled_at,
			correlation_id, causation_id, retry_count, last_error,
			created_at, updated_at
		FROM deliveries WHERE id = $1
	`, id)
	return scanDelivery(row)
}

func (r *DeliveryRepo) FindByIdempotencyKey(ctx context.Context, key string, ch notification.Channel) (*notification.Delivery, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT d.id, d.idempotency_key, d.tenant_id, d.channel, d.priority,
			d.recipient, d.template, d.payload, d.status, d.scheduled_at,
			d.correlation_id, d.causation_id, d.retry_count, d.last_error,
			d.created_at, d.updated_at
		FROM deliveries d
		JOIN idempotency_keys ik ON ik.delivery_id = d.id
		WHERE ik.idempotency_key = $1 AND ik.channel = $2 AND ik.expires_at > NOW()
	`, key, ch)
	d, err := scanDelivery(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.ErrNotFound
	}
	return d, err
}

func (r *DeliveryRepo) RecordAttempt(ctx context.Context, deliveryID string, attempt notification.Attempt) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO delivery_attempts (
			id, delivery_id, attempt_number, started_at, finished_at,
			success, error_message, provider_response
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`,
		attempt.ID, deliveryID, attempt.AttemptNumber, attempt.StartedAt,
		attempt.FinishedAt, attempt.Success, nullString(attempt.ErrorMessage),
		nullableJSON(attempt.ProviderResponse),
	)
	return err
}

func scanDelivery(row pgx.Row) (*notification.Delivery, error) {
	var d notification.Delivery
	var template, payload []byte
	var scheduledAt *time.Time
	var lastError *string

	err := row.Scan(
		&d.ID, &d.IdempotencyKey, &d.TenantID, &d.Channel, &d.Priority,
		&d.Recipient, &template, &payload, &d.Status, &scheduledAt,
		&d.CorrelationID, &d.CausationID, &d.RetryCount, &lastError,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if template != nil {
		d.Template = template
	}
	if payload != nil {
		d.Payload = payload
	}
	d.ScheduledAt = scheduledAt
	if lastError != nil {
		d.LastError = *lastError
	}
	return &d, nil
}

func nullableJSON(b json.RawMessage) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) IngestTransactional(
	ctx context.Context,
	d *notification.Delivery,
	expiresAt time.Time,
	outboxSubject string,
	outboxPayload []byte,
	outboxHeaders map[string]string,
	schedule bool,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO deliveries (
			id, idempotency_key, tenant_id, channel, priority,
			recipient, template, payload, status, scheduled_at,
			correlation_id, causation_id, retry_count, last_error,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
	`,
		d.ID, d.IdempotencyKey, d.TenantID, d.Channel, d.Priority,
		d.Recipient, nullableJSON(d.Template), nullableJSON(d.Payload),
		d.Status, d.ScheduledAt, d.CorrelationID, d.CausationID,
		d.RetryCount, nullString(d.LastError), d.CreatedAt, d.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert delivery: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO idempotency_keys (idempotency_key, channel, delivery_id, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (idempotency_key, channel) DO NOTHING
	`, d.IdempotencyKey, d.Channel, d.ID, expiresAt)
	if err != nil {
		return fmt.Errorf("insert idempotency: %w", err)
	}

	var existingID string
	err = tx.QueryRow(ctx, `
		SELECT delivery_id::text FROM idempotency_keys
		WHERE idempotency_key = $1 AND channel = $2
	`, d.IdempotencyKey, d.Channel).Scan(&existingID)
	if err != nil {
		return fmt.Errorf("verify idempotency: %w", err)
	}
	if existingID != d.ID {
		return apperrors.ErrDuplicate
	}

	if schedule && d.ScheduledAt != nil {
		_, err = tx.Exec(ctx, `
			INSERT INTO scheduled_notifications (delivery_id, scheduled_at)
			VALUES ($1, $2)
		`, d.ID, d.ScheduledAt)
		if err != nil {
			return fmt.Errorf("insert scheduled: %w", err)
		}
	} else {
		headersJSON, _ := json.Marshal(outboxHeaders)
		_, err = tx.Exec(ctx, `
			INSERT INTO outbox (delivery_id, subject, payload, headers)
			VALUES ($1, $2, $3, $4)
		`, d.ID, outboxSubject, outboxPayload, headersJSON)
		if err != nil {
			return fmt.Errorf("insert outbox: %w", err)
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events (delivery_id, event_type, payload)
		VALUES ($1, $2, $3)
	`, d.ID, string(notification.EventReceived), json.RawMessage(`{}`))
	if err != nil {
		return fmt.Errorf("insert audit: %w", err)
	}

	return tx.Commit(ctx)
}

func (s *Store) TryRegisterIdempotency(ctx context.Context, key string, ch notification.Channel, deliveryID string, expiresAt time.Time) (bool, error) {
	var existing string
	err := s.pool.QueryRow(ctx, `
		SELECT delivery_id::text FROM idempotency_keys
		WHERE idempotency_key = $1 AND channel = $2 AND expires_at > NOW()
	`, key, ch).Scan(&existing)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	return true, nil
}

func NewAttemptID() string {
	return uuid.NewString()
}
