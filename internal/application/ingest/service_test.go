package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/pkg/apperrors"
	"github.com/gocourier/pkg/telemetry"
)

type fakeClock struct{ now time.Time }

func (f fakeClock) Now() time.Time { return f.now }

type fakeDeliveryRepo struct {
	byKey map[string]*notification.Delivery
}

func (f *fakeDeliveryRepo) Save(_ context.Context, d *notification.Delivery) error {
	f.byKey[d.IdempotencyKey] = d
	return nil
}
func (f *fakeDeliveryRepo) Update(_ context.Context, _ *notification.Delivery) error { return nil }
func (f *fakeDeliveryRepo) UpdateIfStatus(_ context.Context, _ *notification.Delivery, _ notification.Status) (bool, error) {
	return true, nil
}
func (f *fakeDeliveryRepo) FindByID(_ context.Context, id string) (*notification.Delivery, error) {
	for _, d := range f.byKey {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, apperrors.ErrNotFound
}
func (f *fakeDeliveryRepo) FindByIdempotencyKey(_ context.Context, key string, _ notification.Channel) (*notification.Delivery, error) {
	d, ok := f.byKey[key]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	return d, nil
}
func (f *fakeDeliveryRepo) RecordAttempt(_ context.Context, _ string, _ notification.Attempt) error {
	return nil
}

type fakeStore struct {
	repo *fakeDeliveryRepo
}

func (s *fakeStore) IngestTransactional(ctx context.Context, d *notification.Delivery, _ time.Time, _ string, _ []byte, _ map[string]string, _ bool) error {
	if _, ok := s.repo.byKey[d.IdempotencyKey]; ok {
		return apperrors.ErrDuplicate
	}
	s.repo.byKey[d.IdempotencyKey] = d
	return nil
}
func (s *fakeStore) TryRegisterIdempotency(_ context.Context, _ string, _ notification.Channel, _ string, _ time.Time) (bool, error) {
	return true, nil
}

type fakeAudit struct{}

func (fakeAudit) Append(_ context.Context, _ string, _ string, _ map[string]any) error { return nil }

func TestIngestDuplicateReturnsExisting(t *testing.T) {
	repo := &fakeDeliveryRepo{byKey: map[string]*notification.Delivery{}}
	store := &fakeStore{repo: repo}
	svc := NewService(store, repo, fakeAudit{}, fakeClock{now: time.Now()}, "notifications", time.Hour)

	req := notification.IngestRequest{
		SchemaVersion:  notification.SchemaVersion,
		IdempotencyKey: "dup-key",
		Channel:        "email",
		Recipient:      json.RawMessage(`{"address":"a@b.com"}`),
		Template:       json.RawMessage(`{"id":"t"}`),
	}
	first, err := svc.Ingest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Ingest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Duplicate {
		t.Fatal("expected duplicate")
	}
	if second.DeliveryID != first.DeliveryID {
		t.Fatal("expected same delivery id")
	}
}

func TestIngestValidationError(t *testing.T) {
	svc := NewService(&fakeStore{repo: &fakeDeliveryRepo{byKey: map[string]*notification.Delivery{}}}, &fakeDeliveryRepo{byKey: map[string]*notification.Delivery{}}, fakeAudit{}, fakeClock{now: time.Now()}, "notifications", time.Hour)
	_, err := svc.Ingest(context.Background(), notification.IngestRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, apperrors.ErrValidation) {
		t.Fatalf("expected validation error: %v", err)
	}
}

func TestIngestDuplicateFromStoreConflict(t *testing.T) {
	repo := &fakeDeliveryRepo{byKey: map[string]*notification.Delivery{}}
	store := &fakeStore{repo: repo}
	svc := NewService(store, repo, fakeAudit{}, fakeClock{now: time.Now()}, "notifications", time.Hour)

	req := notification.IngestRequest{
		SchemaVersion:  notification.SchemaVersion,
		IdempotencyKey: "race-key",
		Channel:        "email",
		Recipient:      json.RawMessage(`{"address":"a@b.com"}`),
		Template:       json.RawMessage(`{"id":"t"}`),
	}
	existing := &notification.Delivery{ID: "existing-id", IdempotencyKey: "race-key", Status: notification.StatusQueued}
	repo.byKey["race-key"] = existing

	resp, err := svc.Ingest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Duplicate || resp.DeliveryID != "existing-id" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestIngestHappyPath(t *testing.T) {
	repo := &fakeDeliveryRepo{byKey: map[string]*notification.Delivery{}}
	store := &fakeStore{repo: repo}
	svc := NewService(store, repo, fakeAudit{}, fakeClock{now: time.Now()}, "notifications", time.Hour)

	req := notification.IngestRequest{
		SchemaVersion:  notification.SchemaVersion,
		IdempotencyKey: "happy-key",
		Channel:        "email",
		Recipient:      json.RawMessage(`{"address":"a@b.com"}`),
		Template:       json.RawMessage(`{"id":"t"}`),
	}
	resp, err := svc.Ingest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.DeliveryID == "" || resp.Duplicate {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Status != notification.StatusQueued {
		t.Fatalf("expected queued, got %s", resp.Status)
	}
}

func TestIngestExistingKeyBeforeStore(t *testing.T) {
	repo := &fakeDeliveryRepo{byKey: map[string]*notification.Delivery{}}
	store := &fakeStore{repo: repo}
	now := time.Now()
	repo.byKey["early-key"] = &notification.Delivery{
		ID: "existing", IdempotencyKey: "early-key", Status: notification.StatusQueued,
		Channel: notification.ChannelEmail, CreatedAt: now, UpdatedAt: now,
	}
	svc := NewService(store, repo, fakeAudit{}, fakeClock{now: now}, "notifications", time.Hour)

	resp, err := svc.Ingest(context.Background(), notification.IngestRequest{
		SchemaVersion:  notification.SchemaVersion,
		IdempotencyKey: "early-key",
		Channel:        "email",
		Recipient:      json.RawMessage(`{"address":"a@b.com"}`),
		Template:       json.RawMessage(`{"id":"t"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Duplicate || resp.DeliveryID != "existing" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestIngestScheduled(t *testing.T) {
	repo := &fakeDeliveryRepo{byKey: map[string]*notification.Delivery{}}
	store := &schedulingStore{repo: repo}
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	svc := NewService(store, repo, fakeAudit{}, fakeClock{now: now}, "notifications", time.Hour)

	resp, err := svc.Ingest(context.Background(), notification.IngestRequest{
		SchemaVersion:  notification.SchemaVersion,
		IdempotencyKey: "sched-key",
		Channel:        "email",
		Recipient:      json.RawMessage(`{"address":"a@b.com"}`),
		Template:       json.RawMessage(`{"id":"t"}`),
		ScheduledAt:    &future,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !store.scheduled {
		t.Fatal("expected scheduled ingest")
	}
	if resp.Status != notification.StatusPending {
		t.Fatalf("expected pending, got %s", resp.Status)
	}
}

type schedulingStore struct {
	repo      *fakeDeliveryRepo
	scheduled bool
}

func (s *schedulingStore) IngestTransactional(_ context.Context, d *notification.Delivery, _ time.Time, _ string, _ []byte, _ map[string]string, schedule bool) error {
	s.scheduled = schedule
	s.repo.byKey[d.IdempotencyKey] = d
	return nil
}
func (s *schedulingStore) TryRegisterIdempotency(_ context.Context, _ string, _ notification.Channel, _ string, _ time.Time) (bool, error) {
	return true, nil
}

type errDeliveryRepo struct {
	fakeDeliveryRepo
	err error
}

func (r *errDeliveryRepo) FindByIdempotencyKey(_ context.Context, _ string, _ notification.Channel) (*notification.Delivery, error) {
	return nil, r.err
}

func TestIngestWithMetrics(t *testing.T) {
	ctx := context.Background()
	p, err := telemetry.Init(ctx, telemetry.Config{ServiceName: "test", EnableOTLP: false})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Shutdown(ctx) })

	repo := &fakeDeliveryRepo{byKey: map[string]*notification.Delivery{}}
	store := &fakeStore{repo: repo}
	svc := NewService(store, repo, fakeAudit{}, fakeClock{now: time.Now()}, "notifications", time.Hour)
	resp, err := svc.Ingest(ctx, notification.IngestRequest{
		SchemaVersion:  notification.SchemaVersion,
		IdempotencyKey: "metrics-key",
		Channel:        "email",
		Recipient:      json.RawMessage(`{"address":"a@b.com"}`),
		Template:       json.RawMessage(`{"id":"t"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.DeliveryID == "" {
		t.Fatal("expected delivery id")
	}
}

func TestIngestRepoLookupError(t *testing.T) {
	repo := &errDeliveryRepo{
		fakeDeliveryRepo: fakeDeliveryRepo{byKey: map[string]*notification.Delivery{}},
		err:              errors.New("db unavailable"),
	}
	svc := NewService(&fakeStore{repo: &repo.fakeDeliveryRepo}, repo, fakeAudit{}, fakeClock{now: time.Now()}, "notifications", time.Hour)
	_, err := svc.Ingest(context.Background(), notification.IngestRequest{
		SchemaVersion:  notification.SchemaVersion,
		IdempotencyKey: "key",
		Channel:        "email",
		Recipient:      json.RawMessage(`{"address":"a@b.com"}`),
		Template:       json.RawMessage(`{"id":"t"}`),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestIngestDuplicateRecordsMetrics(t *testing.T) {
	ctx := context.Background()
	p, err := telemetry.Init(ctx, telemetry.Config{ServiceName: "test", EnableOTLP: false})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Shutdown(ctx) })

	repo := &fakeDeliveryRepo{byKey: map[string]*notification.Delivery{}}
	store := &fakeStore{repo: repo}
	svc := NewService(store, repo, fakeAudit{}, fakeClock{now: time.Now()}, "notifications", time.Hour)
	req := notification.IngestRequest{
		SchemaVersion:  notification.SchemaVersion,
		IdempotencyKey: "dup-metrics",
		Channel:        "email",
		Recipient:      json.RawMessage(`{"address":"a@b.com"}`),
		Template:       json.RawMessage(`{"id":"t"}`),
	}
	if _, err := svc.Ingest(ctx, req); err != nil {
		t.Fatal(err)
	}
	resp, err := svc.Ingest(ctx, req)
	if err != nil || !resp.Duplicate {
		t.Fatalf("expected duplicate, got %+v err=%v", resp, err)
	}
}
