package ports

import (
	"context"
	"encoding/json"
	"time"

)

type DashboardSummary struct {
	StatusCounts   map[string]int `json:"status_counts"`
	OutboxPending  int            `json:"outbox_pending"`
	ScheduledDue   int            `json:"scheduled_due"`
	RecentActivity []AuditEntry   `json:"recent_activity"`
	Deliveries     []DeliveryRow  `json:"deliveries"`
}

type DeliveryRow struct {
	ID             string     `json:"id"`
	IdempotencyKey string     `json:"idempotency_key"`
	Channel        string     `json:"channel"`
	Priority       string     `json:"priority"`
	Status         string     `json:"status"`
	RetryCount     int        `json:"retry_count"`
	LastError      string     `json:"last_error,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ScheduledAt    *time.Time `json:"scheduled_at,omitempty"`
}

type DeliveryDetail struct {
	Delivery DeliveryRow        `json:"delivery"`
	Attempts []AttemptRow       `json:"attempts"`
	Audit    []AuditEntry       `json:"audit"`
	Outbox   []OutboxStatusRow  `json:"outbox"`
}

type AttemptRow struct {
	ID            string          `json:"id"`
	AttemptNumber int             `json:"attempt_number"`
	Success       bool            `json:"success"`
	ErrorMessage  string          `json:"error_message,omitempty"`
	StartedAt     time.Time       `json:"started_at"`
	FinishedAt    *time.Time      `json:"finished_at,omitempty"`
	Response      json.RawMessage `json:"provider_response,omitempty"`
}

type AuditEntry struct {
	ID         int64           `json:"id"`
	DeliveryID string          `json:"delivery_id"`
	EventType  string          `json:"event_type"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

type OutboxStatusRow struct {
	ID        int64      `json:"id"`
	Status    string     `json:"status"`
	Subject   string     `json:"subject"`
	Attempts  int        `json:"attempts"`
	LastError string     `json:"last_error,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	Published *time.Time `json:"published_at,omitempty"`
}

type DashboardReader interface {
	Summary(ctx context.Context, limit int) (*DashboardSummary, error)
	Detail(ctx context.Context, deliveryID string) (*DeliveryDetail, error)
}
