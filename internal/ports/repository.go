package ports

import (
	"context"
	"time"

	"github.com/gocourier/internal/domain/notification"
)

type DeliveryRepository interface {
	Save(ctx context.Context, d *notification.Delivery) error
	Update(ctx context.Context, d *notification.Delivery) error
	UpdateIfStatus(ctx context.Context, d *notification.Delivery, expected notification.Status) (bool, error)
	FindByID(ctx context.Context, id string) (*notification.Delivery, error)
	FindByIdempotencyKey(ctx context.Context, key string, ch notification.Channel) (*notification.Delivery, error)
	RecordAttempt(ctx context.Context, deliveryID string, attempt notification.Attempt) error
}

type IdempotencyRepository interface {
	Register(ctx context.Context, key string, ch notification.Channel, deliveryID string, expiresAt time.Time) (bool, error)
}

type OutboxRepository interface {
	Enqueue(ctx context.Context, deliveryID string, subject string, payload []byte, headers map[string]string) error
	FetchPending(ctx context.Context, limit int) ([]OutboxMessage, error)
	CountPending(ctx context.Context) (int64, error)
	MarkPublished(ctx context.Context, id int64) error
	MarkFailed(ctx context.Context, id int64, errMsg string) error
	HasPublishedForDelivery(ctx context.Context, deliveryID string) (bool, error)
}

type OutboxMessage struct {
	ID         int64
	DeliveryID string
	Subject    string
	Payload    []byte
	Headers    map[string]string
	Attempts   int
}

type ScheduledRepository interface {
	Enqueue(ctx context.Context, deliveryID string, scheduledAt time.Time) error
	FetchDue(ctx context.Context, before time.Time, limit int) ([]string, error)
	MarkProcessed(ctx context.Context, deliveryID string) error
}

type AuditRepository interface {
	Append(ctx context.Context, deliveryID string, eventType string, payload map[string]any) error
}
