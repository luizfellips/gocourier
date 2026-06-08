package replay

import (
	"context"

	"github.com/gocourier/internal/application/dispatch"
	"github.com/gocourier/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

type Service struct {
	dispatch *dispatch.Service
}

func NewService(d *dispatch.Service) *Service {
	return &Service{dispatch: d}
}

func (s *Service) Replay(ctx context.Context, deliveryID string) error {
	ctx, span := telemetry.StartSpan(ctx, "notification.replay",
		attribute.String("delivery_id", deliveryID),
	)
	defer span.End()
	return s.dispatch.ReplayDelivery(ctx, deliveryID)
}
