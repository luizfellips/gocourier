package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Publisher struct {
	repo     ports.OutboxRepository
	broker   ports.MessageBroker
	log      *slog.Logger
	interval time.Duration
	batch    int
}

func NewPublisher(repo ports.OutboxRepository, broker ports.MessageBroker, log *slog.Logger, interval time.Duration, batch int) *Publisher {
	return &Publisher{
		repo:     repo,
		broker:   broker,
		log:      log,
		interval: interval,
		batch:    batch,
	}
}

func (p *Publisher) Run(ctx context.Context) error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := p.flush(ctx); err != nil {
				p.log.Error("outbox flush failed", "error", err)
			}
		}
	}
}

func (p *Publisher) FlushOnce(ctx context.Context) error {
	return p.flush(ctx)
}

func (p *Publisher) flush(ctx context.Context) error {
	ctx, span := telemetry.StartSpan(ctx, "outbox.flush")
	defer span.End()

	if count, err := p.repo.CountPending(ctx); err == nil {
		if m := telemetry.MetricsGlobal(); m != nil {
			m.QueueDepth.Set(float64(count))
		}
	}

	msgs, err := p.repo.FetchPending(ctx, p.batch)
	if err != nil {
		return err
	}
	for _, m := range msgs {
		published, err := p.repo.HasPublishedForDelivery(ctx, m.DeliveryID)
		if err != nil {
			p.log.Warn("outbox idempotency check failed", "delivery_id", m.DeliveryID, "error", err)
			continue
		}
		if published {
			if err := p.repo.MarkPublished(ctx, m.ID); err != nil {
				p.log.Error("mark duplicate outbox published failed", "id", m.ID, "error", err)
			}
			continue
		}
		start := time.Now()
		if m.Headers == nil {
			m.Headers = map[string]string{}
		}
		telemetry.InjectTrace(ctx, m.Headers)
		if err := p.broker.Publish(ctx, m.Subject, m.Payload, m.Headers); err != nil {
			_ = p.repo.MarkFailed(ctx, m.ID, err.Error())
			p.log.Warn("outbox publish failed", "id", m.ID, "error", err)
			continue
		}
		if m := telemetry.MetricsGlobal(); m != nil {
			m.OutboxPublishDuration.Observe(time.Since(start).Seconds())
		}
		if err := p.repo.MarkPublished(ctx, m.ID); err != nil {
			p.log.Error("mark outbox published failed", "id", m.ID, "error", err)
		}
		span.AddEvent("published", trace.WithAttributes(attribute.String("delivery_id", m.DeliveryID)))
	}
	return nil
}
