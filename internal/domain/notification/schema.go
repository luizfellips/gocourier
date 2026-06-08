package notification

import (
	"encoding/json"
	"fmt"
	"time"
)

const SchemaVersion = "1.0"

type IngestRequest struct {
	SchemaVersion  string          `json:"schema_version"`
	IdempotencyKey string          `json:"idempotency_key"`
	Channel        string          `json:"channel"`
	Priority       string          `json:"priority"`
	Recipient      json.RawMessage `json:"recipient"`
	Template       json.RawMessage `json:"template"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	Metadata       RequestMetadata `json:"metadata"`
	ScheduledAt    *time.Time      `json:"scheduled_at,omitempty"`
}

type RequestMetadata struct {
	CorrelationID string `json:"correlation_id,omitempty"`
	CausationID   string `json:"causation_id,omitempty"`
	TenantID      string `json:"tenant_id,omitempty"`
}

func (r *IngestRequest) Validate() error {
	if r.SchemaVersion == "" {
		r.SchemaVersion = SchemaVersion
	}
	if r.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema_version: %s", r.SchemaVersion)
	}
	if r.IdempotencyKey == "" {
		return fmt.Errorf("idempotency_key is required")
	}
	if len(r.IdempotencyKey) > 256 {
		return fmt.Errorf("idempotency_key exceeds 256 characters")
	}
	if _, err := ParseChannel(r.Channel); err != nil {
		return err
	}
	if _, err := ParsePriority(r.Priority); err != nil {
		return err
	}
	if len(r.Recipient) == 0 {
		return fmt.Errorf("recipient is required")
	}
	if len(r.Template) == 0 && len(r.Payload) == 0 {
		return fmt.Errorf("template or payload is required")
	}
	if r.Metadata.TenantID == "" {
		r.Metadata.TenantID = "default"
	}
	return nil
}

type IngestResponse struct {
	DeliveryID string `json:"delivery_id"`
	Status     Status `json:"status"`
	Duplicate  bool   `json:"duplicate,omitempty"`
}
