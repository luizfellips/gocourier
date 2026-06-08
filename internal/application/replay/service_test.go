package replay

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gocourier/internal/application/dispatch"
	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/apperrors"
	"github.com/gocourier/pkg/logger"
)

type replayRepo struct {
	d *notification.Delivery
}

func (r *replayRepo) Save(_ context.Context, _ *notification.Delivery) error   { return nil }
func (r *replayRepo) Update(_ context.Context, d *notification.Delivery) error { r.d = d; return nil }
func (r *replayRepo) UpdateIfStatus(_ context.Context, _ *notification.Delivery, _ notification.Status) (bool, error) {
	return true, nil
}
func (r *replayRepo) FindByID(_ context.Context, id string) (*notification.Delivery, error) {
	if r.d == nil || r.d.ID != id {
		return nil, apperrors.ErrNotFound
	}
	cp := *r.d
	return &cp, nil
}
func (r *replayRepo) FindByIdempotencyKey(_ context.Context, _ string, _ notification.Channel) (*notification.Delivery, error) {
	return nil, apperrors.ErrNotFound
}
func (r *replayRepo) RecordAttempt(_ context.Context, _ string, _ notification.Attempt) error { return nil }

type replayBroker struct{}

func (replayBroker) Publish(_ context.Context, _ string, _ []byte, _ map[string]string) error { return nil }
func (replayBroker) Subscribe(_ context.Context, _, _ string, _ ports.MessageHandler) error  { return nil }
func (replayBroker) EnsureStreams(_ context.Context) error                                     { return nil }
func (replayBroker) Close() error                                                              { return nil }

type replayAudit struct{}

func (replayAudit) Append(_ context.Context, _ string, _ string, _ map[string]any) error { return nil }

type replayProvider struct{}

func (replayProvider) Channel() notification.Channel { return notification.ChannelEmail }
func (replayProvider) Send(_ context.Context, _ *notification.Delivery) (ports.ProviderResult, error) {
	return ports.ProviderResult{ProviderMessageID: "x"}, nil
}

type replayClock struct{ now time.Time }

func (c replayClock) Now() time.Time { return c.now }

func TestReplayServiceDelegates(t *testing.T) {
	now := time.Now().UTC()
	repo := &replayRepo{d: &notification.Delivery{
		ID: "d1", Channel: notification.ChannelEmail, Priority: notification.PriorityNormal,
		Recipient: json.RawMessage(`{"address":"a@b.com"}`), Template: json.RawMessage(`{"id":"t"}`),
		Status: notification.StatusDLQ, CreatedAt: now, UpdatedAt: now,
	}}
	dispatchSvc := dispatch.NewService(
		repo, replayAudit{}, replayBroker{}, []ports.ChannelProvider{replayProvider{}},
		replayClock{now: now}, logger.New("error"),
		dispatch.Config{
			MaxAttempts: 3, RetryBase: time.Millisecond, RetryMax: time.Second,
			StreamPrefix: "notifications", CBThreshold: 100, CBWindow: time.Second, CBCooldown: time.Second,
		},
	)
	svc := NewService(dispatchSvc)

	if err := svc.Replay(context.Background(), "d1"); err != nil {
		t.Fatal(err)
	}
	if repo.d.Status != notification.StatusQueued {
		t.Fatalf("expected queued, got %s", repo.d.Status)
	}
}
