package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/apperrors"
	"github.com/gocourier/pkg/logger"
)

type memDeliveryRepo struct {
	deliveries map[string]*notification.Delivery
}

func newMemDeliveryRepo() *memDeliveryRepo {
	return &memDeliveryRepo{deliveries: map[string]*notification.Delivery{}}
}

func (m *memDeliveryRepo) Save(_ context.Context, d *notification.Delivery) error {
	m.deliveries[d.ID] = d
	return nil
}
func (m *memDeliveryRepo) Update(_ context.Context, d *notification.Delivery) error {
	m.deliveries[d.ID] = d
	return nil
}
func (m *memDeliveryRepo) UpdateIfStatus(_ context.Context, d *notification.Delivery, expected notification.Status) (bool, error) {
	cur, ok := m.deliveries[d.ID]
	if !ok {
		return false, apperrors.ErrNotFound
	}
	if cur.Status != expected {
		return false, nil
	}
	m.deliveries[d.ID] = d
	return true, nil
}
func (m *memDeliveryRepo) FindByID(_ context.Context, id string) (*notification.Delivery, error) {
	d, ok := m.deliveries[id]
	if !ok {
		return nil, apperrors.ErrNotFound
	}
	cp := *d
	return &cp, nil
}
func (m *memDeliveryRepo) FindByIdempotencyKey(_ context.Context, _ string, _ notification.Channel) (*notification.Delivery, error) {
	return nil, apperrors.ErrNotFound
}
func (m *memDeliveryRepo) RecordAttempt(_ context.Context, _ string, _ notification.Attempt) error {
	return nil
}

type memAudit struct{ events []string }

func (a *memAudit) Append(_ context.Context, _ string, eventType string, _ map[string]any) error {
	a.events = append(a.events, eventType)
	return nil
}

type memBroker struct {
	published []string
}

func (b *memBroker) Publish(_ context.Context, subject string, _ []byte, _ map[string]string) error {
	b.published = append(b.published, subject)
	return nil
}
func (b *memBroker) Subscribe(_ context.Context, _, _ string, _ ports.MessageHandler) error { return nil }
func (b *memBroker) EnsureStreams(_ context.Context) error                                   { return nil }
func (b *memBroker) Close() error                                                            { return nil }

type stubProvider struct {
	channel notification.Channel
	err     error
	calls   int
}

func (p *stubProvider) Channel() notification.Channel { return p.channel }
func (p *stubProvider) Send(_ context.Context, _ *notification.Delivery) (ports.ProviderResult, error) {
	p.calls++
	if p.err != nil {
		return ports.ProviderResult{}, p.err
	}
	return ports.ProviderResult{ProviderMessageID: "msg-1"}, nil
}

type fixedClock struct{ now time.Time }

func (f fixedClock) Now() time.Time { return f.now }

func newTestDispatch(t *testing.T, provider ports.ChannelProvider) (*Service, *memDeliveryRepo, *memAudit, *memBroker) {
	t.Helper()
	repo := newMemDeliveryRepo()
	audit := &memAudit{}
	broker := &memBroker{}
	now := time.Now().UTC()
	svc := NewService(
		repo, audit, broker, []ports.ChannelProvider{provider}, fixedClock{now: now},
		logger.New("error"), Config{
			MaxAttempts: 3, RetryBase: time.Millisecond, RetryMax: time.Second,
			StreamPrefix: "notifications", CBThreshold: 100, CBWindow: time.Second, CBCooldown: time.Second,
		},
	)
	return svc, repo, audit, broker
}

