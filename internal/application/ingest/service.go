package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/internal/domain/routing"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/apperrors"
	"github.com/gocourier/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

type Service struct {
	store        TransactionalStore
	deliveries   ports.DeliveryRepository
	audit        ports.AuditRepository
	clock        ports.Clock
	streamPrefix string
	idempotencyTTL time.Duration
}

type TransactionalStore interface {
	IngestTransactional(
		ctx context.Context,
		d *notification.Delivery,
		expiresAt time.Time,
		outboxSubject string,
		outboxPayload []byte,
		outboxHeaders map[string]string,
		schedule bool,
	) error
	TryRegisterIdempotency(ctx context.Context, key string, ch notification.Channel, deliveryID string, expiresAt time.Time) (bool, error)
}

func NewService(
	store TransactionalStore,
	deliveries ports.DeliveryRepository,
	audit ports.AuditRepository,
	clock ports.Clock,
	streamPrefix string,
	idempotencyTTL time.Duration,
) *Service {
	return &Service{
		store:          store,
		deliveries:     deliveries,
		audit:          audit,
		clock:          clock,
		streamPrefix:   streamPrefix,
		idempotencyTTL: idempotencyTTL,
	}
}

func (s *Service) Ingest(ctx context.Context, req notification.IngestRequest) (*notification.IngestResponse, error) {
	ctx, span := telemetry.StartSpan(ctx, "notification.ingest",
		attribute.String("channel", req.Channel),
		attribute.String("idempotency_key", req.IdempotencyKey),
	)
	defer span.End()

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", apperrors.ErrValidation, err)
	}

	ch, _ := notification.ParseChannel(req.Channel)
	existing, err := s.deliveries.FindByIdempotencyKey(ctx, req.IdempotencyKey, ch)
	if err == nil {
		recordIngestMetric(ch, true)
		return &notification.IngestResponse{
			DeliveryID: existing.ID,
			Status:     existing.Status,
			Duplicate:  true,
		}, nil
	}
	if !errors.Is(err, apperrors.ErrNotFound) {
		return nil, err
	}

	now := s.clock.Now()
	delivery, err := notification.NewDeliveryFromRequest(req, now)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", apperrors.ErrValidation, err)
	}

	schedule := delivery.ShouldSchedule(now)
	if !schedule {
		if err := delivery.Queue(now); err != nil {
			return nil, err
		}
	}

	outboxPayload, err := json.Marshal(map[string]string{"delivery_id": delivery.ID})
	if err != nil {
		return nil, err
	}
	subject := routing.NotificationSubject(s.streamPrefix, delivery.Channel, delivery.Priority)
	headers := map[string]string{
		"delivery_id":     delivery.ID,
		"correlation_id":  delivery.CorrelationID,
		"idempotency_key": delivery.IdempotencyKey,
		"channel":         string(delivery.Channel),
	}
	telemetry.InjectTrace(ctx, headers)

	expiresAt := now.Add(s.idempotencyTTL)
	if err := s.store.IngestTransactional(ctx, delivery, expiresAt, subject, outboxPayload, headers, schedule); err != nil {
		if errors.Is(err, apperrors.ErrDuplicate) {
			existing, findErr := s.deliveries.FindByIdempotencyKey(ctx, req.IdempotencyKey, ch)
			if findErr != nil {
				return nil, findErr
			}
			recordIngestMetric(ch, true)
			return &notification.IngestResponse{
				DeliveryID: existing.ID,
				Status:     existing.Status,
				Duplicate:  true,
			}, nil
		}
		return nil, err
	}

	recordIngestMetric(ch, false)
	span.SetAttributes(attribute.String("delivery_id", delivery.ID))

	return &notification.IngestResponse{
		DeliveryID: delivery.ID,
		Status:     delivery.Status,
	}, nil
}

func recordIngestMetric(ch notification.Channel, duplicate bool) {
	m := telemetry.MetricsGlobal()
	if m == nil {
		return
	}
	dup := "false"
	if duplicate {
		dup = "true"
	}
	m.NotificationsReceived.WithLabelValues(string(ch), dup).Inc()
}
