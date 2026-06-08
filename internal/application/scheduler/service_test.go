package scheduler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/apperrors"
	"github.com/gocourier/pkg/logger"
)

type schedClock struct{ now time.Time }

func (c schedClock) Now() time.Time { return c.now }

type memScheduledRepo struct {
	due []string
}

func (r *memScheduledRepo) Enqueue(_ context.Context, deliveryID string, _ time.Time) error {
	r.due = append(r.due, deliveryID)
	return nil
}
func (r *memScheduledRepo) FetchDue(_ context.Context, _ time.Time, limit int) ([]string, error) {
	if limit > len(r.due) {
		limit = len(r.due)
	}
	return append([]string(nil), r.due[:limit]...), nil
}
func (r *memScheduledRepo) MarkProcessed(_ context.Context, deliveryID string) error {
	for i, id := range r.due {
		if id == deliveryID {
			r.due = append(r.due[:i], r.due[i+1:]...)
			return nil
		}
	}
	return apperrors.ErrNotFound
}

type schedDeliveryRepo struct {
	deliveries map[string]*notification.Delivery
}

func (r *schedDeliveryRepo) Save(_ context.Context, d *notification.Delivery) error {
	r.deliveries[d.ID] = d
	return nil
}
func (r *schedDeliveryRepo) Update(_ context.Context, d *notification.Delivery) error {
	r.deliveries[d.ID] = d
	return nil
}
func (r *schedDeliveryRepo) UpdateIfStatus(_ context.Context, d *notification.Delivery, expected notification.Status) (bool, error) {
	cur, ok := r.deliveries[d.ID]
	if !ok || cur.Status != expected {
		return false, nil
	}
	r.deliveries[d.ID] = d
	return true, nil
}
func (r *schedDeliveryRepo) FindByID(_ context.Context, id string) (*notification.Delivery, error) {
	d, ok := r.deliveries[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	cp := *d
	return &cp, nil
}
func (r *schedDeliveryRepo) FindByIdempotencyKey(_ context.Context, _ string, _ notification.Channel) (*notification.Delivery, error) {
	return nil, apperrors.ErrNotFound
}
func (r *schedDeliveryRepo) RecordAttempt(_ context.Context, _ string, _ notification.Attempt) error {
	return nil
}

type schedOutboxRepo struct {
	msgs []ports.OutboxMessage
}

func (r *schedOutboxRepo) Enqueue(_ context.Context, deliveryID, subject string, payload []byte, headers map[string]string) error {
	r.msgs = append(r.msgs, ports.OutboxMessage{DeliveryID: deliveryID, Subject: subject, Payload: payload, Headers: headers})
	return nil
}
func (r *schedOutboxRepo) FetchPending(_ context.Context, _ int) ([]ports.OutboxMessage, error) {
	return nil, nil
}
func (r *schedOutboxRepo) CountPending(_ context.Context) (int64, error) { return 0, nil }
func (r *schedOutboxRepo) MarkPublished(_ context.Context, _ int64) error { return nil }
func (r *schedOutboxRepo) MarkFailed(_ context.Context, _ int64, _ string) error { return nil }
func (r *schedOutboxRepo) HasPublishedForDelivery(_ context.Context, _ string) (bool, error) {
	return false, nil
}

type schedAudit struct{}

func (schedAudit) Append(_ context.Context, _ string, _ string, _ map[string]any) error { return nil }

func TestSchedulerProcessesDueDelivery(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	scheduledAt := now.Add(-time.Minute)
	deliveries := map[string]*notification.Delivery{
		"d1": {
			ID: "d1", Channel: notification.ChannelEmail, Priority: notification.PriorityNormal,
			Recipient: json.RawMessage(`{"address":"a@b.com"}`), Template: json.RawMessage(`{"id":"t"}`),
			Status: notification.StatusPending, ScheduledAt: &scheduledAt,
			CreatedAt: now, UpdatedAt: now,
		},
	}
	scheduled := &memScheduledRepo{due: []string{"d1"}}
	outbox := &schedOutboxRepo{}
	svc := NewService(
		scheduled,
		&schedDeliveryRepo{deliveries: deliveries},
		outbox,
		schedAudit{},
		schedClock{now: now},
		logger.New("error"),
		"notifications",
		10*time.Millisecond,
		10,
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- svc.Run(ctx) }()

	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil && err != context.Canceled {
			t.Fatalf("run error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop")
	}

	if deliveries["d1"].Status != notification.StatusQueued {
		t.Fatalf("expected queued, got %s", deliveries["d1"].Status)
	}
	if len(outbox.msgs) == 0 {
		t.Fatal("expected outbox enqueue")
	}
}
