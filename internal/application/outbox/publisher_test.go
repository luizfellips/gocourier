package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/logger"
)

type memOutboxRepo struct {
	msgs      []ports.OutboxMessage
	published map[int64]bool
	deliveryPublished map[string]bool
	nextID    int64
}

func newMemOutboxRepo() *memOutboxRepo {
	return &memOutboxRepo{
		published:         map[int64]bool{},
		deliveryPublished: map[string]bool{},
		nextID:            1,
	}
}

func (r *memOutboxRepo) Enqueue(_ context.Context, deliveryID, subject string, payload []byte, headers map[string]string) error {
	r.msgs = append(r.msgs, ports.OutboxMessage{
		ID: r.nextID, DeliveryID: deliveryID, Subject: subject, Payload: payload, Headers: headers,
	})
	r.nextID++
	return nil
}

func (r *memOutboxRepo) FetchPending(_ context.Context, limit int) ([]ports.OutboxMessage, error) {
	var pending []ports.OutboxMessage
	for _, m := range r.msgs {
		if !r.published[m.ID] {
			pending = append(pending, m)
			if len(pending) >= limit {
				break
			}
		}
	}
	return pending, nil
}

func (r *memOutboxRepo) CountPending(_ context.Context) (int64, error) {
	var count int64
	for _, m := range r.msgs {
		if !r.published[m.ID] {
			count++
		}
	}
	return count, nil
}

func (r *memOutboxRepo) MarkPublished(_ context.Context, id int64) error {
	r.published[id] = true
	for _, m := range r.msgs {
		if m.ID == id {
			r.deliveryPublished[m.DeliveryID] = true
		}
	}
	return nil
}

func (r *memOutboxRepo) MarkFailed(_ context.Context, id int64, _ string) error {
	return nil
}

func (r *memOutboxRepo) HasPublishedForDelivery(_ context.Context, deliveryID string) (bool, error) {
	return r.deliveryPublished[deliveryID], nil
}

type memBroker struct {
	count   int
	failNext bool
}

func (b *memBroker) Publish(_ context.Context, _ string, _ []byte, _ map[string]string) error {
	if b.failNext {
		b.failNext = false
		return errors.New("broker down")
	}
	b.count++
	return nil
}
func (b *memBroker) Subscribe(_ context.Context, _, _ string, _ ports.MessageHandler) error { return nil }
func (b *memBroker) EnsureStreams(_ context.Context) error                                   { return nil }
func (b *memBroker) Close() error                                                            { return nil }

func TestPublisherFlushOncePublishesPending(t *testing.T) {
	repo := newMemOutboxRepo()
	_ = repo.Enqueue(context.Background(), "d1", "notifications.email.normal", []byte(`{}`), nil)
	broker := &memBroker{}
	pub := NewPublisher(repo, broker, logger.New("error"), 0, 10)

	if err := pub.FlushOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if broker.count != 1 {
		t.Fatalf("expected 1 publish, got %d", broker.count)
	}
}

func TestPublisherSkipsAlreadyPublishedDelivery(t *testing.T) {
	repo := newMemOutboxRepo()
	_ = repo.Enqueue(context.Background(), "d1", "s", []byte(`{}`), nil)
	_ = repo.Enqueue(context.Background(), "d1", "s", []byte(`{}`), nil)
	repo.deliveryPublished["d1"] = true

	broker := &memBroker{}
	pub := NewPublisher(repo, broker, logger.New("error"), 0, 10)
	if err := pub.FlushOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if broker.count != 0 {
		t.Fatalf("expected no publish, got %d", broker.count)
	}
}

func TestPublisherContinuesOnBrokerFailure(t *testing.T) {
	repo := newMemOutboxRepo()
	_ = repo.Enqueue(context.Background(), "d1", "s", []byte(`{}`), nil)
	_ = repo.Enqueue(context.Background(), "d2", "s", []byte(`{}`), nil)
	broker := &memBroker{failNext: true}
	pub := NewPublisher(repo, broker, logger.New("error"), 0, 10)

	if err := pub.FlushOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if broker.count != 1 {
		t.Fatalf("expected 1 successful publish, got %d", broker.count)
	}
}

func TestPublisherRunStopsOnCancel(t *testing.T) {
	repo := newMemOutboxRepo()
	broker := &memBroker{}
	pub := NewPublisher(repo, broker, logger.New("error"), 10*time.Millisecond, 10)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- pub.Run(ctx) }()

	time.Sleep(25 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancel, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after cancel")
	}
}
