package postgres

import (
	"context"
	"encoding/json"

	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/apperrors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DashboardRepo struct {
	pool *pgxpool.Pool
}

func NewDashboardRepo(pool *pgxpool.Pool) *DashboardRepo {
	return &DashboardRepo{pool: pool}
}

func (r *DashboardRepo) Summary(ctx context.Context, limit int) (*ports.DashboardSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	summary := &ports.DashboardSummary{
		StatusCounts:   make(map[string]int),
		Deliveries:     []ports.DeliveryRow{},
		RecentActivity: []ports.AuditEntry{},
	}

	rows, err := r.pool.Query(ctx, `SELECT status, COUNT(*) FROM deliveries GROUP BY status`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			rows.Close()
			return nil, err
		}
		summary.StatusCounts[status] = count
	}
	rows.Close()

	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox WHERE status = 'pending'`).Scan(&summary.OutboxPending)
	_ = r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM scheduled_notifications WHERE processed = FALSE AND scheduled_at <= NOW()
	`).Scan(&summary.ScheduledDue)

	deliveryRows, err := r.pool.Query(ctx, `
		SELECT id, idempotency_key, channel, priority, status, retry_count,
			COALESCE(last_error, ''), created_at, updated_at, scheduled_at
		FROM deliveries ORDER BY created_at DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer deliveryRows.Close()

	for deliveryRows.Next() {
		row, err := scanDeliveryRow(deliveryRows)
		if err != nil {
			return nil, err
		}
		summary.Deliveries = append(summary.Deliveries, row)
	}

	auditRows, err := r.pool.Query(ctx, `
		SELECT id, delivery_id, event_type, payload, created_at
		FROM audit_events ORDER BY created_at DESC LIMIT 15
	`)
	if err != nil {
		return nil, err
	}
	defer auditRows.Close()

	for auditRows.Next() {
		entry, err := scanAuditEntry(auditRows)
		if err != nil {
			return nil, err
		}
		summary.RecentActivity = append(summary.RecentActivity, entry)
	}

	return summary, deliveryRows.Err()
}

func (r *DashboardRepo) Detail(ctx context.Context, deliveryID string) (*ports.DeliveryDetail, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, idempotency_key, channel, priority, status, retry_count,
			COALESCE(last_error, ''), created_at, updated_at, scheduled_at
		FROM deliveries WHERE id = $1
	`, deliveryID)

	delivery, err := scanDeliveryRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}

	detail := &ports.DeliveryDetail{
		Delivery: delivery,
		Attempts: []ports.AttemptRow{},
		Audit:    []ports.AuditEntry{},
		Outbox:   []ports.OutboxStatusRow{},
	}

	attemptRows, err := r.pool.Query(ctx, `
		SELECT id, attempt_number, success, COALESCE(error_message, ''),
			started_at, finished_at, provider_response
		FROM delivery_attempts WHERE delivery_id = $1 ORDER BY attempt_number
	`, deliveryID)
	if err != nil {
		return nil, err
	}
	defer attemptRows.Close()

	for attemptRows.Next() {
		var a ports.AttemptRow
		var resp []byte
		if err := attemptRows.Scan(&a.ID, &a.AttemptNumber, &a.Success, &a.ErrorMessage,
			&a.StartedAt, &a.FinishedAt, &resp); err != nil {
			return nil, err
		}
		if len(resp) > 0 {
			a.Response = resp
		}
		detail.Attempts = append(detail.Attempts, a)
	}

	auditRows, err := r.pool.Query(ctx, `
		SELECT id, delivery_id, event_type, payload, created_at
		FROM audit_events WHERE delivery_id = $1 ORDER BY created_at
	`, deliveryID)
	if err != nil {
		return nil, err
	}
	defer auditRows.Close()

	for auditRows.Next() {
		entry, err := scanAuditEntry(auditRows)
		if err != nil {
			return nil, err
		}
		detail.Audit = append(detail.Audit, entry)
	}

	outboxRows, err := r.pool.Query(ctx, `
		SELECT id, status, subject, attempts, COALESCE(last_error, ''), created_at, published_at
		FROM outbox WHERE delivery_id = $1 ORDER BY created_at
	`, deliveryID)
	if err != nil {
		return nil, err
	}
	defer outboxRows.Close()

	for outboxRows.Next() {
		var o ports.OutboxStatusRow
		if err := outboxRows.Scan(&o.ID, &o.Status, &o.Subject, &o.Attempts,
			&o.LastError, &o.CreatedAt, &o.Published); err != nil {
			return nil, err
		}
		detail.Outbox = append(detail.Outbox, o)
	}

	return detail, nil
}

func scanDeliveryRow(row pgx.Row) (ports.DeliveryRow, error) {
	var d ports.DeliveryRow
	err := row.Scan(
		&d.ID, &d.IdempotencyKey, &d.Channel, &d.Priority, &d.Status,
		&d.RetryCount, &d.LastError, &d.CreatedAt, &d.UpdatedAt, &d.ScheduledAt,
	)
	return d, err
}

func scanAuditEntry(row pgx.Row) (ports.AuditEntry, error) {
	var e ports.AuditEntry
	var payload []byte
	err := row.Scan(&e.ID, &e.DeliveryID, &e.EventType, &payload, &e.CreatedAt)
	if err != nil {
		return e, err
	}
	if len(payload) > 0 {
		e.Payload = json.RawMessage(payload)
	}
	return e, nil
}
