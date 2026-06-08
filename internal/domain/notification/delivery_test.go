package notification

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewDeliveryFromRequest(t *testing.T) {
	req := IngestRequest{
		SchemaVersion:  SchemaVersion,
		IdempotencyKey: "key-1",
		Channel:        "email",
		Priority:       "normal",
		Recipient:      json.RawMessage(`{"address":"a@b.com"}`),
		Template:       json.RawMessage(`{"id":"welcome"}`),
	}
	now := time.Now().UTC()
	d, err := NewDeliveryFromRequest(req, now)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != StatusPending {
		t.Fatalf("expected pending, got %s", d.Status)
	}
}

func TestDeliveryStateMachine(t *testing.T) {
	now := time.Now().UTC()
	d := &Delivery{ID: "1", Status: StatusPending}

	if err := d.Queue(now); err != nil {
		t.Fatal(err)
	}
	if d.Status != StatusQueued {
		t.Fatalf("expected queued, got %s", d.Status)
	}

	if err := d.StartProcessing(now); err != nil {
		t.Fatal(err)
	}

	attempt := Attempt{ID: "a1", AttemptNumber: 1, StartedAt: now}
	d.MarkSucceeded(attempt, now)
	if d.Status != StatusSucceeded {
		t.Fatalf("expected succeeded, got %s", d.Status)
	}

	if err := d.StartProcessing(now); err == nil {
		t.Fatal("expected error when processing succeeded delivery")
	}
}

func TestDeliveryReplay(t *testing.T) {
	now := time.Now().UTC()
	d := &Delivery{ID: "1", Status: StatusDLQ}
	if err := d.Replay(now); err != nil {
		t.Fatal(err)
	}
	if d.Status != StatusQueued {
		t.Fatalf("expected queued after replay, got %s", d.Status)
	}
}

func TestIngestRequestValidation(t *testing.T) {
	req := IngestRequest{Channel: "email"}
	if err := req.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestDeliveryStateMachineMatrix(t *testing.T) {
	now := time.Now().UTC()

	t.Run("pending to queued invalid from succeeded", func(t *testing.T) {
		d := &Delivery{ID: "1", Status: StatusSucceeded}
		if err := d.Queue(now); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("retrying to processing", func(t *testing.T) {
		d := &Delivery{ID: "1", Status: StatusRetrying}
		if err := d.StartProcessing(now); err != nil {
			t.Fatal(err)
		}
		if d.Status != StatusProcessing {
			t.Fatalf("expected processing, got %s", d.Status)
		}
	})

	t.Run("mark retrying increments count", func(t *testing.T) {
		d := &Delivery{ID: "1", Status: StatusProcessing, RetryCount: 1}
		attempt := Attempt{ID: "a", AttemptNumber: 2, StartedAt: now}
		d.MarkRetrying(attempt, "err", now)
		if d.RetryCount != 2 {
			t.Fatalf("expected retry count 2, got %d", d.RetryCount)
		}
	})

	t.Run("replay from failed", func(t *testing.T) {
		d := &Delivery{ID: "1", Status: StatusFailed}
		if err := d.Replay(now); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("replay invalid from pending", func(t *testing.T) {
		d := &Delivery{ID: "1", Status: StatusPending}
		if err := d.Replay(now); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("mark dlq", func(t *testing.T) {
		d := &Delivery{ID: "1", Status: StatusProcessing}
		attempt := Attempt{ID: "a", AttemptNumber: 1, StartedAt: now}
		d.MarkDLQ(attempt, "permanent", now)
		if d.Status != StatusDLQ {
			t.Fatalf("expected dlq, got %s", d.Status)
		}
	})
}

func TestShouldSchedule(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)
	d := &Delivery{ScheduledAt: &future}
	if !d.ShouldSchedule(now) {
		t.Fatal("expected schedule")
	}
	past := now.Add(-time.Hour)
	d.ScheduledAt = &past
	if d.ShouldSchedule(now) {
		t.Fatal("expected not scheduled")
	}
}

func TestPayloadJSON(t *testing.T) {
	d := &Delivery{
		Recipient: json.RawMessage(`{"address":"a@b.com"}`),
		Template:  json.RawMessage(`{"id":"welcome"}`),
	}
	b, err := d.PayloadJSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("expected payload")
	}
	d2 := &Delivery{Payload: json.RawMessage(`{"custom":true}`)}
	b2, err := d2.PayloadJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(b2) != `{"custom":true}` {
		t.Fatalf("unexpected payload: %s", b2)
	}
}
