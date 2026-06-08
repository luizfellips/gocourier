package bootstrap

import "testing"

func TestSignalContext(t *testing.T) {
	ctx, cancel := SignalContext()
	defer cancel()
	if ctx == nil {
		t.Fatal("expected context")
	}
	select {
	case <-ctx.Done():
	default:
	}
}
