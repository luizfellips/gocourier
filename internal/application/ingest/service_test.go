package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/pkg/apperrors"
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
