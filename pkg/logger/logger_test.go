package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewInfoLevel(t *testing.T) {
	log := New("info")
	log.Info("hello")
}

func TestWithWriterFiltersDebug(t *testing.T) {
	var buf bytes.Buffer
	log := WithWriter(&buf, "info")
	log.Debug("hidden")
	log.Info("visible")
	if strings.Contains(buf.String(), "hidden") {
		t.Fatal("debug should be filtered at info level")
	}
	if !strings.Contains(buf.String(), "visible") {
		t.Fatalf("expected info log, got: %s", buf.String())
	}
}

func TestWithWriterDebugLevel(t *testing.T) {
	var buf bytes.Buffer
	log := WithWriter(&buf, "debug")
	log.Debug("debug-msg")
	if !strings.Contains(buf.String(), "debug-msg") {
		t.Fatalf("expected debug log, got: %s", buf.String())
	}
}

func TestMarshalFields(t *testing.T) {
	got := MarshalFields(map[string]string{"k": "v"})
	if got != `{"k":"v"}` {
		t.Fatalf("unexpected: %s", got)
	}
}

func TestMarshalFieldsFallback(t *testing.T) {
	got := MarshalFields(make(chan int))
	if got == "" {
		t.Fatal("expected fallback string")
	}
}

func TestNewLevelMapping(t *testing.T) {
	for _, level := range []string{"debug", "warn", "error", "unknown"} {
		if New(level) == nil {
			t.Fatalf("nil logger for level %s", level)
		}
	}
}
