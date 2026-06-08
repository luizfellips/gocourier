package testkit

import (
	"encoding/json"
	"fmt"

	"github.com/gocourier/internal/domain/notification"
	"github.com/google/uuid"
)

// SampleIngestRequest returns a valid ingest request with a unique idempotency key.
func SampleIngestRequest(suffix string) notification.IngestRequest {
	return notification.IngestRequest{
		SchemaVersion:  notification.SchemaVersion,
		IdempotencyKey: fmt.Sprintf("test-%s-%s", suffix, uuid.NewString()),
		Channel:        "email",
		Priority:       "normal",
		Recipient:      json.RawMessage(`{"address":"user@example.com"}`),
		Template:       json.RawMessage(`{"id":"welcome"}`),
	}
}

// SampleIngestRequestWithKey returns a valid ingest request with a fixed idempotency key.
func SampleIngestRequestWithKey(key string) notification.IngestRequest {
	req := SampleIngestRequest("fixed")
	req.IdempotencyKey = key
	return req
}
