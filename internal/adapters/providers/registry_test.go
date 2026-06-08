package providers

import "testing"

func TestDefaultProviders(t *testing.T) {
	p := Default()
	if len(p) != 4 {
		t.Fatalf("expected 4 providers, got %d", len(p))
	}
}
