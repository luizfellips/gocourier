package notification

import "time"

type DomainEventType string

const (
	EventReceived      DomainEventType = "NotificationReceived"
	EventValidated     DomainEventType = "NotificationValidated"
	EventQueued        DomainEventType = "NotificationQueued"
	EventDispatchAttempted DomainEventType = "DispatchAttempted"
	EventDispatchSucceeded DomainEventType = "DispatchSucceeded"
	EventDispatchFailed    DomainEventType = "DispatchFailed"
	EventMovedToDLQ        DomainEventType = "NotificationMovedToDLQ"
	EventReplayed          DomainEventType = "NotificationReplayed"
)

type DomainEvent struct {
	Type        DomainEventType `json:"type"`
	DeliveryID  string          `json:"delivery_id"`
	OccurredAt  time.Time       `json:"occurred_at"`
	Payload     map[string]any  `json:"payload,omitempty"`
}
