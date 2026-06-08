package retry

import (
	"math/rand"
	"testing"
	"time"
)

func TestDelayExponentialWithCap(t *testing.T) {
	rand.Seed(42)
	base := time.Second
	max := 30 * time.Minute

	d1 := Delay(1, base, max)
	if d1 < base || d1 > base+base/2 {
		t.Fatalf("attempt 1 delay out of range: %v", d1)
	}

	d5 := Delay(5, base, max)
	expectedBase := 16 * time.Second
	if d5 < expectedBase || d5 > expectedBase+expectedBase/2 {
		t.Fatalf("attempt 5 delay out of range: %v", d5)
	}

	d20 := Delay(20, base, max)
	if d20 < max || d20 > max+max/2 {
		t.Fatalf("attempt 20 should be capped near max: %v", d20)
	}
}

func TestDelayMinimumAttempt(t *testing.T) {
	rand.Seed(1)
	d := Delay(0, time.Second, time.Minute)
	if d < time.Second {
		t.Fatalf("attempt 0 treated as 1: %v", d)
	}
}
