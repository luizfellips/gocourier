package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gocourier/internal/domain/notification"
	"github.com/gocourier/internal/domain/routing"
	"github.com/gocourier/internal/ports"
	"github.com/gocourier/pkg/apperrors"
	"github.com/gocourier/pkg/circuitbreaker"
	"github.com/gocourier/pkg/retry"
	"github.com/gocourier/pkg/telemetry"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type Service struct {
	deliveries   ports.DeliveryRepository
	audit        ports.AuditRepository
	broker       ports.MessageBroker
	providers    map[notification.Channel]ports.ChannelProvider
	breakers     map[notification.Channel]*circuitbreaker.Breaker
	clock        ports.Clock
	log          *slog.Logger
	maxAttempts  int
	retryBase    time.Duration
	retryMax     time.Duration
	streamPrefix string
}

func NewService(
	deliveries ports.DeliveryRepository,
	audit ports.AuditRepository,
	broker ports.MessageBroker,
	providers []ports.ChannelProvider,
	clock ports.Clock,
	log *slog.Logger,
	cfg Config,
) *Service {
	pm := make(map[notification.Channel]ports.ChannelProvider)
	bm := make(map[notification.Channel]*circuitbreaker.Breaker)
	for _, p := range providers {
		ch := p.Channel()
		pm[ch] = p
		bm[ch] = circuitbreaker.New(cfg.CBThreshold, cfg.CBWindow, cfg.CBCooldown)
	}
	return &Service{
		deliveries:   deliveries,
		audit:        audit,
		broker:       broker,
		providers:    pm,
		breakers:     bm,
		clock:        clock,
		log:          log,
		maxAttempts:  cfg.MaxAttempts,
		retryBase:    cfg.RetryBase,
		retryMax:     cfg.RetryMax,
		streamPrefix: cfg.StreamPrefix,
	}
}

