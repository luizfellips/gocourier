package ports

import "testing"

func TestSystemClockNow(t *testing.T) {
	before := SystemClock{}.Now()
	if before.IsZero() {
		t.Fatal("expected non-zero time")
	}
}
