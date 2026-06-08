package circuitbreaker

import (
	"testing"
	"time"

	"github.com/gocourier/pkg/apperrors"
)

func TestBreakerOpensAfterThreshold(t *testing.T) {
	b := New(3, time.Minute, 30*time.Second)
	for i := 0; i < 3; i++ {
		if err := b.Allow(); err != nil {
			t.Fatalf("allow %d: %v", i, err)
		}
		b.RecordFailure()
	}
	if err := b.Allow(); err == nil {
		t.Fatal("expected circuit open")
	} else if err != apperrors.ErrCircuitOpen {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBreakerHalfOpenAfterCooldown(t *testing.T) {
	now := time.Now()
	b := &Breaker{
		threshold: 2,
		window:    time.Minute,
		cooldown:  10 * time.Second,
		state:     stateOpen,
		openedAt:  now,
	}
	b.openedAt = now.Add(-11 * time.Second)
	if err := b.Allow(); err != nil {
		t.Fatalf("expected half-open allow: %v", err)
	}
	b.RecordSuccess()
	if err := b.Allow(); err != nil {
		t.Fatalf("expected closed after success: %v", err)
	}
}

func TestBreakerSuccessResetsFailures(t *testing.T) {
	b := New(5, time.Minute, 30*time.Second)
	for i := 0; i < 4; i++ {
		b.RecordFailure()
	}
	b.RecordSuccess()
	for i := 0; i < 4; i++ {
		b.RecordFailure()
	}
	if err := b.Allow(); err != nil {
		t.Fatalf("should not open yet: %v", err)
	}
}
