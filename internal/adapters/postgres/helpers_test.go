package postgres

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestNullableJSON(t *testing.T) {
	if nullableJSON(nil) != nil {
		t.Fatal("nil input should return nil")
	}
	if nullableJSON(json.RawMessage{}) != nil {
		t.Fatal("empty json should return nil")
	}
	val := nullableJSON(json.RawMessage(`{"a":1}`))
	if val == nil {
		t.Fatal("expected non-nil")
	}
}

func TestNullString(t *testing.T) {
	if nullString("") != nil {
		t.Fatal("empty string should return nil")
	}
	if nullString("err") != "err" {
		t.Fatal("expected string value")
	}
}

func TestNewAttemptID(t *testing.T) {
	id := NewAttemptID()
	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("invalid uuid: %s", id)
	}
}

func TestConstructorsAcceptNilPool(t *testing.T) {
	if NewDeliveryRepo(nil) == nil {
		t.Fatal("expected delivery repo")
	}
	if NewStore(nil) == nil {
		t.Fatal("expected store")
	}
	if NewOutboxRepo(nil) == nil {
		t.Fatal("expected outbox repo")
	}
	if NewScheduledRepo(nil) == nil {
		t.Fatal("expected scheduled repo")
	}
	if NewAuditRepo(nil) == nil {
		t.Fatal("expected audit repo")
	}
	if NewDashboardRepo(nil) == nil {
		t.Fatal("expected dashboard repo")
	}
}