func (s *Service) HandleMessage(ctx context.Context, subject string, data []byte, headers map[string]string) error {
	ctx = telemetry.ExtractTrace(ctx, headers)
	ctx, span := telemetry.StartSpan(ctx, "notification.dispatch",
		attribute.String("subject", subject),
	)
	defer span.End()

	var envelope struct {
		DeliveryID string `json:"delivery_id"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}
	deliveryID := envelope.DeliveryID
	if deliveryID == "" && headers != nil {
		deliveryID = headers["delivery_id"]
	}
	if deliveryID == "" {
		return fmt.Errorf("missing delivery_id")
	}
	span.SetAttributes(attribute.String("delivery_id", deliveryID))
	return s.Dispatch(ctx, deliveryID)
}

func (s *Service) Dispatch(ctx context.Context, deliveryID string) error {
	if m := telemetry.MetricsGlobal(); m != nil {
		m.WorkerActiveJobs.Inc()
		defer m.WorkerActiveJobs.Dec()
	}

	start := time.Now()
	d, err := s.deliveries.FindByID(ctx, deliveryID)
	if err != nil {
		return err
	}

	if d.Status == notification.StatusSucceeded {
		s.log.Info("skipping already succeeded delivery", "delivery_id", deliveryID)
		return nil
	}
	if d.Status == notification.StatusDLQ {
		return nil
	}

	now := s.clock.Now()
	prevStatus := d.Status
	if err := d.StartProcessing(now); err != nil {
		if d.Status == notification.StatusSucceeded {
			return nil
		}
		return err
	}
	updated, err := s.deliveries.UpdateIfStatus(ctx, d, prevStatus)
	if err != nil {
		return err
	}
	if !updated {
		s.log.Info("skipping concurrent dispatch", "delivery_id", deliveryID, "status", prevStatus)
		return nil
	}

	provider, ok := s.providers[d.Channel]
	if !ok {
		return s.moveToDLQ(ctx, d, now, "no provider for channel", nil)
	}

	breaker := s.breakers[d.Channel]
	if err := breaker.Allow(); err != nil {
		attemptNum := d.RetryCount + 1
		attempt := notification.Attempt{
			ID:            uuid.NewString(),
			AttemptNumber: attemptNum,
			StartedAt:     now,
		}
		return s.handleFailure(ctx, d, attempt, apperrors.ErrTransient)
	}

	attemptNum := d.RetryCount + 1
	attempt := notification.Attempt{
		ID:            uuid.NewString(),
		AttemptNumber: attemptNum,
		StartedAt:     now,
	}

	result, err := provider.Send(ctx, d)
	if err != nil {
		breaker.RecordFailure()
		return s.handleFailure(ctx, d, attempt, err)
	}

	breaker.RecordSuccess()
	resp, _ := json.Marshal(result)
	attempt.ProviderResponse = resp
	d.MarkSucceeded(attempt, s.clock.Now())
	if err := s.deliveries.Update(ctx, d); err != nil {
		return err
	}
	if err := s.deliveries.RecordAttempt(ctx, d.ID, attempt); err != nil {
		return err
	}
	recordDispatchMetric(d.Channel, "success", time.Since(start))
	if m := telemetry.MetricsGlobal(); m != nil {
		m.NotificationsSent.WithLabelValues(string(d.Channel)).Inc()
	}
	return s.audit.Append(ctx, d.ID, string(notification.EventDispatchSucceeded), map[string]any{
		"attempt": attemptNum,
	})
}

func (s *Service) handleFailure(ctx context.Context, d *notification.Delivery, attempt notification.Attempt, err error) error {
	now := s.clock.Now()
	errMsg := err.Error()

	if m := telemetry.MetricsGlobal(); m != nil {
		m.NotificationsFailed.WithLabelValues(string(d.Channel), fmt.Sprintf("%t", apperrors.IsPermanent(err))).Inc()
	}

	if apperrors.IsPermanent(err) || d.RetryCount+1 >= s.maxAttempts {
		return s.moveToDLQ(ctx, d, now, errMsg, &attempt)
	}

	if m := telemetry.MetricsGlobal(); m != nil {
		m.NotificationsRetried.Inc()
	}

	attempt.ErrorMessage = errMsg
	attempt.FinishedAt = &now
	d.MarkRetrying(attempt, errMsg, now)
	if err := s.deliveries.Update(ctx, d); err != nil {
		return err
	}
	if err := s.deliveries.RecordAttempt(ctx, d.ID, attempt); err != nil {
		return err
	}
	_ = s.audit.Append(ctx, d.ID, string(notification.EventDispatchFailed), map[string]any{
		"error":   errMsg,
		"attempt": attempt.AttemptNumber,
		"retry":   true,
	})

	delay := retry.Delay(d.RetryCount, s.retryBase, s.retryMax)
	s.log.Warn("dispatch retry scheduled",
		"delivery_id", d.ID,
		"attempt", attempt.AttemptNumber,
		"delay", delay,
		"error", errMsg,
	)
	return apperrors.ErrTransient
}

func (s *Service) moveToDLQ(ctx context.Context, d *notification.Delivery, now time.Time, errMsg string, attempt *notification.Attempt) error {
	if attempt != nil {
		attempt.ErrorMessage = errMsg
		attempt.FinishedAt = &now
		d.MarkDLQ(*attempt, errMsg, now)
		if err := s.deliveries.RecordAttempt(ctx, d.ID, *attempt); err != nil {
			return err
		}
	} else {
		d.MarkDLQ(notification.Attempt{}, errMsg, now)
	}

	if err := s.deliveries.Update(ctx, d); err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]string{
		"delivery_id": d.ID,
		"error":       errMsg,
	})
	subject := routing.DLQSubject(d.Channel)
	headers := map[string]string{"delivery_id": d.ID}
	if err := s.broker.Publish(ctx, subject, payload, headers); err != nil {
		s.log.Error("publish dlq failed", "delivery_id", d.ID, "error", err)
	}

	if m := telemetry.MetricsGlobal(); m != nil {
		m.NotificationsDLQ.WithLabelValues(string(d.Channel)).Inc()
	}
	recordDispatchMetric(d.Channel, "dlq", 0)

	return s.audit.Append(ctx, d.ID, string(notification.EventMovedToDLQ), map[string]any{
		"error": errMsg,
	})
}

func recordDispatchMetric(ch notification.Channel, result string, duration time.Duration) {
	m := telemetry.MetricsGlobal()
	if m == nil {
		return
	}
	m.DispatchDuration.WithLabelValues(string(ch), result).Observe(duration.Seconds())
}

func (s *Service) Run(ctx context.Context, concurrency int, _ []string) error {
	if concurrency < 1 {
		concurrency = 1
	}
	handler := func(ctx context.Context, subject string, data []byte, headers map[string]string) error {
		return s.HandleMessage(ctx, subject, data, headers)
	}
	for i := 0; i < concurrency; i++ {
		consumerName := fmt.Sprintf("worker-%d", i)
		go func(name string) {
			if err := s.broker.Subscribe(ctx, "NOTIFICATIONS", name, handler); err != nil {
				s.log.Error("worker subscribe failed", "consumer", name, "error", err)
			}
		}(consumerName)
	}
	<-ctx.Done()
	return ctx.Err()
}

// ReplayDelivery re-queues a DLQ delivery for another dispatch attempt.
func (s *Service) ReplayDelivery(ctx context.Context, deliveryID string) error {
	ctx, span := telemetry.StartSpan(ctx, "notification.replay",
		attribute.String("delivery_id", deliveryID),
	)
	defer span.End()

	d, err := s.deliveries.FindByID(ctx, deliveryID)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	if err := d.Replay(now); err != nil {
		return err
	}
	d.RetryCount = 0
	if err := s.deliveries.Update(ctx, d); err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]string{"delivery_id": d.ID})
	subject := routing.NotificationSubject(s.streamPrefix, d.Channel, d.Priority)
	headers := map[string]string{
		"delivery_id": d.ID,
		"replay":      "true",
	}
	telemetry.InjectTrace(ctx, headers)
	if err := s.broker.Publish(ctx, subject, payload, headers); err != nil {
		return err
	}
	return s.audit.Append(ctx, d.ID, string(notification.EventReplayed), map[string]any{
		"operator": "api",
	})
}
