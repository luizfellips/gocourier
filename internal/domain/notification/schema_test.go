package notification

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestIngestRequestValidationMatrix(t *testing.T) {
	valid := IngestRequest{
		SchemaVersion:  SchemaVersion,
		IdempotencyKey: "key-1",
		Channel:        "email",
		Priority:       "normal",
		Recipient:      json.RawMessage(`{"address":"a@b.com"}`),
		Template:       json.RawMessage(`{"id":"t1"}`),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid request: %v", err)
	}

	tests := []struct {
		name string
		req  IngestRequest
	}{
		{"missing idempotency", IngestRequest{Channel: "email", Recipient: json.RawMessage(`{}`), Template: json.RawMessage(`{}`)}},
		{"long idempotency", func() IngestRequest {
			r := valid
			r.IdempotencyKey = strings.Repeat("x", 257)
			return r
		}()},
		{"invalid channel", func() IngestRequest {
			r := valid
			r.Channel = "fax"
			return r
		}()},
		{"missing recipient", func() IngestRequest {
			r := valid
			r.Recipient = nil
			return r
		}()},
		{"missing template and payload", func() IngestRequest {
			r := valid
			r.Template = nil
			r.Payload = nil
			return r
		}()},
		{"unsupported schema", func() IngestRequest {
			r := valid
			r.SchemaVersion = "2.0"
			return r
		}()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.req.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestIngestRequestDefaultsSchemaVersion(t *testing.T) {
	req := IngestRequest{
		IdempotencyKey: "k",
		Channel:        "email",
		Recipient:      json.RawMessage(`{}`),
		Template:       json.RawMessage(`{}`),
	}
	if err := req.Validate(); err != nil {
		t.Fatal(err)
	}
	if req.SchemaVersion != SchemaVersion {
		t.Fatalf("expected default schema version")
	}
}

func TestIngestRequestDefaultsTenant(t *testing.T) {
	req := IngestRequest{
		SchemaVersion:  SchemaVersion,
		IdempotencyKey: "k",
		Channel:        "email",
		Recipient:      json.RawMessage(`{}`),
		Template:       json.RawMessage(`{}`),
	}
	if err := req.Validate(); err != nil {
		t.Fatal(err)
	}
	if req.Metadata.TenantID != "default" {
		t.Fatalf("expected default tenant")
	}
}
