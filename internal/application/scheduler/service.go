package scheduler

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/internal/domain/routing"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Service struct {
	scheduled  ports.ScheduledRepository
	deliveries ports.DeliveryRepository
	outbox     ports.OutboxRepository
	audit      ports.AuditRepository
	clock      ports.Clock
	log        *slog.Logger
	streamPrefix string
	interval   time.Duration
	batch      int
}

func NewService(
	scheduled ports.ScheduledRepository,
	deliveries ports.DeliveryRepository,
	outbox ports.OutboxRepository,
	audit ports.AuditRepository,
	clock ports.Clock,
	log *slog.Logger,
	streamPrefix string,
	interval time.Duration,
	batch int,
) *Service {
	return &Service{
		scheduled:    scheduled,
		deliveries:   deliveries,
		outbox:       outbox,
		audit:        audit,
		clock:        clock,
		log:          log,
		streamPrefix: streamPrefix,
		interval:     interval,
		batch:        batch,
	}
}

func (s *Service) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.processDue(ctx); err != nil {
				s.log.Error("scheduler tick failed", "error", err)
			}
		}
	}
}

func (s *Service) processDue(ctx context.Context) error {
	ctx, span := telemetry.StartSpan(ctx, "notification.schedule")
	defer span.End()

	now := s.clock.Now()
	ids, err := s.scheduled.FetchDue(ctx, now, s.batch)
	if err != nil {
		return err
	}
	for _, id := range ids {
		d, err := s.deliveries.FindByID(ctx, id)
		if err != nil {
			s.log.Warn("scheduled delivery not found", "id", id, "error", err)
			continue
		}
		if err := d.Queue(now); err != nil {
			s.log.Warn("queue scheduled delivery failed", "id", id, "error", err)
			continue
		}
		if err := s.deliveries.Update(ctx, d); err != nil {
			return err
		}

		payload, _ := json.Marshal(map[string]string{"delivery_id": d.ID})
		subject := routing.NotificationSubject(s.streamPrefix, d.Channel, d.Priority)
		headers := map[string]string{
			"delivery_id":     d.ID,
			"correlation_id":  d.CorrelationID,
			"idempotency_key": d.IdempotencyKey,
			"channel":         string(d.Channel),
		}
		telemetry.InjectTrace(ctx, headers)
		if err := s.outbox.Enqueue(ctx, d.ID, subject, payload, headers); err != nil {
			return err
		}
		if err := s.scheduled.MarkProcessed(ctx, id); err != nil {
			return err
		}
		_ = s.audit.Append(ctx, d.ID, string(notification.EventQueued), map[string]any{
			"scheduled": true,
		})
		span.AddEvent("scheduled_delivery_queued", trace.WithAttributes(attribute.String("delivery_id", d.ID)))
	}
	return nil
}
