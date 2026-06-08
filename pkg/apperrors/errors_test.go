package apperrors

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsTransient(t *testing.T) {
	wrapped := fmt.Errorf("provider: %w", ErrTransient)
	if !IsTransient(wrapped) {
		t.Fatal("expected transient")
	}
	if IsTransient(ErrPermanent) {
		t.Fatal("permanent should not be transient")
	}
}

func TestIsPermanent(t *testing.T) {
	wrapped := fmt.Errorf("validation: %w", ErrPermanent)
	if !IsPermanent(wrapped) {
		t.Fatal("expected permanent")
	}
	if IsPermanent(errors.New("other")) {
		t.Fatal("unrelated error")
	}
}
