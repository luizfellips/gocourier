package notification

import "testing"

func TestStatusHelpers(t *testing.T) {
	if !StatusSucceeded.IsTerminal() {
		t.Fatal("succeeded is terminal")
	}
	if StatusQueued.IsTerminal() {
		t.Fatal("queued is not terminal")
	}
	if !StatusQueued.CanDispatch() {
		t.Fatal("queued can dispatch")
	}
	if StatusPending.CanDispatch() {
		t.Fatal("pending cannot dispatch")
	}
}