func seedDelivery(repo *memDeliveryRepo, id string, status notification.Status, recipient string) {
	now := time.Now().UTC()
	repo.deliveries[id] = &notification.Delivery{
		ID:        id,
		Channel:   notification.ChannelEmail,
		Priority:  notification.PriorityNormal,
		Recipient: json.RawMessage(fmt.Sprintf(`{"address":"%s"}`, recipient)),
		Template:  json.RawMessage(`{"id":"t"}`),
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestDispatchSkipsSucceeded(t *testing.T) {
	provider := &stubProvider{channel: notification.ChannelEmail}
	svc, repo, _, _ := newTestDispatch(t, provider)
	seedDelivery(repo, "d1", notification.StatusSucceeded, "a@b.com")

	if err := svc.Dispatch(context.Background(), "d1"); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 0 {
		t.Fatal("provider should not be called")
	}
}

func TestDispatchPermanentFailureMovesToDLQ(t *testing.T) {
	provider := &stubProvider{
		channel: notification.ChannelEmail,
		err:     fmt.Errorf("%w: bad", apperrors.ErrPermanent),
	}
	svc, repo, audit, broker := newTestDispatch(t, provider)
	seedDelivery(repo, "d1", notification.StatusQueued, "fail-permanent@example.com")

	if err := svc.Dispatch(context.Background(), "d1"); err != nil {
		t.Fatal(err)
	}
	d := repo.deliveries["d1"]
	if d.Status != notification.StatusDLQ {
		t.Fatalf("expected dlq, got %s", d.Status)
	}
	if len(broker.published) == 0 {
		t.Fatal("expected dlq publish")
	}
	if len(audit.events) == 0 {
		t.Fatal("expected audit event")
	}
}

func TestDispatchTransientFailureReturnsErrTransient(t *testing.T) {
	provider := &stubProvider{
		channel: notification.ChannelEmail,
		err:     fmt.Errorf("%w: down", apperrors.ErrTransient),
	}
	svc, repo, _, _ := newTestDispatch(t, provider)
	seedDelivery(repo, "d1", notification.StatusQueued, "fail-transient@example.com")

	err := svc.Dispatch(context.Background(), "d1")
	if !apperrors.IsTransient(err) {
		t.Fatalf("expected transient: %v", err)
	}
	if repo.deliveries["d1"].Status != notification.StatusRetrying {
		t.Fatalf("expected retrying, got %s", repo.deliveries["d1"].Status)
	}
}

func TestDispatchConcurrentSkip(t *testing.T) {
	provider := &stubProvider{channel: notification.ChannelEmail}
	repo := &concurrentSkipRepo{inner: newMemDeliveryRepo()}
	audit := &memAudit{}
	broker := &memBroker{}
	now := time.Now().UTC()
	svc := NewService(
		repo, audit, broker, []ports.ChannelProvider{provider}, fixedClock{now: now},
		logger.New("error"), Config{
			MaxAttempts: 3, RetryBase: time.Millisecond, RetryMax: time.Second,
			StreamPrefix: "notifications", CBThreshold: 100, CBWindow: time.Second, CBCooldown: time.Second,
		},
	)
	seedDelivery(repo.inner, "d1", notification.StatusQueued, "a@b.com")

	if err := svc.Dispatch(context.Background(), "d1"); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 0 {
		t.Fatal("concurrent dispatch should skip")
	}
}

type concurrentSkipRepo struct {
	inner *memDeliveryRepo
}

func (r *concurrentSkipRepo) Save(ctx context.Context, d *notification.Delivery) error {
	return r.inner.Save(ctx, d)
}
func (r *concurrentSkipRepo) Update(ctx context.Context, d *notification.Delivery) error {
	return r.inner.Update(ctx, d)
}
func (r *concurrentSkipRepo) UpdateIfStatus(_ context.Context, _ *notification.Delivery, _ notification.Status) (bool, error) {
	return false, nil
}
func (r *concurrentSkipRepo) FindByID(ctx context.Context, id string) (*notification.Delivery, error) {
	return r.inner.FindByID(ctx, id)
}
func (r *concurrentSkipRepo) FindByIdempotencyKey(ctx context.Context, key string, ch notification.Channel) (*notification.Delivery, error) {
	return r.inner.FindByIdempotencyKey(ctx, key, ch)
}
func (r *concurrentSkipRepo) RecordAttempt(ctx context.Context, deliveryID string, attempt notification.Attempt) error {
	return r.inner.RecordAttempt(ctx, deliveryID, attempt)
}

func TestHandleMessageMissingDeliveryID(t *testing.T) {
	provider := &stubProvider{channel: notification.ChannelEmail}
	svc, _, _, _ := newTestDispatch(t, provider)
	err := svc.HandleMessage(context.Background(), "", []byte(`{}`), nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
